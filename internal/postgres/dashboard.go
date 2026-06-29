package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sunriseex/capitalflow/internal/repository"
)

type DashboardRepository struct {
	pool *pgxpool.Pool
}

func NewDashboardRepository(pool *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{pool: pool}
}

func (r *DashboardRepository) Summary(ctx context.Context, userID string, from, to time.Time, recentLimit int) (*repository.DashboardSummarySnapshot, error) {
	if recentLimit <= 0 || recentLimit > 100 {
		return nil, fmt.Errorf("dashboard recent limit must be between 1 and 100")
	}
	balances, err := r.AccountBalances(ctx, userID)
	if err != nil {
		return nil, err
	}
	flow, err := r.MonthlyFlow(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	expense, err := r.categoryExpense(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	goals, err := NewFinancialGoalRepository(r.pool).ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	limits, err := NewCategoryLimitRepository(r.pool).ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	categories, err := NewCategoryRepository(r.pool).List(ctx)
	if err != nil {
		return nil, err
	}
	recent, err := NewTransactionRepository(r.pool).ListByUserFiltered(ctx, userID, &repository.TransactionListFilter{Limit: recentLimit})
	if err != nil {
		return nil, err
	}

	return &repository.DashboardSummarySnapshot{
		AccountBalances: balances,
		MonthlyFlow:     flow,
		CategoryExpense: expense,
		Goals:           goals,
		Limits:          limits,
		Categories:      categories,
		Recent:          recent,
	}, nil
}

func (r *DashboardRepository) AccountBalances(ctx context.Context, userID string) ([]repository.DashboardAccountBalance, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.legacy_id, a.owner_user_id, a.name, a.bank, a.type, a.currency,
			a.is_active, a.opened_at, a.created_at, a.updated_at,
			COALESCE(SUM(CASE
				WHEN t.type IN ('initial_balance', 'income', 'transfer_in', 'interest_income', 'adjustment') THEN t.amount
				WHEN t.type IN ('expense', 'transfer_out') THEN -t.amount
				ELSE 0
			END), 0),
			COUNT(t.id)::int
		FROM accounts a
		LEFT JOIN transactions t ON t.account_id = a.id
		WHERE a.owner_user_id = $1
		GROUP BY a.id
		ORDER BY a.created_at, a.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query dashboard account balances: %w", err)
	}
	defer rows.Close()

	balances := make([]repository.DashboardAccountBalance, 0)
	for rows.Next() {
		var balance repository.DashboardAccountBalance
		if err := rows.Scan(
			&balance.Account.ID,
			&balance.Account.LegacyID,
			&balance.Account.OwnerUserID,
			&balance.Account.Name,
			&balance.Account.Bank,
			&balance.Account.Type,
			&balance.Account.Currency,
			&balance.Account.IsActive,
			&balance.Account.OpenedAt,
			&balance.Account.CreatedAt,
			&balance.Account.UpdatedAt,
			&balance.Balance,
			&balance.TransactionCount,
		); err != nil {
			return nil, fmt.Errorf("scan dashboard account balance: %w", err)
		}
		balances = append(balances, balance)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read dashboard account balances: %w", err)
	}
	return balances, nil
}

func (r *DashboardRepository) MonthlyFlow(ctx context.Context, userID string, from, to time.Time) ([]repository.DashboardMonthlyFlow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT to_char(date_trunc('month', t.occurred_at), 'YYYY-MM'), a.currency,
			COALESCE(SUM(t.amount) FILTER (WHERE t.type IN ('income', 'interest_income')), 0),
			COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'expense'), 0),
			COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'interest_income'), 0),
			COUNT(t.id)::int,
			(COUNT(t.id) FILTER (WHERE t.type = 'interest_income'))::int
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		WHERE a.owner_user_id = $1
			AND t.occurred_at >= $2
			AND t.occurred_at < $3
			AND t.type IN ('income', 'expense', 'interest_income')
		GROUP BY date_trunc('month', t.occurred_at), a.currency
		ORDER BY date_trunc('month', t.occurred_at), a.currency
	`, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query dashboard monthly flow: %w", err)
	}
	defer rows.Close()

	flow := make([]repository.DashboardMonthlyFlow, 0)
	for rows.Next() {
		var item repository.DashboardMonthlyFlow
		if err := rows.Scan(&item.Period, &item.Currency, &item.Income, &item.Expense, &item.InterestIncome, &item.TransactionCount, &item.InterestCount); err != nil {
			return nil, fmt.Errorf("scan dashboard monthly flow: %w", err)
		}
		flow = append(flow, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read dashboard monthly flow: %w", err)
	}
	return flow, nil
}

func (r *DashboardRepository) categoryExpense(ctx context.Context, userID string, from, to time.Time) ([]repository.DashboardCategoryExpense, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.category_id, a.currency, SUM(t.amount)
		FROM transactions t
		JOIN accounts a ON a.id = t.account_id
		WHERE a.owner_user_id = $1
			AND t.occurred_at >= $2
			AND t.occurred_at < $3
			AND t.type = 'expense'
			AND t.category_id IS NOT NULL
		GROUP BY t.category_id, a.currency
	`, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query dashboard category expense: %w", err)
	}
	defer rows.Close()

	expense := make([]repository.DashboardCategoryExpense, 0)
	for rows.Next() {
		var item repository.DashboardCategoryExpense
		if err := rows.Scan(&item.CategoryID, &item.Currency, &item.Amount); err != nil {
			return nil, fmt.Errorf("scan dashboard category expense: %w", err)
		}
		expense = append(expense, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read dashboard category expense: %w", err)
	}
	return expense, nil
}
