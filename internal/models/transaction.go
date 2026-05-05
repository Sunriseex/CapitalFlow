package models

import "time"

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
	Type             TransactionType `json:"type"`
	AmountMinor      int64           `json:"amount_minor"`
	CategoryID       *string         `json:"category_id,omitempty"`
	Description      string          `json:"description,omitempty"`
	OccurredAt       time.Time       `json:"occurred_at"`
	CreatedAt        time.Time       `json:"created_at"`
}
