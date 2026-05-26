package money

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	pkgmoney "github.com/sunriseex/capitalflow/pkg/money"
)

func NormalizeCurrency(currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return "RUB"
	}
	return currency
}

func ValidateCurrencyScale(amount decimal.Decimal, currency string) error {
	normalized := NormalizeCurrency(currency)
	if rounded := pkgmoney.RoundForCurrency(amount, normalized); !amount.Equal(rounded) {
		return fmt.Errorf("amount scale exceeds %s minor units", normalized)
	}
	return nil
}
