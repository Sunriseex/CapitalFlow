package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/jobs"
	"github.com/sunriseex/capitalflow/internal/legacyjson"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/internal/services"
)

// CommandModule owns workflows used by non-HTTP adapters. It keeps storage
// selection and domain sequencing out of command-line parsing code.
type CommandModule struct {
	store Store
	app   *Application
}

const DefaultLegacyDepositSnapshotPath = legacyjson.DefaultDepositSnapshotPath

type LegacyImportReport = legacyjson.ImportReport

func newCommandModule(store Store, app *Application) *CommandModule {
	return &CommandModule{store: store, app: app}
}

func (m *CommandModule) requireStore() error {
	if m == nil || m.store == nil || m.app == nil {
		return fmt.Errorf("command module is not configured")
	}
	return nil
}

func (m *CommandModule) Ready(ctx context.Context) error {
	if err := m.requireStore(); err != nil {
		return err
	}
	return m.app.Ready(ctx)
}

func (m *CommandModule) ListAccounts(ctx context.Context) ([]models.Account, error) {
	if err := m.requireStore(); err != nil {
		return nil, err
	}
	accounts, err := m.store.Accounts().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	return accounts, nil
}

type CreateAccountCommand struct {
	OwnerUserID string
	Name        string
	Bank        string
	Type        models.AccountType
	Currency    string
	OpenedAt    time.Time
}

func (m *CommandModule) CreateAccount(ctx context.Context, cmd *CreateAccountCommand) (*models.Account, error) {
	if err := m.requireStore(); err != nil {
		return nil, err
	}
	if cmd == nil {
		return nil, fmt.Errorf("create account command is required")
	}
	ownerUserID, err := resolveOwnerUserID(ctx, m.store.Users(), cmd.OwnerUserID)
	if err != nil {
		return nil, err
	}
	account, err := m.app.Accounts.Create(ctx, &services.CreateAccountRequest{
		OwnerUserID: ownerUserID,
		Name:        cmd.Name,
		Bank:        cmd.Bank,
		Type:        cmd.Type,
		Currency:    cmd.Currency,
		OpenedAt:    cmd.OpenedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	return account, nil
}

func (m *CommandModule) ListTransactions(ctx context.Context, accountID string) ([]models.Transaction, error) {
	if err := m.requireStore(); err != nil {
		return nil, err
	}
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		transactions, err := m.store.Transactions().List(ctx)
		if err != nil {
			return nil, fmt.Errorf("list transactions: %w", err)
		}
		return transactions, nil
	}
	transactions, err := m.store.Transactions().ListByAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("list account transactions: %w", err)
	}
	return transactions, nil
}

type CreateTransactionCommand struct {
	AccountID   string
	Type        models.TransactionType
	Amount      decimal.Decimal
	Description string
	OccurredAt  time.Time
}

func (m *CommandModule) CreateTransaction(ctx context.Context, cmd *CreateTransactionCommand) (*models.Transaction, error) {
	if err := m.requireStore(); err != nil {
		return nil, err
	}
	if cmd == nil {
		return nil, fmt.Errorf("create transaction command is required")
	}
	account, err := m.store.Accounts().GetByID(ctx, strings.TrimSpace(cmd.AccountID))
	if err != nil {
		return nil, fmt.Errorf("get transaction account: %w", err)
	}
	req := &services.CreateTransactionRequest{
		AccountID:       cmd.AccountID,
		Type:            cmd.Type,
		Amount:          cmd.Amount,
		Currency:        account.Currency,
		Description:     cmd.Description,
		OccurredAt:      cmd.OccurredAt,
		AccountOpenedAt: account.OpenedAt,
	}
	if account.OwnerUserID != nil && strings.TrimSpace(*account.OwnerUserID) != "" {
		transaction, err := m.app.Transactions.CreateForUser(ctx, *account.OwnerUserID, req)
		if err != nil {
			return nil, fmt.Errorf("create owned transaction: %w", err)
		}
		return transaction, nil
	}
	transaction, err := m.app.Transactions.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}
	return transaction, nil
}

