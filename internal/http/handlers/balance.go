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
	balance, err := h.app.Accounts.BalanceForUser(r.Context(), accountID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"account_id":        balance.AccountID,
		"balance":           money.NewJSONDecimal(balance.Balance),
		"transaction_count": balance.TransactionCount,
	})
}
