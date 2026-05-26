package money

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestValidateCurrencyScale(t *testing.T) {
	tests := []struct {
		name     string
		amount   string
		currency string
		wantErr  bool
	}{
		{name: "rub minor units", amount: "1.23", currency: "rub"},
		{name: "rub sub minor rejected", amount: "1.234", currency: "RUB", wantErr: true},
		{name: "empty currency defaults to rub", amount: "1.23"},
		{name: "jpy rejects fractional unit", amount: "1.1", currency: "JPY", wantErr: true},
		{name: "usdt allows six decimals", amount: "1.123456", currency: "USDT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCurrencyScale(decimal.RequireFromString(tt.amount), tt.currency)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
