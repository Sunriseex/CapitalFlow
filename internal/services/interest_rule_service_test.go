package services

import (
	"testing"
	"time"

	"github.com/sunriseex/finance-manager/internal/models"
)

func TestInterestRuleServiceCreate(t *testing.T) {
	startDate := time.Date(2026, 5, 1, 12, 0, 0, 0, time.Local)

	rule, err := NewInterestRuleService(nil).Create(t.Context(), &CreateInterestRuleRequest{
		AccountID:               "account-1",
		AnnualRateBps:           1_200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyDaily,
		DayCountConvention:      models.DayCountConventionActual365,
		StartDate:               startDate,
	})
	if err != nil {
		t.Fatalf("create interest rule: %v", err)
	}
	if rule.ID == "" {
		t.Fatal("id is empty")
	}
	if rule.AccountID != "account-1" {
		t.Fatalf("account id = %s, want account-1", rule.AccountID)
	}
	if !rule.StartDate.Equal(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("start date = %s, want date only UTC", rule.StartDate)
	}
	if !rule.IsActive {
		t.Fatal("rule must be active")
	}
}

func TestInterestRuleServiceCreateDefaults(t *testing.T) {
	rule, err := NewInterestRuleService(nil).Create(t.Context(), &CreateInterestRuleRequest{
		AccountID:     "account-1",
		AnnualRateBps: 1_200,
	})
	if err != nil {
		t.Fatalf("create interest rule: %v", err)
	}
	if rule.AccrualFrequency != models.AccrualFrequencyDaily {
		t.Fatalf("accrual frequency = %s, want daily", rule.AccrualFrequency)
	}
	if rule.CapitalizationFrequency != models.CapitalizationFrequencyNone {
		t.Fatalf("capitalization frequency = %s, want none", rule.CapitalizationFrequency)
	}
	if rule.DayCountConvention != models.DayCountConventionActual365 {
		t.Fatalf("day count convention = %s, want actual_365", rule.DayCountConvention)
	}
}

func TestInterestRuleServiceCreateNormalizesDatePointers(t *testing.T) {
	promoRate := int64(2_400)
	promoEndDate := time.Date(2026, 5, 31, 23, 59, 59, 0, time.Local)
	endDate := time.Date(2026, 12, 31, 23, 59, 59, 0, time.Local)

	rule, err := NewInterestRuleService(nil).Create(t.Context(), &CreateInterestRuleRequest{
		AccountID:     "account-1",
		AnnualRateBps: 1_200,
		PromoRateBps:  &promoRate,
		PromoEndDate:  &promoEndDate,
		EndDate:       &endDate,
	})
	if err != nil {
		t.Fatalf("create interest rule: %v", err)
	}
	if rule.PromoEndDate == nil || rule.PromoEndDate.Format(time.RFC3339) != "2026-05-31T00:00:00Z" {
		t.Fatalf("promo end date = %v, want 2026-05-31 UTC date", rule.PromoEndDate)
	}
	if rule.EndDate == nil || rule.EndDate.Format(time.RFC3339) != "2026-12-31T00:00:00Z" {
		t.Fatalf("end date = %v, want 2026-12-31 UTC date", rule.EndDate)
	}
}

