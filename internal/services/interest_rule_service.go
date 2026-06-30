package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	domaininterest "github.com/sunriseex/capitalflow/internal/domain/interest"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/pkg/money"
)

type InterestRuleService struct {
	rules    repository.InterestRuleRepository
	accounts repository.AccountRepository
}

func NewInterestRuleService(rules repository.InterestRuleRepository, accounts repository.AccountRepository) *InterestRuleService {
	return &InterestRuleService{rules: rules, accounts: accounts}
}

// InterestEngine calculates accruals without reading or writing persistence.
type InterestEngine struct{}

func NewInterestEngine() *InterestEngine { return &InterestEngine{} }

type CreateInterestRuleRequest struct {
	UserID                  string
	AccountID               string
	AnnualRateBps           int64
	PromoRateBps            *int64
	PromoEndDate            *time.Time
	AccrualFrequency        models.AccrualFrequency
	CapitalizationFrequency models.CapitalizationFrequency
	DayCountConvention      models.DayCountConvention
	StartDate               time.Time
	EndDate                 *time.Time
}

type AccrueRuleInterestRequest struct {
	Rule             models.InterestRule
	Currency         string
	Balance          decimal.Decimal
	AccrualDate      time.Time
	Transactions     []models.Transaction
	ExistingAccruals []models.InterestAccrual
}

type AccrueRuleInterestResponse struct {
	Transaction *models.Transaction
	Accrual     *models.InterestAccrual
	Skipped     bool
}

type RecalculateRuleInterestRequest struct {
	Rule             models.InterestRule
	Currency         string
	Transactions     []models.Transaction
	ExistingAccruals []models.InterestAccrual
	FromDate         time.Time
	ToDate           time.Time
	Today            time.Time
}

type RecalculateRuleInterestResponse struct {
	AccountID       string
	RuleID          string
	FromDate        time.Time
	ToDate          time.Time
	DeletedAccruals int64
	CreatedAccruals int64
	SkippedDays     int64
	TotalAmount     decimal.Decimal
	Transactions    []models.Transaction
	Accruals        []models.InterestAccrual
}

type ForecastRuleInterestRequest struct {
	Rule             models.InterestRule
	Currency         string
	Transactions     []models.Transaction
	ExistingAccruals []models.InterestAccrual
	FromDate         time.Time
	Days             int
	Today            time.Time
}

type ForecastRuleInterestResponse struct {
	AccountID        string
	RuleID           string
	FromDate         time.Time
	ToDate           time.Time
	Days             int
	ProjectedAmount  decimal.Decimal
	ProjectedBalance decimal.Decimal
	Accruals         []models.InterestAccrual
}

