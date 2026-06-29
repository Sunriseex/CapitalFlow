package handlers

import (
	"net/http"
	"time"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/services"
)

func (h *Handler) listAccounts(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	accounts, err := h.app.Accounts.ListByUser(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.AccountsFromModels(accounts))
}

func (h *Handler) createAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	var req dto.CreateAccountRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	openedAt, err := parseOptionalDate(req.OpenedAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}

	account, err := h.app.Accounts.Create(r.Context(), &services.CreateAccountRequest{
		OwnerUserID: userID,
		Name:        req.Name,
		Bank:        req.Bank,
		Type:        req.Type,
		Currency:    req.Currency,
		OpenedAt:    openedAt,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.AccountFromModel(account))
}

func (h *Handler) getAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	accountID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}

	account, err := h.app.Accounts.GetByIDForUser(r.Context(), accountID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.AccountFromModel(account))
}

func (h *Handler) updateAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	accountID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req dto.UpdateAccountRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	var openedAt *time.Time
	if req.OpenedAt != nil {
		parsed, err := parseOptionalDate(*req.OpenedAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
			return
		}
		openedAt = &parsed
	}

	account, err := h.app.Accounts.UpdateForUser(r.Context(), &services.UpdateAccountRequest{
		ID:       accountID,
		UserID:   userID,
		Name:     req.Name,
		Bank:     req.Bank,
		Type:     req.Type,
		Currency: req.Currency,
		OpenedAt: openedAt,
		IsActive: req.IsActive,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.AccountFromModel(account))
}

func (h *Handler) archiveAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	accountID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}

	if err := h.app.Accounts.ArchiveForUser(r.Context(), accountID, userID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
