package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestCreateListAndUpdateCategoryLimit(t *testing.T) {
	const categoryID = "11111111-1111-1111-1111-111111111111"
	tokens, pair := testProfileTokenPair(t)
	limits := &testCategoryLimitRepo{}
	store := newTestProfileStore()
	store.limits = limits
	store.categories = categoryLimitCategoryRepo{category: &models.Category{ID: categoryID, Name: "Food"}}
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")
	router := NewRouter(store, &RouterConfig{TokenService: tokens})

	create := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/category-limits", strings.NewReader(`{"category_id":"11111111-1111-1111-1111-111111111111","amount":"100000","currency":"rub"}`))
	create.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	create.Header.Set("Idempotency-Key", "create-limit")
	created := httptest.NewRecorder()
	router.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", created.Code, created.Body.String())
	}
	if len(limits.limits) != 1 || limits.limits[0].Currency != "RUB" || !limits.limits[0].IsActive {
		t.Fatalf("created limit = %#v", limits.limits)
	}

	id := limits.limits[0].ID
	update := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/category-limits/"+id, strings.NewReader(`{"amount":"120000","is_active":false}`))
	update.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	updated := httptest.NewRecorder()
	router.ServeHTTP(updated, update)
	if updated.Code != http.StatusOK {
		t.Fatalf("update status = %d: %s", updated.Code, updated.Body.String())
	}
	if limits.limits[0].Amount.String() != "120000" || limits.limits[0].IsActive {
		t.Fatalf("updated limit = %#v", limits.limits[0])
	}

	list := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/category-limits", nil)
	list.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	listed := httptest.NewRecorder()
	router.ServeHTTP(listed, list)
	if listed.Code != http.StatusOK || limits.listUserID != "user-1" {
		t.Fatalf("list status = %d, user = %q", listed.Code, limits.listUserID)
	}
}

type testCategoryLimitRepo struct {
	limits     []models.CategoryLimit
	listUserID string
}

func (r *testCategoryLimitRepo) Create(_ context.Context, limit *models.CategoryLimit) error {
	r.limits = append(r.limits, *limit)
	return nil
}

func (r *testCategoryLimitRepo) GetByIDForUser(_ context.Context, id, userID string) (*models.CategoryLimit, error) {
	for i := range r.limits {
		if r.limits[i].ID == id && r.limits[i].OwnerUserID == userID {
			clone := r.limits[i]
			return &clone, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *testCategoryLimitRepo) ListByUser(_ context.Context, userID string) ([]models.CategoryLimit, error) {
	r.listUserID = userID
	return r.limits, nil
}

func (r *testCategoryLimitRepo) UpdateForUser(_ context.Context, limit *models.CategoryLimit, userID string) error {
	for i := range r.limits {
		if r.limits[i].ID == limit.ID && r.limits[i].OwnerUserID == userID {
			r.limits[i] = *limit
			return nil
		}
	}
	return repository.ErrNotFound
}

type categoryLimitCategoryRepo struct{ category *models.Category }

func (r categoryLimitCategoryRepo) Create(context.Context, *models.Category) error { return nil }
func (r categoryLimitCategoryRepo) GetByID(_ context.Context, id string) (*models.Category, error) {
	if r.category != nil && r.category.ID == id {
		clone := *r.category
		return &clone, nil
	}
	return nil, repository.ErrNotFound
}

func (categoryLimitCategoryRepo) GetBySlug(context.Context, string) (*models.Category, error) {
	return nil, repository.ErrNotFound
}

func (r categoryLimitCategoryRepo) List(context.Context) ([]models.Category, error) {
	if r.category == nil {
		return nil, nil
	}
	return []models.Category{*r.category}, nil
}
