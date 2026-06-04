package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

const (
	passkeyCeremonyRegistration = "registration"
	passkeyCeremonyLogin        = "login"
	passkeyChallengeTTL         = 5 * time.Minute
	passkeyChallengeCleanupTTL  = time.Hour
)

// WebAuthnConfig contains relying party settings for passkey ceremonies.
type WebAuthnConfig struct {
	RPDisplayName string
	RPID          string
	Origins       []string
}

// PasskeyRegistrationOptionsRequest starts passkey registration.
type PasskeyRegistrationOptionsRequest struct {
	UserID   string
	Password string
}

// PasskeyRenameRequest renames a passkey.
type PasskeyRenameRequest struct {
	UserID string
	ID     string
	Name   string
}

// PasskeyDeleteRequest deletes a passkey.
type PasskeyDeleteRequest struct {
	UserID string
	ID     string
}

// PasskeyService manages WebAuthn passkey registration and login.
type PasskeyService struct {
	users          repository.UserRepository
	passkeys       repository.PasskeyRepository
	auth           *AuthService
	audit          repository.AuthAuditRepository
	webAuthn       passkeyRelyingParty
	parseCreation  func([]byte) (*protocol.ParsedCredentialCreationData, error)
	parseAssertion func([]byte) (*protocol.ParsedCredentialAssertionData, error)
	now            func() time.Time
	challengeTTL   time.Duration
	cleanupEvery   time.Duration
	cleanupMu      sync.Mutex
	lastCleanup    time.Time
}

type passkeyRelyingParty interface {
	BeginRegistration(user webauthn.User, opts ...webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error)
	CreateCredential(user webauthn.User, session webauthn.SessionData, parsedResponse *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error)
	BeginDiscoverableLogin(opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error)
	ValidatePasskeyLogin(handler webauthn.DiscoverableUserHandler, session webauthn.SessionData, parsedResponse *protocol.ParsedCredentialAssertionData) (webauthn.User, *webauthn.Credential, error)
}

// NewPasskeyService creates a PasskeyService.
func NewPasskeyService(users repository.UserRepository, passkeys repository.PasskeyRepository, authService *AuthService, audit repository.AuthAuditRepository, cfg WebAuthnConfig) (*PasskeyService, error) {
	if authService == nil {
		return nil, validationError("auth service is required")
	}
	rpDisplayName := strings.TrimSpace(cfg.RPDisplayName)
	if rpDisplayName == "" {
		rpDisplayName = "CapitalFlow"
	}
	rpID := strings.TrimSpace(cfg.RPID)
	if rpID == "" {
		return nil, validationError("webauthn rp id is required")
	}
	if len(cfg.Origins) == 0 {
		return nil, validationError("webauthn origins are required")
	}
	for _, origin := range cfg.Origins {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
			parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
			return nil, validationError("webauthn origins must be valid http or https origins without path, query, or fragment")
		}
	}
	webAuthn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: rpDisplayName,
		RPID:          rpID,
		RPOrigins:     cfg.Origins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationPreferred,
		},
		Timeouts: webauthn.TimeoutsConfig{
			Login:        webauthn.TimeoutConfig{Enforce: true, Timeout: passkeyChallengeTTL, TimeoutUVD: passkeyChallengeTTL},
			Registration: webauthn.TimeoutConfig{Enforce: true, Timeout: passkeyChallengeTTL, TimeoutUVD: passkeyChallengeTTL},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create webauthn: %w", err)
	}
	return &PasskeyService{
		users:          users,
		passkeys:       passkeys,
		auth:           authService,
		audit:          audit,
		webAuthn:       webAuthn,
		parseCreation:  protocol.ParseCredentialCreationResponseBytes,
		parseAssertion: protocol.ParseCredentialRequestResponseBytes,
		now:            time.Now,
		challengeTTL:   passkeyChallengeTTL,
		cleanupEvery:   passkeyChallengeCleanupTTL,
	}, nil
}

// List returns passkeys for a user.
func (s *PasskeyService) List(ctx context.Context, userID string) ([]models.PasskeyCredential, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, validationError("user is required")
	}
	credentials, err := s.passkeys.ListCredentialsByUser(ctx, userID, false)
	if err != nil {
		return nil, fmt.Errorf("list passkeys: %w", err)
	}
	return credentials, nil
}