func (m *CommandModule) Balance(ctx context.Context, accountID string) (*services.CalculateBalanceResponse, error) {
	transactions, err := m.ListTransactions(ctx, accountID)
	if err != nil {
		return nil, err
	}
	balance, err := services.NewBalanceService().Calculate(ctx, services.CalculateBalanceRequest{
		AccountID:    strings.TrimSpace(accountID),
		Transactions: transactions,
	})
	if err != nil {
		return nil, fmt.Errorf("calculate account balance: %w", err)
	}
	return balance, nil
}

func (m *CommandModule) ImportLegacyDeposits(ctx context.Context, path, ownerUserID string) (*legacyjson.ImportReport, error) {
	if err := m.requireStore(); err != nil {
		return nil, err
	}
	ownerUserID, err := resolveOwnerUserID(ctx, m.store.Users(), ownerUserID)
	if err != nil {
		return nil, err
	}
	migration, ok := m.store.(repository.DepositMigrationRepository)
	if !ok {
		return nil, fmt.Errorf("deposit migration repository is required")
	}
	report, err := legacyjson.NewImporter(
		m.store.Accounts(),
		m.store.Transactions(),
		m.store.InterestRules(),
		legacyjson.WithDepositMigrationRepository(migration),
		legacyjson.WithOwnerUserID(ownerUserID),
	).Import(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("import legacy deposits: %w", err)
	}
	return report, nil
}

type InterestJobCommandResult struct {
	Acquired bool
	Run      *jobs.InterestJobRunResult
}

type advisoryLocker interface {
	WithAdvisoryLock(context.Context, string, func(context.Context) error) (bool, error)
}

func (m *CommandModule) RunInterestJob(ctx context.Context, name string, date time.Time) (*InterestJobCommandResult, error) {
	if err := m.requireStore(); err != nil {
		return nil, err
	}
	rules, ok := m.store.InterestRules().(repository.InterestRuleJobRepository)
	if !ok {
		return nil, fmt.Errorf("interest rule repository does not support jobs")
	}
	locker, ok := m.store.(advisoryLocker)
	if !ok {
		return nil, fmt.Errorf("advisory lock is required")
	}
	job := &jobs.InterestJob{Rules: rules, Lifecycle: m.app.InterestLifecycle, Now: func() time.Time { return date }}
	result := &InterestJobCommandResult{}
	acquired, err := locker.WithAdvisoryLock(ctx, "capitalflow:"+name, func(ctx context.Context) error {
		var runErr error
		switch name {
		case jobs.DailyInterestAccrualJobName:
			result.Run, runErr = job.RunDailyInterestAccrual(ctx)
		case jobs.MonthlyInterestAccrualJobName:
			result.Run, runErr = job.RunMonthlyInterestAccrual(ctx)
		case jobs.DepositMaturityCheckJobName:
			result.Run, runErr = job.RunDepositMaturityCheck(ctx)
		default:
			return fmt.Errorf("unknown job name: %s", name)
		}
		if runErr != nil {
			return fmt.Errorf("run interest job: %w", runErr)
		}
		return nil
	})
	result.Acquired = acquired
	if err != nil {
		return result, fmt.Errorf("run locked interest job: %w", err)
	}
	return result, nil
}

type InterestCommand struct {
	AccountID string
	RuleID    string
	Date      time.Time
	FromDate  time.Time
	ToDate    time.Time
	Days      int
}

type AccrueInterestResult struct {
	RuleID  string
	Accrual *models.InterestAccrual
	Skipped bool
}

type interestAccrualWriter interface {
	CreateWithTransaction(ctx context.Context, transaction *models.Transaction, accrual *models.InterestAccrual) error
	ReplaceRangeWithTransactions(ctx context.Context, accountID, ruleID string, fromDate, toDate time.Time, transactions []models.Transaction, accruals []models.InterestAccrual) (int64, error)
}

