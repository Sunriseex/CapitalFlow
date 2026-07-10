package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/capitalflow/internal/auth"
	domainaccount "github.com/sunriseex/capitalflow/internal/domain/account"
	domainauth "github.com/sunriseex/capitalflow/internal/domain/auth"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/pkg/security"
)

const loginLockoutThreshold = 5

const (
	refreshRevokedReasonLogout         = "logout"
	refreshRevokedReasonManual         = "manual"
	refreshRevokedReasonPasswordChange = "password_change"
	refreshRevokedReasonReuseDetected  = "reuse_detected"
	refreshRevokedReasonRotated        = "rotated"
)

var loginLockoutDelays = []time.Duration{
	5 * time.Minute,
	15 * time.Minute,
	time.Hour,
	6 * time.Hour,
	24 * time.Hour,
}

// dummyPasswordHash has the production Argon2 parameters and is used to keep
// unknown-email login attempts close to the cost of real password checks.
const dummyPasswordHash = "$argon2id$v=19$m=65536,t=3,p=2$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" //nolint:gosec // Public non-secret timing equalizer.

type AuthService struct {
	users          repository.UserRepository
	refresh        repository.RefreshTokenRepository
	accounts       repository.AccountRepository
	authentication *AuthenticationPolicy
	passwordFunc   func(string, security.PasswordParams) (string, error)
	verifyFunc     func(string, string) (bool, error)
	now            func() time.Time
}

func NewAuthService(users repository.UserRepository, refresh repository.RefreshTokenRepository, tokens *auth.TokenService, audit ...repository.AuthAuditRepository) *AuthService {
	var auditRepo repository.AuthAuditRepository
	if len(audit) > 0 {
		auditRepo = audit[0]
	}

	service := &AuthService{
		users:        users,
		refresh:      refresh,
		passwordFunc: security.HashPassword,
		verifyFunc:   security.VerifyPassword,
		now:          time.Now,
	}
	service.authentication = NewAuthenticationPolicy(users, refresh, tokens, auditRepo)
	service.authentication.now = func() time.Time { return service.now() }
	service.authentication.verifyFunc = func(password, hash string) (bool, error) {
		return service.verifyFunc(password, hash)
	}
	return service
}

func (s *AuthService) AuthenticationPolicy() *AuthenticationPolicy {
	if s == nil {
		return nil
	}
	return s.authentication
}

func (s *AuthService) WithAccountRepository(repo repository.AccountRepository) *AuthService {
	s.accounts = repo
	return s
}

type AuthRequest struct {
	Email           string
	Password        string
	PrimaryCurrency string
}

type ChangePasswordRequest struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

type SessionInfo struct {
	ID        string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
	Active    bool
	Current   bool
}

func (s *AuthService) Setup(ctx context.Context, req AuthRequest) (*AuthSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("setup auth: %w", err)
	}
	if s.users == nil || s.refresh == nil || !s.authentication.configured() {
		return nil, fmt.Errorf("auth service is not configured")
	}

	count, err := s.users.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		s.authentication.Audit(ctx, "setup_failed", req.Email, nil, false, "setup_complete")
		return nil, validationError("setup is already complete")
	}

	user, err := s.buildUser(req)
	if err != nil {
		s.authentication.Audit(ctx, "setup_failed", req.Email, nil, false, "validation_error")
		return nil, err
	}
	if setupRepo, ok := s.users.(repository.AuthSetupRepository); ok {
		session, refreshToken, err := s.authentication.buildSession(user)
		if err != nil {
			s.authentication.Audit(ctx, "setup_failed", req.Email, &user.ID, false, "issue_session_failed")
			return nil, err
		}

		var auditEvent *models.AuthAuditEvent
		if s.authentication.hasAuditRepository() {
			auditEvent = s.authentication.NewAuditEvent("setup_success", user.Email, &user.ID, true, "")
		}
		if err := setupRepo.Setup(ctx, user, refreshToken, auditEvent); err != nil {
			if errors.Is(err, repository.ErrConflict) {
				s.authentication.Audit(ctx, "setup_failed", req.Email, nil, false, "setup_complete")
				return nil, validationError("setup is already complete")
			}
			s.authentication.Audit(ctx, "setup_failed", req.Email, &user.ID, false, "setup_transaction_failed")
			return nil, fmt.Errorf("setup auth transaction: %w", err)
		}
		recordAuthEventMetric("setup_success", true, "")
		return session, nil
	}

	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			s.authentication.Audit(ctx, "setup_failed", req.Email, nil, false, "setup_complete")
			return nil, validationError("setup is already complete")
		}
		s.authentication.Audit(ctx, "setup_failed", req.Email, nil, false, "save_failed")
		return nil, fmt.Errorf("save user: %w", err)
	}

	if s.accounts != nil {
		if err := s.accounts.ClaimUnowned(ctx, user.ID); err != nil {
			s.authentication.Audit(ctx, "setup_failed", req.Email, &user.ID, false, "claim_unowned_accounts_failed")
			return nil, fmt.Errorf("claim unowned accounts: %w", err)
		}
	}

	session, err := s.authentication.issueSession(ctx, user)
	if err != nil {
		s.authentication.Audit(ctx, "setup_failed", req.Email, &user.ID, false, "issue_session_failed")
		return nil, err
	}
	s.authentication.Audit(ctx, "setup_success", user.Email, &user.ID, true, "")
	return session, nil
}

