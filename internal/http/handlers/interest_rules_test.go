package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestListUserInterestRulesUsesCurrentUserScope(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	rule := &models.InterestRule{
		ID:                      "22222222-2222-2222-2222-222222222222",
		AccountID:               "11111111-1111-1111-1111-111111111111",
		AnnualRateBps:           1_200,
		AccrualFrequency:        models.AccrualFrequencyDaily,
		CapitalizationFrequency: models.CapitalizationFrequencyNone,
		DayCountConvention:      models.DayCountConventionActual365,
		IsActive:                true,
		StartDate:               time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	rules := &testInterestRuleRepo{rule: rule}
	store := newTestProfileStore()
	store.rules = rules
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")

	router := newTestRouter(store, &RouterConfig{}, tokens)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/interest-rules", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rules.listByUserID != "user-1" {
		t.Fatalf("ListByUser user id = %q, want user-1", rules.listByUserID)
	}
}

type testInterestRuleRepo struct {
	rule         *models.InterestRule
	listByUserID string
}

func (r *testInterestRuleRepo) Create(context.Context, *models.InterestRule) error {
	return nil
}

func (r *testInterestRuleRepo) GetByID(_ context.Context, id string) (*models.InterestRule, error) {
	if r.rule != nil && r.rule.ID == id {
		return r.rule, nil
	}
	return nil, repository.ErrNotFound
}

func (r *testInterestRuleRepo) ListByAccount(_ context.Context, accountID string) ([]models.InterestRule, error) {
	if r.rule != nil && r.rule.AccountID == accountID {
		return []models.InterestRule{*r.rule}, nil
	}
	return nil, nil
}

func (r *testInterestRuleRepo) ListByUser(_ context.Context, userID string) ([]models.InterestRule, error) {
	r.listByUserID = userID
	if r.rule != nil {
		return []models.InterestRule{*r.rule}, nil
	}
	return nil, nil
}

func (r *testInterestRuleRepo) Update(context.Context, *models.InterestRule) error {
	return nil
}
