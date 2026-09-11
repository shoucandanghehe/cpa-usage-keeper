package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"cpa-usage-keeper/internal/auth"
	"cpa-usage-keeper/internal/config"
	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/pricing"
	"cpa-usage-keeper/internal/repository"
	"cpa-usage-keeper/internal/repository/dto"
	"cpa-usage-keeper/internal/service"
)

func TestBillingHTTPAdminIsolationAndFailClosedSnapshots(t *testing.T) {
	db, err := repository.OpenDatabase(config.Config{SQLitePath: filepath.Join(t.TempDir(), "billing-http.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Fatal(err)
		}
	})
	secret := "custom-participant-secret-value"
	key := entities.CPAAPIKey{APIKey: secret, KeyAlias: "Member " + secret}
	if err := db.Create(&key).Error; err != nil {
		t.Fatal(err)
	}
	alias, plan := secret, "Pro20x sk-unlisted-display-secret"
	identity := entities.UsageIdentity{AuthType: entities.UsageIdentityAuthTypeAuthFile, Identity: "account-http", Name: "Account " + secret, Alias: &alias, PlanType: &plan}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	event := entities.UsageEvent{AuthIndex: identity.Identity, APIGroupKey: secret, Model: "priced", Timestamp: start, InputTokens: 1000000, TotalTokens: 1000000}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	snapshot, err := pricing.CompileSnapshot([]pricing.ModelConfig{{Pricing: entities.ModelPriceSetting{Model: "priced", PromptPricePer1M: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	catalog := pricing.NewCatalog(snapshot)
	sessions := auth.NewSessionManager(time.Hour)
	admin, _, err := sessions.Create()
	if err != nil {
		t.Fatal(err)
	}
	viewer, _, err := sessions.CreateAPIKeyViewer(key.ID)
	if err != nil {
		t.Fatal(err)
	}
	authConfig := AuthConfig{Enabled: true, LoginPassword: "test-password", SessionTTL: time.Hour, BasePath: "/keeper"}
	router := NewRouter(fstest.MapFS{"index.html": {Data: []byte("<html>billing-shell</html>")}}, nil, nil, nil, authConfig, NewAuthHandler(authConfig, sessions), "/keeper", OptionalProviders{
		Billing: service.NewBillingService(db, catalog), CPAAPIKeys: service.NewCPAAPIKeyService(db),
	})
	call := func(method, path, body, token string, intent bool) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, "/keeper"+path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		if intent {
			request.Header.Set(requestIntentHeaderName, requestIntentHeaderValueFetch)
		}
		if token != "" {
			request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	body := `{"auth_index":"account-http","cycle":1,"window_start":"2026-01-02T03:04:05Z","window_end":"2026-02-02T03:04:05Z","fee":"1.23","fee_note":"subscription"}`
	response := call(http.MethodPut, "/api/v1/billing/bills", body, admin, true)
	if response.Code != http.StatusOK {
		t.Fatalf("save failed: %d %s", response.Code, response.Body.String())
	}
	var bill dto.BillingBill
	if err := json.Unmarshal(response.Body.Bytes(), &bill); err != nil {
		t.Fatal(err)
	}
	if bill.Events != 1 || bill.FeeCents != 123 || len(bill.Members) != 1 || bill.Members[0].AmountCents != 123 || bill.Members[0].APIKeyID != key.ID {
		t.Fatalf("wrong bill contract: %+v", bill)
	}
	billPath := fmt.Sprintf("/api/v1/billing/bills/%d", bill.ID)
	for _, role := range []struct {
		name, token string
		status      int
	}{{"anonymous", "", 401}, {"key viewer", viewer, 403}} {
		for _, route := range []struct{ method, path string }{
			{http.MethodGet, "/api/v1/billing/accounts"}, {http.MethodGet, "/api/v1/billing/bills?auth_index=account-http"},
			{http.MethodPut, "/api/v1/billing/bills"}, {http.MethodDelete, billPath},
		} {
			denied := call(route.method, route.path, body, role.token, true)
			if denied.Code != role.status {
				t.Fatalf("%s %s %s: %d %s", role.name, route.method, route.path, denied.Code, denied.Body.String())
			}
		}
	}
	for _, route := range []struct{ method, path string }{{http.MethodPut, "/api/v1/billing/bills"}, {http.MethodDelete, billPath}} {
		if denied := call(route.method, route.path, body, admin, false); denied.Code != http.StatusForbidden {
			t.Fatalf("mutation without request intent accepted: %d", denied.Code)
		}
	}
	for _, path := range []string{"/api/v1/billing/accounts", "/api/v1/billing/bills?auth_index=account-http"} {
		response = call(http.MethodGet, path, "", admin, true)
		if response.Code != http.StatusOK || strings.Contains(response.Body.String(), secret) || strings.Contains(response.Body.String(), "sk-unlisted-display-secret") {
			t.Fatalf("billing exposed a raw display secret: %d %s", response.Code, response.Body.String())
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("billing response was cacheable")
		}
	}
	for _, input := range []struct {
		method, path, body string
		status             int
	}{
		{http.MethodGet, "/api/v1/billing/bills", "", 400},
		{http.MethodGet, "/api/v1/billing/bills?auth_index=missing", "", 404},
		{http.MethodPut, "/api/v1/billing/bills", strings.Replace(body, `"1.23"`, `1.23`, 1), 400},
		{http.MethodPut, "/api/v1/billing/bills", strings.Replace(body, `"cycle":1`, `"cycle":1.5`, 1), 400},
		{http.MethodPut, "/api/v1/billing/bills", strings.Replace(body, `"cycle":1`, `"cycle":2`, 1), 409},
		{http.MethodDelete, "/api/v1/billing/bills/not-an-id", "", 400},
		{http.MethodDelete, "/api/v1/billing/bills/99999", "", 404},
	} {
		response = call(input.method, input.path, input.body, admin, true)
		if response.Code != input.status {
			t.Fatalf("%s %s: %d %s", input.method, input.path, response.Code, response.Body.String())
		}
	}
	catalog.Replace(pricing.EmptySnapshot())
	response = call(http.MethodPut, "/api/v1/billing/bills", body, admin, true)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unpriced recalculation accepted: %d %s", response.Code, response.Body.String())
	}
	response = call(http.MethodGet, "/api/v1/billing/bills?auth_index=account-http", "", admin, true)
	var saved struct {
		Bills []dto.BillingBill `json:"bills"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Bills) != 1 || saved.Bills[0].ID != bill.ID || saved.Bills[0].FeeCents != 123 || saved.Bills[0].TotalCostUSD != 1 {
		t.Fatalf("failed or unauthorized mutation changed snapshot: %+v", saved.Bills)
	}
	response = call(http.MethodGet, "/billing", "", "", false)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "billing-shell") {
		t.Fatalf("billing deep link did not load SPA: %d %s", response.Code, response.Body.String())
	}
	response = call(http.MethodDelete, billPath, "", admin, true)
	if response.Code != http.StatusNoContent {
		t.Fatalf("admin delete failed: %d %s", response.Code, response.Body.String())
	}
	response = call(http.MethodGet, "/api/v1/billing/bills?auth_index=account-http", "", admin, true)
	if response.Code != http.StatusOK || response.Body.String() != `{"bills":[]}` {
		t.Fatalf("deleted bill still returned: %d %s", response.Code, response.Body.String())
	}
}
