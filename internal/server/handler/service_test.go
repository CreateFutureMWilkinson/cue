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
	"github.com/CreateFutureMWilkinson/cue/internal/service/servicemanager"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// mockServiceManager implements handler.ServiceManager for testing.
type mockServiceManager struct {
	// Slack
	slackAccounts    []*repository.SlackAccount
	slackAccount     *repository.SlackAccount
	slackAccountErr  error
	createdSlack     *repository.SlackAccount
	createSlackErr   error
	updatedSlack     *repository.SlackAccount
	updateSlackErr   error
	deleteSlackErr   error
	toggleSlackErr   error
	createSlackInput *repository.SlackAccount

	// Email
	emailAccounts    []*repository.EmailAccount
	emailAccount     *repository.EmailAccount
	emailAccountErr  error
	createdEmail     *repository.EmailAccount
	createEmailErr   error
	updatedEmail     *repository.EmailAccount
	updateEmailErr   error
	deleteEmailErr   error
	toggleEmailErr   error
	createEmailInput *repository.EmailAccount

	// Calendar
	calendarAccounts    []*repository.CalendarAccount
	calendarAccount     *repository.CalendarAccount
	calendarAccountErr  error
	createdCalendar     *repository.CalendarAccount
	createCalendarErr   error
	updatedCalendar     *repository.CalendarAccount
	updateCalendarErr   error
	deleteCalendarErr   error
	toggleCalendarErr   error
	createCalendarInput *repository.CalendarAccount

	// Status
	statuses  []servicemanager.ServiceStatus
	statusErr error

	// Captured args
	capturedID      uuid.UUID
	capturedEnabled bool
}

func (m *mockServiceManager) ListSlackAccounts(_ context.Context) ([]*repository.SlackAccount, error) {
	return m.slackAccounts, m.slackAccountErr
}

func (m *mockServiceManager) ListEmailAccounts(_ context.Context) ([]*repository.EmailAccount, error) {
	return m.emailAccounts, m.emailAccountErr
}

func (m *mockServiceManager) ListCalendarAccounts(_ context.Context) ([]*repository.CalendarAccount, error) {
	return m.calendarAccounts, m.calendarAccountErr
}

func (m *mockServiceManager) GetSlackAccount(_ context.Context, id uuid.UUID) (*repository.SlackAccount, error) {
	m.capturedID = id
	return m.slackAccount, m.slackAccountErr
}

func (m *mockServiceManager) GetEmailAccount(_ context.Context, id uuid.UUID) (*repository.EmailAccount, error) {
	m.capturedID = id
	return m.emailAccount, m.emailAccountErr
}

func (m *mockServiceManager) GetCalendarAccount(_ context.Context, id uuid.UUID) (*repository.CalendarAccount, error) {
	m.capturedID = id
	return m.calendarAccount, m.calendarAccountErr
}

func (m *mockServiceManager) CreateSlackAccount(_ context.Context, acct *repository.SlackAccount) (*repository.SlackAccount, error) {
	m.createSlackInput = acct
	return m.createdSlack, m.createSlackErr
}

func (m *mockServiceManager) CreateEmailAccount(_ context.Context, acct *repository.EmailAccount) (*repository.EmailAccount, error) {
	m.createEmailInput = acct
	return m.createdEmail, m.createEmailErr
}

func (m *mockServiceManager) CreateCalendarAccount(_ context.Context, acct *repository.CalendarAccount) (*repository.CalendarAccount, error) {
	m.createCalendarInput = acct
	return m.createdCalendar, m.createCalendarErr
}

func (m *mockServiceManager) UpdateSlackAccount(_ context.Context, id uuid.UUID, acct *repository.SlackAccount) (*repository.SlackAccount, error) {
	m.capturedID = id
	m.createSlackInput = acct
	return m.updatedSlack, m.updateSlackErr
}

func (m *mockServiceManager) UpdateEmailAccount(_ context.Context, id uuid.UUID, acct *repository.EmailAccount) (*repository.EmailAccount, error) {
	m.capturedID = id
	m.createEmailInput = acct
	return m.updatedEmail, m.updateEmailErr
}

