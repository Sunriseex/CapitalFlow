package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/auth"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/pkg/security"
)

func TestAuthServiceSetupCreatesFirstUserSession(t *testing.T) {
	service, users, refresh := newTestAuthService(t)

	session, err := service.Setup(t.Context(), AuthRequest{
		Email:           " User@Example.COM ",
		Password:        "correct horse battery staple",
		PrimaryCurrency: "usd",
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if session.User.Email != "user@example.com" {
		t.Fatalf("email = %q", session.User.Email)
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Fatal("expected issued tokens")
	}
	if session.User.PrimaryCurrency != "USD" {
		t.Fatalf("primary currency = %q, want USD", session.User.PrimaryCurrency)
	}
	if len(users.byID) != 1 || len(refresh.byHash) != 1 {
		t.Fatalf("expected persisted user and refresh token")
	}
}

func TestAuthServiceSetupRejectsInvalidPrimaryCurrency(t *testing.T) {
	service, _, _ := newTestAuthService(t)

	_, err := service.Setup(t.Context(), AuthRequest{
		Email:           "user@example.com",
		Password:        "correct horse battery staple",
		PrimaryCurrency: "RU",
	})
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestAuthServiceSetupRejectsSecondUser(t *testing.T) {
	service, users, _ := newTestAuthService(t)
	users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com"}

	_, err := service.Setup(t.Context(), AuthRequest{
		Email:    "other@example.com",
		Password: "correct horse battery staple",
	})
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestAuthServiceLoginRejectsWrongPasswordWithSafeMessage(t *testing.T) {
	service, users, _ := newTestAuthService(t)
	users.byID["user-1"] = &models.User{
		ID:           "user-1",
		Email:        "user@example.com",
		PasswordHash: "secret",
	}

	_, err := service.Login(t.Context(), AuthRequest{
		Email:    "user@example.com",
		Password: "wrong password",
	})
	if err == nil || err.Error() != "invalid email or password" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthServiceRefreshRotatesToken(t *testing.T) {
	service, users, refresh := newTestAuthService(t)
	users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com"}
	oldRaw := "old-refresh-token"
	oldHash := auth.HashRefreshToken(oldRaw)
	refresh.byHash[oldHash] = &models.RefreshToken{
		ID:        "token-1",
		UserID:    "user-1",
		TokenHash: oldHash,
		ExpiresAt: service.now().Add(time.Hour),
		CreatedAt: service.now(),
	}

	session, err := service.Refresh(t.Context(), oldRaw)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if session.RefreshToken == "" || session.RefreshToken == oldRaw {
		t.Fatal("expected rotated refresh token")
	}
	if refresh.byHash[oldHash].RevokedAt == nil {
		t.Fatal("expected old refresh token to be revoked")
	}
	if len(refresh.byHash) != 2 {
		t.Fatalf("refresh token count = %d, want 2", len(refresh.byHash))
	}
}

func TestAuthServiceLogoutRevokesRefreshToken(t *testing.T) {
	service, _, refresh := newTestAuthService(t)
	raw := "refresh-token"
	hash := auth.HashRefreshToken(raw)
	refresh.byHash[hash] = &models.RefreshToken{
		ID:        "token-1",
		UserID:    "user-1",
		TokenHash: hash,
		ExpiresAt: service.now().Add(time.Hour),
		CreatedAt: service.now(),
	}

	if err := service.Logout(t.Context(), raw); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if refresh.byHash[hash].RevokedAt == nil {
		t.Fatal("expected refresh token to be revoked")
	}
}

func newTestAuthService(t *testing.T) (*AuthService, *fakeUserRepo, *fakeRefreshRepo) {
	t.Helper()

	tokens, err := auth.NewTokenService("01234567890123456789012345678901", "capitalflow", time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}

	users := &fakeUserRepo{byID: map[string]*models.User{}}
	refresh := &fakeRefreshRepo{byHash: map[string]*models.RefreshToken{}}
	service := NewAuthService(users, refresh, tokens)
	service.now = func() time.Time {
		return time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	}
	service.passwordFunc = func(password string, _ security.PasswordParams) (string, error) {
		return "hash:" + password, nil
	}
	service.verifyFunc = func(password, encodedHash string) (bool, error) {
		return encodedHash == "hash:"+password, nil
	}

	return service, users, refresh
}

type fakeUserRepo struct {
	byID map[string]*models.User
}

func (r *fakeUserRepo) Create(_ context.Context, user *models.User) error {
	r.byID[user.ID] = user
	return nil
}

func (r *fakeUserRepo) Count(context.Context) (int64, error) {
	return int64(len(r.byID)), nil
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*models.User, error) {
	for _, user := range r.byID {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *fakeUserRepo) GetByID(_ context.Context, id string) (*models.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return user, nil
}

func (r *fakeUserRepo) UpdatePrimaryCurrency(_ context.Context, id, primaryCurrency string, updatedAt time.Time) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.PrimaryCurrency = primaryCurrency
	user.UpdatedAt = updatedAt
	return nil
}

type fakeRefreshRepo struct {
	byHash map[string]*models.RefreshToken
}

func (r *fakeRefreshRepo) Create(_ context.Context, token *models.RefreshToken) error {
	if token.TokenHash == "" {
		return errors.New("token hash is required")
	}
	r.byHash[token.TokenHash] = token
	return nil
}

func (r *fakeRefreshRepo) GetByID(_ context.Context, id string) (*models.RefreshToken, error) {
	for _, token := range r.byHash {
		if token.ID == id {
			return token, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *fakeRefreshRepo) GetByHash(_ context.Context, tokenHash string) (*models.RefreshToken, error) {
	token, ok := r.byHash[tokenHash]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return token, nil
}

func (r *fakeRefreshRepo) Revoke(_ context.Context, id string, revokedAt time.Time) error {
	for _, token := range r.byHash {
		if token.ID == id {
			token.RevokedAt = &revokedAt
			return nil
		}
	}
	return repository.ErrNotFound
}

func (r *fakeRefreshRepo) RevokeByUser(_ context.Context, userID string, revokedAt time.Time) error {
	for _, token := range r.byHash {
		if token.UserID == userID && token.RevokedAt == nil {
			token.RevokedAt = &revokedAt
		}
	}
	return nil
}
