package adapters_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/cmd/cue/adapters"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// TasksAdapterSuite covers TodoQuerier (list/insert/update/complete)
// and CategoryQuerier (list) round-trips through httptest.
type TasksAdapterSuite struct {
	suite.Suite
}

func TestTasksAdapter(t *testing.T) {
	suite.Run(t, new(TasksAdapterSuite))
}

// AC: tasks list/insert/complete/update flow through the SDK,
// preserving the single-category embed and timestamps.
func (s *TasksAdapterSuite) TestTasksRoundTrip() {
	created := uuid.New()
	var lastUpdate map[string]any
	var completeBody map[string]any

	taskDTO := func(id uuid.UUID, title string, completedAt *string) map[string]any {
		out := map[string]any{
			"id":                         id.String(),
			"title":                      title,
			"description":                "do the thing",
			"priority":                   2,
			"due_date":                   nil,
			"category":                   map[string]any{"key": "work", "name": "Work"},
			"estimate_minutes":           50,
			"llm_estimate_minutes":       nil,
			"effective_estimate_minutes": 50,
			"created_at":                 "2026-04-27T09:00:00Z",
		}
		if completedAt != nil {
			out["completed_at"] = *completedAt
		}
		return out
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/todo/tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			s.Equal("incomplete", r.URL.Query().Get("status"))
			body := map[string]any{
				"tasks": []any{taskDTO(created, "Write report", nil)},
				"total": 1,
			}
			s.Require().NoError(json.NewEncoder(w).Encode(body))
		case http.MethodPost:
			s.Require().NoError(json.NewEncoder(w).Encode(taskDTO(created, "fresh", nil)))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/todo/tasks/"+created.String(), func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var payload map[string]any
		if len(body) > 0 {
			s.Require().NoError(json.Unmarshal(body, &payload))
		}

		switch {
		case r.Method == http.MethodPut && payload["completed_at"] != nil && len(payload) == 1:
			completeBody = payload
		case r.Method == http.MethodPut:
			lastUpdate = payload
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		s.Require().NoError(json.NewEncoder(w).Encode(taskDTO(created, "updated", nil)))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	a := adapters.NewTasksAdapter(client.NewTaskClient(api))
	ctx := context.Background()

	// QueryFiltered.
	rows, total, err := a.QueryFiltered(ctx, repository.TaskFilter{Status: "incomplete"})
	s.Require().NoError(err)
	s.Equal(1, total)
	s.Require().Len(rows, 1)
	got := rows[0]
	s.Equal(created, got.ID)
	s.Require().NotNil(got.CategoryKey)
	s.Equal("work", *got.CategoryKey)
	s.False(got.CreatedAt.IsZero())

	// Insert.
	fresh := &repository.Task{Title: "fresh", Priority: 1}
	s.Require().NoError(a.Insert(ctx, fresh))
	s.Equal(created, fresh.ID, "Insert must populate ID from the server response")

	// Update with category clear.
	clearedKey := ""
	upd := &repository.Task{
		ID:          created,
		Title:       "updated",
		Description: "do the thing",
		Priority:    2,
		CategoryKey: &clearedKey,
	}
	s.Require().NoError(a.Update(ctx, upd))
	s.Require().NotNil(lastUpdate)
	s.Equal(nil, lastUpdate["category"], "ClearCategory must emit category:null on the wire")

	// Complete.
	s.Require().NoError(a.Complete(ctx, created, time.Date(2026, 4, 27, 11, 0, 0, 0, time.UTC)))
	s.Require().NotNil(completeBody)
	s.Equal("2026-04-27T11:00:00Z", completeBody["completed_at"])
}

// AC: categories list maps to repository.CategoryWithCount.
func (s *TasksAdapterSuite) TestCategoriesRoundTrip() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/todo/categories", func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		body := []any{
			map[string]any{
				"key":        "work",
				"name":       "Work",
				"colour":     "#336699",
				"created_at": "2026-04-27T08:00:00Z",
				"task_count": 4,
			},
			map[string]any{
				"key":        "home",
				"name":       "Home",
				"colour":     nil,
				"created_at": "2026-04-27T08:00:00Z",
				"task_count": 0,
			},
		}
		s.Require().NoError(json.NewEncoder(w).Encode(body))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	a := adapters.NewCategoriesAdapter(client.NewCategoryClient(api))

	cats, err := a.QueryAll(context.Background(), true)
	s.Require().NoError(err)
	s.Require().Len(cats, 2)
	s.Equal("work", cats[0].NameKey)
	s.Equal(4, cats[0].TaskCount)
	s.Require().NotNil(cats[0].Colour)
	s.Equal("#336699", *cats[0].Colour)
	s.Equal("home", cats[1].NameKey)
	s.Nil(cats[1].Colour, "nullable colour must round-trip as nil")
	s.Equal(0, cats[1].TaskCount)
}
