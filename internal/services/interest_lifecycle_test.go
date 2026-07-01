package services

import (
	"context"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestInterestLifecycleAccruePersistsInsideLock(t *testing.T) {
	t.Parallel()

	date := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	rule := lifecycleRule(date.AddDate(0, 0, -1))
	snapshot := &lifecycleSnapshot{
		rules: []models.InterestRule{rule},
		transactions: []models.Transaction{{
			ID:         "principal",
			AccountID:  rule.AccountID,
			Type:       models.TransactionTypeIncome,
			Amount:     dec("100"),
			OccurredAt: date.AddDate(0, 0, -1),
		}},
	}
	lifecycle := newTestInterestLifecycle(&lifecycleRepo{snapshot: snapshot})

	result, err := lifecycle.Accrue(t.Context(), &AccrueAccountInterestRequest{
		AccountID:   rule.AccountID,
		UserID:      "user-1",
		Currency:    "RUB",
		RuleID:      rule.ID,
		AccrualDate: date,
	})
	if err != nil {
		t.Fatalf("accrue: %v", err)
	}
	if result.Skipped || len(snapshot.createdAccruals) != 1 || len(snapshot.createdTransactions) != 1 {
		t.Fatalf("result=%+v transactions=%d accruals=%d", result, len(snapshot.createdTransactions), len(snapshot.createdAccruals))
	}
	transaction := snapshot.createdTransactions[0]
	if transaction.Description != "Проценты по вкладу Накопительный счёт" {
		t.Fatalf("description = %q", transaction.Description)
	}
	if transaction.CategoryID == nil || *transaction.CategoryID != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("category id = %v", transaction.CategoryID)
	}
}

func TestInterestLifecycleAccrueTreatsConflictAsSkipped(t *testing.T) {
	t.Parallel()

	date := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	rule := lifecycleRule(date.AddDate(0, 0, -1))
	repo := &lifecycleRepo{snapshot: &lifecycleSnapshot{
		rules: []models.InterestRule{rule},
		transactions: []models.Transaction{{
			ID: "principal", AccountID: rule.AccountID, Type: models.TransactionTypeIncome,
			Amount: dec("100"), OccurredAt: date.AddDate(0, 0, -1),
		}},
		createErr: repository.ErrConflict,
	}}

	result, err := newTestInterestLifecycle(repo).Accrue(t.Context(), &AccrueAccountInterestRequest{
		AccountID: rule.AccountID, UserID: "user-1", Currency: "RUB", RuleID: rule.ID, AccrualDate: date,
	})
	if err != nil {
		t.Fatalf("accrue: %v", err)
	}
	if !result.Skipped || !repo.rolledBack {
		t.Fatalf("result=%+v rolledBack=%t", result, repo.rolledBack)
	}
}

func TestInterestLifecycleAccrueSelectsLatestActiveRule(t *testing.T) {
	t.Parallel()

	date := time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC)
	oldRule := lifecycleRule(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))
	oldRule.ID = "11111111-2222-3333-4444-555555555555"
	oldRule.EndDate = new(time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC))
	currentRule := lifecycleRule(time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC))
	snapshot := &lifecycleSnapshot{
		rules: []models.InterestRule{oldRule, currentRule},
		transactions: []models.Transaction{{
			ID: "principal", AccountID: currentRule.AccountID, Type: models.TransactionTypeIncome,
			Amount: dec("100"), OccurredAt: date.AddDate(0, 0, -1),
		}},
	}

	result, err := newTestInterestLifecycle(&lifecycleRepo{snapshot: snapshot}).Accrue(t.Context(), &AccrueAccountInterestRequest{
		AccountID: currentRule.AccountID, UserID: "user-1", Currency: "RUB", AccrualDate: date,
	})
	if err != nil {
		t.Fatalf("accrue: %v", err)
	}
	if result.Accrual.RuleID != currentRule.ID {
		t.Fatalf("rule id = %s, want %s", result.Accrual.RuleID, currentRule.ID)
	}
}

