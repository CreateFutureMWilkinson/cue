package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// CategorySuite covers the CategoryClient adapter over
// /api/v1/todo/categories. Stub-driven RED phase: every method is expected
// to fail with not-implemented until Loop 8 GREEN replaces the bodies with
// real HTTP transport.
type CategorySuite struct {
	suite.Suite
}

func TestCategory(t *testing.T) {
	suite.Run(t, new(CategorySuite))
}

// TestListCategoriesDecodesResponse verifies that a list-categories call
// hits /api/v1/todo/categories and decodes a JSON array into []Category
// with all fields populated (key, name, colour, created_at, task_count).
func (s *CategorySuite) TestListCategoriesDecodesResponse() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/todo/categories", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"key":        "foo_bar",
				"name":       "Foo Bar",
				"colour":     "#3aa3aa",
				"created_at": "2026-04-01T10:00:00Z",
				"task_count": 4,
			},
			{
				"key":        "home",
				"name":       "Home",
				"colour":     nil,
				"created_at": "2026-04-02T11:00:00Z",
				"task_count": 0,
			},
		})
	}))
	defer ts.Close()

	cc := client.NewCategoryClient(client.New(ts.URL))
	cats, err := cc.ListCategories(context.Background())
	s.Require().NoError(err)
	s.Require().Len(cats, 2)

	first := cats[0]
	s.Equal("foo_bar", first.Key)
	s.Equal("Foo Bar", first.Name)
	s.Require().NotNil(first.Colour)
	s.Equal("#3aa3aa", *first.Colour)
	s.Equal("2026-04-01T10:00:00Z", first.CreatedAt)
	s.Equal(4, first.TaskCount)

	second := cats[1]
	s.Equal("home", second.Key)
	s.Nil(second.Colour, "null colour on the wire must decode to nil pointer")
	s.Equal(0, second.TaskCount)
}

// TestGetCategoryByDisplayForm verifies that GetCategory URL-escapes the
// path segment so `Foo Bar` reaches the server as `Foo%20Bar` (the server
// then normalizes to the canonical key).
func (s *CategorySuite) TestGetCategoryByDisplayForm() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		// r.URL.Path is the decoded form; r.URL.RawPath preserves escaping.
		s.Equal("/api/v1/todo/categories/Foo Bar", r.URL.Path)
		s.Equal("/api/v1/todo/categories/Foo%20Bar", r.URL.RawPath)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key":        "foo_bar",
			"name":       "Foo Bar",
			"colour":     "#abcdef",
			"created_at": "2026-04-01T10:00:00Z",
			"task_count": 2,
		})
	}))
	defer ts.Close()

	cc := client.NewCategoryClient(client.New(ts.URL))
	cat, err := cc.GetCategory(context.Background(), "Foo Bar")
	s.Require().NoError(err)
	s.Require().NotNil(cat)
	s.Equal("foo_bar", cat.Key)
	s.Equal("Foo Bar", cat.Name)
	s.Require().NotNil(cat.Colour)
	s.Equal("#abcdef", *cat.Colour)
	s.Equal(2, cat.TaskCount)
}

// TestGetCategoryNotFound verifies that a 404 from the server surfaces as
// an *APIError with ErrCodeNotFound.
func (s *CategorySuite) TestGetCategoryNotFound() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "category not found",
		})
	}))
	defer ts.Close()

	cc := client.NewCategoryClient(client.New(ts.URL))
	cat, err := cc.GetCategory(context.Background(), "ghost")
	s.Require().Error(err)
	s.Nil(cat)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "expected *APIError, got %T", err)
	s.Equal(client.ErrCodeNotFound, apiErr.Code)
}