func (s *AuthService) Login(ctx context.Context, req AuthRequest) (*AuthSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	if s.users == nil || s.refresh == nil || !s.authentication.configured() {
		return nil, fmt.Errorf("auth service is not configured")
	}

	email, err := normalizeEmail(req.Email)
	if err != nil {
		s.authentication.Audit(ctx, "login_failed", req.Email, nil, false, "validation_error")
		return nil, err
	}
	if strings.TrimSpace(req.Password) == "" {
		s.authentication.Audit(ctx, "login_failed", email, nil, false, "invalid_credentials")
		return nil, validationError("invalid email or password")
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			if _, verifyErr := s.verifyFunc(req.Password, dummyPasswordHash); verifyErr != nil {
				return nil, fmt.Errorf("verify dummy password: %w", verifyErr)
			}
			s.authentication.Audit(ctx, "login_failed", email, nil, false, "invalid_credentials")
			return nil, validationError("invalid email or password")
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	confirmation, err := s.authentication.ConfirmPassword(ctx, user, req.Password)
	if err != nil {
		return nil, err
	}
	if !confirmation.OK {
		reason := "invalid_credentials"
		if confirmation.Locked {
			reason = "account_locked"
		}
		s.authentication.Audit(ctx, "login_failed", email, &user.ID, false, reason)
		return nil, validationError("invalid email or password")
	}

	session, err := s.authentication.issueSession(ctx, user)
	if err != nil {
		s.authentication.Audit(ctx, "login_failed", email, &user.ID, false, "issue_session_failed")
		return nil, err
	}
	s.authentication.Audit(ctx, "login_success", email, &user.ID, true, "")
	return session, nil
}

func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (*AuthSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("refresh session: %w", err)
	}
	if s.users == nil || s.refresh == nil || !s.authentication.configured() {
		return nil, fmt.Errorf("auth service is not configured")
	}

	rawRefreshToken = strings.TrimSpace(rawRefreshToken)
	if rawRefreshToken == "" {
		s.authentication.Audit(ctx, "refresh_failed", "", nil, false, "missing_refresh_token")
		return nil, validationError("refresh token is required")
	}

	now := s.now()
	token, err := s.refresh.GetByHash(ctx, auth.HashRefreshToken(rawRefreshToken))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.authentication.Audit(ctx, "refresh_failed", "", nil, false, "invalid_refresh_token")
			return nil, validationError("invalid refresh token")
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	if !token.IsActive(now) {
		if isRefreshTokenReuseCandidate(token) {
			if err := s.refresh.RevokeByUser(ctx, token.UserID, now, refreshRevokedReasonReuseDetected); err != nil {
				return nil, fmt.Errorf("revoke refresh token family: %w", err)
			}

			s.authentication.Audit(ctx, "refresh_reuse_detected", "", &token.UserID, false, "revoked_refresh_token_reused")
			return nil, validationError("invalid refresh token")
		}

		s.authentication.Audit(ctx, "refresh_failed", "", &token.UserID, false, "inactive_refresh_token")
		return nil, validationError("invalid refresh token")
	}

	user, err := s.users.GetByID(ctx, token.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	var session *AuthSession
	if rotator, ok := s.refresh.(repository.RefreshTokenRotator); ok {
		var refreshToken *models.RefreshToken
		session, refreshToken, err = s.authentication.buildSession(user)
		if err == nil {
			err = rotator.Rotate(ctx, token.ID, refreshToken, now, refreshRevokedReasonRotated)
		}
	} else {
		err = s.refresh.Revoke(ctx, token.ID, now, refreshRevokedReasonRotated)
		if err == nil {
			session, err = s.authentication.issueSession(ctx, user)
		}
	}
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.authentication.Audit(ctx, "refresh_failed", "", &token.UserID, false, "refresh_token_already_rotated")
			return nil, validationError("invalid refresh token")
		}
		s.authentication.Audit(ctx, "refresh_failed", user.Email, &user.ID, false, "issue_session_failed")
		return nil, fmt.Errorf("rotate refresh token: %w", err)
	}
	s.authentication.Audit(ctx, "refresh_success", user.Email, &user.ID, true, "")
	return session, nil
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	if s.refresh == nil {
		return fmt.Errorf("auth service is not configured")
	}

	rawRefreshToken = strings.TrimSpace(rawRefreshToken)
	if rawRefreshToken == "" {
		s.authentication.Audit(ctx, "logout", "", nil, true, "missing_refresh_token")
		return nil
	}

	token, err := s.refresh.GetByHash(ctx, auth.HashRefreshToken(rawRefreshToken))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.authentication.Audit(ctx, "logout", "", nil, true, "unknown_refresh_token")
			return nil
		}
		return fmt.Errorf("get refresh token: %w", err)
	}
	if err := s.refresh.Revoke(ctx, token.ID, s.now(), refreshRevokedReasonLogout); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	s.authentication.Audit(ctx, "logout", "", &token.UserID, true, "")
	return nil
}

