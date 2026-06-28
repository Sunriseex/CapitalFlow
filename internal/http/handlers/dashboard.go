package handlers

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/services"
	"github.com/sunriseex/capitalflow/pkg/money"
)

const (
	dashboardRecentTransactionsLimit = 10
	defaultDashboardMonths           = 6
)

func (h *Handler) getDashboardSummary(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	accounts, err := h.store.Accounts().ListByUser(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	transactions, err := h.store.Transactions().ListByUser(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	goals, err := h.store.FinancialGoals().ListByUser(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	limits, err := h.store.CategoryLimits().ListByUser(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	categories, err := h.store.Categories().List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	summary, err := buildDashboardSummary(
		r.Context(),
		time.Now(),
		accounts,
		transactions,
		goals,
		limits,
		categories,
		dashboardRecentTransactionsLimit,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) getDashboardNetWorth(w http.ResponseWriter, r *http.Request) {
	accounts, transactions, ok := h.dashboardData(w, r)
	if !ok {
		return
	}

	response, err := buildDashboardNetWorth(r.Context(), time.Now(), accounts, transactions)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) getDashboardCashflow(w http.ResponseWriter, r *http.Request) {
	months, ok := dashboardMonthsParam(w, r)
	if !ok {
		return
	}

	accounts, transactions, ok := h.dashboardData(w, r)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, buildDashboardCashflow(time.Now(), accounts, transactions, months))
}

func (h *Handler) getDashboardInterestIncome(w http.ResponseWriter, r *http.Request) {
	months, ok := dashboardMonthsParam(w, r)
	if !ok {
		return
	}

	accounts, transactions, ok := h.dashboardData(w, r)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, buildDashboardInterestIncome(time.Now(), accounts, transactions, months))
}

func (h *Handler) dashboardData(w http.ResponseWriter, r *http.Request) ([]models.Account, []models.Transaction, bool) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return nil, nil, false
	}

	accounts, err := h.store.Accounts().ListByUser(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return nil, nil, false
	}

	transactions, err := h.store.Transactions().ListByUser(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return nil, nil, false
	}

	return accounts, transactions, true
}

func dashboardMonthsParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("months")
	if raw == "" {
		return defaultDashboardMonths, true
	}

	months, err := strconv.Atoi(raw)
	if err != nil || months <= 0 || months > 60 {
		writeError(w, http.StatusBadRequest, "validation_error", "months must be between 1 and 60", nil)
		return 0, false
	}

	return months, true
}

func buildDashboardSummary(
	ctx context.Context,
	now time.Time,
	accounts []models.Account,
	transactions []models.Transaction,
	goals []models.FinancialGoal,
	limits []models.CategoryLimit,
	categories []models.Category,
	recentLimit int,
) (*dto.DashboardSummaryResponse, error) {
	if recentLimit < 0 {
		recentLimit = 0
	}

	accountByID := make(map[string]models.Account, len(accounts))
	for i := range accounts {
		accountByID[accounts[i].ID] = accounts[i]
	}

	summary := &dto.DashboardSummaryResponse{
		GeneratedAt:                now,
		AccountsCount:              len(accounts),
		RecentTransactionsLimit:    recentLimit,
		RecentTransactionsReturned: min(recentLimit, len(transactions)),
	}

	balances := make(map[string]decimal.Decimal)
	income := make(map[string]decimal.Decimal)
	expense := make(map[string]decimal.Decimal)
	interestIncome := make(map[string]decimal.Decimal)
	accountBalances := make(map[string]decimal.Decimal, len(accounts))

	for i := range accounts {
		account := &accounts[i]
		if account.IsActive {
			summary.ActiveAccountsCount++
		}
		balance, err := services.NewBalanceService().Calculate(ctx, services.CalculateBalanceRequest{
			AccountID:    account.ID,
			Transactions: transactions,
		})
		if err != nil {
			return nil, fmt.Errorf("calculate dashboard account balance: %w", err)
		}
		accountBalances[account.ID] = balance.Balance

		if account.IsActive {
			balances[account.Currency] = balances[account.Currency].Add(balance.Balance)
		}

		summary.AccountBalances = append(summary.AccountBalances, dto.DashboardAccountBalanceResponse{
			AccountID:        account.ID,
			Name:             account.Name,
			Bank:             account.Bank,
			Type:             account.Type,
			Currency:         account.Currency,
			IsActive:         account.IsActive,
			Balance:          money.NewJSONDecimal(balance.Balance),
			TransactionCount: balance.Count,
		})
	}

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	nextMonthStart := monthStart.AddDate(0, 1, 0)
	categoryExpense := make(map[string]decimal.Decimal)
	for i := range transactions {
		tx := &transactions[i]
		if tx.OccurredAt.Before(monthStart) || !tx.OccurredAt.Before(nextMonthStart) {
			continue
		}

		account, ok := accountByID[tx.AccountID]
		if !ok {
			continue
		}

		switch tx.Type {
		case models.TransactionTypeIncome:
			income[account.Currency] = income[account.Currency].Add(tx.Amount)
		case models.TransactionTypeExpense:
			expense[account.Currency] = expense[account.Currency].Add(tx.Amount)
			if tx.CategoryID != nil {
				key := dashboardCategoryCurrencyKey(*tx.CategoryID, account.Currency)
				categoryExpense[key] = categoryExpense[key].Add(tx.Amount)
			}
		case models.TransactionTypeInterestIncome:
			income[account.Currency] = income[account.Currency].Add(tx.Amount)
			interestIncome[account.Currency] = interestIncome[account.Currency].Add(tx.Amount)
		}
	}

	for i := range goals {
		goal := &goals[i]
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
		summary.FinancialGoals = append(summary.FinancialGoals, dto.DashboardGoalProgressResponse{
			ID:            goal.ID,
			AccountID:     *goal.AccountID,
			Name:          goal.Name,
			CurrentAmount: money.NewJSONDecimal(balance),
			TargetAmount:  money.NewJSONDecimal(goal.TargetAmount),
			Currency:      goal.Currency,
			TargetDate:    targetDate,
			Status:        goal.Status,
		})
	}

	categoryNames := make(map[string]string, len(categories))
	for i := range categories {
		categoryNames[categories[i].ID] = categories[i].Name
	}
	for i := range limits {
		limit := &limits[i]
		if !limit.IsActive {
			continue
		}
		summary.CategoryLimits = append(summary.CategoryLimits, dto.DashboardCategoryLimitProgressResponse{
			ID:            limit.ID,
			CategoryID:    limit.CategoryID,
			CategoryName:  categoryNames[limit.CategoryID],
			CurrentAmount: money.NewJSONDecimal(categoryExpense[dashboardCategoryCurrencyKey(limit.CategoryID, limit.Currency)]),
			TargetAmount:  money.NewJSONDecimal(limit.Amount),
			Currency:      limit.Currency,
		})
	}

	recent := slices.Clone(transactions)
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
	summary.RecentTransactions = dto.TransactionsFromModels(recent)
	summary.RecentTransactionsReturned = len(recent)

	return summary, nil
}

