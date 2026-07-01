package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

// InterestLifecycle owns rule selection, calculation, and persistence for one
// account-level interest operation.
type InterestLifecycle struct {
	repo       repository.InterestAccrualTransactionalRepository
	accounts   interestAccountReader
	categories interestCategoryReader
	engine     *InterestEngine
}

type interestAccountReader interface {
	GetByIDForUser(ctx context.Context, id, userID string) (*models.Account, error)
}

type interestCategoryReader interface {
	GetBySlug(ctx context.Context, slug string) (*models.Category, error)
}

func (l *InterestLifecycle) WithCategoryRepository(repo interestCategoryReader) *InterestLifecycle {
	l.categories = repo
	return l
}

func (l *InterestLifecycle) WithAccountRepository(repo interestAccountReader) *InterestLifecycle {
	l.accounts = repo
	return l
}

func NewInterestLifecycle(repo repository.InterestAccrualTransactionalRepository, engine *InterestEngine) *InterestLifecycle {
	return &InterestLifecycle{repo: repo, engine: engine}
}

type AccrueAccountInterestRequest struct {
	AccountID   string
	UserID      string
	Currency    string
	RuleID      string
	AccrualDate time.Time
}

type RecalculateAccountInterestRequest struct {
	AccountID string
	UserID    string
	Currency  string
	RuleID    string
	RuleDate  time.Time
	FromDate  time.Time
	ToDate    time.Time
}

func (l *InterestLifecycle) Accrue(ctx context.Context, req *AccrueAccountInterestRequest) (*AccrueRuleInterestResponse, error) {
	if l == nil || l.repo == nil {
		return nil, fmt.Errorf("accrue account interest: transactional interest repository is required")
	}
	if l.engine == nil {
		return nil, fmt.Errorf("accrue account interest: interest engine is required")
	}
	if req == nil {
		return nil, validationError("accrue account interest request is required")
	}
	if err := validateInterestLifecycleAccount(req.AccountID, req.UserID); err != nil {
		return nil, err
	}
	account, category, err := l.interestTransactionMetadata(ctx, req.AccountID, req.UserID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Currency) == "" {
		req.Currency = account.Currency
	}

	accrualDate := dateOnly(req.AccrualDate)
	if accrualDate.IsZero() {
		accrualDate = dateOnly(time.Now())
	}

	var result *AccrueRuleInterestResponse
	err = l.repo.WithAccountInterestLock(ctx, req.AccountID, req.UserID, func(ctx context.Context, snapshot repository.InterestCalculationRepository) error {
		rule, err := selectInterestRule(ctx, snapshot, req.AccountID, req.RuleID, accrualDate, true)
		if err != nil {
			return err
		}

		transactions, accruals, err := loadInterestSnapshot(ctx, snapshot, req.AccountID, req.UserID)
		if err != nil {
			return err
		}
		transactions = PrincipalTransactionsForRuleAt(transactionsUpToDate(transactions, accrualDate), accruals, rule, accrualDate)

		balance, err := NewBalanceService().Calculate(ctx, CalculateBalanceRequest{
			AccountID:    req.AccountID,
			Transactions: transactions,
		})
		if err != nil {
			return fmt.Errorf("calculate account balance: %w", err)
		}

		calculated, err := l.engine.Accrue(ctx, &AccrueRuleInterestRequest{
			Rule:             *rule,
			AccountName:      account.Name,
			CategoryID:       category.ID,
			Currency:         req.Currency,
			Balance:          balance.Balance,
			AccrualDate:      accrualDate,
			Transactions:     transactions,
			ExistingAccruals: accruals,
		})
		if err != nil {
			return fmt.Errorf("calculate account interest: %w", err)
		}
		if !calculated.Skipped {
			if err := snapshot.CreateInterestAccrualWithTransaction(ctx, calculated.Transaction, calculated.Accrual); err != nil {
				return fmt.Errorf("create account interest accrual: %w", err)
			}
		}
		result = calculated
		return nil
	})
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return &AccrueRuleInterestResponse{Skipped: true}, nil
		}
		return nil, fmt.Errorf("accrue account interest: %w", err)
	}
	return result, nil
}

