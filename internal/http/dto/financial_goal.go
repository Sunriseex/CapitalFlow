package dto

import (
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
)

type CreateFinancialGoalRequest struct {
	AccountID    string `json:"account_id"`
	Name         string `json:"name"`
	TargetAmount string `json:"target_amount"`
	TargetDate   string `json:"target_date"`
}

type UpdateFinancialGoalRequest struct {
	AccountID    *string                     `json:"account_id"`
	Name         *string                     `json:"name"`
	TargetAmount *string                     `json:"target_amount"`
	TargetDate   *string                     `json:"target_date"`
	Status       *models.FinancialGoalStatus `json:"status"`
}

type FinancialGoalResponse struct {
	ID           string                     `json:"id"`
	AccountID    *string                    `json:"account_id,omitempty"`
	Name         string                     `json:"name"`
	TargetAmount string                     `json:"target_amount"`
	Currency     string                     `json:"currency"`
	TargetDate   *string                    `json:"target_date,omitempty"`
	Status       models.FinancialGoalStatus `json:"status"`
	CreatedAt    time.Time                  `json:"created_at"`
	UpdatedAt    time.Time                  `json:"updated_at"`
}

func FinancialGoalFromModel(goal *models.FinancialGoal) FinancialGoalResponse {
	var targetDate *string
	if goal.TargetDate != nil {
		formatted := goal.TargetDate.Format(time.DateOnly)
		targetDate = &formatted
	}
	return FinancialGoalResponse{
		ID:           goal.ID,
		AccountID:    goal.AccountID,
		Name:         goal.Name,
		TargetAmount: goal.TargetAmount.String(),
		Currency:     goal.Currency,
		TargetDate:   targetDate,
		Status:       goal.Status,
		CreatedAt:    goal.CreatedAt,
		UpdatedAt:    goal.UpdatedAt,
	}
}

func FinancialGoalsFromModels(goals []models.FinancialGoal) []FinancialGoalResponse {
	response := make([]FinancialGoalResponse, 0, len(goals))
	for i := range goals {
		response = append(response, FinancialGoalFromModel(&goals[i]))
	}
	return response
}
