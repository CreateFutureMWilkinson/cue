package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// categoriesPath is the base URL path for the categories REST resource.
// Per Feature 109 Decision 5, categories live under the /api/v1/todo/
// bounded-context prefix alongside tasks.
const categoriesPath = "/api/v1/todo/categories"

// errCategoriesNotImplemented is the stub sentinel returned by every method
// on the unimplemented CategoryClient. Replaced in the GREEN phase with real
// HTTP transport logic.
var errCategoriesNotImplemented = errors.New("not implemented")

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

// ListCategories — STUB (Loop 8 RED). Replaced in GREEN with
// GET /api/v1/todo/categories returning []Category.
func (a *categoryAdapter) ListCategories(_ context.Context) ([]Category, error) {
	return nil, errCategoriesNotImplemented
}

// GetCategory — STUB (Loop 8 RED). Replaced in GREEN with
// GET /api/v1/todo/categories/{name}; the path segment is URL-escaped and
// the server normalizes any form (raw display, mixed case, canonical key).
func (a *categoryAdapter) GetCategory(_ context.Context, _ string) (*Category, error) {
	return nil, errCategoriesNotImplemented
}

// CreateCategory — STUB (Loop 8 RED). Replaced in GREEN with
// POST /api/v1/todo/categories carrying the JSON request body.
func (a *categoryAdapter) CreateCategory(_ context.Context, _ CreateCategoryRequest) (*Category, error) {
	return nil, errCategoriesNotImplemented
}

// UpdateCategory — STUB (Loop 8 RED). Replaced in GREEN with
// PUT /api/v1/todo/categories/{name}; tri-state colour encoding handled by
// UpdateCategoryRequest.MarshalJSON.
func (a *categoryAdapter) UpdateCategory(_ context.Context, _ string, _ UpdateCategoryRequest) (*Category, error) {
	return nil, errCategoriesNotImplemented
}

// DeleteCategory — STUB (Loop 8 RED). Replaced in GREEN with
// DELETE /api/v1/todo/categories/{name}; server responds 204 on success
// and cascades SET NULL on dependent tasks.
func (a *categoryAdapter) DeleteCategory(_ context.Context, _ string) error {
	return errCategoriesNotImplemented
}
