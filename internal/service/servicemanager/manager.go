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

// ServiceManager coordinates service configuration CRUD with watcher lifecycle management.
type ServiceManager struct {
	repo           ServiceConfigRepo
	watchers       WatcherLifecycle
	watcherFactory WatcherFactory
	messageDeleter MessageDeleter
}

// NewServiceManager creates a ServiceManager with the given dependencies.
// Returns an error if any dependency is nil.
func NewServiceManager(repo ServiceConfigRepo, watchers WatcherLifecycle, factory WatcherFactory, messageDeleter MessageDeleter) (*ServiceManager, error) {
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
	return &ServiceManager{
		repo:           repo,
		watchers:       watchers,
		watcherFactory: factory,
		messageDeleter: messageDeleter,
	}, nil
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
