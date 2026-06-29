package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/services"
	"github.com/sunriseex/capitalflow/pkg/money"
)

func (h *Handler) listInterestRules(w http.ResponseWriter, r *http.Request) {
	accountID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	rules, err := h.app.InterestRules.ListByAccountForUser(r.Context(), accountID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.InterestRulesFromModels(rules))
}

func (h *Handler) listUserInterestRules(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	rules, err := h.app.InterestRules.ListByUser(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.InterestRulesFromModels(rules))
}

func (h *Handler) createInterestRule(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	var req dto.CreateInterestRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	startDate, err := parseOptionalDate(req.StartDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}
	promoEndDate, err := parseOptionalDatePtr(req.PromoEndDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}
	endDate, err := parseOptionalDatePtr(req.EndDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}

	accountID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}
	rule, err := h.app.InterestRules.Create(r.Context(), &services.CreateInterestRuleRequest{
		UserID:                  userID,
		AccountID:               accountID,
		AnnualRateBps:           req.AnnualRateBps,
		PromoRateBps:            req.PromoRateBps,
		PromoEndDate:            promoEndDate,
		AccrualFrequency:        req.AccrualFrequency,
		CapitalizationFrequency: req.CapitalizationFrequency,
		DayCountConvention:      req.DayCountConvention,
		StartDate:               startDate,
		EndDate:                 endDate,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.InterestRuleFromModel(rule))
}

func (h *Handler) updateInterestRule(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	ruleID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req dto.UpdateInterestRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	var promoRate *int64
	if req.PromoRateBps.Set {
		if req.PromoRateBps.Valid {
			value := req.PromoRateBps.Value
			promoRate = &value
		}
	}
	var promoEndDate *time.Time
	if req.PromoEndDate.Set {
		if req.PromoEndDate.Valid && strings.TrimSpace(req.PromoEndDate.Value) != "" {
			date, err := parseOptionalDate(req.PromoEndDate.Value)
			if err != nil {
				writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
				return
			}
			promoEndDate = &date
		}
	}
	var startDate *time.Time
	if req.StartDate != nil {
		date, err := parseOptionalDate(*req.StartDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
			return
		}
		if !date.IsZero() {
			startDate = &date
		}
	}
	var endDate *time.Time
	if req.EndDate.Set {
		if req.EndDate.Valid && strings.TrimSpace(req.EndDate.Value) != "" {
			date, err := parseOptionalDate(req.EndDate.Value)
			if err != nil {
				writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
				return
			}
			endDate = &date
		}
	}
	rule, err := h.app.InterestRules.UpdateForUser(r.Context(), &services.UpdateInterestRuleRequest{
		ID: ruleID, UserID: userID, AnnualRateBps: req.AnnualRateBps,
		PromoRateSet: req.PromoRateBps.Set, PromoRateBps: promoRate,
		PromoEndDateSet: req.PromoEndDate.Set, PromoEndDate: promoEndDate,
		AccrualFrequency: req.AccrualFrequency, CapitalizationFrequency: req.CapitalizationFrequency,
		DayCountConvention: req.DayCountConvention, IsActive: req.IsActive,
		StartDate: startDate, EndDateSet: req.EndDate.Set, EndDate: endDate,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.InterestRuleFromModel(rule))
}

func (h *Handler) accrueInterest(w http.ResponseWriter, r *http.Request) {
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}

	accountID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}
	var req dto.AccrueInterestRequest

	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	accrualDate, err := parseOptionalDate(req.Date)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}
	if !validateOptionalUUID(w, req.RuleID, "rule_id") {
		return
	}

	result, err := h.app.InterestLifecycle.Accrue(r.Context(), &services.AccrueAccountInterestRequest{
		AccountID:   accountID,
		UserID:      userID,
		RuleID:      req.RuleID,
		AccrualDate: accrualDate,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if result.Skipped {
		writeJSON(w, http.StatusOK, map[string]any{"skipped": true})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"skipped":     false,
		"transaction": dto.TransactionFromModel(result.Transaction),
		"accrual":     result.Accrual,
	})
}

func (h *Handler) recalculateInterest(w http.ResponseWriter, r *http.Request) {
	accountID, ok := routeUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req dto.RecalculateInterestRequest
	if err := decodeOptionalJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}
	if !validateOptionalUUID(w, req.RuleID, "rule_id") {
		return
	}

	fromDate, err := parseOptionalDate(req.FromDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}
	toDate, err := parseOptionalDate(req.ToDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error(), nil)
		return
	}
	userID, ok := currentUserID(w, r)
	if !ok {
		return
	}
	ruleDate := toDate
	if ruleDate.IsZero() {
		ruleDate = dateOnly(time.Now())
	}

	result, err := h.app.InterestLifecycle.Recalculate(r.Context(), &services.RecalculateAccountInterestRequest{
		AccountID: accountID,
		UserID:    userID,
		RuleID:    req.RuleID,
		RuleDate:  ruleDate,
		FromDate:  fromDate,
		ToDate:    toDate,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.RecalculateInterestResponse{
		AccountID:       result.AccountID,
		RuleID:          result.RuleID,
		FromDate:        result.FromDate,
		ToDate:          result.ToDate,
		DeletedAccruals: result.DeletedAccruals,
		CreatedAccruals: result.CreatedAccruals,
		SkippedDays:     result.SkippedDays,
		TotalAmount:     money.NewJSONDecimal(result.TotalAmount),
	})
}

func parseOptionalDatePtr(input *string) (*time.Time, error) {
	if input == nil {
		//nolint:nilnil // nil date pointer means optional date was not provided.
		return nil, nil
	}
	date, err := parseOptionalDate(*input)
	if err != nil {
		return nil, fmt.Errorf("parse optional date: %w", err)
	}
	if date.IsZero() {
		//nolint:nilnil // empty date string clears optional date.
		return nil, nil
	}
	return &date, nil
}

func dateOnly(date time.Time) time.Time {
	if date.IsZero() {
		return time.Time{}
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}
