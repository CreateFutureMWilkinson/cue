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

	getSlackAccount    *repository.SlackAccount
	getSlackErr        error
	getEmailAccount    *repository.EmailAccount
	getEmailErr        error
	getCalendarAccount *repository.CalendarAccount
	getCalendarErr     error

	upsertSlackAccount *repository.SlackAccount
	upsertSlackErr     error

	upsertEmailAccount *repository.EmailAccount
	upsertEmailErr     error

	upsertCalendarAccount *repository.CalendarAccount
	upsertCalendarErr     error
}

func (m *mockRepo) ListSlackAccounts(_ context.Context) ([]*repository.SlackAccount, error) {
	return m.slackAccounts, m.slackErr
}

func (m *mockRepo) GetSlackAccount(_ context.Context, _ uuid.UUID) (*repository.SlackAccount, error) {
	return m.getSlackAccount, m.getSlackErr
}

func (m *mockRepo) UpsertSlackAccount(_ context.Context, acct *repository.SlackAccount) error {
	m.upsertSlackAccount = acct
	return m.upsertSlackErr
}

func (m *mockRepo) DeleteSlackAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockRepo) ListEmailAccounts(_ context.Context) ([]*repository.EmailAccount, error) {
	return m.emailAccounts, m.emailErr
}

func (m *mockRepo) GetEmailAccount(_ context.Context, _ uuid.UUID) (*repository.EmailAccount, error) {
	return m.getEmailAccount, m.getEmailErr
}

func (m *mockRepo) UpsertEmailAccount(_ context.Context, acct *repository.EmailAccount) error {
	m.upsertEmailAccount = acct
	return m.upsertEmailErr
}

func (m *mockRepo) DeleteEmailAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockRepo) ListCalendarAccounts(_ context.Context) ([]*repository.CalendarAccount, error) {
	return m.calendarAccounts, m.calendarErr
}

func (m *mockRepo) GetCalendarAccount(_ context.Context, _ uuid.UUID) (*repository.CalendarAccount, error) {
	return m.getCalendarAccount, m.getCalendarErr
}

func (m *mockRepo) UpsertCalendarAccount(_ context.Context, acct *repository.CalendarAccount) error {
	m.upsertCalendarAccount = acct
	return m.upsertCalendarErr
}

func (m *mockRepo) DeleteCalendarAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}

type mockWatchers struct {
	removedNames []string
}

func (m *mockWatchers) AddWatcher(_ string, _ orchestrator.Watcher) {}

func (m *mockWatchers) RemoveWatcher(name string) {
	m.removedNames = append(m.removedNames, name)
}

func (m *mockWatchers) ListWatcherNames() []string {
	return nil
}

type mockMessageDeleter struct{}

func (m *mockMessageDeleter) DeleteBySourceAccount(_ context.Context, _, _ string) (int64, error) {
	return 0, nil
}

type trackingFactory struct {
	calledType string
	calledID   uuid.UUID
	err        error
}

func (f *trackingFactory) create(accountType string, accountID uuid.UUID) error {
	f.calledType = accountType
	f.calledID = accountID
	return f.err
}

func stubFactory(_ string, _ uuid.UUID) error {
	return nil
}

type mockSlackValidator struct {
	err error
}

func (v *mockSlackValidator) ValidateSlack(_ context.Context, _ string) error {
	return v.err
}

type mockEmailValidator struct {
	err error
}

func (v *mockEmailValidator) ValidateEmail(_ context.Context, _ string, _ int, _, _, _ string) error {
	return v.err
}

type mockCalendarValidator struct {
	err error
}

