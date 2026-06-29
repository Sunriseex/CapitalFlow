package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	domainaccount "github.com/sunriseex/capitalflow/internal/domain/account"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type AccountService struct {
	repo         repository.AccountRepository
	transactions repository.TransactionRepository
}

func (s *AccountService) WithTransactionRepository(repo repository.TransactionRepository) *AccountService {
	s.transactions = repo
	return s
}

type AccountBalance struct {
	AccountID        string
	Balance          decimal.Decimal
	TransactionCount int64
}

func (s *AccountService) ListByUser(ctx context.Context, userID string) ([]models.Account, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("account repository is required")
	}
	accounts, err := s.repo.ListByUser(ctx, strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	return accounts, nil
}

func (s *AccountService) GetByIDForUser(ctx context.Context, accountID, userID string) (*models.Account, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("account repository is required")
	}
	account, err := s.repo.GetByIDForUser(ctx, strings.TrimSpace(accountID), strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	return account, nil
}

func (s *AccountService) BalanceForUser(ctx context.Context, accountID, userID string) (*AccountBalance, error) {
	accountID = strings.TrimSpace(accountID)
	userID = strings.TrimSpace(userID)
	if _, err := s.GetByIDForUser(ctx, accountID, userID); err != nil {
		return nil, err
	}
	if s.transactions == nil {
		return nil, fmt.Errorf("transaction repository is required")
	}
	balance, count, err := s.transactions.GetBalanceByAccountForUser(ctx, accountID, userID)
	if err != nil {
		return nil, fmt.Errorf("get account balance: %w", err)
	}
	return &AccountBalance{AccountID: accountID, Balance: balance, TransactionCount: count}, nil
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

type UpdateAccountRequest struct {
	ID       string
	UserID   string
	Name     *string
	Bank     *string
	Type     *models.AccountType
	Currency *string
	OpenedAt *time.Time
	IsActive *bool
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

func (s *AccountService) UpdateForUser(ctx context.Context, req *UpdateAccountRequest) (*models.Account, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("update account: %w", ctx.Err())
	default:
	}
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("account repository is required")
	}
	if req == nil {
		return nil, validationError("update account request is required")
	}

	accountID := strings.TrimSpace(req.ID)
	userID := strings.TrimSpace(req.UserID)
	if accountID == "" {
		return nil, validationError("account id is required")
	}
	if userID == "" {
		return nil, validationError("user is required")
	}

	account, err := s.repo.GetByIDForUser(ctx, accountID, userID)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	if req.Name != nil {
		account.Name = strings.TrimSpace(*req.Name)
	}
	if req.Bank != nil {
		account.Bank = strings.TrimSpace(*req.Bank)
	}
	if req.Type != nil {
		account.Type = *req.Type
	}
	if req.Currency != nil {
		account.Currency = domainaccount.NormalizeCurrency(*req.Currency)
	}
	if req.OpenedAt != nil && !req.OpenedAt.IsZero() {
		account.OpenedAt = *req.OpenedAt
	}
	if req.IsActive != nil {
		account.IsActive = *req.IsActive
	}

	if err := domainaccount.ValidateCreate(account.Name, account.Type, account.Currency); err != nil {
		return nil, validationError(err.Error())
	}
	account.UpdatedAt = time.Now()

	if err := s.repo.UpdateForUserEnforcingCurrencyInvariant(ctx, account, userID); err != nil {
		return nil, fmt.Errorf("update account: %w", err)
	}
	return account, nil
}

func (s *AccountService) ArchiveForUser(ctx context.Context, id, userID string) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("archive account: %w", ctx.Err())
	default:
	}
	if s == nil || s.repo == nil {
		return fmt.Errorf("account repository is required")
	}
	id = strings.TrimSpace(id)
	userID = strings.TrimSpace(userID)
	if id == "" {
		return validationError("account id is required")
	}
	if userID == "" {
		return validationError("user is required")
	}
	if err := s.repo.ArchiveForUser(ctx, id, userID); err != nil {
		return fmt.Errorf("archive account: %w", err)
	}
	return nil
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
