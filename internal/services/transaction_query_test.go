package services

import (
	"context"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
)

func TestTransactionQueryRejectsInvalidType(t *testing.T) {
	_, err := NewTransactionQuery(&recordingTransactionQuery{}).ListByUser(t.Context(), "user-1", &TransactionListFilter{Type: "bad"})
	if !IsValidationError(err) {
		t.Fatalf("error = %v, want validation error", err)
	}
}

func TestTransactionQueryDelegatesBoundedFilter(t *testing.T) {
	repo := &recordingTransactionQuery{transactions: []models.Transaction{{ID: "tx-1"}}}
	filter := &TransactionListFilter{
		AccountID: "account-1",
		Limit:     10,
		Page:      2,
		FromDate:  time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	got, err := NewTransactionQuery(repo).ListByUser(t.Context(), " user-1 ", filter)
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if len(got) != 1 || got[0].ID != "tx-1" || repo.userID != "user-1" || repo.filter != filter {
		t.Fatalf("query = user %q filter %#v result %#v", repo.userID, repo.filter, got)
	}
}

func TestTransactionQueryAppliesDefaultAndMaximumLimit(t *testing.T) {
	repo := &recordingTransactionQuery{}
	filter := &TransactionListFilter{}
	if _, err := NewTransactionQuery(repo).ListByUser(t.Context(), "user-1", filter); err != nil {
		t.Fatalf("default limit: %v", err)
	}
	if filter.Limit != defaultTransactionListLimit || filter.Page != 1 {
		t.Fatalf("pagination = limit %d page %d", filter.Limit, filter.Page)
	}

	_, err := NewTransactionQuery(repo).ListByUser(t.Context(), "user-1", &TransactionListFilter{Limit: maxTransactionListLimit + 1})
	if !IsValidationError(err) {
		t.Fatalf("error = %v, want validation error", err)
	}
}

type recordingTransactionQuery struct {
	userID       string
	filter       *TransactionListFilter
	transactions []models.Transaction
}

func (q *recordingTransactionQuery) ListByUserFiltered(_ context.Context, userID string, filter *TransactionListFilter) ([]models.Transaction, error) {
	q.userID = userID
	q.filter = filter
	return q.transactions, nil
}
