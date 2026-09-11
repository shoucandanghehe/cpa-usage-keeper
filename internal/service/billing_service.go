package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/helper"
	"cpa-usage-keeper/internal/pricing"
	"cpa-usage-keeper/internal/repository"
	"cpa-usage-keeper/internal/repository/dto"
	"cpa-usage-keeper/internal/timeutil"

	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

var (
	ErrBillingInvalidInput = errors.New("invalid billing input")
	ErrBillingIncomplete   = errors.New("billing data is incomplete")
	billingFeePattern      = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,2})?$`)
	billingTimePattern     = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]{1,9})?(?:Z|[+-](?:[01][0-9]|2[0-3]):[0-5][0-9])$`)
	billingKeyPattern      = regexp.MustCompile(`(?i)\b(?:sk-|key-)[a-z0-9_./+=-]+`)
)

const billingMaxSafeInteger int64 = 9_007_199_254_740_991

type BillingProvider interface {
	ListBillingAccounts(context.Context) ([]dto.BillingAccount, error)
	ListBillingBills(context.Context, string) ([]dto.BillingBill, error)
	SaveBillingBill(context.Context, dto.SaveBillingBillRequest) (dto.BillingBill, error)
	DeleteBillingBill(context.Context, int64) error
}

type billingService struct {
	db      *gorm.DB
	catalog *pricing.Catalog
}

func NewBillingService(db *gorm.DB, catalog *pricing.Catalog) BillingProvider {
	return &billingService{db: db, catalog: catalog}
}

func (s *billingService) ListBillingAccounts(ctx context.Context) ([]dto.BillingAccount, error) {
	accounts := []dto.BillingAccount{}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		identities, err := repository.ListBillingAccountIdentities(tx)
		if err != nil {
			return err
		}
		keys, err := repository.ListBillingAPIKeys(tx)
		if err != nil {
			return err
		}
		redact := billingRedactor(keys)
		seen := make(map[string]bool, len(identities))
		for _, identity := range identities {
			accounts = append(accounts, billingAccount(identity, redact))
			seen[identity.Identity] = true
		}
		// A physically removed identity must not hide its saved bills. These
		// fallback names are already redacted snapshots, not live metadata.
		latest, err := repository.ListLatestBillingBills(tx)
		if err != nil {
			return err
		}
		for _, row := range latest {
			if seen[row.AuthIndex] {
				continue
			}
			bill, err := decodeBillingBill(row)
			if err != nil {
				return err
			}
			accounts = append(accounts, dto.BillingAccount{
				AuthIndex: bill.AuthIndex, Name: bill.Account, Alias: bill.Alias, Plan: bill.Plan,
			})
		}
		return nil
	})
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].AuthIndex < accounts[j].AuthIndex })
	return accounts, err
}

func (s *billingService) ListBillingBills(ctx context.Context, authIndex string) ([]dto.BillingBill, error) {
	if strings.TrimSpace(authIndex) == "" {
		return nil, fmt.Errorf("%w: auth_index is required", ErrBillingInvalidInput)
	}
	bills := []dto.BillingBill{}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, err := repository.ListBillingBills(tx, authIndex)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			if _, err := repository.FindBillingAccountIdentity(tx, authIndex); err != nil {
				return err
			}
		}
		for _, row := range rows {
			bill, err := decodeBillingBill(row)
			if err != nil {
				return err
			}
			bills = append(bills, bill)
		}
		return nil
	})
	return bills, err
}

