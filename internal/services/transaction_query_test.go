package services

import (
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
)

func TestApplyTransactionListFilter(t *testing.T) {
	categoryID := "category-1"
	transactions := []models.Transaction{
		{ID: "old", Type: models.TransactionTypeIncome, CategoryID: &categoryID, Description: "Salary May", OccurredAt: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)},
		{ID: "expense", Type: models.TransactionTypeExpense, Description: "Food", OccurredAt: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)},
		{ID: "new", Type: models.TransactionTypeIncome, CategoryID: &categoryID, Description: "Salary June", OccurredAt: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)},
	}
	got := applyTransactionListFilter(transactions, &TransactionListFilter{
		CategoryID: categoryID, Type: models.TransactionTypeIncome, FromDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		ToDate: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), Search: "salary", Limit: 1, Page: 2,
	})
	if len(got) != 1 || got[0].ID != "old" {
		t.Fatalf("filtered = %#v, want old", got)
	}
}

func TestApplyTransactionListFilterUsesStableNewestFirstOrder(t *testing.T) {
	stamp := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	transactions := []models.Transaction{
		{ID: "1", OccurredAt: stamp, CreatedAt: stamp},
		{ID: "3", OccurredAt: stamp, CreatedAt: stamp},
		{ID: "2", OccurredAt: stamp, CreatedAt: stamp},
	}
	for page, want := range []string{"3", "2", "1"} {
		got := applyTransactionListFilter(transactions, &TransactionListFilter{Limit: 1, Page: page + 1})
		if len(got) != 1 || got[0].ID != want {
			t.Fatalf("page %d = %#v, want %s", page+1, got, want)
		}
	}
}

func TestTransactionServiceListRejectsInvalidType(t *testing.T) {
	_, err := NewTransactionService(&recordingTransactionRepo{}).ListByUser(t.Context(), "user-1", &TransactionListFilter{Type: "bad"})
	if !IsValidationError(err) {
		t.Fatalf("error = %v, want validation error", err)
	}
}
