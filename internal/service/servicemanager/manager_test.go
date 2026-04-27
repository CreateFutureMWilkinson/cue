package servicemanager_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/orchestrator"
	"github.com/CreateFutureMWilkinson/cue/internal/service/servicemanager"
)

// --- Test Mocks ---

type mockRepo struct {
	slackAccounts    []*repository.SlackAccount
	slackErr         error
	emailAccounts    []*repository.EmailAccount
	emailErr         error
	calendarAccounts []*repository.CalendarAccount
	calendarErr      error
}

func (m *mockRepo) ListSlackAccounts(_ context.Context) ([]*repository.SlackAccount, error) {
	return m.slackAccounts, m.slackErr
}

func (m *mockRepo) GetSlackAccount(_ context.Context, _ uuid.UUID) (*repository.SlackAccount, error) {
	return nil, nil
}

func (m *mockRepo) UpsertSlackAccount(_ context.Context, _ *repository.SlackAccount) error {
	return nil
}

func (m *mockRepo) DeleteSlackAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockRepo) ListEmailAccounts(_ context.Context) ([]*repository.EmailAccount, error) {
	return m.emailAccounts, m.emailErr
}

func (m *mockRepo) GetEmailAccount(_ context.Context, _ uuid.UUID) (*repository.EmailAccount, error) {
	return nil, nil
}

func (m *mockRepo) UpsertEmailAccount(_ context.Context, _ *repository.EmailAccount) error {
	return nil
}

func (m *mockRepo) DeleteEmailAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockRepo) ListCalendarAccounts(_ context.Context) ([]*repository.CalendarAccount, error) {
	return m.calendarAccounts, m.calendarErr
}

func (m *mockRepo) GetCalendarAccount(_ context.Context, _ uuid.UUID) (*repository.CalendarAccount, error) {
	return nil, nil
}

func (m *mockRepo) UpsertCalendarAccount(_ context.Context, _ *repository.CalendarAccount) error {
	return nil
}

func (m *mockRepo) DeleteCalendarAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}

type mockWatchers struct{}

func (m *mockWatchers) AddWatcher(_ string, _ orchestrator.Watcher) {}

func (m *mockWatchers) RemoveWatcher(_ string) {}

func (m *mockWatchers) ListWatcherNames() []string {
	return nil
}

type mockMessageDeleter struct{}

func (m *mockMessageDeleter) DeleteBySourceAccount(_ context.Context, _, _ string) (int64, error) {
	return 0, nil
}

func stubFactory(_ string, _ uuid.UUID) error {
	return nil
}

// --- Test Suite ---

type ServiceManagerSuite struct {
	suite.Suite
}

func TestServiceManager(t *testing.T) {
	suite.Run(t, new(ServiceManagerSuite))
}

func (s *ServiceManagerSuite) TestNewServiceManager_Valid() {
	mgr, err := servicemanager.NewServiceManager(&mockRepo{}, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.NoError(err)
	s.NotNil(mgr)
}

func (s *ServiceManagerSuite) TestNewServiceManager_NilRepo() {
	mgr, err := servicemanager.NewServiceManager(nil, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Error(err)
	s.Nil(mgr)
	s.Contains(err.Error(), "repo")
}

func (s *ServiceManagerSuite) TestNewServiceManager_NilWatchers() {
	mgr, err := servicemanager.NewServiceManager(&mockRepo{}, nil, stubFactory, &mockMessageDeleter{})
	s.Error(err)
	s.Nil(mgr)
	s.Contains(err.Error(), "watcher")
}

func (s *ServiceManagerSuite) TestNewServiceManager_NilFactory() {
	mgr, err := servicemanager.NewServiceManager(&mockRepo{}, &mockWatchers{}, nil, &mockMessageDeleter{})
	s.Error(err)
	s.Nil(mgr)
	s.Contains(err.Error(), "factory")
}

func (s *ServiceManagerSuite) TestNewServiceManager_NilMessageDeleter() {
	mgr, err := servicemanager.NewServiceManager(&mockRepo{}, &mockWatchers{}, stubFactory, nil)
	s.Error(err)
	s.Nil(mgr)
	s.Contains(err.Error(), "message")
}

// --- ListSlackAccounts ---

func (s *ServiceManagerSuite) TestListSlackAccounts_ReturnsFromRepo() {
	accounts := []*repository.SlackAccount{
		{ID: uuid.New(), Enabled: true, WorkspaceID: "T001", FriendlyName: "workspace-1"},
		{ID: uuid.New(), Enabled: false, WorkspaceID: "T002", FriendlyName: "workspace-2"},
	}
	repo := &mockRepo{slackAccounts: accounts}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.ListSlackAccounts(context.Background())
	s.NoError(err)
	s.Equal(accounts, got)
}

func (s *ServiceManagerSuite) TestListSlackAccounts_RepoError() {
	repoErr := errors.New("slack list failed")
	repo := &mockRepo{slackErr: repoErr}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.ListSlackAccounts(context.Background())
	s.ErrorIs(err, repoErr)
	s.Nil(got)
}

// --- ListEmailAccounts ---

func (s *ServiceManagerSuite) TestListEmailAccounts_ReturnsFromRepo() {
	accounts := []*repository.EmailAccount{
		{ID: uuid.New(), Enabled: true, IMAPHost: "imap.example.com", FriendlyName: "personal"},
		{ID: uuid.New(), Enabled: false, IMAPHost: "imap.work.com", FriendlyName: "work"},
	}
	repo := &mockRepo{emailAccounts: accounts}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.ListEmailAccounts(context.Background())
	s.NoError(err)
	s.Equal(accounts, got)
}

func (s *ServiceManagerSuite) TestListEmailAccounts_RepoError() {
	repoErr := errors.New("email list failed")
	repo := &mockRepo{emailErr: repoErr}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.ListEmailAccounts(context.Background())
	s.ErrorIs(err, repoErr)
	s.Nil(got)
}

// --- ListCalendarAccounts ---

func (s *ServiceManagerSuite) TestListCalendarAccounts_ReturnsFromRepo() {
	accounts := []*repository.CalendarAccount{
		{ID: uuid.New(), Enabled: true, Name: "personal-cal", ICSURL: "https://cal.example.com/a.ics"},
		{ID: uuid.New(), Enabled: false, Name: "work-cal", ICSURL: "https://cal.work.com/b.ics"},
	}
	repo := &mockRepo{calendarAccounts: accounts}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.ListCalendarAccounts(context.Background())
	s.NoError(err)
	s.Equal(accounts, got)
}

func (s *ServiceManagerSuite) TestListCalendarAccounts_RepoError() {
	repoErr := errors.New("calendar list failed")
	repo := &mockRepo{calendarErr: repoErr}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.ListCalendarAccounts(context.Background())
	s.ErrorIs(err, repoErr)
	s.Nil(got)
}
