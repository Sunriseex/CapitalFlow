package money

import (
	"encoding/json"
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

type JSONDecimal struct {
	decimal.Decimal
}

func NewJSONDecimal(amount decimal.Decimal) JSONDecimal {
	return JSONDecimal{Decimal: amount}
}

func (d JSONDecimal) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(d.String())
	if err != nil {
		return nil, fmt.Errorf("marshal money decimal: %w", err)
	}
	return data, nil
}

func (d *JSONDecimal) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("money value must be a decimal string")
	}
	amount, err := ParseDecimalString(raw)
	if err != nil {
		return err
	}
	d.Decimal = amount
	return nil
}

func ParseDecimalString(input string) (decimal.Decimal, error) {
	value := strings.TrimSpace(strings.ReplaceAll(input, ",", "."))
	if value == "" {
		return decimal.Zero, fmt.Errorf("amount is empty")
	}
	amount, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse amount: %w", err)
	}
	if err := ValidateStorageDecimal(amount); err != nil {
		return decimal.Zero, err
	}
	return amount, nil
}

func ValidateStorageDecimal(amount decimal.Decimal) error {
	if -amount.Exponent() > StorageScale {
		return fmt.Errorf("amount scale exceeds %d: %s", StorageScale, amount.String())
	}

	intDigits := len(amount.Abs().Truncate(0).String())
	if amount.Abs().LessThan(decimal.NewFromInt(1)) {
		intDigits = 0
	}
	maxIntDigits := StoragePrecision - StorageScale
	if intDigits > maxIntDigits {
		return fmt.Errorf("amount precision exceeds NUMERIC(%d,%d): %s", StoragePrecision, StorageScale, amount.String())
	}
	return nil
}

func CurrencyScale(string) int32 {
	return 2
}

func RoundForCurrency(amount decimal.Decimal, currency string) decimal.Decimal {
	return amount.Round(CurrencyScale(currency))
}

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

// Deprecated: compatibility helper for legacy integer minor-unit data only.
func MinorUnitsToDecimal(amount int64) decimal.Decimal {
	return decimal.NewFromInt(amount)
}

// Deprecated: compatibility helper for legacy integer minor-unit APIs only.
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
