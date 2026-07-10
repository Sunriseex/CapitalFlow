package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type FinancialGoalStatus string

const (
	FinancialGoalActive    FinancialGoalStatus = "active"
	FinancialGoalCompleted FinancialGoalStatus = "completed"
	FinancialGoalArchived  FinancialGoalStatus = "archived"
)

type FinancialGoal struct {
	ID           string              `json:"id"`
	OwnerUserID  string              `json:"owner_user_id"`
	AccountID    *string             `json:"account_id,omitempty"`
	Name         string              `json:"name"`
	TargetAmount decimal.Decimal     `json:"target_amount"`
	Currency     string              `json:"currency"`
	TargetDate   *time.Time          `json:"target_date,omitempty"`
	Status       FinancialGoalStatus `json:"status"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	Version      int64               `json:"version"`
}
