package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	domainmoney "github.com/sunriseex/capitalflow/internal/domain/money"
	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/pkg/money"
)

func (h *Handler) listCategoryLimits(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	limits, err := h.app.Store.CategoryLimits().ListByUser(r.Context(), userID)
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
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}
	categoryID := strings.TrimSpace(req.CategoryID)
	if _, err := uuid.Parse(categoryID); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "category_id must be a UUID", nil)
		return
	}
	if _, err := h.app.Store.Categories().GetByID(r.Context(), categoryID); err != nil {
		writeServiceError(w, err)
		return
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	amount, ok := parseCategoryLimitAmount(w, req.Amount, currency)
	if !ok {
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	now := time.Now().UTC()
	limit := &models.CategoryLimit{
		ID:          uuid.NewString(),
		OwnerUserID: userID,
		CategoryID:  categoryID,
		Amount:      amount,
		Currency:    currency,
		IsActive:    isActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.app.Store.CategoryLimits().Create(r.Context(), limit); err != nil {
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
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}
	limit, err := h.app.Store.CategoryLimits().GetByIDForUser(r.Context(), limitID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if req.CategoryID != nil {
		categoryID := strings.TrimSpace(*req.CategoryID)
		if _, err := uuid.Parse(categoryID); err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "category_id must be a UUID", nil)
			return
		}
		if _, err := h.app.Store.Categories().GetByID(r.Context(), categoryID); err != nil {
			writeServiceError(w, err)
			return
		}
		limit.CategoryID = categoryID
	}
	if req.Currency != nil {
		limit.Currency = strings.ToUpper(strings.TrimSpace(*req.Currency))
	}
	if req.Amount != nil || req.Currency != nil {
		rawAmount := limit.Amount.String()
		if req.Amount != nil {
			rawAmount = *req.Amount
		}
		amount, ok := parseCategoryLimitAmount(w, rawAmount, limit.Currency)
		if !ok {
			return
		}
		limit.Amount = amount
	}
	if req.IsActive != nil {
		limit.IsActive = *req.IsActive
	}
	limit.UpdatedAt = time.Now().UTC()
	if err := h.app.Store.CategoryLimits().UpdateForUser(r.Context(), limit, userID); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.CategoryLimitFromModel(limit))
}

func parseCategoryLimitAmount(w http.ResponseWriter, rawAmount, currency string) (decimal.Decimal, bool) {
	if !currencyCodePattern.MatchString(currency) {
		writeError(w, http.StatusBadRequest, "validation_error", "Currency must be a 3-letter code", nil)
		return decimal.Zero, false
	}
	amount, err := money.ParseDecimalString(rawAmount)
	if err != nil || !amount.IsPositive() {
		writeError(w, http.StatusBadRequest, "validation_error", "Limit amount must be positive", nil)
		return decimal.Zero, false
	}
	if err := domainmoney.ValidateCurrencyScale(amount, currency); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return decimal.Zero, false
	}
	return amount, true
}
