package services

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
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
			if !got.Balance.Equal(decimal.NewFromInt(tt.want)) {
				t.Fatalf("decimal balance = %s, want %d", got.Balance, tt.want)
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

func TestBalanceServiceCalculateRejectsOverflow(t *testing.T) {
	tests := []struct {
		name         string
		transactions []models.Transaction
	}{
		{
			name: "positive overflow",
			transactions: []models.Transaction{
				{AccountID: "account-1", Type: models.TransactionTypeIncome, AmountMinor: math.MaxInt64},
				{AccountID: "account-1", Type: models.TransactionTypeIncome, AmountMinor: 1},
			},
		},
		{
			name: "negative overflow",
			transactions: []models.Transaction{
				{AccountID: "account-1", Type: models.TransactionTypeAdjustment, AmountMinor: math.MinInt64 + 1},
				{AccountID: "account-1", Type: models.TransactionTypeExpense, AmountMinor: 2},
			},
		},
		{
			name: "legacy output overflow",
			transactions: []models.Transaction{
				{AccountID: "account-1", Type: models.TransactionTypeExpense, AmountMinor: math.MinInt64},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewBalanceService().Calculate(t.Context(), CalculateBalanceRequest{
				AccountID:    "account-1",
				Transactions: tt.transactions,
			})
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
