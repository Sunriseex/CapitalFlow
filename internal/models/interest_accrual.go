package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type InterestAccrual struct {
	ID            string          `json:"id"`
	AccountID     string          `json:"account_id"`
	RuleID        string          `json:"rule_id"`
	TransactionID string          `json:"transaction_id"`
	AccrualDate   time.Time       `json:"accrual_date"`
	Amount        decimal.Decimal `json:"amount"`
	Balance       decimal.Decimal `json:"balance"`
	AnnualRateBps int64           `json:"annual_rate_bps"`
	CreatedAt     time.Time       `json:"created_at"`
}