func (s *InterestRuleService) Create(ctx context.Context, req *CreateInterestRuleRequest) (*models.InterestRule, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("create interest rule: %w", ctx.Err())
	default:
	}
	if req == nil {
		return nil, validationError("create interest rule request is required")
	}

	accountID := strings.TrimSpace(req.AccountID)
	if accountID == "" {
		return nil, validationError("account id is required")
	}
	if strings.TrimSpace(req.UserID) != "" && s != nil && s.accounts != nil {
		if _, err := s.accounts.GetByIDForUser(ctx, accountID, strings.TrimSpace(req.UserID)); err != nil {
			return nil, fmt.Errorf("get interest rule account: %w", err)
		}
	}
	if req.AnnualRateBps <= 0 {
		return nil, validationError("annual rate must be positive")
	}
	if req.PromoRateBps != nil && *req.PromoRateBps <= 0 {
		return nil, validationError("promo rate must be positive")
	}
	if req.PromoRateBps != nil && req.PromoEndDate == nil {
		return nil, validationError("promo end date is required when promo rate is set")
	}
	if req.PromoRateBps == nil && req.PromoEndDate != nil {
		return nil, validationError("promo rate is required when promo end date is set")
	}

	accrualFrequency := req.AccrualFrequency
	if accrualFrequency == "" {
		accrualFrequency = models.AccrualFrequencyDaily
	}
	if !domaininterest.ValidAccrualFrequency(accrualFrequency) {
		return nil, validationError(fmt.Sprintf("invalid accrual frequency: %s", accrualFrequency))
	}

	capitalizationFrequency := req.CapitalizationFrequency
	if capitalizationFrequency == "" {
		capitalizationFrequency = models.CapitalizationFrequencyNone
	}
	if !domaininterest.ValidCapitalizationFrequency(capitalizationFrequency) {
		return nil, validationError(fmt.Sprintf("invalid capitalization frequency: %s", capitalizationFrequency))
	}

	dayCountConvention := req.DayCountConvention
	if dayCountConvention == "" {
		dayCountConvention = models.DayCountConventionActual365
	}
	if !domaininterest.ValidDayCountConvention(dayCountConvention) {
		return nil, validationError(fmt.Sprintf("invalid day count convention: %s", dayCountConvention))
	}

	startDate := dateOnly(req.StartDate)
	if startDate.IsZero() {
		startDate = dateOnly(time.Now())
	}
	if req.EndDate != nil && dateOnly(*req.EndDate).Before(startDate) {
		return nil, validationError("end date must be on or after start date")
	}
	if req.PromoEndDate != nil && dateOnly(*req.PromoEndDate).Before(startDate) {
		return nil, validationError("promo end date must be on or after start date")
	}

	var endDate *time.Time
	if req.EndDate != nil {
		normalized := dateOnly(*req.EndDate)
		endDate = &normalized
	}

	var promoEndDate *time.Time
	if req.PromoEndDate != nil {
		normalized := dateOnly(*req.PromoEndDate)
		promoEndDate = &normalized
	}

	rule := &models.InterestRule{
		ID:                      uuid.NewString(),
		AccountID:               accountID,
		AnnualRateBps:           req.AnnualRateBps,
		PromoRateBps:            req.PromoRateBps,
		PromoEndDate:            promoEndDate,
		AccrualFrequency:        accrualFrequency,
		CapitalizationFrequency: capitalizationFrequency,
		DayCountConvention:      dayCountConvention,
		IsActive:                true,
		StartDate:               startDate,
		EndDate:                 endDate,
	}

	if s == nil || s.rules == nil {
		return nil, fmt.Errorf("interest rule repository is required")
	}
	if err := s.rules.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("save interest rule: %w", err)
	}

	return rule, nil
}

func (e *InterestEngine) Accrue(ctx context.Context, req *AccrueRuleInterestRequest) (*AccrueRuleInterestResponse, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("accrue interest: %w", ctx.Err())
	default:
	}

	if req == nil {
		return nil, validationError("accrue interest request is required")
	}

	if err := validateRuleForAccrual(&req.Rule); err != nil {
		return nil, err
	}

	accrualDate := dateOnly(req.AccrualDate)
	if accrualDate.IsZero() {
		accrualDate = dateOnly(time.Now())
	}

	if !ruleActiveOn(&req.Rule, accrualDate) {
		return nil, validationError(fmt.Sprintf("interest rule is not active on %s", accrualDate.Format(time.DateOnly)))
	}

	if !shouldPostAccrual(&req.Rule, accrualDate) {
		return nil, validationError(fmt.Sprintf("interest rule is not payable on %s", accrualDate.Format(time.DateOnly)))
	}

	if hasInterestAccrual(req.ExistingAccruals, &req.Rule, accrualDate) {
		return &AccrueRuleInterestResponse{Skipped: true}, nil
	}

	periodStart := nextAccrualPeriodStart(&req.Rule, req.ExistingAccruals, accrualDate)
	calculationTransactions := PrincipalTransactionsForRuleAt(req.Transactions, req.ExistingAccruals, &req.Rule, accrualDate)
	currency := interestCurrency(req.Currency)
	amount, balance, rateBps, err := calculateAccrualAmount(ctx, &req.Rule, currency, calculationTransactions, req.Balance, periodStart, accrualDate)
	if err != nil {
		return nil, err
	}
	if !amount.IsPositive() {
		return nil, validationError("calculated interest is zero")
	}

	tx, err := buildTransaction(ctx, &CreateTransactionRequest{
		AccountID:       req.Rule.AccountID,
		Type:            models.TransactionTypeInterestIncome,
		Amount:          amount,
		Currency:        currency,
		Description:     interestAccrualDescription(req.Rule.ID, accrualDate),
		OccurredAt:      accrualDate,
		AllowFutureDate: true,
	}, false)
	if err != nil {
		return nil, fmt.Errorf("build interest income transaction: %w", err)
	}

	accrual := &models.InterestAccrual{
		ID:            uuid.NewString(),
		AccountID:     req.Rule.AccountID,
		RuleID:        req.Rule.ID,
		TransactionID: tx.ID,
		AccrualDate:   accrualDate,
		Amount:        amount,
		Balance:       balance,
		AnnualRateBps: rateBps,
		CreatedAt:     time.Now(),
	}

	return &AccrueRuleInterestResponse{
		Transaction: tx,
		Accrual:     accrual,
	}, nil
}

