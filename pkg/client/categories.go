package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// categoriesPath is the base URL path for the categories REST resource.
// Per Feature 109 Decision 5, categories live under the /api/v1/todo/
// bounded-context prefix alongside tasks.
const categoriesPath = "/api/v1/todo/categories"

// Category mirrors the server's categoryItem DTO returned by
// /api/v1/todo/categories routes. Per Feature 109 Decision 4:
//
//   - Key is the canonical lowercase form (with underscores for spaces) and
//     serves as the primary key on the server. All FK references and lookup
//     paths use this form.
//   - Name is the derived presentation form computed by PresentCategoryName.
//     It is informational only — clients should treat Key as authoritative.
//   - Colour is nullable (`#RRGGBB` hex, validated server-side).
//   - TaskCount is the number of tasks currently linked to this category.
//     Always populated on read; recomputed per request.
type Category struct {
	Key       string  `json:"key"`
	Name      string  `json:"name"`
	Colour    *string `json:"colour"`
	CreatedAt string  `json:"created_at"`
	TaskCount int     `json:"task_count"`
}

// CreateCategoryRequest is the POST body for creating a category via
// POST /api/v1/todo/categories. Name is raw input — any case/spacing form —
// the server normalizes via NormalizeCategoryKey before insertion. Colour
// is optional; nil omits the field from the body so the server stores NULL.
type CreateCategoryRequest struct {
	Name   string  `json:"name"`
	Colour *string `json:"colour,omitempty"`
}

// UpdateCategoryRequest is the PUT body for partial updates via
// PUT /api/v1/todo/categories/{name}. Tri-state colour mirrors the pattern
// used on UpdateTaskRequest.Category:
//
//   - Name == nil: omit name from the outgoing JSON (no rename).
//   - Name != nil: emit `"name":"<value>"`; the server renames the
//     category and cascades the new key through `tasks.category_key`.
//   - Colour == nil and ClearColour == false: omit colour (no change).
//   - Colour != nil: emit `"colour":"<hex>"` (server validates).
//   - Colour == nil and ClearColour == true: emit `"colour":null` (clear).
//
// See MarshalJSON for the encoding implementation.
type UpdateCategoryRequest struct {
	Name        *string `json:"name,omitempty"`
	Colour      *string `json:"-"`
	ClearColour bool    `json:"-"`
}

// MarshalJSON emits the tri-state colour encoding documented on
// UpdateCategoryRequest. The pattern matches UpdateTaskRequest.MarshalJSON:
// marshal an alias with the custom field hidden, decode to a map, then
// splice "colour" in as a string, null, or omitted.
func (r UpdateCategoryRequest) MarshalJSON() ([]byte, error) {
	type alias UpdateCategoryRequest
	raw, err := json.Marshal(alias(r))
	if err != nil {
		return nil, fmt.Errorf("marshal category update: %w", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("re-decode category update: %w", err)
	}

	switch {
	case r.Colour != nil:
		b, err := json.Marshal(*r.Colour)
		if err != nil {
			return nil, fmt.Errorf("marshal colour value: %w", err)
		}
		fields["colour"] = b
	case r.ClearColour:
		fields["colour"] = json.RawMessage("null")
	}

	out, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("re-encode category update: %w", err)
	}
	return out, nil
}

// CategoryClient wraps /api/v1/todo/categories routes: listing, fetching,
// creating, updating (partial — rename and/or colour), and deleting
// categories. Path-segment lookups (`{name}`) accept any form — the server
// normalizes before resolving the canonical key.
type CategoryClient interface {
	ListCategories(ctx context.Context) ([]Category, error)
	GetCategory(ctx context.Context, name string) (*Category, error)
	CreateCategory(ctx context.Context, req CreateCategoryRequest) (*Category, error)
	UpdateCategory(ctx context.Context, name string, req UpdateCategoryRequest) (*Category, error)
	DeleteCategory(ctx context.Context, name string) error
}

// categoryAdapter is the concrete CategoryClient backed by an *APIClient.
// All methods are stubs in the RED phase — they return
// errCategoriesNotImplemented so suite assertions fail meaningfully rather
// than the build breaking.
type categoryAdapter struct {
	client *APIClient
}

// NewCategoryClient returns a CategoryClient backed by the given APIClient.
func NewCategoryClient(c *APIClient) CategoryClient {
	return &categoryAdapter{client: c}
}

// ListCategories issues GET /api/v1/todo/categories and decodes the JSON
// array of Category items returned by the server.
func (a *categoryAdapter) ListCategories(ctx context.Context) ([]Category, error) {
	var out []Category
	if err := a.client.doJSON(ctx, http.MethodGet, categoriesPath, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCategory issues GET /api/v1/todo/categories/{name}. The name segment is
// URL-escaped via url.PathEscape so display-form input ("Foo Bar") reaches
// the server intact ("Foo%20Bar") for normalization. A 404 surfaces as
// *APIError with Code == ErrCodeNotFound.
func (a *categoryAdapter) GetCategory(ctx context.Context, name string) (*Category, error) {
	var cat Category
	path := categoriesPath + "/" + url.PathEscape(name)
	if err := a.client.doJSON(ctx, http.MethodGet, path, nil, &cat); err != nil {
		return nil, err
	}
	return &cat, nil
}

// CreateCategory issues POST /api/v1/todo/categories with req as the JSON
// body and decodes the created Category from the 201 response. A 409
// (duplicate normalized key) surfaces as *APIError with Code == ErrCodeConflict.
func (a *categoryAdapter) CreateCategory(ctx context.Context, req CreateCategoryRequest) (*Category, error) {
	var cat Category
	if err := a.client.doJSON(ctx, http.MethodPost, categoriesPath, req, &cat); err != nil {
		return nil, err
	}
	return &cat, nil
}

// UpdateCategory issues PUT /api/v1/todo/categories/{name} with the
// tri-state-aware UpdateCategoryRequest body (see its MarshalJSON). The
// name segment is URL-escaped. A 404 surfaces as *APIError with Code ==
// ErrCodeNotFound.
func (a *categoryAdapter) UpdateCategory(ctx context.Context, name string, req UpdateCategoryRequest) (*Category, error) {
	var cat Category
	path := categoriesPath + "/" + url.PathEscape(name)
	if err := a.client.doJSON(ctx, http.MethodPut, path, req, &cat); err != nil {
		return nil, err
	}
	return &cat, nil
}

// DeleteCategory issues DELETE /api/v1/todo/categories/{name}. The server
// responds with 204 No Content on success and cascades SET NULL on dependent
// tasks. A 404 surfaces as *APIError with Code == ErrCodeNotFound.
func (a *categoryAdapter) DeleteCategory(ctx context.Context, name string) error {
	path := categoriesPath + "/" + url.PathEscape(name)
	return a.client.doJSON(ctx, http.MethodDelete, path, nil, nil)
}
