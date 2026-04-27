package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/server/handler"
	"github.com/stretchr/testify/suite"
)

// mockCategoryServicer implements handler.CategoryServicer for tests.
//
// Each method records its inputs and returns the configured result.
// The "called*" booleans let assertions verify which service path the
// handler took (e.g. rename vs colour-only) without inferring it from
// the response body.
type mockCategoryServicer struct {
	// CreateCategory
	createReturn   *repository.Category
	createErr      error
	createCalled   bool
	createRawName  string
	createColour   *string
	createColourIs bool // true if create was called and a non-nil colour was passed

	// RenameCategory
	renameReturn  *repository.Category
	renameErr     error
	renameCalled  bool
	renameOldKey  string
	renameNewName string

	// SetCategoryColour
	setColourErr      error
	setColourCalled   bool
	setColourKey      string
	setColourValue    *string
	setColourValueSet bool // true if a non-nil colour was passed

	// DeleteCategory
	deleteErr    error
	deleteCalled bool
	deleteKey    string

	// GetCategory
	getReturn *repository.Category
	getErr    error
	getCalled bool
	getInput  string

	// ListCategories
	listReturn []*repository.CategoryWithCount
	listErr    error
	listCalled bool
	listCounts bool
}

func (m *mockCategoryServicer) CreateCategory(_ context.Context, rawName string, colour *string) (*repository.Category, error) {
	m.createCalled = true
	m.createRawName = rawName
	m.createColour = colour
	m.createColourIs = colour != nil
	return m.createReturn, m.createErr
}

func (m *mockCategoryServicer) RenameCategory(_ context.Context, oldKey, newRawName string) (*repository.Category, error) {
	m.renameCalled = true
	m.renameOldKey = oldKey
	m.renameNewName = newRawName
	return m.renameReturn, m.renameErr
}

func (m *mockCategoryServicer) SetCategoryColour(_ context.Context, key string, colour *string) error {
	m.setColourCalled = true
	m.setColourKey = key
	m.setColourValue = colour
	m.setColourValueSet = colour != nil
	return m.setColourErr
}

func (m *mockCategoryServicer) DeleteCategory(_ context.Context, key string) error {
	m.deleteCalled = true
	m.deleteKey = key
	return m.deleteErr
}

func (m *mockCategoryServicer) GetCategory(_ context.Context, rawNameOrKey string) (*repository.Category, error) {
	m.getCalled = true
	m.getInput = rawNameOrKey
	return m.getReturn, m.getErr
}

func (m *mockCategoryServicer) ListCategories(_ context.Context, withCounts bool) ([]*repository.CategoryWithCount, error) {
	m.listCalled = true
	m.listCounts = withCounts
	return m.listReturn, m.listErr
}

// CategoriesHandlerSuite exercises the categories handler end-to-end via a
// real http.ServeMux so route patterns and {name} extraction match runtime.
type CategoriesHandlerSuite struct {
	suite.Suite
}

func TestCategoriesHandler(t *testing.T) {
	suite.Run(t, new(CategoriesHandlerSuite))
}

// newMux registers the five category routes against the provided mock.
func (s *CategoriesHandlerSuite) newMux(svc handler.CategoryServicer) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/todo/categories", handler.ListCategoriesHandler(svc))
	mux.Handle("POST /api/v1/todo/categories", handler.CreateCategoryHandler(svc))
	mux.Handle("GET /api/v1/todo/categories/{name}", handler.GetCategoryHandler(svc))
	mux.Handle("PUT /api/v1/todo/categories/{name}", handler.UpdateCategoryHandler(svc))
	mux.Handle("DELETE /api/v1/todo/categories/{name}", handler.DeleteCategoryHandler(svc))
	return mux
}

func ptrStr(v string) *string { return &v }

// --- List ---

func (s *CategoriesHandlerSuite) TestList() {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	mock := &mockCategoryServicer{
		listReturn: []*repository.CategoryWithCount{
			{
				Category: repository.Category{
					NameKey:   "foo_bar",
					Colour:    ptrStr("#3aa3aa"),
					CreatedAt: now,
				},
				TaskCount: 4,
			},
			{
				Category: repository.Category{
					NameKey:   "work",
					Colour:    nil,
					CreatedAt: now,
				},
				TaskCount: 0,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todo/categories", nil)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.True(mock.listCalled)

	var body []map[string]any
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&body))
	s.Require().Len(body, 2)

	s.Equal("foo_bar", body[0]["key"])
	s.Equal("Foo Bar", body[0]["name"])
	s.Equal("#3aa3aa", body[0]["colour"])
	s.Equal(float64(4), body[0]["task_count"])
	s.Contains(body[0], "created_at")

	s.Equal("work", body[1]["key"])
	s.Equal("Work", body[1]["name"])
	s.Nil(body[1]["colour"])
	s.Equal(float64(0), body[1]["task_count"])
}

func (s *CategoriesHandlerSuite) TestListEmpty() {
	mock := &mockCategoryServicer{listReturn: []*repository.CategoryWithCount{}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todo/categories", nil)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var body []map[string]any
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&body))
	s.Empty(body)
}

