package services

import (
	"strings"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
)

func TestValidateInterestRuleConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*models.InterestRule)
		wantErr string
	}{
		{"defaults", func(rule *models.InterestRule) {
			rule.AccrualFrequency = ""
			rule.CapitalizationFrequency = ""
			rule.DayCountConvention = ""
		}, ""},
		{"invalid frequency", func(rule *models.InterestRule) { rule.AccrualFrequency = "weekly" }, "invalid accrual frequency"},
		{"invalid capitalization", func(rule *models.InterestRule) { rule.CapitalizationFrequency = "yearly" }, "invalid capitalization frequency"},
		{"invalid day count", func(rule *models.InterestRule) { rule.DayCountConvention = "30_360" }, "invalid day count convention"},
		{"end before start", func(rule *models.InterestRule) {
			end := rule.StartDate.AddDate(0, 0, -1)
			rule.EndDate = &end
		}, "end date must be on or after start date"},
		{"incomplete promo", func(rule *models.InterestRule) {
			rate := int64(100)
			rule.PromoRateBps = &rate
		}, "promo rate and promo end date must be set together"},
		{"negative promo", func(rule *models.InterestRule) {
			rate := int64(-1)
			end := rule.StartDate.AddDate(0, 1, 0)
			rule.PromoRateBps = &rate
			rule.PromoEndDate = &end
		}, "promo rate must be positive"},
		{"promo before start", func(rule *models.InterestRule) {
			rate := int64(100)
			end := rule.StartDate.AddDate(0, 0, -1)
			rule.PromoRateBps = &rate
			rule.PromoEndDate = &end
		}, "promo end date must be on or after start date"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := validManagementInterestRule()
			tt.mutate(rule)
			err := validateInterestRuleConfiguration(rule)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validate: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func validManagementInterestRule() *models.InterestRule {
	return &models.InterestRule{
		ID: "rule-1", AccountID: "account-1", AnnualRateBps: 1200,
		AccrualFrequency: models.AccrualFrequencyDaily, CapitalizationFrequency: models.CapitalizationFrequencyNone,
		DayCountConvention: models.DayCountConventionActual365, IsActive: true,
		StartDate: time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC),
	}
}
