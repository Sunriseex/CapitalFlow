package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

var categorySlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`)

type CategoryService struct {
	repo repository.CategoryRepository
}

type CreateCategoryRequest struct {
	Name string
	Slug string
}

func NewCategoryService(repo repository.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) List(ctx context.Context) ([]models.Category, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("category repository is required")
	}
	categories, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return categories, nil
}

func (s *CategoryService) Create(ctx context.Context, req *CreateCategoryRequest) (*models.Category, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("category repository is required")
	}
	if req == nil {
		return nil, validationError("create category request is required")
	}
	name := strings.TrimSpace(req.Name)
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if name == "" || len([]rune(name)) > 80 {
		return nil, validationError("category name must contain 1 to 80 characters")
	}
	if len(slug) > 80 || !categorySlugPattern.MatchString(slug) {
		return nil, validationError("category slug must use lowercase Latin letters, numbers, hyphens, or underscores")
	}
	now := time.Now().UTC()
	category := &models.Category{ID: uuid.NewString(), Slug: slug, Name: name, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.Create(ctx, category); err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}
	return category, nil
}
