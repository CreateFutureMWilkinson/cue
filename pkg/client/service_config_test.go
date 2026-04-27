package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// ServiceConfigSuite covers the ServiceConfigClient adapter over
// /api/v1/services/{slack,email,calendar} plus /api/v1/services/status.
type ServiceConfigSuite struct {
	suite.Suite
}

func TestServiceConfig(t *testing.T) {
	suite.Run(t, new(ServiceConfigSuite))
}

// testSlackAccountID is a deterministic UUID used across Slack suite tests
// so path interpolation can be asserted directly.
var testSlackAccountID = uuid.MustParse("cafebabe-dead-beef-cafe-babecafebabe")

// testEmailAccountID is a deterministic UUID used across Email suite tests.
var testEmailAccountID = uuid.MustParse("deadbeef-cafe-babe-dead-beefdeadbeef")

// testCalendarAccountID is a deterministic UUID used across Calendar suite
// tests.
var testCalendarAccountID = uuid.MustParse("abcdefab-cdef-abcd-efab-cdefabcdefab")

// --- Slack tests (full coverage) ---

// TestListSlackAccountsReturnsArray verifies ListSlackAccounts issues
// GET /api/v1/services/slack and decodes the list payload into a slice of
// SlackAccount with snake_case fields intact.
func (s *ServiceConfigSuite) TestListSlackAccountsReturnsArray() {
	secondID := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/services/slack", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":           testSlackAccountID.String(),
				"name":         "Primary workspace",
				"workspace_id": "T12345",
				"enabled":      true,
				"created_at":   "2026-04-20T10:00:00Z",
			},
			{
				"id":           secondID.String(),
				"name":         "Side gig",
				"workspace_id": "T67890",
				"enabled":      false,
				"created_at":   "2026-04-21T11:00:00Z",
			},
		})
	}))
	defer ts.Close()

	sc := client.NewServiceConfigClient(client.New(ts.URL))
	accounts, err := sc.ListSlackAccounts(context.Background())
	s.Require().NoError(err)
	s.Require().Len(accounts, 2)

	s.Equal(testSlackAccountID, accounts[0].ID)
	s.Equal("Primary workspace", accounts[0].Name)
	s.Equal("T12345", accounts[0].WorkspaceID)
	s.True(accounts[0].Enabled)
	s.Equal("2026-04-20T10:00:00Z", accounts[0].CreatedAt)

	s.Equal(secondID, accounts[1].ID)
	s.Equal("Side gig", accounts[1].Name)
	s.Equal("T67890", accounts[1].WorkspaceID)
	s.False(accounts[1].Enabled)
}

// TestGetSlackAccountReturnsAccount verifies GetSlackAccount issues
// GET /api/v1/services/slack/{id} and decodes the SlackAccount payload.
func (s *ServiceConfigSuite) TestGetSlackAccountReturnsAccount() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/services/slack/"+testSlackAccountID.String(), r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":           testSlackAccountID.String(),
			"name":         "Fetched workspace",
			"workspace_id": "TFETCH",
			"enabled":      true,
			"created_at":   "2026-03-01T09:00:00Z",
		})
	}))
	defer ts.Close()

	sc := client.NewServiceConfigClient(client.New(ts.URL))
	acct, err := sc.GetSlackAccount(context.Background(), testSlackAccountID)
	s.Require().NoError(err)
	s.Require().NotNil(acct)
	s.Equal(testSlackAccountID, acct.ID)
	s.Equal("Fetched workspace", acct.Name)
	s.Equal("TFETCH", acct.WorkspaceID)
	s.True(acct.Enabled)
	s.Equal("2026-03-01T09:00:00Z", acct.CreatedAt)
}

