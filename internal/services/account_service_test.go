package services

import (
	"context"
	"testing"

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

type recordingAccountRepo struct {
	account *models.Account
}

func (r *recordingAccountRepo) Create(_ context.Context, account *models.Account) error {
	accountCopy := *account
	r.account = &accountCopy
	return nil
}

func (r *recordingAccountRepo) GetByID(context.Context, string) (*models.Account, error) {
	return nil, repository.ErrNotFound
}

func (r *recordingAccountRepo) GetByIDForUser(context.Context, string, string) (*models.Account, error) {
	return nil, repository.ErrNotFound
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

func (r *recordingAccountRepo) UpdateForUserEnforcingCurrencyInvariant(context.Context, *models.Account, string) error {
	return nil
}

func (r *recordingAccountRepo) Archive(context.Context, string) error {
	return nil
}

func (r *recordingAccountRepo) ArchiveForUser(context.Context, string, string) error {
	return nil
}

func (r *recordingAccountRepo) ClaimUnowned(context.Context, string) error {
	return nil
}

func TestAccountServiceCreateValidatesCurrency(t *testing.T) {
	_, err := NewAccountService().Create(t.Context(), &CreateAccountRequest{
		Name:     "Savings",
		Type:     models.AccountTypeSavings,
		Currency: "RUB1",
	})
	if err == nil {
		t.Fatal("expected error")
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
