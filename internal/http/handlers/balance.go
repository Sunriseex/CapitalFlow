package handlers

import (
	"net/http"

	"github.com/sunriseex/capitalflow/pkg/money"
)

func (h *Handler) getAccountBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	accountID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}
	if _, err := h.store.Accounts().GetByIDForUser(r.Context(), accountID, userID); err != nil {
		writeServiceError(w, err)
		return
	}

	balance, count, err := h.store.Transactions().GetBalanceByAccountForUser(r.Context(), accountID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"account_id":        accountID,
		"balance":           money.NewJSONDecimal(balance),
		"transaction_count": count,
	})
}
