package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestPasskeyRegistrationOptionsRequiresPasswordForFirstPasskey(t *testing.T) {
	service, users, passkeys := newTestPasskeyService(t)
	users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com", PasswordHash: "hash:correct password"}

	_, err := service.RegistrationOptions(t.Context(), PasskeyRegistrationOptionsRequest{UserID: "user-1"})
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}

	options, err := service.RegistrationOptions(t.Context(), PasskeyRegistrationOptionsRequest{
		UserID:   "user-1",
		Password: "correct password",
	})
	if err != nil {
		t.Fatalf("registration options: %v", err)
	}
	if options == nil {
		t.Fatal("expected options")
	}
	if len(passkeys.challenges) != 1 {
		t.Fatalf("challenges = %d, want 1", len(passkeys.challenges))
	}
}

func TestPasskeyRegistrationOptionsSkipsPasswordWhenPasskeyExists(t *testing.T) {
	service, users, passkeys := newTestPasskeyService(t)
	users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com", PasswordHash: "hash:correct password"}
	passkeys.credentialsByID["passkey-1"] = &models.PasskeyCredential{
		ID:           "passkey-1",
		UserID:       "user-1",
		CredentialID: []byte("credential"),
		PublicKey:    []byte("public"),
		Name:         "Existing",
	}

	if _, err := service.RegistrationOptions(t.Context(), PasskeyRegistrationOptionsRequest{UserID: "user-1"}); err != nil {
		t.Fatalf("registration options: %v", err)
	}
}

