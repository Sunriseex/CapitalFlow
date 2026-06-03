package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sunriseex/capitalflow/internal/repository"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Accounts() repository.AccountRepository {
	return NewAccountRepository(s.pool)
}

func (s *Store) Transactions() repository.TransactionRepository {
	return NewTransactionRepository(s.pool)
}

func (s *Store) Categories() repository.CategoryRepository {
	return NewCategoryRepository(s.pool)
}

func (s *Store) InterestRules() repository.InterestRuleRepository {
	return NewInterestRuleRepository(s.pool)
}

func (s *Store) InterestAccruals() repository.InterestAccrualRepository {
	return NewInterestAccrualRepository(s.pool)
}

func (s *Store) Users() repository.UserRepository {
	return NewUserRepository(s.pool)
}

func (s *Store) RefreshTokens() repository.RefreshTokenRepository {
	return NewRefreshTokenRepository(s.pool)
}

func (s *Store) AuthAuditEvents() repository.AuthAuditRepository {
	return NewAuthAuditRepository(s.pool)
}

func (s *Store) Idempotency() repository.IdempotencyRepository {
	return NewIdempotencyRepository(s.pool)
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return repository.ErrNotFound
	}
	return err
}

func (s *Store) Ping(ctx context.Context) error {
	if err := s.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

func (s *Store) WithAdvisoryLock(ctx context.Context, lockName string, fn func(context.Context) error) (bool, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, fmt.Errorf("begin advisory lock transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var acquired bool
	if err := tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended($1, 0))`, lockName).Scan(&acquired); err != nil {
		return false, fmt.Errorf("acquire advisory lock: %w", err)
	}
	if !acquired {
		return false, nil
	}

	if err := fn(ctx); err != nil {
		return true, err
	}
	if err := tx.Commit(ctx); err != nil {
		return true, fmt.Errorf("commit advisory lock transaction: %w", err)
	}
	return true, nil
}
