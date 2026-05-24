package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestCurrencyServiceConvertDecimalAmount(t *testing.T) {
	service := NewCurrencyService(staticExchangeRateProvider{
		rates: &ExchangeRates{
			Base: "RUB",
			Rates: map[string]decimal.Decimal{
				"RUB": decimal.NewFromInt(1),
				"KRW": decimal.RequireFromString("16.25"),
			},
		},
	})

	amount, rate, err := service.ConvertDecimalAmount(t.Context(), decimal.RequireFromString("10000"), "rub", "krw")
	if err != nil {
		t.Fatalf("convert decimal amount: %v", err)
	}
	if !amount.Equal(decimal.RequireFromString("162500")) {
		t.Fatalf("amount = %s, want 162500", amount)
	}
	if rate.String() != "16.25" {
		t.Fatalf("rate = %s, want 16.25", rate)
	}
}

func TestCurrencyServiceConvertDecimalAmountDoesNotRound(t *testing.T) {
	service := NewCurrencyService(staticExchangeRateProvider{
		rates: &ExchangeRates{
			Base: "USD",
			Rates: map[string]decimal.Decimal{
				"USD": decimal.NewFromInt(1),
				"EUR": decimal.RequireFromString("0.333"),
			},
		},
	})

	amount, rate, err := service.ConvertDecimalAmount(t.Context(), decimal.RequireFromString("10"), "usd", "eur")
	if err != nil {
		t.Fatalf("convert amount: %v", err)
	}
	if !amount.Equal(decimal.RequireFromString("3.330")) {
		t.Fatalf("amount = %s, want 3.330", amount)
	}
	if !rate.Equal(decimal.RequireFromString("0.333")) {
		t.Fatalf("rate = %s, want 0.333", rate)
	}
}

func TestCurrencyServiceConvertDecimalAmountSameCurrency(t *testing.T) {
	service := NewCurrencyService(staticExchangeRateProvider{})

	amount, rate, err := service.ConvertDecimalAmount(t.Context(), decimal.RequireFromString("100"), "USD", "USD")
	if err != nil {
		t.Fatalf("convert decimal amount: %v", err)
	}
	if !amount.Equal(decimal.RequireFromString("100")) {
		t.Fatalf("amount = %s, want 100", amount)
	}
	if !rate.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("rate = %s, want 1", rate)
	}
}

func TestHTTPExchangeRateProviderLatestCachesRates(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/RUB" {
			t.Fatalf("path = %s, want /RUB", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"result": "success",
			"base_code": "RUB",
			"time_last_update_utc": "Sat, 09 May 2026 00:00:01 +0000",
			"rates": {"KRW": 16.25}
		}`))
	}))
	defer server.Close()

	provider := NewHTTPExchangeRateProvider(server.Client(), server.URL, time.Hour)

	first, err := provider.Latest(t.Context(), "rub")
	if err != nil {
		t.Fatalf("latest first: %v", err)
	}
	second, err := provider.Latest(t.Context(), "RUB")
	if err != nil {
		t.Fatalf("latest second: %v", err)
	}

	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
	if first != second {
		t.Fatal("expected cached pointer on second call")
	}
	if first.Rates["KRW"].String() != "16.25" {
		t.Fatalf("KRW rate = %s, want 16.25", first.Rates["KRW"])
	}
}

type staticExchangeRateProvider struct {
	rates *ExchangeRates
}

func (p staticExchangeRateProvider) Latest(context.Context, string) (*ExchangeRates, error) {
	if p.rates == nil {
		return &ExchangeRates{
			Base:      "USD",
			FetchedAt: time.Now(),
			Rates:     map[string]decimal.Decimal{},
		}, nil
	}
	return p.rates, nil
}
