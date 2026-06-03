package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestPasskeyRegistrationOptionsRequiresPasswordForFirstPasskey(t *testing.T) {
	service, users, _, passkeys := newTestPasskeyService(t)
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
	service, users, _, passkeys := newTestPasskeyService(t)
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

func TestPasskeyProtocolRejectsWrongOriginAndRPID(t *testing.T) {
	t.Run("wrong origin rejected", func(t *testing.T) {
		clientData := protocol.CollectedClientData{
			Type:      protocol.AssertCeremony,
			Challenge: "challenge",
			Origin:    "https://evil.example",
		}

		err := clientData.Verify(
			"challenge",
			protocol.AssertCeremony,
			[]string{"https://capitalflow.example"},
			nil,
			protocol.TopOriginIgnoreVerificationMode,
		)
		if err == nil {
			t.Fatal("expected wrong origin to be rejected")
		}
	})

	t.Run("wrong rpID rejected", func(t *testing.T) {
		expected := sha256.Sum256([]byte("capitalflow.example"))
		wrong := sha256.Sum256([]byte("evil.example"))
		authData := protocol.AuthenticatorData{RPIDHash: wrong[:]}

		if err := authData.Verify(expected[:], nil, false, false); err == nil {
			t.Fatal("expected wrong rpID hash to be rejected")
		}
	})
}

func TestPasskeyVerifyRegistrationRejectsCredentialCollision(t *testing.T) {
	service, users, _, passkeys := newTestPasskeyService(t)
	users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com", PasswordHash: "hash:correct password"}
	fakeRP := &fakePasskeyRP{
		registrationChallenge: "registration-challenge",
		registrationCredential: &webauthn.Credential{
			ID:        []byte("credential-1"),
			PublicKey: []byte("public-key"),
		},
	}
	service.webAuthn = fakeRP
	service.parseCreation = fakeParseCreation("registration-challenge")
	passkeys.credentialsByID["existing"] = &models.PasskeyCredential{
		ID:           "existing",
		UserID:       "user-1",
		CredentialID: []byte("credential-1"),
		PublicKey:    []byte("existing-public-key"),
		Name:         "Existing",
	}

	if _, err := service.RegistrationOptions(t.Context(), PasskeyRegistrationOptionsRequest{
		UserID:   "user-1",
		Password: "correct password",
	}); err != nil {
		t.Fatalf("registration options: %v", err)
	}

	_, err := service.VerifyRegistration(t.Context(), "user-1", []byte(`{}`))
	if !IsValidationError(err) {
		t.Fatalf("expected validation error for credential collision, got %v", err)
	}
}

func TestPasskeyDiscoverableUserHandlerRejectsRevokedDeletedAndWrongUser(t *testing.T) {
	service, users, _, passkeys := newTestPasskeyService(t)
	users.byID["user-1"] = &models.User{ID: "user-1", Email: "user@example.com"}
	revokedAt := service.now()
	passkeys.credentialsByID["credential-1"] = &models.PasskeyCredential{
		ID:           "credential-1",
		UserID:       "user-1",
		CredentialID: []byte("raw-credential-1"),
		PublicKey:    []byte("public-key"),
		Name:         "Deleted",
		RevokedAt:    &revokedAt,
	}

	handler := service.discoverableUserHandler(t.Context())
	if _, err := handler([]byte("raw-credential-1"), []byte("user-1")); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("revoked credential error = %v, want ErrNotFound", err)
	}

	passkeys.credentialsByID["credential-1"].RevokedAt = nil
	if _, err := handler([]byte("raw-credential-1"), []byte("user-2")); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("wrong user handle error = %v, want ErrNotFound", err)
	}
}

func TestPasskeyE2ESmokeInMemoryCreatesRefreshSession(t *testing.T) {
	service, users, refresh, passkeys := newTestPasskeyService(t)
	users.byID["user-1"] = &models.User{
		ID:              "user-1",
		Email:           "user@example.com",
		PasswordHash:    "hash:correct password",
		PrimaryCurrency: "RUB",
	}
	fakeRP := &fakePasskeyRP{
		registrationChallenge: "registration-challenge",
		loginChallenge:        "login-challenge",
		registrationCredential: &webauthn.Credential{
			ID:        []byte("credential-1"),
			PublicKey: []byte("public-key"),
			Flags: webauthn.CredentialFlags{
				BackupEligible: true,
				BackupState:    true,
			},
			Authenticator: webauthn.Authenticator{SignCount: 1},
		},
		loginRawID:      []byte("credential-1"),
		loginUserHandle: []byte("user-1"),
		loginCredential: &webauthn.Credential{
			ID:            []byte("credential-1"),
			PublicKey:     []byte("public-key"),
			Flags:         webauthn.CredentialFlags{BackupEligible: true, BackupState: true},
			Authenticator: webauthn.Authenticator{SignCount: 2},
		},
	}
	service.webAuthn = fakeRP
	service.parseCreation = fakeParseCreation("registration-challenge")
	service.parseAssertion = fakeParseAssertion("login-challenge")

	if _, err := service.RegistrationOptions(t.Context(), PasskeyRegistrationOptionsRequest{
		UserID:   "user-1",
		Password: "correct password",
	}); err != nil {
		t.Fatalf("registration options: %v", err)
	}
	if _, err := service.VerifyRegistration(t.Context(), "user-1", []byte(`{}`)); err != nil {
		t.Fatalf("verify registration: %v", err)
	}
	if len(passkeys.credentialsByID) != 1 {
		t.Fatalf("passkeys = %d, want 1", len(passkeys.credentialsByID))
	}

	if _, err := service.LoginOptions(t.Context()); err != nil {
		t.Fatalf("login options: %v", err)
	}
	session, err := service.VerifyLogin(t.Context(), []byte(`{}`))
	if err != nil {
		t.Fatalf("verify login: %v", err)
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Fatal("expected normal auth session tokens")
	}
	if len(refresh.byHash) != 1 {
		t.Fatalf("refresh tokens = %d, want 1", len(refresh.byHash))
	}
	for _, credential := range passkeys.credentialsByID {
		if credential.SignCount != 2 || credential.LastUsedAt == nil {
			t.Fatalf("credential after login = %+v, want sign count 2 and last used", credential)
		}
	}
}

