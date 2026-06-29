package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	domaintransfer "github.com/sunriseex/capitalflow/internal/domain/transfer"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/pkg/money"
)

type TransferService struct {
	transactions *TransactionService
	currency     *CurrencyService
	accounts     repository.AccountRepository
}

func (s *TransferService) WithAccountRepository(repo repository.AccountRepository) *TransferService {
	s.accounts = repo
	return s
}

func (s *TransferService) ListByUser(ctx context.Context, userID string) ([]models.Transfer, error) {
	if s == nil || s.transactions == nil {
		return nil, fmt.Errorf("transfer service requires transaction service")
	}
	return s.transactions.ListTransfersByUser(ctx, userID)
}

func NewTransferService(transactions *TransactionService) *TransferService {
	return &TransferService{transactions: transactions, currency: NewCurrencyService(nil)}
}

type CreateTransferRequest struct {
	UserID         string
	FromAccountID  string
	ToAccountID    string
	FromCurrency   string
	ToCurrency     string
	Amount         decimal.Decimal
	FeeAmount      decimal.Decimal
	FeeCurrency    string
	Description    string
	IdempotencyKey string
}

type CreateTransferResponse struct {
	Out          *models.Transaction
	In           *models.Transaction
	Fee          *models.Transaction
	ExchangeRate string
}

func (s *TransferService) Create(ctx context.Context, req *CreateTransferRequest) (*CreateTransferResponse, error) {
	if s == nil || s.transactions == nil {
		return nil, fmt.Errorf("transfer service requires transaction service")
	}
	if req == nil {
		return nil, validationError("transfer request is required")
	}

	fromAccountID := strings.TrimSpace(req.FromAccountID)
	toAccountID := strings.TrimSpace(req.ToAccountID)
	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	fromCurrency := strings.ToUpper(strings.TrimSpace(req.FromCurrency))
	toCurrency := strings.ToUpper(strings.TrimSpace(req.ToCurrency))
	if s.accounts != nil {
		userID := strings.TrimSpace(req.UserID)
		fromAccount, err := s.accounts.GetByIDForUser(ctx, fromAccountID, userID)
		if err != nil {
			return nil, fmt.Errorf("get source account: %w", err)
		}
		toAccount, err := s.accounts.GetByIDForUser(ctx, toAccountID, userID)
		if err != nil {
			return nil, fmt.Errorf("get destination account: %w", err)
		}
		fromCurrency = fromAccount.Currency
		toCurrency = toAccount.Currency
	}
	feeCurrency := strings.ToUpper(strings.TrimSpace(req.FeeCurrency))
	if feeCurrency == "" && req.FeeAmount.IsPositive() {
		feeCurrency = fromCurrency
	}
	if req.FeeAmount.IsPositive() && feeCurrency != fromCurrency {
		return nil, validationError("transfer fee currency must match source account currency")
	}
	if err := domaintransfer.ValidateCreate(&domaintransfer.CreateValidation{
		FromAccountID:  fromAccountID,
		ToAccountID:    toAccountID,
		FromCurrency:   fromCurrency,
		Amount:         req.Amount,
		FeeAmount:      req.FeeAmount,
		FeeCurrency:    feeCurrency,
		IdempotencyKey: idempotencyKey,
	}); err != nil {
		return nil, validationError(err.Error())
	}
	fromAmount := money.RoundForCurrency(req.Amount, fromCurrency)
	inAmount := fromAmount
	exchangeRate := "1"
	if fromCurrency != "" || toCurrency != "" {
		convertedAmount, rate, err := s.currency.ConvertDecimalAmount(ctx, fromAmount, fromCurrency, toCurrency)
		if err != nil {
			return nil, fmt.Errorf("convert transfer amount: %w", err)
		}
		inAmount = money.RoundForCurrency(convertedAmount, toCurrency)
		exchangeRate = rate.String()
	}

	now := time.Now().UTC()
	transfer := &models.Transfer{
		ID:                   uuid.NewString(),
		UserID:               strings.TrimSpace(req.UserID),
		FromAccountID:        fromAccountID,
		ToAccountID:          toAccountID,
		FromAmount:           fromAmount,
		ToAmount:             inAmount,
		FromCurrency:         fromCurrency,
		ToCurrency:           toCurrency,
		ExchangeRate:         exchangeRate,
		ExchangeRateScale:    18,
		ExchangeRateProvider: "internal",
		ExchangeRateDate:     now,
		FeeAmount:            req.FeeAmount,
		Status:               "completed",
		IdempotencyKey:       idempotencyKey,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if req.FeeAmount.IsPositive() {
		transfer.FeeCurrency = &feeCurrency
	}

	inRelatedID := fromAccountID
	outRelatedID := toAccountID
	createReqs := []*CreateTransactionRequest{{
		AccountID:        fromAccountID,
		RelatedAccountID: &outRelatedID,
		Type:             models.TransactionTypeTransferOut,
		Amount:           fromAmount,
		Currency:         fromCurrency,
		Description:      req.Description,
	}, {
		AccountID:        toAccountID,
		RelatedAccountID: &inRelatedID,
		Type:             models.TransactionTypeTransferIn,
		Amount:           inAmount,
		Currency:         toCurrency,
		Description:      req.Description,
	}}
	if req.FeeAmount.IsPositive() {
		createReqs = append(createReqs, &CreateTransactionRequest{
			AccountID:   fromAccountID,
			Type:        models.TransactionTypeExpense,
			Amount:      req.FeeAmount,
			Currency:    feeCurrency,
			Description: transferFeeDescription(req.Description),
		})
	}

	created, err := s.transactions.CreateTransfer(ctx, transfer, createReqs...)
	if err != nil {
		return nil, fmt.Errorf("create transfer transactions: %w", err)
	}

	response := &CreateTransferResponse{Out: &created[0], In: &created[1], ExchangeRate: exchangeRate}
	if len(created) == 3 {
		response.Fee = &created[2]
	}
	return response, nil
}

func transferFeeDescription(description string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return "Transfer fee"
	}
	return "Transfer fee: " + description
}
