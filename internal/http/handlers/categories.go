package handlers

import (
	"net/http"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	"github.com/sunriseex/capitalflow/internal/services"
)

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.app.Categories.List(r.Context())
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
	category, err := h.app.Categories.Create(r.Context(), &services.CreateCategoryRequest{Name: req.Name, Slug: req.Slug})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.CategoryFromModel(category))
}
