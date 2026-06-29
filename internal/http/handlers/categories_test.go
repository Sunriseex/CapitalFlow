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

func TestCategoriesRouteRequiresAuth(t *testing.T) {
	router := newTestRouter(nil, &RouterConfig{APIAuthToken: "01234567890123456789012345678901"})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/categories", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestCreateCategory(t *testing.T) {
	tokens, pair := testProfileTokenPair(t)
	store := newTestProfileStore()
	repo := &testCategoryRepo{}
	store.categories = repo
	store.refresh.byID[pair.RefreshTokenID] = activeTestRefreshToken(pair, "user-1")
	router := newTestRouter(store, &RouterConfig{}, tokens)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/categories", strings.NewReader(`{"name":"Home repair","slug":"home-repair"}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Idempotency-Key", "create-category-1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if repo.created == nil || repo.created.Name != "Home repair" || repo.created.Slug != "home-repair" {
		t.Fatalf("created category = %#v", repo.created)
	}
}

type testCategoryRepo struct{ created *models.Category }

func (r *testCategoryRepo) Create(_ context.Context, category *models.Category) error {
	r.created = category
	return nil
}

func (r *testCategoryRepo) GetByID(context.Context, string) (*models.Category, error) {
	return nil, repository.ErrNotFound
}

func (r *testCategoryRepo) GetBySlug(context.Context, string) (*models.Category, error) {
	return nil, repository.ErrNotFound
}
func (r *testCategoryRepo) List(context.Context) ([]models.Category, error) { return nil, nil }

func TestCategoriesPreflightSkipsAuth(t *testing.T) {
	router := newTestRouter(nil, &RouterConfig{
		APIAuthToken:       "01234567890123456789012345678901",
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/api/v1/categories", http.NoBody)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("allow origin = %q", got)
	}
}
