package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCategoriesRouteRequiresAuth(t *testing.T) {
	router := NewRouter(nil, "test-token")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/categories", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
