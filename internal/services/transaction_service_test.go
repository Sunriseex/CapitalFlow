package services

import (
	"testing"

	"github.com/sunriseex/finance-manager/internal/models"
)

func TestTransactionServiceCreate(t *testing.T) {
	tx, err := NewTransactionService().Create(t.Context(), &CreateTransactionRequest{
		AccountID:   "account-1",
		Type:        models.TransactionTypeIncome,
		AmountMinor: 10_000,
		Description: " Salary ",
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	if tx.ID == "" {
		t.Fatal("id is empty")
	}
	if tx.Description != "Salary" {
		t.Fatalf("description = %q, want Salary", tx.Description)
	}
	if tx.OccurredAt.IsZero() {
		t.Fatal("occurred at is zero")
	}
}

func TestTransactionServiceCreateValidatesInput(t *testing.T) {
	_, err := NewTransactionService().Create(t.Context(), &CreateTransactionRequest{
		AccountID:   "account-1",
		Type:        models.TransactionTypeIncome,
		AmountMinor: 0,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTransactionServiceCreateRejectsNegativeNonAdjustmentAmounts(t *testing.T) {
	tests := []models.TransactionType{
		models.TransactionTypeInitialBalance,
		models.TransactionTypeIncome,
		models.TransactionTypeExpense,
		models.TransactionTypeTransferIn,
		models.TransactionTypeTransferOut,
		models.TransactionTypeInterestIncome,
	}

	for _, transactionType := range tests {
		t.Run(string(transactionType), func(t *testing.T) {
			_, err := NewTransactionService().Create(t.Context(), &CreateTransactionRequest{
				AccountID:   "account-1",
				Type:        transactionType,
				AmountMinor: -1,
			})
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestTransactionServiceCreateAllowsNegativeAdjustments(t *testing.T) {
	tx, err := NewTransactionService().Create(t.Context(), &CreateTransactionRequest{
		AccountID:   "account-1",
		Type:        models.TransactionTypeAdjustment,
		AmountMinor: -1_000,
	})
	if err != nil {
		t.Fatalf("create adjustment transaction: %v", err)
	}
	if tx.AmountMinor != -1_000 {
		t.Fatalf("amount = %d, want -1000", tx.AmountMinor)
	}
}
