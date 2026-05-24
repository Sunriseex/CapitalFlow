package dto

import "github.com/sunriseex/capitalflow/pkg/money"

type CreateTransferRequest struct {
	FromAccountID string            `json:"from_account_id"`
	ToAccountID   string            `json:"to_account_id"`
	Amount        money.JSONDecimal `json:"amount"`
	Description   string            `json:"description"`
}

type TransferResponse struct {
	Out          TransactionResponse `json:"out"`
	In           TransactionResponse `json:"in"`
	ExchangeRate string              `json:"exchange_rate"`
}
