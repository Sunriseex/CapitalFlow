package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	domainmoney "github.com/sunriseex/capitalflow/internal/domain/money"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type FinancialGoalService struct {
	goals    repository.FinancialGoalRepository
	accounts repository.AccountRepository
}

type CreateFinancialGoalRequest struct {
	UserID       string
	AccountID    string
	Name         string
	TargetAmount decimal.Decimal
	TargetDate   *time.Time
}

type UpdateFinancialGoalRequest struct {
	ID            string
	UserID        string
	AccountID     *string
	Name          *string
	TargetAmount  *decimal.Decimal
	TargetDateSet bool
	TargetDate    *time.Time
	Status        *models.FinancialGoalStatus
}

func NewFinancialGoalService(goals repository.FinancialGoalRepository, accounts repository.AccountRepository) *FinancialGoalService {
	return &FinancialGoalService{goals: goals, accounts: accounts}
}

func (s *FinancialGoalService) ListByUser(ctx context.Context, userID string) ([]models.FinancialGoal, error) {
	if err := s.requireRepositories(); err != nil {
		return nil, err
	}
	goals, err := s.goals.ListByUser(ctx, strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("list financial goals: %w", err)
	}
	return goals, nil
}

func (s *FinancialGoalService) Create(ctx context.Context, req *CreateFinancialGoalRequest) (*models.FinancialGoal, error) {
	if err := s.requireRepositories(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, validationError("create financial goal request is required")
	}
	userID := strings.TrimSpace(req.UserID)
	accountID := strings.TrimSpace(req.AccountID)
	account, err := s.accounts.GetByIDForUser(ctx, accountID, userID)
	if err != nil {
		return nil, fmt.Errorf("get financial goal account: %w", err)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 100 || !req.TargetAmount.IsPositive() {
		return nil, validationError("name and a positive target amount are required")
	}
	if err := domainmoney.ValidateCurrencyScale(req.TargetAmount, account.Currency); err != nil {
		return nil, validationError(err.Error())
	}
	now := time.Now().UTC()
	goal := &models.FinancialGoal{
		ID: uuid.NewString(), OwnerUserID: userID, AccountID: &accountID, Name: name,
		TargetAmount: req.TargetAmount, Currency: account.Currency, TargetDate: req.TargetDate,
		Status: models.FinancialGoalActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.goals.Create(ctx, goal); err != nil {
		return nil, fmt.Errorf("create financial goal: %w", err)
	}
	return goal, nil
}

func (s *FinancialGoalService) Update(ctx context.Context, req *UpdateFinancialGoalRequest) (*models.FinancialGoal, error) {
	if err := s.requireRepositories(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, validationError("update financial goal request is required")
	}
	userID := strings.TrimSpace(req.UserID)
	goal, err := s.goals.GetByIDForUser(ctx, strings.TrimSpace(req.ID), userID)
	if err != nil {
		return nil, fmt.Errorf("get financial goal: %w", err)
	}
	accountChanged := req.AccountID != nil
	if accountChanged {
		accountID := strings.TrimSpace(*req.AccountID)
		account, err := s.accounts.GetByIDForUser(ctx, accountID, userID)
		if err != nil {
			return nil, fmt.Errorf("get financial goal account: %w", err)
		}
		goal.AccountID = &accountID
		goal.Currency = account.Currency
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || len([]rune(name)) > 100 {
			return nil, validationError("name must contain 1 to 100 characters")
		}
		goal.Name = name
	}
	if req.TargetAmount != nil {
		if !req.TargetAmount.IsPositive() {
			return nil, validationError("target amount must be positive")
		}
		goal.TargetAmount = *req.TargetAmount
	}
	if req.TargetAmount != nil || accountChanged {
		if err := domainmoney.ValidateCurrencyScale(goal.TargetAmount, goal.Currency); err != nil {
			return nil, validationError(err.Error())
		}
	}
	if req.TargetDateSet {
		goal.TargetDate = req.TargetDate
	}
	if req.Status != nil {
		if !validFinancialGoalStatus(*req.Status) {
			return nil, validationError("invalid financial goal status")
		}
		goal.Status = *req.Status
	}
	goal.UpdatedAt = time.Now().UTC()
	if err := s.goals.UpdateForUser(ctx, goal, userID); err != nil {
		return nil, fmt.Errorf("update financial goal: %w", err)
	}
	return goal, nil
}

func (s *FinancialGoalService) requireRepositories() error {
	if s == nil || s.goals == nil || s.accounts == nil {
		return fmt.Errorf("financial goal repositories are required")
	}
	return nil
}

func validFinancialGoalStatus(status models.FinancialGoalStatus) bool {
	switch status {
	case models.FinancialGoalActive, models.FinancialGoalCompleted, models.FinancialGoalArchived:
		return true
	default:
		return false
	}
}
