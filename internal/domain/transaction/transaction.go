package transaction

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/pkg/money"
)

var MaxAmount = decimal.RequireFromString("1000000000000")

type CreateValidation struct {
	AccountID     string
	Type          models.TransactionType
	Amount        decimal.Decimal
	Currency      string
	OccurredAt    time.Time
	AccountOpened time.Time
	Now           time.Time
	AllowFuture   bool
}

func ValidateCreate(input *CreateValidation) error {
	if input == nil {
		return fmt.Errorf("transaction validation input is required")
	}
	if strings.TrimSpace(input.AccountID) == "" {
		return fmt.Errorf("account id is required")
	}
	if !ValidType(input.Type) {
		return fmt.Errorf("invalid transaction type: %s", input.Type)
	}
	if input.Amount.IsZero() {
		return fmt.Errorf("amount must be non-zero")
	}
	if input.Amount.LessThan(MaxAmount.Neg()) || input.Amount.GreaterThan(MaxAmount) {
		return fmt.Errorf("amount must be between %s and %s", MaxAmount.Neg(), MaxAmount)
	}
	if rounded := money.RoundForCurrency(input.Amount, input.Currency); !input.Amount.Equal(rounded) {
		currency := strings.ToUpper(strings.TrimSpace(input.Currency))
		if currency == "" {
			currency = "RUB"
		}
		return fmt.Errorf("amount scale exceeds %s minor units", currency)
	}
	if input.Type != models.TransactionTypeAdjustment && input.Amount.IsNegative() {
		return fmt.Errorf("amount must be positive for %s transactions", input.Type)
	}
	if !input.AllowFuture {
		now := input.Now
		if now.IsZero() {
			now = time.Now()
		}
		if input.OccurredAt.After(now) {
			return fmt.Errorf("transaction date must not be in the future")
		}
	}
	if !input.AccountOpened.IsZero() && input.OccurredAt.Before(input.AccountOpened) {
		return fmt.Errorf("transaction date must be on or after account opened date")
	}
	return nil
}

func ValidType(transactionType models.TransactionType) bool {
	switch transactionType {
	case models.TransactionTypeInitialBalance,
		models.TransactionTypeIncome,
		models.TransactionTypeExpense,
		models.TransactionTypeTransferIn,
		models.TransactionTypeTransferOut,
		models.TransactionTypeInterestIncome,
		models.TransactionTypeAdjustment:
		return true
	default:
		return false
	}
}
