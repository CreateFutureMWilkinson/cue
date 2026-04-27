package servicemanager

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/orchestrator"
)

// ErrNotImplemented is returned by stubs that have not yet been replaced with real logic.
var ErrNotImplemented = errors.New("not implemented")

// ServiceConfigRepo defines the subset of repository.ServiceConfigRepository used by ServiceManager
// for managing service account configurations across Slack, Email, and Calendar providers.
type ServiceConfigRepo interface {
	ListSlackAccounts(ctx context.Context) ([]*repository.SlackAccount, error)
	GetSlackAccount(ctx context.Context, id uuid.UUID) (*repository.SlackAccount, error)
	UpsertSlackAccount(ctx context.Context, acct *repository.SlackAccount) error
	DeleteSlackAccount(ctx context.Context, id uuid.UUID) error
	ListEmailAccounts(ctx context.Context) ([]*repository.EmailAccount, error)
	GetEmailAccount(ctx context.Context, id uuid.UUID) (*repository.EmailAccount, error)
	UpsertEmailAccount(ctx context.Context, acct *repository.EmailAccount) error
	DeleteEmailAccount(ctx context.Context, id uuid.UUID) error
	ListCalendarAccounts(ctx context.Context) ([]*repository.CalendarAccount, error)
	GetCalendarAccount(ctx context.Context, id uuid.UUID) (*repository.CalendarAccount, error)
	UpsertCalendarAccount(ctx context.Context, acct *repository.CalendarAccount) error
	DeleteCalendarAccount(ctx context.Context, id uuid.UUID) error
}

// WatcherLifecycle defines the subset of orchestrator.Orchestrator used for watcher
// lifecycle management - adding, removing, and listing active watchers.
type WatcherLifecycle interface {
	AddWatcher(name string, w orchestrator.Watcher)
	RemoveWatcher(name string)
	ListWatcherNames() []string
}

// WatcherFactory creates and registers a watcher for the given account type and ID.
type WatcherFactory func(accountType string, accountID uuid.UUID) error

// MessageDeleter deletes messages associated with a source account, providing
// cascade deletion when service accounts are removed.
type MessageDeleter interface {
	DeleteBySourceAccount(ctx context.Context, source, sourceAccount string) (int64, error)
}

// SlackValidator validates Slack credentials before saving.
type SlackValidator interface {
	ValidateSlack(ctx context.Context, token string) error
}

// EmailValidator validates IMAP credentials before saving.
type EmailValidator interface {
	ValidateEmail(ctx context.Context, host string, port int, username, password, encryption string) error
}

// CalendarValidator validates a calendar ICS URL before saving.
type CalendarValidator interface {
	ValidateCalendar(ctx context.Context, url string) error
}

// Option is a functional option for configuring ServiceManager.
type Option func(*ServiceManager)

// WithSlackValidator sets the Slack credential validator.
func WithSlackValidator(v SlackValidator) Option {
	return func(m *ServiceManager) {
		m.slackValidator = v
	}
}

// WithEmailValidator sets the Email credential validator.
func WithEmailValidator(v EmailValidator) Option {
	return func(m *ServiceManager) {
		m.emailValidator = v
	}
}

// WithCalendarValidator sets the Calendar credential validator.
func WithCalendarValidator(v CalendarValidator) Option {
	return func(m *ServiceManager) {
		m.calendarValidator = v
	}
}

// Default poll intervals (in seconds) per service type.
const (
	DefaultSlackPollInterval    = 60
	DefaultEmailPollInterval    = 600
	DefaultCalendarPollInterval = 600
)

// ServiceManager coordinates service configuration CRUD with watcher lifecycle management.
type ServiceManager struct {
	repo              ServiceConfigRepo
	watchers          WatcherLifecycle
	watcherFactory    WatcherFactory
	messageDeleter    MessageDeleter
	slackValidator    SlackValidator
	emailValidator    EmailValidator
	calendarValidator CalendarValidator
}

