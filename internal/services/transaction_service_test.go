package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestTransactionServiceCreatePreservesExplicitSource(t *testing.T) {
	refID := "11111111-1111-1111-1111-111111111111"
	tx, err := NewTransactionService(&recordingCreateForUserRepo{}).Create(t.Context(), &CreateTransactionRequest{
		AccountID:      "account-1",
		Type:           models.TransactionTypeIncome,
		Amount:         dec("100"),
		SourceType:     models.TransactionSourceCSVImport,
		SourceRefID:    &refID,
		SourceMetadata: json.RawMessage(`{"parser_version":"1"}`),
	})
	if err != nil {
		t.Fatalf("create sourced transaction: %v", err)
	}
	if tx.SourceType != models.TransactionSourceCSVImport {
		t.Fatalf("source type = %q, want csv_import", tx.SourceType)
	}
	if tx.SourceRefID == nil || *tx.SourceRefID != refID {
		t.Fatalf("source ref = %v, want %q", tx.SourceRefID, refID)
	}
	if string(tx.SourceMetadata) != `{"parser_version":"1"}` {
		t.Fatalf("source metadata = %s", tx.SourceMetadata)
	}
}

func TestTransactionServiceCreateRejectsInvalidSource(t *testing.T) {
	invalidRef := "not-a-uuid"
	tests := []struct {
		name     string
		source   models.TransactionSource
		refID    *string
		metadata json.RawMessage
	}{
		{name: "unknown type", source: "unknown"},
		{name: "invalid reference", source: models.TransactionSourceCSVImport, refID: &invalidRef},
		{name: "invalid json", source: models.TransactionSourceManual, metadata: json.RawMessage(`{`)},
		{name: "non-object metadata", source: models.TransactionSourceManual, metadata: json.RawMessage(`[]`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewTransactionService(&recordingCreateForUserRepo{}).Create(t.Context(), &CreateTransactionRequest{
				AccountID:      "account-1",
				Type:           models.TransactionTypeIncome,
				Amount:         dec("100"),
				SourceType:     test.source,
				SourceRefID:    test.refID,
				SourceMetadata: test.metadata,
			})
			if err == nil || !IsValidationError(err) {
				t.Fatalf("error = %v, want validation error", err)
			}
		})
	}
}

func TestTransactionServiceCreate(t *testing.T) {
	repo := &recordingCreateForUserRepo{}
	tx, err := NewTransactionService(repo).Create(t.Context(), &CreateTransactionRequest{
		AccountID:   "account-1",
		Type:        models.TransactionTypeIncome,
		Amount:      dec("100"),
		Description: " Salary ",
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	if tx.ID == "" {
		t.Fatal("id is empty")
	}
	if tx.Description != "Salary" {
		t.Fatalf("description = %q, want Salary", tx.Description)
	}
	if tx.SourceType != models.TransactionSourceManual {
		t.Fatalf("source type = %q, want manual", tx.SourceType)
	}
	if tx.OccurredAt.IsZero() {
		t.Fatal("occurred at is zero")
	}
	if repo.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", repo.createCalls)
	}
}

func TestTransactionServiceCreateForUser(t *testing.T) {
	repo := &recordingCreateForUserRepo{}
	tx, err := NewTransactionService(repo).CreateForUser(t.Context(), " user-1 ", &CreateTransactionRequest{
		AccountID:   "account-1",
		Type:        models.TransactionTypeIncome,
		Amount:      dec("100"),
		Description: " Salary ",
	})
	if err != nil {
		t.Fatalf("create transaction for user: %v", err)
	}
	if tx.ID == "" {
		t.Fatal("id is empty")
	}
	if repo.createCalls != 0 {
		t.Fatalf("old Create calls = %d, want 0", repo.createCalls)
	}
	if repo.createForUserCalls != 1 {
		t.Fatalf("CreateForUser calls = %d, want 1", repo.createForUserCalls)
	}
	if repo.userID != "user-1" {
		t.Fatalf("userID = %q, want user-1", repo.userID)
	}
	if repo.transaction == nil || repo.transaction.ID != tx.ID {
		t.Fatal("repo did not receive created transaction")
	}
}

func TestTransactionServiceCreateForUserRejectsForeignAccounts(t *testing.T) {
	foreignRelatedAccountID := "foreign-related-account"
	tests := []struct {
		name             string
		accountID        string
		relatedAccountID *string
	}{
		{
			name:      "foreign account",
			accountID: "foreign-account",
		},
		{
			name:             "foreign related account",
			accountID:        "owned-account",
			relatedAccountID: &foreignRelatedAccountID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &userScopedTransactionRepo{
				userID:          "user-1",
				accountID:       "owned-account",
				relatedAccounts: map[string]bool{"owned-related-account": true},
			}
			_, err := NewTransactionService(repo).CreateForUser(t.Context(), "user-1", &CreateTransactionRequest{
				AccountID:        tt.accountID,
				RelatedAccountID: tt.relatedAccountID,
				Type:             models.TransactionTypeIncome,
				Amount:           dec("100"),
			})
			if !errors.Is(err, repository.ErrNotFound) {
				t.Fatalf("error = %v, want ErrNotFound", err)
			}
			if repo.createCalls != 0 {
				t.Fatalf("old Create calls = %d, want 0", repo.createCalls)
			}
		})
	}
}