func (e *InterestEngine) Recalculate(ctx context.Context, req *RecalculateRuleInterestRequest) (*RecalculateRuleInterestResponse, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("recalculate interest: %w", ctx.Err())
	default:
	}

	if req == nil {
		return nil, validationError("recalculate interest request is required")
	}
	if err := validateRuleForRecalculation(&req.Rule); err != nil {
		return nil, err
	}
	fromDate := dateOnly(req.FromDate)
	if fromDate.IsZero() {
		fromDate = dateOnly(req.Rule.StartDate)
	}
	toDate := dateOnly(req.ToDate)
	if toDate.IsZero() {
		toDate = dateOnly(req.Today)
		if toDate.IsZero() {
			toDate = dateOnly(time.Now())
		}
	}
	if fromDate.IsZero() {
		return nil, validationError("from date is required")
	}
	if toDate.Before(fromDate) {
		return nil, validationError("to date must be on or after from date")
	}
	currency := interestCurrency(req.Currency)

	calculationFromDate := recalculationStartDate(&req.Rule, fromDate, toDate)
	baseTransactions := excludeAccrualTransactions(req.Transactions, req.ExistingAccruals, &req.Rule, fromDate, toDate)
	workingTransactions := PrincipalTransactionsForRuleAt(baseTransactions, req.ExistingAccruals, &req.Rule, calculationFromDate)
	response := &RecalculateRuleInterestResponse{
		AccountID: req.Rule.AccountID,
		RuleID:    req.Rule.ID,
		FromDate:  fromDate,
		ToDate:    toDate,
	}

	pendingAmount := decimal.Zero
	pendingBalance := decimal.Zero
	var pendingRate int64
	pendingCapitalization := pendingCapitalizationTransactionsBefore(baseTransactions, req.ExistingAccruals, &req.Rule, calculationFromDate, toDate)

	for day := calculationFromDate; !day.After(toDate); day = day.AddDate(0, 0, 1) {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("recalculate interest: %w", ctx.Err())
		default:
		}

		if !ruleActiveOn(&req.Rule, day) {
			response.SkippedDays++
			continue
		}

		balance, err := NewBalanceService().Calculate(ctx, CalculateBalanceRequest{
			AccountID:    req.Rule.AccountID,
			Transactions: transactionsUpToDate(workingTransactions, day),
		})
		if err != nil {
			return nil, fmt.Errorf("calculate balance for interest recalculation: %w", err)
		}
		if !balance.Balance.IsPositive() {
			response.SkippedDays++
		} else {
			rateBps := effectiveRateBps(&req.Rule, day)
			amount := calculateDailyInterestAmount(balance.Balance, rateBps, req.Rule.DayCountConvention, day, currency)
			if !amount.IsPositive() {
				response.SkippedDays++
			} else {
				pendingAmount = pendingAmount.Add(amount)
				pendingBalance = balance.Balance
				pendingRate = rateBps
			}
		}

		if !shouldPostAccrual(&req.Rule, day) || !pendingAmount.IsPositive() {
			continue
		}

		tx, err := buildTransaction(ctx, &CreateTransactionRequest{
			AccountID:       req.Rule.AccountID,
			Type:            models.TransactionTypeInterestIncome,
			Amount:          pendingAmount,
			Currency:        currency,
			Description:     interestAccrualDescription(req.Rule.ID, day),
			OccurredAt:      day,
			AllowFutureDate: true,
		}, false)
		if err != nil {
			return nil, fmt.Errorf("build recalculated interest transaction: %w", err)
		}

		accrual := models.InterestAccrual{
			ID:            uuid.NewString(),
			AccountID:     req.Rule.AccountID,
			RuleID:        req.Rule.ID,
			TransactionID: tx.ID,
			AccrualDate:   day,
			Amount:        pendingAmount,
			Balance:       pendingBalance,
			AnnualRateBps: pendingRate,
			CreatedAt:     time.Now(),
		}

		response.Transactions = append(response.Transactions, *tx)
		response.Accruals = append(response.Accruals, accrual)
		response.CreatedAccruals++
		response.TotalAmount = response.TotalAmount.Add(pendingAmount)

		switch {
		case req.Rule.CapitalizationFrequency == models.CapitalizationFrequencyDaily:
			workingTransactions = append(workingTransactions, *tx)
		case shouldCapitalizeOn(&req.Rule, day):
			pendingCapitalization = append(pendingCapitalization, *tx)
			workingTransactions = append(workingTransactions, pendingCapitalization...)
			pendingCapitalization = nil
		case req.Rule.CapitalizationFrequency != models.CapitalizationFrequencyNone &&
			req.Rule.CapitalizationFrequency != "":
			pendingCapitalization = append(pendingCapitalization, *tx)
		}
		pendingAmount = decimal.Zero
		pendingBalance = decimal.Zero
	}

	return response, nil
}

