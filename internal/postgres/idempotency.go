package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type IdempotencyRepository struct {
	pool queryExecer
}

func NewIdempotencyRepository(pool queryExecer) *IdempotencyRepository {
	return &IdempotencyRepository{pool: pool}
}

func (r *IdempotencyRepository) Get(ctx context.Context, key, userID, method, path string) (*models.IdempotencyRecord, error) {
	var record models.IdempotencyRecord
	if err := r.pool.QueryRow(ctx, `
		SELECT id, key, user_id, method, path, endpoint, request_hash, status, status_code,
			response_body, locked_until, created_at, updated_at, expires_at
		FROM idempotency_keys
		WHERE key = $1 AND user_id = $2 AND method = $3 AND path = $4 AND expires_at > now()
	`, key, userID, method, path).Scan(
		&record.ID,
		&record.Key,
		&record.UserID,
		&record.Method,
		&record.Path,
		&record.Endpoint,
		&record.RequestHash,
		&record.Status,
		&record.StatusCode,
		&record.ResponseBody,
		&record.LockedUntil,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.ExpiresAt,
	); err != nil {
		return nil, fmt.Errorf("get idempotency key: %w", mapNotFound(err))
	}
	return &record, nil
}

func (r *IdempotencyRepository) CreatePending(ctx context.Context, record *models.IdempotencyRecord) (bool, error) {
	now := record.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	record.Endpoint = record.Method + " " + record.Path
	record.Status = "pending"
	record.CreatedAt = now
	record.UpdatedAt = now
	lockedUntil := now.Add(30 * time.Second)
	record.LockedUntil = &lockedUntil
	if _, err := r.pool.Exec(ctx, `DELETE FROM idempotency_keys WHERE expires_at <= now()`); err != nil {
		return false, fmt.Errorf("delete expired idempotency keys: %w", err)
	}

	tag, err := r.pool.Exec(ctx, `
		INSERT INTO idempotency_keys (id, key, user_id, method, path, endpoint, request_hash, status, locked_until, created_at, updated_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $9, $10, $11)
		ON CONFLICT (key, user_id, method, path) DO UPDATE
		SET id = EXCLUDED.id,
			request_hash = EXCLUDED.request_hash,
			status = 'pending',
			status_code = NULL,
			response_body = NULL,
			locked_until = EXCLUDED.locked_until,
			created_at = EXCLUDED.created_at,
			updated_at = EXCLUDED.updated_at,
			expires_at = EXCLUDED.expires_at
		WHERE idempotency_keys.status = 'pending'
			AND idempotency_keys.locked_until <= now()
			AND idempotency_keys.request_hash = EXCLUDED.request_hash
	`, record.ID, record.Key, record.UserID, record.Method, record.Path, record.Endpoint, record.RequestHash, record.LockedUntil, record.CreatedAt, record.UpdatedAt, record.ExpiresAt)
	if err != nil {
		return false, fmt.Errorf("create idempotency key: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *IdempotencyRepository) Complete(ctx context.Context, recordID, key, userID, method, path string, statusCode int, responseBody []byte) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE idempotency_keys
		SET status = 'completed', status_code = $6, response_body = $7, locked_until = NULL, updated_at = now()
		WHERE id = $1 AND key = $2 AND user_id = $3 AND method = $4 AND path = $5 AND status = 'pending'
	`, recordID, key, userID, method, path, statusCode, responseBody)
	if err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("complete idempotency key: %w", repository.ErrNotFound)
	}
	return nil
}
