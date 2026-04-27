package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// CategoryServicer is the subset of todo.Service needed by category handlers.
//
// Mirrors the methods on *internal/service/todo/Service that deal with
// categories. The service is responsible for normalizing raw user input
// via repository.NormalizeCategoryKey before reaching the repo layer.
type CategoryServicer interface {
	CreateCategory(ctx context.Context, rawName string, colour *string) (*repository.Category, error)
	RenameCategory(ctx context.Context, oldKey, newRawName string) (*repository.Category, error)
	SetCategoryColour(ctx context.Context, key string, colour *string) error
	DeleteCategory(ctx context.Context, key string) error
	GetCategory(ctx context.Context, rawNameOrKey string) (*repository.Category, error)
	ListCategories(ctx context.Context, withCounts bool) ([]*repository.CategoryWithCount, error)
}

// categoryItem is the JSON wire representation of a category in responses.
//
// Per Feature 109 Decision 8, "colour" is always present (never omitted)
// and is null when unset. "task_count" is included on both list and get
// reads (zero is acceptable).
type categoryItem struct {
	Key       string  `json:"key"`
	Name      string  `json:"name"`
	Colour    *string `json:"colour"`
	CreatedAt string  `json:"created_at"`
	TaskCount int     `json:"task_count"`
}

// createCategoryRequest is the JSON body for POST /api/v1/todo/categories.
type createCategoryRequest struct {
	Name   string  `json:"name"`
	Colour *string `json:"colour"`
}

// toCategoryItem builds the wire-format response shape from a Category and
// an optional task count.
func toCategoryItem(cat *repository.Category, taskCount int) categoryItem {
	return categoryItem{
		Key:       cat.NameKey,
		Name:      repository.PresentCategoryName(cat.NameKey),
		Colour:    cat.Colour,
		CreatedAt: cat.CreatedAt.Format(time.RFC3339),
		TaskCount: taskCount,
	}
}

// writeCategoryError maps service-layer errors to HTTP statuses.
//
//	ErrValidation -> 400
//	ErrNotFound   -> 404
//	ErrDuplicate  -> 409
//	default       -> 500
func writeCategoryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrValidation):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "not found")
	case errors.Is(err, repository.ErrDuplicate):
		writeJSONError(w, http.StatusConflict, "already exists")
	case looksLikeValidation(err):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, "internal error")
	}
}

// looksLikeValidation is a defensive fallback for service errors that bubble
// up from repository.NormalizeCategoryKey or colour validation but were not
// wrapped with repository.ErrValidation (e.g. tests using bare errors).
func looksLikeValidation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"underscore",
		"colour",
		"color",
		"empty",
		"too long",
		"exceeds",
		"invalid character",
		"non-ascii",
		"normalize",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

// taskCountFor looks up a category's task count by querying the full list.
// Returns 0 if the category is not present in the list or if the list call
// fails (caller decides whether that is acceptable).
func taskCountFor(ctx context.Context, svc CategoryServicer, key string) int {
	list, err := svc.ListCategories(ctx, true)
	if err != nil {
		return 0
	}
	for _, c := range list {
		if c.NameKey == key {
			return c.TaskCount
		}
	}
	return 0
}

// ListCategoriesHandler returns an http.HandlerFunc for
// GET /api/v1/todo/categories.
//
// @Summary      List categories
// @Tags         todo-categories
// @Produce      json
// @Success      200  {array}  handler.categoryItem
// @Failure      500  {object} map[string]string
// @Router       /api/v1/todo/categories [get]
func ListCategoriesHandler(svc CategoryServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cats, err := svc.ListCategories(r.Context(), true)
		if err != nil {
			writeCategoryError(w, err)
			return
		}
		items := make([]categoryItem, len(cats))
		for i, c := range cats {
			items[i] = toCategoryItem(&c.Category, c.TaskCount)
		}
		writeJSON(w, http.StatusOK, items)
	}
}