// --- Get ---

func (s *CategoriesHandlerSuite) TestGetExistingByKey() {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	mock := &mockCategoryServicer{
		getReturn: &repository.Category{
			NameKey:   "foo_bar",
			Colour:    ptrStr("#3aa3aa"),
			CreatedAt: now,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todo/categories/foo_bar", nil)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.True(mock.getCalled)
	s.Equal("foo_bar", mock.getInput)

	var body map[string]any
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&body))
	s.Equal("foo_bar", body["key"])
	s.Equal("Foo Bar", body["name"])
	s.Equal("#3aa3aa", body["colour"])
	s.Contains(body, "created_at")
	s.Contains(body, "task_count")
}

func (s *CategoriesHandlerSuite) TestGetExistingByDisplay() {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	mock := &mockCategoryServicer{
		getReturn: &repository.Category{
			NameKey:   "foo_bar",
			Colour:    nil,
			CreatedAt: now,
		},
	}

	// URL-encoded "Foo Bar"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/todo/categories/Foo%20Bar", nil)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.True(mock.getCalled)
	// Service receives the URL-decoded raw form. The service is
	// responsible for normalization fallback.
	s.Equal("Foo Bar", mock.getInput)

	var body map[string]any
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&body))
	s.Equal("foo_bar", body["key"])
	s.Equal("Foo Bar", body["name"])
	s.Nil(body["colour"])
}

func (s *CategoriesHandlerSuite) TestGetNotFound() {
	mock := &mockCategoryServicer{getErr: repository.ErrNotFound}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todo/categories/missing", nil)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusNotFound, rec.Code)
}

// --- Create ---

func (s *CategoriesHandlerSuite) TestCreateHappyPath() {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	mock := &mockCategoryServicer{
		createReturn: &repository.Category{
			NameKey:   "foo_bar",
			Colour:    ptrStr("#abcdef"),
			CreatedAt: now,
		},
	}

	body := strings.NewReader(`{"name":"foo BAR","colour":"#abcdef"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todo/categories", body)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.True(mock.createCalled)
	s.Equal("foo BAR", mock.createRawName)
	s.Require().True(mock.createColourIs, "expected non-nil colour passed to service")
	s.Equal("#abcdef", *mock.createColour)

	// Match either 200 or 201 — project convention varies between handlers;
	// tasks uses 201, services uses 201, so we assert 201 here.
	s.Equal(http.StatusCreated, rec.Code)

	var resp map[string]any
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&resp))
	s.Equal("foo_bar", resp["key"])
	s.Equal("Foo Bar", resp["name"])
	s.Equal("#abcdef", resp["colour"])
	s.Contains(resp, "created_at")
	s.Contains(resp, "task_count")
}

func (s *CategoriesHandlerSuite) TestCreateNoColour() {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	mock := &mockCategoryServicer{
		createReturn: &repository.Category{
			NameKey:   "work",
			Colour:    nil,
			CreatedAt: now,
		},
	}

	body := strings.NewReader(`{"name":"work"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todo/categories", body)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.True(mock.createCalled)
	s.Equal("work", mock.createRawName)
	s.False(mock.createColourIs, "expected nil colour passed to service when omitted")

	s.Equal(http.StatusCreated, rec.Code)

	var resp map[string]any
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&resp))
	s.Equal("work", resp["key"])
	// Per Decision 8, "colour" is always present and null when unset.
	_, present := resp["colour"]
	s.True(present, "colour key must be present even when null")
	s.Nil(resp["colour"])
}

