package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestPersistCalculatedAccrualOwnsConflictHandling(t *testing.T) {
	wantErr := errors.New("write failed")
	tests := []struct {
		name         string
		writeErr     error
		wantConflict bool
		wantErr      error
	}{
		{name: "success"},
		{name: "duplicate is skipped", writeErr: repository.ErrConflict, wantConflict: true},
		{name: "write failure", writeErr: wantErr, wantErr: wantErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &commandAccrualWriter{createErr: tt.writeErr}
			conflict, err := persistCalculatedAccrual(t.Context(), writer, &models.Transaction{ID: "tx-1"}, &models.InterestAccrual{ID: "accrual-1"})
			if conflict != tt.wantConflict || !errors.Is(err, tt.wantErr) {
				t.Fatalf("conflict = %t, error = %v", conflict, err)
			}
			if writer.created == nil || writer.created.ID != "accrual-1" {
				t.Fatalf("created accrual = %#v", writer.created)
			}
		})
	}
}

func TestResolveOwnerUserID(t *testing.T) {
	tests := []struct {
		name    string
		users   map[string]*models.User
		input   string
		want    string
		wantErr string
	}{
		{name: "single user remains implicit", users: userMap("user-1"), want: ""},
		{name: "multiple users require owner", users: userMap("user-1", "user-2"), wantErr: "owner-user-id is required"},
		{name: "before setup remains unowned", users: userMap(), want: ""},
		{name: "provided owner is validated", users: userMap("user-1"), input: " user-1 ", want: "user-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveOwnerUserID(t.Context(), &commandUserRepo{byID: tt.users}, tt.input)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("owner = %q, error = %v, want %q", got, err, tt.want)
			}
		})
	}
}

func userMap(ids ...string) map[string]*models.User {
	users := make(map[string]*models.User, len(ids))
	for _, id := range ids {
		users[id] = &models.User{ID: id, Email: id + "@example.com"}
	}
	return users
}

type commandUserRepo struct{ byID map[string]*models.User }

type commandAccrualWriter struct {
	createErr error
	created   *models.InterestAccrual
}

func (w *commandAccrualWriter) CreateWithTransaction(_ context.Context, _ *models.Transaction, accrual *models.InterestAccrual) error {
	w.created = accrual
	return w.createErr
}

func (*commandAccrualWriter) ReplaceRangeWithTransactions(context.Context, string, string, time.Time, time.Time, []models.Transaction, []models.InterestAccrual) (int64, error) {
	return 0, nil
}

func (r *commandUserRepo) Create(_ context.Context, user *models.User) error {
	r.byID[user.ID] = user
	return nil
}
func (r *commandUserRepo) Count(context.Context) (int64, error) { return int64(len(r.byID)), nil }
func (r *commandUserRepo) GetByEmail(_ context.Context, email string) (*models.User, error) {
	for _, user := range r.byID {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *commandUserRepo) GetByID(_ context.Context, id string) (*models.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return user, nil
}

func (r *commandUserRepo) RecordLoginFailure(context.Context, string, int, []time.Duration, time.Time) (int, *time.Time, error) {
	return 0, nil, nil
}

func (r *commandUserRepo) ClearLoginFailures(context.Context, string, time.Time) error { return nil }

func (r *commandUserRepo) UpdatePassword(context.Context, string, string, time.Time) error {
	return nil
}

func (r *commandUserRepo) ChangePasswordAndRevokeSessions(context.Context, string, string, time.Time, string) error {
	return nil
}

func (r *commandUserRepo) UpdatePrimaryCurrency(context.Context, string, string, time.Time) error {
	return nil
}
