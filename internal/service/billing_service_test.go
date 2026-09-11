package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"cpa-usage-keeper/internal/config"
	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/pricing"
	"cpa-usage-keeper/internal/repository"
	"cpa-usage-keeper/internal/repository/dto"

	"gorm.io/gorm"
)

type billingFixture struct {
	db      *gorm.DB
	readDB  *gorm.DB
	catalog *pricing.Catalog
	service BillingProvider
	cfg     config.Config
	request dto.SaveBillingBillRequest
	start   time.Time
	end     time.Time
	keys    []entities.CPAAPIKey
}

func newBillingFixture(t *testing.T) *billingFixture {
	t.Helper()
	f := &billingFixture{cfg: config.Config{SQLitePath: filepath.Join(t.TempDir(), "billing.db")}}
	var err error
	f.db, f.readDB, err = repository.OpenDatabasePools(f.cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.close(t) })
	f.catalog = billingTestCatalog(t, 1)
	f.service = NewBillingService(f.db, f.catalog)
	f.start = time.Date(2026, 5, 1, 0, 0, 0, 500000000, time.UTC)
	f.end = f.start.Add(2 * time.Second)
	f.request = dto.SaveBillingBillRequest{AuthIndex: "account-a", Cycle: 1,
		WindowStart: "2026-04-30T20:00:00.500000000-04:00", WindowEnd: "2026-05-01T08:00:02.500000000+08:00", Fee: "1.00"}
	plan := "Pro20x"
	identities := []entities.UsageIdentity{
		{AuthType: entities.UsageIdentityAuthTypeAuthFile, Identity: "account-a", Name: "Account A", PlanType: &plan, ActiveStart: &f.start, ActiveUntil: &f.end},
		{AuthType: entities.UsageIdentityAuthTypeAuthFile, Identity: "account-b", Name: "Account B", IsDeleted: true},
		{AuthType: entities.UsageIdentityAuthTypeAIProvider, Identity: "provider-a", Name: "Not a subscription account"},
	}
	if err := f.db.Create(&identities).Error; err != nil {
		t.Fatal(err)
	}
	f.keys = []entities.CPAAPIKey{
		{APIKey: "sk-billing-member-one-secret", KeyAlias: "Shared label"},
		{APIKey: "sk-billing-member-two-secret", KeyAlias: "Shared label", IsDeleted: true},
	}
	if err := f.db.Create(&f.keys).Error; err != nil {
		t.Fatal(err)
	}
	alias := "priced-model"
	events := []entities.UsageEvent{
		{ID: 101, AuthIndex: "account-a", APIGroupKey: f.keys[0].APIKey, Timestamp: f.start, Model: "unlisted-upstream-model", ModelAlias: &alias, ResponseServiceTier: "priority", InputTokens: 1000000, TotalTokens: 1000000},
		{ID: 102, AuthIndex: "account-a", APIGroupKey: f.keys[1].APIKey, Timestamp: f.start.Add(time.Second), Model: "priced-model", Failed: true, InputTokens: 1000000, TotalTokens: 1000000},
		{ID: 103, AuthIndex: "account-a", APIGroupKey: f.keys[0].APIKey, Timestamp: f.end.Add(-time.Nanosecond), Model: "priced-model", Failed: true, InputTokens: 1000000, TotalTokens: 1000000},
		{ID: 104, AuthIndex: "account-a", APIGroupKey: "unknown-before", Timestamp: f.start.Add(-time.Nanosecond), Model: "unpriced", InputTokens: 1000000},
		{ID: 105, AuthIndex: "account-a", APIGroupKey: "unknown-at-exclusive-end", Timestamp: f.end, Model: "unpriced", InputTokens: 1000000},
		{ID: 106, AuthIndex: "account-b", APIGroupKey: f.keys[0].APIKey, Timestamp: f.start, Model: "priced-model", InputTokens: 1000000, TotalTokens: 1000000},
		{ID: 107, AuthIndex: "", APIGroupKey: "unknown-unattributed", Timestamp: f.start, Model: "unpriced", InputTokens: 1000000},
	}
	if err := f.db.Create(&events).Error; err != nil {
		t.Fatal(err)
	}
	// Same instant, deliberately different stored offset/fraction representation.
	if err := f.db.Model(&entities.UsageEvent{}).Where("id = ?", 102).
		UpdateColumn("timestamp", "2026-05-01T08:00:01.500000000+08:00").Error; err != nil {
		t.Fatal(err)
	}
	columns := entities.UsageEventStorageColumns
	if err := f.db.Exec("INSERT INTO usage_events_archive (" + columns + ") SELECT " + columns + " FROM usage_events WHERE id IN (101, 102)").Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Delete(&entities.UsageEvent{}, 102).Error; err != nil {
		t.Fatal(err)
	}
	return f
}

