package interest

import (
	"fmt"

	"github.com/sunriseex/capitalflow/internal/models"
)

func ValidateFrequencies(accrual models.AccrualFrequency, capitalization models.CapitalizationFrequency, dayCount models.DayCountConvention) error {
	if !ValidAccrualFrequency(accrual) {
		return fmt.Errorf("invalid accrual frequency: %s", accrual)
	}
	if !ValidCapitalizationFrequency(capitalization) {
		return fmt.Errorf("invalid capitalization frequency: %s", capitalization)
	}
	if !ValidDayCountConvention(dayCount) {
		return fmt.Errorf("invalid day count convention: %s", dayCount)
	}
	return nil
}

func ValidAccrualFrequency(frequency models.AccrualFrequency) bool {
	switch frequency {
	case models.AccrualFrequencyDaily,
		models.AccrualFrequencyMonthly,
		models.AccrualFrequencyEndOfTerm:
		return true
	default:
		return false
	}
}

func ValidCapitalizationFrequency(frequency models.CapitalizationFrequency) bool {
	switch frequency {
	case "",
		models.CapitalizationFrequencyDaily,
		models.CapitalizationFrequencyMonthly,
		models.CapitalizationFrequencyEndOfTerm,
		models.CapitalizationFrequencyNone:
		return true
	default:
		return false
	}
}

func ValidDayCountConvention(convention models.DayCountConvention) bool {
	switch convention {
	case models.DayCountConventionActual365,
		models.DayCountConventionActual366,
		models.DayCountConventionActualActual:
		return true
	default:
		return false
	}
}
