package dto

import (
	"encoding/json"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/pkg/money"
)

type TransactionResponse struct {
	ID               string                   `json:"id"`
	AccountID        string                   `json:"account_id"`
	RelatedAccountID *string                  `json:"related_account_id,omitempty"`
	TransferID       *string                  `json:"transfer_id,omitempty"`
	SourceType       models.TransactionSource `json:"source_type"`
	SourceRefID      *string                  `json:"source_ref_id,omitempty"`
	SourceMetadata   json.RawMessage          `json:"source_metadata"`
	Type             models.TransactionType   `json:"type"`
	Amount           money.JSONDecimal        `json:"amount"`
	CategoryID       *string                  `json:"category_id,omitempty"`
	Description      string                   `json:"description,omitempty"`
	OccurredAt       time.Time                `json:"occurred_at"`
	CreatedAt        time.Time                `json:"created_at"`
}

type CreateTransactionRequest struct {
	AccountID        string                 `json:"account_id"`
	RelatedAccountID *string                `json:"related_account_id"`
	Type             models.TransactionType `json:"type"`
	Amount           money.JSONDecimal      `json:"amount"`
	CategoryID       *string                `json:"category_id"`
	Description      string                 `json:"description"`
	OccurredAt       string                 `json:"occurred_at"`
}

func TransactionFromModel(transaction *models.Transaction) TransactionResponse {
	sourceType := transaction.SourceType
	if sourceType == "" {
		sourceType = models.TransactionSourceManual
	}
	sourceMetadata := transaction.SourceMetadata
	if len(sourceMetadata) == 0 {
		sourceMetadata = json.RawMessage(`{}`)
	}
	return TransactionResponse{
		ID:               transaction.ID,
		AccountID:        transaction.AccountID,
		RelatedAccountID: transaction.RelatedAccountID,
		TransferID:       transaction.TransferID,
		SourceType:       sourceType,
		SourceRefID:      transaction.SourceRefID,
		SourceMetadata:   sourceMetadata,
		Type:             transaction.Type,
		Amount:           money.NewJSONDecimal(transaction.Amount),
		CategoryID:       transaction.CategoryID,
		Description:      transaction.Description,
		OccurredAt:       transaction.OccurredAt,
		CreatedAt:        transaction.CreatedAt,
	}
}

func TransactionsFromModels(transactions []models.Transaction) []TransactionResponse {
	response := make([]TransactionResponse, 0, len(transactions))
	for i := range transactions {
		response = append(response, TransactionFromModel(&transactions[i]))
	}
	return response
}