func (e *InterestEngine) Forecast(ctx context.Context, req *ForecastRuleInterestRequest) (*ForecastRuleInterestResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("forecast interest: %w", err)
	}
	if req == nil {
		return nil, validationError("forecast interest request is required")
	}
	if err := validateRuleForRecalculation(&req.Rule); err != nil {
		return nil, err
	}
	if req.Days <= 0 {
		return nil, validationError("forecast days must be positive")
	}

	fromDate := dateOnly(req.FromDate)
	if fromDate.IsZero() {
		fromDate = dateOnly(req.Today)
		if fromDate.IsZero() {
			fromDate = dateOnly(time.Now())
		}
	}
	toDate := fromDate.AddDate(0, 0, req.Days-1)

	forecastRule := req.Rule
	forecastRule.AccrualFrequency = models.AccrualFrequencyDaily
	result, err := e.Recalculate(ctx, &RecalculateRuleInterestRequest{
		Rule:             forecastRule,
		Currency:         req.Currency,
		Transactions:     req.Transactions,
		ExistingAccruals: req.ExistingAccruals,
		FromDate:         fromDate,
		ToDate:           toDate,
		Today:            req.Today,
	})
	if err != nil {
		return nil, err
	}

	balance, err := NewBalanceService().Calculate(ctx, CalculateBalanceRequest{
		AccountID:    req.Rule.AccountID,
		Transactions: append(transactionsUpToDate(PrincipalTransactionsForRuleAt(req.Transactions, req.ExistingAccruals, &req.Rule, toDate), toDate), result.Transactions...),
	})
	if err != nil {
		return nil, fmt.Errorf("calculate forecast balance: %w", err)
	}

	return &ForecastRuleInterestResponse{
		AccountID:        req.Rule.AccountID,
		RuleID:           req.Rule.ID,
		FromDate:         fromDate,
		ToDate:           toDate,
		Days:             req.Days,
		ProjectedAmount:  result.TotalAmount,
		ProjectedBalance: balance.Balance,
		Accruals:         result.Accruals,
	}, nil
}

// PrincipalTransactionsForRule excludes this rule's uncapitalized accrual transactions from principal calculations.
func PrincipalTransactionsForRule(
	transactions []models.Transaction,
	accruals []models.InterestAccrual,
	rule *models.InterestRule,
) []models.Transaction {
	return PrincipalTransactionsForRuleAt(transactions, accruals, rule, time.Time{})
}

// PrincipalTransactionsForRuleAt excludes accrual transactions that are not capitalized by asOfDate.
func PrincipalTransactionsForRuleAt(
	transactions []models.Transaction,
	accruals []models.InterestAccrual,
	rule *models.InterestRule,
	asOfDate time.Time,
) []models.Transaction {
	asOfDate = dateOnly(asOfDate)
	capitalizedTransactionIDs := make(map[string]struct{})
	uncapitalizedTransactionIDs := make(map[string]struct{})

	for i := range accruals {
		accrual := &accruals[i]
		if accrual.AccountID != rule.AccountID || accrual.RuleID != rule.ID {
			continue
		}
		if accrualCapitalizedBy(rule, accrual.AccrualDate, asOfDate) {
			capitalizedTransactionIDs[accrual.TransactionID] = struct{}{}
			continue
		}
		uncapitalizedTransactionIDs[accrual.TransactionID] = struct{}{}
	}

	filtered := make([]models.Transaction, 0, len(transactions))
	for i := range transactions {
		if _, ok := capitalizedTransactionIDs[transactions[i].ID]; ok {
			filtered = append(filtered, transactions[i])
			continue
		}
		if _, ok := uncapitalizedTransactionIDs[transactions[i].ID]; ok {
			continue
		}
		filtered = append(filtered, transactions[i])
	}

	return filtered
}

