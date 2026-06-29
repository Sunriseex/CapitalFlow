package handlers

import (
	"net/http"
	"strconv"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/services"
	"github.com/sunriseex/capitalflow/pkg/money"
)

const defaultDashboardMonths = 6

func (h *Handler) getDashboardSummary(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	report, err := h.app.Dashboard.Summary(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboardSummaryResponse(report))
}

func (h *Handler) getDashboardNetWorth(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	report, err := h.app.Dashboard.NetWorth(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboardNetWorthResponse(report))
}

func (h *Handler) getDashboardCashflow(w http.ResponseWriter, r *http.Request) {
	months, ok := dashboardMonthsParam(w, r)
	if !ok {
		return
	}
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	report, err := h.app.Dashboard.Cashflow(r.Context(), userID, months)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboardCashflowResponse(report))
}

func (h *Handler) getDashboardInterestIncome(w http.ResponseWriter, r *http.Request) {
	months, ok := dashboardMonthsParam(w, r)
	if !ok {
		return
	}
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	report, err := h.app.Dashboard.InterestIncome(r.Context(), userID, months)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboardInterestIncomeResponse(report))
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

func dashboardSummaryResponse(report *services.DashboardSummary) dto.DashboardSummaryResponse {
	return dto.DashboardSummaryResponse{
		GeneratedAt: report.GeneratedAt, AccountsCount: report.AccountsCount,
		ActiveAccountsCount: report.ActiveAccountsCount, Balances: dashboardAmountsResponse(report.Balances),
		MonthlyIncome: dashboardAmountsResponse(report.MonthlyIncome), MonthlyExpense: dashboardAmountsResponse(report.MonthlyExpense),
		MonthlyInterestIncome:   dashboardAmountsResponse(report.MonthlyInterestIncome),
		AccountBalances:         dashboardAccountBalancesResponse(report.AccountBalances),
		FinancialGoals:          dashboardGoalProgressResponse(report.FinancialGoals),
		CategoryLimits:          dashboardCategoryLimitProgressResponse(report.CategoryLimits),
		RecentTransactions:      dto.TransactionsFromModels(report.RecentTransactions),
		RecentTransactionsLimit: report.RecentTransactionsLimit, RecentTransactionsReturned: report.RecentTransactionsReturned,
	}
}

func dashboardNetWorthResponse(report *services.DashboardNetWorth) dto.DashboardNetWorthResponse {
	return dto.DashboardNetWorthResponse{
		GeneratedAt:     report.GeneratedAt,
		Balances:        dashboardAmountsResponse(report.Balances),
		AccountBalances: dashboardAccountBalancesResponse(report.AccountBalances),
	}
}

func dashboardCashflowResponse(report *services.DashboardCashflow) dto.DashboardCashflowResponse {
	response := dto.DashboardCashflowResponse{GeneratedAt: report.GeneratedAt, Months: report.Months}
	for _, bucket := range report.Buckets {
		response.Buckets = append(response.Buckets, dto.DashboardCashflowBucketResponse{
			Period: bucket.Period, Income: dashboardAmountsResponse(bucket.Income),
			Expense: dashboardAmountsResponse(bucket.Expense), NetCashflow: dashboardAmountsResponse(bucket.NetCashflow),
			TransactionCount: bucket.TransactionCount,
		})
	}
	return response
}

func dashboardInterestIncomeResponse(report *services.DashboardInterestIncome) dto.DashboardInterestIncomeResponse {
	response := dto.DashboardInterestIncomeResponse{
		GeneratedAt: report.GeneratedAt, Months: report.Months, Total: dashboardAmountsResponse(report.Total),
	}
	for _, bucket := range report.Buckets {
		response.Buckets = append(response.Buckets, dto.DashboardInterestIncomeBucketResponse{
			Period: bucket.Period, InterestIncome: dashboardAmountsResponse(bucket.InterestIncome),
			TransactionCount: bucket.TransactionCount,
		})
	}
	return response
}

func dashboardAmountsResponse(amounts []services.DashboardAmount) []dto.DashboardAmountResponse {
	response := make([]dto.DashboardAmountResponse, 0, len(amounts))
	for _, amount := range amounts {
		response = append(response, dto.DashboardAmountResponse{Currency: amount.Currency, Amount: money.NewJSONDecimal(amount.Amount)})
	}
	return response
}

func dashboardAccountBalancesResponse(items []services.DashboardAccountBalance) []dto.DashboardAccountBalanceResponse {
	response := make([]dto.DashboardAccountBalanceResponse, 0, len(items))
	for _, item := range items {
		response = append(response, dto.DashboardAccountBalanceResponse{
			AccountID: item.AccountID, Name: item.Name, Bank: item.Bank, Type: item.Type,
			Currency: item.Currency, IsActive: item.IsActive, Balance: money.NewJSONDecimal(item.Balance),
			TransactionCount: item.TransactionCount,
		})
	}
	return response
}

func dashboardGoalProgressResponse(items []services.DashboardGoalProgress) []dto.DashboardGoalProgressResponse {
	response := make([]dto.DashboardGoalProgressResponse, 0, len(items))
	for _, item := range items {
		response = append(response, dto.DashboardGoalProgressResponse{
			ID: item.ID, AccountID: item.AccountID, Name: item.Name,
			CurrentAmount: money.NewJSONDecimal(item.CurrentAmount), TargetAmount: money.NewJSONDecimal(item.TargetAmount),
			Currency: item.Currency, TargetDate: item.TargetDate, Status: item.Status,
		})
	}
	return response
}

func dashboardCategoryLimitProgressResponse(items []services.DashboardCategoryLimitProgress) []dto.DashboardCategoryLimitProgressResponse {
	response := make([]dto.DashboardCategoryLimitProgressResponse, 0, len(items))
	for _, item := range items {
		response = append(response, dto.DashboardCategoryLimitProgressResponse{
			ID: item.ID, CategoryID: item.CategoryID, CategoryName: item.CategoryName,
			CurrentAmount: money.NewJSONDecimal(item.CurrentAmount), TargetAmount: money.NewJSONDecimal(item.TargetAmount),
			Currency: item.Currency,
		})
	}
	return response
}
