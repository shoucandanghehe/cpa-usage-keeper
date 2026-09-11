package dto

import "time"

type BillingAccount struct {
	AuthIndex   string     `json:"auth_index"`
	Name        string     `json:"name"`
	Alias       string     `json:"alias"`
	Plan        string     `json:"plan"`
	ActiveStart *time.Time `json:"active_start"`
	ActiveUntil *time.Time `json:"active_until"`
}

type BillingMember struct {
	APIKeyID        int64   `json:"api_key_id"`
	Name            string  `json:"name"`
	Requests        int64   `json:"requests"`
	Success         int64   `json:"success"`
	Failure         int64   `json:"failure"`
	TotalTokens     int64   `json:"total_tokens"`
	InputTokens     int64   `json:"input_tokens"`
	OutputTokens    int64   `json:"output_tokens"`
	ReasoningTokens int64   `json:"reasoning_tokens"`
	CacheReadTokens int64   `json:"cache_read_tokens"`
	CostUSD         float64 `json:"cost_usd"`
	Share           float64 `json:"share"`
	AmountCents     int64   `json:"amount_cents"`
}

// BillingSnapshot is written only on explicit save, never on metadata/price sync.
// Indexed metadata is not duplicated in this JSON payload.
type BillingSnapshot struct {
	Account      string          `json:"account"`
	Alias        string          `json:"alias"`
	Plan         string          `json:"plan"`
	FeeCents     int64           `json:"fee_cents"`
	FeeNote      string          `json:"fee_note"`
	CalculatedAt time.Time       `json:"calculated_at"`
	Events       int64           `json:"events"`
	TotalCostUSD float64         `json:"total_cost_usd"`
	Members      []BillingMember `json:"members"`
}

type BillingBill struct {
	ID          int64     `json:"id"`
	AuthIndex   string    `json:"auth_index"`
	Cycle       int64     `json:"cycle"`
	WindowStart time.Time `json:"window_start"`
	WindowEnd   time.Time `json:"window_end"`
	BillingSnapshot
}

type SaveBillingBillRequest struct {
	AuthIndex   string `json:"auth_index"`
	Cycle       int64  `json:"cycle"`
	WindowStart string `json:"window_start"`
	WindowEnd   string `json:"window_end"`
	Fee         string `json:"fee"`
	FeeNote     string `json:"fee_note"`
}
