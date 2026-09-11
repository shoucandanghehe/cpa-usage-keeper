package service

import (
	"errors"
	"math"
	"testing"

	"cpa-usage-keeper/internal/repository/dto"
)

func TestBillingAllocationMatchesHistoricalStatement(t *testing.T) {
	costs := []float64{10843.5376, 6516.6671, 4804.5137, 1251.1443, 112.5226, 3.8689}
	members := make([]billingAccumulator, len(costs))
	for i, cost := range costs {
		members[i] = billingAccumulator{cost: cost, member: dto.BillingMember{APIKeyID: int64(i + 1)}}
	}
	got, total, err := allocateBillingFee(members, 84122)
	if err != nil {
		t.Fatal(err)
	}
	want := []int64{38763, 23295, 17175, 4473, 402, 14}
	var sum int64
	var shares float64
	for i, member := range got {
		if member.AmountCents != want[i] || member.CostUSD != costs[i] {
			t.Fatalf("participant %d: got %+v, want cost %v and cents %d", i, member, costs[i], want[i])
		}
		sum += member.AmountCents
		shares += member.Share
	}
	if sum != 84122 || math.Abs(shares-1) > 1e-12 || math.Abs(total-23532.2542) > 1e-8 {
		t.Fatalf("incorrect totals: cents %d shares %v cost %v", sum, shares, total)
	}
}

func TestBillingAllocationHalfEvenAndStableResidual(t *testing.T) {
	members := []billingAccumulator{
		{cost: 1, member: dto.BillingMember{APIKeyID: 9}},
		{cost: 1, member: dto.BillingMember{APIKeyID: 2}},
	}
	got, _, err := allocateBillingFee(members, 5)
	if err != nil {
		t.Fatal(err)
	}
	// Both 2.5-cent shares round to even 2; the one-cent residual goes to
	// the lowest stable key ID, not iteration order or a display alias.
	if got[0].APIKeyID != 2 || got[0].AmountCents != 3 || got[1].APIKeyID != 9 || got[1].AmountCents != 2 {
		t.Fatalf("unexpected half-even allocation: %+v", got)
	}
	quantized, _, err := allocateBillingFee([]billingAccumulator{
		{cost: .00025, member: dto.BillingMember{APIKeyID: 1}},
		{cost: .00035, member: dto.BillingMember{APIKeyID: 2}},
	}, 6)
	if err != nil {
		t.Fatal(err)
	}
	if quantized[0].CostUSD != .0004 || quantized[0].AmountCents != 4 || quantized[1].CostUSD != .0002 || quantized[1].AmountCents != 2 {
		t.Fatalf("per-member four-decimal quantization: %+v", quantized)
	}
}

func TestBillingAllocationRejectsNegativeResidualAndZeroWeight(t *testing.T) {
	members := make([]billingAccumulator, 7)
	for i := range members {
		members[i] = billingAccumulator{cost: 1, member: dto.BillingMember{APIKeyID: int64(i + 1)}}
	}
	if _, _, err := allocateBillingFee(members, 4); !errors.Is(err, ErrBillingIncomplete) {
		t.Fatalf("expected impossible nonnegative residual to fail closed, got %v", err)
	}
	for _, members := range [][]billingAccumulator{nil, {{cost: .00001}}, {{cost: math.Inf(1)}}} {
		if _, _, err := allocateBillingFee(members, 10); !errors.Is(err, ErrBillingIncomplete) {
			t.Fatalf("expected invalid weights to fail, got %v", err)
		}
	}
}

func TestBillingRequestValidatesExactFeeAndWindow(t *testing.T) {
	valid := dto.SaveBillingBillRequest{
		AuthIndex: "account-a", Cycle: billingMaxSafeInteger,
		WindowStart: "2026-01-01T10:30:00.123456789+08:00", WindowEnd: "2026-01-01T02:30:00.123456790Z",
		Fee: "999999999.99",
	}
	start, end, fee, err := validateBillingRequest(valid)
	if err != nil || fee != 99999999999 || end.Sub(start).Nanoseconds() != 1 {
		t.Fatalf("exact fee/window rejected: %v %d %s %s", err, fee, start, end)
	}
	for _, change := range []struct {
		name  string
		apply func(*dto.SaveBillingBillRequest)
	}{
		{"blank account", func(r *dto.SaveBillingBillRequest) { r.AuthIndex = " " }},
		{"unsafe cycle", func(r *dto.SaveBillingBillRequest) { r.Cycle++ }},
		{"zero cycle", func(r *dto.SaveBillingBillRequest) { r.Cycle = 0 }},
		{"no seconds", func(r *dto.SaveBillingBillRequest) { r.WindowStart = "2026-01-01T02:30Z" }},
		{"invalid offset", func(r *dto.SaveBillingBillRequest) { r.WindowStart = "2026-01-01T02:30:00+24:00" }},
		{"empty window", func(r *dto.SaveBillingBillRequest) { r.WindowEnd = r.WindowStart }},
		{"reversed window", func(r *dto.SaveBillingBillRequest) { r.WindowEnd = "2025-12-31T00:00:00Z" }},
		{"zero fee", func(r *dto.SaveBillingBillRequest) { r.Fee = "0.00" }},
		{"negative fee", func(r *dto.SaveBillingBillRequest) { r.Fee = "-1" }},
		{"exponent", func(r *dto.SaveBillingBillRequest) { r.Fee = "1e2" }},
		{"fractional cents", func(r *dto.SaveBillingBillRequest) { r.Fee = "1.001" }},
		{"too large", func(r *dto.SaveBillingBillRequest) { r.Fee = "1000000000" }},
	} {
		t.Run(change.name, func(t *testing.T) {
			request := valid
			change.apply(&request)
			if _, _, _, err := validateBillingRequest(request); !errors.Is(err, ErrBillingInvalidInput) {
				t.Fatalf("invalid input accepted: %+v, %v", request, err)
			}
		})
	}
}
