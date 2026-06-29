package services

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

const DashboardRecentTransactionsLimit = 10

type DashboardReporting struct {
	repo repository.DashboardRepository
	now  func() time.Time
}

func NewDashboardReporting(repo repository.DashboardRepository) *DashboardReporting {
	return &DashboardReporting{repo: repo, now: time.Now}
}

type DashboardAmount struct {
	Currency string
	Amount   decimal.Decimal
}

type DashboardAccountBalance struct {
	AccountID        string
	Name             string
	Bank             string
	Type             models.AccountType
	Currency         string
	IsActive         bool
	Balance          decimal.Decimal
	TransactionCount int
}

type DashboardGoalProgress struct {
	ID            string
	AccountID     string
	Name          string
	CurrentAmount decimal.Decimal
	TargetAmount  decimal.Decimal
	Currency      string
	TargetDate    *string
	Status        models.FinancialGoalStatus
}

type DashboardCategoryLimitProgress struct {
	ID            string
	CategoryID    string
	CategoryName  string
	CurrentAmount decimal.Decimal
	TargetAmount  decimal.Decimal
	Currency      string
}

type DashboardSummary struct {
	GeneratedAt                time.Time
	AccountsCount              int
	ActiveAccountsCount        int
	Balances                   []DashboardAmount
	MonthlyIncome              []DashboardAmount
	MonthlyExpense             []DashboardAmount
	MonthlyInterestIncome      []DashboardAmount
	AccountBalances            []DashboardAccountBalance
	FinancialGoals             []DashboardGoalProgress
	CategoryLimits             []DashboardCategoryLimitProgress
	RecentTransactions         []models.Transaction
	RecentTransactionsLimit    int
	RecentTransactionsReturned int
}

type DashboardNetWorth struct {
	GeneratedAt     time.Time
	Balances        []DashboardAmount
	AccountBalances []DashboardAccountBalance
}

type DashboardCashflowBucket struct {
	Period           string
	Income           []DashboardAmount
	Expense          []DashboardAmount
	NetCashflow      []DashboardAmount
	TransactionCount int
}

type DashboardCashflow struct {
	GeneratedAt time.Time
	Months      int
	Buckets     []DashboardCashflowBucket
}

type DashboardInterestIncomeBucket struct {
	Period           string
	InterestIncome   []DashboardAmount
	TransactionCount int
}

type DashboardInterestIncome struct {
	GeneratedAt time.Time
	Months      int
	Total       []DashboardAmount
	Buckets     []DashboardInterestIncomeBucket
}

func (s *DashboardReporting) Summary(ctx context.Context, userID string) (*DashboardSummary, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("dashboard repository is not configured")
	}
	now := s.now()
	from := monthStart(now)
	snapshot, err := s.repo.Summary(ctx, userID, from, from.AddDate(0, 1, 0), DashboardRecentTransactionsLimit)
	if err != nil {
		return nil, fmt.Errorf("read dashboard summary: %w", err)
	}
	return BuildDashboardSummary(now, snapshot, DashboardRecentTransactionsLimit), nil
}

func (s *DashboardReporting) NetWorth(ctx context.Context, userID string) (*DashboardNetWorth, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("dashboard repository is not configured")
	}
	now := s.now()
	balances, err := s.repo.AccountBalances(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("read dashboard net worth: %w", err)
	}
	return BuildDashboardNetWorth(now, balances), nil
}

func (s *DashboardReporting) Cashflow(ctx context.Context, userID string, months int) (*DashboardCashflow, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("dashboard repository is not configured")
	}
	if err := validateDashboardMonths(months); err != nil {
		return nil, err
	}
	now := s.now()
	from := monthStart(now).AddDate(0, -(months - 1), 0)
	flow, err := s.repo.MonthlyFlow(ctx, userID, from, monthStart(now).AddDate(0, 1, 0))
	if err != nil {
		return nil, fmt.Errorf("read dashboard cashflow: %w", err)
	}
	return BuildDashboardCashflow(now, flow, months), nil
}

func (s *DashboardReporting) InterestIncome(ctx context.Context, userID string, months int) (*DashboardInterestIncome, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("dashboard repository is not configured")
	}
	if err := validateDashboardMonths(months); err != nil {
		return nil, err
	}
	now := s.now()
	from := monthStart(now).AddDate(0, -(months - 1), 0)
	flow, err := s.repo.MonthlyFlow(ctx, userID, from, monthStart(now).AddDate(0, 1, 0))
	if err != nil {
		return nil, fmt.Errorf("read dashboard interest income: %w", err)
	}
	return BuildDashboardInterestIncome(now, flow, months), nil
}