func dashboardCategoryCurrencyKey(categoryID, currency string) string {
	return categoryID + "\x00" + currency
}

func buildDashboardNetWorth(ctx context.Context, now time.Time, accounts []models.Account, transactions []models.Transaction) (*dto.DashboardNetWorthResponse, error) {
	balances := make(map[string]decimal.Decimal)
	response := &dto.DashboardNetWorthResponse{
		GeneratedAt: now,
	}

	for i := range accounts {
		account := &accounts[i]
		balance, err := services.NewBalanceService().Calculate(ctx, services.CalculateBalanceRequest{
			AccountID:    account.ID,
			Transactions: transactions,
		})
		if err != nil {
			return nil, fmt.Errorf("calculate dashboard net worth account balance: %w", err)
		}

		if account.IsActive {
			balances[account.Currency] = balances[account.Currency].Add(balance.Balance)
		}

		response.AccountBalances = append(response.AccountBalances, dto.DashboardAccountBalanceResponse{
			AccountID:        account.ID,
			Name:             account.Name,
			Bank:             account.Bank,
			Type:             account.Type,
			Currency:         account.Currency,
			IsActive:         account.IsActive,
			Balance:          money.NewJSONDecimal(balance.Balance),
			TransactionCount: balance.Count,
		})
	}

	response.Balances = dashboardAmountsFromMap(balances)
	return response, nil
}

func buildDashboardCashflow(now time.Time, accounts []models.Account, transactions []models.Transaction, months int) dto.DashboardCashflowResponse {
	accountByID := dashboardAccountByID(accounts)
	buckets := dashboardMonthBuckets(now, months)
	bucketByPeriod := make(map[string]*dto.DashboardCashflowBucketResponse, len(buckets))
	for i := range buckets {
		bucketByPeriod[buckets[i].Period] = &buckets[i]
	}

	for i := range transactions {
		tx := &transactions[i]
		period := tx.OccurredAt.Format("2006-01")
		bucket, ok := bucketByPeriod[period]
		if !ok {
			continue
		}

		account, ok := accountByID[tx.AccountID]
		if !ok {
			continue
		}

		switch tx.Type {
		case models.TransactionTypeIncome, models.TransactionTypeInterestIncome:
			addDashboardAmount(&bucket.Income, account.Currency, tx.Amount)
			addDashboardAmount(&bucket.NetCashflow, account.Currency, tx.Amount)
			bucket.TransactionCount++
		case models.TransactionTypeExpense:
			addDashboardAmount(&bucket.Expense, account.Currency, tx.Amount)
			addDashboardAmount(&bucket.NetCashflow, account.Currency, tx.Amount.Neg())
			bucket.TransactionCount++
		}
	}

	return dto.DashboardCashflowResponse{
		GeneratedAt: now,
		Months:      months,
		Buckets:     normalizeCashflowBuckets(buckets),
	}
}

