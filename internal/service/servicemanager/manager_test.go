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
