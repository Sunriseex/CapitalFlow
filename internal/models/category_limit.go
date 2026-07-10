package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type CategoryLimit struct {
	ID          string          `json:"id"`
	OwnerUserID string          `json:"owner_user_id"`
	CategoryID  string          `json:"category_id"`
	Amount      decimal.Decimal `json:"amount"`
	Currency    string          `json:"currency"`
	IsActive    bool            `json:"is_active"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Version     int64           `json:"version"`
}