func (m *mockServiceManager) UpdateCalendarAccount(_ context.Context, id uuid.UUID, acct *repository.CalendarAccount) (*repository.CalendarAccount, error) {
	m.capturedID = id
	m.createCalendarInput = acct
	return m.updatedCalendar, m.updateCalendarErr
}

func (m *mockServiceManager) DeleteSlackAccount(_ context.Context, id uuid.UUID) error {
	m.capturedID = id
	return m.deleteSlackErr
}

func (m *mockServiceManager) DeleteEmailAccount(_ context.Context, id uuid.UUID) error {
	m.capturedID = id
	return m.deleteEmailErr
}

func (m *mockServiceManager) DeleteCalendarAccount(_ context.Context, id uuid.UUID) error {
	m.capturedID = id
	return m.deleteCalendarErr
}

func (m *mockServiceManager) ToggleSlackAccount(_ context.Context, id uuid.UUID, enabled bool) error {
	m.capturedID = id
	m.capturedEnabled = enabled
	return m.toggleSlackErr
}

func (m *mockServiceManager) ToggleEmailAccount(_ context.Context, id uuid.UUID, enabled bool) error {
	m.capturedID = id
	m.capturedEnabled = enabled
	return m.toggleEmailErr
}

func (m *mockServiceManager) ToggleCalendarAccount(_ context.Context, id uuid.UUID, enabled bool) error {
	m.capturedID = id
	m.capturedEnabled = enabled
	return m.toggleCalendarErr
}

func (m *mockServiceManager) Status(_ context.Context) ([]servicemanager.ServiceStatus, error) {
	return m.statuses, m.statusErr
}

// ServiceHandlerSuite tests the service handler endpoints.
type ServiceHandlerSuite struct {
	suite.Suite
}

func TestServiceHandler(t *testing.T) {
	suite.Run(t, new(ServiceHandlerSuite))
}

// --- Slack tests ---