func BuildDashboardSummary(now time.Time, snapshot *repository.DashboardSummarySnapshot, recentLimit int) *DashboardSummary {
	if recentLimit < 0 {
		recentLimit = 0
	}
	if snapshot == nil {
		snapshot = &repository.DashboardSummarySnapshot{}
	}

	summary := &DashboardSummary{
		GeneratedAt:             now,
		AccountsCount:           len(snapshot.AccountBalances),
		RecentTransactionsLimit: recentLimit,
	}
	balances := make(map[string]decimal.Decimal)
	accountBalances := make(map[string]decimal.Decimal, len(snapshot.AccountBalances))
	for i := range snapshot.AccountBalances {
		item := &snapshot.AccountBalances[i]
		account := item.Account
		if account.IsActive {
			summary.ActiveAccountsCount++
			balances[account.Currency] = balances[account.Currency].Add(item.Balance)
		}
		accountBalances[account.ID] = item.Balance
		summary.AccountBalances = append(summary.AccountBalances, dashboardAccountBalance(item))
	}

	income := make(map[string]decimal.Decimal)
	expense := make(map[string]decimal.Decimal)
	interestIncome := make(map[string]decimal.Decimal)
	for _, item := range snapshot.MonthlyFlow {
		if !item.Income.IsZero() {
			income[item.Currency] = income[item.Currency].Add(item.Income)
		}
		if !item.Expense.IsZero() {
			expense[item.Currency] = expense[item.Currency].Add(item.Expense)
		}
		if !item.InterestIncome.IsZero() {
			interestIncome[item.Currency] = interestIncome[item.Currency].Add(item.InterestIncome)
		}
	}

	categoryExpense := make(map[string]decimal.Decimal, len(snapshot.CategoryExpense))
	for _, item := range snapshot.CategoryExpense {
		categoryExpense[dashboardCategoryCurrencyKey(item.CategoryID, item.Currency)] = item.Amount
	}
	for i := range snapshot.Goals {
		goal := &snapshot.Goals[i]
		if goal.Status != models.FinancialGoalActive || goal.AccountID == nil {
			continue
		}
		balance, ok := accountBalances[*goal.AccountID]
		if !ok {
			continue
		}
		if balance.IsNegative() {
			balance = decimal.Zero
		}
		var targetDate *string
		if goal.TargetDate != nil {
			formatted := goal.TargetDate.Format(time.DateOnly)
			targetDate = &formatted
		}
		summary.FinancialGoals = append(summary.FinancialGoals, DashboardGoalProgress{
			ID: goal.ID, AccountID: *goal.AccountID, Name: goal.Name, CurrentAmount: balance,
			TargetAmount: goal.TargetAmount, Currency: goal.Currency, TargetDate: targetDate, Status: goal.Status,
		})
	}

	categoryNames := make(map[string]string, len(snapshot.Categories))
	for _, category := range snapshot.Categories {
		categoryNames[category.ID] = category.Name
	}
	for i := range snapshot.Limits {
		limit := &snapshot.Limits[i]
		if !limit.IsActive {
			continue
		}
		summary.CategoryLimits = append(summary.CategoryLimits, DashboardCategoryLimitProgress{
			ID: limit.ID, CategoryID: limit.CategoryID, CategoryName: categoryNames[limit.CategoryID],
			CurrentAmount: categoryExpense[dashboardCategoryCurrencyKey(limit.CategoryID, limit.Currency)],
			TargetAmount:  limit.Amount, Currency: limit.Currency,
		})
	}

	recent := slices.Clone(snapshot.Recent)
	slices.SortFunc(recent, func(a, b models.Transaction) int {
		if byOccurredAt := b.OccurredAt.Compare(a.OccurredAt); byOccurredAt != 0 {
			return byOccurredAt
		}
		if byCreatedAt := b.CreatedAt.Compare(a.CreatedAt); byCreatedAt != 0 {
			return byCreatedAt
		}
		return cmp.Compare(b.ID, a.ID)
	})
	if len(recent) > recentLimit {
		recent = recent[:recentLimit]
	}
	summary.Balances = dashboardAmountsFromMap(balances)
	summary.MonthlyIncome = dashboardAmountsFromMap(income)
	summary.MonthlyExpense = dashboardAmountsFromMap(expense)
	summary.MonthlyInterestIncome = dashboardAmountsFromMap(interestIncome)
	summary.RecentTransactions = recent
	summary.RecentTransactionsReturned = len(recent)
	return summary
}

func BuildDashboardNetWorth(now time.Time, accountBalances []repository.DashboardAccountBalance) *DashboardNetWorth {
	balances := make(map[string]decimal.Decimal)
	response := &DashboardNetWorth{GeneratedAt: now}
	for i := range accountBalances {
		item := &accountBalances[i]
		if item.Account.IsActive {
			balances[item.Account.Currency] = balances[item.Account.Currency].Add(item.Balance)
		}
		response.AccountBalances = append(response.AccountBalances, dashboardAccountBalance(item))
	}
	response.Balances = dashboardAmountsFromMap(balances)
	return response
}

