package models

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

type TransactionSource string

const (
	TransactionSourceManual                   TransactionSource = "manual"
	TransactionSourceCSVImport                TransactionSource = "csv_import"
	TransactionSourceTransfer                 TransactionSource = "transfer"
	TransactionSourceDepositInterest          TransactionSource = "deposit_interest"
	TransactionSourceSavingsAllocation        TransactionSource = "savings_allocation"
	TransactionSourceSubscription             TransactionSource = "subscription"
	TransactionSourceReconciliationAdjustment TransactionSource = "reconciliation_adjustment"
	TransactionSourceAutomationRule           TransactionSource = "automation_rule"
	TransactionSourceLLMDraft                 TransactionSource = "llm_draft"
	TransactionSourceSystem                   TransactionSource = "system"
)

func (source TransactionSource) IsValid() bool {
	switch source {
	case TransactionSourceManual,
		TransactionSourceCSVImport,
		TransactionSourceTransfer,
		TransactionSourceDepositInterest,
		TransactionSourceSavingsAllocation,
		TransactionSourceSubscription,
		TransactionSourceReconciliationAdjustment,
		TransactionSourceAutomationRule,
		TransactionSourceLLMDraft,
		TransactionSourceSystem:
		return true
	default:
		return false
	}
}

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

type TransactionStatus string

const (
	TransactionStatusPending     TransactionStatus = "pending"
	TransactionStatusConfirmed   TransactionStatus = "confirmed"
	TransactionStatusCancelled   TransactionStatus = "cancelled"
	TransactionStatusReversed    TransactionStatus = "reversed"
	TransactionStatusSoftDeleted TransactionStatus = "soft_deleted"
)

func (status TransactionStatus) IsValid() bool {
	switch status {
	case TransactionStatusPending,
		TransactionStatusConfirmed,
		TransactionStatusCancelled,
		TransactionStatusReversed,
		TransactionStatusSoftDeleted:
		return true
	default:
		return false
	}
}

func (status TransactionStatus) AffectsBalance() bool {
	return status == "" || status == TransactionStatusConfirmed || status == TransactionStatusReversed
}

type Transaction struct {
	ID               string            `json:"id"`
	AccountID        string            `json:"account_id"`
	RelatedAccountID *string           `json:"related_account_id,omitempty"`
	TransferID       *string           `json:"transfer_id,omitempty"`
	SourceType       TransactionSource `json:"source_type"`
	SourceRefID      *string           `json:"source_ref_id,omitempty"`
	SourceMetadata   json.RawMessage   `json:"source_metadata,omitempty"`
	Type             TransactionType   `json:"type"`
	Status           TransactionStatus `json:"status"`
	Amount           decimal.Decimal   `json:"amount"`
	CategoryID       *string           `json:"category_id,omitempty"`
	Description      string            `json:"description,omitempty"`
	OccurredAt       time.Time         `json:"occurred_at"`
	CreatedAt        time.Time         `json:"created_at"`
}

type Transfer struct {
	ID                   string          `json:"id"`
	UserID               string          `json:"user_id"`
	FromAccountID        string          `json:"from_account_id"`
	ToAccountID          string          `json:"to_account_id"`
	FromTransactionID    string          `json:"from_transaction_id"`
	ToTransactionID      string          `json:"to_transaction_id"`
	FeeTransactionID     *string         `json:"fee_transaction_id,omitempty"`
	FromAmount           decimal.Decimal `json:"from_amount"`
	ToAmount             decimal.Decimal `json:"to_amount"`
	FromCurrency         string          `json:"from_currency"`
	ToCurrency           string          `json:"to_currency"`
	ExchangeRate         string          `json:"exchange_rate"`
	ExchangeRateScale    int             `json:"exchange_rate_scale"`
	ExchangeRateProvider string          `json:"exchange_rate_provider"`
	ExchangeRateDate     time.Time       `json:"exchange_rate_date"`
	FeeAmount            decimal.Decimal `json:"fee_amount"`
	FeeCurrency          *string         `json:"fee_currency,omitempty"`
	Status               string          `json:"status"`
	IdempotencyKey       string          `json:"idempotency_key,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}
