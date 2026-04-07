package presenter

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// Default poll intervals (in seconds) per service type.
const (
	DefaultSlackPollInterval    = 60
	DefaultEmailPollInterval    = 600
	DefaultCalendarPollInterval = 600
)

// WatcherRemover is the minimal interface the presenter needs for managing watchers.
type WatcherRemover interface {
	RemoveWatcher(name string)
}

// WatcherFactory creates and registers a watcher for the given account type and ID.
type WatcherFactory func(accountType string, accountID uuid.UUID) error

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

// ServiceSettingsOption is a functional option for configuring ServiceSettingsPresenter.
type ServiceSettingsOption func(*ServiceSettingsPresenter)

// WithSlackValidator sets the Slack credential validator.
func WithSlackValidator(v SlackValidator) ServiceSettingsOption {
	return func(p *ServiceSettingsPresenter) {
		p.slackValidator = v
	}
}

// WithEmailValidator sets the Email credential validator.
func WithEmailValidator(v EmailValidator) ServiceSettingsOption {
	return func(p *ServiceSettingsPresenter) {
		p.emailValidator = v
	}
}

// WithCalendarValidator sets the Calendar credential validator.
func WithCalendarValidator(v CalendarValidator) ServiceSettingsOption {
	return func(p *ServiceSettingsPresenter) {
		p.calendarValidator = v
	}
}

// ServiceSettingsPresenter mediates between the UI and the service config repository.
type ServiceSettingsPresenter struct {
	repo              repository.ServiceConfigRepository
	mgr               WatcherRemover
	factory           WatcherFactory
	slackValidator    SlackValidator
	emailValidator    EmailValidator
	calendarValidator CalendarValidator
}