func (s *AuthService) ChangePassword(ctx context.Context, req ChangePasswordRequest) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("change password: %w", err)
	}
	if s.users == nil {
		return fmt.Errorf("auth service is not configured")
	}

	userID := strings.TrimSpace(req.UserID)
	if userID == "" {
		return validationError("user is required")
	}
	if strings.TrimSpace(req.CurrentPassword) == "" {
		s.authentication.Audit(ctx, "change_password_failed", "", &userID, false, "invalid_current_password")
		return validationError("current password is required")
	}
	if strings.TrimSpace(req.NewPassword) == "" {
		s.authentication.Audit(ctx, "change_password_failed", "", &userID, false, "validation_error")
		return validationError("new password is required")
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	ok, err := s.verifyFunc(req.CurrentPassword, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		s.authentication.Audit(ctx, "change_password_failed", user.Email, &user.ID, false, "invalid_current_password")
		return validationError("invalid current password")
	}
	if req.CurrentPassword == req.NewPassword {
		s.authentication.Audit(ctx, "change_password_failed", user.Email, &user.ID, false, "password_reuse")
		return validationError("new password must be different")
	}
	if err := validatePasswordPolicy(req.NewPassword, user.Email); err != nil {
		s.authentication.Audit(ctx, "change_password_failed", user.Email, &user.ID, false, "validation_error")
		return err
	}

	hash, err := s.passwordFunc(req.NewPassword, security.DefaultPasswordParams())
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	now := s.now()
	if err := s.users.ChangePasswordAndRevokeSessions(ctx, user.ID, hash, now, refreshRevokedReasonPasswordChange); err != nil {
		s.authentication.Audit(ctx, "change_password_failed", user.Email, &user.ID, false, "save_failed")
		return fmt.Errorf("change password and revoke sessions: %w", err)
	}
	s.authentication.Audit(ctx, "change_password_success", user.Email, &user.ID, true, "")
	return nil
}