// TestCreateSlackAccountPostsBodyWithToken verifies that CreateSlackAccount
// POSTs a body carrying the bot_token field and decodes the server's
// response (which deliberately omits bot_token).
func (s *ServiceConfigSuite) TestCreateSlackAccountPostsBodyWithToken() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/services/slack", r.URL.Path)

		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		s.Contains(body, "name")
		s.Contains(body, "bot_token")
		s.Contains(body, "workspace_id")
		s.Contains(body, "enabled")

		var name string
		s.Require().NoError(json.Unmarshal(body["name"], &name))
		s.Equal("New workspace", name)

		var botToken string
		s.Require().NoError(json.Unmarshal(body["bot_token"], &botToken))
		s.Equal("xoxb-secret", botToken)

		var workspaceID string
		s.Require().NoError(json.Unmarshal(body["workspace_id"], &workspaceID))
		s.Equal("TNEW", workspaceID)

		var enabled bool
		s.Require().NoError(json.Unmarshal(body["enabled"], &enabled))
		s.True(enabled)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":           testSlackAccountID.String(),
			"name":         "New workspace",
			"workspace_id": "TNEW",
			"enabled":      true,
			"created_at":   "2026-04-24T12:00:00Z",
		})
	}))
	defer ts.Close()

	sc := client.NewServiceConfigClient(client.New(ts.URL))
	acct, err := sc.CreateSlackAccount(context.Background(), client.CreateSlackAccountRequest{
		Name:        "New workspace",
		BotToken:    "xoxb-secret",
		WorkspaceID: "TNEW",
		Enabled:     true,
	})
	s.Require().NoError(err)
	s.Require().NotNil(acct)
	s.Equal(testSlackAccountID, acct.ID)
	s.Equal("New workspace", acct.Name)
	s.Equal("TNEW", acct.WorkspaceID)
	s.True(acct.Enabled)
	s.Equal("2026-04-24T12:00:00Z", acct.CreatedAt)
}

// TestUpdateSlackAccountPutsBody verifies UpdateSlackAccount issues
// PUT /api/v1/services/slack/{id} with the full body (including bot_token)
// and decodes the updated SlackAccount response.
func (s *ServiceConfigSuite) TestUpdateSlackAccountPutsBody() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPut, r.Method)
		s.Equal("/api/v1/services/slack/"+testSlackAccountID.String(), r.URL.Path)

		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		s.Contains(body, "name")
		s.Contains(body, "bot_token")
		s.Contains(body, "workspace_id")
		s.Contains(body, "enabled")

		var name string
		s.Require().NoError(json.Unmarshal(body["name"], &name))
		s.Equal("Updated workspace", name)

		var enabled bool
		s.Require().NoError(json.Unmarshal(body["enabled"], &enabled))
		s.False(enabled)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":           testSlackAccountID.String(),
			"name":         "Updated workspace",
			"workspace_id": "TUPD",
			"enabled":      false,
			"created_at":   "2026-03-01T10:00:00Z",
		})
	}))
	defer ts.Close()

	sc := client.NewServiceConfigClient(client.New(ts.URL))
	acct, err := sc.UpdateSlackAccount(context.Background(), testSlackAccountID, client.UpdateSlackAccountRequest{
		Name:        "Updated workspace",
		BotToken:    "xoxb-new-secret",
		WorkspaceID: "TUPD",
		Enabled:     false,
	})
	s.Require().NoError(err)
	s.Require().NotNil(acct)
	s.Equal(testSlackAccountID, acct.ID)
	s.Equal("Updated workspace", acct.Name)
	s.Equal("TUPD", acct.WorkspaceID)
	s.False(acct.Enabled)
}

// TestDeleteSlackAccountReturns204 verifies DeleteSlackAccount issues
// DELETE /api/v1/services/slack/{id} and tolerates a 204 No Content response.
func (s *ServiceConfigSuite) TestDeleteSlackAccountReturns204() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodDelete, r.Method)
		s.Equal("/api/v1/services/slack/"+testSlackAccountID.String(), r.URL.Path)

		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	sc := client.NewServiceConfigClient(client.New(ts.URL))
	err := sc.DeleteSlackAccount(context.Background(), testSlackAccountID)
	s.Require().NoError(err)
}

// TestToggleSlackAccountPostsEnabled verifies ToggleSlackAccount issues
// POST /api/v1/services/slack/{id}/toggle with a {"enabled": bool} body,
// for both true and false values (table-driven).
func (s *ServiceConfigSuite) TestToggleSlackAccountPostsEnabled() {
	for _, tc := range []struct {
		name    string
		enabled bool
	}{
		{"enable", true},
		{"disable", false},
	} {
		s.Run(tc.name, func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s.Equal(http.MethodPost, r.Method)
				s.Equal("/api/v1/services/slack/"+testSlackAccountID.String()+"/toggle", r.URL.Path)

				var body struct {
					Enabled bool `json:"enabled"`
				}
				s.Require().NoError(json.NewDecoder(r.Body).Decode(&body))
				s.Equal(tc.enabled, body.Enabled)

				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()

			sc := client.NewServiceConfigClient(client.New(ts.URL))
			err := sc.ToggleSlackAccount(context.Background(), testSlackAccountID, tc.enabled)
			s.Require().NoError(err)
		})
	}
}

