package client

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// slackServicePath, emailServicePath, and calendarServicePath are the base
// URL paths for the per-service REST resources. Each exposes the same
// CRUD + toggle verbs; servicesStatusPath is the cross-service aggregate.
const (
	slackServicePath    = "/api/v1/services/slack"
	emailServicePath    = "/api/v1/services/email"
	calendarServicePath = "/api/v1/services/calendar"
	servicesStatusPath  = "/api/v1/services/status"
)

// SlackAccount mirrors the server's slackAccountItem DTO returned by
// /api/v1/services/slack routes. The bot_token is NEVER returned on
// responses — only accepted on create/update request bodies.
type SlackAccount struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	WorkspaceID         string    `json:"workspace_id"`
	Username            string    `json:"username"`
	WebURL              string    `json:"web_url"`
	PollIntervalSeconds int       `json:"poll_interval_seconds"`
	Enabled             bool      `json:"enabled"`
	CreatedAt           string    `json:"created_at"`
}

// CreateSlackAccountRequest is the POST body for creating a Slack account via
// POST /api/v1/services/slack. The bot_token is carried here but omitted
// from the response DTO; the server stores it and references it from the
// watcher registration.
type CreateSlackAccountRequest struct {
	Name                string `json:"name"`
	BotToken            string `json:"bot_token"`
	WorkspaceID         string `json:"workspace_id"`
	Username            string `json:"username"`
	WebURL              string `json:"web_url"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
	Enabled             bool   `json:"enabled"`
}

// UpdateSlackAccountRequest is the PUT body for full replacement via
// PUT /api/v1/services/slack/{id}. It is structurally identical to
// CreateSlackAccountRequest so we alias the type.
type UpdateSlackAccountRequest = CreateSlackAccountRequest

// EmailAccount mirrors the server's emailAccountItem DTO returned by
// /api/v1/services/email routes. The password is NEVER returned on
// responses — only accepted on create/update request bodies.
type EmailAccount struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	IMAPHost            string    `json:"imap_host"`
	IMAPPort            int       `json:"imap_port"`
	Username            string    `json:"username"`
	Encryption          string    `json:"encryption"`
	WebURL              string    `json:"web_url"`
	PollIntervalSeconds int       `json:"poll_interval_seconds"`
	Enabled             bool      `json:"enabled"`
	CreatedAt           string    `json:"created_at"`
}

// CreateEmailAccountRequest is the POST body for creating an Email account
// via POST /api/v1/services/email. The password is carried here but omitted
// from the response DTO.
type CreateEmailAccountRequest struct {
	Name                string `json:"name"`
	IMAPHost            string `json:"imap_host"`
	IMAPPort            int    `json:"imap_port"`
	Username            string `json:"username"`
	Password            string `json:"password"`
	Encryption          string `json:"encryption"`
	WebURL              string `json:"web_url"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
	Enabled             bool   `json:"enabled"`
}

// UpdateEmailAccountRequest is the PUT body for full replacement via
// PUT /api/v1/services/email/{id}. Structurally identical to
// CreateEmailAccountRequest.
type UpdateEmailAccountRequest = CreateEmailAccountRequest

// CalendarAccount mirrors the server's calendarAccountItem DTO returned by
// /api/v1/services/calendar routes.
type CalendarAccount struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	ICSURL              string    `json:"ics_url"`
	PollIntervalSeconds int       `json:"poll_interval_seconds"`
	Enabled             bool      `json:"enabled"`
	CreatedAt           string    `json:"created_at"`
}