func TestInterestRuleServiceCreateRejectsIncompletePromo(t *testing.T) {
	promoRate := int64(2_400)
	promoEndDate := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		req  CreateInterestRuleRequest
	}{
		{
			name: "promo rate without end date",
			req: CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 1_200,
				PromoRateBps:  &promoRate,
			},
		},
		{
			name: "promo end date without rate",
			req: CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 1_200,
				PromoEndDate:  &promoEndDate,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewInterestRuleService(nil).Create(t.Context(), &tt.req)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestInterestRuleServiceAccrue(t *testing.T) {
	rule := models.InterestRule{
		ID:                 "rule-1",
		AccountID:          "account-1",
		AnnualRateBps:      1_200,
		AccrualFrequency:   models.AccrualFrequencyDaily,
		DayCountConvention: models.DayCountConventionActual365,
		IsActive:           true,
		StartDate:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	got, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule:         rule,
		BalanceMinor: 100_000_00,
		AccrualDate:  time.Date(2026, 5, 4, 15, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if got.Skipped {
		t.Fatal("accrual must not be skipped")
	}
	if got.Transaction == nil {
		t.Fatal("transaction is nil")
	}
	if got.Accrual == nil {
		t.Fatal("accrual is nil")
	}
	if got.Transaction.Type != models.TransactionTypeInterestIncome {
		t.Fatalf("transaction type = %s, want interest_income", got.Transaction.Type)
	}
	if got.Transaction.AmountMinor != 3_288 {
		t.Fatalf("amount = %d, want 3288", got.Transaction.AmountMinor)
	}
	if got.Transaction.OccurredAt.Format(time.DateOnly) != "2026-05-04" {
		t.Fatalf("occurred at = %s, want 2026-05-04", got.Transaction.OccurredAt.Format(time.DateOnly))
	}
	if got.Accrual.RuleID != "rule-1" {
		t.Fatalf("accrual rule id = %s, want rule-1", got.Accrual.RuleID)
	}
}

func TestInterestRuleServiceAccrueUsesPromoRate(t *testing.T) {
	promoRate := int64(2_400)
	promoEndDate := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	rule := models.InterestRule{
		ID:                 "rule-1",
		AccountID:          "account-1",
		AnnualRateBps:      1_200,
		PromoRateBps:       &promoRate,
		PromoEndDate:       &promoEndDate,
		AccrualFrequency:   models.AccrualFrequencyDaily,
		DayCountConvention: models.DayCountConventionActual365,
		IsActive:           true,
		StartDate:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	got, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule:         rule,
		BalanceMinor: 100_000_00,
		AccrualDate:  time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if got.Transaction.AmountMinor != 6_575 {
		t.Fatalf("amount = %d, want 6575", got.Transaction.AmountMinor)
	}
}

func TestInterestRuleServiceAccrueSkipsDuplicate(t *testing.T) {
	rule := models.InterestRule{
		ID:                 "rule-1",
		AccountID:          "account-1",
		AnnualRateBps:      1_200,
		AccrualFrequency:   models.AccrualFrequencyDaily,
		DayCountConvention: models.DayCountConventionActual365,
		IsActive:           true,
		StartDate:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	accrualDate := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)

	got, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule:         rule,
		BalanceMinor: 100_000_00,
		AccrualDate:  accrualDate,
		ExistingAccruals: []models.InterestAccrual{
			{
				AccountID:   "account-1",
				RuleID:      "rule-1",
				AccrualDate: accrualDate,
			},
		},
	})
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if !got.Skipped {
		t.Fatal("accrual must be skipped")
	}
	if got.Transaction != nil {
		t.Fatal("transaction must be nil")
	}
	if got.Accrual != nil {
		t.Fatal("accrual must be nil")
	}
}

func TestInterestRuleServiceAccrueValidatesRuleDate(t *testing.T) {
	rule := models.InterestRule{
		ID:                 "rule-1",
		AccountID:          "account-1",
		AnnualRateBps:      1_200,
		AccrualFrequency:   models.AccrualFrequencyDaily,
		DayCountConvention: models.DayCountConventionActual365,
		IsActive:           true,
		StartDate:          time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
	}

	_, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule:         rule,
		BalanceMinor: 100_000_00,
		AccrualDate:  time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInterestRuleServiceAccrueRejectsUnsupportedCapitalization(t *testing.T) {
	rule := models.InterestRule{
		ID:                      "rule-1",
		AccountID:               "account-1",
		AnnualRateBps:           1_200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyMonthly,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	_, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule:         rule,
		BalanceMinor: 100_000_00,
		AccrualDate:  time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