func (s *ServiceHandlerSuite) TestListSlackAccounts_Success() {
	now := time.Now().UTC().Truncate(time.Second)
	id1 := uuid.New()

	mock := &mockServiceManager{
		slackAccounts: []*repository.SlackAccount{
			{
				ID:                  id1,
				FriendlyName:        "Work Slack",
				WorkspaceID:         "T12345",
				Username:            "alice",
				WebURL:              "https://slack.example.com",
				PollIntervalSeconds: 600,
				Enabled:             true,
				CreatedAt:           now,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/services/slack", nil)
	rec := httptest.NewRecorder()

	handler.ListSlackAccountsHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err)

	accounts, ok := body["accounts"].([]any)
	s.Require().True(ok)
	s.Len(accounts, 1)

	s.Equal(float64(1), body["count"])

	acct := accounts[0].(map[string]any)
	s.Equal(id1.String(), acct["id"])
	s.Equal("Work Slack", acct["name"])
	s.Equal("T12345", acct["workspace_id"])
	s.Equal(true, acct["enabled"])
	s.Equal("alice", acct["username"])
	s.Equal("https://slack.example.com", acct["web_url"])
	s.Equal(float64(600), acct["poll_interval_seconds"])
}

func (s *ServiceHandlerSuite) TestGetSlackAccount_Success() {
	now := time.Now().UTC().Truncate(time.Second)
	id1 := uuid.New()

	mock := &mockServiceManager{
		slackAccount: &repository.SlackAccount{
			ID:           id1,
			FriendlyName: "Work Slack",
			WorkspaceID:  "T12345",
			Token:        "***", // ServiceManager masks credentials
			Enabled:      true,
			CreatedAt:    now,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/services/slack/"+id1.String(), nil)
	req.SetPathValue("id", id1.String())
	rec := httptest.NewRecorder()

	handler.GetSlackAccountHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err)

	s.Equal(id1.String(), body["id"])
	s.Equal("Work Slack", body["name"])
	// Token should not appear in the response item.
	_, hasToken := body["token"]
	s.False(hasToken, "token must not appear in response")
	_, hasBotToken := body["bot_token"]
	s.False(hasBotToken, "bot_token must not appear in response")
}

func (s *ServiceHandlerSuite) TestGetSlackAccount_InvalidID() {
	mock := &mockServiceManager{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/services/slack/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()

	handler.GetSlackAccountHandler(mock)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *ServiceHandlerSuite) TestGetSlackAccount_NotFound() {
	mock := &mockServiceManager{
		slackAccountErr: repository.ErrNotFound,
	}

	unknownID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/services/slack/"+unknownID.String(), nil)
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.GetSlackAccountHandler(mock)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code)
}

func (s *ServiceHandlerSuite) TestCreateSlackAccount_Success() {
	now := time.Now().UTC().Truncate(time.Second)
	createdID := uuid.New()

	mock := &mockServiceManager{
		createdSlack: &repository.SlackAccount{
			ID:           createdID,
			FriendlyName: "New Slack",
			WorkspaceID:  "T99999",
			Token:        "***",
			Enabled:      true,
			CreatedAt:    now,
		},
	}

	body := `{"name":"New Slack","bot_token":"xoxb-secret","workspace_id":"T99999","username":"alice","web_url":"https://slack.example.com","poll_interval_seconds":600,"enabled":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/services/slack", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateSlackAccountHandler(mock)(rec, req)

	s.Equal(http.StatusCreated, rec.Code)

	var respBody map[string]any
	err := json.NewDecoder(rec.Body).Decode(&respBody)
	s.Require().NoError(err)

	s.Equal(createdID.String(), respBody["id"])
	s.Equal("New Slack", respBody["name"])
	s.Equal("T99999", respBody["workspace_id"])

	// Verify the handler passed through token + new fields to the service layer.
	s.Equal("xoxb-secret", mock.createSlackInput.Token)
	s.Equal("alice", mock.createSlackInput.Username)
	s.Equal("https://slack.example.com", mock.createSlackInput.WebURL)
	s.Equal(600, mock.createSlackInput.PollIntervalSeconds)
}

func (s *ServiceHandlerSuite) TestCreateSlackAccount_BadJSON() {
	mock := &mockServiceManager{}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/services/slack", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateSlackAccountHandler(mock)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}

func (s *ServiceHandlerSuite) TestUpdateSlackAccount_Success() {
	now := time.Now().UTC().Truncate(time.Second)
	id1 := uuid.New()

	mock := &mockServiceManager{
		updatedSlack: &repository.SlackAccount{
			ID:           id1,
			FriendlyName: "Updated Slack",
			WorkspaceID:  "T12345",
			Token:        "***",
			Enabled:      true,
			CreatedAt:    now,
		},
	}

	body := `{"name":"Updated Slack","bot_token":"","workspace_id":"T12345","username":"bob","web_url":"https://updated.example.com","poll_interval_seconds":300,"enabled":true}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/services/slack/"+id1.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", id1.String())
	rec := httptest.NewRecorder()

	handler.UpdateSlackAccountHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var respBody map[string]any
	err := json.NewDecoder(rec.Body).Decode(&respBody)
	s.Require().NoError(err)

	s.Equal(id1.String(), respBody["id"])
	s.Equal("Updated Slack", respBody["name"])

	// Verify update forwarded new fields.
	s.Equal("bob", mock.createSlackInput.Username)
	s.Equal("https://updated.example.com", mock.createSlackInput.WebURL)
	s.Equal(300, mock.createSlackInput.PollIntervalSeconds)
}

func (s *ServiceHandlerSuite) TestDeleteSlackAccount_Success() {
	mock := &mockServiceManager{}

	id1 := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/services/slack/"+id1.String(), nil)
	req.SetPathValue("id", id1.String())
	rec := httptest.NewRecorder()

	handler.DeleteSlackAccountHandler(mock)(rec, req)

	s.Equal(http.StatusNoContent, rec.Code)
	s.Equal(id1, mock.capturedID)
}

func (s *ServiceHandlerSuite) TestToggleSlackAccount_Success() {
	mock := &mockServiceManager{}

	id1 := uuid.New()
	body := `{"enabled":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/services/slack/"+id1.String()+"/toggle", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", id1.String())
	rec := httptest.NewRecorder()

	handler.ToggleSlackAccountHandler(mock)(rec, req)

	s.Equal(http.StatusNoContent, rec.Code)
	s.Equal(id1, mock.capturedID)
	s.Equal(false, mock.capturedEnabled)
}

// --- Email tests ---

func (s *ServiceHandlerSuite) TestListEmailAccounts_Success() {
	now := time.Now().UTC().Truncate(time.Second)
	id1 := uuid.New()

	mock := &mockServiceManager{
		emailAccounts: []*repository.EmailAccount{
			{
				ID:                  id1,
				FriendlyName:        "Work Email",
				IMAPHost:            "imap.example.com",
				IMAPPort:            993,
				Username:            "user@example.com",
				Encryption:          "tls",
				WebURL:              "https://mail.example.com",
				PollIntervalSeconds: 600,
				Enabled:             true,
				CreatedAt:           now,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/services/email", nil)
	rec := httptest.NewRecorder()

	handler.ListEmailAccountsHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err)

	accounts, ok := body["accounts"].([]any)
	s.Require().True(ok)
	s.Len(accounts, 1)

	acct := accounts[0].(map[string]any)
	s.Equal(id1.String(), acct["id"])
	s.Equal("Work Email", acct["name"])
	s.Equal("imap.example.com", acct["imap_host"])
	s.Equal(float64(993), acct["imap_port"])
	s.Equal("https://mail.example.com", acct["web_url"])
	s.Equal(float64(600), acct["poll_interval_seconds"])
	// Password must not appear in the list response.
	_, hasPassword := acct["password"]
	s.False(hasPassword, "password must not appear in response")
}

func (s *ServiceHandlerSuite) TestCreateEmailAccount_Success() {
	now := time.Now().UTC().Truncate(time.Second)
	createdID := uuid.New()

	mock := &mockServiceManager{
		createdEmail: &repository.EmailAccount{
			ID:           createdID,
			FriendlyName: "New Email",
			IMAPHost:     "imap.example.com",
			IMAPPort:     993,
			Username:     "user@example.com",
			Password:     "***",
			Encryption:   "tls",
			Enabled:      true,
			CreatedAt:    now,
		},
	}

	body := `{"name":"New Email","imap_host":"imap.example.com","imap_port":993,"username":"user@example.com","password":"secret","encryption":"tls","web_url":"https://mail.example.com","poll_interval_seconds":600,"enabled":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/services/email", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateEmailAccountHandler(mock)(rec, req)

	s.Equal(http.StatusCreated, rec.Code)

	var respBody map[string]any
	err := json.NewDecoder(rec.Body).Decode(&respBody)
	s.Require().NoError(err)

	s.Equal(createdID.String(), respBody["id"])
	s.Equal("New Email", respBody["name"])

	// Verify the handler passed password + new fields to the service layer.
	s.Equal("secret", mock.createEmailInput.Password)
	s.Equal("https://mail.example.com", mock.createEmailInput.WebURL)
	s.Equal(600, mock.createEmailInput.PollIntervalSeconds)
}

// --- Calendar tests ---

func (s *ServiceHandlerSuite) TestListCalendarAccounts_Success() {
	now := time.Now().UTC().Truncate(time.Second)
	id1 := uuid.New()

	mock := &mockServiceManager{
		calendarAccounts: []*repository.CalendarAccount{
			{
				ID:                  id1,
				Name:                "Work Calendar",
				ICSURL:              "https://example.com/cal.ics",
				PollIntervalSeconds: 1800,
				Enabled:             true,
				CreatedAt:           now,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/services/calendar", nil)
	rec := httptest.NewRecorder()

	handler.ListCalendarAccountsHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err)

	accounts, ok := body["accounts"].([]any)
	s.Require().True(ok)
	s.Len(accounts, 1)

	acct := accounts[0].(map[string]any)
	s.Equal(id1.String(), acct["id"])
	s.Equal("Work Calendar", acct["name"])
	s.Equal("https://example.com/cal.ics", acct["ics_url"])
	s.Equal(float64(1800), acct["poll_interval_seconds"])
}

func (s *ServiceHandlerSuite) TestCreateCalendarAccount_Success() {
	now := time.Now().UTC().Truncate(time.Second)
	createdID := uuid.New()

	mock := &mockServiceManager{
		createdCalendar: &repository.CalendarAccount{
			ID:        createdID,
			Name:      "New Calendar",
			ICSURL:    "https://example.com/new.ics",
			Enabled:   true,
			CreatedAt: now,
		},
	}

	body := `{"name":"New Calendar","ics_url":"https://example.com/new.ics","poll_interval_seconds":1800,"enabled":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/services/calendar", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateCalendarAccountHandler(mock)(rec, req)

	s.Equal(http.StatusCreated, rec.Code)

	var respBody map[string]any
	err := json.NewDecoder(rec.Body).Decode(&respBody)
	s.Require().NoError(err)

	s.Equal(createdID.String(), respBody["id"])
	s.Equal("New Calendar", respBody["name"])
	s.Equal("https://example.com/new.ics", respBody["ics_url"])

	// Verify create forwarded the poll interval to the service layer.
	s.Equal(1800, mock.createCalendarInput.PollIntervalSeconds)
}

// --- Status test ---

func (s *ServiceHandlerSuite) TestServiceStatus_Success() {
	id1 := uuid.New()
	id2 := uuid.New()

	mock := &mockServiceManager{
		statuses: []servicemanager.ServiceStatus{
			{
				ID:                id1,
				Type:              "slack",
				Name:              "Work Slack",
				Enabled:           true,
				WatcherRegistered: true,
			},
			{
				ID:                id2,
				Type:              "email",
				Name:              "Work Email",
				Enabled:           true,
				WatcherRegistered: false,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/services/status", nil)
	rec := httptest.NewRecorder()

	handler.ServiceStatusHandler(mock)(rec, req)

	s.Equal(http.StatusOK, rec.Code)

	var body map[string]any
	err := json.NewDecoder(rec.Body).Decode(&body)
	s.Require().NoError(err)

	services, ok := body["services"].([]any)
	s.Require().True(ok)
	s.Len(services, 2)

	s.Equal(float64(2), body["count"])

	svc1 := services[0].(map[string]any)
	s.Equal(id1.String(), svc1["id"])
	s.Equal("slack", svc1["type"])
	s.Equal("Work Slack", svc1["name"])
	s.Equal(true, svc1["enabled"])
	s.Equal(true, svc1["watcher_registered"])

	svc2 := services[1].(map[string]any)
	s.Equal(id2.String(), svc2["id"])
	s.Equal("email", svc2["type"])
	s.Equal(false, svc2["watcher_registered"])
}

// --- Error path tests ---

func (s *ServiceHandlerSuite) TestDeleteSlackAccount_NotFound() {
	mock := &mockServiceManager{
		deleteSlackErr: fmt.Errorf("account not found"),
	}

	unknownID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/services/slack/"+unknownID.String(), nil)
	req.SetPathValue("id", unknownID.String())
	rec := httptest.NewRecorder()

	handler.DeleteSlackAccountHandler(mock)(rec, req)

	s.Equal(http.StatusNotFound, rec.Code)
}

func (s *ServiceHandlerSuite) TestToggleSlackAccount_BadJSON() {
	mock := &mockServiceManager{}

	id1 := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/services/slack/"+id1.String()+"/toggle", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", id1.String())
	rec := httptest.NewRecorder()

	handler.ToggleSlackAccountHandler(mock)(rec, req)

	s.Equal(http.StatusBadRequest, rec.Code)
}