func (s *billingService) SaveBillingBill(ctx context.Context, request dto.SaveBillingBillRequest) (dto.BillingBill, error) {
	start, end, fee, err := validateBillingRequest(request)
	if err != nil {
		return dto.BillingBill{}, err
	}
	// One resolver per operation keeps model aliases, tiers, and request/token
	// normalization on the same canonical price/rule snapshot as Usage.
	resolver := s.catalog.NewResolver()
	var bill dto.BillingBill
	err = s.db.WithContext(ctx).Clauses(dbresolver.Write).Transaction(func(tx *gorm.DB) error {
		identity, err := repository.FindBillingAccountIdentity(tx, request.AuthIndex)
		if err != nil {
			return err
		}
		keys, err := repository.ListBillingAPIKeys(tx)
		if err != nil {
			return err
		}
		redact := billingRedactor(keys)
		account := billingAccount(identity, redact)
		members, events, total, err := calculateBillingUsage(tx, resolver, keys, redact, request.AuthIndex, start, end, fee)
		if err != nil {
			return err
		}
		bill = dto.BillingBill{
			AuthIndex: request.AuthIndex, Cycle: request.Cycle, WindowStart: start, WindowEnd: end,
			BillingSnapshot: dto.BillingSnapshot{
				Account: account.Name, Alias: account.Alias, Plan: account.Plan,
				FeeCents: fee, FeeNote: redact(request.FeeNote), CalculatedAt: time.Now().UTC(),
				Events: events, TotalCostUSD: total, Members: members,
			},
		}
		payload, err := json.Marshal(bill.BillingSnapshot)
		if err != nil {
			return err
		}
		row := entities.BillingBill{
			AuthIndex: request.AuthIndex, Cycle: request.Cycle,
			WindowStart: timeutil.FormatSortableStorageTime(start), WindowEnd: timeutil.FormatSortableStorageTime(end),
			SnapshotJSON: string(payload),
		}
		if err := repository.SaveBillingBillSnapshot(tx, &row); err != nil {
			return err
		}
		bill.ID = row.ID
		return nil
	})
	if err != nil {
		return dto.BillingBill{}, err
	}
	return bill, nil
}

func (s *billingService) DeleteBillingBill(ctx context.Context, id int64) error {
	if id <= 0 || id > billingMaxSafeInteger {
		return fmt.Errorf("%w: invalid bill id", ErrBillingInvalidInput)
	}
	return repository.DeleteBillingBill(s.db.WithContext(ctx), id)
}

func validateBillingRequest(request dto.SaveBillingBillRequest) (time.Time, time.Time, int64, error) {
	invalid := func(message string) (time.Time, time.Time, int64, error) {
		return time.Time{}, time.Time{}, 0, fmt.Errorf("%w: %s", ErrBillingInvalidInput, message)
	}
	if strings.TrimSpace(request.AuthIndex) == "" || request.Cycle <= 0 || request.Cycle > billingMaxSafeInteger {
		return invalid("auth_index and a positive safe integer cycle are required")
	}
	if !billingTimePattern.MatchString(request.WindowStart) || !billingTimePattern.MatchString(request.WindowEnd) {
		return invalid("window timestamps must be RFC3339 with seconds and timezone")
	}
	start, startErr := time.Parse(time.RFC3339Nano, request.WindowStart)
	end, endErr := time.Parse(time.RFC3339Nano, request.WindowEnd)
	if startErr != nil || endErr != nil || !start.Before(end) {
		return invalid("window_end must be after a valid window_start (exclusive end)")
	}
	start, end = start.UTC(), end.UTC()
	if start.Year() < 0 || end.Year() > 9999 {
		return invalid("window is outside the RFC3339 UTC range")
	}
	if !billingFeePattern.MatchString(request.Fee) {
		return invalid("fee must be a positive yuan decimal with at most two decimal places")
	}
	parts := strings.SplitN(request.Fee, ".", 2)
	yuan, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || yuan > 999_999_999 {
		return invalid("fee must not exceed 999999999.99 yuan")
	}
	cents := yuan * 100
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
		value, _ := strconv.ParseInt(fraction, 10, 64)
		cents += value
	}
	if cents <= 0 {
		return invalid("fee must be greater than zero")
	}
	return start, end, cents, nil
}

