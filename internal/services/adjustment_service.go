package services

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	AmountMinor int64
	Description string
	OccurredAt  time.Time
}

func (s *AdjustmentService) Create(ctx context.Context, req CreateAdjustmentRequest) (*models.Transaction, error) {
	if s == nil || s.transactions == nil {
		return nil, fmt.Errorf("adjustment service requires transaction service")
	}
	accountID := strings.TrimSpace(req.AccountID)
	if accountID == "" {
		return nil, fmt.Errorf("account id is required")
	}
	if req.AmountMinor == 0 {
		return nil, fmt.Errorf("adjustment amount must be non-zero")
	}

	tx, err := s.transactions.Create(ctx, &CreateTransactionRequest{
		AccountID:   accountID,
		Type:        models.TransactionTypeAdjustment,
		AmountMinor: req.AmountMinor,
		Description: req.Description,
		OccurredAt:  req.OccurredAt,
	})
	if err != nil {
		return nil, fmt.Errorf("create adjustment transaction: %w", err)
	}

	return tx, nil
}
