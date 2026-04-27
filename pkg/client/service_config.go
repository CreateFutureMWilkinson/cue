package client

import (
	"context"

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
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	WorkspaceID string    `json:"workspace_id"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   string    `json:"created_at"`
}

// CreateSlackAccountRequest is the POST body for creating a Slack account via
// POST /api/v1/services/slack. The bot_token is carried here but omitted
// from the response DTO; the server stores it and references it from the
// watcher registration.
type CreateSlackAccountRequest struct {
	Name        string `json:"name"`
	BotToken    string `json:"bot_token"`
	WorkspaceID string `json:"workspace_id"`
	Enabled     bool   `json:"enabled"`
}

// UpdateSlackAccountRequest is the PUT body for full replacement via
// PUT /api/v1/services/slack/{id}. It is structurally identical to
// CreateSlackAccountRequest so we alias the type.
type UpdateSlackAccountRequest = CreateSlackAccountRequest

// EmailAccount mirrors the server's emailAccountItem DTO returned by
// /api/v1/services/email routes. The password is NEVER returned on
// responses — only accepted on create/update request bodies.
type EmailAccount struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	IMAPHost   string    `json:"imap_host"`
	IMAPPort   int       `json:"imap_port"`
	Username   string    `json:"username"`
	Encryption string    `json:"encryption"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  string    `json:"created_at"`
}

// CreateEmailAccountRequest is the POST body for creating an Email account
// via POST /api/v1/services/email. The password is carried here but omitted
// from the response DTO.
type CreateEmailAccountRequest struct {
	Name       string `json:"name"`
	IMAPHost   string `json:"imap_host"`
	IMAPPort   int    `json:"imap_port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Encryption string `json:"encryption"`
	Enabled    bool   `json:"enabled"`
}

// UpdateEmailAccountRequest is the PUT body for full replacement via
// PUT /api/v1/services/email/{id}. Structurally identical to
// CreateEmailAccountRequest.
type UpdateEmailAccountRequest = CreateEmailAccountRequest

// CalendarAccount mirrors the server's calendarAccountItem DTO returned by
// /api/v1/services/calendar routes.
type CalendarAccount struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	ICSURL    string    `json:"ics_url"`
	Enabled   bool      `json:"enabled"`
	CreatedAt string    `json:"created_at"`
}

// CreateCalendarAccountRequest is the POST body for creating a Calendar
// account via POST /api/v1/services/calendar.
type CreateCalendarAccountRequest struct {
	Name    string `json:"name"`
	ICSURL  string `json:"ics_url"`
	Enabled bool   `json:"enabled"`
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

// --- Slack stubs ---

// ListSlackAccounts is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) ListSlackAccounts(ctx context.Context) ([]SlackAccount, error) {
	return nil, ErrNotImplemented
}

// GetSlackAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) GetSlackAccount(ctx context.Context, id uuid.UUID) (*SlackAccount, error) {
	return nil, ErrNotImplemented
}

// CreateSlackAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) CreateSlackAccount(ctx context.Context, req CreateSlackAccountRequest) (*SlackAccount, error) {
	return nil, ErrNotImplemented
}

// UpdateSlackAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) UpdateSlackAccount(ctx context.Context, id uuid.UUID, req UpdateSlackAccountRequest) (*SlackAccount, error) {
	return nil, ErrNotImplemented
}

// DeleteSlackAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) DeleteSlackAccount(ctx context.Context, id uuid.UUID) error {
	return ErrNotImplemented
}

// ToggleSlackAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) ToggleSlackAccount(ctx context.Context, id uuid.UUID, enabled bool) error {
	return ErrNotImplemented
}

// --- Email stubs ---

// ListEmailAccounts is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) ListEmailAccounts(ctx context.Context) ([]EmailAccount, error) {
	return nil, ErrNotImplemented
}

// GetEmailAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) GetEmailAccount(ctx context.Context, id uuid.UUID) (*EmailAccount, error) {
	return nil, ErrNotImplemented
}

// CreateEmailAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) CreateEmailAccount(ctx context.Context, req CreateEmailAccountRequest) (*EmailAccount, error) {
	return nil, ErrNotImplemented
}

// UpdateEmailAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) UpdateEmailAccount(ctx context.Context, id uuid.UUID, req UpdateEmailAccountRequest) (*EmailAccount, error) {
	return nil, ErrNotImplemented
}

// DeleteEmailAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) DeleteEmailAccount(ctx context.Context, id uuid.UUID) error {
	return ErrNotImplemented
}

// ToggleEmailAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) ToggleEmailAccount(ctx context.Context, id uuid.UUID, enabled bool) error {
	return ErrNotImplemented
}

// --- Calendar stubs ---

// ListCalendarAccounts is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) ListCalendarAccounts(ctx context.Context) ([]CalendarAccount, error) {
	return nil, ErrNotImplemented
}

// GetCalendarAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) GetCalendarAccount(ctx context.Context, id uuid.UUID) (*CalendarAccount, error) {
	return nil, ErrNotImplemented
}

// CreateCalendarAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) CreateCalendarAccount(ctx context.Context, req CreateCalendarAccountRequest) (*CalendarAccount, error) {
	return nil, ErrNotImplemented
}

// UpdateCalendarAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) UpdateCalendarAccount(ctx context.Context, id uuid.UUID, req UpdateCalendarAccountRequest) (*CalendarAccount, error) {
	return nil, ErrNotImplemented
}

// DeleteCalendarAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) DeleteCalendarAccount(ctx context.Context, id uuid.UUID) error {
	return ErrNotImplemented
}

// ToggleCalendarAccount is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) ToggleCalendarAccount(ctx context.Context, id uuid.UUID, enabled bool) error {
	return ErrNotImplemented
}

// --- Cross-service stubs ---

// ServiceStatus is a stub returning ErrNotImplemented.
func (a *serviceConfigAdapter) ServiceStatus(ctx context.Context) ([]ServiceStatus, error) {
	return nil, ErrNotImplemented
}