func billingTestCatalog(t *testing.T, multiplier float64) *pricing.Catalog {
	t.Helper()
	snapshot, err := pricing.CompileSnapshot([]pricing.ModelConfig{{
		Pricing: entities.ModelPriceSetting{Model: "priced-model", PromptPricePer1M: 1, CompletionPricePer1M: 2, CacheReadPricePer1M: .5, CacheWritePricePer1M: 1.5, PriceMultiplier: &multiplier},
		Rules:   []pricing.RuleConfig{{Key: "response_service_tier", Value: "priority", Multiplier: 2}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return pricing.NewCatalog(snapshot)
}

func (f *billingFixture) close(t *testing.T) {
	t.Helper()
	if f.readDB != nil && f.readDB != f.db {
		db, err := f.readDB.DB()
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		f.readDB = nil
	}
	if f.db != nil {
		db, err := f.db.DB()
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		f.db = nil
	}
}

func TestBillingExactWindowArchivesAliasesAndSnapshots(t *testing.T) {
	f := newBillingFixture(t)
	ctx := context.Background()
	f.request.FeeNote = "subscription " + f.keys[0].APIKey
	bill, err := f.service.SaveBillingBill(ctx, f.request)
	if err != nil {
		t.Fatal(err)
	}
	if bill.Events != 3 || bill.TotalCostUSD != 4 || len(bill.Members) != 2 {
		t.Fatalf("wrong exact hot/archive usage: %+v", bill)
	}
	first, second := bill.Members[0], bill.Members[1]
	if first.APIKeyID != f.keys[0].ID || first.Requests != 2 || first.Success != 1 || first.Failure != 1 || first.TotalTokens != 2000000 || first.AmountCents != 75 || second.APIKeyID != f.keys[1].ID || second.Failure != 1 || second.AmountCents != 25 {
		t.Fatalf("alias collision/deleted key lost participant: %+v", bill.Members)
	}
	payload, err := json.Marshal(bill)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range f.keys {
		if strings.Contains(string(payload), key.APIKey) {
			t.Fatal("snapshot leaked a raw key")
		}
	}
	accounts, err := f.service.ListBillingAccounts(ctx)
	if err != nil || len(accounts) != 2 || accounts[1].AuthIndex != "account-b" {
		t.Fatalf("auth-file accounts should include deleted accounts but not providers: %+v, %v", accounts, err)
	}
	// Display metadata and prices change, but listing must return the same saved calculation.
	if err := f.db.Model(&entities.UsageIdentity{}).Where("identity = ?", "account-a").Updates(map[string]any{"name": "Renamed account", "plan_type": "new plan"}).Error; err != nil {
		t.Fatal(err)
	}
	f.catalog.Replace(billingTestCatalog(t, 2).Snapshot())
	saved, err := f.service.ListBillingBills(ctx, "account-a")
	if err != nil || len(saved) != 1 || !reflect.DeepEqual(saved[0], bill) {
		t.Fatalf("live metadata/prices changed the snapshot: %+v, %v", saved, err)
	}
	f.request.Fee = "2.50"
	recalculated, err := f.service.SaveBillingBill(ctx, f.request)
	if err != nil {
		t.Fatal(err)
	}
	if recalculated.ID != bill.ID || recalculated.TotalCostUSD != 8 || recalculated.Account != "Renamed account" || recalculated.FeeCents != 250 || !recalculated.WindowStart.Equal(f.start) || !recalculated.WindowEnd.Equal(f.end) {
		t.Fatalf("explicit replacement did not retain identity or update calculation: %+v", recalculated)
	}
	f.close(t)
	f.db, f.readDB, err = repository.OpenDatabasePools(f.cfg)
	if err != nil {
		t.Fatal(err)
	}
	f.service = NewBillingService(f.db, pricing.NewCatalog(pricing.EmptySnapshot()))
	reloaded, err := f.service.ListBillingBills(ctx, "account-a")
	if err != nil || len(reloaded) != 1 || !reflect.DeepEqual(reloaded[0], recalculated) {
		t.Fatalf("restart did not restore the canonical snapshot without live prices: %+v, %v", reloaded, err)
	}
	if err := f.service.DeleteBillingBill(ctx, recalculated.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.service.DeleteBillingBill(ctx, recalculated.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("second delete should be missing: %v", err)
	}
	empty, err := f.service.ListBillingBills(ctx, "account-a")
	if err != nil || len(empty) != 0 {
		t.Fatalf("delete did not persist: %+v %v", empty, err)
	}
}

func TestBillingFailurePreservesOldBillAndIsolatesCycles(t *testing.T) {
	f := newBillingFixture(t)
	ctx := context.Background()
	bill, err := f.service.SaveBillingBill(ctx, f.request)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []struct {
		name, key, model string
		tokens           int64
		failed           bool
	}{
		{"unknown member", "unknown-participant", "priced-model", 1, false},
		{"unpriced model", f.keys[0].APIKey, "not-configured", 1, false},
		{"unpriced zero-token failure", f.keys[0].APIKey, "not-configured", 0, true},
	} {
		t.Run(bad.name, func(t *testing.T) {
			event := entities.UsageEvent{AuthIndex: "account-a", APIGroupKey: bad.key, Model: bad.model, Timestamp: f.start.Add(time.Second), InputTokens: bad.tokens, Failed: bad.failed}
			if err := f.db.Create(&event).Error; err != nil {
				t.Fatal(err)
			}
			if _, err := f.service.SaveBillingBill(ctx, f.request); !errors.Is(err, ErrBillingIncomplete) {
				t.Fatalf("expected fail closed, got %v", err)
			}
			if err := f.db.Delete(&event).Error; err != nil {
				t.Fatal(err)
			}
			saved, err := f.service.ListBillingBills(ctx, "account-a")
			if err != nil || len(saved) != 1 || !reflect.DeepEqual(saved[0], bill) {
				t.Fatalf("failed recalculation replaced old bill: %+v %v", saved, err)
			}
		})
	}
	request := f.request
	request.Cycle = 2
	if _, err := f.service.SaveBillingBill(ctx, request); !errors.Is(err, repository.ErrBillingOverlap) {
		t.Fatalf("overlap accepted: %v", err)
	}
	request.AuthIndex = "account-b"
	other, err := f.service.SaveBillingBill(ctx, request)
	if err != nil || other.Events != 1 || other.TotalCostUSD != 1 || other.ID == bill.ID {
		t.Fatalf("cross-account isolation: %+v %v", other, err)
	}
	request = f.request
	request.WindowStart, request.WindowEnd = "2027-01-01T00:00:00Z", "2027-02-01T00:00:00Z"
	if _, err := f.service.SaveBillingBill(ctx, request); !errors.Is(err, ErrBillingIncomplete) {
		t.Fatalf("zero-use window accepted: %v", err)
	}
	request.AuthIndex = "absent"
	if _, err := f.service.SaveBillingBill(ctx, request); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing account accepted: %v", err)
	}
	// Touching endpoints are not overlaps, and nanos remain authoritative.
	if err := f.db.Delete(&entities.UsageEvent{}, 105).Error; err != nil {
		t.Fatal(err)
	}
	event := entities.UsageEvent{AuthIndex: "account-a", APIGroupKey: f.keys[0].APIKey, Model: "priced-model", Timestamp: f.end, InputTokens: 1000000, TotalTokens: 1000000}
	if err := f.db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	request = f.request
	request.Cycle, request.WindowStart, request.WindowEnd = 2, f.end.Format(time.RFC3339Nano), f.end.Add(time.Second).Format(time.RFC3339Nano)
	adjacent, err := f.service.SaveBillingBill(ctx, request)
	if err != nil || adjacent.Events != 1 {
		t.Fatalf("touching window rejected: %+v %v", adjacent, err)
	}
	listed, err := f.service.ListBillingBills(ctx, "account-a")
	if err != nil || len(listed) != 2 || listed[0].Cycle != 2 || !reflect.DeepEqual(listed[1], bill) {
		t.Fatalf("cycle order/isolation: %+v %v", listed, err)
	}
	if err := f.db.Where("identity = ?", "account-b").Delete(&entities.UsageIdentity{}).Error; err != nil {
		t.Fatal(err)
	}
	accounts, err := f.service.ListBillingAccounts(ctx)
	if err != nil || len(accounts) != 2 || accounts[1].Name != "Account B" {
		t.Fatalf("removed identity hid historical account: %+v %v", accounts, err)
	}
}

func TestBillingConcurrentOverlappingCyclesCannotBothSave(t *testing.T) {
	f := newBillingFixture(t)
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, cycle := range []int64{1, 2} {
		go func(cycle int64) {
			request := f.request
			request.Cycle = cycle
			<-start
			_, err := f.service.SaveBillingBill(context.Background(), request)
			results <- err
		}(cycle)
	}
	close(start)
	successes, conflicts := 0, 0
	for range 2 {
		err := <-results
		if err == nil {
			successes++
		} else if errors.Is(err, repository.ErrBillingOverlap) {
			conflicts++
		} else {
			t.Fatalf("unexpected concurrent save failure: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("overlapping saves escaped transaction isolation: success=%d conflicts=%d", successes, conflicts)
	}
	bills, err := f.service.ListBillingBills(context.Background(), "account-a")
	if err != nil || len(bills) != 1 || bills[0].FeeCents != 100 {
		t.Fatalf("unexpected committed bills: %+v %v", bills, err)
	}
}

func TestBillingMigratesExistingDatabaseWithoutChangingUsage(t *testing.T) {
	f := newBillingFixture(t)
	if err := f.db.Migrator().DropTable(&entities.BillingBill{}); err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec("DROP INDEX idx_usage_archive_auth_index_timestamp").Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec("DELETE FROM schema_migrations WHERE version = ?", "20260911_create_billing").Error; err != nil {
		t.Fatal(err)
	}
	f.close(t)
	var err error
	f.db, f.readDB, err = repository.OpenDatabasePools(f.cfg)
	if err != nil {
		t.Fatal(err)
	}
	f.service = NewBillingService(f.db, f.catalog)
	bill, err := f.service.SaveBillingBill(context.Background(), f.request)
	if err != nil || bill.Events != 3 || bill.TotalCostUSD != 4 {
		t.Fatalf("existing database migration lost usage: %+v %v", bill, err)
	}
	// DB uniqueness is a final defense independent of service-level reads.
	duplicate := entities.BillingBill{AuthIndex: bill.AuthIndex, Cycle: bill.Cycle, WindowStart: "x", WindowEnd: "y", SnapshotJSON: "{}"}
	if err := f.db.Create(&duplicate).Error; err == nil {
		t.Fatal("duplicate account/cycle was accepted")
	}
	if got := fmt.Sprint(bill.Members[0].AmountCents, "/", bill.Members[1].AmountCents); got != "75/25" {
		t.Fatalf("migration changed allocations: %s", got)
	}
}
