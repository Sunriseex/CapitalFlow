package services

import (
	"context"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestAccountServiceCreate(t *testing.T) {
	repo := &recordingAccountRepo{}
	account, err := NewAccountService(repo).Create(t.Context(), &CreateAccountRequest{
		Name:     "Savings",
		Bank:     "Yandex",
		Type:     models.AccountTypeSavings,
		Currency: "rub",
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if account.ID == "" {
		t.Fatal("id is empty")
	}
	if account.Currency != "RUB" {
		t.Fatalf("currency = %s, want RUB", account.Currency)
	}
	if !account.IsActive {
		t.Fatal("account must be active")
	}
	if repo.account == nil || repo.account.ID != account.ID {
		t.Fatal("repo did not receive account")
	}
}

func TestAccountServiceCreateStoresOwnerUserID(t *testing.T) {
	account, err := NewAccountService(&recordingAccountRepo{}).Create(t.Context(), &CreateAccountRequest{
		OwnerUserID: "user-1",
		Name:        "Savings",
		Type:        models.AccountTypeSavings,
		Currency:    "RUB",
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if account.OwnerUserID == nil || *account.OwnerUserID != "user-1" {
		t.Fatalf("owner user id = %v, want user-1", account.OwnerUserID)
	}
}

func TestAccountServiceCreateRejectsMissingRepository(t *testing.T) {
	_, err := NewAccountService().Create(t.Context(), &CreateAccountRequest{
		Name:     "Savings",
		Type:     models.AccountTypeSavings,
		Currency: "RUB",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if IsValidationError(err) {
		t.Fatalf("expected wiring error, got validation error: %v", err)
	}
}

func TestAccountServiceCreateValidatesInput(t *testing.T) {
	_, err := NewAccountService().Create(t.Context(), &CreateAccountRequest{
		Name: "Savings",
		Type: "invalid",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAccountServiceUpdateForUserValidatesAndPersists(t *testing.T) {
	repo := &recordingAccountRepo{
		existing: &models.Account{
			ID:        "account-1",
			Name:      "Old",
			Type:      models.AccountTypeSavings,
			Currency:  "RUB",
			IsActive:  true,
			OpenedAt:  time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			CreatedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	name := " New "
	currency := "usd"

	account, err := NewAccountService(repo).UpdateForUser(t.Context(), &UpdateAccountRequest{
		ID:       " account-1 ",
		UserID:   " user-1 ",
		Name:     &name,
		Currency: &currency,
	})
	if err != nil {
		t.Fatalf("update account: %v", err)
	}
	if account.Name != "New" || account.Currency != "USD" {
		t.Fatalf("account = %+v, want normalized update", account)
	}
	if repo.updateForUserID != "user-1" || repo.updatedAccount == nil || repo.updatedAccount.ID != "account-1" {
		t.Fatalf("repo update = user %q account %+v", repo.updateForUserID, repo.updatedAccount)
	}
}

func TestAccountServiceUpdateForUserRejectsInvalidCurrency(t *testing.T) {
	repo := &recordingAccountRepo{
		existing: &models.Account{
			ID:       "account-1",
			Name:     "Main",
			Type:     models.AccountTypeSavings,
			Currency: "RUB",
		},
	}
	currency := "BTC"

	_, err := NewAccountService(repo).UpdateForUser(t.Context(), &UpdateAccountRequest{
		ID:       "account-1",
		UserID:   "user-1",
		Currency: &currency,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
}

func TestAccountServiceArchiveForUser(t *testing.T) {
	repo := &recordingAccountRepo{}

	if err := NewAccountService(repo).ArchiveForUser(t.Context(), " account-1 ", " user-1 "); err != nil {
		t.Fatalf("archive account: %v", err)
	}
	if repo.archivedID != "account-1" || repo.archivedUserID != "user-1" {
		t.Fatalf("archive args = %q/%q, want account-1/user-1", repo.archivedID, repo.archivedUserID)
	}
}

type recordingAccountRepo struct {
	account         *models.Account
	existing        *models.Account
	updatedAccount  *models.Account
	updateForUserID string
	archivedID      string
	archivedUserID  string
}

func (r *recordingAccountRepo) Create(_ context.Context, account *models.Account) error {
	accountCopy := *account
	r.account = &accountCopy
	return nil
}

func (r *recordingAccountRepo) GetByID(context.Context, string) (*models.Account, error) {
	return nil, repository.ErrNotFound
}

func (r *recordingAccountRepo) GetByIDForUser(_ context.Context, id, _ string) (*models.Account, error) {
	if r.existing == nil || r.existing.ID != id {
		return nil, repository.ErrNotFound
	}
	accountCopy := *r.existing
	return &accountCopy, nil
}

func (r *recordingAccountRepo) GetByLegacyID(context.Context, string) (*models.Account, error) {
	return nil, repository.ErrNotFound
}

func (r *recordingAccountRepo) List(context.Context) ([]models.Account, error) {
	return nil, nil
}

func (r *recordingAccountRepo) ListByUser(context.Context, string) ([]models.Account, error) {
	return nil, nil
}

func (r *recordingAccountRepo) Update(context.Context, *models.Account) error {
	return nil
}

func (r *recordingAccountRepo) UpdateForUser(context.Context, *models.Account, string) error {
	return nil
}

func (r *recordingAccountRepo) UpdateForUserEnforcingCurrencyInvariant(_ context.Context, account *models.Account, userID string) error {
	accountCopy := *account
	r.updatedAccount = &accountCopy
	r.updateForUserID = userID
	return nil
}

func (r *recordingAccountRepo) Archive(context.Context, string) error {
	return nil
}

func (r *recordingAccountRepo) ArchiveForUser(_ context.Context, id, userID string) error {
	r.archivedID = id
	r.archivedUserID = userID
	return nil
}

func (r *recordingAccountRepo) ClaimUnowned(context.Context, string) error {
	return nil
}

func TestAccountServiceCreateValidatesCurrency(t *testing.T) {
	tests := []string{"RUB1", "RUR", "BTC"}

	for _, currency := range tests {
		t.Run(currency, func(t *testing.T) {
			_, err := NewAccountService().Create(t.Context(), &CreateAccountRequest{
				Name:     "Savings",
				Type:     models.AccountTypeSavings,
				Currency: currency,
			})
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestAccountServiceCreateReturnsValidationError(t *testing.T) {
	tests := []struct {
		name string
		req  *CreateAccountRequest
	}{
		{"nil request", nil},
		{"missing name", &CreateAccountRequest{Type: models.AccountTypeSavings, Currency: "RUB"}},
		{"invalid type", &CreateAccountRequest{Name: "Main", Type: models.AccountType("invalid"), Currency: "RUB"}},
		{"invalid currency", &CreateAccountRequest{Name: "Main", Type: models.AccountTypeSavings, Currency: "12$"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewAccountService()

			_, err := service.Create(context.Background(), tt.req)
			if err == nil {
				t.Fatal("expected error")
			}

			if !IsValidationError(err) {
				t.Fatalf("expected validation error, got %T: %v", err, err)
			}
		})
	}
}
