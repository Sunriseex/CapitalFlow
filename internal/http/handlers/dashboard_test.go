package handlers

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/models"
)

func TestBuildDashboardSummary(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	accounts := []models.Account{
		{
			ID:       "account-1",
			Name:     "Main",
			Type:     models.AccountTypeCard,
			Currency: "RUB",
			IsActive: true,
		},
		{
			ID:       "account-2",
			Name:     "Archived",
			Type:     models.AccountTypeSavings,
			Currency: "RUB",
			IsActive: false,
		},
		{
			ID:       "account-3",
			Name:     "USD",
			Type:     models.AccountTypeBroker,
			Currency: "USD",
			IsActive: true,
		},
	}
	transactions := []models.Transaction{
		{
			ID:         "initial-1",
			AccountID:  "account-1",
			Type:       models.TransactionTypeInitialBalance,
			Amount:     dec("1000"),
			OccurredAt: time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
			CreatedAt:  time.Date(2026, 4, 30, 1, 0, 0, 0, time.UTC),
		},
		{
			ID:         "income-1",
			AccountID:  "account-1",
			Type:       models.TransactionTypeIncome,
			Amount:     dec("500"),
			OccurredAt: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
			CreatedAt:  time.Date(2026, 5, 2, 1, 0, 0, 0, time.UTC),
		},
		{
			ID:         "expense-1",
			AccountID:  "account-1",
			Type:       models.TransactionTypeExpense,
			Amount:     dec("200"),
			OccurredAt: time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
			CreatedAt:  time.Date(2026, 5, 3, 1, 0, 0, 0, time.UTC),
		},
		{
			ID:         "interest-1",
			AccountID:  "account-1",
			Type:       models.TransactionTypeInterestIncome,
			Amount:     dec("10"),
			OccurredAt: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
			CreatedAt:  time.Date(2026, 5, 4, 1, 0, 0, 0, time.UTC),
		},
		{
			ID:         "archived-initial",
			AccountID:  "account-2",
			Type:       models.TransactionTypeInitialBalance,
			Amount:     dec("9999.99"),
			OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			CreatedAt:  time.Date(2026, 5, 1, 1, 0, 0, 0, time.UTC),
		},
		{
			ID:         "usd-initial",
			AccountID:  "account-3",
			Type:       models.TransactionTypeInitialBalance,
			Amount:     dec("100"),
			OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			CreatedAt:  time.Date(2026, 5, 1, 1, 0, 0, 0, time.UTC),
		},
	}

	got, err := buildDashboardSummary(t.Context(), now, accounts, transactions, nil, nil, nil, 3)
	if err != nil {
		t.Fatalf("build summary: %v", err)
	}

	if got.AccountsCount != 3 {
		t.Fatalf("accounts count = %d, want 3", got.AccountsCount)
	}
	if got.ActiveAccountsCount != 2 {
		t.Fatalf("active accounts count = %d, want 2", got.ActiveAccountsCount)
	}
	assertDashboardAmount(t, got.Balances, "RUB", 131_000)
	assertDashboardAmount(t, got.Balances, "USD", 10_000)
	assertDashboardAmount(t, got.MonthlyIncome, "RUB", 51_000)
	assertDashboardAmount(t, got.MonthlyExpense, "RUB", 20_000)
	assertDashboardAmount(t, got.MonthlyInterestIncome, "RUB", 1_000)

	if got.RecentTransactionsReturned != 3 {
		t.Fatalf("recent returned = %d, want 3", got.RecentTransactionsReturned)
	}
	if got.RecentTransactions[0].ID != "interest-1" {
		t.Fatalf("first recent transaction = %s, want interest-1", got.RecentTransactions[0].ID)
	}
}

func TestBuildDashboardNetWorth(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	accounts := []models.Account{
		{
			ID:       "active",
			Name:     "Active",
			Type:     models.AccountTypeCard,
			Currency: "RUB",
			IsActive: true,
		},
		{
			ID:       "archived",
			Name:     "Archived",
			Type:     models.AccountTypeSavings,
			Currency: "RUB",
			IsActive: false,
		},
	}
	transactions := []models.Transaction{
		{
			ID:         "active-balance",
			AccountID:  "active",
			Type:       models.TransactionTypeInitialBalance,
			Amount:     dec("1000"),
			OccurredAt: now,
		},
		{
			ID:         "archived-balance",
			AccountID:  "archived",
			Type:       models.TransactionTypeInitialBalance,
			Amount:     dec("9999.99"),
			OccurredAt: now,
		},
	}

	got, err := buildDashboardNetWorth(t.Context(), now, accounts, transactions)
	if err != nil {
		t.Fatalf("build net worth: %v", err)
	}

	assertDashboardAmount(t, got.Balances, "RUB", 100_000)
	if len(got.AccountBalances) != 2 {
		t.Fatalf("account balances len = %d, want 2", len(got.AccountBalances))
	}
}

