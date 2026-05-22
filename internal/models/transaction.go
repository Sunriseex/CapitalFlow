package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type TransactionType string

const (
	TransactionTypeInitialBalance TransactionType = "initial_balance"
	TransactionTypeIncome         TransactionType = "income"
	TransactionTypeExpense        TransactionType = "expense"
	TransactionTypeTransferIn     TransactionType = "transfer_in"
	TransactionTypeTransferOut    TransactionType = "transfer_out"
	TransactionTypeInterestIncome TransactionType = "interest_income"
	TransactionTypeAdjustment     TransactionType = "adjustment"
)

type Transaction struct {
	ID               string          `json:"id"`
	AccountID        string          `json:"account_id"`
	RelatedAccountID *string         `json:"related_account_id,omitempty"`
	TransferID       *string         `json:"transfer_id,omitempty"`
	Type             TransactionType `json:"type"`
	Amount           decimal.Decimal `json:"amount"`
	CategoryID       *string         `json:"category_id,omitempty"`
	Description      string          `json:"description,omitempty"`
	OccurredAt       time.Time       `json:"occurred_at"`
	CreatedAt        time.Time       `json:"created_at"`
}

type Transfer struct {
	ID                   string          `json:"id"`
	UserID               string          `json:"user_id"`
	FromAccountID        string          `json:"from_account_id"`
	ToAccountID          string          `json:"to_account_id"`
	FromTransactionID    string          `json:"from_transaction_id"`
	ToTransactionID      string          `json:"to_transaction_id"`
	FromAmount           decimal.Decimal `json:"from_amount"`
	ToAmount             decimal.Decimal `json:"to_amount"`
	FromCurrency         string          `json:"from_currency"`
	ToCurrency           string          `json:"to_currency"`
	ExchangeRate         string          `json:"exchange_rate"`
	ExchangeRateProvider string          `json:"exchange_rate_provider"`
	ExchangeRateDate     time.Time       `json:"exchange_rate_date"`
	IdempotencyKey       string          `json:"idempotency_key,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
}
