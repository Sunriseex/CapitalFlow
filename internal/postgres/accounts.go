package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

func (r *AccountRepository) Create(ctx context.Context, account *models.Account) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create account: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := insertAccount(ctx, tx, account); err != nil {
		return fmt.Errorf("create account: %w", err)
	}
	auditEvent, err := newAuditEvent(account.OwnerUserID, "account.created", "account", account.ID, accountAuditSummary(account))
	if err != nil {
		return fmt.Errorf("build account audit event: %w", err)
	}
	if err := insertAuditEvent(ctx, tx, auditEvent); err != nil {
		return fmt.Errorf("create account audit event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create account: %w", err)
	}
	return nil
}

func (r *AccountRepository) GetByID(ctx context.Context, id string) (*models.Account, error) {
	return r.getAccount(ctx, accountSelectSQL+` WHERE id = $1`, id)
}

func (r *AccountRepository) GetByIDForUser(ctx context.Context, id, userID string) (*models.Account, error) {
	return r.getAccount(ctx, accountSelectSQL+` WHERE id = $1 AND owner_user_id = $2`, id, userID)
}

func (r *AccountRepository) GetByLegacyID(ctx context.Context, legacyID string) (*models.Account, error) {
	return r.getAccount(ctx, accountSelectSQL+` WHERE legacy_id = $1`, legacyID)
}

func (r *AccountRepository) List(ctx context.Context) ([]models.Account, error) {
	return r.list(ctx, accountSelectSQL+` ORDER BY created_at, name`)
}

func (r *AccountRepository) ListByUser(ctx context.Context, userID string) ([]models.Account, error) {
	return r.list(ctx, accountSelectSQL+` WHERE owner_user_id = $1 ORDER BY created_at, name`, userID)
}

func (r *AccountRepository) list(ctx context.Context, query string, args ...any) ([]models.Account, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		account, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, *account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list accounts rows: %w", err)
	}
	return accounts, nil
}

func (r *AccountRepository) Update(ctx context.Context, account *models.Account) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE accounts
		SET name = $2, bank = $3, type = $4, currency = $5, is_active = $6, opened_at = $7, updated_at = $8
		WHERE id = $1
	`, account.ID, account.Name, account.Bank, account.Type, account.Currency, account.IsActive, account.OpenedAt, account.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update account: %w", repository.ErrNotFound)
	}
	return nil
}

func (r *AccountRepository) UpdateForUser(ctx context.Context, account *models.Account, userID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE accounts
		SET name = $3, bank = $4, type = $5, currency = $6, is_active = $7, opened_at = $8, updated_at = $9
		WHERE id = $1 AND owner_user_id = $2
	`, account.ID, userID, account.Name, account.Bank, account.Type, account.Currency, account.IsActive, account.OpenedAt, account.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update account: %w", repository.ErrNotFound)
	}
	return nil
}

