package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/application"
	"github.com/sunriseex/capitalflow/internal/auth"
	"github.com/sunriseex/capitalflow/internal/models"
)

func TestPasskeyRegistrationRoutesRequireJWT(t *testing.T) {
	router := newTestAuthRouter(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/passkeys/registration/options", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestPasskeyLoginOptionsReturnsPublicKeyOptions(t *testing.T) {
	router := newTestAuthRouter(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/passkeys/login/options", http.NoBody)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"publicKey"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestPasskeyLoginOptionsUsesDedicatedRateLimit(t *testing.T) {
	tokens, err := auth.NewTokenService(testJWTSecret, "capitalflow", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}
	router := newTestRouter(newTestProfileStore(), &RouterConfig{
		AuthRateLimitRequests:           20,
		AuthRateLimitWindow:             time.Minute,
		PasskeyOptionsRateLimitRequests: 1,
		PasskeyOptionsRateLimitWindow:   time.Minute,
	}, tokens)

	for index, want := range []int{http.StatusOK, http.StatusTooManyRequests} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/passkeys/login/options", http.NoBody)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("request %d status = %d, want %d: %s", index+1, rec.Code, want, rec.Body.String())
		}
	}
}

func TestPasskeyLoginOptionsUnavailableWhenServiceIsNotConfigured(t *testing.T) {
	router := newTestRouter(newTestProfileStore(), &RouterConfig{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/passkeys/login/options", http.NoBody)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}

func TestPasskeyInvalidConfigPanicsAtRouterConstruction(t *testing.T) {
	tokens, err := auth.NewTokenService(testJWTSecret, "capitalflow", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected router construction to panic")
		}
	}()

	newTestRouterWithApplicationConfig(newTestProfileStore(), &RouterConfig{}, application.Config{
		TokenService:    tokens,
		WebAuthnOrigins: []string{"://bad-origin"},
	})
}

func TestPasskeyServiceIsReusedBetweenRequests(t *testing.T) {
	tokens, err := auth.NewTokenService(testJWTSecret, "capitalflow", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}
	store := newTestProfileStore()
	router := newTestRouter(store, &RouterConfig{
		PasskeyOptionsRateLimitRequests: 10,
	}, tokens)

	for range 2 {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/passkeys/login/options", http.NoBody)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	}
	if store.passkeys.deleteCalls != 1 {
		t.Fatalf("challenge cleanup calls = %d, want 1", store.passkeys.deleteCalls)
	}
}

func TestPasskeyListRenameDeleteScopeToUser(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	store.users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com", PrimaryCurrency: "RUB"}
	store.refresh.byID[pair.RefreshTokenID] = &models.RefreshToken{
		ID:        pair.RefreshTokenID,
		UserID:    "user-1",
		TokenHash: pair.RefreshTokenHash,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	store.passkeys.credentialsByID["passkey-1"] = &models.PasskeyCredential{
		ID:           "passkey-1",
		UserID:       "user-1",
		CredentialID: []byte("credential"),
		PublicKey:    []byte("public"),
		Name:         "Old",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	router := newTestRouter(store, &RouterConfig{}, tokens)

	renameReq := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/auth/passkeys/passkey-1", strings.NewReader(`{"name":"Laptop"}`))
	renameReq.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	renameReq.Header.Set("Idempotency-Key", "rename-passkey")
	renameRec := httptest.NewRecorder()
	router.ServeHTTP(renameRec, renameReq)
	if renameRec.Code != http.StatusOK {
		t.Fatalf("rename status = %d, want %d: %s", renameRec.Code, http.StatusOK, renameRec.Body.String())
	}
	if store.passkeys.credentialsByID["passkey-1"].Name != "Laptop" {
		t.Fatalf("name = %q, want Laptop", store.passkeys.credentialsByID["passkey-1"].Name)
	}

	deleteReq := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/v1/auth/passkeys/passkey-1", http.NoBody)
	deleteReq.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	deleteReq.Header.Set("Idempotency-Key", "delete-passkey")
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d: %s", deleteRec.Code, http.StatusNoContent, deleteRec.Body.String())
	}
	if store.passkeys.credentialsByID["passkey-1"].RevokedAt == nil {
		t.Fatal("passkey was not revoked")
	}
}
