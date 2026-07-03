package services

import (
	"context"
	"fmt"
	"strings"

	domaintransaction "github.com/sunriseex/capitalflow/internal/domain/transaction"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

const (
	defaultTransactionListLimit = 50
	maxTransactionListLimit     = 500
)

type TransactionListFilter = repository.TransactionListFilter

type TransactionQuery struct {
	repo repository.TransactionQueryRepository
}

func NewTransactionQuery(repo repository.TransactionQueryRepository) *TransactionQuery {
	return &TransactionQuery{repo: repo}
}

func (q *TransactionQuery) ListByUser(ctx context.Context, userID string, filter *TransactionListFilter) ([]models.Transaction, error) {
	if q == nil || q.repo == nil {
		return nil, fmt.Errorf("transaction query repository is required")
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
	if filter.Limit == 0 {
		filter.Limit = defaultTransactionListLimit
	}
	if filter.Limit > maxTransactionListLimit {
		return nil, validationError("limit must not exceed 500")
	}
	if filter.Page == 0 {
		filter.Page = 1
	}
	listed, err := q.repo.ListByUserFiltered(ctx, userID, filter)
	if err != nil {
		return nil, fmt.Errorf("list filtered transactions: %w", err)
	}
	return listed, nil
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