func decodeBillingBill(row entities.BillingBill) (dto.BillingBill, error) {
	bill := dto.BillingBill{ID: row.ID, AuthIndex: row.AuthIndex, Cycle: row.Cycle}
	var err error
	bill.WindowStart, err = time.Parse(time.RFC3339Nano, row.WindowStart)
	if err != nil {
		return dto.BillingBill{}, fmt.Errorf("decode billing window: %w", err)
	}
	bill.WindowEnd, err = time.Parse(time.RFC3339Nano, row.WindowEnd)
	if err != nil {
		return dto.BillingBill{}, fmt.Errorf("decode billing window: %w", err)
	}
	if err := json.Unmarshal([]byte(row.SnapshotJSON), &bill.BillingSnapshot); err != nil {
		return dto.BillingBill{}, fmt.Errorf("decode billing snapshot: %w", err)
	}
	return bill, nil
}

func billingAccount(identity entities.UsageIdentity, redact func(string) string) dto.BillingAccount {
	name := strings.TrimSpace(identity.Name)
	if name == "" {
		name = identity.Identity
	}
	account := dto.BillingAccount{
		AuthIndex: identity.Identity, Name: redact(name), ActiveStart: identity.ActiveStart, ActiveUntil: identity.ActiveUntil,
	}
	if identity.Alias != nil {
		account.Alias = redact(*identity.Alias)
	}
	if identity.PlanType != nil {
		account.Plan = redact(*identity.PlanType)
	}
	return account
}

func billingRedactor(keys []entities.CPAAPIKey) func(string) string {
	// Longest keys first prevents a short known key from exposing the remainder
	// of a longer key. A Replacer does not recursively rewrite masked output.
	secrets := make([]string, 0, len(keys))
	for _, key := range keys {
		if key.APIKey != "" {
			secrets = append(secrets, key.APIKey)
		}
	}
	sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
	pairs := make([]string, 0, len(secrets)*2)
	for _, secret := range secrets {
		pairs = append(pairs, secret, helper.RedactSensitiveValue(secret))
	}
	replacer := strings.NewReplacer(pairs...)
	return func(value string) string {
		value = billingKeyPattern.ReplaceAllStringFunc(value, helper.RedactSensitiveValue)
		return replacer.Replace(value)
	}
}

type billingAccumulator struct {
	member       dto.BillingMember
	cost         float64
	compensation float64
	costUnits    int64
}

func calculateBillingUsage(tx *gorm.DB, resolver pricing.Resolver, keys []entities.CPAAPIKey, redact func(string) string, authIndex string, start, end time.Time, fee int64) ([]dto.BillingMember, int64, float64, error) {
	byKey := make(map[string]entities.CPAAPIKey, len(keys))
	for _, key := range keys {
		byKey[key.APIKey] = key
	}
	byMember := make(map[int64]*billingAccumulator)
	var events int64
	err := repository.StreamBillingUsage(tx, authIndex, start, end, func(event entities.UsageEvent) error {
		key, known := byKey[event.APIGroupKey]
		if event.APIGroupKey == "" || !known || key.ID <= 0 || key.ID > billingMaxSafeInteger {
			return fmt.Errorf("%w: an event has no known participant API key; sync key metadata before saving", ErrBillingIncomplete)
		}
		cost := resolver.Calculate(repository.UsageEventCostSubject(event))
		if !cost.Available || cost.MatchedModel == "" {
			return fmt.Errorf("%w: a model used in this window has no configured price", ErrBillingIncomplete)
		}
		if math.IsNaN(cost.Cost.TotalCostUSD) || math.IsInf(cost.Cost.TotalCostUSD, 0) || cost.Cost.TotalCostUSD < 0 {
			return fmt.Errorf("%w: invalid calculated usage cost", ErrBillingIncomplete)
		}
		member := byMember[key.ID]
		if member == nil {
			member = &billingAccumulator{member: dto.BillingMember{APIKeyID: key.ID, Name: redact(helper.CPAAPIKeyDisplayName(key))}}
			byMember[key.ID] = member
		}
		for _, counter := range []struct {
			target *int64
			value  int64
		}{
			{&events, 1}, {&member.member.Requests, 1},
			{&member.member.TotalTokens, event.TotalTokens}, {&member.member.InputTokens, event.InputTokens},
			{&member.member.OutputTokens, event.OutputTokens}, {&member.member.ReasoningTokens, event.ReasoningTokens},
			{&member.member.CacheReadTokens, event.CacheReadTokens},
		} {
			if counter.value < 0 || *counter.target > billingMaxSafeInteger-counter.value {
				return fmt.Errorf("%w: token or request counters exceed the supported exact range", ErrBillingIncomplete)
			}
			*counter.target += counter.value
		}
		if event.Failed {
			member.member.Failure++
		} else {
			member.member.Success++
		}
		// Compensated summation avoids a large history's accumulation drift
		// before the agreed per-member four-decimal weighting quantization.
		increment := cost.Cost.TotalCostUSD - member.compensation
		total := member.cost + increment
		member.compensation = (total - member.cost) - increment
		member.cost = total
		return nil
	})
	if err != nil {
		return nil, 0, 0, err
	}
	members := make([]billingAccumulator, 0, len(byMember))
	for _, member := range byMember {
		members = append(members, *member)
	}
	allocated, total, err := allocateBillingFee(members, fee)
	return allocated, events, total, err
}