// TestCreateCategoryWithColour verifies POST body decodes correctly with
// both name (raw input — server normalizes) and colour set.
func (s *CategorySuite) TestCreateCategoryWithColour() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/todo/categories", r.URL.Path)

		var body struct {
			Name   string  `json:"name"`
			Colour *string `json:"colour"`
		}
		s.Require().NoError(json.NewDecoder(r.Body).Decode(&body))
		s.Equal("foo BAR", body.Name)
		s.Require().NotNil(body.Colour)
		s.Equal("#3aa3aa", *body.Colour)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key":        "foo_bar",
			"name":       "Foo Bar",
			"colour":     "#3aa3aa",
			"created_at": "2026-04-01T10:00:00Z",
			"task_count": 0,
		})
	}))
	defer ts.Close()

	cc := client.NewCategoryClient(client.New(ts.URL))
	colour := "#3aa3aa"
	cat, err := cc.CreateCategory(context.Background(), client.CreateCategoryRequest{
		Name:   "foo BAR",
		Colour: &colour,
	})
	s.Require().NoError(err)
	s.Require().NotNil(cat)
	s.Equal("foo_bar", cat.Key)
	s.Equal("Foo Bar", cat.Name)
}

// TestCreateCategoryWithoutColourOmitsKey verifies that CreateCategoryRequest
// with Colour == nil produces a body without a "colour" key (omitempty).
func (s *CategorySuite) TestCreateCategoryWithoutColourOmitsKey() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		s.Contains(body, "name")
		s.NotContains(body, "colour", "nil Colour must omit the JSON key entirely")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key":        "home",
			"name":       "Home",
			"colour":     nil,
			"created_at": "2026-04-01T10:00:00Z",
			"task_count": 0,
		})
	}))
	defer ts.Close()

	cc := client.NewCategoryClient(client.New(ts.URL))
	cat, err := cc.CreateCategory(context.Background(), client.CreateCategoryRequest{
		Name: "home",
	})
	s.Require().NoError(err)
	s.Require().NotNil(cat)
	s.Nil(cat.Colour)
}

// TestCreateCategoryDuplicateConflict verifies that a 409 from the server
// (duplicate normalized key) surfaces as an *APIError with ErrCodeConflict.
func (s *CategorySuite) TestCreateCategoryDuplicateConflict() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "category key already exists",
		})
	}))
	defer ts.Close()

	cc := client.NewCategoryClient(client.New(ts.URL))
	cat, err := cc.CreateCategory(context.Background(), client.CreateCategoryRequest{
		Name: "Foo Bar",
	})
	s.Require().Error(err)
	s.Nil(cat)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "expected *APIError, got %T", err)
	s.Equal(client.ErrCodeConflict, apiErr.Code)
}

// TestUpdateCategoryRenameOnly verifies that an UpdateCategoryRequest with
// only Name set emits `{"name":"new"}` (no colour key).
func (s *CategorySuite) TestUpdateCategoryRenameOnly() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPut, r.Method)
		s.Equal("/api/v1/todo/categories/foo_bar", r.URL.Path)

		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		s.Require().Contains(body, "name")
		var name string
		s.Require().NoError(json.Unmarshal(body["name"], &name))
		s.Equal("new name", name)
		s.NotContains(body, "colour")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key":        "new_name",
			"name":       "New Name",
			"colour":     nil,
			"created_at": "2026-04-01T10:00:00Z",
			"task_count": 0,
		})
	}))
	defer ts.Close()

	cc := client.NewCategoryClient(client.New(ts.URL))
	newName := "new name"
	cat, err := cc.UpdateCategory(context.Background(), "foo_bar", client.UpdateCategoryRequest{
		Name: &newName,
	})
	s.Require().NoError(err)
	s.Require().NotNil(cat)
	s.Equal("new_name", cat.Key)
}

