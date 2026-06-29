package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	domainmoney "github.com/sunriseex/capitalflow/internal/domain/money"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

type CategoryLimitService struct {
	limits     repository.CategoryLimitRepository
	categories repository.CategoryRepository
}

type CreateCategoryLimitRequest struct {
	UserID     string
	CategoryID string
	Amount     decimal.Decimal
	Currency   string
	IsActive   *bool
}

type UpdateCategoryLimitRequest struct {
	ID         string
	UserID     string
	CategoryID *string
	Amount     *decimal.Decimal
	Currency   *string
	IsActive   *bool
}

func NewCategoryLimitService(limits repository.CategoryLimitRepository, categories repository.CategoryRepository) *CategoryLimitService {
	return &CategoryLimitService{limits: limits, categories: categories}
}

func (s *CategoryLimitService) ListByUser(ctx context.Context, userID string) ([]models.CategoryLimit, error) {
	if err := s.requireRepositories(); err != nil {
		return nil, err
	}
	limits, err := s.limits.ListByUser(ctx, strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("list category limits: %w", err)
	}
	return limits, nil
}

func (s *CategoryLimitService) Create(ctx context.Context, req *CreateCategoryLimitRequest) (*models.CategoryLimit, error) {
	if err := s.requireRepositories(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, validationError("create category limit request is required")
	}
	categoryID := strings.TrimSpace(req.CategoryID)
	if _, err := s.categories.GetByID(ctx, categoryID); err != nil {
		return nil, fmt.Errorf("get category limit category: %w", err)
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if err := validateCategoryLimitAmount(req.Amount, currency); err != nil {
		return nil, err
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	now := time.Now().UTC()
	limit := &models.CategoryLimit{
		ID: uuid.NewString(), OwnerUserID: strings.TrimSpace(req.UserID), CategoryID: categoryID,
		Amount: req.Amount, Currency: currency, IsActive: isActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.limits.Create(ctx, limit); err != nil {
		return nil, fmt.Errorf("create category limit: %w", err)
	}
	return limit, nil
}

func (s *CategoryLimitService) Update(ctx context.Context, req *UpdateCategoryLimitRequest) (*models.CategoryLimit, error) {
	if err := s.requireRepositories(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, validationError("update category limit request is required")
	}
	userID := strings.TrimSpace(req.UserID)
	limit, err := s.limits.GetByIDForUser(ctx, strings.TrimSpace(req.ID), userID)
	if err != nil {
		return nil, fmt.Errorf("get category limit: %w", err)
	}
	if req.CategoryID != nil {
		categoryID := strings.TrimSpace(*req.CategoryID)
		if _, err := s.categories.GetByID(ctx, categoryID); err != nil {
			return nil, fmt.Errorf("get category limit category: %w", err)
		}
		limit.CategoryID = categoryID
	}
	if req.Currency != nil {
		limit.Currency = strings.ToUpper(strings.TrimSpace(*req.Currency))
	}
	if req.Amount != nil {
		limit.Amount = *req.Amount
	}
	if req.Amount != nil || req.Currency != nil {
		if err := validateCategoryLimitAmount(limit.Amount, limit.Currency); err != nil {
			return nil, err
		}
	}
	if req.IsActive != nil {
		limit.IsActive = *req.IsActive
	}
	limit.UpdatedAt = time.Now().UTC()
	if err := s.limits.UpdateForUser(ctx, limit, userID); err != nil {
		return nil, fmt.Errorf("update category limit: %w", err)
	}
	return limit, nil
}

func (s *CategoryLimitService) requireRepositories() error {
	if s == nil || s.limits == nil || s.categories == nil {
		return fmt.Errorf("category limit repositories are required")
	}
	return nil
}

func validateCategoryLimitAmount(amount decimal.Decimal, currency string) error {
	if !currencyCodePattern.MatchString(currency) {
		return validationError("currency must be a 3-letter code")
	}
	if !amount.IsPositive() {
		return validationError("limit amount must be positive")
	}
	if err := domainmoney.ValidateCurrencyScale(amount, currency); err != nil {
		return validationError(err.Error())
	}
	return nil
}
