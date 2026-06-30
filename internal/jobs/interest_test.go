package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/internal/services"
)

func TestDailyInterestAccrualJobPostsDueRule(t *testing.T) {
	t.Parallel()

	accrualDate := time.Date(2026, 5, 21, 10, 0, 0, 0, time.FixedZone("test", 3*60*60))
	rule := testInterestRule(models.AccrualFrequencyDaily, nil)
	snapshot := &fakeInterestSnapshot{
		rules: []models.InterestRule{rule},
		transactions: []models.Transaction{
			{
				ID:         "tx-initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: accrualDate.AddDate(0, 0, -1),
				CreatedAt:  accrualDate.AddDate(0, 0, -1),
			},
		},
	}
	job := &InterestJob{
		Rules: &fakeInterestRuleJobRepo{
			targets: []repository.InterestRuleJobTarget{{Rule: rule, OwnerUserID: "user-1"}},
		},
		Lifecycle: services.NewInterestLifecycle(&fakeInterestAccrualTxRepo{snapshot: snapshot}, services.NewInterestEngine()),
		Now:       func() time.Time { return accrualDate },
	}

	result, err := job.RunDailyInterestAccrual(context.Background())
	if err != nil {
		t.Fatalf("RunDailyInterestAccrual() error = %v", err)
	}
	if result.Scanned != 1 || result.Posted != 1 || result.Skipped != 0 || result.Failed != 0 {
		t.Fatalf("result = %+v, want scanned=1 posted=1 skipped=0 failed=0", result)
	}
	if len(snapshot.createdTransactions) != 1 || len(snapshot.createdAccruals) != 1 {
		t.Fatalf("created transactions=%d accruals=%d, want 1 each", len(snapshot.createdTransactions), len(snapshot.createdAccruals))
	}
	if snapshot.createdAccruals[0].AccrualDate != dateOnly(accrualDate) {
		t.Fatalf("accrual date = %s, want %s", snapshot.createdAccruals[0].AccrualDate, dateOnly(accrualDate))
	}
}

func TestMonthlyInterestAccrualJobSkipsBeforePayableDate(t *testing.T) {
	t.Parallel()

	accrualDate := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)
	rule := testInterestRule(models.AccrualFrequencyMonthly, nil)
	snapshot := &fakeInterestSnapshot{}
	job := &InterestJob{
		Rules: &fakeInterestRuleJobRepo{
			targets: []repository.InterestRuleJobTarget{{Rule: rule, OwnerUserID: "user-1"}},
		},
		Lifecycle: services.NewInterestLifecycle(&fakeInterestAccrualTxRepo{snapshot: snapshot}, services.NewInterestEngine()),
		Now:       func() time.Time { return accrualDate },
	}

	result, err := job.RunMonthlyInterestAccrual(context.Background())
	if err != nil {
		t.Fatalf("RunMonthlyInterestAccrual() error = %v", err)
	}
	if result.Scanned != 1 || result.Posted != 0 || result.Skipped != 1 || result.Failed != 0 {
		t.Fatalf("result = %+v, want scanned=1 posted=0 skipped=1 failed=0", result)
	}
	if snapshot.locked {
		t.Fatal("snapshot lock was taken for a non-payable monthly rule")
	}
}

func TestDepositMaturityCheckJobPostsEndOfTermRule(t *testing.T) {
	t.Parallel()

	maturityDate := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)
	rule := testInterestRule(models.AccrualFrequencyEndOfTerm, &maturityDate)
	snapshot := &fakeInterestSnapshot{
		rules: []models.InterestRule{rule},
		transactions: []models.Transaction{
			{
				ID:         "tx-initial",
				AccountID:  rule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("50000"),
				OccurredAt: maturityDate.AddDate(0, 0, -30),
				CreatedAt:  maturityDate.AddDate(0, 0, -30),
			},
		},
	}
	job := &InterestJob{
		Rules: &fakeInterestRuleJobRepo{
			targets: []repository.InterestRuleJobTarget{{Rule: rule, OwnerUserID: "user-1"}},
		},
		Lifecycle: services.NewInterestLifecycle(&fakeInterestAccrualTxRepo{snapshot: snapshot}, services.NewInterestEngine()),
		Now:       func() time.Time { return maturityDate },
	}

	result, err := job.RunDepositMaturityCheck(context.Background())
	if err != nil {
		t.Fatalf("RunDepositMaturityCheck() error = %v", err)
	}
	if result.Posted != 1 || result.Skipped != 0 || result.Failed != 0 {
		t.Fatalf("result = %+v, want posted=1 skipped=0 failed=0", result)
	}
}

