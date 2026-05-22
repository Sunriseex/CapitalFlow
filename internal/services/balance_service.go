package services

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/pkg/money"
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
	Balance      decimal.Decimal
	BalanceMinor int64
	Count        int
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
	balanceMinor, err := money.DecimalToMinorUnits(balance)
	if err != nil {
		return nil, fmt.Errorf("balance cannot be represented as legacy minor units: %w", err)
	}

	return &CalculateBalanceResponse{
		AccountID:    req.AccountID,
		Balance:      balance,
		BalanceMinor: balanceMinor,
		Count:        count,
	}, nil
}

func transactionDelta(tx *models.Transaction) (decimal.Decimal, error) {
	amount := money.MinorUnitsToDecimal(tx.AmountMinor)
	switch tx.Type {
	case models.TransactionTypeInitialBalance,
		models.TransactionTypeIncome,
		models.TransactionTypeTransferIn,
		models.TransactionTypeInterestIncome,
		models.TransactionTypeAdjustment:
		return amount, nil
	case models.TransactionTypeExpense,
		models.TransactionTypeTransferOut:
		return amount.Neg(), nil
	default:
		return decimal.Zero, fmt.Errorf("unknown transaction type: %s", tx.Type)
	}
}