func allocateBillingFee(members []billingAccumulator, fee int64) ([]dto.BillingMember, float64, error) {
	var totalUnits int64
	for i := range members {
		units := math.RoundToEven(members[i].cost * 10_000)
		if math.IsNaN(units) || math.IsInf(units, 0) || units < 0 || units > float64(billingMaxSafeInteger) {
			return nil, 0, fmt.Errorf("%w: usage cost exceeds the supported exact range", ErrBillingIncomplete)
		}
		members[i].costUnits = int64(units)
		if totalUnits > billingMaxSafeInteger-members[i].costUnits {
			return nil, 0, fmt.Errorf("%w: total usage cost exceeds the supported exact range", ErrBillingIncomplete)
		}
		totalUnits += members[i].costUnits
	}
	if totalUnits <= 0 {
		return nil, 0, fmt.Errorf("%w: no positive priced usage in this account window", ErrBillingIncomplete)
	}
	sort.Slice(members, func(i, j int) bool {
		if members[i].costUnits != members[j].costUnits {
			return members[i].costUnits > members[j].costUnits
		}
		return members[i].member.APIKeyID < members[j].member.APIKeyID
	})
	result := make([]dto.BillingMember, len(members))
	var numerator, denominator, quotient, remainder, multiplier big.Int
	denominator.SetInt64(totalUnits)
	multiplier.SetInt64(fee)
	var allocated int64
	for i := range members {
		member := members[i].member
		member.CostUSD = float64(members[i].costUnits) / 10_000
		member.Share = float64(members[i].costUnits) / float64(totalUnits)
		// Exact integer weights and half-even cents avoid binary-float penny
		// drift. Reused big.Int values avoid overflow in fee * weight.
		numerator.SetInt64(members[i].costUnits)
		numerator.Mul(&numerator, &multiplier)
		quotient.QuoRem(&numerator, &denominator, &remainder)
		remainder.Lsh(&remainder, 1)
		member.AmountCents = quotient.Int64()
		comparison := remainder.Cmp(&denominator)
		if comparison > 0 || (comparison == 0 && member.AmountCents%2 != 0) {
			member.AmountCents++
		}
		allocated += member.AmountCents
		result[i] = member
	}
	residual := fee - allocated
	if result[0].AmountCents+residual < 0 {
		return nil, 0, fmt.Errorf("%w: fee is too small to apply rounding residual to the highest-cost participant without a negative amount", ErrBillingIncomplete)
	}
	result[0].AmountCents += residual
	return result, float64(totalUnits) / 10_000, nil
}
