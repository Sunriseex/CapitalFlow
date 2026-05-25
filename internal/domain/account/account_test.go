package account

import (
	"testing"

	"github.com/sunriseex/capitalflow/internal/models"
)

func TestNormalizeCurrency(t *testing.T) {
	if got := NormalizeCurrency(" rub "); got != "RUB" {
		t.Fatalf("currency = %q, want RUB", got)
	}
}

func TestValidCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		want     bool
	}{
		{name: "rub", currency: "RUB", want: true},
		{name: "usd", currency: "USD", want: true},
		{name: "krw", currency: "KRW", want: true},
		{name: "kwd", currency: "KWD", want: true},
		{name: "lowercase normalized", currency: "rub", want: true},
		{name: "obsolete rur", currency: "RUR"},
		{name: "unsupported crypto", currency: "BTC"},
		{name: "four letters", currency: "USDT"},
		{name: "digits", currency: "R1B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidCurrency(tt.currency); got != tt.want {
				t.Fatalf("ValidCurrency(%q) = %t, want %t", tt.currency, got, tt.want)
			}
		})
	}
}

func TestValidateCreate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		typ     models.AccountType
		cur     string
		wantErr bool
	}{
		{name: "valid", input: "Main", typ: models.AccountTypeCard, cur: "RUB"},
		{name: "missing name", typ: models.AccountTypeCard, cur: "RUB", wantErr: true},
		{name: "invalid type", input: "Main", typ: models.AccountType("bad"), cur: "RUB", wantErr: true},
		{name: "invalid currency", input: "Main", typ: models.AccountTypeCard, cur: "BTC", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreate(tt.input, tt.typ, tt.cur)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