func (s *AuthService) ListSessions(ctx context.Context, userID, currentRefreshTokenID string) ([]SessionInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	if s.refresh == nil {
		return nil, fmt.Errorf("auth service is not configured")
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, validationError("user is required")
	}

	tokens, err := s.refresh.ListByUser(ctx, userID)
	if err != nil {
		s.authentication.Audit(ctx, "sessions_list_failed", "", &userID, false, "list_failed")
		return nil, fmt.Errorf("list refresh tokens: %w", err)
	}

	now := s.now()
	sessions := make([]SessionInfo, 0, len(tokens))
	for _, token := range tokens {
		sessions = append(sessions, SessionInfo{
			ID:        token.ID,
			ExpiresAt: token.ExpiresAt,
			RevokedAt: token.RevokedAt,
			CreatedAt: token.CreatedAt,
			Active:    token.IsActive(now),
			Current:   token.ID == currentRefreshTokenID,
		})
	}
	s.authentication.Audit(ctx, "sessions_listed", "", &userID, true, "")
	return sessions, nil
}

func (s *AuthService) SetupRequired(ctx context.Context) (bool, error) {
	if s == nil || s.users == nil {
		return false, fmt.Errorf("user repository is required")
	}
	count, err := s.users.Count(ctx)
	if err != nil {
		return false, fmt.Errorf("count users: %w", err)
	}
	return count == 0, nil
}

func (s *AuthService) ValidateSession(ctx context.Context, userID, sessionID string, now time.Time) (bool, error) {
	if s == nil || s.refresh == nil {
		return false, fmt.Errorf("refresh token repository is required")
	}
	session, err := s.refresh.GetByID(ctx, strings.TrimSpace(sessionID))
	if errors.Is(err, repository.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get session: %w", err)
	}
	return session.UserID == strings.TrimSpace(userID) && session.IsActive(now), nil
}

func (s *AuthService) RevokeSession(ctx context.Context, userID, sessionID string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if s.refresh == nil {
		return fmt.Errorf("auth service is not configured")
	}

	userID = strings.TrimSpace(userID)
	sessionID = strings.TrimSpace(sessionID)
	if userID == "" {
		return validationError("user is required")
	}
	if sessionID == "" {
		s.authentication.Audit(ctx, "session_revoke_failed", "", &userID, false, "validation_error")
		return validationError("session is required")
	}

	if err := s.refresh.RevokeByUserSession(ctx, userID, sessionID, s.now(), refreshRevokedReasonManual); err != nil {
		reason := "revoke_failed"
		if errors.Is(err, repository.ErrNotFound) {
			reason = "session_not_found"
		}
		s.authentication.Audit(ctx, "session_revoke_failed", "", &userID, false, reason)
		return fmt.Errorf("revoke session: %w", err)
	}
	s.authentication.Audit(ctx, "session_revoked", "", &userID, true, "")
	return nil
}

func (s *AuthService) buildUser(req AuthRequest) (*models.User, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if err := validatePasswordPolicy(req.Password, email); err != nil {
		return nil, err
	}
	primaryCurrency := normalizePrimaryCurrency(req.PrimaryCurrency)
	if err := validateCurrency(primaryCurrency); err != nil {
		return nil, err
	}

	hash, err := s.passwordFunc(req.Password, security.DefaultPasswordParams())
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	now := s.now()

	return &models.User{
		ID:              uuid.NewString(),
		Email:           email,
		PasswordHash:    hash,
		PrimaryCurrency: primaryCurrency,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func normalizeEmail(email string) (string, error) {
	normalized, err := domainauth.NormalizeEmail(email)
	if err != nil {
		return "", validationError(err.Error())
	}
	return normalized, nil
}

func normalizePrimaryCurrency(currency string) string {
	currency = domainaccount.NormalizeCurrency(currency)
	if currency == "" {
		return "RUB"
	}
	return currency
}

func validateCurrency(currency string) error {
	if !domainaccount.ValidCurrency(currency) {
		return validationError("invalid currency: " + currency)
	}
	return nil
}

func validatePasswordPolicy(password, email string) error {
	if err := domainauth.ValidatePassword(password, email); err != nil {
		return validationError(err.Error())
	}
	return nil
}

func isRefreshTokenReuseCandidate(token *models.RefreshToken) bool {
	if token == nil || token.RevokedAt == nil {
		return false
	}

	if token.RevokedReason == nil {
		return true
	}

	return *token.RevokedReason == refreshRevokedReasonRotated
}
