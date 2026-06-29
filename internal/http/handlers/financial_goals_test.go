package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestCreateAndListFinancialGoals(t *testing.T) {
	const accountID = "11111111-1111-1111-1111-111111111111"
	tokens, pair := testProfileTokenPair(t)
	repo := &testFinancialGoalRepo{}
	store := newTestProfileStore()
	store.goals = repo
	store.accounts = &testAccountRepo{byID: map[string]*models.Account{
		accountID: testAccount(accountID, "user-1", "RUB"),
	}}
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")
	router := NewRouter(store, &RouterConfig{TokenService: tokens})

	create := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/financial-goals", strings.NewReader(`{
		"account_id":"11111111-1111-1111-1111-111111111111","name":"Emergency fund","target_amount":"300000","target_date":"2027-01-01"
	}`))
	create.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	create.Header.Set("Idempotency-Key", "create-goal-1")
	created := httptest.NewRecorder()
	router.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", created.Code, http.StatusCreated, created.Body.String())
	}
	var response dto.FinancialGoalResponse
	if err := json.NewDecoder(created.Body).Decode(&response); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if response.Currency != "RUB" || response.TargetDate == nil || *response.TargetDate != "2027-01-01" {
		t.Fatalf("created goal = %#v", response)
	}

	list := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/financial-goals", nil)
	list.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	listed := httptest.NewRecorder()
	router.ServeHTTP(listed, list)
	if listed.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d: %s", listed.Code, http.StatusOK, listed.Body.String())
	}
	if repo.listUserID != "user-1" {
		t.Fatalf("list user = %q, want user-1", repo.listUserID)
	}
}

func TestCreateFinancialGoalRejectsInvalidAmount(t *testing.T) {
	const accountID = "11111111-1111-1111-1111-111111111111"
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	store.goals = &testFinancialGoalRepo{}
	store.accounts = &testAccountRepo{byID: map[string]*models.Account{
		accountID: testAccount(accountID, "user-1", "RUB"),
	}}
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")
	router := NewRouter(store, &RouterConfig{TokenService: tokens})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/financial-goals", strings.NewReader(`{"account_id":"11111111-1111-1111-1111-111111111111","name":"Goal","target_amount":"0","target_date":""}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Idempotency-Key", "invalid-goal")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestCreateFinancialGoalRejectsForeignAccount(t *testing.T) {
	const accountID = "11111111-1111-1111-1111-111111111111"
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	store.goals = &testFinancialGoalRepo{}
	store.accounts = &testAccountRepo{byID: map[string]*models.Account{
		accountID: testAccount(accountID, "other-user", "RUB"),
	}}
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")
	router := NewRouter(store, &RouterConfig{TokenService: tokens})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/financial-goals", strings.NewReader(`{"account_id":"11111111-1111-1111-1111-111111111111","name":"Goal","target_amount":"100"}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Idempotency-Key", "foreign-account-goal")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestUpdateFinancialGoalRoute(t *testing.T) {
	const (
		goalID    = "31765cde-dce6-531e-9bc3-b048a0bfc14b"
		accountID = "11111111-1111-1111-1111-111111111111"
	)
	tokens, pair := testProfileTokenPair(t)
	accountIDValue := accountID
	repo := &testFinancialGoalRepo{goals: []models.FinancialGoal{{
		ID:           goalID,
		OwnerUserID:  "user-1",
		AccountID:    &accountIDValue,
		Name:         "Emergency fund",
		TargetAmount: dec("300000"),
		Currency:     "RUB",
		Status:       models.FinancialGoalActive,
	}}}
	store := newTestProfileStore()
	store.goals = repo
	store.accounts = &testAccountRepo{byID: map[string]*models.Account{
		accountID: testAccount(accountID, "user-1", "RUB"),
	}}
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")
	router := NewRouter(store, &RouterConfig{TokenService: tokens})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/financial-goals/"+goalID, strings.NewReader(`{
		"account_id":"11111111-1111-1111-1111-111111111111",
		"name":"Travel fund",
		"target_amount":"360000",
		"target_date":"2027-06-30",
		"status":"completed"
	}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Idempotency-Key", "update-goal-route")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response dto.FinancialGoalResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if response.Name != "Travel fund" || response.TargetAmount != "360000" || response.Status != models.FinancialGoalCompleted {
		t.Fatalf("updated goal = %#v", response)
	}
}

type testFinancialGoalRepo struct {
	goals      []models.FinancialGoal
	listUserID string
}

func (r *testFinancialGoalRepo) Create(_ context.Context, goal *models.FinancialGoal) error {
	r.goals = append(r.goals, *goal)
	return nil
}

func (r *testFinancialGoalRepo) ListByUser(_ context.Context, userID string) ([]models.FinancialGoal, error) {
	r.listUserID = userID
	return r.goals, nil
}

func (r *testFinancialGoalRepo) GetByIDForUser(_ context.Context, id, userID string) (*models.FinancialGoal, error) {
	for i := range r.goals {
		if r.goals[i].ID == id && r.goals[i].OwnerUserID == userID {
			goal := r.goals[i]
			return &goal, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *testFinancialGoalRepo) UpdateForUser(_ context.Context, goal *models.FinancialGoal, userID string) error {
	for i := range r.goals {
		if r.goals[i].ID == goal.ID && r.goals[i].OwnerUserID == userID {
			r.goals[i] = *goal
			return nil
		}
	}
	return repository.ErrNotFound
}
