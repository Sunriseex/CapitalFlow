package dto

import "github.com/sunriseex/capitalflow/pkg/money"

type CreateTransferRequest struct {
	FromAccountID string            `json:"from_account_id"`
	ToAccountID   string            `json:"to_account_id"`
	Amount        money.JSONDecimal `json:"amount"`
	FeeAmount     money.JSONDecimal `json:"fee_amount"`
	FeeCurrency   string            `json:"fee_currency"`
	Description   string            `json:"description"`
}

type TransferResponse struct {
	Out          TransactionResponse  `json:"out"`
	In           TransactionResponse  `json:"in"`
	Fee          *TransactionResponse `json:"fee,omitempty"`
	ExchangeRate string               `json:"exchange_rate"`
}

type TransferEventResponse struct {
	ID                   string            `json:"id"`
	UserID               string            `json:"user_id"`
	FromAccountID        string            `json:"from_account_id"`
	ToAccountID          string            `json:"to_account_id"`
	FromTransactionID    string            `json:"from_transaction_id"`
	ToTransactionID      string            `json:"to_transaction_id"`
	FeeTransactionID     *string           `json:"fee_transaction_id,omitempty"`
	FromAmount           money.JSONDecimal `json:"from_amount"`
	ToAmount             money.JSONDecimal `json:"to_amount"`
	FromCurrency         string            `json:"from_currency"`
	ToCurrency           string            `json:"to_currency"`
	ExchangeRate         string            `json:"exchange_rate"`
	ExchangeRateScale    int               `json:"exchange_rate_scale"`
	ExchangeRateProvider string            `json:"exchange_rate_provider"`
	ExchangeRateDate     string            `json:"exchange_rate_date"`
	FeeAmount            money.JSONDecimal `json:"fee_amount"`
	FeeCurrency          *string           `json:"fee_currency,omitempty"`
	Status               string            `json:"status"`
	CreatedAt            string            `json:"created_at"`
	UpdatedAt            string            `json:"updated_at"`
}
