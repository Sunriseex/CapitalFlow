package services

import (
	"fmt"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/pkg/calculator"
	"github.com/sunriseex/capitalflow/pkg/dates"
)

func CheckPromoStatus(deposit models.Deposit) (active bool, daysRemaining int) {
	return calculator.CheckPromoStatus(deposit)
}

func CalculateMaturityDate(startDate string, termMonths int) (string, error) {
	maturityDate, err := dates.CalculateMaturityDate(startDate, termMonths)
	if err != nil {
		return "", fmt.Errorf("calculate maturity date: %w", err)
	}
	return maturityDate, nil
}

func CalculateTopUpEndDate(startDate string) string {
	return dates.CalculateTopUpEndDate(startDate)
}

func IsDepositExpired(deposit models.Deposit) bool {
	return dates.IsDepositExpired(deposit.EndDate)
}

func CanBeProlonged(deposit models.Deposit) bool {
	return dates.CanBeProlonged(deposit.EndDate)
}

func DaysUntil(dateStr string) int {
	return dates.DaysUntil(dateStr)
}
