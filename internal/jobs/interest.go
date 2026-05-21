package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
	"github.com/sunriseex/capitalflow/internal/services"
)

const (
	DailyInterestAccrualJobName   = "daily_interest_accrual_job"
	MonthlyInterestAccrualJobName = "monthly_interest_accrual_job"
	DepositMaturityCheckJobName   = "deposit_maturity_check_job"
)

type InterestJob struct {
	Rules    repository.InterestRuleJobRepository
	Accruals repository.InterestAccrualTransactionalRepository
	Logger   *slog.Logger
	Now      func() time.Time
}

type InterestJobRunResult struct {
	JobName     string
	AccrualDate time.Time
	Scanned     int
	Posted      int
	Skipped     int
	Failed      int
}

func (j *InterestJob) RunDailyInterestAccrual(ctx context.Context) (*InterestJobRunResult, error) {
	return j.run(ctx, DailyInterestAccrualJobName, models.AccrualFrequencyDaily, j.today())
}

func (j *InterestJob) RunMonthlyInterestAccrual(ctx context.Context) (*InterestJobRunResult, error) {
	return j.run(ctx, MonthlyInterestAccrualJobName, models.AccrualFrequencyMonthly, j.today())
}

func (j *InterestJob) RunDepositMaturityCheck(ctx context.Context) (*InterestJobRunResult, error) {
	return j.run(ctx, DepositMaturityCheckJobName, models.AccrualFrequencyEndOfTerm, j.today())
}

func (j *InterestJob) run(ctx context.Context, jobName string, frequency models.AccrualFrequency, accrualDate time.Time) (*InterestJobRunResult, error) {
	if j.Rules == nil {
		return nil, fmt.Errorf("%s: interest rule repository is required", jobName)
	}
	if j.Accruals == nil {
		return nil, fmt.Errorf("%s: interest accrual repository is required", jobName)
	}

	accrualDate = dateOnly(accrualDate)
	targets, err := j.Rules.ListActiveForAccrual(ctx, frequency, accrualDate)
	if err != nil {
		return nil, fmt.Errorf("%s: list active rules: %w", jobName, err)
	}

	result := &InterestJobRunResult{
		JobName:     jobName,
		AccrualDate: accrualDate,
		Scanned:     len(targets),
	}

	var runErrs []error
	for i := range targets {
		target := &targets[i]

		if err := ctx.Err(); err != nil {
			return result, fmt.Errorf("%s: %w", jobName, err)
		}

		if !rulePayableOn(&target.Rule, accrualDate) {
			result.Skipped++
			continue
		}

		posted, err := j.accrueTarget(ctx, target, accrualDate)
		if err != nil {
			result.Failed++
			runErrs = append(runErrs, err)
			j.logger().Warn("interest job target failed",
				"job", jobName,
				"rule_id", target.Rule.ID,
				"account_id", target.Rule.AccountID,
				"error", err,
			)
			continue
		}
		if posted {
			result.Posted++
		} else {
			result.Skipped++
		}
	}

	j.logger().Info("interest job finished", "job", jobName, "date", accrualDate.Format(time.DateOnly), "scanned", result.Scanned, "posted", result.Posted, "skipped", result.Skipped, "failed", result.Failed)

	if len(runErrs) > 0 {
		return result, fmt.Errorf("%s: %w", jobName, errors.Join(runErrs...))
	}
	return result, nil
}

func (j *InterestJob) accrueTarget(ctx context.Context, target *repository.InterestRuleJobTarget, accrualDate time.Time) (bool, error) {
	var posted bool
	err := j.Accruals.WithAccountInterestLock(ctx, target.Rule.AccountID, target.OwnerUserID, func(ctx context.Context, snapshot repository.InterestCalculationRepository) error {
		transactions, err := snapshot.ListTransactionsByAccountForUser(ctx, target.Rule.AccountID, target.OwnerUserID)
		if err != nil {
			return fmt.Errorf("list account transactions: %w", err)
		}
		accruals, err := snapshot.ListInterestAccrualsByAccount(ctx, target.Rule.AccountID)
		if err != nil {
			return fmt.Errorf("list account accruals: %w", err)
		}

		principal := services.PrincipalTransactionsForRuleAt(transactions, accruals, &target.Rule, accrualDate)
		balance, err := services.NewBalanceService().Calculate(ctx, services.CalculateBalanceRequest{
			AccountID:    target.Rule.AccountID,
			Transactions: transactionsUpToDate(principal, accrualDate),
		})
		if err != nil {
			return fmt.Errorf("calculate account balance: %w", err)
		}

		response, err := services.NewInterestRuleService(nil).Accrue(ctx, &services.AccrueRuleInterestRequest{
			Rule:             target.Rule,
			BalanceMinor:     balance.BalanceMinor,
			AccrualDate:      accrualDate,
			Transactions:     principal,
			ExistingAccruals: accruals,
		})
		if err != nil {
			if services.IsValidationError(err) {
				return nil
			}
			return fmt.Errorf("accrue for rule %s account %s: %w", target.Rule.ID, target.Rule.AccountID, err)
		}
		if response.Skipped {
			return nil
		}
		if err := snapshot.CreateInterestAccrualWithTransaction(ctx, response.Transaction, response.Accrual); err != nil {
			return fmt.Errorf("create interest accrual: %w", err)
		}
		posted = true
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("accrue rule %s account %s: %w", target.Rule.ID, target.Rule.AccountID, err)
	}
	return posted, nil
}

func (j *InterestJob) today() time.Time {
	if j.Now != nil {
		return dateOnly(j.Now())
	}
	return dateOnly(time.Now())
}

func (j *InterestJob) logger() *slog.Logger {
	if j.Logger != nil {
		return j.Logger
	}
	return slog.Default()
}

func rulePayableOn(rule *models.InterestRule, date time.Time) bool {
	date = dateOnly(date)
	if date.Before(dateOnly(rule.StartDate)) {
		return false
	}
	if rule.EndDate != nil && date.After(dateOnly(*rule.EndDate)) {
		return false
	}

	switch rule.AccrualFrequency {
	case models.AccrualFrequencyMonthly:
		return lastActiveDayOfMonth(rule, date).Equal(date)
	case models.AccrualFrequencyEndOfTerm:
		return rule.EndDate != nil && dateOnly(*rule.EndDate).Equal(date)
	default:
		return true
	}
}

func lastActiveDayOfMonth(rule *models.InterestRule, date time.Time) time.Time {
	date = dateOnly(date)
	monthEnd := time.Date(date.Year(), date.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	if rule.EndDate != nil && dateOnly(*rule.EndDate).Before(monthEnd) {
		return dateOnly(*rule.EndDate)
	}
	return monthEnd
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