func (m *CommandModule) AccrueInterest(ctx context.Context, cmd *InterestCommand) (*AccrueInterestResult, error) {
	if cmd == nil {
		return nil, fmt.Errorf("accrue interest command is required")
	}
	account, rule, transactions, accruals, err := m.interestSnapshot(ctx, cmd.AccountID, cmd.RuleID, cmd.Date)
	if err != nil {
		return nil, fmt.Errorf("load interest snapshot: %w", err)
	}
	balance, err := services.NewBalanceService().Calculate(ctx, services.CalculateBalanceRequest{
		AccountID: account.ID, Transactions: transactionsUpToDate(transactions, cmd.Date),
	})
	if err != nil {
		return nil, fmt.Errorf("calculate accrual balance: %w", err)
	}
	if rule.CapitalizationFrequency == models.CapitalizationFrequencyNone || rule.CapitalizationFrequency == "" {
		transactions = services.PrincipalTransactionsForRuleAt(transactions, accruals, rule, time.Time{})
	}
	category, err := m.store.Categories().GetBySlug(ctx, "deposit_interest")
	if err != nil {
		return nil, fmt.Errorf("get deposit interest category: %w", err)
	}
	result, err := m.app.InterestEngine.Accrue(ctx, &services.AccrueRuleInterestRequest{
		Rule: *rule, AccountName: account.Name, CategoryID: category.ID,
		Currency: account.Currency, Balance: balance.Balance, AccrualDate: cmd.Date,
		Transactions: transactions, ExistingAccruals: accruals,
	})
	if err != nil {
		return nil, fmt.Errorf("accrue interest: %w", err)
	}
	if !result.Skipped {
		conflict, err := persistCalculatedAccrual(ctx, m.store.InterestAccruals(), result.Transaction, result.Accrual)
		if err != nil {
			return nil, fmt.Errorf("save interest accrual: %w", err)
		}
		if conflict {
			return &AccrueInterestResult{RuleID: rule.ID, Skipped: true}, nil
		}
	}
	return &AccrueInterestResult{RuleID: rule.ID, Accrual: result.Accrual, Skipped: result.Skipped}, nil
}

func (m *CommandModule) ForecastInterest(ctx context.Context, cmd *InterestCommand) (*services.ForecastRuleInterestResponse, error) {
	if cmd == nil {
		return nil, fmt.Errorf("forecast interest command is required")
	}
	account, rule, transactions, accruals, err := m.interestSnapshot(ctx, cmd.AccountID, cmd.RuleID, cmd.Date)
	if err != nil {
		return nil, err
	}
	result, err := m.app.InterestEngine.Forecast(ctx, &services.ForecastRuleInterestRequest{
		Rule: *rule, Currency: account.Currency, Transactions: transactions, ExistingAccruals: accruals,
		FromDate: cmd.Date, Days: cmd.Days,
	})
	if err != nil {
		return nil, fmt.Errorf("forecast interest: %w", err)
	}
	return result, nil
}

func (m *CommandModule) RecalculateInterest(ctx context.Context, cmd *InterestCommand) (*services.RecalculateRuleInterestResponse, error) {
	if cmd == nil {
		return nil, fmt.Errorf("recalculate interest command is required")
	}
	account, rule, transactions, accruals, err := m.interestSnapshot(ctx, cmd.AccountID, cmd.RuleID, cmd.Date)
	if err != nil {
		return nil, err
	}
	category, err := m.store.Categories().GetBySlug(ctx, "deposit_interest")
	if err != nil {
		return nil, fmt.Errorf("get deposit interest category: %w", err)
	}
	result, err := m.app.InterestEngine.Recalculate(ctx, &services.RecalculateRuleInterestRequest{
		Rule: *rule, AccountName: account.Name, CategoryID: category.ID,
		Currency: account.Currency, Transactions: transactions, ExistingAccruals: accruals,
		FromDate: cmd.FromDate, ToDate: cmd.ToDate,
	})
	if err != nil {
		return nil, fmt.Errorf("recalculate interest: %w", err)
	}
	deleted, err := replaceCalculatedAccruals(ctx, m.store.InterestAccruals(), result)
	if err != nil {
		return nil, fmt.Errorf("replace recalculated interest accruals: %w", err)
	}
	result.DeletedAccruals = deleted
	return result, nil
}

func persistCalculatedAccrual(ctx context.Context, writer interestAccrualWriter, transaction *models.Transaction, accrual *models.InterestAccrual) (bool, error) {
	if writer == nil {
		return false, fmt.Errorf("interest accrual writer is required")
	}
	if err := writer.CreateWithTransaction(ctx, transaction, accrual); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return true, nil
		}
		return false, fmt.Errorf("create interest accrual with transaction: %w", err)
	}
	return false, nil
}

