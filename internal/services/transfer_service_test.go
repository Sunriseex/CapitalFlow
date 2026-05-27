package services

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
)

func TestTransferServiceCreate(t *testing.T) {
	got, err := NewTransferService(NewTransactionService(&batchTransactionRepo{})).Create(t.Context(), &CreateTransferRequest{
		FromAccountID:  "account-1",
		ToAccountID:    "account-2",
		Amount:         dec("250"),
		Description:    "Move savings",
		IdempotencyKey: "transfer-create",
	})
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	if got.Out.Type != models.TransactionTypeTransferOut {
		t.Fatalf("out type = %s", got.Out.Type)
	}
	if got.In.Type != models.TransactionTypeTransferIn {
		t.Fatalf("in type = %s", got.In.Type)
	}
	if got.Out.Amount != got.In.Amount {
		t.Fatalf("amount mismatch: out=%d in=%d", got.Out.Amount, got.In.Amount)
	}
	if got.Out.RelatedAccountID == nil || *got.Out.RelatedAccountID != "account-2" {
		t.Fatalf("out related account = %v", got.Out.RelatedAccountID)
	}
	if got.In.RelatedAccountID == nil || *got.In.RelatedAccountID != "account-1" {
		t.Fatalf("in related account = %v", got.In.RelatedAccountID)
	}
}

func TestTransferServiceCreateKeepsSameCurrencyAmounts(t *testing.T) {
	got, err := NewTransferService(NewTransactionService(&batchTransactionRepo{})).Create(t.Context(), &CreateTransferRequest{
		FromAccountID:  "account-1",
		ToAccountID:    "account-2",
		FromCurrency:   "RUB",
		ToCurrency:     "RUB",
		Amount:         dec("1.23"),
		IdempotencyKey: "same-currency",
	})
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	if !got.Out.Amount.Equal(dec("1.23")) || !got.In.Amount.Equal(dec("1.23")) {
		t.Fatalf("amounts = %s/%s, want 1.23/1.23", got.Out.Amount, got.In.Amount)
	}
}

func TestTransferServiceCreateRejectsSourceOverPrecision(t *testing.T) {
	tests := []struct {
		name     string
		amount   string
		currency string
	}{
		{name: "RUB", amount: "1.234", currency: "RUB"},
		{name: "JPY rounds up", amount: "0.6", currency: "JPY"},
		{name: "JPY rounds down", amount: "0.4", currency: "JPY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTransferService(NewTransactionService(&batchTransactionRepo{})).Create(t.Context(), &CreateTransferRequest{
				FromAccountID:  "account-1",
				ToAccountID:    "account-2",
				FromCurrency:   tt.currency,
				ToCurrency:     tt.currency,
				Amount:         dec(tt.amount),
				IdempotencyKey: "source-over-precision-" + tt.name,
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if !IsValidationError(err) {
				t.Fatalf("expected validation error, got %T: %v", err, err)
			}
		})
	}
}

