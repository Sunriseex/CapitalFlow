package money

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
)

func TestParseRUB(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "whole rubles", input: "100", want: "100.00"},
		{name: "kopecks", input: "100.50", want: "100.50"},
		{name: "comma", input: "100,50", want: "100.50"},
		{name: "zero", input: "0", want: "0.00"},
		{name: "negative", input: "-10", want: "-10.00"},
		{name: "invalid", input: "abc", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRUB(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.StringFixed(2) != tt.want {
				t.Fatalf("got %s, want %s", got.StringFixed(2), tt.want)
			}
		})
	}
}

func TestJSONDecimalUnmarshalRequiresString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "decimal string", input: `{"amount":"123.45"}`, want: "123.45"},
		{name: "integer string", input: `{"amount":"123"}`, want: "123"},
		{name: "json number rejected", input: `{"amount":123.45}`, wantErr: true},
		{name: "empty string rejected", input: `{"amount":""}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got struct {
				Amount JSONDecimal `json:"amount"`
			}
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Amount.String() != tt.want {
				t.Fatalf("got %s, want %s", got.Amount.String(), tt.want)
			}
		})
	}
}

func TestJSONDecimalMarshalWritesString(t *testing.T) {
	got, err := json.Marshal(struct {
		Amount JSONDecimal `json:"amount"`
	}{Amount: NewJSONDecimal(decimal.RequireFromString("123.45"))})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != `{"amount":"123.45"}` {
		t.Fatalf("got %s", got)
	}
}

func TestParsePositiveRUB(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "positive", input: "100"},
		{name: "zero", input: "0", wantErr: true},
		{name: "negative", input: "-1", wantErr: true},
		{name: "too large", input: "1000000.01", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePositiveRUB(tt.input)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCurrencyScale(t *testing.T) {
	tests := []struct {
		currency string
		want     int32
	}{
		{currency: "RUB", want: 2},
		{currency: "JPY", want: 0},
		{currency: "krw", want: 0},
		{currency: "KWD", want: 3},
		{currency: "CLF", want: 4},
		{currency: "USDT", want: 6},
		{currency: "UNKNOWN", want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.currency, func(t *testing.T) {
			if got := CurrencyScale(tt.currency); got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLegacyKopeckConversion(t *testing.T) {
	amount := LegacyKopecksToDecimal(10050)
	if amount.StringFixed(2) != "100.50" {
		t.Fatalf("got %s", amount.StringFixed(2))
	}

	kopecks, err := DecimalToLegacyKopecks(decimal.RequireFromString("100.50"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kopecks != 10050 {
		t.Fatalf("got %d, want 10050", kopecks)
	}
}

func TestParseRate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "whole", input: "12", want: "12"},
		{name: "fraction", input: "12.5", want: "12.5"},
		{name: "comma", input: "12,5", want: "12.5"},
		{name: "zero", input: "0", wantErr: true},
		{name: "negative", input: "-1", wantErr: true},
		{name: "invalid", input: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRate(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.String() != tt.want {
				t.Fatalf("got %s, want %s", got.String(), tt.want)
			}
		})
	}
}