func BuildDashboardCashflow(now time.Time, flow []repository.DashboardMonthlyFlow, months int) *DashboardCashflow {
	buckets := dashboardCashflowMonthBuckets(now, months)
	bucketByPeriod := make(map[string]*DashboardCashflowBucket, len(buckets))
	for i := range buckets {
		bucketByPeriod[buckets[i].Period] = &buckets[i]
	}
	for _, item := range flow {
		bucket, ok := bucketByPeriod[item.Period]
		if !ok {
			continue
		}
		addDashboardAmount(&bucket.Income, item.Currency, item.Income)
		addDashboardAmount(&bucket.Expense, item.Currency, item.Expense)
		addDashboardAmount(&bucket.NetCashflow, item.Currency, item.Income.Sub(item.Expense))
		bucket.TransactionCount += item.TransactionCount
	}
	for i := range buckets {
		slices.SortFunc(buckets[i].Income, compareDashboardAmount)
		slices.SortFunc(buckets[i].Expense, compareDashboardAmount)
		slices.SortFunc(buckets[i].NetCashflow, compareDashboardAmount)
	}
	return &DashboardCashflow{GeneratedAt: now, Months: months, Buckets: buckets}
}

func BuildDashboardInterestIncome(now time.Time, flow []repository.DashboardMonthlyFlow, months int) *DashboardInterestIncome {
	buckets := dashboardInterestMonthBuckets(now, months)
	bucketByPeriod := make(map[string]*DashboardInterestIncomeBucket, len(buckets))
	for i := range buckets {
		bucketByPeriod[buckets[i].Period] = &buckets[i]
	}
	total := make(map[string]decimal.Decimal)
	for _, item := range flow {
		if item.InterestIncome.IsZero() {
			continue
		}
		bucket, ok := bucketByPeriod[item.Period]
		if !ok {
			continue
		}
		addDashboardAmount(&bucket.InterestIncome, item.Currency, item.InterestIncome)
		total[item.Currency] = total[item.Currency].Add(item.InterestIncome)
		bucket.TransactionCount += item.InterestCount
	}
	for i := range buckets {
		slices.SortFunc(buckets[i].InterestIncome, compareDashboardAmount)
	}
	return &DashboardInterestIncome{GeneratedAt: now, Months: months, Total: dashboardAmountsFromMap(total), Buckets: buckets}
}

func dashboardAccountBalance(item *repository.DashboardAccountBalance) DashboardAccountBalance {
	return DashboardAccountBalance{
		AccountID: item.Account.ID, Name: item.Account.Name, Bank: item.Account.Bank,
		Type: item.Account.Type, Currency: item.Account.Currency, IsActive: item.Account.IsActive,
		Balance: item.Balance, TransactionCount: item.TransactionCount,
	}
}

func dashboardAmountsFromMap(amounts map[string]decimal.Decimal) []DashboardAmount {
	currencies := make([]string, 0, len(amounts))
	for currency := range amounts {
		currencies = append(currencies, currency)
	}
	slices.Sort(currencies)
	response := make([]DashboardAmount, 0, len(currencies))
	for _, currency := range currencies {
		response = append(response, DashboardAmount{Currency: currency, Amount: amounts[currency]})
	}
	return response
}

func addDashboardAmount(amounts *[]DashboardAmount, currency string, delta decimal.Decimal) {
	if delta.IsZero() {
		return
	}
	for i := range *amounts {
		if (*amounts)[i].Currency == currency {
			(*amounts)[i].Amount = (*amounts)[i].Amount.Add(delta)
			return
		}
	}
	*amounts = append(*amounts, DashboardAmount{Currency: currency, Amount: delta})
}

func dashboardCashflowMonthBuckets(now time.Time, months int) []DashboardCashflowBucket {
	start := monthStart(now).AddDate(0, -(months - 1), 0)
	buckets := make([]DashboardCashflowBucket, 0, months)
	for i := range months {
		buckets = append(buckets, DashboardCashflowBucket{Period: start.AddDate(0, i, 0).Format("2006-01")})
	}
	return buckets
}

func dashboardInterestMonthBuckets(now time.Time, months int) []DashboardInterestIncomeBucket {
	start := monthStart(now).AddDate(0, -(months - 1), 0)
	buckets := make([]DashboardInterestIncomeBucket, 0, months)
	for i := range months {
		buckets = append(buckets, DashboardInterestIncomeBucket{Period: start.AddDate(0, i, 0).Format("2006-01")})
	}
	return buckets
}

func monthStart(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}

func dashboardCategoryCurrencyKey(categoryID, currency string) string {
	return categoryID + "\x00" + currency
}

func compareDashboardAmount(a, b DashboardAmount) int {
	return cmp.Compare(a.Currency, b.Currency)
}

func validateDashboardMonths(months int) error {
	if months <= 0 || months > 60 {
		return fmt.Errorf("dashboard months must be between 1 and 60")
	}
	return nil
}