func (l *InterestLifecycle) Recalculate(ctx context.Context, req *RecalculateAccountInterestRequest) (*RecalculateRuleInterestResponse, error) {
	if l == nil || l.repo == nil {
		return nil, fmt.Errorf("recalculate account interest: transactional interest repository is required")
	}
	if l.engine == nil {
		return nil, fmt.Errorf("recalculate account interest: interest engine is required")
	}
	if req == nil {
		return nil, validationError("recalculate account interest request is required")
	}
	if err := validateInterestLifecycleAccount(req.AccountID, req.UserID); err != nil {
		return nil, err
	}
	account, category, err := l.interestTransactionMetadata(ctx, req.AccountID, req.UserID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Currency) == "" {
		req.Currency = account.Currency
	}

	ruleDate := dateOnly(req.RuleDate)
	if ruleDate.IsZero() {
		ruleDate = dateOnly(req.ToDate)
	}
	if ruleDate.IsZero() {
		ruleDate = dateOnly(time.Now())
	}

	var result *RecalculateRuleInterestResponse
	err = l.repo.WithAccountInterestLock(ctx, req.AccountID, req.UserID, func(ctx context.Context, snapshot repository.InterestCalculationRepository) error {
		rule, err := selectInterestRule(ctx, snapshot, req.AccountID, req.RuleID, ruleDate, false)
		if err != nil {
			return err
		}
		transactions, accruals, err := loadInterestSnapshot(ctx, snapshot, req.AccountID, req.UserID)
		if err != nil {
			return err
		}

		calculated, err := l.engine.Recalculate(ctx, &RecalculateRuleInterestRequest{
			Rule:             *rule,
			AccountName:      account.Name,
			CategoryID:       category.ID,
			Currency:         req.Currency,
			Transactions:     transactions,
			ExistingAccruals: accruals,
			FromDate:         req.FromDate,
			ToDate:           req.ToDate,
			Today:            ruleDate,
		})
		if err != nil {
			return fmt.Errorf("calculate account interest range: %w", err)
		}

		deleted, err := snapshot.ReplaceInterestAccrualRangeWithTransactions(
			ctx,
			calculated.AccountID,
			calculated.RuleID,
			calculated.FromDate,
			calculated.ToDate,
			calculated.Transactions,
			calculated.Accruals,
		)
		if err != nil {
			return fmt.Errorf("replace account interest range: %w", err)
		}
		calculated.DeletedAccruals = deleted
		result = calculated
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("recalculate account interest: %w", err)
	}
	return result, nil
}

func (l *InterestLifecycle) interestTransactionMetadata(ctx context.Context, accountID, userID string) (*models.Account, *models.Category, error) {
	if l.accounts == nil {
		return nil, nil, fmt.Errorf("interest account repository is required")
	}
	if l.categories == nil {
		return nil, nil, fmt.Errorf("interest category repository is required")
	}
	account, err := l.accounts.GetByIDForUser(ctx, accountID, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("get interest account: %w", err)
	}
	category, err := l.categories.GetBySlug(ctx, "deposit_interest")
	if err != nil {
		return nil, nil, fmt.Errorf("get deposit interest category: %w", err)
	}
	return account, category, nil
}

func validateInterestLifecycleAccount(accountID, userID string) error {
	if strings.TrimSpace(accountID) == "" {
		return validationError("account id is required")
	}
	if strings.TrimSpace(userID) == "" {
		return validationError("user is required")
	}
	return nil
}

func loadInterestSnapshot(ctx context.Context, snapshot repository.InterestCalculationRepository, accountID, userID string) ([]models.Transaction, []models.InterestAccrual, error) {
	transactions, err := snapshot.ListTransactionsByAccountForUser(ctx, accountID, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("list account transactions: %w", err)
	}
	accruals, err := snapshot.ListInterestAccrualsByAccount(ctx, accountID)
	if err != nil {
		return nil, nil, fmt.Errorf("list account interest accruals: %w", err)
	}
	return transactions, accruals, nil
}

func selectInterestRule(ctx context.Context, snapshot repository.InterestCalculationRepository, accountID, ruleID string, date time.Time, requireActive bool) (*models.InterestRule, error) {
	ruleID = strings.TrimSpace(ruleID)
	if ruleID != "" {
		rule, err := snapshot.GetInterestRuleByID(ctx, ruleID)
		if err != nil {
			return nil, fmt.Errorf("get interest rule: %w", err)
		}
		if rule.AccountID != accountID {
			return nil, repository.ErrNotFound
		}
		if requireActive && (!rule.IsActive || !ruleActiveOn(rule, date)) {
			return nil, validationError("interest rule is not active on " + date.Format(time.DateOnly))
		}
		return rule, nil
	}

	rules, err := snapshot.ListInterestRulesByAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("list account interest rules: %w", err)
	}
	var selected *models.InterestRule
	for i := range rules {
		rule := &rules[i]
		if !rule.IsActive || !ruleActiveOn(rule, date) {
			continue
		}
		if selected == nil || dateOnly(rule.StartDate).After(dateOnly(selected.StartDate)) {
			selected = rule
		}
	}
	if selected == nil {
		return nil, repository.ErrNotFound
	}
	return selected, nil
}
