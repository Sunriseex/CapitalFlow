package legacyjson

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/internal/services"
)

func newTestImporter() (
	*Importer,
	*fakeAccountRepo,
	*fakeTransactionRepo,
	*fakeInterestRuleRepo,
	*fakeDepositMigrationRepo,
) {
	accounts := newFakeAccountRepo()
	transactions := newFakeTransactionRepo()
	rules := newFakeInterestRuleRepo()

	migrationRepo := &fakeDepositMigrationRepo{
		accounts:     accounts,
		transactions: transactions,
		rules:        rules,
	}

	migrator := NewImporter(
		accounts,
		transactions,
		rules,
		WithDepositMigrationRepository(migrationRepo),
	)

	return migrator, accounts, transactions, rules, migrationRepo
}

func TestImporterImportReadsSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deposits.json")
	if err := os.WriteFile(path, []byte(`{"deposits":[{"id":"legacy-1","name":"Reserve","bank":"Bank","type":"savings","amount":10000,"interest_rate":10,"start_date":"2026-01-01","capitalization":"daily"}]}`), 0o600); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	importer, _, _, _, _ := newTestImporter()

	report, err := importer.Import(t.Context(), path)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if report.TotalDeposits != 1 || report.CreatedAccounts != 1 || !report.BalanceMatchesSource {
		t.Fatalf("report = %+v", report)
	}
}

