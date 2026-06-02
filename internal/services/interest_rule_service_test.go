package services

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestCalculateDailyInterestAmountUsesCurrencyScale(t *testing.T) {
	date := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		currency string
		scale    int32
		want     string
	}{
		{name: "jpy rounds to whole units", currency: "JPY", scale: 0, want: "3"},
		{name: "kwd rounds to three decimals", currency: "KWD", scale: 3, want: "2.740"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateDailyInterestAmount(dec("10000"), 1000, models.DayCountConventionActual365, date, tt.currency)
			if got.StringFixed(tt.scale) != tt.want {
				t.Fatalf("got %s, want %s", got.StringFixed(tt.scale), tt.want)
			}
		})
	}
}

func TestInterestRuleServiceCreate(t *testing.T) {
	startDate := time.Date(2026, 5, 1, 12, 0, 0, 0, time.Local)

	rules := &recordingInterestRuleRepo{}
	rule, err := NewInterestRuleService(nil, WithInterestRuleRepository(rules)).Create(t.Context(), &CreateInterestRuleRequest{
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
	if rules.rule == nil || rules.rule.ID != rule.ID {
		t.Fatal("repo did not receive interest rule")
	}
}

func TestInterestRuleServiceCreateDefaults(t *testing.T) {
	rule, err := NewInterestRuleService(nil, WithInterestRuleRepository(&recordingInterestRuleRepo{})).Create(t.Context(), &CreateInterestRuleRequest{
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
	startDate := time.Date(2026, 5, 1, 23, 59, 59, 0, time.Local)
	promoEndDate := time.Date(2026, 5, 31, 23, 59, 59, 0, time.Local)
	endDate := time.Date(2026, 12, 31, 23, 59, 59, 0, time.Local)

	rule, err := NewInterestRuleService(nil, WithInterestRuleRepository(&recordingInterestRuleRepo{})).Create(t.Context(), &CreateInterestRuleRequest{
		AccountID:     "account-1",
		AnnualRateBps: 1_200,
		PromoRateBps:  &promoRate,
		PromoEndDate:  &promoEndDate,
		StartDate:     startDate,
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

func TestInterestRuleServiceCreateRejectsMissingRepository(t *testing.T) {
	_, err := NewInterestRuleService(nil).Create(t.Context(), &CreateInterestRuleRequest{
		AccountID:     "account-1",
		AnnualRateBps: 1_200,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if IsValidationError(err) {
		t.Fatalf("expected wiring error, got validation error: %v", err)
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
		Rule:        rule,
		Balance:     dec("100000"),
		AccrualDate: time.Date(2026, 5, 4, 15, 0, 0, 0, time.Local),
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
	if !got.Transaction.Amount.Equal(dec("32.88")) {
		t.Fatalf("amount = %d, want 3288", got.Transaction.Amount)
	}
	if got.Transaction.OccurredAt.Format(time.DateOnly) != "2026-05-04" {
		t.Fatalf("occurred at = %s, want 2026-05-04", got.Transaction.OccurredAt.Format(time.DateOnly))
	}
	if got.Accrual.RuleID != "rule-1" {
		t.Fatalf("accrual rule id = %s, want rule-1", got.Accrual.RuleID)
	}
}

func TestInterestRuleServiceAccruePersistsTransactionAndAccrualAtomically(t *testing.T) {
	accruals := &recordingInterestAccrualRepo{}
	transactions := &recordingTransactionRepo{}
	service := NewInterestRuleService(
		NewTransactionService(transactions),
		WithInterestAccrualRepository(accruals),
	)
	rule := models.InterestRule{
		ID:                 "rule-1",
		AccountID:          "account-1",
		AnnualRateBps:      1_200,
		AccrualFrequency:   models.AccrualFrequencyDaily,
		DayCountConvention: models.DayCountConventionActual365,
		IsActive:           true,
		StartDate:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	got, err := service.Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule:        rule,
		Balance:     dec("100000"),
		AccrualDate: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if got.Transaction == nil || got.Accrual == nil {
		t.Fatal("transaction and accrual must be returned")
	}
	if transactions.createCalls != 0 {
		t.Fatalf("transaction repo create calls = %d, want 0", transactions.createCalls)
	}
	if accruals.createCalls != 0 {
		t.Fatalf("accrual repo create calls = %d, want 0", accruals.createCalls)
	}
	if accruals.createWithTransactionCalls != 1 {
		t.Fatalf("atomic create calls = %d, want 1", accruals.createWithTransactionCalls)
	}
	if accruals.transaction == nil || accruals.accrual == nil {
		t.Fatal("atomic create must receive transaction and accrual")
	}
	if accruals.transaction.ID != got.Transaction.ID {
		t.Fatalf("atomic transaction id = %s, want %s", accruals.transaction.ID, got.Transaction.ID)
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
		Rule:        rule,
		Balance:     dec("100000"),
		AccrualDate: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if !got.Transaction.Amount.Equal(dec("65.75")) {
		t.Fatalf("amount = %d, want 6575", got.Transaction.Amount)
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
		Rule:        rule,
		Balance:     dec("100000"),
		AccrualDate: accrualDate,
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

func TestInterestRuleServiceAccrueDailyRuleUsesOnlyAccrualDate(t *testing.T) {
	rule := models.InterestRule{
		ID:                      "rule-1",
		AccountID:               "account-1",
		AnnualRateBps:           1_200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyDaily,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	got, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: rule.StartDate,
			},
		},
		AccrualDate: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if !got.Transaction.Amount.Equal(dec("32.88")) {
		t.Fatalf("amount = %d, want one daily accrual 3288", got.Transaction.Amount)
	}
}

func TestInterestRuleServiceAccrueBalanceOnlyMonthlyRuleUsesWholePeriod(t *testing.T) {
	rule := models.InterestRule{
		ID:                      "rule-1",
		AccountID:               "account-1",
		AnnualRateBps:           1_200,
		AccrualFrequency:        models.AccrualFrequencyMonthly,
		CapitalizationFrequency: models.CapitalizationFrequencyNone,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}

	got, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule:        rule,
		Balance:     dec("100000"),
		AccrualDate: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if !got.Transaction.Amount.Equal(dec("1019.28")) {
		t.Fatalf("amount = %d, want full May accrual", got.Transaction.Amount)
	}
}

func TestInterestRuleServiceAccrueDoesNotCapitalizeBeforePayableDayInterest(t *testing.T) {
	rule := models.InterestRule{
		ID:                      "rule-1",
		AccountID:               "account-1",
		AnnualRateBps:           1_200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyMonthly,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	transactions := []models.Transaction{
		{
			ID:         "initial",
			AccountID:  rule.AccountID,
			Type:       models.TransactionTypeInitialBalance,
			Amount:     dec("100000"),
			OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:         "may-01-interest",
			AccountID:  rule.AccountID,
			Type:       models.TransactionTypeInterestIncome,
			Amount:     dec("32.88"),
			OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	accruals := []models.InterestAccrual{
		{
			AccountID:     rule.AccountID,
			RuleID:        rule.ID,
			TransactionID: "may-01-interest",
			AccrualDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	got, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule:             rule,
		Transactions:     transactions,
		ExistingAccruals: accruals,
		AccrualDate:      time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if !got.Transaction.Amount.Equal(dec("32.88")) {
		t.Fatalf("amount = %d, want May 31 pre-capitalization daily accrual", got.Transaction.Amount)
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
		Rule:        rule,
		Balance:     dec("100000"),
		AccrualDate: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInterestRuleServiceAccrueAllowsMonthlyCapitalization(t *testing.T) {
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

	got, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule:        rule,
		Balance:     dec("100000"),
		AccrualDate: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}
	if got.Accrual == nil || !got.Accrual.Amount.Equal(dec("32.88")) {
		t.Fatalf("accrual = %+v, want daily rounded amount 3288", got.Accrual)
	}
}

func TestInterestRuleServiceRecalculateDefaultsRange(t *testing.T) {
	accruals := &recordingInterestAccrualRepo{}
	service := NewInterestRuleService(
		NewTransactionService(),
		WithInterestAccrualRepository(accruals),
	)
	rule := validAccrualTestRule()

	got, err := service.Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "tx-1",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: rule.StartDate,
			},
		},
		Today: time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}
	if got.FromDate.Format(time.DateOnly) != "2026-05-01" {
		t.Fatalf("from date = %s, want 2026-05-01", got.FromDate.Format(time.DateOnly))
	}
	if got.ToDate.Format(time.DateOnly) != "2026-05-03" {
		t.Fatalf("to date = %s, want 2026-05-03", got.ToDate.Format(time.DateOnly))
	}
	if got.CreatedAccruals != 3 {
		t.Fatalf("created accruals = %d, want 3", got.CreatedAccruals)
	}
}

func TestInterestRuleServiceRecalculateRejectsInvalidRange(t *testing.T) {
	_, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule:     validAccrualTestRule(),
		FromDate: time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
}

func TestInterestRuleServiceRecalculateReplacesExistingAccruals(t *testing.T) {
	accruals := &recordingInterestAccrualRepo{replaceDeleted: 1}
	service := NewInterestRuleService(
		NewTransactionService(),
		WithInterestAccrualRepository(accruals),
	)
	rule := validAccrualTestRule()

	got, err := service.Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ID:         "old-interest",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInterestIncome,
				Amount:     dec("999.99"),
				OccurredAt: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		ExistingAccruals: []models.InterestAccrual{
			{
				AccountID:     rule.AccountID,
				RuleID:        rule.ID,
				TransactionID: "old-interest",
				AccrualDate:   time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}
	if got.DeletedAccruals != 1 {
		t.Fatalf("deleted accruals = %d, want 1", got.DeletedAccruals)
	}
	if got.CreatedAccruals != 1 {
		t.Fatalf("created accruals = %d, want 1", got.CreatedAccruals)
	}
	if !got.TotalAmount.Equal(dec("32.88")) {
		t.Fatalf("total amount = %d, want 3288", got.TotalAmount)
	}
	if accruals.replaceCalls != 1 {
		t.Fatalf("replace calls = %d, want 1", accruals.replaceCalls)
	}
}

func TestInterestRuleServiceRecalculateSkipsNonPositiveBalanceDays(t *testing.T) {
	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: validAccrualTestRule(),
		Transactions: []models.Transaction{
			{
				ID:         "tx-1",
				AccountID:  "account-1",
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("0"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}
	if got.CreatedAccruals != 0 {
		t.Fatalf("created accruals = %d, want 0", got.CreatedAccruals)
	}
	if got.SkippedDays != 1 {
		t.Fatalf("skipped days = %d, want 1", got.SkippedDays)
	}
}

type recordingInterestAccrualRepo struct {
	createCalls                int
	createWithTransactionCalls int
	replaceCalls               int
	replaceDeleted             int64
	transaction                *models.Transaction
	accrual                    *models.InterestAccrual
}

type recordingInterestRuleRepo struct {
	rule *models.InterestRule
}

func (r *recordingInterestRuleRepo) Create(_ context.Context, rule *models.InterestRule) error {
	ruleCopy := *rule
	r.rule = &ruleCopy
	return nil
}

func (r *recordingInterestRuleRepo) GetByID(context.Context, string) (*models.InterestRule, error) {
	return nil, repository.ErrNotFound
}

func (r *recordingInterestRuleRepo) ListByAccount(context.Context, string) ([]models.InterestRule, error) {
	return nil, nil
}

func (r *recordingInterestRuleRepo) Update(context.Context, *models.InterestRule) error {
	return nil
}

func (r *recordingInterestAccrualRepo) Create(context.Context, *models.InterestAccrual) error {
	r.createCalls++
	return nil
}

func (r *recordingInterestAccrualRepo) CreateWithTransaction(_ context.Context, transaction *models.Transaction, accrual *models.InterestAccrual) error {
	r.createWithTransactionCalls++
	r.transaction = transaction
	r.accrual = accrual
	return nil
}

func (r *recordingInterestAccrualRepo) ReplaceRangeWithTransactions(_ context.Context, _, _ string, _, _ time.Time, transactions []models.Transaction, accruals []models.InterestAccrual) (int64, error) {
	r.replaceCalls++
	if len(transactions) > 0 {
		r.transaction = &transactions[0]
	}
	if len(accruals) > 0 {
		r.accrual = &accruals[0]
	}
	return r.replaceDeleted, nil
}

func (r *recordingInterestAccrualRepo) GetByAccountDateRule(context.Context, string, string, string) (*models.InterestAccrual, error) {
	return nil, errNotImplemented
}

func (r *recordingInterestAccrualRepo) ListByAccount(context.Context, string) ([]models.InterestAccrual, error) {
	return nil, nil
}

type recordingTransactionRepo struct {
	createCalls int
}

func (r *recordingTransactionRepo) Create(context.Context, *models.Transaction) error {
	r.createCalls++
	return nil
}

func (r *recordingTransactionRepo) CreateForUser(context.Context, string, *models.Transaction) error {
	return errNotImplemented
}

func (r *recordingTransactionRepo) CreateMany(context.Context, []models.Transaction) error {
	return nil
}

func (r *recordingTransactionRepo) CreateTransfer(context.Context, *models.Transfer, []models.Transaction) error {
	return nil
}

func (r *recordingTransactionRepo) ListTransfersByUser(context.Context, string) ([]models.Transfer, error) {
	return nil, nil
}

func (r *recordingTransactionRepo) GetByID(context.Context, string) (*models.Transaction, error) {
	return nil, errNotImplemented
}

func (r *recordingTransactionRepo) GetByIDForUser(context.Context, string, string) (*models.Transaction, error) {
	return nil, errNotImplemented
}

func (r *recordingTransactionRepo) List(context.Context) ([]models.Transaction, error) {
	return nil, nil
}

func (r *recordingTransactionRepo) ListByUser(context.Context, string) ([]models.Transaction, error) {
	return nil, nil
}

func (r *recordingTransactionRepo) ListByAccount(context.Context, string) ([]models.Transaction, error) {
	return nil, nil
}

func (r *recordingTransactionRepo) ListByAccountForUser(context.Context, string, string) ([]models.Transaction, error) {
	return nil, nil
}

func (r *recordingTransactionRepo) GetBalanceByAccountForUser(context.Context, string, string) (balance decimal.Decimal, transactionCount int64, err error) {
	return decimal.Zero, 0, nil
}

func TestInterestRuleServiceCreateReturnsValidationError(t *testing.T) {
	tests := []struct {
		name string
		req  *CreateInterestRuleRequest
	}{
		{
			name: "nil request",
			req:  nil,
		},
		{
			name: "missing account id",
			req: &CreateInterestRuleRequest{
				AnnualRateBps: 1200,
			},
		},
		{
			name: "zero annual rate",
			req: &CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 0,
			},
		},
		{
			name: "negative promo rate",
			req: &CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 1200,
				PromoRateBps:  ptrInt64(-100),
				PromoEndDate:  ptrTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
		{
			name: "promo rate without promo end date",
			req: &CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 1200,
				PromoRateBps:  ptrInt64(1500),
			},
		},
		{
			name: "promo end date without promo rate",
			req: &CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 1200,
				PromoEndDate:  ptrTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
		{
			name: "invalid accrual frequency",
			req: &CreateInterestRuleRequest{
				AccountID:        "account-1",
				AnnualRateBps:    1200,
				AccrualFrequency: models.AccrualFrequency("weekly"),
			},
		},
		{
			name: "invalid capitalization frequency",
			req: &CreateInterestRuleRequest{
				AccountID:               "account-1",
				AnnualRateBps:           1200,
				CapitalizationFrequency: models.CapitalizationFrequency("yearly"),
			},
		},
		{
			name: "invalid day count convention",
			req: &CreateInterestRuleRequest{
				AccountID:          "account-1",
				AnnualRateBps:      1200,
				DayCountConvention: models.DayCountConvention("30_360"),
			},
		},
		{
			name: "end date before start date",
			req: &CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 1200,
				StartDate:     time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
				EndDate:       ptrTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
		{
			name: "promo end date before start date",
			req: &CreateInterestRuleRequest{
				AccountID:     "account-1",
				AnnualRateBps: 1200,
				PromoRateBps:  ptrInt64(1500),
				StartDate:     time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
				PromoEndDate:  ptrTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewInterestRuleService(nil)

			_, err := service.Create(context.Background(), tt.req)
			if err == nil {
				t.Fatal("expected error")
			}

			if !IsValidationError(err) {
				t.Fatalf("expected validation error, got %T: %v", err, err)
			}
		})
	}
}

func TestInterestRuleServiceAccrueReturnsValidationError(t *testing.T) {
	tests := []struct {
		name string
		req  *AccrueRuleInterestRequest
	}{
		{"nil request", nil},
		{
			name: "missing rule id",
			req: &AccrueRuleInterestRequest{
				Rule: models.InterestRule{
					AccountID:               "account-1",
					IsActive:                true,
					AnnualRateBps:           1200,
					AccrualFrequency:        models.AccrualFrequencyDaily,
					CapitalizationFrequency: models.CapitalizationFrequencyNone,
					DayCountConvention:      models.DayCountConventionActual365,
					StartDate:               time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
				},
				Balance:     dec("1000"),
				AccrualDate: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "non-positive balance",
			req: &AccrueRuleInterestRequest{
				Rule:        validAccrualTestRule(),
				Balance:     dec("0"),
				AccrualDate: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "rule inactive",
			req: &AccrueRuleInterestRequest{
				Rule: func() models.InterestRule {
					rule := validAccrualTestRule()
					rule.IsActive = false
					return rule
				}(),
				Balance:     dec("1000"),
				AccrualDate: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewInterestRuleService(NewTransactionService())

			_, err := service.Accrue(context.Background(), tt.req)
			if err == nil {
				t.Fatal("expected error")
			}

			if !IsValidationError(err) {
				t.Fatalf("expected validation error, got %T: %v", err, err)
			}
		})
	}
}

func TestInterestRuleServiceRecalculateAllowsInactiveRuleCleanup(t *testing.T) {
	rule := validAccrualTestRule()
	rule.IsActive = false

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		ExistingAccruals: []models.InterestAccrual{
			{
				AccountID:     rule.AccountID,
				RuleID:        rule.ID,
				TransactionID: "old-interest",
				AccrualDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}

	if got.CreatedAccruals != 1 {
		t.Fatalf("created accruals = %d, want 1", got.CreatedAccruals)
	}
}

func TestInterestRuleServiceRecalculateDoesNotUsePriorAccrualsWhenCapitalizationNone(t *testing.T) {
	rule := validAccrualTestRule()
	rule.CapitalizationFrequency = models.CapitalizationFrequencyNone

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ID:         "prior-interest",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInterestIncome,
				Amount:     dec("32.88"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		ExistingAccruals: []models.InterestAccrual{
			{
				AccountID:     rule.AccountID,
				RuleID:        rule.ID,
				TransactionID: "prior-interest",
				AccrualDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}

	if got.CreatedAccruals != 1 {
		t.Fatalf("created accruals = %d, want 1", got.CreatedAccruals)
	}

	if len(got.Transactions) != 1 {
		t.Fatalf("transactions len = %d, want 1", len(got.Transactions))
	}

	if !got.Transactions[0].Amount.Equal(dec("32.88")) {
		t.Fatalf("amount = %d, want 3288 without prior interest compounding", got.Transactions[0].Amount)
	}
}

func TestInterestRuleServiceAccrueExcludesUncapitalizedPriorAccruals(t *testing.T) {
	rule := models.InterestRule{
		ID:                      "rule-1",
		AccountID:               "account-1",
		AnnualRateBps:           1_200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyMonthly,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}

	priorInterestDate := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)

	got, err := NewInterestRuleService(nil).Accrue(t.Context(), &AccrueRuleInterestRequest{
		Rule:        rule,
		AccrualDate: time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: rule.StartDate,
			},
			{
				ID:         "prior-interest",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInterestIncome,
				Amount:     dec("32.88"),
				OccurredAt: priorInterestDate,
			},
		},
		ExistingAccruals: []models.InterestAccrual{
			{
				AccountID:     rule.AccountID,
				RuleID:        rule.ID,
				TransactionID: "prior-interest",
				AccrualDate:   priorInterestDate,
				Amount:        dec("32.88"),
			},
		},
	})
	if err != nil {
		t.Fatalf("accrue interest: %v", err)
	}

	if !got.Transaction.Amount.Equal(dec("32.88")) {
		t.Fatalf("amount = %d, want 3288 without early capitalization", got.Transaction.Amount)
	}
}

func TestInterestRuleServiceForecastProjectedBalanceIgnoresTransactionsAfterHorizon(t *testing.T) {
	rule := models.InterestRule{
		ID:                      "rule-1",
		AccountID:               "account-1",
		AnnualRateBps:           1_200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyNone,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}

	got, err := NewInterestRuleService(nil).Forecast(t.Context(), &ForecastRuleInterestRequest{
		Rule:     rule,
		FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		Days:     30,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ID:         "future-expense",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeExpense,
				Amount:     dec("50000"),
				OccurredAt: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	})
	if err != nil {
		t.Fatalf("forecast interest: %v", err)
	}

	if !got.ProjectedBalance.GreaterThan(dec("50000")) {
		t.Fatalf("projected balance = %d, future expense after horizon was included", got.ProjectedBalance)
	}
}

func TestInterestRuleServiceRecalculateCompoundsWhenCapitalizationDaily(t *testing.T) {
	rule := validAccrualTestRule()
	rule.CapitalizationFrequency = models.CapitalizationFrequencyDaily

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}

	if got.CreatedAccruals != 3 {
		t.Fatalf("created accruals = %d, want 3", got.CreatedAccruals)
	}

	if !got.Transactions[0].Amount.Equal(dec("32.88")) {
		t.Fatalf("day 1 amount = %d, want 3288", got.Transactions[0].Amount)
	}
	if !got.Transactions[1].Amount.GreaterThan(got.Transactions[0].Amount) {
		t.Fatalf("day 2 amount = %d, want more than day 1 due to daily capitalization", got.Transactions[1].Amount)
	}
}

func TestInterestRuleServiceRecalculateDoesNotCompoundWhenCapitalizationNone(t *testing.T) {
	rule := validAccrualTestRule()
	rule.CapitalizationFrequency = models.CapitalizationFrequencyNone

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}

	if got.CreatedAccruals != 3 {
		t.Fatalf("created accruals = %d, want 3", got.CreatedAccruals)
	}

	for _, tx := range got.Transactions {
		if !tx.Amount.Equal(dec("32.88")) {
			t.Fatalf("amount = %d, want 3288 without compounding", tx.Amount)
		}
	}
}

func TestInterestRuleServiceRecalculateMonthlyAccrual(t *testing.T) {
	rule := validAccrualTestRule()
	rule.AccrualFrequency = models.AccrualFrequencyMonthly
	rule.CapitalizationFrequency = models.CapitalizationFrequencyNone

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}
	if got.CreatedAccruals != 1 {
		t.Fatalf("created accruals = %d, want 1 monthly posting", got.CreatedAccruals)
	}
	if !got.TotalAmount.Equal(dec("1019.28")) {
		t.Fatalf("total amount = %d, want 101928", got.TotalAmount)
	}
	if got.Accruals[0].AccrualDate.Format(time.DateOnly) != "2026-05-31" {
		t.Fatalf("accrual date = %s, want month end", got.Accruals[0].AccrualDate.Format(time.DateOnly))
	}
}

func TestInterestRuleServiceRecalculateMonthlyAccrualExpandsPartialPayableRange(t *testing.T) {
	rule := validAccrualTestRule()
	rule.AccrualFrequency = models.AccrualFrequencyMonthly
	rule.CapitalizationFrequency = models.CapitalizationFrequencyNone

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		ExistingAccruals: []models.InterestAccrual{
			{
				AccountID:     rule.AccountID,
				RuleID:        rule.ID,
				TransactionID: "old-may-interest",
				AccrualDate:   time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}
	if got.CreatedAccruals != 1 {
		t.Fatalf("created accruals = %d, want 1 monthly posting", got.CreatedAccruals)
	}
	if !got.TotalAmount.Equal(dec("1019.28")) {
		t.Fatalf("total amount = %d, want full May accrual", got.TotalAmount)
	}
}

func TestInterestRuleServiceRecalculateFlushesPendingOnPayableDateWithZeroBalance(t *testing.T) {
	rule := validAccrualTestRule()
	rule.AccrualFrequency = models.AccrualFrequencyMonthly
	rule.CapitalizationFrequency = models.CapitalizationFrequencyNone

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ID:         "withdrawal",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeExpense,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}
	if got.CreatedAccruals != 1 {
		t.Fatalf("created accruals = %d, want 1 monthly posting", got.CreatedAccruals)
	}
	if !got.TotalAmount.Equal(dec("986.4")) {
		t.Fatalf("total amount = %d, want 30 earned days before withdrawal", got.TotalAmount)
	}
}

func TestInterestRuleServiceRecalculateCapitalizesFullMonthlyPendingPeriod(t *testing.T) {
	rule := validAccrualTestRule()
	rule.AccrualFrequency = models.AccrualFrequencyDaily
	rule.CapitalizationFrequency = models.CapitalizationFrequencyMonthly

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}
	if got.CreatedAccruals != 32 {
		t.Fatalf("created accruals = %d, want 32 daily postings", got.CreatedAccruals)
	}
	if !got.Transactions[30].Amount.Equal(dec("32.88")) {
		t.Fatalf("may 31 amount = %d, want uncapitalized daily amount 3288", got.Transactions[30].Amount)
	}
	if !got.Transactions[31].Amount.GreaterThan(got.Transactions[30].Amount) {
		t.Fatalf("jun 1 amount = %d, want higher after monthly capitalization", got.Transactions[31].Amount)
	}
}

func TestInterestRuleServiceRecalculateRestoresPreRangeAccrualsBeforeCapitalization(t *testing.T) {
	rule := validAccrualTestRule()
	rule.AccrualFrequency = models.AccrualFrequencyDaily
	rule.CapitalizationFrequency = models.CapitalizationFrequencyMonthly
	mayFirstInterest := models.Transaction{
		ID:         "may-01-interest",
		AccountID:  rule.AccountID,
		Type:       models.TransactionTypeInterestIncome,
		Amount:     dec("32.88"),
		OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	mayFourteenthInterest := models.Transaction{
		ID:         "may-14-interest",
		AccountID:  rule.AccountID,
		Type:       models.TransactionTypeInterestIncome,
		Amount:     dec("32.88"),
		OccurredAt: time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC),
	}

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
			mayFirstInterest,
			mayFourteenthInterest,
		},
		ExistingAccruals: []models.InterestAccrual{
			{
				AccountID:     rule.AccountID,
				RuleID:        rule.ID,
				TransactionID: mayFirstInterest.ID,
				AccrualDate:   mayFirstInterest.OccurredAt,
			},
			{
				AccountID:     rule.AccountID,
				RuleID:        rule.ID,
				TransactionID: mayFourteenthInterest.ID,
				AccrualDate:   mayFourteenthInterest.OccurredAt,
			},
		},
		FromDate: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}
	if got.CreatedAccruals != 18 {
		t.Fatalf("created accruals = %d, want May 15 through Jun 1", got.CreatedAccruals)
	}

	if !got.Transactions[17].Amount.IsPositive() {
		t.Fatalf("jun 1 amount = %s, want positive amount with pre-range accruals capitalized", got.Transactions[17].Amount)
	}
}

func TestInterestRuleServiceRecalculateEndOfTermAccrual(t *testing.T) {
	rule := validAccrualTestRule()
	rule.AccrualFrequency = models.AccrualFrequencyEndOfTerm
	rule.CapitalizationFrequency = models.CapitalizationFrequencyEndOfTerm
	rule.EndDate = ptrTime(time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC))

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}
	if got.CreatedAccruals != 1 {
		t.Fatalf("created accruals = %d, want 1 end-of-term posting", got.CreatedAccruals)
	}
	if !got.TotalAmount.Equal(dec("98.64")) {
		t.Fatalf("total amount = %d, want 9864", got.TotalAmount)
	}
}

func TestInterestRuleServiceRecalculateSplitsPromoRate(t *testing.T) {
	rule := validAccrualTestRule()
	promoRate := int64(2400)
	rule.PromoRateBps = &promoRate
	rule.PromoEndDate = ptrTime(time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC))

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}
	if !got.TotalAmount.Equal(dec("98.63")) {
		t.Fatalf("total amount = %d, want promo day 6575 + base day 3288", got.TotalAmount)
	}
}

func TestInterestRuleServiceRecalculateTenPercentSavings(t *testing.T) {
	rule := validAccrualTestRule()
	rule.AnnualRateBps = 1_000

	got, err := NewInterestRuleService(nil).Recalculate(t.Context(), &RecalculateRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("recalculate interest: %v", err)
	}
	if !got.TotalAmount.Equal(dec("27.4")) {
		t.Fatalf("total amount = %d, want daily rounded 10%% amount 2740", got.TotalAmount)
	}
}

func TestInterestRuleServiceForecastSupportsCommonRanges(t *testing.T) {
	rule := validAccrualTestRule()
	rule.CapitalizationFrequency = models.CapitalizationFrequencyDaily
	transactions := []models.Transaction{
		{
			ID:         "initial",
			AccountID:  rule.AccountID,
			Type:       models.TransactionTypeInitialBalance,
			Amount:     dec("100000"),
			OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, days := range []int{30, 90, 365} {
		t.Run(time.Duration(days*24).String(), func(t *testing.T) {
			got, err := NewInterestRuleService(nil).Forecast(t.Context(), &ForecastRuleInterestRequest{
				Rule:         rule,
				Transactions: transactions,
				FromDate:     time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
				Days:         days,
			})
			if err != nil {
				t.Fatalf("forecast interest: %v", err)
			}
			if got.Days != days {
				t.Fatalf("days = %d, want %d", got.Days, days)
			}
			if len(got.Accruals) != days {
				t.Fatalf("accruals len = %d, want %d", len(got.Accruals), days)
			}
			if !got.ProjectedAmount.IsPositive() || !got.ProjectedBalance.GreaterThan(transactions[0].Amount) {
				t.Fatalf("projected minor = %d, projected balance = %d", got.ProjectedAmount, got.ProjectedBalance)
			}
		})
	}
}

func TestInterestRuleServiceForecastIgnoresUncapitalizedAndFutureTransactions(t *testing.T) {
	rule := validAccrualTestRule()
	rule.CapitalizationFrequency = models.CapitalizationFrequencyNone

	got, err := NewInterestRuleService(nil).Forecast(t.Context(), &ForecastRuleInterestRequest{
		Rule: rule,
		Transactions: []models.Transaction{
			{
				ID:         "initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ID:         "prior-interest",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInterestIncome,
				Amount:     dec("32.88"),
				OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ID:         "future-income",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeIncome,
				Amount:     dec("50000"),
				OccurredAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		ExistingAccruals: []models.InterestAccrual{
			{
				AccountID:     rule.AccountID,
				RuleID:        rule.ID,
				TransactionID: "prior-interest",
				AccrualDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		FromDate: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
		Days:     1,
	})
	if err != nil {
		t.Fatalf("forecast interest: %v", err)
	}
	if !got.ProjectedAmount.Equal(dec("32.88")) {
		t.Fatalf("projected minor = %d, want one day from initial principal", got.ProjectedAmount)
	}
	if !got.ProjectedBalance.Equal(dec("100032.88")) {
		t.Fatalf("projected balance = %d, want horizon balance without prior accrual or future income", got.ProjectedBalance)
	}
}

func TestPrincipalTransactionsForRuleAtIncludesClosedMonthlyCapitalizationPeriod(t *testing.T) {
	rule := validAccrualTestRule()
	rule.CapitalizationFrequency = models.CapitalizationFrequencyMonthly

	transactions := []models.Transaction{
		{
			ID:         "initial",
			AccountID:  rule.AccountID,
			Type:       models.TransactionTypeInitialBalance,
			Amount:     dec("100000"),
			OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:         "may-01-interest",
			AccountID:  rule.AccountID,
			Type:       models.TransactionTypeInterestIncome,
			Amount:     dec("32.88"),
			OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:         "may-31-interest",
			AccountID:  rule.AccountID,
			Type:       models.TransactionTypeInterestIncome,
			Amount:     dec("32.88"),
			OccurredAt: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:         "jun-01-interest",
			AccountID:  rule.AccountID,
			Type:       models.TransactionTypeInterestIncome,
			Amount:     dec("33.21"),
			OccurredAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	accruals := []models.InterestAccrual{
		{
			AccountID:     rule.AccountID,
			RuleID:        rule.ID,
			TransactionID: "may-01-interest",
			AccrualDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			AccountID:     rule.AccountID,
			RuleID:        rule.ID,
			TransactionID: "may-31-interest",
			AccrualDate:   time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			AccountID:     rule.AccountID,
			RuleID:        rule.ID,
			TransactionID: "jun-01-interest",
			AccrualDate:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	got := PrincipalTransactionsForRuleAt(transactions, accruals, &rule, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))

	if len(got) != 3 {
		t.Fatalf("len = %d, want initial plus two closed May accruals", len(got))
	}
	for _, tx := range got {
		if tx.ID == "jun-01-interest" {
			t.Fatal("current open month accrual must not be principal")
		}
	}
}

func validAccrualTestRule() models.InterestRule {
	return models.InterestRule{
		ID:                      "rule-1",
		AccountID:               "account-1",
		IsActive:                true,
		AnnualRateBps:           1200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyNone,
		DayCountConvention:      models.DayCountConventionActual365,
		StartDate:               time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
}

func ptrInt64(value int64) *int64 {
	return &value
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
