package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	domainaccount "github.com/sunriseex/capitalflow/internal/domain/account"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type AccountService struct {
	repo repository.AccountRepository
}

func NewAccountService(repos ...repository.AccountRepository) *AccountService {
	var repo repository.AccountRepository
	if len(repos) > 0 {
		repo = repos[0]
	}
	return &AccountService{repo: repo}
}

type CreateAccountRequest struct {
	OwnerUserID string
	Name        string
	Bank        string
	Type        models.AccountType
	Currency    string
	OpenedAt    time.Time
}

func (s *AccountService) Create(ctx context.Context, req *CreateAccountRequest) (*models.Account, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("create account: %w", ctx.Err())
	default:
	}
	if req == nil {
		return nil, validationError("create account request is required")
	}

	name := strings.TrimSpace(req.Name)
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "RUB"
	}
	currency = domainaccount.NormalizeCurrency(currency)
	if err := domainaccount.ValidateCreate(name, req.Type, currency); err != nil {
		return nil, validationError(err.Error())
	}

	openedAt := req.OpenedAt
	if openedAt.IsZero() {
		openedAt = time.Now()
	}
	now := time.Now()

	account := &models.Account{
		ID:          uuid.NewString(),
		OwnerUserID: ownerUserID(req.OwnerUserID),
		Name:        name,
		Bank:        strings.TrimSpace(req.Bank),
		Type:        req.Type,
		Currency:    currency,
		IsActive:    true,
		OpenedAt:    openedAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("account repository is required")
	}
	if err := s.repo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("save account: %w", err)
	}

	return account, nil
}

func ownerUserID(id string) *string {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	return &id
}

func ValidCurrency(currency string) bool {
	return domainaccount.ValidCurrency(currency)
}

func ValidAccountType(accountType models.AccountType) bool {
	return domainaccount.ValidAccountType(accountType)
}
