package services

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
)

func TestBalanceServiceCalculate(t *testing.T) {
	tests := []struct {
		name         string
		accountID    string
		transactions []models.Transaction
		want         decimal.Decimal
		wantCount    int
	}{
		{
			name:      "adds income and subtracts expenses",
			accountID: "account-1",
			transactions: []models.Transaction{
				{AccountID: "account-1", Type: models.TransactionTypeInitialBalance, Amount: decimal.RequireFromString("1000")},
				{AccountID: "account-1", Type: models.TransactionTypeIncome, Amount: decimal.RequireFromString("500")},
				{AccountID: "account-1", Type: models.TransactionTypeExpense, Amount: decimal.RequireFromString("200")},
			},
			want:      decimal.RequireFromString("1300"),
			wantCount: 3,
		},
		{
			name:      "ignores other accounts",
			accountID: "account-1",
			transactions: []models.Transaction{
				{AccountID: "account-1", Type: models.TransactionTypeInitialBalance, Amount: decimal.RequireFromString("1000")},
				{AccountID: "account-2", Type: models.TransactionTypeIncome, Amount: decimal.RequireFromString("500")},
			},
			want:      decimal.RequireFromString("1000"),
			wantCount: 1,
		},
		{
			name:      "handles transfer directions and adjustments",
			accountID: "account-1",
			transactions: []models.Transaction{
				{AccountID: "account-1", Type: models.TransactionTypeInitialBalance, Amount: decimal.RequireFromString("1000")},
				{AccountID: "account-1", Type: models.TransactionTypeTransferOut, Amount: decimal.RequireFromString("250")},
				{AccountID: "account-1", Type: models.TransactionTypeTransferIn, Amount: decimal.RequireFromString("100")},
				{AccountID: "account-1", Type: models.TransactionTypeAdjustment, Amount: decimal.RequireFromString("-50")},
			},
			want:      decimal.RequireFromString("800"),
			wantCount: 4,
		},
		{
			name:      "ignores non-confirmed statuses",
			accountID: "account-1",
			transactions: []models.Transaction{
				{AccountID: "account-1", Type: models.TransactionTypeIncome, Status: models.TransactionStatusConfirmed, Amount: decimal.RequireFromString("100")},
				{AccountID: "account-1", Type: models.TransactionTypeIncome, Status: models.TransactionStatusPending, Amount: decimal.RequireFromString("200")},
				{AccountID: "account-1", Type: models.TransactionTypeExpense, Status: models.TransactionStatusCancelled, Amount: decimal.RequireFromString("50")},
				{AccountID: "account-1", Type: models.TransactionTypeExpense, Status: models.TransactionStatusReversed, Amount: decimal.RequireFromString("25")},
				{AccountID: "account-1", Type: models.TransactionTypeExpense, Status: models.TransactionStatusSoftDeleted, Amount: decimal.RequireFromString("10")},
			},
			want:      decimal.RequireFromString("75"),
			wantCount: 2,
		},
	}

	service := NewBalanceService()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.Calculate(t.Context(), CalculateBalanceRequest{
				AccountID:    tt.accountID,
				Transactions: tt.transactions,
			})
			if err != nil {
				t.Fatalf("calculate balance: %v", err)
			}
			if !got.Balance.Equal(tt.want) {
				t.Fatalf("balance = %s, want %s", got.Balance, tt.want)
			}
			if got.Count != tt.wantCount {
				t.Fatalf("count = %d, want %d", got.Count, tt.wantCount)
			}
		})
	}
}

func TestBalanceServiceCalculateCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := NewBalanceService().Calculate(ctx, CalculateBalanceRequest{
		AccountID: "account-1",
		Transactions: []models.Transaction{
			{AccountID: "account-1", Type: models.TransactionTypeIncome, Amount: decimal.NewFromInt(1)},
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
