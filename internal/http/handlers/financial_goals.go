package handlers

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	domainmoney "github.com/sunriseex/capitalflow/internal/domain/money"
	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/pkg/money"
)

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

func (h *Handler) listFinancialGoals(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	goals, err := h.app.Store.FinancialGoals().ListByUser(r.Context(), userID)
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
	account, err := h.app.Store.Accounts().GetByIDForUser(r.Context(), accountID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	amount, err := money.ParseDecimalString(req.TargetAmount)
	if name == "" || len([]rune(name)) > 100 || err != nil || !amount.IsPositive() {
		writeError(w, http.StatusBadRequest, "validation_error", "Name and a positive target amount are required", nil)
		return
	}
	currency := account.Currency
	if err := domainmoney.ValidateCurrencyScale(amount, currency); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}
	var targetDate *time.Time
	if req.TargetDate != "" {
		parsed, parseErr := time.Parse(time.DateOnly, req.TargetDate)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "Target date must use YYYY-MM-DD", nil)
			return
		}
		targetDate = &parsed
	}
	now := time.Now().UTC()
	goal := &models.FinancialGoal{
		ID:           uuid.NewString(),
		OwnerUserID:  userID,
		AccountID:    &accountID,
		Name:         name,
		TargetAmount: amount,
		Currency:     currency,
		TargetDate:   targetDate,
		Status:       models.FinancialGoalActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := h.app.Store.FinancialGoals().Create(r.Context(), goal); err != nil {
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
	goal, err := h.app.Store.FinancialGoals().GetByIDForUser(r.Context(), goalID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	accountChanged := req.AccountID != nil
	if accountChanged {
		accountID := strings.TrimSpace(*req.AccountID)
		if _, err := uuid.Parse(accountID); err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "account_id must be a UUID", nil)
			return
		}
		account, err := h.app.Store.Accounts().GetByIDForUser(r.Context(), accountID, userID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		goal.AccountID = &accountID
		goal.Currency = account.Currency
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || len([]rune(name)) > 100 {
			writeError(w, http.StatusBadRequest, "validation_error", "Name must contain 1 to 100 characters", nil)
			return
		}
		goal.Name = name
	}
	if req.TargetAmount != nil {
		amount, err := money.ParseDecimalString(*req.TargetAmount)
		if err != nil || !amount.IsPositive() {
			writeError(w, http.StatusBadRequest, "validation_error", "Target amount must be positive", nil)
			return
		}
		if err := domainmoney.ValidateCurrencyScale(amount, goal.Currency); err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
			return
		}
		goal.TargetAmount = amount
	}
	if accountChanged && req.TargetAmount == nil {
		if err := domainmoney.ValidateCurrencyScale(goal.TargetAmount, goal.Currency); err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
			return
		}
	}
	if req.TargetDate != nil {
		goal.TargetDate = nil
		if strings.TrimSpace(*req.TargetDate) != "" {
			parsed, err := time.Parse(time.DateOnly, strings.TrimSpace(*req.TargetDate))
			if err != nil {
				writeError(w, http.StatusBadRequest, "validation_error", "target_date must use YYYY-MM-DD", nil)
				return
			}
			goal.TargetDate = &parsed
		}
	}
	if req.Status != nil {
		if !validFinancialGoalStatus(*req.Status) {
			writeError(w, http.StatusBadRequest, "validation_error", "invalid financial goal status", nil)
			return
		}
		goal.Status = *req.Status
	}
	goal.UpdatedAt = time.Now().UTC()
	if err := h.app.Store.FinancialGoals().UpdateForUser(r.Context(), goal, userID); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.FinancialGoalFromModel(goal))
}

func validFinancialGoalStatus(status models.FinancialGoalStatus) bool {
	switch status {
	case models.FinancialGoalActive, models.FinancialGoalCompleted, models.FinancialGoalArchived:
		return true
	default:
		return false
	}
}
