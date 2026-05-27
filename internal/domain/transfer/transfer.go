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
	FeeAmount      decimal.Decimal
	FeeCurrency    string
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
	if input.FeeAmount.IsNegative() {
		return fmt.Errorf("transfer fee must not be negative")
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
	if input.FeeAmount.IsPositive() {
		feeCurrency := strings.TrimSpace(input.FeeCurrency)
		if feeCurrency == "" {
			feeCurrency = input.FromCurrency
		}
		if rounded := money.RoundForCurrency(input.FeeAmount, feeCurrency); !input.FeeAmount.Equal(rounded) {
			if feeCurrency == "" {
				feeCurrency = "RUB"
			}
			return fmt.Errorf("transfer fee scale exceeds %s minor units", feeCurrency)
		}
	}
	return nil
}
