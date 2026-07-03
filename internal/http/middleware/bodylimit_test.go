package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLimitRequestBodyRejectsKnownOversizeBody(t *testing.T) {
	called := false
	handler := LimitRequestBody(4)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader("12345"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge || called {
		t.Fatalf("status=%d called=%v", rec.Code, called)
	}
}

func TestLimitRequestBodyCapsChunkedBody(t *testing.T) {
	handler := LimitRequestBody(4)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buffer := make([]byte, 5)
		if _, err := r.Body.Read(buffer); err == nil {
			t.Fatal("expected body limit error")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader("12345"))
	req.ContentLength = -1
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rec.Code)
	}
}