func accrualCapitalizedBy(rule *models.InterestRule, accrualDate, asOfDate time.Time) bool {
	if asOfDate.IsZero() {
		return shouldCapitalizeOn(rule, accrualDate)
	}
	capitalizationDate, ok := capitalizationDateForAccrual(rule, accrualDate)
	return ok && capitalizationDate.Before(asOfDate)
}

func validateRuleForAccrual(rule *models.InterestRule) error {
	if strings.TrimSpace(rule.ID) == "" {
		return validationError("interest rule id is required")
	}
	if strings.TrimSpace(rule.AccountID) == "" {
		return validationError("account id is required")
	}
	if !rule.IsActive {
		return validationError("interest rule is inactive")
	}
	if rule.AnnualRateBps <= 0 {
		return validationError("annual rate must be positive")
	}
	if err := domaininterest.ValidateFrequencies(rule.AccrualFrequency, rule.CapitalizationFrequency, rule.DayCountConvention); err != nil {
		return validationError(err.Error())
	}
	return nil
}

func excludeAccrualTransactions(transactions []models.Transaction, accruals []models.InterestAccrual, rule *models.InterestRule, fromDate, toDate time.Time) []models.Transaction {
	replacedTransactionIDs := make(map[string]struct{})
	for i := range accruals {
		accrual := &accruals[i]
		accrualDate := dateOnly(accrual.AccrualDate)
		if accrual.AccountID == rule.AccountID &&
			accrual.RuleID == rule.ID &&
			!accrualDate.Before(fromDate) &&
			!accrualDate.After(toDate) {
			replacedTransactionIDs[accrual.TransactionID] = struct{}{}
		}
	}

	filtered := make([]models.Transaction, 0, len(transactions))
	for i := range transactions {
		if _, ok := replacedTransactionIDs[transactions[i].ID]; ok {
			continue
		}
		filtered = append(filtered, transactions[i])
	}
	return filtered
}

func pendingCapitalizationTransactionsBefore(
	transactions []models.Transaction,
	accruals []models.InterestAccrual,
	rule *models.InterestRule,
	fromDate,
	toDate time.Time,
) []models.Transaction {
	if rule.CapitalizationFrequency == models.CapitalizationFrequencyNone ||
		rule.CapitalizationFrequency == "" ||
		rule.CapitalizationFrequency == models.CapitalizationFrequencyDaily {
		return nil
	}

	fromDate = dateOnly(fromDate)
	toDate = dateOnly(toDate)
	transactionByID := make(map[string]models.Transaction, len(transactions))
	for i := range transactions {
		transactionByID[transactions[i].ID] = transactions[i]
	}

	pending := make([]models.Transaction, 0)
	for i := range accruals {
		accrual := &accruals[i]
		accrualDate := dateOnly(accrual.AccrualDate)
		if accrual.AccountID != rule.AccountID ||
			accrual.RuleID != rule.ID ||
			!accrualDate.Before(fromDate) {
			continue
		}

		capitalizationDate, ok := capitalizationDateForAccrual(rule, accrualDate)
		if !ok || capitalizationDate.Before(fromDate) || capitalizationDate.After(toDate) {
			continue
		}

		if tx, ok := transactionByID[accrual.TransactionID]; ok {
			pending = append(pending, tx)
		}
	}
	return pending
}

func transactionsUpToDate(transactions []models.Transaction, date time.Time) []models.Transaction {
	date = dateOnly(date)
	filtered := make([]models.Transaction, 0, len(transactions))
	for i := range transactions {
		if !dateOnly(transactions[i].OccurredAt).After(date) {
			filtered = append(filtered, transactions[i])
		}
	}
	return filtered
}