func TestInterestJobContinuesAfterTargetFailure(t *testing.T) {
	t.Parallel()

	accrualDate := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)
	rule := testInterestRule(models.AccrualFrequencyDaily, nil)
	job := &InterestJob{
		Rules: &fakeInterestRuleJobRepo{
			targets: []repository.InterestRuleJobTarget{{Rule: rule, OwnerUserID: "user-1"}},
		},
		Lifecycle: services.NewInterestLifecycle(&fakeInterestAccrualTxRepo{lockErr: errors.New("lock failed")}, services.NewInterestEngine()),
		Now:       func() time.Time { return accrualDate },
	}

	result, err := job.RunDailyInterestAccrual(context.Background())
	if err == nil {
		t.Fatal("RunDailyInterestAccrual() error = nil, want error")
	}
	if result.Scanned != 1 || result.Posted != 0 || result.Skipped != 0 || result.Failed != 1 {
		t.Fatalf("result = %+v, want scanned=1 failed=1", result)
	}
}

func TestDailyInterestAccrualJobUsesOnlyLatestOverlappingRule(t *testing.T) {
	t.Parallel()

	accrualDate := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)
	oldRule := testInterestRule(models.AccrualFrequencyDaily, nil)
	oldRule.ID = "rule-old"
	oldRule.StartDate = accrualDate.AddDate(0, 0, -10)
	latestRule := testInterestRule(models.AccrualFrequencyDaily, nil)
	latestRule.ID = "rule-latest"
	latestRule.StartDate = accrualDate.AddDate(0, 0, -1)

	snapshot := &fakeInterestSnapshot{
		rules: []models.InterestRule{latestRule},
		transactions: []models.Transaction{
			{
				ID:         "tx-initial",
				AccountID:  latestRule.AccountID,
				Type:       models.TransactionTypeInitialBalance,
				Amount:     dec("100000"),
				OccurredAt: accrualDate.AddDate(0, 0, -30),
				CreatedAt:  accrualDate.AddDate(0, 0, -30),
			},
		},
	}
	job := &InterestJob{
		Rules: &fakeInterestRuleJobRepo{
			targets: []repository.InterestRuleJobTarget{
				{Rule: oldRule, OwnerUserID: "user-1"},
				{Rule: latestRule, OwnerUserID: "user-1"},
			},
		},
		Lifecycle: services.NewInterestLifecycle(&fakeInterestAccrualTxRepo{snapshot: snapshot}, services.NewInterestEngine()),
		Now:       func() time.Time { return accrualDate },
	}

	result, err := job.RunDailyInterestAccrual(context.Background())
	if err != nil {
		t.Fatalf("RunDailyInterestAccrual() error = %v", err)
	}
	if result.Scanned != 1 || result.Posted != 1 || result.Skipped != 0 || result.Failed != 0 {
		t.Fatalf("result = %+v, want scanned=1 posted=1 skipped=0 failed=0", result)
	}
	if len(snapshot.createdAccruals) != 1 {
		t.Fatalf("created accruals = %d, want 1", len(snapshot.createdAccruals))
	}
	if snapshot.createdAccruals[0].RuleID != latestRule.ID {
		t.Fatalf("created accrual rule = %s, want %s", snapshot.createdAccruals[0].RuleID, latestRule.ID)
	}
}

func testInterestRule(frequency models.AccrualFrequency, endDate *time.Time) models.InterestRule {
	return models.InterestRule{
		ID:                      "rule-1",
		AccountID:               "account-1",
		AnnualRateBps:           1_200,
		AccrualFrequency:        frequency,
		CapitalizationFrequency: models.CapitalizationFrequencyNone,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:                 endDate,
	}
}

type fakeInterestRuleJobRepo struct {
	targets []repository.InterestRuleJobTarget
	err     error
}