// NewServiceSettingsPresenter constructs a ServiceSettingsPresenter.
func NewServiceSettingsPresenter(repo repository.ServiceConfigRepository, mgr WatcherRemover, factory WatcherFactory, opts ...ServiceSettingsOption) *ServiceSettingsPresenter {
	p := &ServiceSettingsPresenter{
		repo:    repo,
		mgr:     mgr,
		factory: factory,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// ListSlackAccounts returns all configured Slack accounts.
func (p *ServiceSettingsPresenter) ListSlackAccounts(ctx context.Context) ([]*repository.SlackAccount, error) {
	return p.repo.ListSlackAccounts(ctx)
}

// ListEmailAccounts returns all configured email accounts.
func (p *ServiceSettingsPresenter) ListEmailAccounts(ctx context.Context) ([]*repository.EmailAccount, error) {
	return p.repo.ListEmailAccounts(ctx)
}

// SaveSlackAccount validates and persists a new Slack account, then starts its watcher.
func (p *ServiceSettingsPresenter) SaveSlackAccount(ctx context.Context, acct *repository.SlackAccount) error {
	if acct.PollIntervalSeconds == 0 {
		acct.PollIntervalSeconds = DefaultSlackPollInterval
	}
	if err := validateSlackAccount(acct); err != nil {
		return err
	}
	if p.slackValidator != nil {
		if err := p.slackValidator.ValidateSlack(ctx, acct.Token); err != nil {
			return fmt.Errorf("slack credential validation failed: %w", err)
		}
	}
	if err := p.repo.UpsertSlackAccount(ctx, acct); err != nil {
		return fmt.Errorf("saving slack account: %w", err)
	}
	if err := p.factory("slack", acct.ID); err != nil {
		return fmt.Errorf("creating slack watcher: %w", err)
	}
	return nil
}

// SaveEmailAccount validates and persists a new email account, then starts its watcher.
func (p *ServiceSettingsPresenter) SaveEmailAccount(ctx context.Context, acct *repository.EmailAccount) error {
	if acct.PollIntervalSeconds == 0 {
		acct.PollIntervalSeconds = DefaultEmailPollInterval
	}
	if err := validateEmailAccount(acct); err != nil {
		return err
	}
	if p.emailValidator != nil {
		if err := p.emailValidator.ValidateEmail(ctx, acct.IMAPHost, acct.IMAPPort, acct.Username, acct.Password, acct.Encryption); err != nil {
			return fmt.Errorf("email credential validation failed: %w", err)
		}
	}
	if err := p.repo.UpsertEmailAccount(ctx, acct); err != nil {
		return fmt.Errorf("saving email account: %w", err)
	}
	if err := p.factory("email", acct.ID); err != nil {
		return fmt.Errorf("creating email watcher: %w", err)
	}
	return nil
}

// EditSlackAccount persists changes, removes the old watcher, and starts a new one.
func (p *ServiceSettingsPresenter) EditSlackAccount(ctx context.Context, acct *repository.SlackAccount, oldWorkspaceID string) error {
	if p.slackValidator != nil {
		if err := p.slackValidator.ValidateSlack(ctx, acct.Token); err != nil {
			return fmt.Errorf("slack credential validation failed: %w", err)
		}
	}
	if err := p.repo.UpsertSlackAccount(ctx, acct); err != nil {
		return fmt.Errorf("updating slack account: %w", err)
	}
	p.mgr.RemoveWatcher(slackWatcherName(oldWorkspaceID))
	if err := p.factory("slack", acct.ID); err != nil {
		return fmt.Errorf("creating slack watcher: %w", err)
	}
	return nil
}

// EditEmailAccount persists changes, removes the old watcher, and starts a new one.
func (p *ServiceSettingsPresenter) EditEmailAccount(ctx context.Context, acct *repository.EmailAccount, oldUsername string) error {
	if p.emailValidator != nil {
		if err := p.emailValidator.ValidateEmail(ctx, acct.IMAPHost, acct.IMAPPort, acct.Username, acct.Password, acct.Encryption); err != nil {
			return fmt.Errorf("email credential validation failed: %w", err)
		}
	}
	if err := p.repo.UpsertEmailAccount(ctx, acct); err != nil {
		return fmt.Errorf("updating email account: %w", err)
	}
	p.mgr.RemoveWatcher(emailWatcherName(oldUsername))
	if err := p.factory("email", acct.ID); err != nil {
		return fmt.Errorf("creating email watcher: %w", err)
	}
	return nil
}

// DeleteSlackAccount removes the watcher and deletes the account from the repository.
func (p *ServiceSettingsPresenter) DeleteSlackAccount(ctx context.Context, id uuid.UUID) error {
	acct, err := p.repo.GetSlackAccount(ctx, id)
	if err != nil {
		return fmt.Errorf("getting slack account for delete: %w", err)
	}
	p.mgr.RemoveWatcher(slackWatcherName(acct.WorkspaceID))
	if err := p.repo.DeleteSlackAccount(ctx, id); err != nil {
		return fmt.Errorf("deleting slack account: %w", err)
	}
	return nil
}

// DeleteEmailAccount removes the watcher and deletes the account from the repository.
func (p *ServiceSettingsPresenter) DeleteEmailAccount(ctx context.Context, id uuid.UUID) error {
	acct, err := p.repo.GetEmailAccount(ctx, id)
	if err != nil {
		return fmt.Errorf("getting email account for delete: %w", err)
	}
	p.mgr.RemoveWatcher(emailWatcherName(acct.Username))
	if err := p.repo.DeleteEmailAccount(ctx, id); err != nil {
		return fmt.Errorf("deleting email account: %w", err)
	}
	return nil
}

// ToggleSlackAccount enables or disables a Slack account, starting or stopping its watcher.
func (p *ServiceSettingsPresenter) ToggleSlackAccount(ctx context.Context, id uuid.UUID, enabled bool) error {
	acct, err := p.repo.GetSlackAccount(ctx, id)
	if err != nil {
		return fmt.Errorf("getting slack account for toggle: %w", err)
	}
	acct.Enabled = enabled
	if err := p.repo.UpsertSlackAccount(ctx, acct); err != nil {
		return fmt.Errorf("updating slack account: %w", err)
	}
	if enabled {
		if err := p.factory("slack", acct.ID); err != nil {
			return fmt.Errorf("creating slack watcher: %w", err)
		}
	} else {
		p.mgr.RemoveWatcher(slackWatcherName(acct.WorkspaceID))
	}
	return nil
}

// ToggleEmailAccount enables or disables an email account, starting or stopping its watcher.
func (p *ServiceSettingsPresenter) ToggleEmailAccount(ctx context.Context, id uuid.UUID, enabled bool) error {
	acct, err := p.repo.GetEmailAccount(ctx, id)
	if err != nil {
		return fmt.Errorf("getting email account for toggle: %w", err)
	}
	acct.Enabled = enabled
	if err := p.repo.UpsertEmailAccount(ctx, acct); err != nil {
		return fmt.Errorf("updating email account: %w", err)
	}
	if enabled {
		if err := p.factory("email", acct.ID); err != nil {
			return fmt.Errorf("creating email watcher: %w", err)
		}
	} else {
		p.mgr.RemoveWatcher(emailWatcherName(acct.Username))
	}
	return nil
}

// ListCalendarAccounts returns all configured calendar accounts.
func (p *ServiceSettingsPresenter) ListCalendarAccounts(ctx context.Context) ([]*repository.CalendarAccount, error) {
	return p.repo.ListCalendarAccounts(ctx)
}

// SaveCalendarAccount validates and persists a new calendar account.
func (p *ServiceSettingsPresenter) SaveCalendarAccount(ctx context.Context, acct *repository.CalendarAccount) error {
	if acct.PollIntervalSeconds == 0 {
		acct.PollIntervalSeconds = DefaultCalendarPollInterval
	}
	if p.calendarValidator != nil {
		if err := p.calendarValidator.ValidateCalendar(ctx, acct.ICSURL); err != nil {
			return fmt.Errorf("calendar credential validation failed: %w", err)
		}
	}
	if err := p.repo.UpsertCalendarAccount(ctx, acct); err != nil {
		return fmt.Errorf("saving calendar account: %w", err)
	}
	return nil
}

// EditCalendarAccount persists changes to a calendar account.
func (p *ServiceSettingsPresenter) EditCalendarAccount(ctx context.Context, acct *repository.CalendarAccount, oldName string) error {
	if p.calendarValidator != nil {
		if err := p.calendarValidator.ValidateCalendar(ctx, acct.ICSURL); err != nil {
			return fmt.Errorf("calendar credential validation failed: %w", err)
		}
	}
	if err := p.repo.UpsertCalendarAccount(ctx, acct); err != nil {
		return fmt.Errorf("updating calendar account: %w", err)
	}
	return nil
}

// DeleteCalendarAccount deletes a calendar account from the repository.
func (p *ServiceSettingsPresenter) DeleteCalendarAccount(ctx context.Context, id uuid.UUID) error {
	_, err := p.repo.GetCalendarAccount(ctx, id)
	if err != nil {
		return fmt.Errorf("getting calendar account for delete: %w", err)
	}
	if err := p.repo.DeleteCalendarAccount(ctx, id); err != nil {
		return fmt.Errorf("deleting calendar account: %w", err)
	}
	return nil
}

// ToggleCalendarAccount enables or disables a calendar account.
func (p *ServiceSettingsPresenter) ToggleCalendarAccount(ctx context.Context, id uuid.UUID, enabled bool) error {
	acct, err := p.repo.GetCalendarAccount(ctx, id)
	if err != nil {
		return fmt.Errorf("getting calendar account for toggle: %w", err)
	}
	acct.Enabled = enabled
	if err := p.repo.UpsertCalendarAccount(ctx, acct); err != nil {
		return fmt.Errorf("updating calendar account: %w", err)
	}
	return nil
}

func slackWatcherName(workspaceID string) string { return "slack:" + workspaceID }
func emailWatcherName(username string) string    { return "email:" + username }

func validateSlackAccount(acct *repository.SlackAccount) error {
	if acct.Token == "" {
		return fmt.Errorf("token is required")
	}
	if acct.WorkspaceID == "" {
		return fmt.Errorf("workspace ID is required")
	}
	if acct.PollIntervalSeconds <= 0 {
		return fmt.Errorf("poll interval must be positive")
	}
	return nil
}

func validateEmailAccount(acct *repository.EmailAccount) error {
	if acct.IMAPHost == "" {
		return fmt.Errorf("IMAP host is required")
	}
	if acct.IMAPPort <= 0 {
		return fmt.Errorf("IMAP port must be positive")
	}
	if acct.Username == "" {
		return fmt.Errorf("username is required")
	}
	if acct.Password == "" {
		return fmt.Errorf("password is required")
	}
	if acct.PollIntervalSeconds <= 0 {
		return fmt.Errorf("poll interval must be positive")
	}
	return nil
}

// DefaultPollInterval returns the default poll interval in seconds for the given service type.
func DefaultPollInterval(serviceType string) int {
	return 0
}
