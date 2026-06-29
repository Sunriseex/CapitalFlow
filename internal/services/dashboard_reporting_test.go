package services

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestBuildDashboardSummary(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	accountID := "account-1"
	foodID := "food"
	snapshot := &repository.DashboardSummarySnapshot{
		AccountBalances: []repository.DashboardAccountBalance{
			{Account: models.Account{ID: accountID, Name: "Main", Type: models.AccountTypeCard, Currency: "RUB", IsActive: true}, Balance: decimal.RequireFromString("1310"), TransactionCount: 4},
			{Account: models.Account{ID: "archived", Name: "Old", Type: models.AccountTypeSavings, Currency: "RUB"}, Balance: decimal.RequireFromString("9999.99"), TransactionCount: 1},
		},
		MonthlyFlow:     []repository.DashboardMonthlyFlow{{Currency: "RUB", Income: decimal.RequireFromString("510"), Expense: decimal.RequireFromString("200"), InterestIncome: decimal.RequireFromString("10")}},
		CategoryExpense: []repository.DashboardCategoryExpense{{CategoryID: foodID, Currency: "RUB", Amount: decimal.RequireFromString("83")}},
		Goals: []models.FinancialGoal{
			{ID: "active", AccountID: &accountID, Name: "Reserve", TargetAmount: decimal.RequireFromString("1000"), Currency: "RUB", Status: models.FinancialGoalActive},
			{ID: "unlinked", Name: "Old", TargetAmount: decimal.RequireFromString("1000"), Currency: "RUB", Status: models.FinancialGoalActive},
		},
		Limits:     []models.CategoryLimit{{ID: "limit", CategoryID: foodID, Amount: decimal.RequireFromString("100"), Currency: "RUB", IsActive: true}},
		Categories: []models.Category{{ID: foodID, Name: "Food"}},
		Recent: []models.Transaction{
			{ID: "older", OccurredAt: now.Add(-time.Hour)},
			{ID: "newer", OccurredAt: now},
		},
	}

	got := BuildDashboardSummary(now, snapshot, 1)
	if got.AccountsCount != 2 || got.ActiveAccountsCount != 1 {
		t.Fatalf("account counts = %d/%d, want 2/1", got.AccountsCount, got.ActiveAccountsCount)
	}
	assertDashboardAmount(t, got.Balances, "RUB", "1310")
	assertDashboardAmount(t, got.MonthlyIncome, "RUB", "510")
	assertDashboardAmount(t, got.MonthlyExpense, "RUB", "200")
	assertDashboardAmount(t, got.MonthlyInterestIncome, "RUB", "10")
	if len(got.FinancialGoals) != 1 || !got.FinancialGoals[0].CurrentAmount.Equal(decimal.RequireFromString("1310")) {
		t.Fatalf("financial goals = %#v", got.FinancialGoals)
	}
	if len(got.CategoryLimits) != 1 || !got.CategoryLimits[0].CurrentAmount.Equal(decimal.RequireFromString("83")) {
		t.Fatalf("category limits = %#v", got.CategoryLimits)
	}
	if got.RecentTransactionsReturned != 1 || got.RecentTransactions[0].ID != "newer" {
		t.Fatalf("recent transactions = %#v", got.RecentTransactions)
	}
}

func TestBuildDashboardNetWorthExcludesArchivedAccounts(t *testing.T) {
	balances := []repository.DashboardAccountBalance{
		{Account: models.Account{ID: "active", Currency: "RUB", IsActive: true}, Balance: decimal.RequireFromString("1000")},
		{Account: models.Account{ID: "archived", Currency: "RUB"}, Balance: decimal.RequireFromString("9999.99")},
	}
	got := BuildDashboardNetWorth(time.Now(), balances)
	assertDashboardAmount(t, got.Balances, "RUB", "1000")
	if len(got.AccountBalances) != 2 {
		t.Fatalf("account balances = %d, want 2", len(got.AccountBalances))
	}
}

func TestBuildDashboardCashflow(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	flow := []repository.DashboardMonthlyFlow{
		{Period: "2026-05", Currency: "RUB", Income: decimal.RequireFromString("1050"), Expense: decimal.RequireFromString("400"), TransactionCount: 3},
		{Period: "2026-03", Currency: "RUB", Income: decimal.RequireFromString("9999"), TransactionCount: 1},
	}
	got := BuildDashboardCashflow(now, flow, 2)
	if len(got.Buckets) != 2 || got.Buckets[0].Period != "2026-04" || got.Buckets[1].Period != "2026-05" {
		t.Fatalf("buckets = %#v", got.Buckets)
	}
	assertDashboardAmount(t, got.Buckets[1].Income, "RUB", "1050")
	assertDashboardAmount(t, got.Buckets[1].Expense, "RUB", "400")
	assertDashboardAmount(t, got.Buckets[1].NetCashflow, "RUB", "650")
}

func TestBuildDashboardInterestIncome(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	flow := []repository.DashboardMonthlyFlow{
		{Period: "2026-04", Currency: "RUB", InterestIncome: decimal.RequireFromString("40"), InterestCount: 1},
		{Period: "2026-05", Currency: "RUB", InterestIncome: decimal.RequireFromString("50"), InterestCount: 2},
	}
	got := BuildDashboardInterestIncome(now, flow, 2)
	assertDashboardAmount(t, got.Total, "RUB", "90")
	assertDashboardAmount(t, got.Buckets[0].InterestIncome, "RUB", "40")
	assertDashboardAmount(t, got.Buckets[1].InterestIncome, "RUB", "50")
	if got.Buckets[1].TransactionCount != 2 {
		t.Fatalf("interest count = %d, want 2", got.Buckets[1].TransactionCount)
	}
}

func assertDashboardAmount(t *testing.T, amounts []DashboardAmount, currency, want string) {
	t.Helper()
	for _, amount := range amounts {
		if amount.Currency == currency {
			if !amount.Amount.Equal(decimal.RequireFromString(want)) {
				t.Fatalf("%s amount = %s, want %s", currency, amount.Amount, want)
			}
			return
		}
	}
	t.Fatalf("currency %s not found", currency)
}
