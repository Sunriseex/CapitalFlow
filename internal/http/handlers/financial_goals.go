package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/services"
	"github.com/sunriseex/capitalflow/pkg/money"
)

func (h *Handler) listFinancialGoals(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	goals, err := h.app.FinancialGoals.ListByUser(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.FinancialGoalsFromModels(goals))
}

func (h *Handler) createFinancialGoal(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req dto.CreateFinancialGoalRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}
	accountID := strings.TrimSpace(req.AccountID)
	if _, err := uuid.Parse(accountID); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "account_id must be a UUID", nil)
		return
	}
	amount, _ := money.ParseDecimalString(req.TargetAmount)
	var targetDate *time.Time
	if req.TargetDate != "" {
		parsed, err := time.Parse(time.DateOnly, req.TargetDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "Target date must use YYYY-MM-DD", nil)
			return
		}
		targetDate = &parsed
	}
	goal, err := h.app.FinancialGoals.Create(r.Context(), &services.CreateFinancialGoalRequest{
		UserID: userID, AccountID: accountID, Name: req.Name, TargetAmount: amount, TargetDate: targetDate,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.FinancialGoalFromModel(goal))
}

func (h *Handler) updateFinancialGoal(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	goalID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req dto.UpdateFinancialGoalRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}
	if req.AccountID != nil {
		accountID := strings.TrimSpace(*req.AccountID)
		if _, err := uuid.Parse(accountID); err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "account_id must be a UUID", nil)
			return
		}
		req.AccountID = &accountID
	}
	var targetAmount *decimal.Decimal
	if req.TargetAmount != nil {
		amount, _ := money.ParseDecimalString(*req.TargetAmount)
		targetAmount = &amount
	}
	var targetDate *time.Time
	if req.TargetDate != nil && strings.TrimSpace(*req.TargetDate) != "" {
		parsed, err := time.Parse(time.DateOnly, strings.TrimSpace(*req.TargetDate))
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "target_date must use YYYY-MM-DD", nil)
			return
		}
		targetDate = &parsed
	}
	goal, err := h.app.FinancialGoals.Update(r.Context(), &services.UpdateFinancialGoalRequest{
		ID: goalID, UserID: userID, AccountID: req.AccountID, Name: req.Name, TargetAmount: targetAmount,
		TargetDateSet: req.TargetDate != nil, TargetDate: targetDate, Status: req.Status,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.FinancialGoalFromModel(goal))
}