// TestUpdateCategoryColourOnly verifies that setting only Colour emits
// `{"colour":"#abcdef"}` and omits the name key.
func (s *CategorySuite) TestUpdateCategoryColourOnly() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		s.NotContains(body, "name")
		s.Require().Contains(body, "colour")
		var colour string
		s.Require().NoError(json.Unmarshal(body["colour"], &colour))
		s.Equal("#abcdef", colour)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key":        "home",
			"name":       "Home",
			"colour":     "#abcdef",
			"created_at": "2026-04-01T10:00:00Z",
			"task_count": 1,
		})
	}))
	defer ts.Close()

	cc := client.NewCategoryClient(client.New(ts.URL))
	colour := "#abcdef"
	cat, err := cc.UpdateCategory(context.Background(), "home", client.UpdateCategoryRequest{
		Colour: &colour,
	})
	s.Require().NoError(err)
	s.Require().NotNil(cat)
}

// TestUpdateCategoryClearColour verifies that ClearColour == true (with
// Colour == nil) emits `"colour":null` so the server clears the field.
func (s *CategorySuite) TestUpdateCategoryClearColour() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		s.Require().Contains(body, "colour")
		s.Equal("null", string(body["colour"]), "ClearColour must serialise as JSON null")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key":        "home",
			"name":       "Home",
			"colour":     nil,
			"created_at": "2026-04-01T10:00:00Z",
			"task_count": 0,
		})
	}))
	defer ts.Close()

	cc := client.NewCategoryClient(client.New(ts.URL))
	cat, err := cc.UpdateCategory(context.Background(), "home", client.UpdateCategoryRequest{
		ClearColour: true,
	})
	s.Require().NoError(err)
	s.Require().NotNil(cat)
	s.Nil(cat.Colour)
}

// TestUpdateCategoryBoth verifies that setting both Name and Colour emits
// both keys in the outgoing JSON body.
func (s *CategorySuite) TestUpdateCategoryBoth() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		s.Contains(body, "name")
		s.Contains(body, "colour")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key":        "renamed",
			"name":       "Renamed",
			"colour":     "#123456",
			"created_at": "2026-04-01T10:00:00Z",
			"task_count": 0,
		})
	}))
	defer ts.Close()

	cc := client.NewCategoryClient(client.New(ts.URL))
	name := "renamed"
	colour := "#123456"
	cat, err := cc.UpdateCategory(context.Background(), "old", client.UpdateCategoryRequest{
		Name:   &name,
		Colour: &colour,
	})
	s.Require().NoError(err)
	s.Require().NotNil(cat)
}

// TestUpdateCategoryNotFound verifies a 404 surfaces as ErrCodeNotFound.
func (s *CategorySuite) TestUpdateCategoryNotFound() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "category not found",
		})
	}))
	defer ts.Close()

	cc := client.NewCategoryClient(client.New(ts.URL))
	name := "x"
	cat, err := cc.UpdateCategory(context.Background(), "ghost", client.UpdateCategoryRequest{
		Name: &name,
	})
	s.Require().Error(err)
	s.Nil(cat)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "expected *APIError, got %T", err)
	s.Equal(client.ErrCodeNotFound, apiErr.Code)
}

// TestDeleteCategory verifies DELETE /api/v1/todo/categories/{name} with a
// 204 No Content response returns no error.
func (s *CategorySuite) TestDeleteCategory() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodDelete, r.Method)
		s.Equal("/api/v1/todo/categories/foo_bar", r.URL.Path)

		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	cc := client.NewCategoryClient(client.New(ts.URL))
	err := cc.DeleteCategory(context.Background(), "foo_bar")
	s.Require().NoError(err)
}

// TestDeleteCategoryNotFound verifies that a 404 on delete surfaces as
// ErrCodeNotFound.
func (s *CategorySuite) TestDeleteCategoryNotFound() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "category not found",
		})
	}))
	defer ts.Close()

	cc := client.NewCategoryClient(client.New(ts.URL))
	err := cc.DeleteCategory(context.Background(), "ghost")
	s.Require().Error(err)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "expected *APIError, got %T", err)
	s.Equal(client.ErrCodeNotFound, apiErr.Code)
}
