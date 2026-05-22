package services

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
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
	AccountID string
	Balance   decimal.Decimal
	Count     int
}

func (s *BalanceService) Calculate(ctx context.Context, req CalculateBalanceRequest) (*CalculateBalanceResponse, error) {
	if req.AccountID == "" {
		return nil, fmt.Errorf("account id is required")
	}

	balance := decimal.Zero
	var count int
	for i := range req.Transactions {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("calculate balance: %w", ctx.Err())
		default:
		}

		tx := &req.Transactions[i]
		if tx.AccountID != req.AccountID {
			continue
		}

		delta, err := transactionDelta(tx)
		if err != nil {
			return nil, err
		}
		balance = balance.Add(delta)
		count++
	}
	return &CalculateBalanceResponse{
		AccountID: req.AccountID,
		Balance:   balance,
		Count:     count,
	}, nil
}

func transactionDelta(tx *models.Transaction) (decimal.Decimal, error) {
	switch tx.Type {
	case models.TransactionTypeInitialBalance,
		models.TransactionTypeIncome,
		models.TransactionTypeTransferIn,
		models.TransactionTypeInterestIncome,
		models.TransactionTypeAdjustment:
		return tx.Amount, nil
	case models.TransactionTypeExpense,
		models.TransactionTypeTransferOut:
		return tx.Amount.Neg(), nil
	default:
		return decimal.Zero, fmt.Errorf("unknown transaction type: %s", tx.Type)
	}
}
