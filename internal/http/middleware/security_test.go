package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersPresentOnSuccess(t *testing.T) {
	handler := SecurityHeaders(SecurityHeadersConfig{
		PublicOrigin: "https://capitalflow.home.arpa",
		CookieSecure: true,
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health", http.NoBody))

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q", got)
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Fatal("missing HSTS for secure https public origin")
	}
}

func TestSecurityHeadersSkipHSTSWhenNotSecure(t *testing.T) {
	handler := SecurityHeaders(SecurityHeadersConfig{
		PublicOrigin: "http://localhost:5173",
		CookieSecure: false,
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health", http.NoBody))

	if got := rec.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("HSTS = %q, want empty", got)
	}
}

func TestAuthHostPolicyAllowsConfiguredHost(t *testing.T) {
	handler := AuthHostPolicy(HostPolicyConfig{
		AppEnv:           "production",
		PublicOrigin:     "https://capitalflow.home.arpa",
		PublicOriginHost: "capitalflow.home.arpa",
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", http.NoBody)
	req.Host = "capitalflow.home.arpa"
	req.Header.Set("Origin", "https://capitalflow.home.arpa")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthHostPolicyRejectsDirectIPInProduction(t *testing.T) {
	handler := AuthHostPolicy(HostPolicyConfig{
		AppEnv:             "production",
		PublicOrigin:       "https://capitalflow.home.arpa",
		PublicOriginHost:   "capitalflow.home.arpa",
		AllowDirectIPLogin: false,
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", http.NoBody)
	req.Host = "192.168.1.10"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAuthHostPolicyAllowsDevLoopback(t *testing.T) {
	handler := AuthHostPolicy(HostPolicyConfig{
		AppEnv:             "development",
		AllowDirectIPLogin: true,
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", http.NoBody)
	req.Host = "127.0.0.1"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthHostPolicyRejectsMalformedHost(t *testing.T) {
	handler := AuthHostPolicy(HostPolicyConfig{
		AppEnv:           "production",
		PublicOrigin:     "https://capitalflow.home.arpa",
		PublicOriginHost: "capitalflow.home.arpa",
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", http.NoBody)
	req.Host = "evil.example.com"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestAuthHostPolicyDoesNotBlockHealth(t *testing.T) {
	handler := AuthHostPolicy(HostPolicyConfig{
		AppEnv:           "production",
		PublicOrigin:     "https://capitalflow.home.arpa",
		PublicOriginHost: "capitalflow.home.arpa",
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health", http.NoBody)
	req.Host = "192.168.1.10"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestAuthHostPolicyRejectsWrongOrigin(t *testing.T) {
	handler := AuthHostPolicy(HostPolicyConfig{
		AppEnv:           "production",
		PublicOrigin:     "https://capitalflow.home.arpa",
		PublicOriginHost: "capitalflow.home.arpa",
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/login", http.NoBody)
	req.Host = "capitalflow.home.arpa"
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
