package repository

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
)

// DashboardAccountBalance is an account with its ledger balance already
// aggregated by the read adapter.
type DashboardAccountBalance struct {
	Account          models.Account
	Balance          decimal.Decimal
	TransactionCount int
}

type DashboardMonthlyFlow struct {
	Period           string
	Currency         string
	Income           decimal.Decimal
	Expense          decimal.Decimal
	InterestIncome   decimal.Decimal
	TransactionCount int
	InterestCount    int
}

type DashboardCategoryExpense struct {
	CategoryID string
	Currency   string
	Amount     decimal.Decimal
}

type DashboardSummarySnapshot struct {
	AccountBalances []DashboardAccountBalance
	MonthlyFlow     []DashboardMonthlyFlow
	CategoryExpense []DashboardCategoryExpense
	Goals           []models.FinancialGoal
	Limits          []models.CategoryLimit
	Categories      []models.Category
	Recent          []models.Transaction
}

// DashboardRepository is a bounded, read-only projection of the ledger.
type DashboardRepository interface {
	Summary(ctx context.Context, userID string, from, to time.Time, recentLimit int) (*DashboardSummarySnapshot, error)
	AccountBalances(ctx context.Context, userID string) ([]DashboardAccountBalance, error)
	MonthlyFlow(ctx context.Context, userID string, from, to time.Time) ([]DashboardMonthlyFlow, error)
}
