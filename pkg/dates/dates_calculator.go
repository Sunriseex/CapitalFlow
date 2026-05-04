package dates

import (
	"fmt"
	"time"
)

func DaysBetween(startDate, endDate string) (int, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0, nil
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return 0, fmt.Errorf("parse end date: %w", err)
	}
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	hours := end.Sub(start).Hours()
	days := int(hours / 24)

	return days, nil
}

func DaysUntil(dateStr string) int {
	if dateStr == "" {
		return 999
	}
	today := time.Now()
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 999
	}
	return int(date.Sub(today) / 24)

}

func CalculateMaturityDate(startDate string, termMonths int) (string, error) {

	date, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return "", fmt.Errorf("parse start date: %w", err)
	}
	maturityDate := date.AddDate(0, termMonths, 0)
	return maturityDate.Format("2006-01-02"), nil

}
func CalculateTopUpEndDate(startDate string) string {
	date, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		date = time.Now()
	}
	topUpEnd := date.AddDate(0, 0, 7)
	return topUpEnd.Format("2006-01-02")
}

func IsDepositExpired(endDate string) bool {
	if endDate == "" {
		return false
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return false
	}
	return time.Now().After(end)
}

func CanBeProlonged(endDate string) bool {
	if endDate == "" {
		return false
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return false
	}
	daysUntilEnd := int(time.Until(end).Hours() / 24)
	return daysUntilEnd <= 7
}
