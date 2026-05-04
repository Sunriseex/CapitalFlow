package services

import (
	"context"
	"fmt"

	"github.com/sunriseex/finance-manager/internal/models"
)

type BalanceService struct{}

func NewBalanceService() *BalanceService {
	return &BalanceService{}
}

type CalculateBalanceRequest struct {
	AccountID    string
	Transactions []models.Transaction
}

type CalculateBalanceResponse struct {
	AccountID    string
	BalanceMinor int64
	Count        int
}

func (s *BalanceService) Calculate(ctx context.Context, req CalculateBalanceRequest) (*CalculateBalanceResponse, error) {
	if req.AccountID == "" {
		return nil, fmt.Errorf("account id is required")
	}

	var balance int64
	var count int
	for _, tx := range req.Transactions {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("calculate balance: %w", ctx.Err())
		default:
		}

		if tx.AccountID != req.AccountID {
			continue
		}

		delta, err := transactionDelta(tx)
		if err != nil {
			return nil, err
		}
		balance += delta
		count++
	}

	return &CalculateBalanceResponse{
		AccountID:    req.AccountID,
		BalanceMinor: balance,
		Count:        count,
	}, nil
}

func transactionDelta(tx models.Transaction) (int64, error) {
	switch tx.Type {
	case models.TransactionTypeInitialBalance,
		models.TransactionTypeIncome,
		models.TransactionTypeTransferIn,
		models.TransactionTypeInterestIncome,
		models.TransactionTypeAdjustment:
		return tx.AmountMinor, nil
	case models.TransactionTypeExpense,
		models.TransactionTypeTransferOut:
		return -tx.AmountMinor, nil
	default:
		return 0, fmt.Errorf("unknown transaction type: %s", tx.Type)
	}
}