// --- Email tests (minimal coverage) ---

// TestCreateEmailAccountPostsWithPassword verifies that CreateEmailAccount
// POSTs a body with all email fields including the password, to the Email
// route specifically (not Slack), and decodes the response (which omits
// password).
func (s *ServiceConfigSuite) TestCreateEmailAccountPostsWithPassword() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/services/email", r.URL.Path)

		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		s.Contains(body, "name")
		s.Contains(body, "imap_host")
		s.Contains(body, "imap_port")
		s.Contains(body, "username")
		s.Contains(body, "password")
		s.Contains(body, "encryption")
		s.Contains(body, "enabled")

		var password string
		s.Require().NoError(json.Unmarshal(body["password"], &password))
		s.Equal("app-specific-pw", password)

		var port int
		s.Require().NoError(json.Unmarshal(body["imap_port"], &port))
		s.Equal(993, port)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         testEmailAccountID.String(),
			"name":       "Primary inbox",
			"imap_host":  "imap.gmail.com",
			"imap_port":  993,
			"username":   "user@gmail.com",
			"encryption": "tls",
			"enabled":    true,
			"created_at": "2026-04-24T12:00:00Z",
		})
	}))
	defer ts.Close()

	sc := client.NewServiceConfigClient(client.New(ts.URL))
	acct, err := sc.CreateEmailAccount(context.Background(), client.CreateEmailAccountRequest{
		Name:       "Primary inbox",
		IMAPHost:   "imap.gmail.com",
		IMAPPort:   993,
		Username:   "user@gmail.com",
		Password:   "app-specific-pw",
		Encryption: "tls",
		Enabled:    true,
	})
	s.Require().NoError(err)
	s.Require().NotNil(acct)
	s.Equal(testEmailAccountID, acct.ID)
	s.Equal("Primary inbox", acct.Name)
	s.Equal("imap.gmail.com", acct.IMAPHost)
	s.Equal(993, acct.IMAPPort)
	s.Equal("user@gmail.com", acct.Username)
	s.Equal("tls", acct.Encryption)
	s.True(acct.Enabled)
	s.Equal("2026-04-24T12:00:00Z", acct.CreatedAt)
}

// TestListEmailAccountsReturnsArray verifies ListEmailAccounts hits the
// Email route and decodes the list payload into EmailAccount values.
func (s *ServiceConfigSuite) TestListEmailAccountsReturnsArray() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/services/email", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id":         testEmailAccountID.String(),
				"name":       "Work inbox",
				"imap_host":  "mail.example.com",
				"imap_port":  993,
				"username":   "me@example.com",
				"encryption": "tls",
				"enabled":    true,
				"created_at": "2026-04-20T10:00:00Z",
			},
		})
	}))
	defer ts.Close()

	sc := client.NewServiceConfigClient(client.New(ts.URL))
	accounts, err := sc.ListEmailAccounts(context.Background())
	s.Require().NoError(err)
	s.Require().Len(accounts, 1)
	s.Equal(testEmailAccountID, accounts[0].ID)
	s.Equal("Work inbox", accounts[0].Name)
	s.Equal("mail.example.com", accounts[0].IMAPHost)
	s.Equal(993, accounts[0].IMAPPort)
	s.Equal("me@example.com", accounts[0].Username)
	s.Equal("tls", accounts[0].Encryption)
	s.True(accounts[0].Enabled)
}

// --- Calendar tests (minimal coverage) ---

// TestCreateCalendarAccountPostsICSURL verifies CreateCalendarAccount POSTs
// a body with the ics_url field to the Calendar route and decodes the
// response.
func (s *ServiceConfigSuite) TestCreateCalendarAccountPostsICSURL() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/services/calendar", r.URL.Path)

		raw, err := io.ReadAll(r.Body)
		s.Require().NoError(err)
		var body map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(raw, &body))

		s.Contains(body, "name")
		s.Contains(body, "ics_url")
		s.Contains(body, "enabled")

		var icsURL string
		s.Require().NoError(json.Unmarshal(body["ics_url"], &icsURL))
		s.Equal("https://calendar.example.com/feed.ics", icsURL)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         testCalendarAccountID.String(),
			"name":       "Work calendar",
			"ics_url":    "https://calendar.example.com/feed.ics",
			"enabled":    true,
			"created_at": "2026-04-24T12:00:00Z",
		})
	}))
	defer ts.Close()

	sc := client.NewServiceConfigClient(client.New(ts.URL))
	acct, err := sc.CreateCalendarAccount(context.Background(), client.CreateCalendarAccountRequest{
		Name:    "Work calendar",
		ICSURL:  "https://calendar.example.com/feed.ics",
		Enabled: true,
	})
	s.Require().NoError(err)
	s.Require().NotNil(acct)
	s.Equal(testCalendarAccountID, acct.ID)
	s.Equal("Work calendar", acct.Name)
	s.Equal("https://calendar.example.com/feed.ics", acct.ICSURL)
	s.True(acct.Enabled)
	s.Equal("2026-04-24T12:00:00Z", acct.CreatedAt)
}