func (v *mockCalendarValidator) ValidateCalendar(_ context.Context, _ string) error {
	return v.err
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

// --- GetSlackAccount ---

func (s *ServiceManagerSuite) TestGetSlackAccount_ReturnsWithMaskedToken() {
	id := uuid.New()
	acct := &repository.SlackAccount{
		ID:           id,
		Enabled:      true,
		Token:        "xoxp-real-token",
		WorkspaceID:  "T001",
		Username:     "testuser",
		FriendlyName: "my-workspace",
	}
	repo := &mockRepo{getSlackAccount: acct}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.GetSlackAccount(context.Background(), id)
	s.NoError(err)
	s.Require().NotNil(got)
	s.Equal(servicemanager.CredentialMask, got.Token)
	s.Equal(id, got.ID)
	s.Equal("T001", got.WorkspaceID)
	s.Equal("my-workspace", got.FriendlyName)
}

func (s *ServiceManagerSuite) TestGetSlackAccount_NotFound() {
	repoErr := errors.New("not found")
	repo := &mockRepo{getSlackErr: repoErr}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.GetSlackAccount(context.Background(), uuid.New())
	s.ErrorIs(err, repoErr)
	s.Nil(got)
}

// --- GetEmailAccount ---

func (s *ServiceManagerSuite) TestGetEmailAccount_ReturnsWithMaskedPassword() {
	id := uuid.New()
	acct := &repository.EmailAccount{
		ID:           id,
		Enabled:      true,
		IMAPHost:     "imap.example.com",
		IMAPPort:     993,
		Username:     "user@example.com",
		Password:     "secret123",
		Encryption:   "tls",
		FriendlyName: "personal",
	}
	repo := &mockRepo{getEmailAccount: acct}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.GetEmailAccount(context.Background(), id)
	s.NoError(err)
	s.Require().NotNil(got)
	s.Equal(servicemanager.CredentialMask, got.Password)
	s.Equal(id, got.ID)
	s.Equal("imap.example.com", got.IMAPHost)
	s.Equal("personal", got.FriendlyName)
}

func (s *ServiceManagerSuite) TestGetEmailAccount_NotFound() {
	repoErr := errors.New("not found")
	repo := &mockRepo{getEmailErr: repoErr}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.GetEmailAccount(context.Background(), uuid.New())
	s.ErrorIs(err, repoErr)
	s.Nil(got)
}

// --- GetCalendarAccount ---

func (s *ServiceManagerSuite) TestGetCalendarAccount_Returns() {
	id := uuid.New()
	acct := &repository.CalendarAccount{
		ID:      id,
		Enabled: true,
		Name:    "personal-cal",
		ICSURL:  "https://cal.example.com/a.ics",
	}
	repo := &mockRepo{getCalendarAccount: acct}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.GetCalendarAccount(context.Background(), id)
	s.NoError(err)
	s.Require().NotNil(got)
	s.Equal(id, got.ID)
	s.Equal("personal-cal", got.Name)
	s.Equal("https://cal.example.com/a.ics", got.ICSURL)
}

func (s *ServiceManagerSuite) TestGetCalendarAccount_NotFound() {
	repoErr := errors.New("not found")
	repo := &mockRepo{getCalendarErr: repoErr}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.GetCalendarAccount(context.Background(), uuid.New())
	s.ErrorIs(err, repoErr)
	s.Nil(got)
}

// --- CreateSlackAccount ---

func (s *ServiceManagerSuite) TestCreateSlackAccount_Success() {
	repo := &mockRepo{}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.SlackAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Token:               "xoxp-real-token",
		WorkspaceID:         "T001",
		Username:            "testuser",
		FriendlyName:        "my-workspace",
		PollIntervalSeconds: 120,
	}

	got, err := mgr.CreateSlackAccount(context.Background(), acct)
	s.NoError(err)
	s.Require().NotNil(got)

	// Repo should have been called with the account
	s.NotNil(repo.upsertSlackAccount)
	s.Equal(acct.ID, repo.upsertSlackAccount.ID)

	// Factory should have been called
	s.Equal("slack", factory.calledType)
	s.Equal(acct.ID, factory.calledID)

	// Returned account has masked token
	s.Equal(servicemanager.CredentialMask, got.Token)
	s.Equal(acct.ID, got.ID)
	s.Equal("T001", got.WorkspaceID)
}

