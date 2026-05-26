package transaction

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	domainmoney "github.com/sunriseex/capitalflow/internal/domain/money"
	"github.com/sunriseex/capitalflow/internal/models"
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
	AllowTransfer bool
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
	if IsTransferType(input.Type) && !input.AllowTransfer {
		return fmt.Errorf("transfer transactions must be created through the transfer flow")
	}
	if input.Amount.IsZero() {
		return fmt.Errorf("amount must be non-zero")
	}
	if input.Amount.LessThan(MaxAmount.Neg()) || input.Amount.GreaterThan(MaxAmount) {
		return fmt.Errorf("amount must be between %s and %s", MaxAmount.Neg(), MaxAmount)
	}
	if err := domainmoney.ValidateCurrencyScale(input.Amount, input.Currency); err != nil {
		return err
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
	if !input.AccountOpened.IsZero() && dateOnly(input.OccurredAt).Before(dateOnly(input.AccountOpened)) {
		return fmt.Errorf("transaction date must be on or after account opened date")
	}
	return nil
}

func IsTransferType(transactionType models.TransactionType) bool {
	return transactionType == models.TransactionTypeTransferIn ||
		transactionType == models.TransactionTypeTransferOut
}

func dateOnly(date time.Time) time.Time {
	if date.IsZero() {
		return time.Time{}
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
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