// CreateCategoryHandler returns an http.HandlerFunc for
// POST /api/v1/todo/categories.
//
// @Summary      Create category
// @Tags         todo-categories
// @Accept       json
// @Produce      json
// @Param        request  body      handler.createCategoryRequest  true  "Category fields"
// @Success      201      {object}  handler.categoryItem
// @Failure      400      {object}  map[string]string
// @Failure      409      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/todo/categories [post]
func CreateCategoryHandler(svc CategoryServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createCategoryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		cat, err := svc.CreateCategory(r.Context(), req.Name, req.Colour)
		if err != nil {
			writeCategoryError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toCategoryItem(cat, 0))
	}
}

// GetCategoryHandler returns an http.HandlerFunc for
// GET /api/v1/todo/categories/{name}. Path param accepts any form
// (raw display, mixed case, URL-encoded); the handler defers to the
// service for normalization.
//
// @Summary      Get category
// @Tags         todo-categories
// @Produce      json
// @Param        name  path      string  true  "Category key or display name"
// @Success      200   {object}  handler.categoryItem
// @Failure      404   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /api/v1/todo/categories/{name} [get]
func GetCategoryHandler(svc CategoryServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		cat, err := svc.GetCategory(r.Context(), name)
		if err != nil {
			writeCategoryError(w, err)
			return
		}
		count := taskCountFor(r.Context(), svc, cat.NameKey)
		writeJSON(w, http.StatusOK, toCategoryItem(cat, count))
	}
}

// UpdateCategoryHandler returns an http.HandlerFunc for
// PUT /api/v1/todo/categories/{name}. Supports renaming and/or
// updating the colour in a single request. JSON null on "colour"
// clears the colour; absent "colour" leaves it untouched.
//
// @Summary      Update category
// @Tags         todo-categories
// @Accept       json
// @Produce      json
// @Param        name     path      string  true  "Category key or display name"
// @Param        request  body      object  true  "Partial update body"
// @Success      200      {object}  handler.categoryItem
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      409      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/todo/categories/{name} [put]
func UpdateCategoryHandler(svc CategoryServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")

		// Decode into a raw map so we can distinguish absent vs explicit null,
		// particularly for "colour" where null means "clear".
		var raw map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Resolve the canonical existing category from the path param.
		cat, err := svc.GetCategory(r.Context(), name)
		if err != nil {
			writeCategoryError(w, err)
			return
		}

		// Optional rename.
		if rawName, ok := raw["name"]; ok {
			var newName string
			if err := json.Unmarshal(rawName, &newName); err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid name")
				return
			}
			renamed, err := svc.RenameCategory(r.Context(), cat.NameKey, newName)
			if err != nil {
				writeCategoryError(w, err)
				return
			}
			cat = renamed
		}

		// Optional colour update (null => clear).
		if rawColour, ok := raw["colour"]; ok {
			var colour *string
			if string(rawColour) != "null" {
				var s string
				if err := json.Unmarshal(rawColour, &s); err != nil {
					writeJSONError(w, http.StatusBadRequest, "invalid colour")
					return
				}
				colour = &s
			}
			if err := svc.SetCategoryColour(r.Context(), cat.NameKey, colour); err != nil {
				writeCategoryError(w, err)
				return
			}
			// Re-fetch to surface the updated colour.
			refreshed, err := svc.GetCategory(r.Context(), cat.NameKey)
			if err != nil {
				writeCategoryError(w, err)
				return
			}
			cat = refreshed
		}

		count := taskCountFor(r.Context(), svc, cat.NameKey)
		writeJSON(w, http.StatusOK, toCategoryItem(cat, count))
	}
}

// DeleteCategoryHandler returns an http.HandlerFunc for
// DELETE /api/v1/todo/categories/{name}.
//
// @Summary      Delete category
// @Tags         todo-categories
// @Param        name  path  string  true  "Category key or display name"
// @Success      204  "No Content"
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/todo/categories/{name} [delete]
func DeleteCategoryHandler(svc CategoryServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		cat, err := svc.GetCategory(r.Context(), name)
		if err != nil {
			writeCategoryError(w, err)
			return
		}
		if err := svc.DeleteCategory(r.Context(), cat.NameKey); err != nil {
			writeCategoryError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