func (s *ServiceManagerSuite) TestCreateSlackAccount_EmptyToken() {
	mgr, err := servicemanager.NewServiceManager(&mockRepo{}, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.SlackAccount{
		ID:                  uuid.New(),
		WorkspaceID:         "T001",
		Token:               "",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateSlackAccount(context.Background(), acct)
	s.Error(err)
	s.Nil(got)
	s.Contains(err.Error(), "token")
}

func (s *ServiceManagerSuite) TestCreateSlackAccount_EmptyWorkspaceID() {
	mgr, err := servicemanager.NewServiceManager(&mockRepo{}, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.SlackAccount{
		ID:                  uuid.New(),
		Token:               "xoxp-valid",
		WorkspaceID:         "",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateSlackAccount(context.Background(), acct)
	s.Error(err)
	s.Nil(got)
	s.Contains(err.Error(), "workspace")
}

func (s *ServiceManagerSuite) TestCreateSlackAccount_ValidationFails() {
	repo := &mockRepo{}
	validationErr := errors.New("invalid slack token")
	validator := &mockSlackValidator{err: validationErr}
	mgr, err := servicemanager.NewServiceManager(
		repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{},
		servicemanager.WithSlackValidator(validator),
	)
	s.Require().NoError(err)

	acct := &repository.SlackAccount{
		ID:                  uuid.New(),
		Token:               "xoxp-bad-token",
		WorkspaceID:         "T001",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateSlackAccount(context.Background(), acct)
	s.Error(err)
	s.Nil(got)
	// Repo should NOT have been called
	s.Nil(repo.upsertSlackAccount)
}

func (s *ServiceManagerSuite) TestCreateSlackAccount_RepoError() {
	repoErr := errors.New("db write failed")
	repo := &mockRepo{upsertSlackErr: repoErr}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.SlackAccount{
		ID:                  uuid.New(),
		Token:               "xoxp-valid",
		WorkspaceID:         "T001",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateSlackAccount(context.Background(), acct)
	s.Error(err)
	s.Nil(got)
}

func (s *ServiceManagerSuite) TestCreateSlackAccount_WatcherFactoryError() {
	factory := &trackingFactory{err: errors.New("factory failed")}
	mgr, err := servicemanager.NewServiceManager(&mockRepo{}, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.SlackAccount{
		ID:                  uuid.New(),
		Token:               "xoxp-valid",
		WorkspaceID:         "T001",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateSlackAccount(context.Background(), acct)
	s.Error(err)
	s.Nil(got)
}

func (s *ServiceManagerSuite) TestCreateSlackAccount_DefaultPollInterval() {
	repo := &mockRepo{}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.SlackAccount{
		ID:                  uuid.New(),
		Token:               "xoxp-valid",
		WorkspaceID:         "T001",
		PollIntervalSeconds: 0, // should default
	}

	got, err := mgr.CreateSlackAccount(context.Background(), acct)
	s.NoError(err)
	s.Require().NotNil(got)

	// The upserted account should have the default poll interval
	s.Require().NotNil(repo.upsertSlackAccount)
	s.Equal(servicemanager.DefaultSlackPollInterval, repo.upsertSlackAccount.PollIntervalSeconds)
}

func (s *ServiceManagerSuite) TestCreateSlackAccount_GeneratesUUID() {
	repo := &mockRepo{}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.SlackAccount{
		ID:                  uuid.Nil, // zero UUID
		Token:               "xoxp-valid",
		WorkspaceID:         "T001",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateSlackAccount(context.Background(), acct)
	s.NoError(err)
	s.Require().NotNil(got)

	// UUID should have been generated (non-zero)
	s.NotEqual(uuid.Nil, got.ID)
	s.Require().NotNil(repo.upsertSlackAccount)
	s.NotEqual(uuid.Nil, repo.upsertSlackAccount.ID)
}

// --- CreateEmailAccount ---

func (s *ServiceManagerSuite) TestCreateEmailAccount_Success() {
	repo := &mockRepo{}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.EmailAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		IMAPHost:            "imap.example.com",
		IMAPPort:            993,
		Username:            "user@example.com",
		Password:            "secret123",
		Encryption:          "tls",
		FriendlyName:        "personal",
		PollIntervalSeconds: 120,
	}

	got, err := mgr.CreateEmailAccount(context.Background(), acct)
	s.NoError(err)
	s.Require().NotNil(got)

	// Repo should have been called with the account
	s.NotNil(repo.upsertEmailAccount)
	s.Equal(acct.ID, repo.upsertEmailAccount.ID)

	// Factory should have been called
	s.Equal("email", factory.calledType)
	s.Equal(acct.ID, factory.calledID)

	// Returned account has masked password
	s.Equal(servicemanager.CredentialMask, got.Password)
	s.Equal(acct.ID, got.ID)
	s.Equal("imap.example.com", got.IMAPHost)
}

func (s *ServiceManagerSuite) TestCreateEmailAccount_EmptyHost() {
	mgr, err := servicemanager.NewServiceManager(&mockRepo{}, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.EmailAccount{
		ID:                  uuid.New(),
		IMAPHost:            "",
		IMAPPort:            993,
		Username:            "user@example.com",
		Password:            "secret",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateEmailAccount(context.Background(), acct)
	s.Error(err)
	s.Nil(got)
	s.Contains(err.Error(), "host")
}

func (s *ServiceManagerSuite) TestCreateEmailAccount_EmptyUsername() {
	mgr, err := servicemanager.NewServiceManager(&mockRepo{}, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.EmailAccount{
		ID:                  uuid.New(),
		IMAPHost:            "imap.example.com",
		IMAPPort:            993,
		Username:            "",
		Password:            "secret",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateEmailAccount(context.Background(), acct)
	s.Error(err)
	s.Nil(got)
	s.Contains(err.Error(), "username")
}

func (s *ServiceManagerSuite) TestCreateEmailAccount_EmptyPassword() {
	mgr, err := servicemanager.NewServiceManager(&mockRepo{}, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.EmailAccount{
		ID:                  uuid.New(),
		IMAPHost:            "imap.example.com",
		IMAPPort:            993,
		Username:            "user@example.com",
		Password:            "",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateEmailAccount(context.Background(), acct)
	s.Error(err)
	s.Nil(got)
	s.Contains(err.Error(), "password")
}

func (s *ServiceManagerSuite) TestCreateEmailAccount_InvalidPort() {
	mgr, err := servicemanager.NewServiceManager(&mockRepo{}, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.EmailAccount{
		ID:                  uuid.New(),
		IMAPHost:            "imap.example.com",
		IMAPPort:            0,
		Username:            "user@example.com",
		Password:            "secret",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateEmailAccount(context.Background(), acct)
	s.Error(err)
	s.Nil(got)
	s.Contains(err.Error(), "port")
}

func (s *ServiceManagerSuite) TestCreateEmailAccount_ValidationFails() {
	repo := &mockRepo{}
	validationErr := errors.New("IMAP connection failed")
	validator := &mockEmailValidator{err: validationErr}
	mgr, err := servicemanager.NewServiceManager(
		repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{},
		servicemanager.WithEmailValidator(validator),
	)
	s.Require().NoError(err)

	acct := &repository.EmailAccount{
		ID:                  uuid.New(),
		IMAPHost:            "imap.example.com",
		IMAPPort:            993,
		Username:            "user@example.com",
		Password:            "bad-password",
		Encryption:          "tls",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateEmailAccount(context.Background(), acct)
	s.Error(err)
	s.Nil(got)
	// Repo should NOT have been called
	s.Nil(repo.upsertEmailAccount)
}

func (s *ServiceManagerSuite) TestCreateEmailAccount_DefaultPollInterval() {
	repo := &mockRepo{}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.EmailAccount{
		ID:                  uuid.New(),
		IMAPHost:            "imap.example.com",
		IMAPPort:            993,
		Username:            "user@example.com",
		Password:            "secret",
		PollIntervalSeconds: 0, // should default
	}

	got, err := mgr.CreateEmailAccount(context.Background(), acct)
	s.NoError(err)
	s.Require().NotNil(got)

	s.Require().NotNil(repo.upsertEmailAccount)
	s.Equal(servicemanager.DefaultEmailPollInterval, repo.upsertEmailAccount.PollIntervalSeconds)
}

func (s *ServiceManagerSuite) TestCreateEmailAccount_GeneratesUUID() {
	repo := &mockRepo{}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.EmailAccount{
		ID:                  uuid.Nil,
		IMAPHost:            "imap.example.com",
		IMAPPort:            993,
		Username:            "user@example.com",
		Password:            "secret",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateEmailAccount(context.Background(), acct)
	s.NoError(err)
	s.Require().NotNil(got)

	s.NotEqual(uuid.Nil, got.ID)
	s.Require().NotNil(repo.upsertEmailAccount)
	s.NotEqual(uuid.Nil, repo.upsertEmailAccount.ID)
}

// --- CreateCalendarAccount ---

func (s *ServiceManagerSuite) TestCreateCalendarAccount_Success() {
	repo := &mockRepo{}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.CalendarAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Name:                "personal-cal",
		ICSURL:              "https://cal.example.com/a.ics",
		PollIntervalSeconds: 120,
	}

	got, err := mgr.CreateCalendarAccount(context.Background(), acct)
	s.NoError(err)
	s.Require().NotNil(got)

	// Repo should have been called
	s.NotNil(repo.upsertCalendarAccount)
	s.Equal(acct.ID, repo.upsertCalendarAccount.ID)

	// Factory should NOT have been called (no watcher for calendar)
	s.Equal("", factory.calledType)
	s.Equal(uuid.Nil, factory.calledID)

	// Returned account matches
	s.Equal(acct.ID, got.ID)
	s.Equal("personal-cal", got.Name)
	s.Equal("https://cal.example.com/a.ics", got.ICSURL)
}

func (s *ServiceManagerSuite) TestCreateCalendarAccount_EmptyName() {
	mgr, err := servicemanager.NewServiceManager(&mockRepo{}, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.CalendarAccount{
		ID:                  uuid.New(),
		Name:                "",
		ICSURL:              "https://cal.example.com/a.ics",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateCalendarAccount(context.Background(), acct)
	s.Error(err)
	s.Nil(got)
	s.Contains(err.Error(), "name")
}

func (s *ServiceManagerSuite) TestCreateCalendarAccount_EmptyICSURL() {
	mgr, err := servicemanager.NewServiceManager(&mockRepo{}, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.CalendarAccount{
		ID:                  uuid.New(),
		Name:                "my-cal",
		ICSURL:              "",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateCalendarAccount(context.Background(), acct)
	s.Error(err)
	s.Nil(got)
	s.Contains(err.Error(), "URL")
}

func (s *ServiceManagerSuite) TestCreateCalendarAccount_ValidationFails() {
	repo := &mockRepo{}
	validationErr := errors.New("invalid ICS URL")
	validator := &mockCalendarValidator{err: validationErr}
	mgr, err := servicemanager.NewServiceManager(
		repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{},
		servicemanager.WithCalendarValidator(validator),
	)
	s.Require().NoError(err)

	acct := &repository.CalendarAccount{
		ID:                  uuid.New(),
		Name:                "my-cal",
		ICSURL:              "https://bad-url.example.com/a.ics",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateCalendarAccount(context.Background(), acct)
	s.Error(err)
	s.Nil(got)
	// Repo should NOT have been called
	s.Nil(repo.upsertCalendarAccount)
}

func (s *ServiceManagerSuite) TestCreateCalendarAccount_DefaultPollInterval() {
	repo := &mockRepo{}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.CalendarAccount{
		ID:                  uuid.New(),
		Name:                "my-cal",
		ICSURL:              "https://cal.example.com/a.ics",
		PollIntervalSeconds: 0, // should default
	}

	got, err := mgr.CreateCalendarAccount(context.Background(), acct)
	s.NoError(err)
	s.Require().NotNil(got)

	s.Require().NotNil(repo.upsertCalendarAccount)
	s.Equal(servicemanager.DefaultCalendarPollInterval, repo.upsertCalendarAccount.PollIntervalSeconds)
}

func (s *ServiceManagerSuite) TestCreateCalendarAccount_GeneratesUUID() {
	repo := &mockRepo{}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	acct := &repository.CalendarAccount{
		ID:                  uuid.Nil,
		Name:                "my-cal",
		ICSURL:              "https://cal.example.com/a.ics",
		PollIntervalSeconds: 60,
	}

	got, err := mgr.CreateCalendarAccount(context.Background(), acct)
	s.NoError(err)
	s.Require().NotNil(got)

	s.NotEqual(uuid.Nil, got.ID)
	s.Require().NotNil(repo.upsertCalendarAccount)
	s.NotEqual(uuid.Nil, repo.upsertCalendarAccount.ID)
}

// --- UpdateSlackAccount ---

func (s *ServiceManagerSuite) TestUpdateSlackAccount_Success() {
	existingID := uuid.New()
	existing := &repository.SlackAccount{
		ID:                  existingID,
		Enabled:             true,
		Token:               "xoxp-old-token",
		WorkspaceID:         "T001",
		Username:            "olduser",
		FriendlyName:        "old-workspace",
		WebURL:              "https://old.slack.com",
		PollIntervalSeconds: 60,
	}
	repo := &mockRepo{getSlackAccount: existing}
	watchers := &mockWatchers{}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, watchers, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	update := &repository.SlackAccount{
		Token:               "xoxp-new-token",
		WorkspaceID:         "T002",
		Username:            "newuser",
		FriendlyName:        "new-workspace",
		WebURL:              "https://new.slack.com",
		PollIntervalSeconds: 120,
		Enabled:             true,
	}

	got, err := mgr.UpdateSlackAccount(context.Background(), existingID, update)
	s.NoError(err)
	s.Require().NotNil(got)

	// Upserted with new token
	s.Require().NotNil(repo.upsertSlackAccount)
	s.Equal("xoxp-new-token", repo.upsertSlackAccount.Token)
	s.Equal("T002", repo.upsertSlackAccount.WorkspaceID)
	s.Equal("newuser", repo.upsertSlackAccount.Username)
	s.Equal("new-workspace", repo.upsertSlackAccount.FriendlyName)
	s.Equal(120, repo.upsertSlackAccount.PollIntervalSeconds)
	s.Equal(existingID, repo.upsertSlackAccount.ID)

	// Old watcher removed
	s.Contains(watchers.removedNames, "slack:T001")

	// Factory called for new watcher
	s.Equal("slack", factory.calledType)
	s.Equal(existingID, factory.calledID)

	// Returned with masked token
	s.Equal(servicemanager.CredentialMask, got.Token)
	s.Equal(existingID, got.ID)
	s.Equal("T002", got.WorkspaceID)
}

func (s *ServiceManagerSuite) TestUpdateSlackAccount_PreservesCredentialWhenEmpty() {
	existingID := uuid.New()
	existing := &repository.SlackAccount{
		ID:                  existingID,
		Enabled:             true,
		Token:               "xoxp-existing-token",
		WorkspaceID:         "T001",
		Username:            "user",
		PollIntervalSeconds: 60,
	}
	repo := &mockRepo{getSlackAccount: existing}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	update := &repository.SlackAccount{
		Token:       "", // empty — should preserve existing
		WorkspaceID: "T001",
		Username:    "user",
	}

	got, err := mgr.UpdateSlackAccount(context.Background(), existingID, update)
	s.NoError(err)
	s.Require().NotNil(got)

	// Existing token preserved in upsert
	s.Require().NotNil(repo.upsertSlackAccount)
	s.Equal("xoxp-existing-token", repo.upsertSlackAccount.Token)
}

func (s *ServiceManagerSuite) TestUpdateSlackAccount_PreservesCredentialWhenMasked() {
	existingID := uuid.New()
	existing := &repository.SlackAccount{
		ID:                  existingID,
		Enabled:             true,
		Token:               "xoxp-existing-token",
		WorkspaceID:         "T001",
		Username:            "user",
		PollIntervalSeconds: 60,
	}
	repo := &mockRepo{getSlackAccount: existing}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	update := &repository.SlackAccount{
		Token:       servicemanager.CredentialMask, // "***" — should preserve existing
		WorkspaceID: "T001",
		Username:    "user",
	}

	got, err := mgr.UpdateSlackAccount(context.Background(), existingID, update)
	s.NoError(err)
	s.Require().NotNil(got)

	// Existing token preserved in upsert
	s.Require().NotNil(repo.upsertSlackAccount)
	s.Equal("xoxp-existing-token", repo.upsertSlackAccount.Token)
}

func (s *ServiceManagerSuite) TestUpdateSlackAccount_NotFound() {
	repoErr := errors.New("not found")
	repo := &mockRepo{getSlackErr: repoErr}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.UpdateSlackAccount(context.Background(), uuid.New(), &repository.SlackAccount{
		Token:       "xoxp-token",
		WorkspaceID: "T001",
	})
	s.ErrorIs(err, repoErr)
	s.Nil(got)
}

func (s *ServiceManagerSuite) TestUpdateSlackAccount_RepoError() {
	existingID := uuid.New()
	existing := &repository.SlackAccount{
		ID:                  existingID,
		Token:               "xoxp-existing",
		WorkspaceID:         "T001",
		PollIntervalSeconds: 60,
	}
	upsertErr := errors.New("db write failed")
	repo := &mockRepo{getSlackAccount: existing, upsertSlackErr: upsertErr}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.UpdateSlackAccount(context.Background(), existingID, &repository.SlackAccount{
		Token:       "xoxp-new",
		WorkspaceID: "T001",
	})
	s.ErrorIs(err, upsertErr)
	s.Nil(got)
}

// --- UpdateEmailAccount ---

func (s *ServiceManagerSuite) TestUpdateEmailAccount_Success() {
	existingID := uuid.New()
	existing := &repository.EmailAccount{
		ID:                  existingID,
		Enabled:             true,
		IMAPHost:            "imap.old.com",
		IMAPPort:            993,
		Username:            "old@example.com",
		Password:            "old-secret",
		Encryption:          "tls",
		FriendlyName:        "old-email",
		WebURL:              "https://old.mail.com",
		PollIntervalSeconds: 600,
	}
	repo := &mockRepo{getEmailAccount: existing}
	watchers := &mockWatchers{}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, watchers, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	update := &repository.EmailAccount{
		IMAPHost:            "imap.new.com",
		IMAPPort:            143,
		Username:            "new@example.com",
		Password:            "new-secret",
		Encryption:          "starttls",
		FriendlyName:        "new-email",
		WebURL:              "https://new.mail.com",
		PollIntervalSeconds: 120,
		Enabled:             true,
	}

	got, err := mgr.UpdateEmailAccount(context.Background(), existingID, update)
	s.NoError(err)
	s.Require().NotNil(got)

	// Upserted with new values
	s.Require().NotNil(repo.upsertEmailAccount)
	s.Equal("new-secret", repo.upsertEmailAccount.Password)
	s.Equal("imap.new.com", repo.upsertEmailAccount.IMAPHost)
	s.Equal(143, repo.upsertEmailAccount.IMAPPort)
	s.Equal("new@example.com", repo.upsertEmailAccount.Username)
	s.Equal("starttls", repo.upsertEmailAccount.Encryption)
	s.Equal(existingID, repo.upsertEmailAccount.ID)

	// Old watcher removed
	s.Contains(watchers.removedNames, "email:old@example.com")

	// Factory called for new watcher
	s.Equal("email", factory.calledType)
	s.Equal(existingID, factory.calledID)

	// Returned with masked password
	s.Equal(servicemanager.CredentialMask, got.Password)
	s.Equal(existingID, got.ID)
}

func (s *ServiceManagerSuite) TestUpdateEmailAccount_PreservesPasswordWhenEmpty() {
	existingID := uuid.New()
	existing := &repository.EmailAccount{
		ID:                  existingID,
		IMAPHost:            "imap.example.com",
		IMAPPort:            993,
		Username:            "user@example.com",
		Password:            "existing-secret",
		PollIntervalSeconds: 600,
	}
	repo := &mockRepo{getEmailAccount: existing}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	update := &repository.EmailAccount{
		Password: "", // empty — should preserve existing
		IMAPHost: "imap.example.com",
		IMAPPort: 993,
		Username: "user@example.com",
	}

	got, err := mgr.UpdateEmailAccount(context.Background(), existingID, update)
	s.NoError(err)
	s.Require().NotNil(got)

	s.Require().NotNil(repo.upsertEmailAccount)
	s.Equal("existing-secret", repo.upsertEmailAccount.Password)
}

func (s *ServiceManagerSuite) TestUpdateEmailAccount_PreservesPasswordWhenMasked() {
	existingID := uuid.New()
	existing := &repository.EmailAccount{
		ID:                  existingID,
		IMAPHost:            "imap.example.com",
		IMAPPort:            993,
		Username:            "user@example.com",
		Password:            "existing-secret",
		PollIntervalSeconds: 600,
	}
	repo := &mockRepo{getEmailAccount: existing}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	update := &repository.EmailAccount{
		Password: servicemanager.CredentialMask, // "***" — should preserve existing
		IMAPHost: "imap.example.com",
		IMAPPort: 993,
		Username: "user@example.com",
	}

	got, err := mgr.UpdateEmailAccount(context.Background(), existingID, update)
	s.NoError(err)
	s.Require().NotNil(got)

	s.Require().NotNil(repo.upsertEmailAccount)
	s.Equal("existing-secret", repo.upsertEmailAccount.Password)
}

// --- UpdateCalendarAccount ---

func (s *ServiceManagerSuite) TestUpdateCalendarAccount_Success() {
	existingID := uuid.New()
	existing := &repository.CalendarAccount{
		ID:                  existingID,
		Enabled:             true,
		Name:                "old-cal",
		ICSURL:              "https://old.cal.com/a.ics",
		PollIntervalSeconds: 600,
	}
	repo := &mockRepo{getCalendarAccount: existing}
	factory := &trackingFactory{}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, factory.create, &mockMessageDeleter{})
	s.Require().NoError(err)

	update := &repository.CalendarAccount{
		Name:                "new-cal",
		ICSURL:              "https://new.cal.com/b.ics",
		PollIntervalSeconds: 120,
		Enabled:             true,
	}

	got, err := mgr.UpdateCalendarAccount(context.Background(), existingID, update)
	s.NoError(err)
	s.Require().NotNil(got)

	// Upserted with new values
	s.Require().NotNil(repo.upsertCalendarAccount)
	s.Equal("new-cal", repo.upsertCalendarAccount.Name)
	s.Equal("https://new.cal.com/b.ics", repo.upsertCalendarAccount.ICSURL)
	s.Equal(120, repo.upsertCalendarAccount.PollIntervalSeconds)
	s.Equal(existingID, repo.upsertCalendarAccount.ID)

	// Factory should NOT have been called (no watcher for calendar)
	s.Equal("", factory.calledType)
	s.Equal(uuid.Nil, factory.calledID)

	// Returned account matches
	s.Equal(existingID, got.ID)
	s.Equal("new-cal", got.Name)
}

func (s *ServiceManagerSuite) TestUpdateCalendarAccount_NotFound() {
	repoErr := errors.New("not found")
	repo := &mockRepo{getCalendarErr: repoErr}
	mgr, err := servicemanager.NewServiceManager(repo, &mockWatchers{}, stubFactory, &mockMessageDeleter{})
	s.Require().NoError(err)

	got, err := mgr.UpdateCalendarAccount(context.Background(), uuid.New(), &repository.CalendarAccount{
		Name:   "cal",
		ICSURL: "https://cal.example.com/a.ics",
	})
	s.ErrorIs(err, repoErr)
	s.Nil(got)
}