func TestInterestLifecycleRecalculateReplacesInsideLock(t *testing.T) {
	t.Parallel()

	date := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	rule := lifecycleRule(date.AddDate(0, 0, -1))
	snapshot := &lifecycleSnapshot{
		rules: []models.InterestRule{rule},
		transactions: []models.Transaction{{
			ID: "principal", AccountID: rule.AccountID, Type: models.TransactionTypeIncome,
			Amount: dec("100"), OccurredAt: date.AddDate(0, 0, -1),
		}},
		deleted: 2,
	}

	result, err := newTestInterestLifecycle(&lifecycleRepo{snapshot: snapshot}).Recalculate(t.Context(), &RecalculateAccountInterestRequest{
		AccountID: rule.AccountID, UserID: "user-1", Currency: "RUB", RuleID: rule.ID,
		RuleDate: date, FromDate: date, ToDate: date,
	})
	if err != nil {
		t.Fatalf("recalculate: %v", err)
	}
	if !snapshot.replaced || result.DeletedAccruals != 2 || result.CreatedAccruals != 1 {
		t.Fatalf("result=%+v replaced=%t", result, snapshot.replaced)
	}
	if len(snapshot.replacedTransactions) != 1 {
		t.Fatalf("replaced transactions = %d", len(snapshot.replacedTransactions))
	}
	transaction := snapshot.replacedTransactions[0]
	if transaction.Description != "Проценты по вкладу Накопительный счёт" {
		t.Fatalf("description = %q", transaction.Description)
	}
	if transaction.CategoryID == nil || *transaction.CategoryID != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("category id = %v", transaction.CategoryID)
	}
}

func lifecycleRule(startDate time.Time) models.InterestRule {
	return models.InterestRule{
		ID: "22222222-2222-2222-2222-222222222222", AccountID: "11111111-1111-1111-1111-111111111111",
		AnnualRateBps: 36500, AccrualFrequency: models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyNone,
		DayCountConvention:      models.DayCountConventionActual365, IsActive: true, StartDate: startDate,
	}
}

func newTestInterestLifecycle(repo repository.InterestAccrualTransactionalRepository) *InterestLifecycle {
	return NewInterestLifecycle(repo, NewInterestEngine()).
		WithAccountRepository(&recordingAccountRepo{existing: &models.Account{
			ID: "11111111-1111-1111-1111-111111111111", Name: "Накопительный счёт", Currency: "RUB",
		}}).
		WithCategoryRepository(&moduleCategoryRepo{category: &models.Category{
			ID: "33333333-3333-3333-3333-333333333333", Slug: "deposit_interest", Name: "Проценты по вкладам",
		}})
}

type lifecycleRepo struct {
	snapshot   *lifecycleSnapshot
	lockErr    error
	rolledBack bool
}

func (r *lifecycleRepo) WithAccountInterestLock(ctx context.Context, _, _ string, fn func(context.Context, repository.InterestCalculationRepository) error) error {
	if r.lockErr != nil {
		return r.lockErr
	}
	if err := fn(ctx, r.snapshot); err != nil {
		r.rolledBack = true
		return err
	}
	return nil
}

type lifecycleSnapshot struct {
	rules                []models.InterestRule
	transactions         []models.Transaction
	accruals             []models.InterestAccrual
	createdTransactions  []models.Transaction
	createdAccruals      []models.InterestAccrual
	replacedTransactions []models.Transaction
	createErr            error
	replaced             bool
	deleted              int64
}

func (s *lifecycleSnapshot) GetInterestRuleByID(_ context.Context, id string) (*models.InterestRule, error) {
	for i := range s.rules {
		if s.rules[i].ID == id {
			return &s.rules[i], nil
		}
	}
	return nil, repository.ErrNotFound
}

func (s *lifecycleSnapshot) ListInterestRulesByAccount(_ context.Context, accountID string) ([]models.InterestRule, error) {
	var rules []models.InterestRule
	for i := range s.rules {
		if s.rules[i].AccountID == accountID {
			rules = append(rules, s.rules[i])
		}
	}
	return rules, nil
}

func (s *lifecycleSnapshot) ListTransactionsByAccountForUser(context.Context, string, string) ([]models.Transaction, error) {
	return append([]models.Transaction(nil), s.transactions...), nil
}

func (s *lifecycleSnapshot) ListInterestAccrualsByAccount(context.Context, string) ([]models.InterestAccrual, error) {
	return append([]models.InterestAccrual(nil), s.accruals...), nil
}

func (s *lifecycleSnapshot) CreateInterestAccrualWithTransaction(_ context.Context, transaction *models.Transaction, accrual *models.InterestAccrual) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.createdTransactions = append(s.createdTransactions, *transaction)
	s.createdAccruals = append(s.createdAccruals, *accrual)
	return nil
}

func (s *lifecycleSnapshot) ReplaceInterestAccrualRangeWithTransactions(_ context.Context, _, _ string, _, _ time.Time, transactions []models.Transaction, _ []models.InterestAccrual) (int64, error) {
	s.replaced = true
	s.replacedTransactions = append([]models.Transaction(nil), transactions...)
	return s.deleted, nil
}

var (
	_ repository.InterestAccrualTransactionalRepository = (*lifecycleRepo)(nil)
	_ repository.InterestCalculationRepository          = (*lifecycleSnapshot)(nil)
)
