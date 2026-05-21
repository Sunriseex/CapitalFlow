package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/config"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestRunTransactionsCreateRejectsTransferTypes(t *testing.T) {
	oldConfig := config.AppConfig
	config.AppConfig = &config.Config{
		DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
	}
	t.Cleanup(func() {
		config.AppConfig = oldConfig
	})

	tests := []struct {
		name            string
		transactionType string
	}{
		{
			name:            "transfer in",
			transactionType: "transfer_in",
		},
		{
			name:            "transfer out",
			transactionType: "transfer_out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runTransactionsCreate(context.Background(), []string{
				"--account", "account-1",
				"--type", tt.transactionType,
				"--amount", "100.00",
			})

			if err == nil {
				t.Fatal("expected error")
			}

			if !strings.Contains(err.Error(), "transfer transactions") {
				t.Fatalf("error = %q, want transfer rejection", err.Error())
			}
		})
	}
}

func TestRunJobsRunRejectsUnknownJobBeforeOpeningDatabase(t *testing.T) {
	oldConfig := config.AppConfig
	config.AppConfig = &config.Config{
		DatabaseURL: "postgres://invalid:invalid@127.0.0.1:1/invalid?sslmode=disable",
	}
	t.Cleanup(func() {
		config.AppConfig = oldConfig
	})

	err := runJobsRun(context.Background(), []string{"--name", "unknown_job"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown job name: unknown_job") {
		t.Fatalf("error = %q, want unknown job rejection", err.Error())
	}
}

func TestValidInterestJobName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "daily_interest_accrual_job", want: true},
		{name: "monthly_interest_accrual_job", want: true},
		{name: "deposit_maturity_check_job", want: true},
		{name: "unknown_job", want: false},
		{name: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validInterestJobName(tt.name); got != tt.want {
				t.Fatalf("validInterestJobName(%q) = %t, want %t", tt.name, got, tt.want)
			}
		})
	}
}

func TestResolveOwnerUserIDAllowsImplicitOwnerForSingleUser(t *testing.T) {
	users := &fakeCLIUserRepo{
		byID: map[string]*models.User{
			"user-1": {ID: "user-1", Email: "user@example.com"},
		},
	}

	ownerUserID, err := resolveOwnerUserID(t.Context(), users, "")
	if err != nil {
		t.Fatalf("resolve owner user id: %v", err)
	}
	if ownerUserID != "" {
		t.Fatalf("owner user id = %q, want empty for repository single-user fallback", ownerUserID)
	}
}

func TestResolveOwnerUserIDRequiresOwnerForMultipleUsers(t *testing.T) {
	users := &fakeCLIUserRepo{
		byID: map[string]*models.User{
			"user-1": {ID: "user-1", Email: "one@example.com"},
			"user-2": {ID: "user-2", Email: "two@example.com"},
		},
	}

	_, err := resolveOwnerUserID(t.Context(), users, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "owner-user-id is required when multiple users exist") {
		t.Fatalf("error = %q, want multiple-user owner requirement", err.Error())
	}
}

func TestResolveOwnerUserIDAllowsUnownedBeforeSetup(t *testing.T) {
	users := &fakeCLIUserRepo{byID: map[string]*models.User{}}

	ownerUserID, err := resolveOwnerUserID(t.Context(), users, "")
	if err != nil {
		t.Fatalf("resolve owner user id: %v", err)
	}
	if ownerUserID != "" {
		t.Fatalf("owner user id = %q, want empty", ownerUserID)
	}
}

func TestResolveOwnerUserIDValidatesProvidedOwner(t *testing.T) {
	users := &fakeCLIUserRepo{
		byID: map[string]*models.User{
			"user-1": {ID: "user-1", Email: "user@example.com"},
		},
	}

	ownerUserID, err := resolveOwnerUserID(t.Context(), users, " user-1 ")
	if err != nil {
		t.Fatalf("resolve owner user id: %v", err)
	}
	if ownerUserID != "user-1" {
		t.Fatalf("owner user id = %q, want user-1", ownerUserID)
	}
}

type fakeCLIUserRepo struct {
	byID map[string]*models.User
}

func (r *fakeCLIUserRepo) Create(_ context.Context, user *models.User) error {
	r.byID[user.ID] = user
	return nil
}

func (r *fakeCLIUserRepo) Count(context.Context) (int64, error) {
	return int64(len(r.byID)), nil
}

func (r *fakeCLIUserRepo) GetByEmail(_ context.Context, email string) (*models.User, error) {
	for _, user := range r.byID {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *fakeCLIUserRepo) GetByID(_ context.Context, id string) (*models.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return user, nil
}

func (r *fakeCLIUserRepo) RecordLoginFailure(_ context.Context, id string, threshold int, delays []time.Duration, updatedAt time.Time) (int, *time.Time, error) {
	user, ok := r.byID[id]
	if !ok {
		return 0, nil, repository.ErrNotFound
	}
	attempts := user.FailedLoginAttempts + 1
	var lockedUntil *time.Time
	if attempts >= threshold && len(delays) > 0 {
		delayIndex := min(attempts-threshold, len(delays)-1)
		lockoutUntil := updatedAt.Add(delays[delayIndex])
		lockedUntil = &lockoutUntil
	}
	user.FailedLoginAttempts = attempts
	user.LockedUntil = lockedUntil
	user.UpdatedAt = updatedAt
	return attempts, lockedUntil, nil
}

func (r *fakeCLIUserRepo) ClearLoginFailures(_ context.Context, id string, updatedAt time.Time) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	user.UpdatedAt = updatedAt
	return nil
}

func (r *fakeCLIUserRepo) UpdatePassword(_ context.Context, id, passwordHash string, updatedAt time.Time) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.PasswordHash = passwordHash
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	user.UpdatedAt = updatedAt
	return nil
}

func (r *fakeCLIUserRepo) ChangePasswordAndRevokeSessions(ctx context.Context, id, passwordHash string, updatedAt time.Time, _ string) error {
	return r.UpdatePassword(ctx, id, passwordHash, updatedAt)
}

func (r *fakeCLIUserRepo) UpdatePrimaryCurrency(_ context.Context, id, primaryCurrency string, updatedAt time.Time) error {
	user, ok := r.byID[id]
	if !ok {
		return repository.ErrNotFound
	}
	user.PrimaryCurrency = primaryCurrency
	user.UpdatedAt = updatedAt
	return nil
}
