package handlers

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/internal/services"
)

func (h *Handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	filter, ok := parseTransactionListFilter(w, r)
	if !ok {
		return
	}

	transactionsRepo := h.app.Store.Transactions()
	transactions, err := listTransactionsForUser(r.Context(), transactionsRepo, userID, &filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.TransactionsFromModels(transactions))
}

type filteredTransactionLister interface {
	ListByUserFiltered(ctx context.Context, userID string, filter *repository.TransactionListFilter) ([]models.Transaction, error)
}

func listTransactionsForUser(ctx context.Context, transactions repository.TransactionRepository, userID string, filter *repository.TransactionListFilter) ([]models.Transaction, error) {
	if filtered, ok := transactions.(filteredTransactionLister); ok {
		listed, err := filtered.ListByUserFiltered(ctx, userID, filter)
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
		listed, err = transactions.ListByUser(ctx, userID)
	} else {
		listed, err = transactions.ListByAccountForUser(ctx, filter.AccountID, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	return applyTransactionListFilter(listed, filter), nil
}

func parseTransactionListFilter(w http.ResponseWriter, r *http.Request) (repository.TransactionListFilter, bool) {
	query := r.URL.Query()
	filter := repository.TransactionListFilter{
		AccountID:  strings.TrimSpace(query.Get("account_id")),
		CategoryID: strings.TrimSpace(query.Get("category_id")),
		Type:       models.TransactionType(strings.TrimSpace(query.Get("type"))),
		Search:     strings.ToLower(strings.TrimSpace(query.Get("search"))),
		Page:       1,
	}

	if !validateOptionalUUID(w, filter.AccountID, "account_id") ||
		!validateOptionalUUID(w, filter.CategoryID, "category_id") {
		return repository.TransactionListFilter{}, false
	}

	if filter.Type != "" && !validTransactionFilterType(filter.Type) {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid type: "+string(filter.Type), nil)
		return repository.TransactionListFilter{}, false
	}

	var err error
	filter.FromDate, err = parseOptionalDate(query.Get("from_date"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return repository.TransactionListFilter{}, false
	}
	filter.ToDate, err = parseOptionalDate(query.Get("to_date"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return repository.TransactionListFilter{}, false
	}
	if !filter.FromDate.IsZero() && !filter.ToDate.IsZero() && filter.ToDate.Before(filter.FromDate) {
		writeError(w, http.StatusBadRequest, "validation_error", "to_date must be on or after from_date", nil)
		return repository.TransactionListFilter{}, false
	}

	filter.Limit, err = parseOptionalPositiveInt(query.Get("limit"), "limit")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return repository.TransactionListFilter{}, false
	}
	filter.Page, err = parseOptionalPositiveInt(query.Get("page"), "page")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return repository.TransactionListFilter{}, false
	}
	if filter.Page == 0 {
		filter.Page = 1
	}

	return filter, true
}

func validTransactionFilterType(transactionType models.TransactionType) bool {
	switch transactionType {
	case models.TransactionTypeInitialBalance,
		models.TransactionTypeIncome,
		models.TransactionTypeExpense,
		models.TransactionTypeTransferIn,
		models.TransactionTypeTransferOut,
		models.TransactionTypeInterestIncome,
		models.TransactionTypeAdjustment:
		return true
	default:
		return false
	}
}

func parseOptionalPositiveInt(input, field string) (int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, nil
	}

	value, err := strconv.Atoi(input)
	if err != nil || value <= 0 {
		return 0, errValidation(field + " must be a positive integer")
	}
	return value, nil
}

func applyTransactionListFilter(transactions []models.Transaction, filter *repository.TransactionListFilter) []models.Transaction {
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
		occurredAt := dateOnly(transaction.OccurredAt)
		if !filter.FromDate.IsZero() && occurredAt.Before(dateOnly(filter.FromDate)) {
			continue
		}
		if !filter.ToDate.IsZero() && occurredAt.After(dateOnly(filter.ToDate)) {
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

	start := (filter.Page - 1) * filter.Limit
	if start >= len(filtered) {
		return []models.Transaction{}
	}
	end := min(start+filter.Limit, len(filtered))
	return filtered[start:end]
}

func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	var req dto.CreateTransactionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	occurredAt, err := parseOptionalDate(req.OccurredAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}

	accountID := strings.TrimSpace(req.AccountID)
	if !validateOptionalUUID(w, accountID, "account_id") {
		return
	}

	var relatedAccountID *string
	if req.RelatedAccountID != nil {
		normalized := strings.TrimSpace(*req.RelatedAccountID)
		if !validateOptionalUUID(w, normalized, "related_account_id") {
			return
		}

		if normalized != "" {
			relatedAccountID = &normalized
		}
	}

	var categoryID *string
	if req.CategoryID != nil {
		normalized := strings.TrimSpace(*req.CategoryID)
		if !validateOptionalUUID(w, normalized, "category_id") {
			return
		}

		if normalized != "" {
			categoryID = &normalized
		}
	}

	transaction, err := h.app.Transactions.CreateForUser(r.Context(), userID, &services.CreateTransactionRequest{
		AccountID:        accountID,
		RelatedAccountID: relatedAccountID,
		Type:             req.Type,
		Amount:           req.Amount.Decimal,
		CategoryID:       categoryID,
		Description:      req.Description,
		OccurredAt:       occurredAt,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.TransactionFromModel(transaction))
}

func (h *Handler) getTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	transactionID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}

	transaction, err := h.app.Store.Transactions().GetByIDForUser(r.Context(), transactionID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.TransactionFromModel(transaction))
}
