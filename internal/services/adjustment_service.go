package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
)

type AdjustmentService struct {
	transactions *TransactionService
}

func NewAdjustmentService(transactions *TransactionService) *AdjustmentService {
	return &AdjustmentService{transactions: transactions}
}

type CreateAdjustmentRequest struct {
	AccountID   string
	Amount      decimal.Decimal
	Currency    string
	Description string
	OccurredAt  time.Time
}

func (s *AdjustmentService) Create(ctx context.Context, req *CreateAdjustmentRequest) (*models.Transaction, error) {
	if req == nil {
		return nil, validationError("adjustment request is required")
	}

	if s == nil || s.transactions == nil {
		return nil, fmt.Errorf("adjustment service requires transaction service")
	}
	accountID := strings.TrimSpace(req.AccountID)
	if accountID == "" {
		return nil, fmt.Errorf("account id is required")
	}
	if req.Amount.IsZero() {
		return nil, fmt.Errorf("adjustment amount must be non-zero")
	}

	tx, err := s.transactions.Create(ctx, &CreateTransactionRequest{
		AccountID:   accountID,
		Type:        models.TransactionTypeAdjustment,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
		OccurredAt:  req.OccurredAt,
	})
	if err != nil {
		return nil, fmt.Errorf("create adjustment transaction: %w", err)
	}

	return tx, nil
}
