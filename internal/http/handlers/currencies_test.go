package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCurrencyRatesRouteRequiresAuth(t *testing.T) {
	router := newTestRouter(nil, &RouterConfig{APIAuthToken: "01234567890123456789012345678901"})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/currency-rates?base=RUB", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestCurrencyRatesRejectsInvalidBase(t *testing.T) {
	router := newTestRouter(nil, &RouterConfig{APIAuthToken: "01234567890123456789012345678901"})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/currency-rates?base=RU", nil)
	req.Header.Set("Authorization", "Bearer 01234567890123456789012345678901")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
