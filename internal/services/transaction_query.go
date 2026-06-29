package services

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	domaintransaction "github.com/sunriseex/capitalflow/internal/domain/transaction"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type TransactionListFilter struct {
	AccountID  string
	CategoryID string
	Type       models.TransactionType
	FromDate   time.Time
	ToDate     time.Time
	Search     string
	Limit      int
	Page       int
}

type filteredTransactionLister interface {
	ListByUserFiltered(ctx context.Context, userID string, filter *repository.TransactionListFilter) ([]models.Transaction, error)
}

func (s *TransactionService) ListByUser(ctx context.Context, userID string, filter *TransactionListFilter) ([]models.Transaction, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("transaction repository is required")
	}
	if filter == nil {
		filter = &TransactionListFilter{}
	}
	userID = strings.TrimSpace(userID)
	if filter.Type != "" && !domaintransaction.ValidType(filter.Type) {
		return nil, validationError("invalid type: " + string(filter.Type))
	}
	if !filter.FromDate.IsZero() && !filter.ToDate.IsZero() && filter.ToDate.Before(filter.FromDate) {
		return nil, validationError("to_date must be on or after from_date")
	}
	if filter.Limit < 0 || filter.Page < 0 {
		return nil, validationError("pagination values must be positive")
	}
	repoFilter := repository.TransactionListFilter{
		AccountID: filter.AccountID, CategoryID: filter.CategoryID, Type: filter.Type,
		FromDate: filter.FromDate, ToDate: filter.ToDate, Search: filter.Search, Limit: filter.Limit, Page: filter.Page,
	}
	if filtered, ok := s.repo.(filteredTransactionLister); ok {
		listed, err := filtered.ListByUserFiltered(ctx, userID, &repoFilter)
		if err != nil {
			return nil, fmt.Errorf("list filtered transactions: %w", err)
		}
		return listed, nil
	}

	var (
		listed []models.Transaction
		err    error
	)
	if filter.AccountID == "" {
		listed, err = s.repo.ListByUser(ctx, userID)
	} else {
		listed, err = s.repo.ListByAccountForUser(ctx, filter.AccountID, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	return applyTransactionListFilter(listed, filter), nil
}

func (s *TransactionService) GetByIDForUser(ctx context.Context, transactionID, userID string) (*models.Transaction, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("transaction repository is required")
	}
	transaction, err := s.repo.GetByIDForUser(ctx, strings.TrimSpace(transactionID), strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("get transaction: %w", err)
	}
	return transaction, nil
}

func (s *TransactionService) ListTransfersByUser(ctx context.Context, userID string) ([]models.Transfer, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("transaction repository is required")
	}
	transfers, err := s.repo.ListTransfersByUser(ctx, strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("list transfers: %w", err)
	}
	return transfers, nil
}

func applyTransactionListFilter(transactions []models.Transaction, filter *TransactionListFilter) []models.Transaction {
	transactions = slices.Clone(transactions)
	slices.SortFunc(transactions, func(a, b models.Transaction) int {
		if byOccurredAt := b.OccurredAt.Compare(a.OccurredAt); byOccurredAt != 0 {
			return byOccurredAt
		}
		if byCreatedAt := b.CreatedAt.Compare(a.CreatedAt); byCreatedAt != 0 {
			return byCreatedAt
		}
		return cmp.Compare(b.ID, a.ID)
	})
	filtered := make([]models.Transaction, 0, len(transactions))
	for i := range transactions {
		transaction := transactions[i]
		if filter.CategoryID != "" && (transaction.CategoryID == nil || *transaction.CategoryID != filter.CategoryID) {
			continue
		}
		if filter.Type != "" && transaction.Type != filter.Type {
			continue
		}
		occurredAt := transactionDateOnly(transaction.OccurredAt)
		if !filter.FromDate.IsZero() && occurredAt.Before(transactionDateOnly(filter.FromDate)) {
			continue
		}
		if !filter.ToDate.IsZero() && occurredAt.After(transactionDateOnly(filter.ToDate)) {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(transaction.Description), filter.Search) {
			continue
		}
		filtered = append(filtered, transaction)
	}
	if filter.Limit <= 0 {
		return filtered
	}
	page := max(filter.Page, 1)
	start := (page - 1) * filter.Limit
	if start >= len(filtered) {
		return []models.Transaction{}
	}
	return filtered[start:min(start+filter.Limit, len(filtered))]
}

func transactionDateOnly(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}
