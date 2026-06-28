package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type CategoryLimitRepository struct{ pool *pgxpool.Pool }

func NewCategoryLimitRepository(pool *pgxpool.Pool) *CategoryLimitRepository {
	return &CategoryLimitRepository{pool: pool}
}

func (r *CategoryLimitRepository) Create(ctx context.Context, limit *models.CategoryLimit) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO category_limits
			(id, owner_user_id, category_id, amount, currency, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, limit.ID, limit.OwnerUserID, limit.CategoryID, limit.Amount, limit.Currency, limit.IsActive, limit.CreatedAt, limit.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create category limit: %w", mapConflict(err))
	}
	return nil
}

func (r *CategoryLimitRepository) GetByIDForUser(ctx context.Context, id, userID string) (*models.CategoryLimit, error) {
	limit, err := scanCategoryLimit(r.pool.QueryRow(ctx, `
		SELECT id, owner_user_id, category_id, amount, currency, is_active, created_at, updated_at
		FROM category_limits WHERE id = $1 AND owner_user_id = $2
	`, id, userID))
	if err != nil {
		return nil, fmt.Errorf("get category limit: %w", mapNotFound(err))
	}
	return limit, nil
}

func (r *CategoryLimitRepository) ListByUser(ctx context.Context, userID string) ([]models.CategoryLimit, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, owner_user_id, category_id, amount, currency, is_active, created_at, updated_at
		FROM category_limits WHERE owner_user_id = $1 ORDER BY updated_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list category limits: %w", err)
	}
	defer rows.Close()

	limits := make([]models.CategoryLimit, 0)
	for rows.Next() {
		limit, err := scanCategoryLimit(rows)
		if err != nil {
			return nil, fmt.Errorf("scan category limit: %w", err)
		}
		limits = append(limits, *limit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list category limit rows: %w", err)
	}
	return limits, nil
}

func (r *CategoryLimitRepository) UpdateForUser(ctx context.Context, limit *models.CategoryLimit, userID string) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE category_limits
		SET category_id = $1, amount = $2, currency = $3, is_active = $4, updated_at = $5
		WHERE id = $6 AND owner_user_id = $7
	`, limit.CategoryID, limit.Amount, limit.Currency, limit.IsActive, limit.UpdatedAt, limit.ID, userID)
	if err != nil {
		return fmt.Errorf("update category limit: %w", mapConflict(err))
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("update category limit: %w", repository.ErrNotFound)
	}
	return nil
}

type categoryLimitScanner interface {
	Scan(dest ...any) error
}

func scanCategoryLimit(row categoryLimitScanner) (*models.CategoryLimit, error) {
	var limit models.CategoryLimit
	if err := row.Scan(
		&limit.ID,
		&limit.OwnerUserID,
		&limit.CategoryID,
		&limit.Amount,
		&limit.Currency,
		&limit.IsActive,
		&limit.CreatedAt,
		&limit.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan category limit row: %w", err)
	}
	return &limit, nil
}