// RegistrationOptions creates WebAuthn registration options.
func (s *PasskeyService) RegistrationOptions(ctx context.Context, req PasskeyRegistrationOptionsRequest) (any, error) {
	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		return nil, validationError("user is required")
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	confirmation, err := s.auth.VerifyPasswordConfirmation(ctx, user, req.Password)
	if err != nil {
		return nil, err
	}
	if !confirmation.OK {
		reason := "fresh_session_required"
		if confirmation.Locked {
			reason = "account_locked"
		}
		s.auditEvent(ctx, "passkey_registration_failed", user.Email, &user.ID, false, reason)
		return nil, validationError("passkey registration requires recent password confirmation")
	}
	s.cleanupChallenges(ctx)
	credentials, err := s.passkeys.ListCredentialsByUser(ctx, user.ID, false)
	if err != nil {
		return nil, fmt.Errorf("list passkeys: %w", err)
	}
	webUser := passkeyUser{user: user, credentials: credentials}
	options, session, err := s.webAuthn.BeginRegistration(
		webUser,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithExclusions(webauthn.Credentials(webUser.WebAuthnCredentials()).CredentialDescriptors()),
	)
	if err != nil {
		s.auditEvent(ctx, "passkey_registration_failed", user.Email, &user.ID, false, "options_failed")
		return nil, fmt.Errorf("begin passkey registration: %w", err)
	}
	if err := s.storeChallenge(ctx, user.ID, passkeyCeremonyRegistration, session); err != nil {
		s.auditEvent(ctx, "passkey_registration_failed", user.Email, &user.ID, false, "challenge_store_failed")
		return nil, err
	}
	s.auditEvent(ctx, "passkey_registration_options", user.Email, &user.ID, true, "")
	return options, nil
}

// VerifyRegistration verifies a WebAuthn registration response and stores the credential.
func (s *PasskeyService) VerifyRegistration(ctx context.Context, userID string, body []byte) (*models.PasskeyCredential, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, validationError("user is required")
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	parsed, err := s.parseCreation(body)
	if err != nil {
		s.auditEvent(ctx, "passkey_registration_failed", user.Email, &user.ID, false, "invalid_response")
		return nil, validationError("passkey registration failed")
	}
	challenge, err := s.consumeChallenge(ctx, passkeyCeremonyRegistration, parsed.Response.CollectedClientData.Challenge, &user.ID)
	if err != nil {
		s.auditEvent(ctx, "passkey_registration_failed", user.Email, &user.ID, false, "invalid_challenge")
		return nil, validationError("passkey registration failed")
	}
	session, err := decodeSessionData(challenge.SessionData)
	if err != nil {
		return nil, err
	}
	credentials, err := s.passkeys.ListCredentialsByUser(ctx, user.ID, false)
	if err != nil {
		return nil, fmt.Errorf("list passkeys: %w", err)
	}
	webUser := passkeyUser{user: user, credentials: credentials}
	credential, err := s.webAuthn.CreateCredential(webUser, session, parsed)
	if err != nil {
		s.auditEvent(ctx, "passkey_registration_failed", user.Email, &user.ID, false, "verification_failed")
		return nil, validationError("passkey registration failed")
	}
	now := s.now()
	record := credentialModelFromWebAuthn(credential, user.ID, defaultPasskeyName(now), now)
	if err := s.passkeys.CreateCredential(ctx, record); err != nil {
		reason := "save_failed"
		if errors.Is(err, repository.ErrConflict) {
			reason = "credential_collision"
		}
		s.auditEvent(ctx, "passkey_registration_failed", user.Email, &user.ID, false, reason)
		if errors.Is(err, repository.ErrConflict) {
			return nil, validationError("passkey registration failed")
		}
		return nil, fmt.Errorf("save passkey: %w", err)
	}
	s.auditEvent(ctx, "passkey_registration_success", user.Email, &user.ID, true, "")
	return record, nil
}

// LoginOptions creates WebAuthn passkey login options.
func (s *PasskeyService) LoginOptions(ctx context.Context) (any, error) {
	s.cleanupChallenges(ctx)
	options, session, err := s.webAuthn.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationPreferred))
	if err != nil {
		s.auditEvent(ctx, "passkey_login_failed", "", nil, false, "options_failed")
		return nil, fmt.Errorf("begin passkey login: %w", err)
	}
	if err := s.storeChallenge(ctx, "", passkeyCeremonyLogin, session); err != nil {
		s.auditEvent(ctx, "passkey_login_failed", "", nil, false, "challenge_store_failed")
		return nil, err
	}
	s.auditEvent(ctx, "passkey_login_options", "", nil, true, "")
	return options, nil
}

