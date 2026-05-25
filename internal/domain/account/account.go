package account

import (
	"fmt"
	"strings"

	"github.com/sunriseex/capitalflow/internal/models"
)

var supportedCurrencies = map[string]struct{}{
	"AED":  {},
	"ARS":  {},
	"AUD":  {},
	"BRL":  {},
	"CAD":  {},
	"CHF":  {},
	"CLF":  {},
	"CNY":  {},
	"EUR":  {},
	"GBP":  {},
	"HKD":  {},
	"INR":  {},
	"JPY":  {},
	"KRW":  {},
	"KWD":  {},
	"MXN":  {},
	"RUB":  {},
	"SGD":  {},
	"TRY":  {},
	"USD":  {},
	"USDT": {},
}

func NormalizeCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}

func ValidCurrency(currency string) bool {
	currency = NormalizeCurrency(currency)
	if currency == "" {
		return false
	}
	_, ok := supportedCurrencies[currency]
	return ok
}

func ValidAccountType(accountType models.AccountType) bool {
	switch accountType {
	case models.AccountTypeCash,
		models.AccountTypeCard,
		models.AccountTypeSavings,
		models.AccountTypeTermDeposit,
		models.AccountTypeBroker,
		models.AccountTypeOther:
		return true
	default:
		return false
	}
}

func ValidateCreate(name string, accountType models.AccountType, currency string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("account name is required")
	}
	if !ValidAccountType(accountType) {
		return fmt.Errorf("invalid account type: %s", accountType)
	}
	if !ValidCurrency(currency) {
		return fmt.Errorf("invalid currency: %s", NormalizeCurrency(currency))
	}
	return nil
}
