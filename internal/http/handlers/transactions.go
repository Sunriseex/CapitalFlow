package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/models"
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

	transactions, err := h.app.TransactionQueries.ListByUser(r.Context(), userID, &filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.TransactionsFromModels(transactions))
}

func parseTransactionListFilter(w http.ResponseWriter, r *http.Request) (services.TransactionListFilter, bool) {
	query := r.URL.Query()
	filter := services.TransactionListFilter{
		AccountID:  strings.TrimSpace(query.Get("account_id")),
		CategoryID: strings.TrimSpace(query.Get("category_id")),
		Type:       models.TransactionType(strings.TrimSpace(query.Get("type"))),
		Search:     strings.ToLower(strings.TrimSpace(query.Get("search"))),
		Page:       1,
	}

	if !validateOptionalUUID(w, filter.AccountID, "account_id") ||
		!validateOptionalUUID(w, filter.CategoryID, "category_id") {
		return services.TransactionListFilter{}, false
	}

	var err error
	filter.FromDate, err = parseOptionalDate(query.Get("from_date"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return services.TransactionListFilter{}, false
	}
	filter.ToDate, err = parseOptionalDate(query.Get("to_date"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return services.TransactionListFilter{}, false
	}
	if !filter.FromDate.IsZero() && !filter.ToDate.IsZero() && filter.ToDate.Before(filter.FromDate) {
		writeError(w, http.StatusBadRequest, "validation_error", "to_date must be on or after from_date", nil)
		return services.TransactionListFilter{}, false
	}

	filter.Limit, err = parseOptionalPositiveInt(query.Get("limit"), "limit")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return services.TransactionListFilter{}, false
	}
	filter.Page, err = parseOptionalPositiveInt(query.Get("page"), "page")
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return services.TransactionListFilter{}, false
	}
	if filter.Page == 0 {
		filter.Page = 1
	}

	return filter, true
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

	transaction, err := h.app.Transactions.GetByIDForUser(r.Context(), transactionID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.TransactionFromModel(transaction))
}