// VerifyLogin verifies a passkey login response and creates a normal auth session.
func (s *PasskeyService) VerifyLogin(ctx context.Context, body []byte) (*AuthSession, error) {
	parsed, err := s.parseAssertion(body)
	if err != nil {
		s.auditEvent(ctx, "passkey_login_failed", "", nil, false, "invalid_response")
		return nil, validationError("passkey login failed")
	}
	challenge, err := s.consumeChallenge(ctx, passkeyCeremonyLogin, parsed.Response.CollectedClientData.Challenge, nil)
	if err != nil {
		s.auditEvent(ctx, "passkey_login_failed", "", nil, false, "invalid_challenge")
		return nil, validationError("passkey login failed")
	}
	sessionData, err := decodeSessionData(challenge.SessionData)
	if err != nil {
		return nil, err
	}
	webUser, credential, err := s.webAuthn.ValidatePasskeyLogin(s.discoverableUserHandler(ctx), sessionData, parsed)
	if err != nil {
		s.auditEvent(ctx, "passkey_login_failed", "", nil, false, "verification_failed")
		return nil, validationError("passkey login failed")
	}
	user := webUser.(passkeyUser).user
	if s.auth.UserLocked(user) {
		s.auditEvent(ctx, "passkey_login_failed", user.Email, &user.ID, false, "account_locked")
		return nil, validationError("passkey login failed")
	}
	now := s.now()
	if err := s.passkeys.UpdateCredentialAfterLogin(ctx, credential.ID, credential.Authenticator.SignCount, credential.Authenticator.CloneWarning, credential.Flags.BackupState, now); err != nil {
		s.auditEvent(ctx, "passkey_login_failed", user.Email, &user.ID, false, "credential_update_failed")
		return nil, fmt.Errorf("update passkey login: %w", err)
	}
	authSession, err := s.auth.IssueSessionForUser(ctx, user.ID)
	if err != nil {
		s.auditEvent(ctx, "passkey_login_failed", user.Email, &user.ID, false, "issue_session_failed")
		return nil, err
	}
	s.auditEvent(ctx, "passkey_login_success", user.Email, &user.ID, true, "")
	return authSession, nil
}

func (s *PasskeyService) cleanupChallenges(ctx context.Context) {
	now := s.now()
	s.cleanupMu.Lock()
	if s.cleanupEvery > 0 && !s.lastCleanup.IsZero() && now.Sub(s.lastCleanup) < s.cleanupEvery {
		s.cleanupMu.Unlock()
		return
	}
	s.lastCleanup = now
	s.cleanupMu.Unlock()

	if err := s.passkeys.DeleteExpiredChallenges(ctx, now); err != nil {
		s.auditEvent(ctx, "passkey_challenge_cleanup_failed", "", nil, false, "cleanup_failed")
	}
}

func (s *PasskeyService) discoverableUserHandler(ctx context.Context) webauthn.DiscoverableUserHandler {
	return func(rawID, userHandle []byte) (webauthn.User, error) {
		credential, err := s.passkeys.GetCredentialByCredentialID(ctx, rawID)
		if err != nil {
			return nil, fmt.Errorf("get passkey credential: %w", err)
		}
		if !credential.IsActive() {
			return nil, repository.ErrNotFound
		}
		if string(userHandle) != credential.UserID {
			return nil, repository.ErrNotFound
		}
		user, err := s.users.GetByID(ctx, credential.UserID)
		if err != nil {
			return nil, fmt.Errorf("get passkey user: %w", err)
		}
		return passkeyUser{user: user, credentials: []models.PasskeyCredential{*credential}}, nil
	}
}

// Rename renames a passkey.
func (s *PasskeyService) Rename(ctx context.Context, req PasskeyRenameRequest) (*models.PasskeyCredential, error) {
	name, err := normalizePasskeyName(req.Name)
	if err != nil {
		return nil, err
	}
	now := s.now()
	if err := s.passkeys.RenameCredential(ctx, strings.TrimSpace(req.ID), strings.TrimSpace(req.UserID), name, now); err != nil {
		s.auditEvent(ctx, "passkey_rename_failed", "", &req.UserID, false, "rename_failed")
		return nil, fmt.Errorf("rename passkey: %w", err)
	}
	credential, err := s.passkeys.GetCredentialByIDForUser(ctx, strings.TrimSpace(req.ID), strings.TrimSpace(req.UserID))
	if err != nil {
		return nil, fmt.Errorf("get renamed passkey: %w", err)
	}
	s.auditEvent(ctx, "passkey_renamed", "", &req.UserID, true, "")
	return credential, nil
}

// Delete revokes a passkey.
func (s *PasskeyService) Delete(ctx context.Context, req PasskeyDeleteRequest) error {
	userID := strings.TrimSpace(req.UserID)
	if err := s.passkeys.RevokeCredential(ctx, strings.TrimSpace(req.ID), userID, s.now()); err != nil {
		s.auditEvent(ctx, "passkey_delete_failed", "", &userID, false, "delete_failed")
		return fmt.Errorf("delete passkey: %w", err)
	}
	s.auditEvent(ctx, "passkey_deleted", "", &userID, true, "")
	return nil
}