func replaceCalculatedAccruals(ctx context.Context, writer interestAccrualWriter, result *services.RecalculateRuleInterestResponse) (int64, error) {
	if writer == nil {
		return 0, fmt.Errorf("interest accrual writer is required")
	}
	deleted, err := writer.ReplaceRangeWithTransactions(
		ctx, result.AccountID, result.RuleID, result.FromDate, result.ToDate, result.Transactions, result.Accruals,
	)
	if err != nil {
		return 0, fmt.Errorf("replace interest accrual range: %w", err)
	}
	return deleted, nil
}

func (m *CommandModule) interestSnapshot(ctx context.Context, accountID, ruleID string, date time.Time) (*models.Account, *models.InterestRule, []models.Transaction, []models.InterestAccrual, error) {
	if err := m.requireStore(); err != nil {
		return nil, nil, nil, nil, err
	}
	accountID = strings.TrimSpace(accountID)
	account, err := m.store.Accounts().GetByID(ctx, accountID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("get interest account: %w", err)
	}
	rule, err := selectInterestRule(ctx, m.store.InterestRules(), accountID, ruleID, date)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	transactions, err := m.store.Transactions().ListByAccount(ctx, accountID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("list interest transactions: %w", err)
	}
	accruals, err := m.store.InterestAccruals().ListByAccount(ctx, accountID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("list interest accruals: %w", err)
	}
	return account, rule, transactions, accruals, nil
}

func resolveOwnerUserID(ctx context.Context, users repository.UserRepository, ownerUserID string) (string, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	count, err := users.Count(ctx)
	if err != nil {
		return "", fmt.Errorf("count users: %w", err)
	}
	if count == 0 {
		if ownerUserID != "" {
			return "", fmt.Errorf("owner-user-id was provided, but setup has not created a user yet")
		}
		return "", nil
	}
	if ownerUserID != "" {
		if _, err := users.GetByID(ctx, ownerUserID); err != nil {
			return "", fmt.Errorf("get owner user: %w", err)
		}
		return ownerUserID, nil
	}
	if count == 1 {
		return "", nil
	}
	return "", fmt.Errorf("owner-user-id is required when multiple users exist")
}

func selectInterestRule(ctx context.Context, repo repository.InterestRuleRepository, accountID, ruleID string, date time.Time) (*models.InterestRule, error) {
	ruleID = strings.TrimSpace(ruleID)
	if ruleID != "" {
		rule, err := repo.GetByID(ctx, ruleID)
		if err != nil {
			return nil, fmt.Errorf("get interest rule: %w", err)
		}
		if rule.AccountID != accountID {
			return nil, repository.ErrNotFound
		}
		return rule, nil
	}
	rules, err := repo.ListByAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("list account interest rules: %w", err)
	}
	var selected *models.InterestRule
	for i := range rules {
		rule := &rules[i]
		if !rule.IsActive || !interestRuleActiveOn(rule, date) {
			continue
		}
		if selected == nil || dateOnly(rule.StartDate).After(dateOnly(selected.StartDate)) {
			selected = rule
		}
	}
	if selected == nil {
		return nil, repository.ErrNotFound
	}
	return selected, nil
}

func interestRuleActiveOn(rule *models.InterestRule, date time.Time) bool {
	date = dateOnly(date)
	if date.IsZero() {
		date = dateOnly(time.Now())
	}
	if date.Before(dateOnly(rule.StartDate)) {
		return false
	}
	return rule.EndDate == nil || !date.After(dateOnly(*rule.EndDate))
}

func transactionsUpToDate(transactions []models.Transaction, date time.Time) []models.Transaction {
	date = dateOnly(date)
	filtered := make([]models.Transaction, 0, len(transactions))
	for i := range transactions {
		if !dateOnly(transactions[i].OccurredAt).After(date) {
			filtered = append(filtered, transactions[i])
		}
	}
	return filtered
}

func dateOnly(date time.Time) time.Time {
	if date.IsZero() {
		return time.Time{}
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}