func calculateAccrualAmount(ctx context.Context, rule *models.InterestRule, currency string, transactions []models.Transaction, balance decimal.Decimal, fromDate, toDate time.Time) (amount, finalBalance decimal.Decimal, finalRateBps int64, err error) {
	if toDate.Before(fromDate) {
		return decimal.Zero, decimal.Zero, 0, validationError("accrual period is empty")
	}
	if len(transactions) == 0 {
		if !balance.IsPositive() {
			return decimal.Zero, decimal.Zero, 0, validationError("balance must be positive")
		}
		total := decimal.Zero
		var lastRate int64
		for day := fromDate; !day.After(toDate); day = day.AddDate(0, 0, 1) {
			select {
			case <-ctx.Done():
				return decimal.Zero, decimal.Zero, 0, fmt.Errorf("calculate accrual amount: %w", ctx.Err())
			default:
			}

			if !ruleActiveOn(rule, day) {
				continue
			}
			rateBps := effectiveRateBps(rule, day)
			total = total.Add(calculateDailyInterestAmount(balance, rateBps, rule.DayCountConvention, day, currency))
			lastRate = rateBps
		}
		return total, balance, lastRate, nil
	}

	total := decimal.Zero
	lastBalance := decimal.Zero
	var lastRate int64
	for day := fromDate; !day.After(toDate); day = day.AddDate(0, 0, 1) {
		select {
		case <-ctx.Done():
			return decimal.Zero, decimal.Zero, 0, fmt.Errorf("calculate accrual amount: %w", ctx.Err())
		default:
		}

		if !ruleActiveOn(rule, day) {
			continue
		}
		balance, err := NewBalanceService().Calculate(ctx, CalculateBalanceRequest{
			AccountID:    rule.AccountID,
			Transactions: transactionsUpToDate(transactions, day),
		})
		if err != nil {
			return decimal.Zero, decimal.Zero, 0, fmt.Errorf("calculate accrual balance: %w", err)
		}
		if !balance.Balance.IsPositive() {
			continue
		}
		rateBps := effectiveRateBps(rule, day)
		total = total.Add(calculateDailyInterestAmount(balance.Balance, rateBps, rule.DayCountConvention, day, currency))
		lastBalance = balance.Balance
		lastRate = rateBps
	}
	return total, lastBalance, lastRate, nil
}

func calculateDailyInterestAmount(balance decimal.Decimal, rateBps int64, convention models.DayCountConvention, date time.Time, currency string) decimal.Decimal {
	rate := decimal.NewFromInt(rateBps).Div(decimal.NewFromInt(10_000))
	days := decimal.NewFromInt(int64(daysInYear(convention, date)))

	return money.RoundForCurrency(balance.Mul(rate).Div(days), interestCurrency(currency))
}

func interestCurrency(currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return "RUB"
	}
	return currency
}

func nextAccrualPeriodStart(rule *models.InterestRule, accruals []models.InterestAccrual, accrualDate time.Time) time.Time {
	if rule.AccrualFrequency == models.AccrualFrequencyDaily {
		return dateOnly(accrualDate)
	}

	start := dateOnly(rule.StartDate)
	for i := range accruals {
		accrual := &accruals[i]
		if accrual.AccountID != rule.AccountID || accrual.RuleID != rule.ID {
			continue
		}
		date := dateOnly(accrual.AccrualDate)
		if date.Before(accrualDate) && !date.Before(start) {
			start = date.AddDate(0, 0, 1)
		}
	}
	return start
}

func recalculationStartDate(rule *models.InterestRule, fromDate, toDate time.Time) time.Time {
	fromDate = dateOnly(fromDate)
	toDate = dateOnly(toDate)
	if rule.AccrualFrequency == models.AccrualFrequencyDaily {
		return fromDate
	}
	for day := fromDate; !day.After(toDate); day = day.AddDate(0, 0, 1) {
		if shouldPostAccrual(rule, day) {
			return minDate(fromDate, accrualPeriodStart(rule, day))
		}
	}
	return fromDate
}

func accrualPeriodStart(rule *models.InterestRule, date time.Time) time.Time {
	date = dateOnly(date)
	start := dateOnly(rule.StartDate)
	switch rule.AccrualFrequency {
	case models.AccrualFrequencyMonthly:
		monthStart := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
		return maxDate(start, monthStart)
	case models.AccrualFrequencyEndOfTerm:
		return start
	default:
		return date
	}
}

func capitalizationDateForAccrual(rule *models.InterestRule, accrualDate time.Time) (time.Time, bool) {
	accrualDate = dateOnly(accrualDate)
	if !ruleActiveOn(rule, accrualDate) {
		return time.Time{}, false
	}
	switch rule.CapitalizationFrequency {
	case models.CapitalizationFrequencyDaily:
		return accrualDate, true
	case models.CapitalizationFrequencyMonthly:
		return lastActiveDayOfMonth(rule, accrualDate), true
	case models.CapitalizationFrequencyEndOfTerm:
		if rule.EndDate == nil {
			return time.Time{}, false
		}
		return dateOnly(*rule.EndDate), true
	default:
		return time.Time{}, false
	}
}

