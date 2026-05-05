package models

import "time"

type InterestAccrual struct {
	ID            string    `json:"id"`
	AccountID     string    `json:"account_id"`
	RuleID        string    `json:"rule_id"`
	TransactionID string    `json:"transaction_id"`
	AccrualDate   time.Time `json:"accrual_date"`
	AmountMinor   int64     `json:"amount_minor"`
	BalanceMinor  int64     `json:"balance_minor"`
	AnnualRateBps int64     `json:"annual_rate_bps"`
	CreatedAt     time.Time `json:"created_at"`
}
