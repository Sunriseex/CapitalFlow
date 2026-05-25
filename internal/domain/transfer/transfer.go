package transfer

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/pkg/money"
)

type CreateValidation struct {
	FromAccountID  string
	ToAccountID    string
	FromCurrency   string
	Amount         decimal.Decimal
	IdempotencyKey string
}

func ValidateCreate(input *CreateValidation) error {
	if input == nil {
		return fmt.Errorf("transfer validation input is required")
	}
	fromAccountID := strings.TrimSpace(input.FromAccountID)
	toAccountID := strings.TrimSpace(input.ToAccountID)
	if fromAccountID == "" {
		return fmt.Errorf("from account id is required")
	}
	if toAccountID == "" {
		return fmt.Errorf("to account id is required")
	}
	if fromAccountID == toAccountID {
		return fmt.Errorf("transfer accounts must be different")
	}
	if !input.Amount.IsPositive() {
		return fmt.Errorf("transfer amount must be positive")
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		return fmt.Errorf("idempotency key is required")
	}
	if rounded := money.RoundForCurrency(input.Amount, input.FromCurrency); !input.Amount.Equal(rounded) {
		currency := strings.TrimSpace(input.FromCurrency)
		if currency == "" {
			currency = "RUB"
		}
		return fmt.Errorf("transfer amount scale exceeds %s minor units", currency)
	}
	return nil
}