func shouldPostAccrual(rule *models.InterestRule, date time.Time) bool {
	date = dateOnly(date)
	if !ruleActiveOn(rule, date) {
		return false
	}

	switch rule.AccrualFrequency {
	case models.AccrualFrequencyMonthly:
		return isLastActiveDayOfMonth(rule, date)
	case models.AccrualFrequencyEndOfTerm:
		return isEndOfTerm(rule, date)
	default:
		return true
	}
}

func shouldCapitalizeOn(rule *models.InterestRule, date time.Time) bool {
	switch rule.CapitalizationFrequency {
	case models.CapitalizationFrequencyDaily:
		return true
	case models.CapitalizationFrequencyMonthly:
		return isLastActiveDayOfMonth(rule, date)
	case models.CapitalizationFrequencyEndOfTerm:
		return isEndOfTerm(rule, date)
	default:
		return false
	}
}

func isLastActiveDayOfMonth(rule *models.InterestRule, date time.Time) bool {
	return lastActiveDayOfMonth(rule, date).Equal(dateOnly(date))
}

func lastActiveDayOfMonth(rule *models.InterestRule, date time.Time) time.Time {
	date = dateOnly(date)
	monthEnd := time.Date(date.Year(), date.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	if rule.EndDate != nil {
		return minDate(monthEnd, dateOnly(*rule.EndDate))
	}
	return monthEnd
}

func isEndOfTerm(rule *models.InterestRule, date time.Time) bool {
	return rule.EndDate != nil && dateOnly(*rule.EndDate).Equal(dateOnly(date))
}

func minDate(a, b time.Time) time.Time {
	a = dateOnly(a)
	b = dateOnly(b)
	if a.Before(b) {
		return a
	}
	return b
}

func maxDate(a, b time.Time) time.Time {
	a = dateOnly(a)
	b = dateOnly(b)
	if a.After(b) {
		return a
	}
	return b
}

func daysInYear(convention models.DayCountConvention, date time.Time) int {
	switch convention {
	case models.DayCountConventionActual366:
		return 366
	case models.DayCountConventionActualActual:
		if isLeapYear(date.Year()) {
			return 366
		}
		return 365
	default:
		return 365
	}
}

func effectiveRateBps(rule *models.InterestRule, date time.Time) int64 {
	if rule.PromoRateBps != nil && rule.PromoEndDate != nil && !date.After(dateOnly(*rule.PromoEndDate)) {
		return *rule.PromoRateBps
	}
	return rule.AnnualRateBps
}

func ruleActiveOn(rule *models.InterestRule, date time.Time) bool {
	if date.Before(dateOnly(rule.StartDate)) {
		return false
	}
	if rule.EndDate != nil && date.After(dateOnly(*rule.EndDate)) {
		return false
	}
	return true
}

func hasInterestAccrual(accruals []models.InterestAccrual, rule *models.InterestRule, date time.Time) bool {
	for i := range accruals {
		accrual := &accruals[i]
		if accrual.AccountID == rule.AccountID &&
			accrual.RuleID == rule.ID &&
			dateOnly(accrual.AccrualDate).Equal(date) {
			return true
		}
	}
	return false
}

func interestAccrualDescription(ruleID string, date time.Time) string {
	return fmt.Sprintf("interest accrual rule=%s date=%s", ruleID, date.Format(time.DateOnly))
}

func dateOnly(date time.Time) time.Time {
	if date.IsZero() {
		return time.Time{}
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func validateRuleForRecalculation(rule *models.InterestRule) error {
	if strings.TrimSpace(rule.ID) == "" {
		return validationError("interest rule id is required")
	}
	if strings.TrimSpace(rule.AccountID) == "" {
		return validationError("account id is required")
	}
	if rule.AnnualRateBps <= 0 {
		return validationError("annual rate must be positive")
	}
	if err := domaininterest.ValidateFrequencies(rule.AccrualFrequency, rule.CapitalizationFrequency, rule.DayCountConvention); err != nil {
		return validationError(err.Error())
	}
	return nil
}
