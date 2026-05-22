package money

import (
	"fmt"
	"math"
	"strings"

	"github.com/shopspring/decimal"
)

const (
	StoragePrecision = 38
	StorageScale     = 18
)

var (
	kopecksPerRub = decimal.NewFromInt(100)
	maxRubAmount  = decimal.NewFromInt(1_000_000)
	maxRate       = decimal.NewFromInt(100)
	maxInt64      = decimal.NewFromInt(math.MaxInt64)
	minInt64      = decimal.NewFromInt(math.MinInt64)
)

func ParseRUB(input string) (decimal.Decimal, error) {
	value := strings.TrimSpace(strings.ReplaceAll(input, ",", "."))
	if value == "" {
		return decimal.Zero, fmt.Errorf("amount is empty")
	}

	amount, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse amount: %w", err)
	}

	return amount.Round(2), nil
}

func ParsePositiveRUB(input string) (decimal.Decimal, error) {
	amount, err := ParseRUB(input)
	if err != nil {
		return decimal.Zero, err
	}
	if !amount.IsPositive() {
		return decimal.Zero, fmt.Errorf("amount must be positive")
	}
	if amount.GreaterThan(maxRubAmount) {
		return decimal.Zero, fmt.Errorf("amount exceeds %s", maxRubAmount.String())
	}
	return amount, nil
}

func LegacyKopecksToDecimal(kopecks int64) decimal.Decimal {
	return decimal.NewFromInt(kopecks).Div(kopecksPerRub).Round(2)
}

func MinorUnitsToDecimal(amountMinor int64) decimal.Decimal {
	return decimal.NewFromInt(amountMinor)
}

func DecimalToMinorUnits(amount decimal.Decimal) (int64, error) {
	if !amount.IsInteger() {
		return 0, fmt.Errorf("amount cannot be represented as integer minor units: %s", amount.String())
	}
	if amount.GreaterThan(maxInt64) || amount.LessThan(minInt64) {
		return 0, fmt.Errorf("amount exceeds int64 minor-unit compatibility range: %s", amount.String())
	}
	return amount.IntPart(), nil
}

func DecimalToLegacyKopecks(amount decimal.Decimal) (int64, error) {
	kopecks := amount.Round(2).Mul(kopecksPerRub)
	if !kopecks.IsInteger() {
		return 0, fmt.Errorf("amount cannot be represented as kopecks: %s", amount.String())
	}

	return kopecks.IntPart(), nil
}

func FormatRUB(amount decimal.Decimal) string {
	return amount.Round(2).StringFixed(2)
}

func FormatLegacyKopecks(kopecks int64) string {
	return FormatRUB(LegacyKopecksToDecimal(kopecks))
}

func ParseRate(input string) (decimal.Decimal, error) {
	value := strings.TrimSpace(strings.ReplaceAll(input, ",", "."))
	if value == "" {
		return decimal.Zero, fmt.Errorf("rate is empty")
	}

	rate, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse rate: %w", err)
	}
	if !rate.IsPositive() {
		return decimal.Zero, fmt.Errorf("rate must be positive")
	}
	if rate.GreaterThan(maxRate) {
		return decimal.Zero, fmt.Errorf("rate exceeds 100")
	}
	return rate, nil
}

func RateToBps(rate decimal.Decimal) int64 {
	return rate.Mul(decimal.NewFromInt(100)).Round(0).IntPart()
}

func BpsToRate(bps int64) decimal.Decimal {
	return decimal.NewFromInt(bps).Div(decimal.NewFromInt(100))
}