func TestTransferServiceCreateRejectsSameAccount(t *testing.T) {
	_, err := NewTransferService(NewTransactionService(&batchTransactionRepo{})).Create(t.Context(), &CreateTransferRequest{
		FromAccountID:  "account-1",
		ToAccountID:    "account-1",
		Amount:         dec("250"),
		IdempotencyKey: "same-account",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTransferServiceCreateRejectsMissingTransactionService(t *testing.T) {
	_, err := NewTransferService(nil).Create(t.Context(), &CreateTransferRequest{
		FromAccountID:  "account-1",
		ToAccountID:    "account-2",
		Amount:         dec("250"),
		IdempotencyKey: "missing-transaction-service",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if IsValidationError(err) {
		t.Fatalf("expected wiring error, got validation error: %v", err)
	}
}

func TestTransferServiceCreatePersistsTransactionsAsBatch(t *testing.T) {
	repo := &batchTransactionRepo{}
	got, err := NewTransferService(NewTransactionService(repo)).Create(t.Context(), &CreateTransferRequest{
		FromAccountID:  "account-1",
		ToAccountID:    "account-2",
		Amount:         dec("250"),
		Description:    "Move savings",
		IdempotencyKey: "persisted-batch",
	})
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	if got.Out == nil || got.In == nil {
		t.Fatal("transfer transactions must be returned")
	}
	if repo.createCalls != 0 {
		t.Fatalf("single create calls = %d, want 0", repo.createCalls)
	}
	if len(repo.batches) != 1 {
		t.Fatalf("batch count = %d, want 1", len(repo.batches))
	}
	if len(repo.batches[0]) != 2 {
		t.Fatalf("batch size = %d, want 2", len(repo.batches[0]))
	}
}

func TestTransferServiceCreateConvertsCrossCurrencyAmount(t *testing.T) {
	repo := &batchTransactionRepo{}
	service := NewTransferService(NewTransactionService(repo))
	service.currency = NewCurrencyService(staticExchangeRateProvider{
		rates: &ExchangeRates{
			Base: "RUB",
			Rates: map[string]decimal.Decimal{
				"KRW": decimal.RequireFromString("16.25"),
			},
		},
	})

	got, err := service.Create(t.Context(), &CreateTransferRequest{
		FromAccountID:  "rub-account",
		ToAccountID:    "krw-account",
		FromCurrency:   "RUB",
		ToCurrency:     "KRW",
		Amount:         dec("10000"),
		IdempotencyKey: "cross-currency",
	})
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	if !got.Out.Amount.Equal(dec("10000")) {
		t.Fatalf("out amount = %d, want 1000000", got.Out.Amount)
	}
	if !got.In.Amount.Equal(dec("162500")) {
		t.Fatalf("in amount = %d, want 16250000", got.In.Amount)
	}
	if got.ExchangeRate != "16.25" {
		t.Fatalf("exchange rate = %s, want 16.25", got.ExchangeRate)
	}
	if repo.fromCurrency != "RUB" || repo.toCurrency != "KRW" {
		t.Fatalf("repo currencies = %s/%s, want RUB/KRW", repo.fromCurrency, repo.toCurrency)
	}
	if repo.transfer == nil {
		t.Fatal("transfer audit record was not persisted")
	}
	if repo.transfer.ExchangeRate != "16.25" ||
		!repo.transfer.FromAmount.Equal(dec("10000")) ||
		!repo.transfer.ToAmount.Equal(dec("162500")) {
		t.Fatalf("transfer audit = rate %s amounts %d/%d, want 16.25 1000000/16250000", repo.transfer.ExchangeRate, repo.transfer.FromAmount, repo.transfer.ToAmount)
	}
	if repo.transfer.IdempotencyKey != "cross-currency" {
		t.Fatalf("transfer idempotency key = %q, want cross-currency", repo.transfer.IdempotencyKey)
	}
	if repo.transfer.FromTransactionID == "" || repo.transfer.ToTransactionID == "" {
		t.Fatalf("transfer audit transaction ids must be set: %+v", repo.transfer)
	}
}

func TestTransferServiceCreateConvertsRubToUSDT(t *testing.T) {
	repo := &batchTransactionRepo{}
	service := NewTransferService(NewTransactionService(repo))
	service.currency = NewCurrencyService(staticExchangeRateProvider{
		rates: &ExchangeRates{
			Base: "RUB",
			Rates: map[string]decimal.Decimal{
				"USDT": decimal.RequireFromString("0.01"),
			},
		},
	})

	got, err := service.Create(t.Context(), &CreateTransferRequest{
		FromAccountID:  "rub-account",
		ToAccountID:    "usdt-account",
		FromCurrency:   "RUB",
		ToCurrency:     "USDT",
		Amount:         dec("125.55"),
		IdempotencyKey: "rub-usdt",
	})
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	if !got.In.Amount.Equal(dec("1.255500")) {
		t.Fatalf("in amount = %s, want 1.255500", got.In.Amount)
	}
	if repo.transfer == nil || repo.transfer.ToCurrency != "USDT" || !repo.transfer.ToAmount.Equal(dec("1.255500")) {
		t.Fatalf("transfer audit = %+v", repo.transfer)
	}
}

func TestTransferServiceCreateRoundsDestinationAfterCrossCurrencyConversion(t *testing.T) {
	repo := &batchTransactionRepo{}
	service := NewTransferService(NewTransactionService(repo))
	service.currency = NewCurrencyService(staticExchangeRateProvider{
		rates: &ExchangeRates{
			Base: "RUB",
			Rates: map[string]decimal.Decimal{
				"KWD": decimal.RequireFromString("0.003333"),
			},
		},
	})

	got, err := service.Create(t.Context(), &CreateTransferRequest{
		FromAccountID:  "rub-account",
		ToAccountID:    "kwd-account",
		FromCurrency:   "RUB",
		ToCurrency:     "KWD",
		Amount:         dec("1.23"),
		IdempotencyKey: "cross-currency-rounding",
	})
	if err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	if !got.Out.Amount.Equal(dec("1.23")) {
		t.Fatalf("out amount = %s, want 1.23", got.Out.Amount)
	}
	if !got.In.Amount.Equal(dec("0.004")) {
		t.Fatalf("in amount = %s, want 0.004", got.In.Amount)
	}
	if repo.transfer == nil || !repo.transfer.FromAmount.Equal(dec("1.23")) || !repo.transfer.ToAmount.Equal(dec("0.004")) {
		t.Fatalf("transfer audit amounts = %v", repo.transfer)
	}
}

func TestTransferServiceCreatePersistsFeeTransaction(t *testing.T) {
	repo := &batchTransactionRepo{}
	got, err := NewTransferService(NewTransactionService(repo)).Create(t.Context(), &CreateTransferRequest{
		FromAccountID:  "account-1",
		ToAccountID:    "account-2",
		FromCurrency:   "RUB",
		ToCurrency:     "RUB",
		Amount:         dec("100"),
		FeeAmount:      dec("1.25"),
		IdempotencyKey: "transfer-with-fee",
		Description:    "Broker top up",
	})
	if err != nil {
		t.Fatalf("create transfer with fee: %v", err)
	}
	if got.Fee == nil {
		t.Fatal("fee transaction is nil")
	}
	if got.Fee.Type != models.TransactionTypeExpense || got.Fee.AccountID != "account-1" || !got.Fee.Amount.Equal(dec("1.25")) {
		t.Fatalf("fee transaction = %+v", got.Fee)
	}
	if got.Fee.TransferID != nil {
		t.Fatalf("fee transaction transfer id = %v, want nil", got.Fee.TransferID)
	}
	if repo.transfer == nil || repo.transfer.FeeTransactionID == nil || *repo.transfer.FeeTransactionID != got.Fee.ID {
		t.Fatalf("transfer fee transaction id = %v, fee id = %s", repo.transfer.FeeTransactionID, got.Fee.ID)
	}
	if !repo.transfer.FeeAmount.Equal(dec("1.25")) || repo.transfer.FeeCurrency == nil || *repo.transfer.FeeCurrency != "RUB" {
		t.Fatalf("transfer fee audit = %+v", repo.transfer)
	}
	if len(repo.batches) != 1 || len(repo.batches[0]) != 3 {
		t.Fatalf("batch sizes = %+v, want one batch of 3", repo.batches)
	}
}

func TestTransferServiceCreateRejectsMismatchedFeeCurrency(t *testing.T) {
	repo := &batchTransactionRepo{}
	_, err := NewTransferService(NewTransactionService(repo)).Create(t.Context(), &CreateTransferRequest{
		FromAccountID:  "account-1",
		ToAccountID:    "account-2",
		FromCurrency:   "RUB",
		ToCurrency:     "RUB",
		Amount:         dec("100"),
		FeeAmount:      dec("1"),
		FeeCurrency:    "USDT",
		IdempotencyKey: "mismatched-fee-currency",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
	if len(repo.batches) != 0 {
		t.Fatalf("created batches = %d, want 0", len(repo.batches))
	}
}

func TestTransferServiceCreateNormalizesFeeCurrency(t *testing.T) {
	repo := &batchTransactionRepo{}
	got, err := NewTransferService(NewTransactionService(repo)).Create(t.Context(), &CreateTransferRequest{
		FromAccountID:  "account-1",
		ToAccountID:    "account-2",
		FromCurrency:   "RUB",
		ToCurrency:     "RUB",
		Amount:         dec("100"),
		FeeAmount:      dec("1"),
		FeeCurrency:    "rub",
		IdempotencyKey: "normalized-fee-currency",
	})
	if err != nil {
		t.Fatalf("create transfer with lowercase fee currency: %v", err)
	}
	if got.Fee == nil {
		t.Fatal("fee transaction is nil")
	}
	if repo.transfer == nil || repo.transfer.FeeCurrency == nil || *repo.transfer.FeeCurrency != "RUB" {
		t.Fatalf("transfer fee currency = %v, want RUB", repo.transfer)
	}
}

type batchTransactionRepo struct {
	createCalls  int
	batches      [][]models.Transaction
	transfer     *models.Transfer
	fromCurrency string
	toCurrency   string
}

func (r *batchTransactionRepo) Create(context.Context, *models.Transaction) error {
	r.createCalls++
	return nil
}

func (r *batchTransactionRepo) CreateForUser(context.Context, string, *models.Transaction) error {
	return errors.New("unexpected user-scoped create")
}

func (r *batchTransactionRepo) CreateMany(_ context.Context, transactions []models.Transaction) error {
	r.batches = append(r.batches, append([]models.Transaction(nil), transactions...))
	return nil
}

func (r *batchTransactionRepo) CreateTransfer(ctx context.Context, transfer *models.Transfer, transactions []models.Transaction) error {
	r.transfer = transfer
	r.fromCurrency = transfer.FromCurrency
	r.toCurrency = transfer.ToCurrency
	return r.CreateMany(ctx, transactions)
}

func (r *batchTransactionRepo) ListTransfersByUser(context.Context, string) ([]models.Transfer, error) {
	return nil, nil
}

func (r *batchTransactionRepo) GetByID(context.Context, string) (*models.Transaction, error) {
	return nil, errNotImplemented
}

func (r *batchTransactionRepo) GetByIDForUser(context.Context, string, string) (*models.Transaction, error) {
	return nil, errNotImplemented
}

func (r *batchTransactionRepo) List(context.Context) ([]models.Transaction, error) {
	return nil, nil
}

func (r *batchTransactionRepo) ListByUser(context.Context, string) ([]models.Transaction, error) {
	return nil, nil
}

func (r *batchTransactionRepo) ListByAccount(context.Context, string) ([]models.Transaction, error) {
	return nil, nil
}

func (r *batchTransactionRepo) ListByAccountForUser(context.Context, string, string) ([]models.Transaction, error) {
	return nil, nil
}

func (r *batchTransactionRepo) GetBalanceByAccountForUser(context.Context, string, string) (balance decimal.Decimal, transactionCount int64, err error) {
	return decimal.Zero, 0, nil
}

var errNotImplemented = errors.New("not implemented")

func TestTransferServiceCreateReturnsValidationError(t *testing.T) {
	tests := []struct {
		name string
		req  CreateTransferRequest
	}{
		{
			name: "missing from account id",
			req: CreateTransferRequest{
				ToAccountID:    "account-2",
				Amount:         dec("1"),
				IdempotencyKey: "missing-from",
			},
		},
		{
			name: "missing to account id",
			req: CreateTransferRequest{
				FromAccountID:  "account-1",
				Amount:         dec("1"),
				IdempotencyKey: "missing-to",
			},
		},
		{
			name: "same accounts",
			req: CreateTransferRequest{
				FromAccountID:  "account-1",
				ToAccountID:    "account-1",
				Amount:         dec("1"),
				IdempotencyKey: "same-account",
			},
		},
		{
			name: "zero amount",
			req: CreateTransferRequest{
				FromAccountID:  "account-1",
				ToAccountID:    "account-2",
				Amount:         dec("0"),
				IdempotencyKey: "zero",
			},
		},
		{
			name: "negative amount",
			req: CreateTransferRequest{
				FromAccountID:  "account-1",
				ToAccountID:    "account-2",
				Amount:         dec("-1"),
				IdempotencyKey: "negative",
			},
		},
		{
			name: "missing idempotency key",
			req: CreateTransferRequest{
				FromAccountID: "account-1",
				ToAccountID:   "account-2",
				Amount:        dec("1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewTransferService(NewTransactionService(&batchTransactionRepo{}))

			_, err := service.Create(context.Background(), &tt.req)
			if err == nil {
				t.Fatal("expected error")
			}

			if !IsValidationError(err) {
				t.Fatalf("expected validation error, got %T: %v", err, err)
			}
		})
	}
}

func TestTransferServiceCreateDoesNotClassifyRepositoryErrorAsValidation(t *testing.T) {
	repoErr := errors.New("database failed")
	txService := NewTransactionService(failingTransactionRepo{err: repoErr})
	service := NewTransferService(txService)

	_, err := service.Create(context.Background(), &CreateTransferRequest{
		FromAccountID:  "account-1",
		ToAccountID:    "account-2",
		Amount:         dec("1"),
		IdempotencyKey: "repo-error",
	})

	if err == nil {
		t.Fatal("expected error")
	}

	if IsValidationError(err) {
		t.Fatalf("expected repository/internal error, got validation error: %v", err)
	}
}