func buildDashboardInterestIncome(now time.Time, accounts []models.Account, transactions []models.Transaction, months int) dto.DashboardInterestIncomeResponse {
	accountByID := dashboardAccountByID(accounts)
	buckets := dashboardInterestMonthBuckets(now, months)
	bucketByPeriod := make(map[string]*dto.DashboardInterestIncomeBucketResponse, len(buckets))
	for i := range buckets {
		bucketByPeriod[buckets[i].Period] = &buckets[i]
	}

	total := make(map[string]decimal.Decimal)
	for i := range transactions {
		tx := &transactions[i]
		if tx.Type != models.TransactionTypeInterestIncome {
			continue
		}

		period := tx.OccurredAt.Format("2006-01")
		bucket, ok := bucketByPeriod[period]
		if !ok {
			continue
		}

		account, ok := accountByID[tx.AccountID]
		if !ok {
			continue
		}

		addDashboardAmount(&bucket.InterestIncome, account.Currency, tx.Amount)
		total[account.Currency] = total[account.Currency].Add(tx.Amount)
		bucket.TransactionCount++
	}

	return dto.DashboardInterestIncomeResponse{
		GeneratedAt: now,
		Months:      months,
		Total:       dashboardAmountsFromMap(total),
		Buckets:     normalizeInterestIncomeBuckets(buckets),
	}
}

func dashboardAmountsFromMap(amounts map[string]decimal.Decimal) []dto.DashboardAmountResponse {
	currencies := make([]string, 0, len(amounts))
	for currency := range amounts {
		currencies = append(currencies, currency)
	}
	slices.Sort(currencies)

	response := make([]dto.DashboardAmountResponse, 0, len(currencies))
	for _, currency := range currencies {
		response = append(response, dto.DashboardAmountResponse{
			Currency: currency,
			Amount:   money.NewJSONDecimal(amounts[currency]),
		})
	}
	return response
}

func dashboardAccountByID(accounts []models.Account) map[string]models.Account {
	accountByID := make(map[string]models.Account, len(accounts))
	for i := range accounts {
		accountByID[accounts[i].ID] = accounts[i]
	}
	return accountByID
}

func dashboardMonthBuckets(now time.Time, months int) []dto.DashboardCashflowBucketResponse {
	if months <= 0 {
		months = defaultDashboardMonths
	}

	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -(months - 1), 0)
	buckets := make([]dto.DashboardCashflowBucketResponse, 0, months)
	for i := range months {
		buckets = append(buckets, dto.DashboardCashflowBucketResponse{
			Period: start.AddDate(0, i, 0).Format("2006-01"),
		})
	}
	return buckets
}

func dashboardInterestMonthBuckets(now time.Time, months int) []dto.DashboardInterestIncomeBucketResponse {
	if months <= 0 {
		months = defaultDashboardMonths
	}

	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -(months - 1), 0)
	buckets := make([]dto.DashboardInterestIncomeBucketResponse, 0, months)
	for i := range months {
		buckets = append(buckets, dto.DashboardInterestIncomeBucketResponse{
			Period: start.AddDate(0, i, 0).Format("2006-01"),
		})
	}
	return buckets
}

func addDashboardAmount(amounts *[]dto.DashboardAmountResponse, currency string, delta decimal.Decimal) {
	for i := range *amounts {
		if (*amounts)[i].Currency == currency {
			(*amounts)[i].Amount = money.NewJSONDecimal((*amounts)[i].Amount.Add(delta))
			return
		}
	}

	*amounts = append(*amounts, dto.DashboardAmountResponse{
		Currency: currency,
		Amount:   money.NewJSONDecimal(delta),
	})
}

func normalizeCashflowBuckets(buckets []dto.DashboardCashflowBucketResponse) []dto.DashboardCashflowBucketResponse {
	for i := range buckets {
		slices.SortFunc(buckets[i].Income, compareDashboardAmount)
		slices.SortFunc(buckets[i].Expense, compareDashboardAmount)
		slices.SortFunc(buckets[i].NetCashflow, compareDashboardAmount)
	}
	return buckets
}

func normalizeInterestIncomeBuckets(buckets []dto.DashboardInterestIncomeBucketResponse) []dto.DashboardInterestIncomeBucketResponse {
	for i := range buckets {
		slices.SortFunc(buckets[i].InterestIncome, compareDashboardAmount)
	}
	return buckets
}

func compareDashboardAmount(a, b dto.DashboardAmountResponse) int {
	return cmp.Compare(a.Currency, b.Currency)
}
