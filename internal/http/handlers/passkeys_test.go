package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	router := NewRouter(store, &RouterConfig{TokenService: tokens})

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
