package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/services"
)

func (h *Handler) listAccounts(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	accounts, err := h.store.Accounts().ListByUser(r.Context(), userID)
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

	account, err := h.accounts.Create(r.Context(), &services.CreateAccountRequest{
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

	account, err := h.store.Accounts().GetByIDForUser(r.Context(), accountID, userID)
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

	account, err := h.store.Accounts().GetByIDForUser(r.Context(), accountID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	var req dto.UpdateAccountRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	if req.Currency != nil {
		currency := strings.ToUpper(strings.TrimSpace(*req.Currency))
		if currency != account.Currency {
			_, transactionCount, err := h.store.Transactions().GetBalanceByAccountForUser(r.Context(), account.ID, userID)
			if err != nil {
				writeServiceError(w, err)
				return
			}
			if transactionCount > 0 {
				writeError(w, http.StatusBadRequest, "validation_error", "account currency cannot be changed after transactions exist", nil)
				return
			}
		}
	}

	if req.Name != nil {
		account.Name = strings.TrimSpace(*req.Name)
	}
	if req.Bank != nil {
		account.Bank = strings.TrimSpace(*req.Bank)
	}
	if req.Type != nil {
		account.Type = *req.Type
	}
	if req.Currency != nil {
		account.Currency = strings.ToUpper(strings.TrimSpace(*req.Currency))
	}
	if req.OpenedAt != nil {
		openedAt, err := parseOptionalDate(*req.OpenedAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
			return
		}
		if !openedAt.IsZero() {
			account.OpenedAt = openedAt
		}
	}
	if req.IsActive != nil {
		account.IsActive = *req.IsActive
	}

	if err := validateAccount(account); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}

	account.UpdatedAt = time.Now()
	if err := h.store.Accounts().UpdateForUser(r.Context(), account, userID); err != nil {
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

	if err := h.store.Accounts().ArchiveForUser(r.Context(), accountID, userID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateAccount(account *models.Account) error {
	if strings.TrimSpace(account.Name) == "" {
		return errValidation("account name is required")
	}
	if !services.ValidAccountType(account.Type) {
		return errValidation("invalid account type: " + string(account.Type))
	}
	if !services.ValidCurrency(account.Currency) {
		return errValidation("invalid currency: " + account.Currency)
	}
	return nil
}

func (h *Handler) ensureAccountExists(w http.ResponseWriter, r *http.Request, accountID string) bool {
	userID, ok := currentUserID(w, r)
	if !ok {
		return false
	}
	if _, err := h.store.Accounts().GetByIDForUser(r.Context(), accountID, userID); err != nil {
		writeServiceError(w, err)
		return false
	}

	return true
}

func (h *Handler) accountByID(w http.ResponseWriter, r *http.Request, accountID, field string) (*models.Account, bool) {
	if strings.TrimSpace(accountID) == "" {
		writeError(w, http.StatusBadRequest, "validation_error", field+" is required", nil)
		return nil, false
	}

	userID, ok := currentUserID(w, r)
	if !ok {
		return nil, false
	}
	account, err := h.store.Accounts().GetByIDForUser(r.Context(), accountID, userID)
	if err != nil {
		writeServiceError(w, err)
		return nil, false
	}

	return account, true
}
