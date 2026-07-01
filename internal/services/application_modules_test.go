package services

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

func TestCategoryServiceOwnsNormalizationAndValidation(t *testing.T) {
	repo := &moduleCategoryRepo{}
	category, err := NewCategoryService(repo).Create(t.Context(), &CreateCategoryRequest{Name: " Home repair ", Slug: " HOME-REPAIR "})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	if category.Name != "Home repair" || category.Slug != "home-repair" || repo.created == nil {
		t.Fatalf("category = %#v", category)
	}
	if _, err := NewCategoryService(repo).Create(t.Context(), &CreateCategoryRequest{Name: "Bad", Slug: "bad slug"}); !IsValidationError(err) {
		t.Fatalf("invalid slug error = %v", err)
	}
}

func TestFinancialGoalServiceOwnsAccountCurrencyAndProgression(t *testing.T) {
	accounts := &recordingAccountRepo{existing: &models.Account{ID: "account-1", Currency: "RUB"}}
	goals := &moduleGoalRepo{}
	service := NewFinancialGoalService(goals, accounts)
	goal, err := service.Create(t.Context(), &CreateFinancialGoalRequest{
		UserID: "user-1", AccountID: "account-1", Name: " Reserve ", TargetAmount: decimal.RequireFromString("1000"),
	})
	if err != nil {
		t.Fatalf("create goal: %v", err)
	}
	if goal.Currency != "RUB" || goal.Name != "Reserve" || goals.goal == nil {
		t.Fatalf("goal = %#v", goal)
	}
	status := models.FinancialGoalCompleted
	updated, err := service.Update(t.Context(), &UpdateFinancialGoalRequest{ID: goal.ID, UserID: "user-1", Status: &status})
	if err != nil {
		t.Fatalf("update goal: %v", err)
	}
	if updated.Status != models.FinancialGoalCompleted {
		t.Fatalf("status = %s", updated.Status)
	}
}

func TestCategoryLimitServiceOwnsCurrencyValidation(t *testing.T) {
	categories := &moduleCategoryRepo{category: &models.Category{ID: "category-1"}}
	limits := &moduleLimitRepo{}
	service := NewCategoryLimitService(limits, categories)
	limit, err := service.Create(t.Context(), &CreateCategoryLimitRequest{
		UserID: "user-1", CategoryID: "category-1", Amount: decimal.RequireFromString("100"), Currency: "rub",
	})
	if err != nil {
		t.Fatalf("create limit: %v", err)
	}
	if limit.Currency != "RUB" || !limit.IsActive {
		t.Fatalf("limit = %#v", limit)
	}
	bad := "RU"
	if _, err := service.Update(t.Context(), &UpdateCategoryLimitRequest{ID: limit.ID, UserID: "user-1", Currency: &bad}); !IsValidationError(err) {
		t.Fatalf("invalid currency error = %v", err)
	}
}

type moduleCategoryRepo struct {
	category *models.Category
	created  *models.Category
}

func (r *moduleCategoryRepo) Create(_ context.Context, category *models.Category) error {
	clone := *category
	r.created = &clone
	r.category = &clone
	return nil
}

func (r *moduleCategoryRepo) GetByID(_ context.Context, id string) (*models.Category, error) {
	if r.category == nil || r.category.ID != id {
		return nil, repository.ErrNotFound
	}
	clone := *r.category
	return &clone, nil
}

func (r *moduleCategoryRepo) GetBySlug(_ context.Context, slug string) (*models.Category, error) {
	if r.category == nil || r.category.Slug != slug {
		return nil, repository.ErrNotFound
	}
	clone := *r.category
	return &clone, nil
}

func (r *moduleCategoryRepo) List(context.Context) ([]models.Category, error) {
	if r.category == nil {
		return nil, nil
	}
	return []models.Category{*r.category}, nil
}

type moduleGoalRepo struct{ goal *models.FinancialGoal }

func (r *moduleGoalRepo) Create(_ context.Context, goal *models.FinancialGoal) error {
	clone := *goal
	r.goal = &clone
	return nil
}

func (r *moduleGoalRepo) GetByIDForUser(_ context.Context, id, userID string) (*models.FinancialGoal, error) {
	if r.goal == nil || r.goal.ID != id || r.goal.OwnerUserID != userID {
		return nil, repository.ErrNotFound
	}
	clone := *r.goal
	return &clone, nil
}

func (r *moduleGoalRepo) ListByUser(context.Context, string) ([]models.FinancialGoal, error) {
	if r.goal == nil {
		return nil, nil
	}
	return []models.FinancialGoal{*r.goal}, nil
}

func (r *moduleGoalRepo) UpdateForUser(_ context.Context, goal *models.FinancialGoal, _ string) error {
	clone := *goal
	r.goal = &clone
	return nil
}

type moduleLimitRepo struct{ limit *models.CategoryLimit }

func (r *moduleLimitRepo) Create(_ context.Context, limit *models.CategoryLimit) error {
	clone := *limit
	r.limit = &clone
	return nil
}

func (r *moduleLimitRepo) GetByIDForUser(_ context.Context, id, userID string) (*models.CategoryLimit, error) {
	if r.limit == nil || r.limit.ID != id || r.limit.OwnerUserID != userID {
		return nil, repository.ErrNotFound
	}
	clone := *r.limit
	return &clone, nil
}

func (r *moduleLimitRepo) ListByUser(context.Context, string) ([]models.CategoryLimit, error) {
	if r.limit == nil {
		return nil, nil
	}
	return []models.CategoryLimit{*r.limit}, nil
}

func (r *moduleLimitRepo) UpdateForUser(_ context.Context, limit *models.CategoryLimit, _ string) error {
	clone := *limit
	r.limit = &clone
	return nil
}
