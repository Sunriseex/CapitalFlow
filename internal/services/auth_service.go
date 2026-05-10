package services

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/capitalflow/internal/auth"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/pkg/security"
)

type AuthService struct {
	users        repository.UserRepository
	refresh      repository.RefreshTokenRepository
	tokens       *auth.TokenService
	passwordFunc func(string, security.PasswordParams) (string, error)
	verifyFunc   func(string, string) (bool, error)
	now          func() time.Time
}

func NewAuthService(users repository.UserRepository, refresh repository.RefreshTokenRepository, tokens *auth.TokenService) *AuthService {
	return &AuthService{
		users:        users,
		refresh:      refresh,
		tokens:       tokens,
		passwordFunc: security.HashPassword,
		verifyFunc:   security.VerifyPassword,
		now:          time.Now,
	}
}

type AuthRequest struct {
	Email           string
	Password        string
	PrimaryCurrency string
}

type AuthSession struct {
	User             *models.User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

func (s *AuthService) Setup(ctx context.Context, req AuthRequest) (*AuthSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("setup auth: %w", err)
	}
	if s.users == nil || s.refresh == nil || s.tokens == nil {
		return nil, fmt.Errorf("auth service is not configured")
	}

	count, err := s.users.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil, validationError("setup is already complete")
	}

	user, err := s.buildUser(req)
	if err != nil {
		return nil, err
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	session, err := s.issueSession(ctx, user)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (s *AuthService) Login(ctx context.Context, req AuthRequest) (*AuthSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	if s.users == nil || s.refresh == nil || s.tokens == nil {
		return nil, fmt.Errorf("auth service is not configured")
	}

	email, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Password) == "" {
		return nil, validationError("invalid email or password")
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, validationError("invalid email or password")
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	ok, err := s.verifyFunc(req.Password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return nil, validationError("invalid email or password")
	}

	return s.issueSession(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (*AuthSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("refresh session: %w", err)
	}
	if s.users == nil || s.refresh == nil || s.tokens == nil {
		return nil, fmt.Errorf("auth service is not configured")
	}

	rawRefreshToken = strings.TrimSpace(rawRefreshToken)
	if rawRefreshToken == "" {
		return nil, validationError("refresh token is required")
	}

	now := s.now()
	token, err := s.refresh.GetByHash(ctx, auth.HashRefreshToken(rawRefreshToken))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, validationError("invalid refresh token")
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	if !token.IsActive(now) {
		if token.RevokedAt != nil {
			_ = s.refresh.RevokeByUser(ctx, token.UserID, now)
		}
		return nil, validationError("invalid refresh token")
	}

	if err := s.refresh.Revoke(ctx, token.ID, now); err != nil {
		return nil, fmt.Errorf("revoke refresh token: %w", err)
	}

	user, err := s.users.GetByID(ctx, token.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	return s.issueSession(ctx, user)
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
		return nil
	}

	token, err := s.refresh.GetByHash(ctx, auth.HashRefreshToken(rawRefreshToken))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("get refresh token: %w", err)
	}
	return s.refresh.Revoke(ctx, token.ID, s.now())
}

func (s *AuthService) buildUser(req AuthRequest) (*models.User, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if len(req.Password) < 12 {
		return nil, validationError("password must be at least 12 characters")
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

func (s *AuthService) issueSession(ctx context.Context, user *models.User) (*AuthSession, error) {
	now := s.now()
	pair, err := s.tokens.IssuePair(user.ID, user.Email, now)
	if err != nil {
		return nil, fmt.Errorf("issue tokens: %w", err)
	}

	refreshToken := &models.RefreshToken{
		ID:        pair.RefreshTokenID,
		UserID:    user.ID,
		TokenHash: pair.RefreshTokenHash,
		ExpiresAt: pair.RefreshExpiresAt,
		CreatedAt: now,
	}
	if err := s.refresh.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &AuthSession{
		User:             user,
		AccessToken:      pair.AccessToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshToken:     pair.RefreshToken,
		RefreshExpiresAt: pair.RefreshExpiresAt,
	}, nil
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", validationError("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", validationError("invalid email")
	}
	return email, nil
}

func normalizePrimaryCurrency(currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return "RUB"
	}
	return currency
}

func validateCurrency(currency string) error {
	if len(currency) != 3 {
		return validationError("invalid currency: " + currency)
	}
	for _, r := range currency {
		if r < 'A' || r > 'Z' {
			return validationError("invalid currency: " + currency)
		}
	}
	return nil
}
