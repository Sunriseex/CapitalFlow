package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	appmiddleware "github.com/sunriseex/capitalflow/internal/http/middleware"
	"github.com/sunriseex/capitalflow/internal/services"
	"github.com/sunriseex/capitalflow/pkg/money"
)

func (h *Handler) listTransfers(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	transfers, err := h.store.Transactions().ListTransfersByUser(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response := make([]dto.TransferEventResponse, 0, len(transfers))
	for i := range transfers {
		transfer := transfers[i]
		response = append(response, dto.TransferEventResponse{
			ID:                   transfer.ID,
			UserID:               transfer.UserID,
			FromAccountID:        transfer.FromAccountID,
			ToAccountID:          transfer.ToAccountID,
			FromTransactionID:    transfer.FromTransactionID,
			ToTransactionID:      transfer.ToTransactionID,
			FeeTransactionID:     transfer.FeeTransactionID,
			FromAmount:           money.NewJSONDecimal(transfer.FromAmount),
			ToAmount:             money.NewJSONDecimal(transfer.ToAmount),
			FromCurrency:         transfer.FromCurrency,
			ToCurrency:           transfer.ToCurrency,
			ExchangeRate:         transfer.ExchangeRate,
			ExchangeRateScale:    transfer.ExchangeRateScale,
			ExchangeRateProvider: transfer.ExchangeRateProvider,
			ExchangeRateDate:     transfer.ExchangeRateDate.Format(time.RFC3339),
			FeeAmount:            money.NewJSONDecimal(transfer.FeeAmount),
			FeeCurrency:          transfer.FeeCurrency,
			Status:               transfer.Status,
			CreatedAt:            transfer.CreatedAt.Format(time.RFC3339),
			UpdatedAt:            transfer.UpdatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) createTransfer(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	var req dto.CreateTransferRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	fromAccountID := strings.TrimSpace(req.FromAccountID)
	toAccountID := strings.TrimSpace(req.ToAccountID)

	if !validateOptionalUUID(w, fromAccountID, "from_account_id") {
		return
	}
	if !validateOptionalUUID(w, toAccountID, "to_account_id") {
		return
	}

	fromAccount, ok := h.accountByID(w, r, fromAccountID, "from_account_id")
	if !ok {
		return
	}
	toAccount, ok := h.accountByID(w, r, toAccountID, "to_account_id")
	if !ok {
		return
	}

	result, err := h.transfers.Create(r.Context(), &services.CreateTransferRequest{
		UserID:         userID,
		FromAccountID:  fromAccountID,
		ToAccountID:    toAccountID,
		FromCurrency:   fromAccount.Currency,
		ToCurrency:     toAccount.Currency,
		Amount:         req.Amount.Decimal,
		FeeAmount:      req.FeeAmount.Decimal,
		FeeCurrency:    req.FeeCurrency,
		Description:    req.Description,
		IdempotencyKey: r.Header.Get(appmiddleware.IdempotencyKeyHeader),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response := dto.TransferResponse{
		Out:          dto.TransactionFromModel(result.Out),
		In:           dto.TransactionFromModel(result.In),
		ExchangeRate: result.ExchangeRate,
	}
	if result.Fee != nil {
		fee := dto.TransactionFromModel(result.Fee)
		response.Fee = &fee
	}
	writeJSON(w, http.StatusCreated, response)
}
