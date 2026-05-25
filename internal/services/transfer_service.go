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
	"github.com/sunriseex/capitalflow/pkg/money"
)

type TransferService struct {
	transactions *TransactionService
	currency     *CurrencyService
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
	Description    string
	IdempotencyKey string
}

type CreateTransferResponse struct {
	Out          *models.Transaction
	In           *models.Transaction
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
	fromCurrency := strings.TrimSpace(req.FromCurrency)
	toCurrency := strings.TrimSpace(req.ToCurrency)
	if err := domaintransfer.ValidateCreate(&domaintransfer.CreateValidation{
		FromAccountID:  fromAccountID,
		ToAccountID:    toAccountID,
		FromCurrency:   fromCurrency,
		Amount:         req.Amount,
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
		ExchangeRateProvider: "internal",
		ExchangeRateDate:     now,
		IdempotencyKey:       idempotencyKey,
		CreatedAt:            now,
	}

	inRelatedID := fromAccountID
	outRelatedID := toAccountID
	created, err := s.transactions.CreateTransfer(ctx, transfer, &CreateTransactionRequest{
		AccountID:        fromAccountID,
		RelatedAccountID: &outRelatedID,
		Type:             models.TransactionTypeTransferOut,
		Amount:           fromAmount,
		Currency:         fromCurrency,
		Description:      req.Description,
	}, &CreateTransactionRequest{
		AccountID:        toAccountID,
		RelatedAccountID: &inRelatedID,
		Type:             models.TransactionTypeTransferIn,
		Amount:           inAmount,
		Currency:         toCurrency,
		Description:      req.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("create transfer transactions: %w", err)
	}

	return &CreateTransferResponse{Out: &created[0], In: &created[1], ExchangeRate: exchangeRate}, nil
}
