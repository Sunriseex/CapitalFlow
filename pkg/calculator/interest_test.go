package calculator

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
)

func TestCalculateIncome(t *testing.T) {
	promoRate := 20.0
	tests := []struct {
		name    string
		deposit models.Deposit
		days    int
		want    string
	}{
		{
			name:    "normal rate",
			deposit: models.Deposit{Amount: 120000, InterestRate: 10, Capitalization: "end"},
			days:    365,
			want:    "120.00",
		},
		{
			name:    "active promo rate",
			deposit: models.Deposit{Amount: 120000, InterestRate: 10, PromoRate: &promoRate, PromoEndDate: "2099-01-01", Capitalization: "end"},
			days:    365,
			want:    "240.00",
		},
		{
			name:    "term promo ended before maturity",
			deposit: models.Deposit{Type: "term", Amount: 120000, InterestRate: 10, PromoRate: &promoRate, PromoEndDate: "2024-01-31", StartDate: "2024-01-01", EndDate: "2025-01-01", Capitalization: "end"},
			days:    366,
			want:    "120.33",
		},
		{
			name:    "zero amount",
			deposit: models.Deposit{Amount: 0, InterestRate: 10, Capitalization: "end"},
			days:    365,
			want:    "0.00",
		},
		{
			name:    "negative amount",
			deposit: models.Deposit{Amount: -10000, InterestRate: 10, Capitalization: "end"},
			days:    365,
			want:    "0.00",
		},
		{
			name:    "leap year period uses actual days over 365 convention",
			deposit: models.Deposit{Amount: 36500, InterestRate: 10, Capitalization: "end"},
			days:    366,
			want:    "36.60",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateIncome(tt.deposit, tt.days).Round(2)
			want := decimal.RequireFromString(tt.want)
			if !got.Equal(want) {
				t.Fatalf("got %s, want %s", got.StringFixed(2), want.StringFixed(2))
			}
		})
	}
}
