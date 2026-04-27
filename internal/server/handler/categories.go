package handler

import (
	"context"
	"net/http"

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

// ListCategoriesHandler returns an http.HandlerFunc for
// GET /api/v1/todo/categories.
//
// Stub: returns 500 so RED tests fail with assertion errors rather than 404.
func ListCategoriesHandler(svc CategoryServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSONError(w, http.StatusInternalServerError, "not implemented")
	}
}

// CreateCategoryHandler returns an http.HandlerFunc for
// POST /api/v1/todo/categories.
//
// Stub: returns 500.
func CreateCategoryHandler(svc CategoryServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSONError(w, http.StatusInternalServerError, "not implemented")
	}
}

// GetCategoryHandler returns an http.HandlerFunc for
// GET /api/v1/todo/categories/{name}. Path param accepts any form
// (raw display, mixed case, URL-encoded); the handler normalizes via
// the service before lookup.
//
// Stub: returns 500.
func GetCategoryHandler(svc CategoryServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSONError(w, http.StatusInternalServerError, "not implemented")
	}
}

// UpdateCategoryHandler returns an http.HandlerFunc for
// PUT /api/v1/todo/categories/{name}. Supports renaming and/or
// updating the colour in a single request.
//
// Stub: returns 500.
func UpdateCategoryHandler(svc CategoryServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSONError(w, http.StatusInternalServerError, "not implemented")
	}
}

// DeleteCategoryHandler returns an http.HandlerFunc for
// DELETE /api/v1/todo/categories/{name}.
//
// Stub: returns 500.
func DeleteCategoryHandler(svc CategoryServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSONError(w, http.StatusInternalServerError, "not implemented")
	}
}
