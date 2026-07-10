package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type FinancialGoalRepository struct{ pool *pgxpool.Pool }

func NewFinancialGoalRepository(pool *pgxpool.Pool) *FinancialGoalRepository {
	return &FinancialGoalRepository{pool: pool}
}

func (r *FinancialGoalRepository) Create(ctx context.Context, goal *models.FinancialGoal) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO financial_goals
			(id, owner_user_id, account_id, name, target_amount, currency, target_date, status, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, goal.ID, goal.OwnerUserID, goal.AccountID, goal.Name, goal.TargetAmount, goal.Currency, goal.TargetDate, goal.Status, goal.CreatedAt, goal.UpdatedAt, goal.Version)
	if err != nil {
		return fmt.Errorf("create financial goal: %w", err)
	}
	return nil
}

func (r *FinancialGoalRepository) GetByIDForUser(ctx context.Context, id, userID string) (*models.FinancialGoal, error) {
	goal, err := scanFinancialGoal(r.pool.QueryRow(ctx, `
		SELECT id, owner_user_id, account_id, name, target_amount, currency, target_date, status, created_at, updated_at, version
		FROM financial_goals WHERE id = $1 AND owner_user_id = $2
	`, id, userID))
	if err != nil {
		return nil, fmt.Errorf("get financial goal: %w", mapNotFound(err))
	}
	return goal, nil
}

func (r *FinancialGoalRepository) ListByUser(ctx context.Context, userID string) ([]models.FinancialGoal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, owner_user_id, account_id, name, target_amount, currency, target_date, status, created_at, updated_at, version
		FROM financial_goals WHERE owner_user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list financial goals: %w", err)
	}
	defer rows.Close()

	goals := make([]models.FinancialGoal, 0)
	for rows.Next() {
		goal, err := scanFinancialGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("scan financial goal: %w", err)
		}
		goals = append(goals, *goal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list financial goals rows: %w", err)
	}
	return goals, nil
}

func (r *FinancialGoalRepository) UpdateForUser(ctx context.Context, goal *models.FinancialGoal, userID string) error {
	err := r.pool.QueryRow(ctx, `
		UPDATE financial_goals
		SET account_id = $1, name = $2, target_amount = $3, currency = $4,
			target_date = $5, status = $6, updated_at = $7, version = version + 1
		WHERE id = $8 AND owner_user_id = $9 AND version = $10
		RETURNING version
	`, goal.AccountID, goal.Name, goal.TargetAmount, goal.Currency, goal.TargetDate, goal.Status, goal.UpdatedAt, goal.ID, userID, goal.Version).Scan(&goal.Version)
	if err != nil {
		if mapped := mapNotFound(err); mapped == repository.ErrNotFound {
			return fmt.Errorf("update financial goal: %w", repository.ErrConflict)
		}
		return fmt.Errorf("update financial goal: %w", err)
	}
	return nil
}

type financialGoalScanner interface {
	Scan(dest ...any) error
}

func scanFinancialGoal(row financialGoalScanner) (*models.FinancialGoal, error) {
	var goal models.FinancialGoal
	if err := row.Scan(
		&goal.ID,
		&goal.OwnerUserID,
		&goal.AccountID,
		&goal.Name,
		&goal.TargetAmount,
		&goal.Currency,
		&goal.TargetDate,
		&goal.Status,
		&goal.CreatedAt,
		&goal.UpdatedAt,
		&goal.Version,
	); err != nil {
		return nil, fmt.Errorf("scan financial goal row: %w", err)
	}
	return &goal, nil
}
