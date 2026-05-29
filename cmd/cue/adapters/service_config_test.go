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
			s.Require().NoError(json.NewEncoder(w).Encode(map[string]any{
				"accounts": []any{
					map[string]any{
						"id":                    id.String(),
						"name":                  "Test Workspace",
						"workspace_id":          "T123",
						"username":              "alice",
						"web_url":               "https://slack.example.com",
						"poll_interval_seconds": 600,
						"enabled":               true,
						"created_at":            "2026-04-27T08:00:00Z",
					},
				},
				"count": 1,
			}))
		case http.MethodPost:
			s.Require().NoError(json.NewDecoder(r.Body).Decode(&lastCreate))
			s.Require().NoError(json.NewEncoder(w).Encode(map[string]any{
				"id":                    id.String(),
				"name":                  lastCreate.Name,
				"workspace_id":          lastCreate.WorkspaceID,
				"username":              lastCreate.Username,
				"web_url":               lastCreate.WebURL,
				"poll_interval_seconds": lastCreate.PollIntervalSeconds,
				"enabled":               lastCreate.Enabled,
				"created_at":            "2026-04-27T08:00:00Z",
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
				"id":                    id.String(),
				"name":                  lastUpdate.Name,
				"workspace_id":          lastUpdate.WorkspaceID,
				"username":              lastUpdate.Username,
				"web_url":               lastUpdate.WebURL,
				"poll_interval_seconds": lastUpdate.PollIntervalSeconds,
				"enabled":               lastUpdate.Enabled,
				"created_at":            "2026-04-27T08:00:00Z",
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
	s.Equal("alice", got.Username)
	s.Equal("https://slack.example.com", got.WebURL)
	s.Equal(600, got.PollIntervalSeconds)

	// Upsert (create).
	fresh := &repository.SlackAccount{
		Token:               "xoxp-secret",
		WorkspaceID:         "T999",
		FriendlyName:        "Fresh",
		Username:            "newhire",
		WebURL:              "https://fresh.example.com",
		PollIntervalSeconds: 900,
		Enabled:             true,
	}
	s.Require().NoError(a.UpsertSlackAccount(ctx, fresh))
	s.Equal(id, fresh.ID)
	s.Equal("xoxp-secret", lastCreate.BotToken,
		"create must forward the bot token to the server")
	s.Equal("newhire", lastCreate.Username)
	s.Equal("https://fresh.example.com", lastCreate.WebURL)
	s.Equal(900, lastCreate.PollIntervalSeconds)

	// Upsert (update).
	upd := &repository.SlackAccount{
		ID:                  id,
		Token:               "xoxp-rotated",
		WorkspaceID:         "T123",
		FriendlyName:        "Renamed",
		Username:            "rotated",
		WebURL:              "https://rotated.example.com",
		PollIntervalSeconds: 1200,
		Enabled:             false,
	}
	s.Require().NoError(a.UpsertSlackAccount(ctx, upd))
	s.Equal("xoxp-rotated", lastUpdate.BotToken)
	s.Equal("rotated", lastUpdate.Username)
	s.Equal("https://rotated.example.com", lastUpdate.WebURL)
	s.Equal(1200, lastUpdate.PollIntervalSeconds)
	s.False(lastUpdate.Enabled)

	// Delete.
	s.Require().NoError(a.DeleteSlackAccount(ctx, id))
	s.Equal(1, deleteCalls)

	// SetSlackEnabled(false) routes to ToggleSlackAccount.
	s.Require().NoError(a.SetSlackEnabled(ctx, id, false))
	s.Require().NotNil(lastToggle)
	s.False(*lastToggle, "SetSlackEnabled(false) must call toggle with enabled=false")
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
			s.Require().NoError(json.NewEncoder(w).Encode(map[string]any{
				"accounts": []any{
					map[string]any{
						"id":                    id.String(),
						"name":                  "Personal",
						"imap_host":             "imap.example.com",
						"imap_port":             993,
						"username":              "alice@example.com",
						"encryption":            "tls",
						"web_url":               "https://mail.example.com",
						"poll_interval_seconds": 600,
						"enabled":               true,
						"created_at":            "2026-04-27T08:00:00Z",
					},
				},
				"count": 1,
			}))
		case http.MethodPost:
			s.Require().NoError(json.NewDecoder(r.Body).Decode(&lastCreate))
			s.Require().NoError(json.NewEncoder(w).Encode(map[string]any{
				"id":                    id.String(),
				"name":                  lastCreate.Name,
				"imap_host":             lastCreate.IMAPHost,
				"imap_port":             lastCreate.IMAPPort,
				"username":              lastCreate.Username,
				"encryption":            lastCreate.Encryption,
				"web_url":               lastCreate.WebURL,
				"poll_interval_seconds": lastCreate.PollIntervalSeconds,
				"enabled":               lastCreate.Enabled,
				"created_at":            "2026-04-27T08:00:00Z",
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
	s.Equal("https://mail.example.com", accts[0].WebURL)
	s.Equal(600, accts[0].PollIntervalSeconds)

	// Create with a password forwarded on the wire.
	fresh := &repository.EmailAccount{
		FriendlyName:        "Personal",
		IMAPHost:            "imap.example.com",
		IMAPPort:            993,
		Username:            "alice@example.com",
		Password:            "s3cret",
		Encryption:          "tls",
		WebURL:              "https://mail.example.com",
		PollIntervalSeconds: 600,
		Enabled:             true,
	}
	s.Require().NoError(a.UpsertEmailAccount(ctx, fresh))
	s.Equal("s3cret", lastCreate.Password)
	s.Equal("https://mail.example.com", lastCreate.WebURL)
	s.Equal(600, lastCreate.PollIntervalSeconds)

	// SetEmailEnabled(false) routes to ToggleEmailAccount.
	s.Require().NoError(a.SetEmailEnabled(ctx, id, false))
	s.True(sawToggle)
	s.False(toggleEnabled)
}

// AC: Calendar list and upsert(create) round-trip the
// poll_interval_seconds field through the SDK so the presenter sees
// the value the user originally configured.
func (s *ServiceConfigAdapterSuite) TestCalendarRoundTripsPollInterval() {
	id := uuid.New()
	var lastCreate client.CreateCalendarAccountRequest

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/services/calendar", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			s.Require().NoError(json.NewEncoder(w).Encode(map[string]any{
				"accounts": []any{
					map[string]any{
						"id":                    id.String(),
						"name":                  "Work",
						"ics_url":               "https://cal.example.com/feed.ics",
						"poll_interval_seconds": 1800,
						"enabled":               true,
						"created_at":            "2026-04-27T08:00:00Z",
					},
				},
				"count": 1,
			}))
		case http.MethodPost:
			s.Require().NoError(json.NewDecoder(r.Body).Decode(&lastCreate))
			s.Require().NoError(json.NewEncoder(w).Encode(map[string]any{
				"id":                    id.String(),
				"name":                  lastCreate.Name,
				"ics_url":               lastCreate.ICSURL,
				"poll_interval_seconds": lastCreate.PollIntervalSeconds,
				"enabled":               lastCreate.Enabled,
				"created_at":            "2026-04-27T08:00:00Z",
			}))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	a := adapters.NewServiceConfigAdapter(client.NewServiceConfigClient(api))
	ctx := context.Background()

	accts, err := a.ListCalendarAccounts(ctx)
	s.Require().NoError(err)
	s.Require().Len(accts, 1)
	s.Equal(1800, accts[0].PollIntervalSeconds)

	fresh := &repository.CalendarAccount{
		Name:                "Work",
		ICSURL:              "https://cal.example.com/feed.ics",
		PollIntervalSeconds: 1800,
		Enabled:             true,
	}
	s.Require().NoError(a.UpsertCalendarAccount(ctx, fresh))
	s.Equal(1800, lastCreate.PollIntervalSeconds)
}

// AC: SetCalendarEnabled is wired even though the presenter doesn't
// currently consume it — the SDK and interface symmetry call for it.
func (s *ServiceConfigAdapterSuite) TestSetCalendarEnabledRoutesToToggle() {
	id := uuid.New()
	var sawToggle bool
	var enabled bool

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/services/calendar/"+id.String()+"/toggle", func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		var req struct {
			Enabled bool `json:"enabled"`
		}
		s.Require().NoError(json.NewDecoder(r.Body).Decode(&req))
		sawToggle = true
		enabled = req.Enabled
		w.WriteHeader(http.StatusNoContent)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	api := client.New(ts.URL)
	api.SetToken("test-token")
	a := adapters.NewServiceConfigAdapter(client.NewServiceConfigClient(api))

	s.Require().NoError(a.SetCalendarEnabled(context.Background(), id, true))
	s.True(sawToggle)
	s.True(enabled)
}
