package handlers

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/services"
	"github.com/sunriseex/capitalflow/pkg/money"
)

func (h *Handler) listCategoryLimits(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	limits, err := h.app.CategoryLimits.ListByUser(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.CategoryLimitsFromModels(limits))
}

func (h *Handler) createCategoryLimit(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req dto.CreateCategoryLimitRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	categoryID := strings.TrimSpace(req.CategoryID)
	if _, err := uuid.Parse(categoryID); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "category_id must be a UUID", nil)
		return
	}
	amount, _ := money.ParseDecimalString(req.Amount)
	limit, err := h.app.CategoryLimits.Create(r.Context(), &services.CreateCategoryLimitRequest{
		UserID: userID, CategoryID: categoryID, Amount: amount, Currency: req.Currency, IsActive: req.IsActive,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.CategoryLimitFromModel(limit))
}

func (h *Handler) updateCategoryLimit(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	limitID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req dto.UpdateCategoryLimitRequest
	if err := decodeJSON(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if req.CategoryID != nil {
		categoryID := strings.TrimSpace(*req.CategoryID)
		if _, err := uuid.Parse(categoryID); err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "category_id must be a UUID", nil)
			return
		}
		req.CategoryID = &categoryID
	}
	var amount *decimal.Decimal
	if req.Amount != nil {
		parsed, _ := money.ParseDecimalString(*req.Amount)
		amount = &parsed
	}
	limit, err := h.app.CategoryLimits.Update(r.Context(), &services.UpdateCategoryLimitRequest{
		ID: limitID, UserID: userID, CategoryID: req.CategoryID, Amount: amount,
		Currency: req.Currency, IsActive: req.IsActive,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.CategoryLimitFromModel(limit))
}