// CreateCalendarAccountRequest is the POST body for creating a Calendar
// account via POST /api/v1/services/calendar.
type CreateCalendarAccountRequest struct {
	Name                string `json:"name"`
	ICSURL              string `json:"ics_url"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
	Enabled             bool   `json:"enabled"`
}

// UpdateCalendarAccountRequest is the PUT body for full replacement via
// PUT /api/v1/services/calendar/{id}. Structurally identical to
// CreateCalendarAccountRequest.
type UpdateCalendarAccountRequest = CreateCalendarAccountRequest

// ServiceStatus describes a single service entry in the cross-service
// /api/v1/services/status response. Type is one of "slack", "email", or
// "calendar"; WatcherRegistered reports whether the orchestrator has a
// live watcher goroutine for this account.
type ServiceStatus struct {
	ID                uuid.UUID `json:"id"`
	Type              string    `json:"type"`
	Name              string    `json:"name"`
	Enabled           bool      `json:"enabled"`
	WatcherRegistered bool      `json:"watcher_registered"`
}

// ServiceConfigClient wraps the /api/v1/services/* routes: per-type CRUD
// and toggle for Slack, Email, and Calendar accounts, plus the cross-type
// status aggregate. Each per-type group mirrors the same verbs
// (List/Get/Create/Update/Delete/Toggle) so the three account kinds remain
// symmetrical for callers.
type ServiceConfigClient interface {
	// Slack
	ListSlackAccounts(ctx context.Context) ([]SlackAccount, error)
	GetSlackAccount(ctx context.Context, id uuid.UUID) (*SlackAccount, error)
	CreateSlackAccount(ctx context.Context, req CreateSlackAccountRequest) (*SlackAccount, error)
	UpdateSlackAccount(ctx context.Context, id uuid.UUID, req UpdateSlackAccountRequest) (*SlackAccount, error)
	DeleteSlackAccount(ctx context.Context, id uuid.UUID) error
	ToggleSlackAccount(ctx context.Context, id uuid.UUID, enabled bool) error

	// Email
	ListEmailAccounts(ctx context.Context) ([]EmailAccount, error)
	GetEmailAccount(ctx context.Context, id uuid.UUID) (*EmailAccount, error)
	CreateEmailAccount(ctx context.Context, req CreateEmailAccountRequest) (*EmailAccount, error)
	UpdateEmailAccount(ctx context.Context, id uuid.UUID, req UpdateEmailAccountRequest) (*EmailAccount, error)
	DeleteEmailAccount(ctx context.Context, id uuid.UUID) error
	ToggleEmailAccount(ctx context.Context, id uuid.UUID, enabled bool) error

	// Calendar
	ListCalendarAccounts(ctx context.Context) ([]CalendarAccount, error)
	GetCalendarAccount(ctx context.Context, id uuid.UUID) (*CalendarAccount, error)
	CreateCalendarAccount(ctx context.Context, req CreateCalendarAccountRequest) (*CalendarAccount, error)
	UpdateCalendarAccount(ctx context.Context, id uuid.UUID, req UpdateCalendarAccountRequest) (*CalendarAccount, error)
	DeleteCalendarAccount(ctx context.Context, id uuid.UUID) error
	ToggleCalendarAccount(ctx context.Context, id uuid.UUID, enabled bool) error

	// Cross-service
	ServiceStatus(ctx context.Context) ([]ServiceStatus, error)
}

// serviceConfigAdapter is the concrete ServiceConfigClient backed by an
// *APIClient.
type serviceConfigAdapter struct {
	client *APIClient
}

// NewServiceConfigClient returns a ServiceConfigClient backed by the given
// APIClient.
func NewServiceConfigClient(c *APIClient) ServiceConfigClient {
	return &serviceConfigAdapter{client: c}
}

// toggleRequest is the shared JSON body for POST /.../{id}/toggle requests
// across all service types. The server accepts the same {"enabled": bool}
// shape regardless of the service being toggled.
type toggleRequest struct {
	Enabled bool `json:"enabled"`
}

// servicePath constructs the full URL path for service operations by combining
// the base service path with the UUID-based resource path.
func servicePath(basePath string, id uuid.UUID) string {
	return basePath + "/" + id.String()
}

// doServiceList performs a GET request to list all accounts for a service
// type. The server wraps list responses as {"accounts": [...], "count": N}
// (see internal/server/handler/service.go); the count field is unused by
// the SDK.
func doServiceList[T any](ctx context.Context, client *APIClient, basePath string) ([]T, error) {
	var resp struct {
		Accounts []T `json:"accounts"`
	}
	if err := client.doJSON(ctx, http.MethodGet, basePath, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Accounts == nil {
		return []T{}, nil
	}
	return resp.Accounts, nil
}

// doServiceGet performs a GET request to retrieve a single account by ID.
func doServiceGet[T any](ctx context.Context, client *APIClient, basePath string, id uuid.UUID) (*T, error) {
	var account T
	if err := client.doJSON(ctx, http.MethodGet, servicePath(basePath, id), nil, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

// doServiceCreate performs a POST request to create a new account.
func doServiceCreate[T, R any](ctx context.Context, client *APIClient, basePath string, req R) (*T, error) {
	var account T
	if err := client.doJSON(ctx, http.MethodPost, basePath, req, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

// doServiceUpdate performs a PUT request to update an account by ID.
func doServiceUpdate[T, R any](ctx context.Context, client *APIClient, basePath string, id uuid.UUID, req R) (*T, error) {
	var account T
	if err := client.doJSON(ctx, http.MethodPut, servicePath(basePath, id), req, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

// doServiceDelete performs a DELETE request to remove an account by ID.
func doServiceDelete(ctx context.Context, client *APIClient, basePath string, id uuid.UUID) error {
	return client.doJSON(ctx, http.MethodDelete, servicePath(basePath, id), nil, nil)
}

// doServiceToggle performs a POST request to toggle the enabled state of an account.
func doServiceToggle(ctx context.Context, client *APIClient, basePath string, id uuid.UUID, enabled bool) error {
	return client.doJSON(ctx, http.MethodPost, servicePath(basePath, id)+"/toggle", toggleRequest{Enabled: enabled}, nil)
}

// --- Slack ---

// ListSlackAccounts issues GET /api/v1/services/slack and decodes the
// server's plain-array response into a slice of SlackAccount.
func (a *serviceConfigAdapter) ListSlackAccounts(ctx context.Context) ([]SlackAccount, error) {
	return doServiceList[SlackAccount](ctx, a.client, slackServicePath)
}

// GetSlackAccount issues GET /api/v1/services/slack/{id} and decodes the
// slackAccountItem payload.
func (a *serviceConfigAdapter) GetSlackAccount(ctx context.Context, id uuid.UUID) (*SlackAccount, error) {
	return doServiceGet[SlackAccount](ctx, a.client, slackServicePath, id)
}

// CreateSlackAccount issues POST /api/v1/services/slack with req as the
// JSON body (carrying the bot_token) and decodes the server's response
// (which omits bot_token).
func (a *serviceConfigAdapter) CreateSlackAccount(ctx context.Context, req CreateSlackAccountRequest) (*SlackAccount, error) {
	return doServiceCreate[SlackAccount](ctx, a.client, slackServicePath, req)
}

// UpdateSlackAccount issues PUT /api/v1/services/slack/{id} with req as the
// full replacement JSON body and decodes the updated slackAccountItem
// response.
func (a *serviceConfigAdapter) UpdateSlackAccount(ctx context.Context, id uuid.UUID, req UpdateSlackAccountRequest) (*SlackAccount, error) {
	return doServiceUpdate[SlackAccount](ctx, a.client, slackServicePath, id, req)
}

// DeleteSlackAccount issues DELETE /api/v1/services/slack/{id}. The server
// returns 204 No Content on success.
func (a *serviceConfigAdapter) DeleteSlackAccount(ctx context.Context, id uuid.UUID) error {
	return doServiceDelete(ctx, a.client, slackServicePath, id)
}

// ToggleSlackAccount issues POST /api/v1/services/slack/{id}/toggle with a
// {"enabled": enabled} body. The server's 200 response body is ignored.
func (a *serviceConfigAdapter) ToggleSlackAccount(ctx context.Context, id uuid.UUID, enabled bool) error {
	return doServiceToggle(ctx, a.client, slackServicePath, id, enabled)
}

// --- Email ---

// ListEmailAccounts issues GET /api/v1/services/email and decodes the
// server's plain-array response into a slice of EmailAccount.
func (a *serviceConfigAdapter) ListEmailAccounts(ctx context.Context) ([]EmailAccount, error) {
	return doServiceList[EmailAccount](ctx, a.client, emailServicePath)
}

// GetEmailAccount issues GET /api/v1/services/email/{id} and decodes the
// emailAccountItem payload.
func (a *serviceConfigAdapter) GetEmailAccount(ctx context.Context, id uuid.UUID) (*EmailAccount, error) {
	return doServiceGet[EmailAccount](ctx, a.client, emailServicePath, id)
}

// CreateEmailAccount issues POST /api/v1/services/email with req as the
// JSON body (carrying the password) and decodes the server's response
// (which omits password).
func (a *serviceConfigAdapter) CreateEmailAccount(ctx context.Context, req CreateEmailAccountRequest) (*EmailAccount, error) {
	return doServiceCreate[EmailAccount](ctx, a.client, emailServicePath, req)
}

// UpdateEmailAccount issues PUT /api/v1/services/email/{id} with req as
// the full replacement JSON body and decodes the updated emailAccountItem
// response.
func (a *serviceConfigAdapter) UpdateEmailAccount(ctx context.Context, id uuid.UUID, req UpdateEmailAccountRequest) (*EmailAccount, error) {
	return doServiceUpdate[EmailAccount](ctx, a.client, emailServicePath, id, req)
}

// DeleteEmailAccount issues DELETE /api/v1/services/email/{id}. The server
// returns 204 No Content on success.
func (a *serviceConfigAdapter) DeleteEmailAccount(ctx context.Context, id uuid.UUID) error {
	return doServiceDelete(ctx, a.client, emailServicePath, id)
}

// ToggleEmailAccount issues POST /api/v1/services/email/{id}/toggle with a
// {"enabled": enabled} body. The server's 200 response body is ignored.
func (a *serviceConfigAdapter) ToggleEmailAccount(ctx context.Context, id uuid.UUID, enabled bool) error {
	return doServiceToggle(ctx, a.client, emailServicePath, id, enabled)
}

// --- Calendar ---

// ListCalendarAccounts issues GET /api/v1/services/calendar and decodes
// the server's plain-array response into a slice of CalendarAccount.
func (a *serviceConfigAdapter) ListCalendarAccounts(ctx context.Context) ([]CalendarAccount, error) {
	return doServiceList[CalendarAccount](ctx, a.client, calendarServicePath)
}

// GetCalendarAccount issues GET /api/v1/services/calendar/{id} and decodes
// the calendarAccountItem payload.
func (a *serviceConfigAdapter) GetCalendarAccount(ctx context.Context, id uuid.UUID) (*CalendarAccount, error) {
	return doServiceGet[CalendarAccount](ctx, a.client, calendarServicePath, id)
}

// CreateCalendarAccount issues POST /api/v1/services/calendar with req as
// the JSON body and decodes the server's calendarAccountItem response.
func (a *serviceConfigAdapter) CreateCalendarAccount(ctx context.Context, req CreateCalendarAccountRequest) (*CalendarAccount, error) {
	return doServiceCreate[CalendarAccount](ctx, a.client, calendarServicePath, req)
}

// UpdateCalendarAccount issues PUT /api/v1/services/calendar/{id} with req
// as the full replacement JSON body and decodes the updated
// calendarAccountItem response.
func (a *serviceConfigAdapter) UpdateCalendarAccount(ctx context.Context, id uuid.UUID, req UpdateCalendarAccountRequest) (*CalendarAccount, error) {
	return doServiceUpdate[CalendarAccount](ctx, a.client, calendarServicePath, id, req)
}

// DeleteCalendarAccount issues DELETE /api/v1/services/calendar/{id}. The
// server returns 204 No Content on success.
func (a *serviceConfigAdapter) DeleteCalendarAccount(ctx context.Context, id uuid.UUID) error {
	return doServiceDelete(ctx, a.client, calendarServicePath, id)
}

// ToggleCalendarAccount issues POST /api/v1/services/calendar/{id}/toggle
// with a {"enabled": enabled} body. The server's 200 response body is
// ignored.
func (a *serviceConfigAdapter) ToggleCalendarAccount(ctx context.Context, id uuid.UUID, enabled bool) error {
	return doServiceToggle(ctx, a.client, calendarServicePath, id, enabled)
}

// --- Cross-service ---

// ServiceStatus issues GET /api/v1/services/status and decodes the
// {services, count} payload, returning just the services slice. The count
// is implicit in len(services) for callers.
func (a *serviceConfigAdapter) ServiceStatus(ctx context.Context) ([]ServiceStatus, error) {
	var out struct {
		Services []ServiceStatus `json:"services"`
		Count    int             `json:"count"`
	}
	if err := a.client.doJSON(ctx, http.MethodGet, servicesStatusPath, nil, &out); err != nil {
		return nil, err
	}
	return out.Services, nil
}