func (s *PasskeyService) storeChallenge(ctx context.Context, userID, ceremony string, session *webauthn.SessionData) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal webauthn session: %w", err)
	}
	now := s.now()
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}
	record := &models.WebAuthnChallenge{
		ID:          uuid.NewString(),
		UserID:      userIDPtr,
		Ceremony:    ceremony,
		Challenge:   session.Challenge,
		SessionData: data,
		ExpiresAt:   now.Add(s.challengeTTL),
		CreatedAt:   now,
	}
	if !session.Expires.IsZero() && session.Expires.Before(record.ExpiresAt) {
		record.ExpiresAt = session.Expires
	}
	if err := s.passkeys.CreateChallenge(ctx, record); err != nil {
		return fmt.Errorf("store webauthn challenge: %w", err)
	}
	return nil
}

func (s *PasskeyService) consumeChallenge(ctx context.Context, ceremony, challenge string, userID *string) (*models.WebAuthnChallenge, error) {
	record, err := s.passkeys.ConsumeChallenge(ctx, ceremony, challenge, userID, s.now())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, validationError("invalid passkey challenge")
		}
		return nil, fmt.Errorf("consume webauthn challenge: %w", err)
	}
	return record, nil
}

func (s *PasskeyService) auditEvent(ctx context.Context, eventType, email string, userID *string, success bool, reason string) {
	if s.audit == nil {
		return
	}
	event := &models.AuthAuditEvent{
		ID:        uuid.NewString(),
		UserID:    userID,
		EventType: eventType,
		Email:     email,
		Success:   success,
		Reason:    reason,
		CreatedAt: s.now(),
	}
	if err := s.audit.Create(ctx, event); err == nil {
		recordAuthEventMetric(eventType, success, reason)
	}
}

func decodeSessionData(data []byte) (webauthn.SessionData, error) {
	var session webauthn.SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return session, fmt.Errorf("decode webauthn session: %w", err)
	}
	return session, nil
}

func credentialModelFromWebAuthn(credential *webauthn.Credential, userID, name string, now time.Time) *models.PasskeyCredential {
	transports := make([]string, 0, len(credential.Transport))
	for _, transport := range credential.Transport {
		transports = append(transports, string(transport))
	}
	var aaguid *string
	if len(credential.Authenticator.AAGUID) == 16 {
		if parsed, err := uuid.FromBytes(credential.Authenticator.AAGUID); err == nil {
			value := parsed.String()
			aaguid = &value
		}
	}
	return &models.PasskeyCredential{
		ID:              uuid.NewString(),
		UserID:          userID,
		CredentialID:    credential.ID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		Transports:      transports,
		SignCount:       credential.Authenticator.SignCount,
		CloneWarning:    credential.Authenticator.CloneWarning,
		BackupEligible:  credential.Flags.BackupEligible,
		BackupState:     credential.Flags.BackupState,
		Name:            name,
		AAGUID:          aaguid,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func webAuthnCredentialFromModel(credential *models.PasskeyCredential) webauthn.Credential {
	transports := make([]protocol.AuthenticatorTransport, 0, len(credential.Transports))
	for _, transport := range credential.Transports {
		transports = append(transports, protocol.AuthenticatorTransport(transport))
	}
	var aaguid []byte
	if credential.AAGUID != nil {
		if parsed, err := uuid.Parse(*credential.AAGUID); err == nil {
			aaguid = parsed[:]
		}
	}
	return webauthn.Credential{
		ID:              credential.CredentialID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		Transport:       transports,
		Flags: webauthn.CredentialFlags{
			BackupEligible: credential.BackupEligible,
			BackupState:    credential.BackupState,
		},
		Authenticator: webauthn.Authenticator{
			AAGUID:       aaguid,
			SignCount:    credential.SignCount,
			CloneWarning: credential.CloneWarning,
		},
	}
}

type passkeyUser struct {
	user        *models.User
	credentials []models.PasskeyCredential
}

func (u passkeyUser) WebAuthnID() []byte {
	return []byte(u.user.ID)
}

func (u passkeyUser) WebAuthnName() string {
	return u.user.Email
}

func (u passkeyUser) WebAuthnDisplayName() string {
	return u.user.Email
}

func (u passkeyUser) WebAuthnCredentials() []webauthn.Credential {
	credentials := make([]webauthn.Credential, 0, len(u.credentials))
	for i := range u.credentials {
		if u.credentials[i].IsActive() {
			credentials = append(credentials, webAuthnCredentialFromModel(&u.credentials[i]))
		}
	}
	return credentials
}

func normalizePasskeyName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", validationError("passkey name is required")
	}
	if len(name) > 80 {
		return "", validationError("passkey name is too long")
	}
	return name, nil
}

func defaultPasskeyName(now time.Time) string {
	return "Passkey " + now.Format("2006-01-02")
}
