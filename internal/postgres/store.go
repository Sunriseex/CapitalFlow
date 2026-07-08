package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (s *Store) TransactionQueries() repository.TransactionQueryRepository {
	return NewTransactionRepository(s.pool)
}

func (s *Store) Categories() repository.CategoryRepository {
	return NewCategoryRepository(s.pool)
}

func (s *Store) FinancialGoals() repository.FinancialGoalRepository {
	return NewFinancialGoalRepository(s.pool)
}

func (s *Store) CategoryLimits() repository.CategoryLimitRepository {
	return NewCategoryLimitRepository(s.pool)
}

func (s *Store) Dashboard() repository.DashboardRepository {
	return NewDashboardRepository(s.pool)
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

func (s *Store) AuditEvents() repository.AuditEventRepository {
	return NewAuditEventRepository(s.pool)
}

func (s *Store) Passkeys() repository.PasskeyRepository {
	return NewPasskeyRepository(s.pool)
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

const maxAdvisoryLockNameLength = 256

// WithAdvisoryLock runs fn while a PostgreSQL transaction-scoped advisory lock
// is held. The callback is not executed inside that database transaction; the
// transaction exists only to scope the distributed lock lifetime.
func (s *Store) WithAdvisoryLock(ctx context.Context, lockName string, fn func(context.Context) error) (acquired bool, err error) {
	if lockName == "" {
		return false, fmt.Errorf("advisory lock name is required")
	}
	if len(lockName) > maxAdvisoryLockNameLength {
		return false, fmt.Errorf("advisory lock name is too long")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, fmt.Errorf("begin advisory lock transaction: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if rollbackErr := tx.Rollback(rollbackCtx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("rollback advisory lock transaction: %w", rollbackErr))
		}
	}()

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
	committed = true
	return true, nil
}