// NewServiceManager creates a ServiceManager with the given dependencies.
// Returns an error if any dependency is nil.
func NewServiceManager(repo ServiceConfigRepo, watchers WatcherLifecycle, factory WatcherFactory, messageDeleter MessageDeleter, opts ...Option) (*ServiceManager, error) {
	if repo == nil {
		return nil, fmt.Errorf("service manager: repository must not be nil")
	}
	if watchers == nil {
		return nil, fmt.Errorf("service manager: watcher lifecycle must not be nil")
	}
	if factory == nil {
		return nil, fmt.Errorf("service manager: watcher factory must not be nil")
	}
	if messageDeleter == nil {
		return nil, fmt.Errorf("service manager: message deleter must not be nil")
	}
	m := &ServiceManager{
		repo:           repo,
		watchers:       watchers,
		watcherFactory: factory,
		messageDeleter: messageDeleter,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m, nil
}

// ListSlackAccounts returns all configured Slack accounts from the repository.
func (m *ServiceManager) ListSlackAccounts(ctx context.Context) ([]*repository.SlackAccount, error) {
	return m.repo.ListSlackAccounts(ctx)
}

// ListEmailAccounts returns all configured Email accounts from the repository.
func (m *ServiceManager) ListEmailAccounts(ctx context.Context) ([]*repository.EmailAccount, error) {
	return m.repo.ListEmailAccounts(ctx)
}

// ListCalendarAccounts returns all configured Calendar accounts from the repository.
func (m *ServiceManager) ListCalendarAccounts(ctx context.Context) ([]*repository.CalendarAccount, error) {
	return m.repo.ListCalendarAccounts(ctx)
}

// CredentialMask is the placeholder used in place of sensitive fields in API responses.
const CredentialMask = "***"

// GetSlackAccount retrieves a Slack account by ID with credentials masked.
func (m *ServiceManager) GetSlackAccount(ctx context.Context, id uuid.UUID) (*repository.SlackAccount, error) {
	acct, err := m.repo.GetSlackAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	acct.Token = CredentialMask
	return acct, nil
}

// GetEmailAccount retrieves an Email account by ID with credentials masked.
func (m *ServiceManager) GetEmailAccount(ctx context.Context, id uuid.UUID) (*repository.EmailAccount, error) {
	acct, err := m.repo.GetEmailAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	acct.Password = CredentialMask
	return acct, nil
}

// GetCalendarAccount retrieves a Calendar account by ID.
func (m *ServiceManager) GetCalendarAccount(ctx context.Context, id uuid.UUID) (*repository.CalendarAccount, error) {
	return m.repo.GetCalendarAccount(ctx, id)
}

// CreateSlackAccount validates, persists, and registers a watcher for a new Slack account.
// Returns the account with credentials masked.
func (m *ServiceManager) CreateSlackAccount(ctx context.Context, acct *repository.SlackAccount) (*repository.SlackAccount, error) {
	if acct.Token == "" {
		return nil, fmt.Errorf("create slack account: token must not be empty")
	}
	if acct.WorkspaceID == "" {
		return nil, fmt.Errorf("create slack account: workspace ID must not be empty")
	}

	if acct.PollIntervalSeconds == 0 {
		acct.PollIntervalSeconds = DefaultSlackPollInterval
	}

	if acct.ID == uuid.Nil {
		acct.ID = uuid.New()
	}

	if m.slackValidator != nil {
		if err := m.slackValidator.ValidateSlack(ctx, acct.Token); err != nil {
			return nil, fmt.Errorf("create slack account: validation failed: %w", err)
		}
	}

	if err := m.repo.UpsertSlackAccount(ctx, acct); err != nil {
		return nil, fmt.Errorf("create slack account: %w", err)
	}

	if err := m.watcherFactory("slack", acct.ID); err != nil {
		return nil, fmt.Errorf("create slack account: watcher factory: %w", err)
	}

	result := *acct
	result.Token = CredentialMask
	return &result, nil
}

// CreateEmailAccount validates, persists, and registers a watcher for a new Email account.
// Returns the account with credentials masked.
func (m *ServiceManager) CreateEmailAccount(ctx context.Context, acct *repository.EmailAccount) (*repository.EmailAccount, error) {
	if acct.IMAPHost == "" {
		return nil, fmt.Errorf("create email account: host must not be empty")
	}
	if acct.IMAPPort <= 0 {
		return nil, fmt.Errorf("create email account: port must be greater than zero")
	}
	if acct.Username == "" {
		return nil, fmt.Errorf("create email account: username must not be empty")
	}
	if acct.Password == "" {
		return nil, fmt.Errorf("create email account: password must not be empty")
	}

	if acct.PollIntervalSeconds == 0 {
		acct.PollIntervalSeconds = DefaultEmailPollInterval
	}

	if acct.ID == uuid.Nil {
		acct.ID = uuid.New()
	}

	if m.emailValidator != nil {
		if err := m.emailValidator.ValidateEmail(ctx, acct.IMAPHost, acct.IMAPPort, acct.Username, acct.Password, acct.Encryption); err != nil {
			return nil, fmt.Errorf("create email account: validation failed: %w", err)
		}
	}

	if err := m.repo.UpsertEmailAccount(ctx, acct); err != nil {
		return nil, fmt.Errorf("create email account: %w", err)
	}

	if err := m.watcherFactory("email", acct.ID); err != nil {
		return nil, fmt.Errorf("create email account: watcher factory: %w", err)
	}

	result := *acct
	result.Password = CredentialMask
	return &result, nil
}

// CreateCalendarAccount validates, persists, and returns a new Calendar account.
// Calendar accounts do not have watchers registered via the factory.
func (m *ServiceManager) CreateCalendarAccount(ctx context.Context, acct *repository.CalendarAccount) (*repository.CalendarAccount, error) {
	if acct.Name == "" {
		return nil, fmt.Errorf("create calendar account: name must not be empty")
	}
	if acct.ICSURL == "" {
		return nil, fmt.Errorf("create calendar account: ICS URL must not be empty")
	}

	if acct.PollIntervalSeconds == 0 {
		acct.PollIntervalSeconds = DefaultCalendarPollInterval
	}

	if acct.ID == uuid.Nil {
		acct.ID = uuid.New()
	}

	if m.calendarValidator != nil {
		if err := m.calendarValidator.ValidateCalendar(ctx, acct.ICSURL); err != nil {
			return nil, fmt.Errorf("create calendar account: validation failed: %w", err)
		}
	}

	if err := m.repo.UpsertCalendarAccount(ctx, acct); err != nil {
		return nil, fmt.Errorf("create calendar account: %w", err)
	}

	result := *acct
	return &result, nil
}

// DeleteSlackAccount removes a Slack account and its associated resources:
// watcher, cascade-deleted messages, and the account record itself.
func (m *ServiceManager) DeleteSlackAccount(ctx context.Context, id uuid.UUID) error {
	acct, err := m.repo.GetSlackAccount(ctx, id)
	if err != nil {
		return err
	}

	m.watchers.RemoveWatcher("slack:" + acct.WorkspaceID)

	if _, err := m.messageDeleter.DeleteBySourceAccount(ctx, "slack", acct.WorkspaceID); err != nil {
		return fmt.Errorf("delete slack account: delete messages: %w", err)
	}

	if err := m.repo.DeleteSlackAccount(ctx, id); err != nil {
		return fmt.Errorf("delete slack account: %w", err)
	}

	return nil
}

// DeleteEmailAccount removes an Email account and its associated resources:
// watcher, cascade-deleted messages, and the account record itself.
func (m *ServiceManager) DeleteEmailAccount(ctx context.Context, id uuid.UUID) error {
	acct, err := m.repo.GetEmailAccount(ctx, id)
	if err != nil {
		return err
	}

	m.watchers.RemoveWatcher("email:" + acct.Username)

	if _, err := m.messageDeleter.DeleteBySourceAccount(ctx, "email", acct.Username); err != nil {
		return fmt.Errorf("delete email account: delete messages: %w", err)
	}

	if err := m.repo.DeleteEmailAccount(ctx, id); err != nil {
		return fmt.Errorf("delete email account: %w", err)
	}

	return nil
}

// DeleteCalendarAccount removes a Calendar account. Calendar accounts have no
// associated watchers or messages, so only the account record is deleted.
func (m *ServiceManager) DeleteCalendarAccount(ctx context.Context, id uuid.UUID) error {
	if _, err := m.repo.GetCalendarAccount(ctx, id); err != nil {
		return err
	}

	if err := m.repo.DeleteCalendarAccount(ctx, id); err != nil {
		return fmt.Errorf("delete calendar account: %w", err)
	}

	return nil
}

// UpdateSlackAccount fetches the existing account, merges non-zero fields from the input,
// preserves credentials when empty or masked, validates, upserts, and re-registers the watcher.
// Returns the updated account with credentials masked.
func (m *ServiceManager) UpdateSlackAccount(ctx context.Context, id uuid.UUID, acct *repository.SlackAccount) (*repository.SlackAccount, error) {
	existing, err := m.repo.GetSlackAccount(ctx, id)
	if err != nil {
		return nil, err
	}

	// Credential preservation: keep existing token if input is empty or masked.
	if acct.Token == "" || acct.Token == CredentialMask {
		acct.Token = existing.Token
	}

	// Merge non-zero fields onto existing.
	merged := *existing
	if acct.Token != "" {
		merged.Token = acct.Token
	}
	if acct.WorkspaceID != "" {
		merged.WorkspaceID = acct.WorkspaceID
	}
	if acct.FriendlyName != "" {
		merged.FriendlyName = acct.FriendlyName
	}
	if acct.Username != "" {
		merged.Username = acct.Username
	}
	if acct.WebURL != "" {
		merged.WebURL = acct.WebURL
	}
	if acct.PollIntervalSeconds != 0 {
		merged.PollIntervalSeconds = acct.PollIntervalSeconds
	}
	merged.Enabled = acct.Enabled

	// Validate merged result (same rules as Create).
	if merged.Token == "" {
		return nil, fmt.Errorf("update slack account: token must not be empty")
	}
	if merged.WorkspaceID == "" {
		return nil, fmt.Errorf("update slack account: workspace ID must not be empty")
	}

	if m.slackValidator != nil {
		if err := m.slackValidator.ValidateSlack(ctx, merged.Token); err != nil {
			return nil, fmt.Errorf("update slack account: validation failed: %w", err)
		}
	}

	if err := m.repo.UpsertSlackAccount(ctx, &merged); err != nil {
		return nil, err
	}

	// Remove old watcher, register new one.
	m.watchers.RemoveWatcher("slack:" + existing.WorkspaceID)
	if err := m.watcherFactory("slack", merged.ID); err != nil {
		return nil, fmt.Errorf("update slack account: watcher factory: %w", err)
	}

	result := merged
	result.Token = CredentialMask
	return &result, nil
}

// UpdateEmailAccount fetches the existing account, merges non-zero fields from the input,
// preserves credentials when empty or masked, validates, upserts, and re-registers the watcher.
// Returns the updated account with credentials masked.
func (m *ServiceManager) UpdateEmailAccount(ctx context.Context, id uuid.UUID, acct *repository.EmailAccount) (*repository.EmailAccount, error) {
	existing, err := m.repo.GetEmailAccount(ctx, id)
	if err != nil {
		return nil, err
	}

	// Credential preservation: keep existing password if input is empty or masked.
	if acct.Password == "" || acct.Password == CredentialMask {
		acct.Password = existing.Password
	}

	// Merge non-zero fields onto existing.
	merged := *existing
	if acct.IMAPHost != "" {
		merged.IMAPHost = acct.IMAPHost
	}
	if acct.IMAPPort != 0 {
		merged.IMAPPort = acct.IMAPPort
	}
	if acct.Username != "" {
		merged.Username = acct.Username
	}
	if acct.Password != "" {
		merged.Password = acct.Password
	}
	if acct.Encryption != "" {
		merged.Encryption = acct.Encryption
	}
	if acct.FriendlyName != "" {
		merged.FriendlyName = acct.FriendlyName
	}
	if acct.WebURL != "" {
		merged.WebURL = acct.WebURL
	}
	if acct.PollIntervalSeconds != 0 {
		merged.PollIntervalSeconds = acct.PollIntervalSeconds
	}
	merged.Enabled = acct.Enabled

	// Validate merged result (same rules as Create).
	if merged.IMAPHost == "" {
		return nil, fmt.Errorf("update email account: host must not be empty")
	}
	if merged.IMAPPort <= 0 {
		return nil, fmt.Errorf("update email account: port must be greater than zero")
	}
	if merged.Username == "" {
		return nil, fmt.Errorf("update email account: username must not be empty")
	}
	if merged.Password == "" {
		return nil, fmt.Errorf("update email account: password must not be empty")
	}

	if m.emailValidator != nil {
		if err := m.emailValidator.ValidateEmail(ctx, merged.IMAPHost, merged.IMAPPort, merged.Username, merged.Password, merged.Encryption); err != nil {
			return nil, fmt.Errorf("update email account: validation failed: %w", err)
		}
	}

	if err := m.repo.UpsertEmailAccount(ctx, &merged); err != nil {
		return nil, err
	}

	// Remove old watcher, register new one.
	m.watchers.RemoveWatcher("email:" + existing.Username)
	if err := m.watcherFactory("email", merged.ID); err != nil {
		return nil, fmt.Errorf("update email account: watcher factory: %w", err)
	}

	result := merged
	result.Password = CredentialMask
	return &result, nil
}

// UpdateCalendarAccount fetches the existing account, merges non-zero fields from the input,
// validates, and upserts. Calendar accounts have no watcher lifecycle.
// Returns the updated account.
func (m *ServiceManager) UpdateCalendarAccount(ctx context.Context, id uuid.UUID, acct *repository.CalendarAccount) (*repository.CalendarAccount, error) {
	existing, err := m.repo.GetCalendarAccount(ctx, id)
	if err != nil {
		return nil, err
	}

	// Merge non-zero fields onto existing.
	merged := *existing
	if acct.Name != "" {
		merged.Name = acct.Name
	}
	if acct.ICSURL != "" {
		merged.ICSURL = acct.ICSURL
	}
	if acct.PollIntervalSeconds != 0 {
		merged.PollIntervalSeconds = acct.PollIntervalSeconds
	}
	merged.Enabled = acct.Enabled

	// Validate merged result (same rules as Create).
	if merged.Name == "" {
		return nil, fmt.Errorf("update calendar account: name must not be empty")
	}
	if merged.ICSURL == "" {
		return nil, fmt.Errorf("update calendar account: ICS URL must not be empty")
	}

	if m.calendarValidator != nil {
		if err := m.calendarValidator.ValidateCalendar(ctx, merged.ICSURL); err != nil {
			return nil, fmt.Errorf("update calendar account: validation failed: %w", err)
		}
	}

	if err := m.repo.UpsertCalendarAccount(ctx, &merged); err != nil {
		return nil, err
	}

	result := merged
	return &result, nil
}
