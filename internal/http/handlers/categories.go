package handlers

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/models"
)

var categorySlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`)

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.app.Store.Categories().List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.CategoriesFromModels(categories))
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateCategoryRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	name := strings.TrimSpace(req.Name)
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if name == "" || len([]rune(name)) > 80 {
		writeError(w, http.StatusBadRequest, "validation_error", "Category name must contain 1 to 80 characters", nil)
		return
	}
	if len(slug) > 80 || !categorySlugPattern.MatchString(slug) {
		writeError(w, http.StatusBadRequest, "validation_error", "Category slug must use lowercase Latin letters, numbers, hyphens, or underscores", nil)
		return
	}

	now := time.Now().UTC()
	category := &models.Category{
		ID:        uuid.NewString(),
		Slug:      slug,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.app.Store.Categories().Create(r.Context(), category); err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.CategoryFromModel(category))
}