func TestTransactionServiceCreateValidatesInput(t *testing.T) {
	_, err := NewTransactionService().Create(t.Context(), &CreateTransactionRequest{
		AccountID: "account-1",
		Type:      models.TransactionTypeIncome,
		Amount:    dec("0"),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTransactionServiceCreateRejectsNegativeNonAdjustmentAmounts(t *testing.T) {
	tests := []models.TransactionType{
		models.TransactionTypeInitialBalance,
		models.TransactionTypeIncome,
		models.TransactionTypeExpense,
		models.TransactionTypeTransferIn,
		models.TransactionTypeTransferOut,
		models.TransactionTypeInterestIncome,
	}

	for _, transactionType := range tests {
		t.Run(string(transactionType), func(t *testing.T) {
			_, err := NewTransactionService().Create(t.Context(), &CreateTransactionRequest{
				AccountID: "account-1",
				Type:      transactionType,
				Amount:    dec("-0.01"),
			})
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestTransactionServiceCreateRejectsDirectTransferTypes(t *testing.T) {
	for _, transactionType := range []models.TransactionType{models.TransactionTypeTransferIn, models.TransactionTypeTransferOut} {
		t.Run(string(transactionType), func(t *testing.T) {
			tests := []struct {
				name string
				run  func() error
			}{
				{
					name: "create",
					run: func() error {
						_, err := NewTransactionService(&recordingCreateForUserRepo{}).Create(t.Context(), &CreateTransactionRequest{
							AccountID: "account-1",
							Type:      transactionType,
							Amount:    dec("1"),
						})
						return err
					},
				},
				{
					name: "create for user",
					run: func() error {
						_, err := NewTransactionService(&recordingCreateForUserRepo{}).CreateForUser(t.Context(), "user-1", &CreateTransactionRequest{
							AccountID: "account-1",
							Type:      transactionType,
							Amount:    dec("1"),
						})
						return err
					},
				},
				{
					name: "create many",
					run: func() error {
						_, err := NewTransactionService(&recordingCreateForUserRepo{}).CreateMany(t.Context(), &CreateTransactionRequest{
							AccountID: "account-1",
							Type:      transactionType,
							Amount:    dec("1"),
						})
						return err
					},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					err := tt.run()
					if err == nil {
						t.Fatal("expected error")
					}
					if !IsValidationError(err) {
						t.Fatalf("expected validation error, got %T: %v", err, err)
					}
				})
			}
		})
	}
}

func TestTransactionServiceCreateTransferAllowsTransferTypes(t *testing.T) {
	repo := &recordingCreateForUserRepo{}
	transfer := &models.Transfer{ID: "transfer-1"}

	transactions, err := NewTransactionService(repo).CreateTransfer(
		t.Context(), transfer,
		&CreateTransactionRequest{AccountID: "account-1", Type: models.TransactionTypeTransferOut, Amount: dec("1")},
		&CreateTransactionRequest{AccountID: "account-2", Type: models.TransactionTypeTransferIn, Amount: dec("1")},
	)
	if err != nil {
		t.Fatalf("create transfer transactions: %v", err)
	}
	if len(transactions) != 2 {
		t.Fatalf("transactions len = %d, want 2", len(transactions))
	}
	if transfer.FromTransactionID == "" || transfer.ToTransactionID == "" {
		t.Fatalf("transfer transaction ids were not set: %+v", transfer)
	}
	for _, transaction := range transactions {
		if transaction.SourceType != models.TransactionSourceTransfer {
			t.Fatalf("source type = %q, want transfer", transaction.SourceType)
		}
		if transaction.SourceRefID == nil || *transaction.SourceRefID != transfer.ID {
			t.Fatalf("source ref = %v, want %q", transaction.SourceRefID, transfer.ID)
		}
	}
}

func TestTransactionServiceCreateAllowsNegativeAdjustments(t *testing.T) {
	tx, err := NewTransactionService(&recordingCreateForUserRepo{}).Create(t.Context(), &CreateTransactionRequest{
		AccountID: "account-1",
		Type:      models.TransactionTypeAdjustment,
		Amount:    dec("-10"),
	})
	if err != nil {
		t.Fatalf("create adjustment transaction: %v", err)
	}
	if !tx.Amount.Equal(dec("-10")) {
		t.Fatalf("amount = %d, want -1000", tx.Amount)
	}
}

func TestTransactionServiceCreateValidatesAmountBounds(t *testing.T) {
	tests := []struct {
		name        string
		transaction models.TransactionType
		amount      decimal.Decimal
		wantErr     bool
	}{
		{name: "allows positive boundary", transaction: models.TransactionTypeIncome, amount: maxTransactionAmount},
		{name: "rejects positive above boundary", transaction: models.TransactionTypeIncome, amount: maxTransactionAmount.Add(decimal.NewFromInt(1)), wantErr: true},
		{name: "allows negative adjustment boundary", transaction: models.TransactionTypeAdjustment, amount: maxTransactionAmount.Neg()},
		{name: "rejects negative adjustment below boundary", transaction: models.TransactionTypeAdjustment, amount: maxTransactionAmount.Neg().Sub(decimal.NewFromInt(1)), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTransactionService(&recordingCreateForUserRepo{}).Create(t.Context(), &CreateTransactionRequest{
				AccountID: "account-1",
				Type:      tt.transaction,
				Amount:    tt.amount,
			})
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if !IsValidationError(err) {
					t.Fatalf("expected validation error, got %T: %v", err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("create transaction: %v", err)
			}
		})
	}
}

func TestTransactionServiceCreateValidatesCurrencyScale(t *testing.T) {
	tests := []struct {
		name     string
		amount   decimal.Decimal
		currency string
		wantErr  bool
	}{
		{name: "rejects rub sub-kopeck", amount: dec("1.234"), currency: "RUB", wantErr: true},
		{name: "rejects jpy fractional unit", amount: dec("0.5"), currency: "JPY", wantErr: true},
		{name: "allows kwd three decimals", amount: dec("1.234"), currency: "KWD"},
		{name: "empty currency keeps rub scale", amount: dec("1.23")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTransactionService(&recordingCreateForUserRepo{}).Create(t.Context(), &CreateTransactionRequest{
				AccountID: "account-1",
				Type:      models.TransactionTypeIncome,
				Amount:    tt.amount,
				Currency:  tt.currency,
			})
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if !IsValidationError(err) {
					t.Fatalf("expected validation error, got %T: %v", err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("create transaction: %v", err)
			}
		})
	}
}

func TestTransactionServiceCreateRejectsFutureDate(t *testing.T) {
	_, err := NewTransactionService(&recordingCreateForUserRepo{}).Create(t.Context(), &CreateTransactionRequest{
		AccountID:  "account-1",
		Type:       models.TransactionTypeIncome,
		Amount:     dec("1"),
		OccurredAt: time.Now().Add(time.Hour),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
}

func TestTransactionServiceCreateRejectsBeforeAccountOpen(t *testing.T) {
	openedAt := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	_, err := NewTransactionService(&recordingCreateForUserRepo{}).Create(t.Context(), &CreateTransactionRequest{
		AccountID:       "account-1",
		Type:            models.TransactionTypeIncome,
		Amount:          dec("1"),
		OccurredAt:      openedAt.AddDate(0, 0, -1),
		AccountOpenedAt: openedAt,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
}

func TestTransactionServiceCreateRejectsMissingRepository(t *testing.T) {
	_, err := NewTransactionService().Create(t.Context(), &CreateTransactionRequest{
		AccountID: "account-1",
		Type:      models.TransactionTypeIncome,
		Amount:    dec("1"),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if IsValidationError(err) {
		t.Fatalf("expected wiring error, got validation error: %v", err)
	}
}

func TestTransactionServiceCreateReturnsValidationError(t *testing.T) {
	tests := []struct {
		name string
		req  *CreateTransactionRequest
	}{
		{
			name: "missing account id",
			req: &CreateTransactionRequest{
				Type:   models.TransactionTypeIncome,
				Amount: dec("1"),
			},
		},
		{
			name: "invalid transaction type",
			req: &CreateTransactionRequest{
				AccountID: "account-1",
				Type:      models.TransactionType("unknown"),
				Amount:    dec("1"),
			},
		},
		{
			name: "zero amount",
			req: &CreateTransactionRequest{
				AccountID: "account-1",
				Type:      models.TransactionTypeIncome,
				Amount:    dec("0"),
			},
		},
		{
			name: "negative income amount",
			req: &CreateTransactionRequest{
				AccountID: "account-1",
				Type:      models.TransactionTypeIncome,
				Amount:    dec("-1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewTransactionService()

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

type failingTransactionRepo struct {
	err error
}

func (r failingTransactionRepo) Create(_ context.Context, _ *models.Transaction) error {
	return r.err
}

func (r failingTransactionRepo) CreateForUser(_ context.Context, _ string, _ *models.Transaction) error {
	return r.err
}

func (r failingTransactionRepo) CreateMany(_ context.Context, _ []models.Transaction) error {
	return r.err
}

func (r failingTransactionRepo) CreateTransfer(_ context.Context, _ *models.Transfer, _ []models.Transaction) error {
	return r.err
}

func (r failingTransactionRepo) ListTransfersByUser(context.Context, string) ([]models.Transfer, error) {
	return nil, r.err
}

func (r failingTransactionRepo) GetByID(_ context.Context, _ string) (*models.Transaction, error) {
	return nil, r.err
}

func (r failingTransactionRepo) GetByIDForUser(_ context.Context, _, _ string) (*models.Transaction, error) {
	return nil, r.err
}

func (r failingTransactionRepo) List(_ context.Context) ([]models.Transaction, error) {
	return nil, r.err
}

func (r failingTransactionRepo) ListByUser(_ context.Context, _ string) ([]models.Transaction, error) {
	return nil, r.err
}

func (r failingTransactionRepo) ListByAccount(_ context.Context, _ string) ([]models.Transaction, error) {
	return nil, r.err
}

func (r failingTransactionRepo) ListByAccountForUser(_ context.Context, _, _ string) ([]models.Transaction, error) {
	return nil, r.err
}

func (r failingTransactionRepo) GetBalanceByAccountForUser(context.Context, string, string) (balance decimal.Decimal, transactionCount int64, err error) {
	return decimal.Zero, 0, r.err
}

type userScopedTransactionRepo struct {
	failingTransactionRepo
	createCalls     int
	userID          string
	accountID       string
	relatedAccounts map[string]bool
}

func (r *userScopedTransactionRepo) Create(_ context.Context, _ *models.Transaction) error {
	r.createCalls++
	return errors.New("unexpected unscoped create")
}

func (r *userScopedTransactionRepo) CreateForUser(_ context.Context, userID string, transaction *models.Transaction) error {
	if userID != r.userID || transaction.AccountID != r.accountID {
		return repository.ErrNotFound
	}
	if transaction.RelatedAccountID != nil && !r.relatedAccounts[*transaction.RelatedAccountID] {
		return repository.ErrNotFound
	}
	return nil
}

func TestTransactionServiceCreateDoesNotClassifyRepositoryErrorAsValidation(t *testing.T) {
	repoErr := errors.New("database failed")
	service := NewTransactionService(failingTransactionRepo{err: repoErr})

	_, err := service.Create(context.Background(), &CreateTransactionRequest{
		AccountID: "account-1",
		Type:      models.TransactionTypeIncome,
		Amount:    dec("1"),
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if IsValidationError(err) {
		t.Fatalf("expected repository/internal error, got validation error: %v", err)
	}
}

func TestTransactionServiceCreateForUserDoesNotClassifyRepositoryErrorAsValidation(t *testing.T) {
	repoErr := errors.New("database failed")
	service := NewTransactionService(failingTransactionRepo{err: repoErr})

	_, err := service.CreateForUser(context.Background(), "user-1", &CreateTransactionRequest{
		AccountID: "account-1",
		Type:      models.TransactionTypeIncome,
		Amount:    dec("1"),
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if IsValidationError(err) {
		t.Fatalf("expected repository/internal error, got validation error: %v", err)
	}
}

type recordingCreateForUserRepo struct {
	failingTransactionRepo
	createCalls        int
	createForUserCalls int
	userID             string
	transaction        *models.Transaction
}

func (r *recordingCreateForUserRepo) Create(context.Context, *models.Transaction) error {
	r.createCalls++
	return nil
}

func (r *recordingCreateForUserRepo) CreateForUser(_ context.Context, userID string, transaction *models.Transaction) error {
	r.createForUserCalls++
	r.userID = userID
	r.transaction = transaction
	return nil
}

func (r *recordingCreateForUserRepo) CreateTransfer(_ context.Context, _ *models.Transfer, transactions []models.Transaction) error {
	r.createForUserCalls++
	if len(transactions) > 0 {
		r.transaction = &transactions[0]
	}
	return nil
}
