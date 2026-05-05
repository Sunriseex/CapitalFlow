package postgres

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/internal/repository"
)

func TestPostgresRepositoriesIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := t.Context()
	pool, err := OpenPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		TRUNCATE interest_accruals, interest_rules, transactions, categories, accounts RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("truncate test tables; run migrations first: %v", err)
	}

	store := NewStore(pool)
	accounts := store.Accounts()
	transactions := store.Transactions()
	categories := store.Categories()
	rules := store.InterestRules()
	accruals := store.InterestAccruals()

	now := time.Now().UTC()
	legacyID := "legacy-" + uuid.NewString()
	account := &models.Account{
		ID:        uuid.NewString(),
		LegacyID:  &legacyID,
		Name:      "Integration Savings",
		Bank:      "Yandex",
		Type:      models.AccountTypeSavings,
		Currency:  "RUB",
		IsActive:  true,
		OpenedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := accounts.Create(ctx, account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	gotAccount, err := accounts.GetByLegacyID(ctx, legacyID)
	if err != nil {
		t.Fatalf("get by legacy id: %v", err)
	}
	if gotAccount.ID != account.ID {
		t.Fatalf("account id = %s, want %s", gotAccount.ID, account.ID)
	}

	category := &models.Category{
		ID:        uuid.NewString(),
		Slug:      "integration-category",
		Name:      "Integration Category",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := categories.Create(ctx, category); err != nil {
		t.Fatalf("create category: %v", err)
	}
	if _, err := categories.GetBySlug(ctx, category.Slug); err != nil {
		t.Fatalf("get category by slug: %v", err)
	}

	initialBalance := models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   account.ID,
		Type:        models.TransactionTypeInitialBalance,
		AmountMinor: 100_000,
		CategoryID:  &category.ID,
		Description: "initial",
		OccurredAt:  now,
		CreatedAt:   now,
	}
	income := models.Transaction{
		ID:          uuid.NewString(),
		AccountID:   account.ID,
		Type:        models.TransactionTypeIncome,
		AmountMinor: 10_000,
		CategoryID:  &category.ID,
		Description: "income",
		OccurredAt:  now.Add(time.Minute),
		CreatedAt:   now,
	}
	if err := transactions.CreateMany(ctx, []models.Transaction{initialBalance, income}); err != nil {
		t.Fatalf("create transaction batch: %v", err)
	}
	gotTransactions, err := transactions.ListByAccount(ctx, account.ID)
	if err != nil {
		t.Fatalf("list account transactions: %v", err)
	}
	if len(gotTransactions) != 2 {
		t.Fatalf("transactions count = %d, want 2", len(gotTransactions))
	}

	rule := &models.InterestRule{
		ID:                      uuid.NewString(),
		AccountID:               account.ID,
		AnnualRateBps:           1_200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyDaily,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               pgDateOnly(now),
	}
	if err := rules.Create(ctx, rule); err != nil {
		t.Fatalf("create interest rule: %v", err)
	}

	accrualDate := pgDateOnly(now)
	accrual := &models.InterestAccrual{
		ID:            uuid.NewString(),
		AccountID:     account.ID,
		RuleID:        rule.ID,
		TransactionID: income.ID,
		AccrualDate:   accrualDate,
		AmountMinor:   100,
		BalanceMinor:  100_000,
		AnnualRateBps: 1_200,
		CreatedAt:     now,
	}
	if err := accruals.Create(ctx, accrual); err != nil {
		t.Fatalf("create interest accrual: %v", err)
	}
	if err := accruals.Create(ctx, accrual); err == nil {
		t.Fatal("duplicate interest accrual must fail")
	}
	gotAccrual, err := accruals.GetByAccountDateRule(ctx, account.ID, accrualDate.Format(time.DateOnly), rule.ID)
	if err != nil {
		t.Fatalf("get interest accrual: %v", err)
	}
	if gotAccrual.ID != accrual.ID {
		t.Fatalf("accrual id = %s, want %s", gotAccrual.ID, accrual.ID)
	}

	if err := accounts.Archive(ctx, account.ID); err != nil {
		t.Fatalf("archive account: %v", err)
	}
	if err := transactions.Delete(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("delete missing err = %v, want ErrNotFound", err)
	}
}

func pgDateOnly(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}