func TestPasskeyChallengeSecurityCases(t *testing.T) {
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	userID := "user-1"

	tests := []struct {
		name      string
		record    *models.WebAuthnChallenge
		ceremony  string
		userID    *string
		usedAt    time.Time
		wantError bool
	}{
		{
			name: "valid challenge",
			record: &models.WebAuthnChallenge{
				Ceremony:  passkeyCeremonyRegistration,
				Challenge: "challenge",
				UserID:    &userID,
				ExpiresAt: now.Add(time.Minute),
			},
			ceremony: passkeyCeremonyRegistration,
			userID:   &userID,
			usedAt:   now,
		},
		{
			name: "expired challenge rejected",
			record: &models.WebAuthnChallenge{
				Ceremony:  passkeyCeremonyRegistration,
				Challenge: "challenge",
				UserID:    &userID,
				ExpiresAt: now.Add(-time.Minute),
			},
			ceremony:  passkeyCeremonyRegistration,
			userID:    &userID,
			usedAt:    now,
			wantError: true,
		},
		{
			name: "challenge from another user rejected",
			record: &models.WebAuthnChallenge{
				Ceremony:  passkeyCeremonyRegistration,
				Challenge: "challenge",
				UserID:    ptr("user-2"),
				ExpiresAt: now.Add(time.Minute),
			},
			ceremony:  passkeyCeremonyRegistration,
			userID:    &userID,
			usedAt:    now,
			wantError: true,
		},
		{
			name: "replayed challenge rejected",
			record: &models.WebAuthnChallenge{
				Ceremony:  passkeyCeremonyRegistration,
				Challenge: "challenge",
				UserID:    &userID,
				ExpiresAt: now.Add(time.Minute),
				UsedAt:    passkeyPtrTime(now.Add(-time.Second)),
			},
			ceremony:  passkeyCeremonyRegistration,
			userID:    &userID,
			usedAt:    now,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakePasskeyRepo{
				credentialsByID: map[string]*models.PasskeyCredential{},
				challenges:      map[string]*models.WebAuthnChallenge{"challenge": tt.record},
			}
			_, err := repo.ConsumeChallenge(t.Context(), tt.ceremony, "challenge", tt.userID, tt.usedAt)
			if (err != nil) != tt.wantError {
				t.Fatalf("ConsumeChallenge() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func newTestPasskeyService(t *testing.T) (*PasskeyService, *fakeUserRepo, *fakePasskeyRepo) {
	t.Helper()
	authService, users, _, audit := newTestAuthService(t)
	passkeys := &fakePasskeyRepo{
		credentialsByID: map[string]*models.PasskeyCredential{},
		challenges:      map[string]*models.WebAuthnChallenge{},
	}
	service, err := NewPasskeyService(users, passkeys, authService, audit, WebAuthnConfig{
		RPDisplayName: "CapitalFlow",
		RPID:          "localhost",
		Origins:       []string{"http://localhost:5173"},
	})
	if err != nil {
		t.Fatalf("new passkey service: %v", err)
	}
	service.now = authService.now
	return service, users, passkeys
}

type fakePasskeyRepo struct {
	credentialsByID map[string]*models.PasskeyCredential
	challenges      map[string]*models.WebAuthnChallenge
}

func (r *fakePasskeyRepo) CreateCredential(_ context.Context, credential *models.PasskeyCredential) error {
	for _, existing := range r.credentialsByID {
		if string(existing.CredentialID) == string(credential.CredentialID) {
			return repository.ErrConflict
		}
	}
	r.credentialsByID[credential.ID] = credential
	return nil
}

func (r *fakePasskeyRepo) ListCredentialsByUser(_ context.Context, userID string, includeRevoked bool) ([]models.PasskeyCredential, error) {
	credentials := []models.PasskeyCredential{}
	for _, credential := range r.credentialsByID {
		if credential.UserID == userID && (includeRevoked || credential.RevokedAt == nil) {
			credentials = append(credentials, *credential)
		}
	}
	return credentials, nil
}

func (r *fakePasskeyRepo) GetCredentialByIDForUser(_ context.Context, id, userID string) (*models.PasskeyCredential, error) {
	credential, ok := r.credentialsByID[id]
	if !ok || credential.UserID != userID {
		return nil, repository.ErrNotFound
	}
	return credential, nil
}

func (r *fakePasskeyRepo) GetCredentialByCredentialID(_ context.Context, credentialID []byte) (*models.PasskeyCredential, error) {
	for _, credential := range r.credentialsByID {
		if string(credential.CredentialID) == string(credentialID) {
			return credential, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *fakePasskeyRepo) CountActiveCredentialsByUser(_ context.Context, userID string) (int64, error) {
	var count int64
	for _, credential := range r.credentialsByID {
		if credential.UserID == userID && credential.RevokedAt == nil {
			count++
		}
	}
	return count, nil
}

func (r *fakePasskeyRepo) UpdateCredentialAfterLogin(_ context.Context, credentialID []byte, signCount uint32, cloneWarning, backupState bool, lastUsedAt time.Time) error {
	credential, err := r.GetCredentialByCredentialID(context.Background(), credentialID)
	if err != nil {
		return err
	}
	credential.SignCount = signCount
	credential.CloneWarning = cloneWarning
	credential.BackupState = backupState
	credential.LastUsedAt = &lastUsedAt
	return nil
}

func (r *fakePasskeyRepo) RenameCredential(_ context.Context, id, userID, name string, updatedAt time.Time) error {
	credential, err := r.GetCredentialByIDForUser(context.Background(), id, userID)
	if err != nil {
		return err
	}
	credential.Name = name
	credential.UpdatedAt = updatedAt
	return nil
}

func (r *fakePasskeyRepo) RevokeCredential(_ context.Context, id, userID string, revokedAt time.Time) error {
	credential, err := r.GetCredentialByIDForUser(context.Background(), id, userID)
	if err != nil {
		return err
	}
	credential.RevokedAt = &revokedAt
	return nil
}

func (r *fakePasskeyRepo) CreateChallenge(_ context.Context, challenge *models.WebAuthnChallenge) error {
	if _, exists := r.challenges[challenge.Challenge]; exists {
		return repository.ErrConflict
	}
	var session webauthn.SessionData
	_ = json.Unmarshal(challenge.SessionData, &session)
	r.challenges[challenge.Challenge] = challenge
	return nil
}

func (r *fakePasskeyRepo) ConsumeChallenge(_ context.Context, ceremony, challenge string, userID *string, usedAt time.Time) (*models.WebAuthnChallenge, error) {
	record, ok := r.challenges[challenge]
	if !ok || record.Ceremony != ceremony || record.UsedAt != nil || !usedAt.Before(record.ExpiresAt) {
		return nil, repository.ErrNotFound
	}
	if userID == nil {
		if record.UserID != nil {
			return nil, repository.ErrNotFound
		}
	} else if record.UserID == nil || *record.UserID != *userID {
		return nil, repository.ErrNotFound
	}
	record.UsedAt = &usedAt
	return record, nil
}

func ptr(value string) *string {
	return &value
}

func passkeyPtrTime(value time.Time) *time.Time {
	return &value
}
