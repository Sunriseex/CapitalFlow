package dto

import (
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/pkg/money"
)

type DashboardAmountResponse struct {
	Currency string            `json:"currency"`
	Amount   money.JSONDecimal `json:"amount"`
}

type DashboardAccountBalanceResponse struct {
	AccountID        string             `json:"account_id"`
	Name             string             `json:"name"`
	Bank             string             `json:"bank,omitempty"`
	Type             models.AccountType `json:"type"`
	Currency         string             `json:"currency"`
	IsActive         bool               `json:"is_active"`
	Balance          money.JSONDecimal  `json:"balance"`
	TransactionCount int                `json:"transaction_count"`
}

type DashboardGoalProgressResponse struct {
	ID            string                     `json:"id"`
	AccountID     string                     `json:"account_id"`
	Name          string                     `json:"name"`
	CurrentAmount money.JSONDecimal          `json:"current_amount"`
	TargetAmount  money.JSONDecimal          `json:"target_amount"`
	Currency      string                     `json:"currency"`
	TargetDate    *string                    `json:"target_date,omitempty"`
	Status        models.FinancialGoalStatus `json:"status"`
}

type DashboardCategoryLimitProgressResponse struct {
	ID            string            `json:"id"`
	CategoryID    string            `json:"category_id"`
	CategoryName  string            `json:"category_name"`
	CurrentAmount money.JSONDecimal `json:"current_amount"`
	TargetAmount  money.JSONDecimal `json:"target_amount"`
	Currency      string            `json:"currency"`
}

type DashboardSummaryResponse struct {
	GeneratedAt                time.Time                                `json:"generated_at"`
	AccountsCount              int                                      `json:"accounts_count"`
	ActiveAccountsCount        int                                      `json:"active_accounts_count"`
	Balances                   []DashboardAmountResponse                `json:"balances"`
	MonthlyIncome              []DashboardAmountResponse                `json:"monthly_income"`
	MonthlyExpense             []DashboardAmountResponse                `json:"monthly_expense"`
	MonthlyInterestIncome      []DashboardAmountResponse                `json:"monthly_interest_income"`
	AccountBalances            []DashboardAccountBalanceResponse        `json:"account_balances"`
	FinancialGoals             []DashboardGoalProgressResponse          `json:"financial_goals"`
	CategoryLimits             []DashboardCategoryLimitProgressResponse `json:"category_limits"`
	RecentTransactions         []TransactionResponse                    `json:"recent_transactions"`
	RecentTransactionsLimit    int                                      `json:"recent_transactions_limit"`
	RecentTransactionsReturned int                                      `json:"recent_transactions_returned"`
}

type DashboardNetWorthResponse struct {
	GeneratedAt     time.Time                         `json:"generated_at"`
	Balances        []DashboardAmountResponse         `json:"balances"`
	AccountBalances []DashboardAccountBalanceResponse `json:"account_balances"`
}

type DashboardCashflowBucketResponse struct {
	Period           string                    `json:"period"`
	Income           []DashboardAmountResponse `json:"income"`
	Expense          []DashboardAmountResponse `json:"expense"`
	NetCashflow      []DashboardAmountResponse `json:"net_cashflow"`
	TransactionCount int                       `json:"transaction_count"`
}

type DashboardCashflowResponse struct {
	GeneratedAt time.Time                         `json:"generated_at"`
	Months      int                               `json:"months"`
	Buckets     []DashboardCashflowBucketResponse `json:"buckets"`
}

type DashboardInterestIncomeBucketResponse struct {
	Period           string                    `json:"period"`
	InterestIncome   []DashboardAmountResponse `json:"interest_income"`
	TransactionCount int                       `json:"transaction_count"`
}

type DashboardInterestIncomeResponse struct {
	GeneratedAt time.Time                               `json:"generated_at"`
	Months      int                                     `json:"months"`
	Total       []DashboardAmountResponse               `json:"total"`
	Buckets     []DashboardInterestIncomeBucketResponse `json:"buckets"`
}
