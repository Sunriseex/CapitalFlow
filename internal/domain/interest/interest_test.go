package interest

import (
	"testing"

	"github.com/sunriseex/capitalflow/internal/models"
)

func TestValidateFrequencies(t *testing.T) {
	if err := ValidateFrequencies(models.AccrualFrequencyDaily, models.CapitalizationFrequencyMonthly, models.DayCountConventionActual365); err != nil {
		t.Fatalf("valid frequencies rejected: %v", err)
	}
	if err := ValidateFrequencies("weekly", models.CapitalizationFrequencyMonthly, models.DayCountConventionActual365); err == nil {
		t.Fatal("expected invalid accrual frequency error")
	}
	if err := ValidateFrequencies(models.AccrualFrequencyDaily, "weekly", models.DayCountConventionActual365); err == nil {
		t.Fatal("expected invalid capitalization frequency error")
	}
	if err := ValidateFrequencies(models.AccrualFrequencyDaily, models.CapitalizationFrequencyMonthly, "30_360"); err == nil {
		t.Fatal("expected invalid day count convention error")
	}
}