func (s *CategoriesHandlerSuite) TestCreateBadName() {
	// Service surfaces a validation error from NormalizeCategoryKey.
	mock := &mockCategoryServicer{
		createErr: fmt.Errorf("underscores not allowed in category name: %w", repository.ErrValidation),
	}

	body := strings.NewReader(`{"name":"foo_bar"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todo/categories", body)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *CategoriesHandlerSuite) TestCreateBadColour() {
	body := strings.NewReader(`{"name":"work","colour":"not-a-hex"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todo/categories", body)
	rec := httptest.NewRecorder()

	mock := &mockCategoryServicer{
		// If the handler pre-validates colour, the service is never reached.
		// If the handler defers to the service, the service returns an error
		// that is NOT ErrDuplicate / ErrNotFound, which should map to 400.
		createErr: fmt.Errorf("colour must match #RRGGBB: %w", repository.ErrValidation),
	}
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *CategoriesHandlerSuite) TestCreateDuplicate() {
	mock := &mockCategoryServicer{createErr: repository.ErrDuplicate}

	body := strings.NewReader(`{"name":"work"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todo/categories", body)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusConflict, rec.Code)
}

// --- PUT ---

func (s *CategoriesHandlerSuite) TestPutRenameOnly() {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	mock := &mockCategoryServicer{
		// Path-key lookup so the handler knows the canonical existing key.
		getReturn: &repository.Category{
			NameKey:   "foo_bar",
			Colour:    ptrStr("#aaaaaa"),
			CreatedAt: now,
		},
		renameReturn: &repository.Category{
			NameKey:   "work",
			Colour:    ptrStr("#aaaaaa"),
			CreatedAt: now,
		},
	}

	body := strings.NewReader(`{"name":"work"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todo/categories/foo_bar", body)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.True(mock.renameCalled, "rename must be called")
	s.Equal("foo_bar", mock.renameOldKey)
	s.Equal("work", mock.renameNewName)
	s.False(mock.setColourCalled, "set colour must NOT be called when colour key absent")

	var resp map[string]any
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&resp))
	s.Equal("work", resp["key"])
	s.Equal("Work", resp["name"])
}

func (s *CategoriesHandlerSuite) TestPutColourOnly() {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	mock := &mockCategoryServicer{
		getReturn: &repository.Category{
			NameKey:   "foo_bar",
			Colour:    ptrStr("#abc123"),
			CreatedAt: now,
		},
	}

	body := strings.NewReader(`{"colour":"#abc123"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todo/categories/foo_bar", body)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.False(mock.renameCalled, "rename must NOT be called when name absent")
	s.True(mock.setColourCalled)
	s.Equal("foo_bar", mock.setColourKey)
	s.Require().True(mock.setColourValueSet)
	s.Equal("#abc123", *mock.setColourValue)
}

func (s *CategoriesHandlerSuite) TestPutColourCleared() {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	mock := &mockCategoryServicer{
		getReturn: &repository.Category{
			NameKey:   "foo_bar",
			Colour:    nil,
			CreatedAt: now,
		},
	}

	// JSON null for colour: handler MUST distinguish from absent.
	body := strings.NewReader(`{"colour":null}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todo/categories/foo_bar", body)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.False(mock.renameCalled)
	s.True(mock.setColourCalled, "set colour must be called when colour:null")
	s.Equal("foo_bar", mock.setColourKey)
	s.False(mock.setColourValueSet, "colour passed to service must be nil for clear")
	s.Nil(mock.setColourValue)
}

func (s *CategoriesHandlerSuite) TestPutBoth() {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	mock := &mockCategoryServicer{
		getReturn: &repository.Category{
			NameKey:   "foo_bar",
			Colour:    ptrStr("#aaaaaa"),
			CreatedAt: now,
		},
		renameReturn: &repository.Category{
			NameKey:   "work",
			Colour:    ptrStr("#aaaaaa"),
			CreatedAt: now,
		},
	}

	body := strings.NewReader(`{"name":"work","colour":"#bbbbbb"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todo/categories/foo_bar", body)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)
	s.True(mock.renameCalled, "rename must be called")
	s.Equal("foo_bar", mock.renameOldKey)
	s.Equal("work", mock.renameNewName)
	s.True(mock.setColourCalled, "set colour must be called after rename")
	// After rename, set colour uses the NEW key.
	s.Equal("work", mock.setColourKey)
	s.Require().True(mock.setColourValueSet)
	s.Equal("#bbbbbb", *mock.setColourValue)
}

func (s *CategoriesHandlerSuite) TestPutNameSameAsKey() {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	existing := &repository.Category{
		NameKey:   "foo_bar",
		Colour:    ptrStr("#aaaaaa"),
		CreatedAt: now,
	}
	// Service.RenameCategory is a no-op when newKey == oldKey and returns
	// the existing row. Handler may either: (a) skip the call when it
	// detects the new key normalizes to the same as the path key, or
	// (b) defer to the service. Both are acceptable — assert via the
	// response shape.
	mock := &mockCategoryServicer{
		getReturn:    existing,
		renameReturn: existing,
	}

	body := strings.NewReader(`{"name":"Foo Bar"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todo/categories/foo_bar", body)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var resp map[string]any
	s.Require().NoError(json.NewDecoder(rec.Body).Decode(&resp))
	s.Equal("foo_bar", resp["key"])
	s.Equal("Foo Bar", resp["name"])
}

// --- Delete ---

func (s *CategoriesHandlerSuite) TestDelete() {
	mock := &mockCategoryServicer{
		getReturn: &repository.Category{
			NameKey:   "foo_bar",
			CreatedAt: time.Now(),
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/todo/categories/foo_bar", nil)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusNoContent, rec.Code)
	s.True(mock.deleteCalled)
	s.Equal("foo_bar", mock.deleteKey)
}

func (s *CategoriesHandlerSuite) TestDeleteNotFound() {
	mock := &mockCategoryServicer{
		// Lookup-before-delete fails with not-found.
		getErr:    repository.ErrNotFound,
		deleteErr: repository.ErrNotFound,
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/todo/categories/missing", nil)
	rec := httptest.NewRecorder()
	s.newMux(mock).ServeHTTP(rec, req)

	s.Equal(http.StatusNotFound, rec.Code)
}
