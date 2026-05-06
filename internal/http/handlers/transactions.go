package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/sunriseex/finance-manager/internal/http/dto"
	"github.com/sunriseex/finance-manager/internal/models"
	"github.com/sunriseex/finance-manager/internal/services"
)

func (h *Handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	accountID := strings.TrimSpace(r.URL.Query().Get("account_id"))

	var (
		transactions []models.Transaction
		err          error
	)
	if accountID == "" {
		transactions, err = h.store.Transactions().List(r.Context())
	} else {
		transactions, err = h.store.Transactions().ListByAccount(r.Context(), accountID)
	}
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.TransactionsFromModels(transactions))
}

func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
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

	transaction, err := services.NewTransactionService(h.store.Transactions()).Create(r.Context(), &services.CreateTransactionRequest{
		AccountID:        req.AccountID,
		RelatedAccountID: req.RelatedAccountID,
		Type:             req.Type,
		AmountMinor:      req.AmountMinor,
		CategoryID:       req.CategoryID,
		Description:      req.Description,
		OccurredAt:       occurredAt,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusCreated, dto.TransactionFromModel(transaction))
}

func (h *Handler) getTransaction(w http.ResponseWriter, r *http.Request) {
	transaction, err := h.store.Transactions().GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.TransactionFromModel(transaction))
}

func (h *Handler) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Transactions().Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
