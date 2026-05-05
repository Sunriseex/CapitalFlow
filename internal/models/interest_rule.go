package models

import "time"

type AccrualFrequency string
type CapitalizationFrequency string
type DayCountConvention string

const (
	AccrualFrequencyDaily     AccrualFrequency = "daily"
	AccrualFrequencyMonthly   AccrualFrequency = "monthly"
	AccrualFrequencyEndOfTerm AccrualFrequency = "end_of_term"
)

const (
	CapitalizationFrequencyDaily     CapitalizationFrequency = "daily"
	CapitalizationFrequencyMonthly   CapitalizationFrequency = "monthly"
	CapitalizationFrequencyEndOfTerm CapitalizationFrequency = "end_of_term"
	CapitalizationFrequencyNone      CapitalizationFrequency = "none"
)

const (
	DayCountConventionActual365    DayCountConvention = "actual_365"
	DayCountConventionActual366    DayCountConvention = "actual_366"
	DayCountConventionActualActual DayCountConvention = "actual_actual"
)

type InterestRule struct {
	ID                      string                  `json:"id"`
	AccountID               string                  `json:"account_id"`
	AnnualRateBps           int64                   `json:"annual_rate_bps"`
	PromoRateBps            *int64                  `json:"promo_rate_bps,omitempty"`
	PromoEndDate            *time.Time              `json:"promo_end_date,omitempty"`
	AccrualFrequency        AccrualFrequency        `json:"accrual_frequency"`
	CapitalizationFrequency CapitalizationFrequency `json:"capitalization_frequency"`
	DayCountConvention      DayCountConvention      `json:"day_count_convention"`
	IsActive                bool                    `json:"is_active"`
	StartDate               time.Time               `json:"start_date"`
	EndDate                 *time.Time              `json:"end_date,omitempty"`
}
