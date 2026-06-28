package dto

import (
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
)

type CreateCategoryLimitRequest struct {
	CategoryID string `json:"category_id"`
	Amount     string `json:"amount"`
	Currency   string `json:"currency"`
	IsActive   *bool  `json:"is_active"`
}

type UpdateCategoryLimitRequest struct {
	CategoryID *string `json:"category_id"`
	Amount     *string `json:"amount"`
	Currency   *string `json:"currency"`
	IsActive   *bool   `json:"is_active"`
}

type CategoryLimitResponse struct {
	ID         string    `json:"id"`
	CategoryID string    `json:"category_id"`
	Amount     string    `json:"amount"`
	Currency   string    `json:"currency"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func CategoryLimitFromModel(limit *models.CategoryLimit) CategoryLimitResponse {
	return CategoryLimitResponse{
		ID:         limit.ID,
		CategoryID: limit.CategoryID,
		Amount:     limit.Amount.String(),
		Currency:   limit.Currency,
		IsActive:   limit.IsActive,
		CreatedAt:  limit.CreatedAt,
		UpdatedAt:  limit.UpdatedAt,
	}
}

func CategoryLimitsFromModels(limits []models.CategoryLimit) []CategoryLimitResponse {
	response := make([]CategoryLimitResponse, 0, len(limits))
	for i := range limits {
		response = append(response, CategoryLimitFromModel(&limits[i]))
	}
	return response
}
