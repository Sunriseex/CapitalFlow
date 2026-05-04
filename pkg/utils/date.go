package utils

import (
	"fmt"
	"log/slog"
	"time"
)

func DaysUntil(dueDate string) int {
	if dueDate == "" {
		return 999
	}

	today := time.Now()
	due, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		slog.Debug("Ошибка парсинга даты", "date", dueDate, "error", err)
		return 999
	}
	return int(due.Sub(today).Hours() / 24)
}
func ParseDate(dateStr string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		slog.Debug("Ошибка парсинга даты", "date_string", dateStr, "error", err)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date: %w", err)
	}
	return date, nil
}
func FormatDate(date time.Time) string {
	return date.Format("2006-01-02")
}
func Today() string {
	return time.Now().Format("2006-01-02")
}
func AddDays(dateStr string, days int) (string, error) {
	date, err := ParseDate(dateStr)
	if err != nil {
		slog.Error("Ошибка добавления дней к дате", "date", dateStr, "days", days, "error", err)
		return "", fmt.Errorf("parse date for add days: %w", err)
	}
	return FormatDate(date.AddDate(0, 0, days)), nil
}

func AddMonths(dateStr string, months int) (string, error) {
	date, err := ParseDate(dateStr)
	if err != nil {
		slog.Error("Ошибка добавления месяцев к дате", "date", dateStr, "months", months, "error", err)
		return "", fmt.Errorf("parse date for add months: %w", err)
	}
	return FormatDate(date.AddDate(0, months, 0)), nil
}
func AddYears(dateStr string, years int) (string, error) {
	date, err := ParseDate(dateStr)
	if err != nil {
		slog.Error("Ошибка добавления лет к дате", "date", dateStr, "years", years, "error", err)
		return "", fmt.Errorf("parse date for add years: %w", err)
	}
	return FormatDate(date.AddDate(years, 0, 0)), nil
}
func DaysBetween(dateStr1, dateStr2 string) int {
	date1, err1 := time.Parse("2006-01-02", dateStr1)
	date2, err2 := time.Parse("2006-01-02", dateStr2)
	if err1 != nil || err2 != nil {
		slog.Debug("Ошибка вычисления дней между датами",
			"date1", dateStr1, "date2", dateStr2,
			"error1", err1, "error2", err2)
		return 0
	}
	date1 = time.Date(date1.Year(), date1.Month(), date1.Day(), 0, 0, 0, 0, time.UTC)
	date2 = time.Date(date2.Year(), date2.Month(), date2.Day(), 0, 0, 0, 0, time.UTC)
	hours := date2.Sub(date1).Hours()
	return int(hours / 24)
}