func (r *fakeInterestRuleJobRepo) ListActiveForAccrual(context.Context, models.AccrualFrequency, time.Time) ([]repository.InterestRuleJobTarget, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]repository.InterestRuleJobTarget(nil), r.targets...), nil
}

type fakeInterestAccrualTxRepo struct {
	snapshot *fakeInterestSnapshot
	lockErr  error
}

func (r *fakeInterestAccrualTxRepo) WithAccountInterestLock(ctx context.Context, _, _ string, fn func(context.Context, repository.InterestCalculationRepository) error) error {
	if r.lockErr != nil {
		return r.lockErr
	}
	if r.snapshot == nil {
		r.snapshot = &fakeInterestSnapshot{}
	}
	r.snapshot.locked = true
	return fn(ctx, r.snapshot)
}

type fakeInterestSnapshot struct {
	locked              bool
	rules               []models.InterestRule
	transactions        []models.Transaction
	accruals            []models.InterestAccrual
	createdTransactions []models.Transaction
	createdAccruals     []models.InterestAccrual
}

func (s *fakeInterestSnapshot) GetInterestRuleByID(_ context.Context, id string) (*models.InterestRule, error) {
	for i := range s.rules {
		if s.rules[i].ID == id {
			return &s.rules[i], nil
		}
	}
	return nil, repository.ErrNotFound
}

func (s *fakeInterestSnapshot) ListInterestRulesByAccount(_ context.Context, accountID string) ([]models.InterestRule, error) {
	var rules []models.InterestRule
	for i := range s.rules {
		if s.rules[i].AccountID == accountID {
			rules = append(rules, s.rules[i])
		}
	}
	return rules, nil
}

func (s *fakeInterestSnapshot) ListTransactionsByAccountForUser(context.Context, string, string) ([]models.Transaction, error) {
	return append([]models.Transaction(nil), s.transactions...), nil
}

func (s *fakeInterestSnapshot) ListInterestAccrualsByAccount(context.Context, string) ([]models.InterestAccrual, error) {
	return append([]models.InterestAccrual(nil), s.accruals...), nil
}

func (s *fakeInterestSnapshot) CreateInterestAccrualWithTransaction(_ context.Context, transaction *models.Transaction, accrual *models.InterestAccrual) error {
	s.createdTransactions = append(s.createdTransactions, *transaction)
	s.createdAccruals = append(s.createdAccruals, *accrual)
	return nil
}

func (s *fakeInterestSnapshot) ReplaceInterestAccrualRangeWithTransactions(context.Context, string, string, time.Time, time.Time, []models.Transaction, []models.InterestAccrual) (int64, error) {
	return 0, nil
}

func TestSelectLatestRulesForAccounts(t *testing.T) {
	now := time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC)

	rule1 := models.InterestRule{
		ID:        "rule1",
		AccountID: "acc1",
		StartDate: now.AddDate(0, 0, -5),
	}
	rule2 := models.InterestRule{
		ID:        "rule2",
		AccountID: "acc1",
		StartDate: now.AddDate(0, 0, -1),
	}
	rule3 := models.InterestRule{
		ID:        "rule3",
		AccountID: "acc2",
		StartDate: now.AddDate(0, 0, -3),
	}
	rule4 := models.InterestRule{
		ID:        "rule4",
		AccountID: "acc2",
		StartDate: now.AddDate(0, 0, 1), // завтра
	}

	targets := []repository.InterestRuleJobTarget{
		{Rule: rule1, OwnerUserID: "user1"},
		{Rule: rule2, OwnerUserID: "user1"},
		{Rule: rule3, OwnerUserID: "user2"},
		{Rule: rule4, OwnerUserID: "user2"},
	}

	selected := selectLatestRulesForAccounts(targets, now)

	if len(selected) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(selected))
	}
	acc1Found := false
	acc2Found := false
	for _, target := range selected {
		switch target.Rule.AccountID {
		case "acc1":
			acc1Found = true
			if target.Rule.ID != "rule2" {
				t.Errorf("for acc1 expected rule2, got %s", target.Rule.ID)
			}
		case "acc2":
			acc2Found = true
			if target.Rule.ID != "rule3" {
				t.Errorf("for acc2 expected rule3, got %s", target.Rule.ID)
			}
		}
	}
	if !acc1Found || !acc2Found {
		t.Fatalf("missing expected accounts in selection")
	}
}
