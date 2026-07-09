package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	if err := insertUser(ctx, r.pool, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepository) Setup(ctx context.Context, user *models.User, refreshToken *models.RefreshToken, auditEvent *models.AuthAuditEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin auth setup: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('capitalflow:auth_setup', 0))`); err != nil {
		return fmt.Errorf("lock auth setup: %w", err)
	}
	var setupComplete bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users)`).Scan(&setupComplete); err != nil {
		return fmt.Errorf("check auth setup: %w", err)
	}
	if setupComplete {
		return repository.ErrConflict
	}

	if err := insertUser(ctx, tx, user); err != nil {
		return fmt.Errorf("create setup user: %w", err)
	}
	if err := claimUnownedAccounts(ctx, tx, user.ID); err != nil {
		return fmt.Errorf("claim setup accounts: %w", err)
	}
	if err := insertRefreshToken(ctx, tx, refreshToken); err != nil {
		return fmt.Errorf("create setup refresh token: %w", err)
	}
	if auditEvent != nil {
		if err := insertAuthAuditEvent(ctx, tx, auditEvent); err != nil {
			return fmt.Errorf("create setup audit event: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit auth setup: %w", err)
	}
	return nil
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return r.get(ctx, `
		SELECT id, email, password_hash, primary_currency,
			email_verified_at, email_verification_token_hash, email_verification_sent_at,
			failed_login_attempts, locked_until,
			created_at, updated_at
		FROM users
		WHERE lower(email) = lower($1)
	`, strings.TrimSpace(email))
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	return r.get(ctx, `
		SELECT id, email, password_hash, primary_currency,
			email_verified_at, email_verification_token_hash, email_verification_sent_at,
			failed_login_attempts, locked_until,
			created_at, updated_at
		FROM users
		WHERE id = $1
	`, id)
}

func (r *UserRepository) UpdatePrimaryCurrency(ctx context.Context, id, primaryCurrency string, updatedAt time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update user primary currency: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	before, err := scanUser(tx.QueryRow(ctx, `
		SELECT id, email, password_hash, primary_currency,
			email_verified_at, email_verification_token_hash, email_verification_sent_at,
			failed_login_attempts, locked_until,
			created_at, updated_at
		FROM users
		WHERE id = $1
		FOR UPDATE
	`, id))
	if err != nil {
		return fmt.Errorf("read user primary currency: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE users
		SET primary_currency = $2, updated_at = $3
		WHERE id = $1
	`, id, primaryCurrency, updatedAt)
	if err != nil {
		return fmt.Errorf("update user primary currency: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update user primary currency: %w", repository.ErrNotFound)
	}
	actorUserID := before.ID
	auditEvent, err := newAuditEventWithSummaries(
		&actorUserID, "settings.profile_updated", "user_settings", id,
		userSettingsAuditSummary(before.PrimaryCurrency),
		userSettingsAuditSummary(primaryCurrency),
	)
	if err != nil {
		return fmt.Errorf("build user primary currency audit event: %w", err)
	}
	if err := insertAuditEvent(ctx, tx, auditEvent); err != nil {
		return fmt.Errorf("create user primary currency audit event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update user primary currency: %w", err)
	}
	return nil
}

func userSettingsAuditSummary(primaryCurrency string) map[string]any {
	return map[string]any{
		"primary_currency": primaryCurrency,
	}
}

func (r *UserRepository) RecordLoginFailure(ctx context.Context, id string, threshold int, delays []time.Duration, updatedAt time.Time) (int, *time.Time, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("begin record login failure: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var attempts int
	if err := tx.QueryRow(ctx, `
		SELECT failed_login_attempts + 1
		FROM users
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&attempts); err != nil {
		return 0, nil, fmt.Errorf("read login failures: %w", mapNotFound(err))
	}

	lockedUntil := loginLockoutUntil(updatedAt, attempts, threshold, delays)
	_, err = tx.Exec(ctx, `
		UPDATE users
		SET failed_login_attempts = $2, locked_until = $3, updated_at = $4
		WHERE id = $1
	`, id, attempts, lockedUntil, updatedAt)
	if err != nil {
		return 0, nil, fmt.Errorf("record login failure: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, nil, fmt.Errorf("commit record login failure: %w", err)
	}
	return attempts, lockedUntil, nil
}

func (r *UserRepository) ClearLoginFailures(ctx context.Context, id string, updatedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET failed_login_attempts = 0, locked_until = NULL, updated_at = $2
		WHERE id = $1
	`, id, updatedAt)
	if err != nil {
		return fmt.Errorf("clear login failures: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $2, failed_login_attempts = 0, locked_until = NULL, updated_at = $3
		WHERE id = $1
	`, id, passwordHash, updatedAt)
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}
	return nil
}