func TestBuildDashboardGoalAndLimitProgress(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	accountID := "rub-account"
	legacyAccountID := "missing-account"
	foodID := "food"
	accounts := []models.Account{{
		ID: accountID, Name: "Savings", Type: models.AccountTypeSavings,
		Currency: "RUB", IsActive: true,
	}}
	transactions := []models.Transaction{
		{ID: "balance", AccountID: accountID, Type: models.TransactionTypeInitialBalance, Amount: dec("1700"), OccurredAt: now.AddDate(0, -1, 0)},
		{ID: "food-current", AccountID: accountID, Type: models.TransactionTypeExpense, Amount: dec("83"), CategoryID: &foodID, OccurredAt: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)},
		{ID: "food-previous", AccountID: accountID, Type: models.TransactionTypeExpense, Amount: dec("999"), CategoryID: &foodID, OccurredAt: time.Date(2026, 4, 30, 23, 59, 0, 0, time.UTC)},
	}
	goals := []models.FinancialGoal{
		{ID: "active", AccountID: &accountID, Name: "Reserve", TargetAmount: dec("1000"), Currency: "RUB", Status: models.FinancialGoalActive},
		{ID: "legacy", Name: "Legacy", TargetAmount: dec("1000"), Currency: "RUB", Status: models.FinancialGoalActive},
		{ID: "missing", AccountID: &legacyAccountID, Name: "Missing", TargetAmount: dec("1000"), Currency: "RUB", Status: models.FinancialGoalActive},
		{ID: "inactive", AccountID: &accountID, Name: "Inactive", TargetAmount: dec("1000"), Currency: "RUB", Status: models.FinancialGoalArchived},
	}
	limits := []models.CategoryLimit{
		{ID: "rub-limit", CategoryID: foodID, Amount: dec("100"), Currency: "RUB", IsActive: true},
		{ID: "usd-limit", CategoryID: foodID, Amount: dec("100"), Currency: "USD", IsActive: true},
		{ID: "inactive-limit", CategoryID: foodID, Amount: dec("100"), Currency: "RUB", IsActive: false},
	}

	got, err := buildDashboardSummary(t.Context(), now, accounts, transactions, goals, limits, []models.Category{{ID: foodID, Name: "Food"}}, 10)
	if err != nil {
		t.Fatalf("build summary: %v", err)
	}
	if len(got.FinancialGoals) != 1 || got.FinancialGoals[0].ID != "active" {
		t.Fatalf("financial goals = %#v, want only linked active goal", got.FinancialGoals)
	}
	if !got.FinancialGoals[0].CurrentAmount.Equal(dec("618")) {
		t.Fatalf("goal current = %s, want 618", got.FinancialGoals[0].CurrentAmount)
	}
	if len(got.CategoryLimits) != 2 {
		t.Fatalf("category limits len = %d, want 2 active limits", len(got.CategoryLimits))
	}
	if !got.CategoryLimits[0].CurrentAmount.Equal(dec("83")) {
		t.Fatalf("RUB limit current = %s, want 83", got.CategoryLimits[0].CurrentAmount)
	}
	if !got.CategoryLimits[1].CurrentAmount.IsZero() {
		t.Fatalf("USD limit current = %s, want 0", got.CategoryLimits[1].CurrentAmount)
	}
}

func TestBuildDashboardCashflow(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	accounts := []models.Account{
		{
			ID:       "account-1",
			Name:     "Main",
			Type:     models.AccountTypeCard,
			Currency: "RUB",
			IsActive: true,
		},
	}
	transactions := []models.Transaction{
		{
			ID:         "income-current",
			AccountID:  "account-1",
			Type:       models.TransactionTypeIncome,
			Amount:     dec("1000"),
			OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:         "expense-current",
			AccountID:  "account-1",
			Type:       models.TransactionTypeExpense,
			Amount:     dec("400"),
			OccurredAt: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:         "interest-current",
			AccountID:  "account-1",
			Type:       models.TransactionTypeInterestIncome,
			Amount:     dec("50"),
			OccurredAt: time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:         "old-income",
			AccountID:  "account-1",
			Type:       models.TransactionTypeIncome,
			Amount:     dec("9999.99"),
			OccurredAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	got := buildDashboardCashflow(now, accounts, transactions, 2)

	if len(got.Buckets) != 2 {
		t.Fatalf("buckets len = %d, want 2", len(got.Buckets))
	}
	if got.Buckets[0].Period != "2026-04" || got.Buckets[1].Period != "2026-05" {
		t.Fatalf("periods = %s, %s; want 2026-04, 2026-05", got.Buckets[0].Period, got.Buckets[1].Period)
	}
	assertDashboardAmount(t, got.Buckets[1].Income, "RUB", 105_000)
	assertDashboardAmount(t, got.Buckets[1].Expense, "RUB", 40_000)
	assertDashboardAmount(t, got.Buckets[1].NetCashflow, "RUB", 65_000)
}

func TestBuildDashboardInterestIncome(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	accounts := []models.Account{
		{
			ID:       "account-1",
			Name:     "Main",
			Type:     models.AccountTypeSavings,
			Currency: "RUB",
			IsActive: true,
		},
	}
	transactions := []models.Transaction{
		{
			ID:         "interest-april",
			AccountID:  "account-1",
			Type:       models.TransactionTypeInterestIncome,
			Amount:     dec("40"),
			OccurredAt: time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:         "interest-may",
			AccountID:  "account-1",
			Type:       models.TransactionTypeInterestIncome,
			Amount:     dec("50"),
			OccurredAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:         "income-may",
			AccountID:  "account-1",
			Type:       models.TransactionTypeIncome,
			Amount:     dec("1000"),
			OccurredAt: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
		},
	}

	got := buildDashboardInterestIncome(now, accounts, transactions, 2)

	assertDashboardAmount(t, got.Total, "RUB", 9_000)
	assertDashboardAmount(t, got.Buckets[0].InterestIncome, "RUB", 4_000)
	assertDashboardAmount(t, got.Buckets[1].InterestIncome, "RUB", 5_000)
}

func assertDashboardAmount(t *testing.T, amounts []dto.DashboardAmountResponse, currency string, want int64) {
	t.Helper()

	for _, amount := range amounts {
		if amount.Currency == currency {
			if !amount.Amount.Equal(decimal.NewFromInt(want).Div(decimal.NewFromInt(100))) {
				t.Fatalf("%s amount = %d, want %d", currency, amount.Amount, want)
			}
			return
		}
	}
	t.Fatalf("currency %s not found", currency)
}