func newTestPasskeyService(t *testing.T) (*PasskeyService, *fakeUserRepo, *fakeRefreshRepo, *fakePasskeyRepo) {
	t.Helper()
	authService, users, refresh, audit := newTestAuthService(t)
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
	return service, users, refresh, passkeys
}

type fakePasskeyRP struct {
	registrationChallenge  string
	registrationCredential *webauthn.Credential
	loginChallenge         string
	loginRawID             []byte
	loginUserHandle        []byte
	loginCredential        *webauthn.Credential
}

func (r *fakePasskeyRP) BeginRegistration(user webauthn.User, _ ...webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	return &protocol.CredentialCreation{}, &webauthn.SessionData{
		Challenge: r.registrationChallenge,
		UserID:    user.WebAuthnID(),
		Expires:   time.Now().Add(time.Minute),
	}, nil
}

func (r *fakePasskeyRP) CreateCredential(webauthn.User, webauthn.SessionData, *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error) {
	return r.registrationCredential, nil
}

func (r *fakePasskeyRP) BeginDiscoverableLogin(_ ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	return &protocol.CredentialAssertion{}, &webauthn.SessionData{
		Challenge: r.loginChallenge,
		Expires:   time.Now().Add(time.Minute),
	}, nil
}

func (r *fakePasskeyRP) ValidatePasskeyLogin(handler webauthn.DiscoverableUserHandler, _ webauthn.SessionData, _ *protocol.ParsedCredentialAssertionData) (webauthn.User, *webauthn.Credential, error) { //nolint:gocritic // Interface requires value session data.
	user, err := handler(r.loginRawID, r.loginUserHandle)
	if err != nil {
		return nil, nil, err
	}
	return user, r.loginCredential, nil
}

func fakeParseCreation(challenge string) func([]byte) (*protocol.ParsedCredentialCreationData, error) {
	return func([]byte) (*protocol.ParsedCredentialCreationData, error) {
		return &protocol.ParsedCredentialCreationData{
			Response: protocol.ParsedAttestationResponse{
				CollectedClientData: protocol.CollectedClientData{Challenge: challenge},
			},
		}, nil
	}
}

func fakeParseAssertion(challenge string) func([]byte) (*protocol.ParsedCredentialAssertionData, error) {
	return func([]byte) (*protocol.ParsedCredentialAssertionData, error) {
		return &protocol.ParsedCredentialAssertionData{
			Response: protocol.ParsedAssertionResponse{
				CollectedClientData: protocol.CollectedClientData{Challenge: challenge},
			},
		}, nil
	}
}

type fakePasskeyRepo struct {
	credentialsByID map[string]*models.PasskeyCredential
	challenges      map[string]*models.WebAuthnChallenge
}

func (r *fakePasskeyRepo) CreateCredential(_ context.Context, credential *models.PasskeyCredential) error {
	for _, existing := range r.credentialsByID {
		if bytes.Equal(existing.CredentialID, credential.CredentialID) {
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
		if bytes.Equal(credential.CredentialID, credentialID) {
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

func (r *fakePasskeyRepo) UpdateCredentialAfterLogin(ctx context.Context, credentialID []byte, signCount uint32, cloneWarning, backupState bool, lastUsedAt time.Time) error {
	credential, err := r.GetCredentialByCredentialID(ctx, credentialID)
	if err != nil {
		return err
	}
	credential.SignCount = signCount
	credential.CloneWarning = cloneWarning
	credential.BackupState = backupState
	credential.LastUsedAt = &lastUsedAt
	return nil
}

func (r *fakePasskeyRepo) RenameCredential(ctx context.Context, id, userID, name string, updatedAt time.Time) error {
	credential, err := r.GetCredentialByIDForUser(ctx, id, userID)
	if err != nil {
		return err
	}
	credential.Name = name
	credential.UpdatedAt = updatedAt
	return nil
}

func (r *fakePasskeyRepo) RevokeCredential(ctx context.Context, id, userID string, revokedAt time.Time) error {
	credential, err := r.GetCredentialByIDForUser(ctx, id, userID)
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