func (r *UserRepository) ChangePasswordAndRevokeSessions(ctx context.Context, id, passwordHash string, updatedAt time.Time, revokedReason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin change password: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	tag, err := tx.Exec(ctx, `
		UPDATE users
		SET password_hash = $2, failed_login_attempts = 0, locked_until = NULL, updated_at = $3
		WHERE id = $1
	`, id, passwordHash, updatedAt)
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update user password: %w", repository.ErrNotFound)
	}

	_, err = tx.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = $2, revoked_reason = $3
		WHERE user_id = $1 AND revoked_at IS NULL
	`, id, updatedAt, revokedReason)
	if err != nil {
		return fmt.Errorf("revoke user refresh tokens: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit change password: %w", err)
	}
	return nil
}

func (r *UserRepository) get(ctx context.Context, query string, args ...any) (*models.User, error) {
	user, err := scanUser(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, fmt.Errorf("get user: %w", mapNotFound(err))
	}
	return user, nil
}

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	if err := insertRefreshToken(ctx, r.pool, token); err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) GetByID(ctx context.Context, id string) (*models.RefreshToken, error) {
	token, err := scanRefreshToken(r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at, revoked_reason, created_at
		FROM refresh_tokens
		WHERE id = $1
	`, id))
	if err != nil {
		return nil, fmt.Errorf("get refresh token by id: %w", mapNotFound(err))
	}
	return token, nil
}

func (r *RefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	token, err := scanRefreshToken(r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at, revoked_reason, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`, tokenHash))
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", mapNotFound(err))
	}
	return token, nil
}

func (r *RefreshTokenRepository) ListByUser(ctx context.Context, userID string) ([]models.RefreshToken, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, token_hash, expires_at, revoked_at, revoked_reason, created_at
		FROM refresh_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list refresh tokens: %w", err)
	}
	defer rows.Close()

	tokens := []models.RefreshToken{}
	for rows.Next() {
		token, err := scanRefreshToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, *token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list refresh tokens rows: %w", err)
	}
	return tokens, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id string, revokedAt time.Time, reason string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = $2, revoked_reason = $3
		WHERE id = $1 AND revoked_at IS NULL
	`, id, revokedAt, reason)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("revoke refresh token: %w", repository.ErrNotFound)
	}
	return nil
}

func (r *RefreshTokenRepository) Rotate(ctx context.Context, oldTokenID string, newToken *models.RefreshToken, revokedAt time.Time, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin refresh token rotation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = $2, revoked_reason = $3
		WHERE id = $1 AND revoked_at IS NULL
	`, oldTokenID, revokedAt, reason)
	if err != nil {
		return fmt.Errorf("revoke rotated refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("revoke rotated refresh token: %w", repository.ErrNotFound)
	}
	if err := insertRefreshToken(ctx, tx, newToken); err != nil {
		return fmt.Errorf("create rotated refresh token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit refresh token rotation: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeByUserSession(ctx context.Context, userID, id string, revokedAt time.Time, reason string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = $3, revoked_reason = $4
		WHERE user_id = $1 AND id = $2 AND revoked_at IS NULL
	`, userID, id, revokedAt, reason)
	if err != nil {
		return fmt.Errorf("revoke user refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("revoke user refresh token: %w", repository.ErrNotFound)
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeByUser(ctx context.Context, userID string, revokedAt time.Time, reason string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = $2, revoked_reason = $3
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID, revokedAt, reason)
	if err != nil {
		return fmt.Errorf("revoke user refresh tokens: %w", err)
	}
	return nil
}

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(row userScanner) (*models.User, error) {
	var user models.User
	if err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.PrimaryCurrency,
		&user.EmailVerifiedAt,
		&user.EmailVerificationTokenHash,
		&user.EmailVerificationSentAt,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan user: %w", mapNotFound(err))
	}
	return &user, nil
}

type refreshTokenScanner interface {
	Scan(dest ...any) error
}

func scanRefreshToken(row refreshTokenScanner) (*models.RefreshToken, error) {
	var token models.RefreshToken
	if err := row.Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt, &token.RevokedReason, &token.CreatedAt); err != nil {
		return nil, fmt.Errorf("scan refresh token: %w", mapNotFound(err))
	}
	return &token, nil
}

func loginLockoutUntil(now time.Time, attempts, threshold int, delays []time.Duration) *time.Time {
	if attempts < threshold || len(delays) == 0 {
		return nil
	}
	delayIndex := min(attempts-threshold, len(delays)-1)
	return new(now.Add(delays[delayIndex]))
}

type AuthAuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuthAuditRepository(pool *pgxpool.Pool) *AuthAuditRepository {
	return &AuthAuditRepository{pool: pool}
}

func (r *AuthAuditRepository) Create(ctx context.Context, event *models.AuthAuditEvent) error {
	if err := insertAuthAuditEvent(ctx, r.pool, event); err != nil {
		return fmt.Errorf("create auth audit event: %w", err)
	}
	return nil
}

func insertUser(ctx context.Context, execer sqlExecer, user *models.User) error {
	_, err := execer.Exec(ctx, `
		INSERT INTO users (
			id, email, password_hash, primary_currency,
			email_verified_at, email_verification_token_hash, email_verification_sent_at,
			failed_login_attempts, locked_until, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, user.ID, user.Email, user.PasswordHash, user.PrimaryCurrency, user.EmailVerifiedAt, user.EmailVerificationTokenHash, user.EmailVerificationSentAt, user.FailedLoginAttempts, user.LockedUntil, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return repository.ErrConflict
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func insertRefreshToken(ctx context.Context, execer sqlExecer, token *models.RefreshToken) error {
	_, err := execer.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, revoked_reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.RevokedAt, token.RevokedReason, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func insertAuthAuditEvent(ctx context.Context, execer sqlExecer, event *models.AuthAuditEvent) error {
	_, err := execer.Exec(ctx, `
		INSERT INTO auth_audit_events (id, user_id, event_type, email, success, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, event.ID, event.UserID, event.EventType, event.Email, event.Success, event.Reason, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert auth audit event: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
