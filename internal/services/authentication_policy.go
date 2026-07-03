package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/capitalflow/internal/auth"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/pkg/security"
)

// AuthenticationPolicy owns lockout, session issuance, and auth auditing for
// every credential mechanism.
type AuthenticationPolicy struct {
	users      repository.UserRepository
	refresh    repository.RefreshTokenRepository
	tokens     *auth.TokenService
	audit      repository.AuthAuditRepository
	verifyFunc func(string, string) (bool, error)
	now        func() time.Time
}

type AuthSession struct {
	User             *models.User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// PasswordConfirmationResult describes password confirmation state for sensitive authenticated actions.
type PasswordConfirmationResult struct {
	OK     bool
	Locked bool
}

func NewAuthenticationPolicy(users repository.UserRepository, refresh repository.RefreshTokenRepository, tokens *auth.TokenService, audit repository.AuthAuditRepository) *AuthenticationPolicy {
	return &AuthenticationPolicy{
		users: users, refresh: refresh, tokens: tokens, audit: audit,
		verifyFunc: security.VerifyPassword,
		now:        time.Now,
	}
}

func (p *AuthenticationPolicy) configured() bool {
	return p != nil && p.users != nil && p.refresh != nil && p.tokens != nil
}

func (p *AuthenticationPolicy) ConfirmPassword(ctx context.Context, user *models.User, password string) (PasswordConfirmationResult, error) {
	if err := ctx.Err(); err != nil {
		return PasswordConfirmationResult{}, fmt.Errorf("verify password confirmation: %w", err)
	}
	if p == nil || p.users == nil {
		return PasswordConfirmationResult{}, fmt.Errorf("authentication policy is not configured")
	}
	if user == nil {
		return PasswordConfirmationResult{}, validationError("invalid password confirmation")
	}
	now := p.now()
	if p.UserLocked(user) {
		return PasswordConfirmationResult{Locked: true}, nil
	}
	ok, err := p.verifyFunc(password, user.PasswordHash)
	if err != nil {
		return PasswordConfirmationResult{}, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		_, lockedUntil, err := p.users.RecordLoginFailure(ctx, user.ID, loginLockoutThreshold, loginLockoutDelays, now)
		if err != nil {
			return PasswordConfirmationResult{}, fmt.Errorf("record login failure: %w", err)
		}
		return PasswordConfirmationResult{Locked: lockedUntil != nil}, nil
	}
	if err := p.clearLoginFailures(ctx, user, now); err != nil {
		return PasswordConfirmationResult{}, err
	}
	return PasswordConfirmationResult{OK: true}, nil
}

func (p *AuthenticationPolicy) IssueSessionForUser(ctx context.Context, userID string) (*AuthSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("issue session for user: %w", err)
	}
	if !p.configured() {
		return nil, fmt.Errorf("authentication policy is not configured")
	}
	user, err := p.users.GetByID(ctx, strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if p.UserLocked(user) {
		return nil, validationError("invalid email or password")
	}
	if err := p.clearLoginFailures(ctx, user, p.now()); err != nil {
		return nil, err
	}
	return p.issueSession(ctx, user)
}

func (p *AuthenticationPolicy) UserLocked(user *models.User) bool {
	return user != nil && user.LockedUntil != nil && p.now().Before(*user.LockedUntil)
}

func (p *AuthenticationPolicy) clearLoginFailures(ctx context.Context, user *models.User, now time.Time) error {
	if user.FailedLoginAttempts == 0 && user.LockedUntil == nil {
		return nil
	}
	if err := p.users.ClearLoginFailures(ctx, user.ID, now); err != nil {
		return fmt.Errorf("clear login failures: %w", err)
	}
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	user.UpdatedAt = now
	return nil
}

func (p *AuthenticationPolicy) issueSession(ctx context.Context, user *models.User) (*AuthSession, error) {
	session, refreshToken, err := p.buildSession(user)
	if err != nil {
		return nil, err
	}
	if err := p.refresh.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}
	return session, nil
}

func (p *AuthenticationPolicy) buildSession(user *models.User) (*AuthSession, *models.RefreshToken, error) {
	if p == nil || p.tokens == nil {
		return nil, nil, fmt.Errorf("authentication policy is not configured")
	}
	now := p.now()
	pair, err := p.tokens.IssuePair(user.ID, user.Email, now)
	if err != nil {
		return nil, nil, fmt.Errorf("issue tokens: %w", err)
	}
	refreshToken := &models.RefreshToken{
		ID: pair.RefreshTokenID, UserID: user.ID, TokenHash: pair.RefreshTokenHash,
		ExpiresAt: pair.RefreshExpiresAt, CreatedAt: now,
	}
	return &AuthSession{
		User: user, AccessToken: pair.AccessToken, AccessExpiresAt: pair.AccessExpiresAt,
		RefreshToken: pair.RefreshToken, RefreshExpiresAt: pair.RefreshExpiresAt,
	}, refreshToken, nil
}

func (p *AuthenticationPolicy) Audit(ctx context.Context, eventType, email string, userID *string, success bool, reason string) {
	recordAuthEventMetric(eventType, success, reason)
	if p == nil || p.audit == nil {
		return
	}
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	event := p.NewAuditEvent(eventType, email, userID, success, reason)
	if err := p.audit.Create(auditCtx, event); err != nil {
		slog.Warn("auth audit event was not persisted", "event_type", eventType, "error", err)
	}
}

func (p *AuthenticationPolicy) NewAuditEvent(eventType, email string, userID *string, success bool, reason string) *models.AuthAuditEvent {
	return &models.AuthAuditEvent{
		ID: uuid.NewString(), UserID: userID, EventType: eventType,
		Email: strings.ToLower(strings.TrimSpace(email)), Success: success, Reason: reason, CreatedAt: p.now(),
	}
}

func (p *AuthenticationPolicy) hasAuditRepository() bool {
	return p != nil && p.audit != nil
}
