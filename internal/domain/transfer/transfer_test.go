package transfer

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestValidateCreate(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateValidation
		wantErr bool
	}{
		{
			name: "valid",
			input: CreateValidation{
				FromAccountID:  "from",
				ToAccountID:    "to",
				FromCurrency:   "RUB",
				Amount:         decimal.RequireFromString("1.23"),
				IdempotencyKey: "key",
			},
		},
		{
			name: "same account",
			input: CreateValidation{
				FromAccountID:  "same",
				ToAccountID:    "same",
				Amount:         decimal.NewFromInt(1),
				IdempotencyKey: "key",
			},
			wantErr: true,
		},
		{
			name: "zero amount",
			input: CreateValidation{
				FromAccountID:  "from",
				ToAccountID:    "to",
				Amount:         decimal.Zero,
				IdempotencyKey: "key",
			},
			wantErr: true,
		},
		{
			name: "missing idempotency key",
			input: CreateValidation{
				FromAccountID: "from",
				ToAccountID:   "to",
				Amount:        decimal.NewFromInt(1),
			},
			wantErr: true,
		},
		{
			name: "source over precision",
			input: CreateValidation{
				FromAccountID:  "from",
				ToAccountID:    "to",
				FromCurrency:   "JPY",
				Amount:         decimal.RequireFromString("0.5"),
				IdempotencyKey: "key",
			},
			wantErr: true,
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
