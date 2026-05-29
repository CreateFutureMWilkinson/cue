package adapters_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/cmd/cue/adapters"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// ServiceConfigAdapterSuite covers the per-type CRUD paths and the
// RemoveWatcher → toggle-off translation.
type ServiceConfigAdapterSuite struct {
	suite.Suite
}

func TestServiceConfigAdapter(t *testing.T) {
	suite.Run(t, new(ServiceConfigAdapterSuite))
}

// AC: Slack list + upsert (create + update) + delete + RemoveWatcher
// (which toggles the matching account off via the workspace_id
// natural key) all flow through the SDK against an httptest server.
func (s *ServiceConfigAdapterSuite) TestSlackRoundTrip() {
	id := uuid.New()
	var lastCreate client.CreateSlackAccountRequest
	var lastUpdate client.CreateSlackAccountRequest
	var deleteCalls int
	var lastToggle *bool

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/services/slack", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			s.Require().NoError(json.NewEncoder(w).Encode([]any{
				map[string]any{
					"id":           id.String(),
					"name":         "Test Workspace",
					"workspace_id": "T123",
					"enabled":      true,
					"created_at":   "2026-04-27T08:00:00Z",
				},
			}))
		case http.MethodPost:
			s.Require().NoError(json.NewDecoder(r.Body).Decode(&lastCreate))
			s.Require().NoError(json.NewEncoder(w).Encode(map[string]any{
				"id":           id.String(),
				"name":         lastCreate.Name,
				"workspace_id": lastCreate.WorkspaceID,
				"enabled":      lastCreate.Enabled,
				"created_at":   "2026-04-27T08:00:00Z",
			}))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/services/slack/"+id.String(), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPut:
			s.Require().NoError(json.NewDecoder(r.Body).Decode(&lastUpdate))
			s.Require().NoError(json.NewEncoder(w).Encode(map[string]any{
				"id":           id.String(),
				"name":         lastUpdate.Name,
				"workspace_id": lastUpdate.WorkspaceID,
				"enabled":      lastUpdate.Enabled,
				"created_at":   "2026-04-27T08:00:00Z",
			}))
		case http.MethodDelete:
			deleteCalls++
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/services/slack/"+id.String()+"/toggle", func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		var req struct {
			Enabled bool `json:"enabled"`
		}
		s.Require().NoError(json.NewDecoder(r.Body).Decode(&req))
		lastToggle = &req.Enabled
		w.WriteHeader(http.StatusNoContent)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	a := adapters.NewServiceConfigAdapter(client.NewServiceConfigClient(api))
	ctx := context.Background()

	// List.
	accts, err := a.ListSlackAccounts(ctx)
	s.Require().NoError(err)
	s.Require().Len(accts, 1)
	got := accts[0]
	s.Equal(id, got.ID)
	s.Equal("T123", got.WorkspaceID)
	s.Equal("Test Workspace", got.FriendlyName)

	// Upsert (create).
	fresh := &repository.SlackAccount{
		Token:        "xoxp-secret",
		WorkspaceID:  "T999",
		FriendlyName: "Fresh",
		Enabled:      true,
	}
	s.Require().NoError(a.UpsertSlackAccount(ctx, fresh))
	s.Equal(id, fresh.ID)
	s.Equal("xoxp-secret", lastCreate.BotToken,
		"create must forward the bot token to the server")

	// Upsert (update).
	upd := &repository.SlackAccount{
		ID:           id,
		Token:        "xoxp-rotated",
		WorkspaceID:  "T123",
		FriendlyName: "Renamed",
		Enabled:      false,
	}
	s.Require().NoError(a.UpsertSlackAccount(ctx, upd))
	s.Equal("xoxp-rotated", lastUpdate.BotToken)
	s.False(lastUpdate.Enabled)

	// Delete.
	s.Require().NoError(a.DeleteSlackAccount(ctx, id))
	s.Equal(1, deleteCalls)

	// RemoveWatcher routes "slack:T123" → ToggleSlackAccount(id, false).
	a.RemoveWatcher("slack:T123")
	s.Require().NotNil(lastToggle)
	s.False(*lastToggle, "RemoveWatcher must call toggle with enabled=false")
}

// AC: Email list/upsert(create)/delete and RemoveWatcher routing on
// the email prefix all work end-to-end.
func (s *ServiceConfigAdapterSuite) TestEmailMinimalRoundTrip() {
	id := uuid.New()
	var lastCreate client.CreateEmailAccountRequest
	var toggleEnabled bool
	var sawToggle bool

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/services/email", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			s.Require().NoError(json.NewEncoder(w).Encode([]any{
				map[string]any{
					"id":         id.String(),
					"name":       "Personal",
					"imap_host":  "imap.example.com",
					"imap_port":  993,
					"username":   "alice@example.com",
					"encryption": "tls",
					"enabled":    true,
					"created_at": "2026-04-27T08:00:00Z",
				},
			}))
		case http.MethodPost:
			s.Require().NoError(json.NewDecoder(r.Body).Decode(&lastCreate))
			s.Require().NoError(json.NewEncoder(w).Encode(map[string]any{
				"id":         id.String(),
				"name":       lastCreate.Name,
				"imap_host":  lastCreate.IMAPHost,
				"imap_port":  lastCreate.IMAPPort,
				"username":   lastCreate.Username,
				"encryption": lastCreate.Encryption,
				"enabled":    lastCreate.Enabled,
				"created_at": "2026-04-27T08:00:00Z",
			}))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/v1/services/email/"+id.String()+"/toggle", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		s.Require().NoError(json.NewDecoder(r.Body).Decode(&req))
		sawToggle = true
		toggleEnabled = req.Enabled
		w.WriteHeader(http.StatusNoContent)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	a := adapters.NewServiceConfigAdapter(client.NewServiceConfigClient(api))
	ctx := context.Background()

	// List (passwords are never returned on the wire).
	accts, err := a.ListEmailAccounts(ctx)
	s.Require().NoError(err)
	s.Require().Len(accts, 1)
	s.Empty(accts[0].Password, "password is never returned by the server")

	// Create with a password forwarded on the wire.
	fresh := &repository.EmailAccount{
		FriendlyName: "Personal",
		IMAPHost:     "imap.example.com",
		IMAPPort:     993,
		Username:     "alice@example.com",
		Password:     "s3cret",
		Encryption:   "tls",
		Enabled:      true,
	}
	s.Require().NoError(a.UpsertEmailAccount(ctx, fresh))
	s.Equal("s3cret", lastCreate.Password)

	// RemoveWatcher matches by username.
	a.RemoveWatcher("email:alice@example.com")
	s.True(sawToggle)
	s.False(toggleEnabled)
}

// AC: an unknown watcher name is a silent no-op rather than a panic.
func (s *ServiceConfigAdapterSuite) TestRemoveWatcherUnknownIsNoop() {
	mux := http.NewServeMux()
	// No handlers registered: any request would 404, but the adapter
	// should never reach a network call for an unparseable name.
	ts := httptest.NewServer(mux)
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	a := adapters.NewServiceConfigAdapter(client.NewServiceConfigClient(api))

	// Empty name, missing colon, empty key — all swallowed.
	a.RemoveWatcher("")
	a.RemoveWatcher("malformed")
	a.RemoveWatcher("slack:")
	// No assertion needed: the test passes as long as we don't panic.
}