// TestToggleCalendarAccountPostsEnabled verifies ToggleCalendarAccount
// issues POST /api/v1/services/calendar/{id}/toggle with a
// {"enabled": false} body.
func (s *ServiceConfigSuite) TestToggleCalendarAccountPostsEnabled() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodPost, r.Method)
		s.Equal("/api/v1/services/calendar/"+testCalendarAccountID.String()+"/toggle", r.URL.Path)

		var body struct {
			Enabled bool `json:"enabled"`
		}
		s.Require().NoError(json.NewDecoder(r.Body).Decode(&body))
		s.False(body.Enabled)

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sc := client.NewServiceConfigClient(client.New(ts.URL))
	err := sc.ToggleCalendarAccount(context.Background(), testCalendarAccountID, false)
	s.Require().NoError(err)
}

// --- Status test ---

// TestServiceStatusReturnsAllServices verifies ServiceStatus issues
// GET /api/v1/services/status and decodes the {services, count} response
// into a slice covering all three service types with watcher_registered
// correctly mapped from snake_case.
func (s *ServiceConfigSuite) TestServiceStatusReturnsAllServices() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/services/status", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"services": []map[string]any{
				{
					"id":                 testSlackAccountID.String(),
					"type":               "slack",
					"name":               "Primary workspace",
					"enabled":            true,
					"watcher_registered": true,
				},
				{
					"id":                 testEmailAccountID.String(),
					"type":               "email",
					"name":               "Work inbox",
					"enabled":            true,
					"watcher_registered": false,
				},
				{
					"id":                 testCalendarAccountID.String(),
					"type":               "calendar",
					"name":               "Work calendar",
					"enabled":            false,
					"watcher_registered": false,
				},
			},
			"count": 3,
		})
	}))
	defer ts.Close()

	sc := client.NewServiceConfigClient(client.New(ts.URL))
	statuses, err := sc.ServiceStatus(context.Background())
	s.Require().NoError(err)
	s.Require().Len(statuses, 3)

	s.Equal(testSlackAccountID, statuses[0].ID)
	s.Equal("slack", statuses[0].Type)
	s.Equal("Primary workspace", statuses[0].Name)
	s.True(statuses[0].Enabled)
	s.True(statuses[0].WatcherRegistered)

	s.Equal(testEmailAccountID, statuses[1].ID)
	s.Equal("email", statuses[1].Type)
	s.True(statuses[1].Enabled)
	s.False(statuses[1].WatcherRegistered)

	s.Equal(testCalendarAccountID, statuses[2].ID)
	s.Equal("calendar", statuses[2].Type)
	s.False(statuses[2].Enabled)
	s.False(statuses[2].WatcherRegistered)
}

// --- Error path ---

// TestGetSlackAccountNotFoundReturnsAPIError verifies that a 404 from the
// server surfaces as an *APIError with ErrCodeNotFound.
func (s *ServiceConfigSuite) TestGetSlackAccountNotFoundReturnsAPIError() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal(http.MethodGet, r.Method)
		s.Equal("/api/v1/services/slack/"+testSlackAccountID.String(), r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "slack account not found",
		})
	}))
	defer ts.Close()

	sc := client.NewServiceConfigClient(client.New(ts.URL))
	acct, err := sc.GetSlackAccount(context.Background(), testSlackAccountID)
	s.Require().Error(err)
	s.Nil(acct)

	var apiErr *client.APIError
	s.Require().True(errors.As(err, &apiErr), "expected *APIError, got %T", err)
	s.Equal(client.ErrCodeNotFound, apiErr.Code)
	s.Equal(http.StatusNotFound, apiErr.StatusCode)
}