func TestCapitalizationForDeposit(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        models.CapitalizationFrequency
		wantErr     bool
		errContains string
	}{
		{
			name:  "empty maps to none",
			input: "",
			want:  models.CapitalizationFrequencyNone,
		},
		{
			name:  "daily maps to daily",
			input: CapitalizationDaily,
			want:  models.CapitalizationFrequencyDaily,
		},
		{
			name:  "monthly maps to monthly",
			input: CapitalizationMonthly,
			want:  models.CapitalizationFrequencyMonthly,
		},
		{
			name:  "end maps to end of term",
			input: CapitalizationEnd,
			want:  models.CapitalizationFrequencyEndOfTerm,
		},
		{
			name:  "trimmed monthly maps to monthly",
			input: "  monthly  ",
			want:  models.CapitalizationFrequencyMonthly,
		},
		{
			name:        "quarterly is rejected",
			input:       "quarterly",
			wantErr:     true,
			errContains: "unsupported legacy capitalization: quarterly",
		},
		{
			name:        "unknown capitalization is rejected",
			input:       "weekly",
			wantErr:     true,
			errContains: `unsupported legacy capitalization: "weekly"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := capitalizationForDeposit(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("error = %q, want contains %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("capitalization = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestImporterimportDeposits(t *testing.T) {
	ctx := t.Context()
	migrator, accounts, transactions, rules, migrationRepo := newTestImporter()

	promoRate := 17.5
	report, err := migrator.importDeposits(ctx, []Deposit{
		{
			ID:             "legacy-1",
			Name:           "Savings",
			Bank:           "Yandex",
			Type:           DepositTypeSavings,
			Amount:         100_000,
			InterestRate:   12,
			PromoRate:      &promoRate,
			PromoEndDate:   "2026-06-01",
			StartDate:      "2026-05-01",
			Capitalization: CapitalizationDaily,
		},
	})
	if err != nil {
		t.Fatalf("migrate deposits: %v", err)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("errors = %v", report.Errors)
	}
	if !report.BalanceMatchesSource {
		t.Fatal("balance must match source")
	}
	if report.CreatedAccounts != 1 || report.CreatedInterestRules != 1 || report.CreatedTransactions != 1 {
		t.Fatalf("created counts = accounts %d rules %d tx %d", report.CreatedAccounts, report.CreatedInterestRules, report.CreatedTransactions)
	}
	if migrationRepo.calls != 1 {
		t.Fatalf("migration repo calls = %d, want 1", migrationRepo.calls)
	}

	account := accounts.byLegacy["legacy-1"]
	if account == nil {
		t.Fatal("account not saved by legacy id")
	}
	if account.Type != models.AccountTypeSavings {
		t.Fatalf("account type = %s, want savings", account.Type)
	}
	if account.LegacyID == nil || *account.LegacyID != "legacy-1" {
		t.Fatalf("legacy id = %v, want legacy-1", account.LegacyID)
	}

	rule := rules.byAccount[account.ID][0]
	if rule.AnnualRateBps != 1_200 {
		t.Fatalf("annual rate bps = %d, want 1200", rule.AnnualRateBps)
	}
	if rule.PromoRateBps == nil || *rule.PromoRateBps != 1_750 {
		t.Fatalf("promo rate bps = %v, want 1750", rule.PromoRateBps)
	}
	if rule.CapitalizationFrequency != models.CapitalizationFrequencyDaily {
		t.Fatalf("capitalization = %s, want daily", rule.CapitalizationFrequency)
	}

	tx := transactions.byAccount[account.ID][0]
	if tx.Type != models.TransactionTypeInitialBalance || !tx.Amount.Equal(dec("1000")) {
		t.Fatalf("transaction = %s %d, want initial_balance 100000", tx.Type, tx.Amount)
	}
}

func TestImporterAssignsOwnerUserID(t *testing.T) {
	ctx := t.Context()
	accounts := newFakeAccountRepo()
	transactions := newFakeTransactionRepo()
	rules := newFakeInterestRuleRepo()
	migrationRepo := &fakeDepositMigrationRepo{
		accounts:     accounts,
		transactions: transactions,
		rules:        rules,
	}
	migrator := NewImporter(
		accounts,
		transactions,
		rules,
		WithDepositMigrationRepository(migrationRepo),
		WithOwnerUserID(" user-1 "),
	)

	report, err := migrator.importDeposits(ctx, []Deposit{
		{
			ID:             "legacy-1",
			Name:           "Savings",
			Type:           DepositTypeSavings,
			Amount:         100_000,
			InterestRate:   12,
			StartDate:      "2026-05-01",
			Capitalization: CapitalizationDaily,
		},
	})
	if err != nil {
		t.Fatalf("migrate deposits: %v", err)
	}
	if report.OwnerUserID != "user-1" {
		t.Fatalf("report owner user id = %q, want user-1", report.OwnerUserID)
	}

	account := accounts.byLegacy["legacy-1"]
	if account == nil {
		t.Fatal("account not saved by legacy id")
	}
	if account.OwnerUserID == nil || *account.OwnerUserID != "user-1" {
		t.Fatalf("owner user id = %v, want user-1", account.OwnerUserID)
	}
}

func TestImporterIsIdempotentByLegacyID(t *testing.T) {
	ctx := t.Context()
	migrator, _, _, _, _ := newTestImporter()
	deposits := []Deposit{
		{
			ID:             "legacy-1",
			Name:           "Savings",
			Type:           DepositTypeSavings,
			Amount:         100_000,
			InterestRate:   12,
			StartDate:      "2026-05-01",
			Capitalization: CapitalizationDaily,
		},
	}

	if _, err := migrator.importDeposits(ctx, deposits); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	report, err := migrator.importDeposits(ctx, deposits)
	if err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if report.SkippedExisting != 1 {
		t.Fatalf("skipped = %d, want 1", report.SkippedExisting)
	}
	if report.CreatedAccounts != 0 || report.CreatedTransactions != 0 || report.CreatedInterestRules != 0 {
		t.Fatalf("second run created accounts=%d tx=%d rules=%d", report.CreatedAccounts, report.CreatedTransactions, report.CreatedInterestRules)
	}
	if !report.BalanceMatchesSource {
		t.Fatal("second run balance must match source")
	}
}

func TestImporterRepairsPartialLegacyMigration(t *testing.T) {
	ctx := t.Context()
	migrator, accounts, _, _, _ := newTestImporter()

	legacyID := "legacy-1"
	legacyIDPtr := legacyID
	account := &models.Account{
		ID:       "account-1",
		LegacyID: &legacyIDPtr,
		Name:     "Savings",
		Type:     models.AccountTypeSavings,
		Currency: "RUB",
		OpenedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := accounts.Create(ctx, account); err != nil {
		t.Fatalf("seed account: %v", err)
	}

	report, err := migrator.importDeposits(ctx, []Deposit{
		{
			ID:             legacyID,
			Name:           "Savings",
			Type:           DepositTypeSavings,
			Amount:         100_000,
			InterestRate:   12,
			StartDate:      "2026-05-01",
			Capitalization: CapitalizationDaily,
		},
	})
	if err != nil {
		t.Fatalf("migrate deposits: %v", err)
	}
	if report.SkippedExisting != 1 {
		t.Fatalf("skipped = %d, want 1", report.SkippedExisting)
	}
	if report.CreatedInterestRules != 1 || report.CreatedTransactions != 1 {
		t.Fatalf("created rules=%d tx=%d, want 1 each", report.CreatedInterestRules, report.CreatedTransactions)
	}
	if !report.BalanceMatchesSource {
		t.Fatal("balance must match source after repair")
	}
}

func TestImporterUsesTrimmedLegacyIDForExistingInitialBalance(t *testing.T) {
	ctx := t.Context()
	migrator, accounts, transactions, rules, _ := newTestImporter()

	legacyID := "legacy-1"
	legacyIDPtr := legacyID
	account := &models.Account{
		ID:       "account-1",
		LegacyID: &legacyIDPtr,
		Name:     "Savings",
		Type:     models.AccountTypeSavings,
		Currency: "RUB",
		OpenedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := accounts.Create(ctx, account); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if err := rules.Create(ctx, &models.InterestRule{
		ID:                 "rule-1",
		AccountID:          account.ID,
		AnnualRateBps:      1_200,
		AccrualFrequency:   models.AccrualFrequencyDaily,
		DayCountConvention: models.DayCountConventionActual365,
		IsActive:           true,
		StartDate:          account.OpenedAt,
	}); err != nil {
		t.Fatalf("seed rule: %v", err)
	}
	if err := transactions.Create(ctx, &models.Transaction{
		ID:          "tx-1",
		AccountID:   account.ID,
		Type:        models.TransactionTypeInitialBalance,
		Amount:      dec("1000"),
		Description: legacyInitialDescription(legacyID),
		OccurredAt:  account.OpenedAt,
		CreatedAt:   account.OpenedAt,
	}); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	report, err := migrator.importDeposits(ctx, []Deposit{
		{
			ID:             "  legacy-1  ",
			Name:           "Savings",
			Type:           DepositTypeSavings,
			Amount:         100_000,
			InterestRate:   12,
			StartDate:      "2026-05-01",
			Capitalization: CapitalizationDaily,
		},
	})
	if err != nil {
		t.Fatalf("migrate deposits: %v", err)
	}
	if report.CreatedTransactions != 0 {
		t.Fatalf("created transactions = %d, want 0", report.CreatedTransactions)
	}
	if !report.BalanceMatchesSource {
		t.Fatal("balance must match source")
	}
}

func TestImporterExistingAccountIgnoresPostMigrationActivityForSourceBalance(t *testing.T) {
	ctx := t.Context()
	migrator, accounts, transactions, rules, _ := newTestImporter()

	legacyID := "legacy-1"
	legacyIDPtr := legacyID
	account := &models.Account{
		ID:       "account-1",
		LegacyID: &legacyIDPtr,
		Name:     "Savings",
		Type:     models.AccountTypeSavings,
		Currency: "RUB",
		OpenedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := accounts.Create(ctx, account); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if err := rules.Create(ctx, &models.InterestRule{
		ID:                 "rule-1",
		AccountID:          account.ID,
		AnnualRateBps:      1_200,
		AccrualFrequency:   models.AccrualFrequencyDaily,
		DayCountConvention: models.DayCountConventionActual365,
		IsActive:           true,
		StartDate:          account.OpenedAt,
	}); err != nil {
		t.Fatalf("seed rule: %v", err)
	}
	seedTransactions := []models.Transaction{
		{
			ID:          "tx-initial",
			AccountID:   account.ID,
			Type:        models.TransactionTypeInitialBalance,
			Amount:      dec("1000"),
			Description: legacyInitialDescription(legacyID),
			OccurredAt:  account.OpenedAt,
			CreatedAt:   account.OpenedAt,
		},
		{
			ID:          "tx-interest",
			AccountID:   account.ID,
			Type:        models.TransactionTypeInterestIncome,
			Amount:      dec("50"),
			Description: "interest accrual",
			OccurredAt:  account.OpenedAt.AddDate(0, 0, 1),
			CreatedAt:   account.OpenedAt.AddDate(0, 0, 1),
		},
		{
			ID:          "tx-expense",
			AccountID:   account.ID,
			Type:        models.TransactionTypeExpense,
			Amount:      dec("20"),
			Description: "card payment",
			OccurredAt:  account.OpenedAt.AddDate(0, 0, 2),
			CreatedAt:   account.OpenedAt.AddDate(0, 0, 2),
		},
	}
	if err := transactions.CreateMany(ctx, seedTransactions); err != nil {
		t.Fatalf("seed transactions: %v", err)
	}

	report, err := migrator.importDeposits(ctx, []Deposit{
		{
			ID:             legacyID,
			Name:           "Savings",
			Type:           DepositTypeSavings,
			Amount:         100_000,
			InterestRate:   12,
			StartDate:      "2026-05-01",
			Capitalization: CapitalizationDaily,
		},
	})
	if err != nil {
		t.Fatalf("migrate deposits: %v", err)
	}
	if report.CreatedTransactions != 0 {
		t.Fatalf("created transactions = %d, want 0", report.CreatedTransactions)
	}
	if !report.MigratedBalance.Equal(dec("1000")) {
		t.Fatalf("migrated balance = %s, want 1000", report.MigratedBalance)
	}
	if !report.BalanceMatchesSource {
		t.Fatal("balance must match source")
	}
}

func TestImporterRejectsUnsupportedLegacyCapitalization(t *testing.T) {
	ctx := t.Context()
	migrator, _, _, _, _ := newTestImporter()

	report, err := migrator.importDeposits(ctx, []Deposit{
		{
			ID:             "legacy-quarterly",
			Name:           "Quarterly Deposit",
			Type:           DepositTypeSavings,
			Amount:         100_000,
			InterestRate:   12,
			StartDate:      "2026-05-01",
			Capitalization: "quarterly",
		},
	})
	if err != nil {
		t.Fatalf("migrate deposits: %v", err)
	}
	if len(report.Errors) != 1 {
		t.Fatalf("errors = %v, want 1 error", report.Errors)
	}
	if report.CreatedAccounts != 0 || report.CreatedInterestRules != 0 || report.CreatedTransactions != 0 {
		t.Fatalf(
			"created accounts=%d rules=%d tx=%d, want all zero",
			report.CreatedAccounts,
			report.CreatedInterestRules,
			report.CreatedTransactions,
		)
	}
	if report.BalanceMatchesSource {
		t.Fatal("balance should not match source when migration has errors")
	}
}

func TestImporterRequiresDepositMigrationRepository(t *testing.T) {
	ctx := t.Context()
	migrator := NewImporter(
		newFakeAccountRepo(),
		newFakeTransactionRepo(),
		newFakeInterestRuleRepo(),
	)

	_, err := migrator.importDeposits(ctx, []Deposit{
		{
			ID:             "legacy-1",
			Name:           "Savings",
			Type:           DepositTypeSavings,
			Amount:         100_000,
			InterestRate:   12,
			StartDate:      "2026-05-01",
			Capitalization: CapitalizationDaily,
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

type fakeDepositMigrationRepo struct {
	accounts     *fakeAccountRepo
	transactions *fakeTransactionRepo
	rules        *fakeInterestRuleRepo
	calls        int
	fail         error
}

func (r *fakeDepositMigrationRepo) CreateMigratedDeposit(ctx context.Context, account *models.Account, rule *models.InterestRule, transaction *models.Transaction) error {
	r.calls++
	if r.fail != nil {
		return r.fail
	}
	if err := r.accounts.Create(ctx, account); err != nil {
		return err
	}
	if err := r.rules.Create(ctx, rule); err != nil {
		return err
	}
	if err := r.transactions.Create(ctx, transaction); err != nil {
		return err
	}
	return nil
}

type fakeAccountRepo struct {
	byID     map[string]*models.Account
	byLegacy map[string]*models.Account
}

func newFakeAccountRepo() *fakeAccountRepo {
	return &fakeAccountRepo{
		byID:     map[string]*models.Account{},
		byLegacy: map[string]*models.Account{},
	}
}

func (r *fakeAccountRepo) Create(_ context.Context, account *models.Account) error {
	cp := *account
	r.byID[account.ID] = &cp
	if account.LegacyID != nil {
		r.byLegacy[*account.LegacyID] = &cp
	}
	return nil
}

func (r *fakeAccountRepo) GetByID(_ context.Context, id string) (*models.Account, error) {
	account, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *account
	return &cp, nil
}

func (r *fakeAccountRepo) GetByIDForUser(ctx context.Context, id, _ string) (*models.Account, error) {
	return r.GetByID(ctx, id)
}

func (r *fakeAccountRepo) GetByLegacyID(_ context.Context, legacyID string) (*models.Account, error) {
	account, ok := r.byLegacy[legacyID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *account
	return &cp, nil
}

func (r *fakeAccountRepo) List(context.Context) ([]models.Account, error) {
	accounts := make([]models.Account, 0, len(r.byID))
	for _, account := range r.byID {
		accounts = append(accounts, *account)
	}
	return accounts, nil
}

func (r *fakeAccountRepo) ListByUser(ctx context.Context, _ string) ([]models.Account, error) {
	return r.List(ctx)
}

func (r *fakeAccountRepo) ClaimUnowned(_ context.Context, userID string) error {
	for _, account := range r.byID {
		if account.OwnerUserID == nil {
			ownerUserID := userID
			account.OwnerUserID = &ownerUserID
		}
	}

	return nil
}

func (r *fakeAccountRepo) Update(_ context.Context, account *models.Account) error {
	if _, ok := r.byID[account.ID]; !ok {
		return repository.ErrNotFound
	}
	cp := *account
	r.byID[account.ID] = &cp
	return nil
}

func (r *fakeAccountRepo) UpdateForUser(ctx context.Context, account *models.Account, _ string) error {
	return r.Update(ctx, account)
}

func (r *fakeAccountRepo) UpdateForUserEnforcingCurrencyInvariant(ctx context.Context, account *models.Account, _ string) error {
	return r.Update(ctx, account)
}

func (r *fakeAccountRepo) Archive(_ context.Context, id string) error {
	account, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	account.IsActive = false
	return nil
}

func (r *fakeAccountRepo) ArchiveForUser(ctx context.Context, id, _ string) error {
	return r.Archive(ctx, id)
}

type fakeTransactionRepo struct {
	byID      map[string]*models.Transaction
	byAccount map[string][]models.Transaction
}

func newFakeTransactionRepo() *fakeTransactionRepo {
	return &fakeTransactionRepo{
		byID:      map[string]*models.Transaction{},
		byAccount: map[string][]models.Transaction{},
	}
}

func (r *fakeTransactionRepo) Create(_ context.Context, transaction *models.Transaction) error {
	cp := *transaction
	r.byID[transaction.ID] = &cp
	r.byAccount[transaction.AccountID] = append(r.byAccount[transaction.AccountID], cp)
	return nil
}

func (r *fakeTransactionRepo) CreateForUser(ctx context.Context, _ string, transaction *models.Transaction) error {
	return r.Create(ctx, transaction)
}

func (r *fakeTransactionRepo) CreateMany(ctx context.Context, transactions []models.Transaction) error {
	for i := range transactions {
		if err := r.Create(ctx, &transactions[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *fakeTransactionRepo) CreateTransfer(ctx context.Context, _ *models.Transfer, transactions []models.Transaction) error {
	return r.CreateMany(ctx, transactions)
}

func (r *fakeTransactionRepo) CancelForUser(ctx context.Context, id, _ string) (*models.Transaction, error) {
	transaction, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	transaction.Status = models.TransactionStatusCancelled
	r.byID[id] = transaction
	return transaction, nil
}

func (r *fakeTransactionRepo) ReverseForUser(ctx context.Context, id, _ string, reversal *models.Transaction) (updated, created *models.Transaction, err error) {
	transaction, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	transaction.Status = models.TransactionStatusReversed
	r.byID[id] = transaction
	if err := r.Create(ctx, reversal); err != nil {
		return nil, nil, err
	}
	return transaction, reversal, nil
}

func (r *fakeTransactionRepo) SoftDeleteForUser(ctx context.Context, id, _ string) (*models.Transaction, error) {
	transaction, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	transaction.Status = models.TransactionStatusSoftDeleted
	r.byID[id] = transaction
	return transaction, nil
}

func (r *fakeTransactionRepo) ListTransfersByUser(context.Context, string) ([]models.Transfer, error) {
	return nil, nil
}

func (r *fakeTransactionRepo) GetByID(_ context.Context, id string) (*models.Transaction, error) {
	transaction, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *transaction
	return &cp, nil
}

func (r *fakeTransactionRepo) GetByIDForUser(ctx context.Context, id, _ string) (*models.Transaction, error) {
	return r.GetByID(ctx, id)
}

func (r *fakeTransactionRepo) List(context.Context) ([]models.Transaction, error) {
	var transactions []models.Transaction
	for _, transaction := range r.byID {
		transactions = append(transactions, *transaction)
	}
	return transactions, nil
}

func (r *fakeTransactionRepo) ListByUser(ctx context.Context, _ string) ([]models.Transaction, error) {
	return r.List(ctx)
}

func (r *fakeTransactionRepo) ListByAccount(_ context.Context, accountID string) ([]models.Transaction, error) {
	return append([]models.Transaction(nil), r.byAccount[accountID]...), nil
}

func (r *fakeTransactionRepo) ListByAccountForUser(ctx context.Context, accountID, _ string) ([]models.Transaction, error) {
	return r.ListByAccount(ctx, accountID)
}

func (r *fakeTransactionRepo) GetBalanceByAccountForUser(ctx context.Context, accountID, _ string) (balance decimal.Decimal, transactionCount int64, err error) {
	transactions, err := r.ListByAccount(ctx, accountID)
	if err != nil {
		return decimal.Zero, 0, fmt.Errorf("calculate fake account balance: %w", err)
	}
	result, err := services.NewBalanceService().Calculate(ctx, services.CalculateBalanceRequest{
		AccountID:    accountID,
		Transactions: transactions,
	})
	if err != nil {
		return decimal.Zero, 0, fmt.Errorf("calculate fake account balance: %w", err)
	}
	return result.Balance, int64(result.Count), nil
}

type fakeInterestRuleRepo struct {
	byID      map[string]*models.InterestRule
	byAccount map[string][]models.InterestRule
}

func newFakeInterestRuleRepo() *fakeInterestRuleRepo {
	return &fakeInterestRuleRepo{
		byID:      map[string]*models.InterestRule{},
		byAccount: map[string][]models.InterestRule{},
	}
}

func (r *fakeInterestRuleRepo) Create(_ context.Context, rule *models.InterestRule) error {
	if rule.ID == "" {
		return errors.New("id is required")
	}
	cp := *rule
	r.byID[rule.ID] = &cp
	r.byAccount[rule.AccountID] = append(r.byAccount[rule.AccountID], cp)
	return nil
}

func (r *fakeInterestRuleRepo) GetByID(_ context.Context, id string) (*models.InterestRule, error) {
	rule, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *rule
	return &cp, nil
}

func (r *fakeInterestRuleRepo) ListByAccount(_ context.Context, accountID string) ([]models.InterestRule, error) {
	return append([]models.InterestRule(nil), r.byAccount[accountID]...), nil
}

func (r *fakeInterestRuleRepo) Update(_ context.Context, rule *models.InterestRule) error {
	if _, ok := r.byID[rule.ID]; !ok {
		return repository.ErrNotFound
	}
	cp := *rule
	r.byID[rule.ID] = &cp
	return nil
}

func TestAccountTypeForDeposit(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        models.AccountType
		wantErr     bool
		errContains string
	}{
		{
			name:  "savings maps to savings account",
			input: DepositTypeSavings,
			want:  models.AccountTypeSavings,
		},
		{
			name:  "term maps to term deposit account",
			input: DepositTypeTerm,
			want:  models.AccountTypeTermDeposit,
		},
		{
			name:  "trimmed term maps to term deposit account",
			input: "  term  ",
			want:  models.AccountTypeTermDeposit,
		},
		{
			name:        "empty type is rejected",
			input:       "",
			wantErr:     true,
			errContains: `unsupported legacy deposit type: ""`,
		},
		{
			name:        "unknown type is rejected",
			input:       "saving",
			wantErr:     true,
			errContains: `unsupported legacy deposit type: "saving"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := accountTypeForDeposit(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("error = %q, want contains %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("account type = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestImporterRejectsUnsupportedLegacyDepositType(t *testing.T) {
	ctx := t.Context()
	migrator, _, _, _, _ := newTestImporter()

	report, err := migrator.importDeposits(ctx, []Deposit{
		{
			ID:             "legacy-invalid-type",
			Name:           "Invalid Type Deposit",
			Type:           "saving",
			Amount:         100_000,
			InterestRate:   12,
			StartDate:      "2026-05-01",
			Capitalization: CapitalizationDaily,
		},
	})
	if err != nil {
		t.Fatalf("migrate deposits: %v", err)
	}
	if len(report.Errors) != 1 {
		t.Fatalf("errors = %v, want 1 error", report.Errors)
	}
	if report.CreatedAccounts != 0 || report.CreatedInterestRules != 0 || report.CreatedTransactions != 0 {
		t.Fatalf(
			"created accounts=%d rules=%d tx=%d, want all zero",
			report.CreatedAccounts,
			report.CreatedInterestRules,
			report.CreatedTransactions,
		)
	}
	if report.BalanceMatchesSource {
		t.Fatal("balance should not match source when migration has errors")
	}
}

func TestParseRequiredDate(t *testing.T) {
	date, err := parseRequiredDate("start date", "2026-05-01")
	if err != nil {
		t.Fatalf("parse required date: %v", err)
	}
	if got := date.Format(time.DateOnly); got != "2026-05-01" {
		t.Fatalf("date = %s, want 2026-05-01", got)
	}

	if _, err := parseRequiredDate("start date", ""); err == nil {
		t.Fatal("expected error for empty date")
	}

	if _, err := parseRequiredDate("start date", "06-05-2026"); err == nil {
		t.Fatal("expected error for malformed date")
	}
}

func TestImporterRejectsMalformedLegacyStartDate(t *testing.T) {
	ctx := t.Context()
	migrator, _, _, _, _ := newTestImporter()

	report, err := migrator.importDeposits(ctx, []Deposit{
		{
			ID:             "legacy-bad-start-date",
			Name:           "Bad Start Date Deposit",
			Type:           DepositTypeSavings,
			Amount:         100_000,
			InterestRate:   12,
			StartDate:      "06-05-2026",
			Capitalization: CapitalizationDaily,
		},
	})
	if err != nil {
		t.Fatalf("migrate deposits: %v", err)
	}
	if len(report.Errors) != 1 {
		t.Fatalf("errors = %v, want 1 error", report.Errors)
	}
	if report.CreatedAccounts != 0 || report.CreatedInterestRules != 0 || report.CreatedTransactions != 0 {
		t.Fatalf(
			"created accounts=%d rules=%d tx=%d, want all zero",
			report.CreatedAccounts,
			report.CreatedInterestRules,
			report.CreatedTransactions,
		)
	}
	if report.BalanceMatchesSource {
		t.Fatal("balance should not match source when migration has errors")
	}
}
