package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	domaininterest "github.com/sunriseex/capitalflow/internal/domain/interest"
	"github.com/sunriseex/capitalflow/internal/models"
)

type interestRuleUserLister interface {
	ListByUser(ctx context.Context, userID string) ([]models.InterestRule, error)
}

type UpdateInterestRuleRequest struct {
	ID                      string
	UserID                  string
	AnnualRateBps           *int64
	PromoRateSet            bool
	PromoRateBps            *int64
	PromoEndDateSet         bool
	PromoEndDate            *time.Time
	AccrualFrequency        *models.AccrualFrequency
	CapitalizationFrequency *models.CapitalizationFrequency
	DayCountConvention      *models.DayCountConvention
	IsActive                *bool
	StartDate               *time.Time
	EndDateSet              bool
	EndDate                 *time.Time
}

func (s *InterestRuleService) ListByAccountForUser(ctx context.Context, accountID, userID string) ([]models.InterestRule, error) {
	if s == nil || s.rules == nil || s.accounts == nil {
		return nil, fmt.Errorf("interest rule repositories are required")
	}
	accountID = strings.TrimSpace(accountID)
	if _, err := s.accounts.GetByIDForUser(ctx, accountID, strings.TrimSpace(userID)); err != nil {
		return nil, fmt.Errorf("get interest rule account: %w", err)
	}
	rules, err := s.rules.ListByAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("list interest rules: %w", err)
	}
	return rules, nil
}

func (s *InterestRuleService) ListByUser(ctx context.Context, userID string) ([]models.InterestRule, error) {
	if s == nil || s.rules == nil {
		return nil, fmt.Errorf("interest rule repository is required")
	}
	lister, ok := s.rules.(interestRuleUserLister)
	if !ok {
		return nil, fmt.Errorf("interest rule listing by user is not supported")
	}
	rules, err := lister.ListByUser(ctx, strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("list user interest rules: %w", err)
	}
	return rules, nil
}

func (s *InterestRuleService) UpdateForUser(ctx context.Context, req *UpdateInterestRuleRequest) (*models.InterestRule, error) {
	if s == nil || s.rules == nil || s.accounts == nil {
		return nil, fmt.Errorf("interest rule repositories are required")
	}
	if req == nil {
		return nil, validationError("update interest rule request is required")
	}
	rule, err := s.rules.GetByID(ctx, strings.TrimSpace(req.ID))
	if err != nil {
		return nil, fmt.Errorf("get interest rule: %w", err)
	}
	if _, err := s.accounts.GetByIDForUser(ctx, rule.AccountID, strings.TrimSpace(req.UserID)); err != nil {
		return nil, fmt.Errorf("get interest rule account: %w", err)
	}
	if req.AnnualRateBps != nil {
		rule.AnnualRateBps = *req.AnnualRateBps
	}
	if req.PromoRateSet {
		rule.PromoRateBps = req.PromoRateBps
		if req.PromoRateBps == nil {
			rule.PromoEndDate = nil
		}
	}
	if req.PromoEndDateSet {
		rule.PromoEndDate = req.PromoEndDate
		if req.PromoEndDate == nil {
			rule.PromoRateBps = nil
		}
	}
	if req.AccrualFrequency != nil {
		rule.AccrualFrequency = *req.AccrualFrequency
	}
	if req.CapitalizationFrequency != nil {
		rule.CapitalizationFrequency = *req.CapitalizationFrequency
	}
	if req.DayCountConvention != nil {
		rule.DayCountConvention = *req.DayCountConvention
	}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}
	if req.StartDate != nil {
		rule.StartDate = dateOnly(*req.StartDate)
	}
	if req.EndDateSet {
		rule.EndDate = req.EndDate
	}
	if err := validateInterestRuleConfiguration(rule); err != nil {
		return nil, err
	}
	if err := s.rules.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("update interest rule: %w", err)
	}
	return rule, nil
}

func validateInterestRuleConfiguration(rule *models.InterestRule) error {
	if rule.AnnualRateBps <= 0 {
		return validationError("annual rate must be positive")
	}
	if rule.PromoRateBps != nil && *rule.PromoRateBps <= 0 {
		return validationError("promo rate must be positive")
	}
	if (rule.PromoRateBps == nil) != (rule.PromoEndDate == nil) {
		return validationError("promo rate and promo end date must be set together")
	}
	if rule.AccrualFrequency == "" {
		rule.AccrualFrequency = models.AccrualFrequencyDaily
	}
	if rule.CapitalizationFrequency == "" {
		rule.CapitalizationFrequency = models.CapitalizationFrequencyNone
	}
	if rule.DayCountConvention == "" {
		rule.DayCountConvention = models.DayCountConventionActual365
	}
	if err := domaininterest.ValidateFrequencies(rule.AccrualFrequency, rule.CapitalizationFrequency, rule.DayCountConvention); err != nil {
		return validationError(err.Error())
	}
	startDate := dateOnly(rule.StartDate)
	if startDate.IsZero() {
		return validationError("start date is required")
	}
	if rule.EndDate != nil && dateOnly(*rule.EndDate).Before(startDate) {
		return validationError("end date must be on or after start date")
	}
	if rule.PromoEndDate != nil && dateOnly(*rule.PromoEndDate).Before(startDate) {
		return validationError("promo end date must be on or after start date")
	}
	return nil
}
