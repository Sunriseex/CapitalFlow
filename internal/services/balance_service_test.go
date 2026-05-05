package services

import (
	"context"
	"errors"
	"testing"

	"github.com/sunriseex/finance-manager/internal/models"
)

func TestBalanceServiceCalculate(t *testing.T) {
	tests := []struct {
		name         string
		accountID    string
		transactions []models.Transaction
		want         int64
		wantCount    int
	}{
		{
			name:      "adds income and subtracts expenses",
			accountID: "account-1",
			transactions: []models.Transaction{
				{AccountID: "account-1", Type: models.TransactionTypeInitialBalance, AmountMinor: 100_000},
				{AccountID: "account-1", Type: models.TransactionTypeIncome, AmountMinor: 50_000},
				{AccountID: "account-1", Type: models.TransactionTypeExpense, AmountMinor: 20_000},
			},
			want:      130_000,
			wantCount: 3,
		},
		{
			name:      "ignores other accounts",
			accountID: "account-1",
			transactions: []models.Transaction{
				{AccountID: "account-1", Type: models.TransactionTypeInitialBalance, AmountMinor: 100_000},
				{AccountID: "account-2", Type: models.TransactionTypeIncome, AmountMinor: 50_000},
			},
			want:      100_000,
			wantCount: 1,
		},
		{
			name:      "handles transfer directions and adjustments",
			accountID: "account-1",
			transactions: []models.Transaction{
				{AccountID: "account-1", Type: models.TransactionTypeInitialBalance, AmountMinor: 100_000},
				{AccountID: "account-1", Type: models.TransactionTypeTransferOut, AmountMinor: 25_000},
				{AccountID: "account-1", Type: models.TransactionTypeTransferIn, AmountMinor: 10_000},
				{AccountID: "account-1", Type: models.TransactionTypeAdjustment, AmountMinor: -5_000},
			},
			want:      80_000,
			wantCount: 4,
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
			if got.BalanceMinor != tt.want {
				t.Fatalf("balance = %d, want %d", got.BalanceMinor, tt.want)
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
			{AccountID: "account-1", Type: models.TransactionTypeIncome, AmountMinor: 1},
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