func (r *AccountRepository) UpdateForUserEnforcingCurrencyInvariant(ctx context.Context, account *models.Account, userID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update account enforcing currency invariant: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	before, err := scanAccount(tx.QueryRow(ctx, accountSelectSQL+`
		WHERE id = $1 AND owner_user_id = $2
		FOR UPDATE
	`, account.ID, userID))
	if err != nil {
		return fmt.Errorf("lock account enforcing currency invariant: %w", err)
	}

	if before.Currency != account.Currency {
		var hasTransactions bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM transactions
				WHERE account_id = $1
			)
		`, account.ID).Scan(&hasTransactions); err != nil {
			return fmt.Errorf("check account transactions: %w", err)
		}
		if hasTransactions {
			return fmt.Errorf("update account enforcing currency invariant: %w", repository.ErrAccountCurrencyInvariant)
		}
	}

	tag, err := tx.Exec(ctx, `
		UPDATE accounts
		SET name = $3, bank = $4, type = $5, currency = $6, is_active = $7, opened_at = $8, updated_at = $9
		WHERE id = $1 AND owner_user_id = $2
	`, account.ID, userID, account.Name, account.Bank, account.Type, account.Currency, account.IsActive, account.OpenedAt, account.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update account enforcing currency invariant: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update account enforcing currency invariant: %w", repository.ErrNotFound)
	}
	auditEvent, err := newAuditEventWithSummaries(&userID, "account.updated", "account", account.ID, accountAuditSummary(before), accountAuditSummary(account))
	if err != nil {
		return fmt.Errorf("build account update audit event: %w", err)
	}
	if err := insertAuditEvent(ctx, tx, auditEvent); err != nil {
		return fmt.Errorf("create account update audit event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update account enforcing currency invariant: %w", err)
	}
	return nil
}

func (r *AccountRepository) Archive(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE accounts SET is_active = false, updated_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("archive account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("archive account: %w", repository.ErrNotFound)
	}
	return nil
}

func (r *AccountRepository) ArchiveForUser(ctx context.Context, id, userID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin archive account: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	before, err := scanAccount(tx.QueryRow(ctx, accountSelectSQL+`
		WHERE id = $1 AND owner_user_id = $2
		FOR UPDATE
	`, id, userID))
	if err != nil {
		return fmt.Errorf("lock account for archive: %w", err)
	}
	after := *before
	after.IsActive = false
	if err := tx.QueryRow(ctx, `
		UPDATE accounts SET is_active = false, updated_at = now()
		WHERE id = $1 AND owner_user_id = $2
		RETURNING updated_at
	`, id, userID).Scan(&after.UpdatedAt); err != nil {
		return fmt.Errorf("archive account: %w", mapNotFound(err))
	}
	auditEvent, err := newAuditEventWithSummaries(&userID, "account.archived", "account", id, accountAuditSummary(before), accountAuditSummary(&after))
	if err != nil {
		return fmt.Errorf("build account archive audit event: %w", err)
	}
	if err := insertAuditEvent(ctx, tx, auditEvent); err != nil {
		return fmt.Errorf("create account archive audit event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit archive account: %w", err)
	}
	return nil
}

func (r *AccountRepository) ClaimUnowned(ctx context.Context, userID string) error {
	if err := claimUnownedAccounts(ctx, r.pool, userID); err != nil {
		return fmt.Errorf("claim unowned accounts: %w", err)
	}

	return nil
}

func claimUnownedAccounts(ctx context.Context, execer sqlExecer, userID string) error {
	_, err := execer.Exec(ctx, `
		UPDATE accounts
		SET owner_user_id = $1, updated_at = now()
		WHERE owner_user_id IS NULL
	`, userID)
	if err != nil {
		return fmt.Errorf("claim unowned accounts: %w", err)
	}
	return nil
}

func (r *AccountRepository) getAccount(ctx context.Context, query string, args ...any) (*models.Account, error) {
	account, err := scanAccount(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, fmt.Errorf("get account: %w", mapNotFound(err))
	}
	return account, nil
}

type accountScanner interface {
	Scan(dest ...any) error
}

const accountSelectSQL = `SELECT id, legacy_id, owner_user_id, name, bank, type, currency, is_active, opened_at, created_at, updated_at FROM accounts`

func scanAccount(row accountScanner) (*models.Account, error) {
	var account models.Account
	if err := row.Scan(&account.ID, &account.LegacyID, &account.OwnerUserID, &account.Name, &account.Bank, &account.Type, &account.Currency, &account.IsActive, &account.OpenedAt, &account.CreatedAt, &account.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan account: %w", mapNotFound(err))
	}
	return &account, nil
}

func insertAccount(ctx context.Context, execer sqlExecer, account *models.Account) error {
	_, err := execer.Exec(ctx, `
		INSERT INTO accounts (id, legacy_id, owner_user_id, name, bank, type, currency, is_active, opened_at, created_at, updated_at)
		VALUES (
			$1,
			$2,
			COALESCE(
				$3,
				(
					SELECT id
					FROM users
					WHERE (SELECT count(*) FROM users) = 1
					ORDER BY created_at ASC
					LIMIT 1
				)
			),
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11
		)
	`, account.ID, account.LegacyID, account.OwnerUserID, account.Name, account.Bank, account.Type, account.Currency, account.IsActive, account.OpenedAt, account.CreatedAt, account.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert account: %w", err)
	}
	return nil
}

func accountAuditSummary(account *models.Account) map[string]any {
	return map[string]any{
		"name":       account.Name,
		"bank":       account.Bank,
		"type":       account.Type,
		"currency":   account.Currency,
		"is_active":  account.IsActive,
		"opened_at":  account.OpenedAt,
		"updated_at": account.UpdatedAt,
	}
}
