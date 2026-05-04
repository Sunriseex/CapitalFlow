package money

import (
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
