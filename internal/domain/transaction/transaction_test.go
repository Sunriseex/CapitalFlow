package transaction

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
)

func TestValidateCreate(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		input   CreateValidation
		wantErr bool
	}{
		{
			name: "valid",
			input: CreateValidation{
				AccountID:  "account-1",
				Type:       models.TransactionTypeIncome,
				Amount:     decimal.RequireFromString("1.23"),
				Currency:   "RUB",
				OccurredAt: now,
				Now:        now,
			},
		},
		{
			name: "zero amount",
			input: CreateValidation{
				AccountID:  "account-1",
				Type:       models.TransactionTypeIncome,
				Amount:     decimal.Zero,
				OccurredAt: now,
				Now:        now,
			},
			wantErr: true,
		},
		{
			name: "negative income",
			input: CreateValidation{
				AccountID:  "account-1",
				Type:       models.TransactionTypeIncome,
				Amount:     decimal.NewFromInt(-1),
				OccurredAt: now,
				Now:        now,
			},
			wantErr: true,
		},
		{
			name: "future date",
			input: CreateValidation{
				AccountID:  "account-1",
				Type:       models.TransactionTypeIncome,
				Amount:     decimal.NewFromInt(1),
				OccurredAt: now.Add(time.Second),
				Now:        now,
			},
			wantErr: true,
		},
		{
			name: "before account open",
			input: CreateValidation{
				AccountID:     "account-1",
				Type:          models.TransactionTypeIncome,
				Amount:        decimal.NewFromInt(1),
				OccurredAt:    now.Add(-time.Hour),
				AccountOpened: now,
				Now:           now,
			},
			wantErr: true,
		},
		{
			name: "transfer generation can allow future",
			input: CreateValidation{
				AccountID:   "account-1",
				Type:        models.TransactionTypeInterestIncome,
				Amount:      decimal.NewFromInt(1),
				OccurredAt:  now.Add(time.Second),
				Now:         now,
				AllowFuture: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreate(&tt.input)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
