package services

import (
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
)

func TestAdjustmentServiceCreate(t *testing.T) {
	occurredAt := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)

	tx, err := NewAdjustmentService(NewTransactionService(&recordingCreateForUserRepo{})).Create(t.Context(), CreateAdjustmentRequest{
		AccountID:   " account-1 ",
		Amount:      dec("-50"),
		Description: " Balance correction ",
		OccurredAt:  occurredAt,
	})
	if err != nil {
		t.Fatalf("create adjustment: %v", err)
	}
	if tx.ID == "" {
		t.Fatal("id is empty")
	}
	if tx.AccountID != "account-1" {
		t.Fatalf("account id = %s, want account-1", tx.AccountID)
	}
	if tx.Type != models.TransactionTypeAdjustment {
		t.Fatalf("type = %s, want adjustment", tx.Type)
	}
	if !tx.Amount.Equal(dec("-50")) {
		t.Fatalf("amount = %d, want -5000", tx.Amount)
	}
	if tx.Description != "Balance correction" {
		t.Fatalf("description = %q, want Balance correction", tx.Description)
	}
	if !tx.OccurredAt.Equal(occurredAt) {
		t.Fatalf("occurred at = %s, want %s", tx.OccurredAt, occurredAt)
	}
}

func TestAdjustmentServiceCreateRejectsMissingTransactionService(t *testing.T) {
	_, err := NewAdjustmentService(nil).Create(t.Context(), CreateAdjustmentRequest{
		AccountID: "account-1",
		Amount:    dec("1"),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAdjustmentServiceCreateValidatesInput(t *testing.T) {
	_, err := NewAdjustmentService(nil).Create(t.Context(), CreateAdjustmentRequest{
		AccountID: "account-1",
		Amount:    dec("0"),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
