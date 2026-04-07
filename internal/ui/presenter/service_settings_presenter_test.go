package presenter_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Mock ServiceConfigRepository ---

type mockServiceConfigRepo struct {
	listSlackFn      func(ctx context.Context) ([]*repository.SlackAccount, error)
	getSlackFn       func(ctx context.Context, id uuid.UUID) (*repository.SlackAccount, error)
	upsertSlackFn    func(ctx context.Context, acct *repository.SlackAccount) error
	deleteSlackFn    func(ctx context.Context, id uuid.UUID) error
	listEmailFn      func(ctx context.Context) ([]*repository.EmailAccount, error)
	getEmailFn       func(ctx context.Context, id uuid.UUID) (*repository.EmailAccount, error)
	upsertEmailFn    func(ctx context.Context, acct *repository.EmailAccount) error
	deleteEmailFn    func(ctx context.Context, id uuid.UUID) error
	listCalendarFn   func(ctx context.Context) ([]*repository.CalendarAccount, error)
	getCalendarFn    func(ctx context.Context, id uuid.UUID) (*repository.CalendarAccount, error)
	upsertCalendarFn func(ctx context.Context, acct *repository.CalendarAccount) error
	deleteCalendarFn func(ctx context.Context, id uuid.UUID) error
}

func (m *mockServiceConfigRepo) ListSlackAccounts(ctx context.Context) ([]*repository.SlackAccount, error) {
	return m.listSlackFn(ctx)
}

func (m *mockServiceConfigRepo) GetSlackAccount(ctx context.Context, id uuid.UUID) (*repository.SlackAccount, error) {
	return m.getSlackFn(ctx, id)
}

func (m *mockServiceConfigRepo) UpsertSlackAccount(ctx context.Context, acct *repository.SlackAccount) error {
	return m.upsertSlackFn(ctx, acct)
}

func (m *mockServiceConfigRepo) DeleteSlackAccount(ctx context.Context, id uuid.UUID) error {
	return m.deleteSlackFn(ctx, id)
}

func (m *mockServiceConfigRepo) ListEmailAccounts(ctx context.Context) ([]*repository.EmailAccount, error) {
	return m.listEmailFn(ctx)
}

func (m *mockServiceConfigRepo) GetEmailAccount(ctx context.Context, id uuid.UUID) (*repository.EmailAccount, error) {
	return m.getEmailFn(ctx, id)
}

func (m *mockServiceConfigRepo) UpsertEmailAccount(ctx context.Context, acct *repository.EmailAccount) error {
	return m.upsertEmailFn(ctx, acct)
}

func (m *mockServiceConfigRepo) DeleteEmailAccount(ctx context.Context, id uuid.UUID) error {
	return m.deleteEmailFn(ctx, id)
}

func (m *mockServiceConfigRepo) ListCalendarAccounts(ctx context.Context) ([]*repository.CalendarAccount, error) {
	return m.listCalendarFn(ctx)
}

func (m *mockServiceConfigRepo) GetCalendarAccount(ctx context.Context, id uuid.UUID) (*repository.CalendarAccount, error) {
	return m.getCalendarFn(ctx, id)
}

func (m *mockServiceConfigRepo) UpsertCalendarAccount(ctx context.Context, acct *repository.CalendarAccount) error {
	return m.upsertCalendarFn(ctx, acct)
}

func (m *mockServiceConfigRepo) DeleteCalendarAccount(ctx context.Context, id uuid.UUID) error {
	return m.deleteCalendarFn(ctx, id)
}

// --- Mock WatcherManager ---

type mockWatcherManager struct {
	addCalls    []struct{ name string }
	removeCalls []string
	names       []string
}

func (m *mockWatcherManager) AddWatcher(name string, w any) {
	m.addCalls = append(m.addCalls, struct{ name string }{name: name})
}

func (m *mockWatcherManager) RemoveWatcher(name string) {
	m.removeCalls = append(m.removeCalls, name)
}

func (m *mockWatcherManager) ListWatcherNames() []string {
	return m.names
}

// --- Mock SlackValidator ---

type mockSlackValidator struct {
	validateFn func(ctx context.Context, token string) error
}

func (m *mockSlackValidator) ValidateSlack(ctx context.Context, token string) error {
	return m.validateFn(ctx, token)
}

// --- Suite ---

type ServiceSettingsSuite struct {
	suite.Suite
}

func TestServiceSettings(t *testing.T) {
	suite.Run(t, new(ServiceSettingsSuite))
}

// --- Helper to build a valid SlackAccount ---

func validSlackAccount() *repository.SlackAccount {
	return &repository.SlackAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Token:               "xoxb-test-token",
		WorkspaceID:         "T12345",
		PollIntervalSeconds: 600,
	}
}

// --- Helper to build a valid EmailAccount ---

func validEmailAccount() *repository.EmailAccount {
	return &repository.EmailAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		IMAPHost:            "imap.gmail.com",
		IMAPPort:            993,
		Username:            "user@gmail.com",
		Password:            "my-secret-password",
		PollIntervalSeconds: 600,
	}
}

// --- Helper to build a valid CalendarAccount ---

func validCalendarAccount() *repository.CalendarAccount {
	return &repository.CalendarAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Name:                "Work Calendar",
		ICSURL:              "https://calendar.example.com/feed.ics",
		PollIntervalSeconds: 3600,
	}
}

// --- List tests ---

func (s *ServiceSettingsSuite) TestListSlackAccounts() {
	acct1 := validSlackAccount()
	acct2 := validSlackAccount()
	repo := &mockServiceConfigRepo{
		listSlackFn: func(ctx context.Context) ([]*repository.SlackAccount, error) {
			return []*repository.SlackAccount{acct1, acct2}, nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error { return nil }

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	accounts, err := p.ListSlackAccounts(context.Background())

	s.Require().NoError(err)
	s.Len(accounts, 2)
	s.Equal(acct1.ID, accounts[0].ID)
	s.Equal(acct2.ID, accounts[1].ID)
}

func (s *ServiceSettingsSuite) TestListSlackAccountsEmpty() {
	repo := &mockServiceConfigRepo{
		listSlackFn: func(ctx context.Context) ([]*repository.SlackAccount, error) {
			return []*repository.SlackAccount{}, nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error { return nil }

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	accounts, err := p.ListSlackAccounts(context.Background())

	s.Require().NoError(err)
	s.Empty(accounts)
}

func (s *ServiceSettingsSuite) TestListEmailAccounts() {
	acct1 := validEmailAccount()
	acct2 := validEmailAccount()
	repo := &mockServiceConfigRepo{
		listEmailFn: func(ctx context.Context) ([]*repository.EmailAccount, error) {
			return []*repository.EmailAccount{acct1, acct2}, nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error { return nil }

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	accounts, err := p.ListEmailAccounts(context.Background())

	s.Require().NoError(err)
	s.Len(accounts, 2)
	s.Equal(acct1.ID, accounts[0].ID)
	s.Equal(acct2.ID, accounts[1].ID)
}

func (s *ServiceSettingsSuite) TestListEmailAccountsEmpty() {
	repo := &mockServiceConfigRepo{
		listEmailFn: func(ctx context.Context) ([]*repository.EmailAccount, error) {
			return []*repository.EmailAccount{}, nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error { return nil }

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	accounts, err := p.ListEmailAccounts(context.Background())

	s.Require().NoError(err)
	s.Empty(accounts)
}

// --- Save new account tests ---

func (s *ServiceSettingsSuite) TestSaveNewSlackAccount() {
	acct := validSlackAccount()
	var upsertedAcct *repository.SlackAccount
	repo := &mockServiceConfigRepo{
		upsertSlackFn: func(ctx context.Context, a *repository.SlackAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	var factoryCalledType string
	var factoryCalledID uuid.UUID
	factory := func(accountType string, accountID uuid.UUID) error {
		factoryCalledType = accountType
		factoryCalledID = accountID
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.SaveSlackAccount(context.Background(), acct)

	s.Require().NoError(err)
	s.Equal(acct, upsertedAcct)
	s.Equal("slack", factoryCalledType)
	s.Equal(acct.ID, factoryCalledID)
}

func (s *ServiceSettingsSuite) TestSaveNewEmailAccount() {
	acct := validEmailAccount()
	var upsertedAcct *repository.EmailAccount
	repo := &mockServiceConfigRepo{
		upsertEmailFn: func(ctx context.Context, a *repository.EmailAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	var factoryCalledType string
	var factoryCalledID uuid.UUID
	factory := func(accountType string, accountID uuid.UUID) error {
		factoryCalledType = accountType
		factoryCalledID = accountID
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.SaveEmailAccount(context.Background(), acct)

	s.Require().NoError(err)
	s.Equal(acct, upsertedAcct)
	s.Equal("email", factoryCalledType)
	s.Equal(acct.ID, factoryCalledID)
}

// --- Edit account tests ---

func (s *ServiceSettingsSuite) TestEditSlackAccount() {
	acct := validSlackAccount()
	acct.WorkspaceID = "T99999"
	var upsertedAcct *repository.SlackAccount
	repo := &mockServiceConfigRepo{
		upsertSlackFn: func(ctx context.Context, a *repository.SlackAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	var factoryCalledType string
	var factoryCalledID uuid.UUID
	factory := func(accountType string, accountID uuid.UUID) error {
		factoryCalledType = accountType
		factoryCalledID = accountID
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.EditSlackAccount(context.Background(), acct, "T00001")

	s.Require().NoError(err)
	s.Equal(acct, upsertedAcct)
	s.Require().Len(mgr.removeCalls, 1)
	s.Equal("slack:T00001", mgr.removeCalls[0])
	s.Equal("slack", factoryCalledType)
	s.Equal(acct.ID, factoryCalledID)
}

func (s *ServiceSettingsSuite) TestEditEmailAccount() {
	acct := validEmailAccount()
	acct.Username = "new@gmail.com"
	var upsertedAcct *repository.EmailAccount
	repo := &mockServiceConfigRepo{
		upsertEmailFn: func(ctx context.Context, a *repository.EmailAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	var factoryCalledType string
	var factoryCalledID uuid.UUID
	factory := func(accountType string, accountID uuid.UUID) error {
		factoryCalledType = accountType
		factoryCalledID = accountID
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.EditEmailAccount(context.Background(), acct, "old@gmail.com")

	s.Require().NoError(err)
	s.Equal(acct, upsertedAcct)
	s.Require().Len(mgr.removeCalls, 1)
	s.Equal("email:old@gmail.com", mgr.removeCalls[0])
	s.Equal("email", factoryCalledType)
	s.Equal(acct.ID, factoryCalledID)
}

// --- Delete account tests ---

func (s *ServiceSettingsSuite) TestDeleteSlackAccount() {
	acct := validSlackAccount()
	acct.WorkspaceID = "TDELETE"
	var deletedID uuid.UUID
	repo := &mockServiceConfigRepo{
		getSlackFn: func(ctx context.Context, id uuid.UUID) (*repository.SlackAccount, error) {
			return acct, nil
		},
		deleteSlackFn: func(ctx context.Context, id uuid.UUID) error {
			deletedID = id
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error { return nil }

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.DeleteSlackAccount(context.Background(), acct.ID)

	s.Require().NoError(err)
	s.Equal(acct.ID, deletedID)
	s.Require().Len(mgr.removeCalls, 1)
	s.Equal("slack:TDELETE", mgr.removeCalls[0])
}

func (s *ServiceSettingsSuite) TestDeleteEmailAccount() {
	acct := validEmailAccount()
	acct.Username = "delete@gmail.com"
	var deletedID uuid.UUID
	repo := &mockServiceConfigRepo{
		getEmailFn: func(ctx context.Context, id uuid.UUID) (*repository.EmailAccount, error) {
			return acct, nil
		},
		deleteEmailFn: func(ctx context.Context, id uuid.UUID) error {
			deletedID = id
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error { return nil }

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.DeleteEmailAccount(context.Background(), acct.ID)

	s.Require().NoError(err)
	s.Equal(acct.ID, deletedID)
	s.Require().Len(mgr.removeCalls, 1)
	s.Equal("email:delete@gmail.com", mgr.removeCalls[0])
}

// --- Toggle enable/disable tests ---

func (s *ServiceSettingsSuite) TestToggleSlackEnable() {
	acct := validSlackAccount()
	acct.Enabled = false
	var upsertedAcct *repository.SlackAccount
	repo := &mockServiceConfigRepo{
		getSlackFn: func(ctx context.Context, id uuid.UUID) (*repository.SlackAccount, error) {
			return acct, nil
		},
		upsertSlackFn: func(ctx context.Context, a *repository.SlackAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	var factoryCalled bool
	factory := func(accountType string, accountID uuid.UUID) error {
		factoryCalled = true
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.ToggleSlackAccount(context.Background(), acct.ID, true)

	s.Require().NoError(err)
	s.True(upsertedAcct.Enabled)
	s.True(factoryCalled)
	s.Empty(mgr.removeCalls)
}

func (s *ServiceSettingsSuite) TestToggleSlackDisable() {
	acct := validSlackAccount()
	acct.Enabled = true
	acct.WorkspaceID = "TDISABLE"
	var upsertedAcct *repository.SlackAccount
	repo := &mockServiceConfigRepo{
		getSlackFn: func(ctx context.Context, id uuid.UUID) (*repository.SlackAccount, error) {
			return acct, nil
		},
		upsertSlackFn: func(ctx context.Context, a *repository.SlackAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error {
		s.Fail("factory should not be called when disabling")
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.ToggleSlackAccount(context.Background(), acct.ID, false)

	s.Require().NoError(err)
	s.False(upsertedAcct.Enabled)
	s.Require().Len(mgr.removeCalls, 1)
	s.Equal("slack:TDISABLE", mgr.removeCalls[0])
}

func (s *ServiceSettingsSuite) TestToggleEmailEnable() {
	acct := validEmailAccount()
	acct.Enabled = false
	var upsertedAcct *repository.EmailAccount
	repo := &mockServiceConfigRepo{
		getEmailFn: func(ctx context.Context, id uuid.UUID) (*repository.EmailAccount, error) {
			return acct, nil
		},
		upsertEmailFn: func(ctx context.Context, a *repository.EmailAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	var factoryCalled bool
	factory := func(accountType string, accountID uuid.UUID) error {
		factoryCalled = true
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.ToggleEmailAccount(context.Background(), acct.ID, true)

	s.Require().NoError(err)
	s.True(upsertedAcct.Enabled)
	s.True(factoryCalled)
	s.Empty(mgr.removeCalls)
}

func (s *ServiceSettingsSuite) TestToggleEmailDisable() {
	acct := validEmailAccount()
	acct.Enabled = true
	acct.Username = "disable@gmail.com"
	var upsertedAcct *repository.EmailAccount
	repo := &mockServiceConfigRepo{
		getEmailFn: func(ctx context.Context, id uuid.UUID) (*repository.EmailAccount, error) {
			return acct, nil
		},
		upsertEmailFn: func(ctx context.Context, a *repository.EmailAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error {
		s.Fail("factory should not be called when disabling")
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.ToggleEmailAccount(context.Background(), acct.ID, false)

	s.Require().NoError(err)
	s.False(upsertedAcct.Enabled)
	s.Require().Len(mgr.removeCalls, 1)
	s.Equal("email:disable@gmail.com", mgr.removeCalls[0])
}

// --- Validation tests ---

func (s *ServiceSettingsSuite) TestValidationSlackEmptyToken() {
	acct := validSlackAccount()
	acct.Token = ""
	repo := &mockServiceConfigRepo{
		upsertSlackFn: func(ctx context.Context, a *repository.SlackAccount) error {
			s.Fail("upsert should not be called on validation failure")
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error {
		s.Fail("factory should not be called on validation failure")
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.SaveSlackAccount(context.Background(), acct)

	s.Error(err)
	s.Contains(err.Error(), "token is required")
}

func (s *ServiceSettingsSuite) TestValidationSlackEmptyWorkspaceID() {
	acct := validSlackAccount()
	acct.WorkspaceID = ""
	repo := &mockServiceConfigRepo{
		upsertSlackFn: func(ctx context.Context, a *repository.SlackAccount) error {
			s.Fail("upsert should not be called on validation failure")
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error {
		s.Fail("factory should not be called on validation failure")
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.SaveSlackAccount(context.Background(), acct)

	s.Error(err)
	s.Contains(err.Error(), "workspace")
}

func (s *ServiceSettingsSuite) TestValidationSlackInvalidPollInterval() {
	acct := validSlackAccount()
	acct.PollIntervalSeconds = 0
	repo := &mockServiceConfigRepo{
		upsertSlackFn: func(ctx context.Context, a *repository.SlackAccount) error {
			s.Fail("upsert should not be called on validation failure")
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error {
		s.Fail("factory should not be called on validation failure")
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.SaveSlackAccount(context.Background(), acct)

	s.Error(err)
	s.Contains(err.Error(), "poll interval")
}

func (s *ServiceSettingsSuite) TestValidationEmailEmptyIMAPHost() {
	acct := validEmailAccount()
	acct.IMAPHost = ""
	repo := &mockServiceConfigRepo{
		upsertEmailFn: func(ctx context.Context, a *repository.EmailAccount) error {
			s.Fail("upsert should not be called on validation failure")
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error {
		s.Fail("factory should not be called on validation failure")
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.SaveEmailAccount(context.Background(), acct)

	s.Error(err)
	s.Contains(err.Error(), "IMAP host")
}

func (s *ServiceSettingsSuite) TestValidationEmailInvalidPort() {
	acct := validEmailAccount()
	acct.IMAPPort = 0
	repo := &mockServiceConfigRepo{
		upsertEmailFn: func(ctx context.Context, a *repository.EmailAccount) error {
			s.Fail("upsert should not be called on validation failure")
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error {
		s.Fail("factory should not be called on validation failure")
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.SaveEmailAccount(context.Background(), acct)

	s.Error(err)
	s.Contains(err.Error(), "IMAP port")
}

func (s *ServiceSettingsSuite) TestValidationEmailEmptyUsername() {
	acct := validEmailAccount()
	acct.Username = ""
	repo := &mockServiceConfigRepo{
		upsertEmailFn: func(ctx context.Context, a *repository.EmailAccount) error {
			s.Fail("upsert should not be called on validation failure")
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error {
		s.Fail("factory should not be called on validation failure")
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.SaveEmailAccount(context.Background(), acct)

	s.Error(err)
	s.Contains(err.Error(), "username")
}

func (s *ServiceSettingsSuite) TestValidationEmailEmptyPassword() {
	acct := validEmailAccount()
	acct.Password = ""
	repo := &mockServiceConfigRepo{
		upsertEmailFn: func(ctx context.Context, a *repository.EmailAccount) error {
			s.Fail("upsert should not be called on validation failure")
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error {
		s.Fail("factory should not be called on validation failure")
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.SaveEmailAccount(context.Background(), acct)

	s.Error(err)
	s.Contains(err.Error(), "password")
}

func (s *ServiceSettingsSuite) TestValidationEmailInvalidPollInterval() {
	acct := validEmailAccount()
	acct.PollIntervalSeconds = 0
	repo := &mockServiceConfigRepo{
		upsertEmailFn: func(ctx context.Context, a *repository.EmailAccount) error {
			s.Fail("upsert should not be called on validation failure")
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error {
		s.Fail("factory should not be called on validation failure")
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.SaveEmailAccount(context.Background(), acct)

	s.Error(err)
	s.Contains(err.Error(), "poll interval")
}

// --- Credential validation tests ---

func (s *ServiceSettingsSuite) TestSaveSlackAccount_ValidationFailure() {
	acct := validSlackAccount()
	validationErr := fmt.Errorf("invalid_auth")
	validator := &mockSlackValidator{
		validateFn: func(ctx context.Context, token string) error {
			s.Equal(acct.Token, token)
			return validationErr
		},
	}
	repo := &mockServiceConfigRepo{
		upsertSlackFn: func(ctx context.Context, a *repository.SlackAccount) error {
			s.Fail("UpsertSlackAccount should not be called when credential validation fails")
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error {
		s.Fail("factory should not be called when credential validation fails")
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory, presenter.WithSlackValidator(validator))
	err := p.SaveSlackAccount(context.Background(), acct)

	s.Error(err)
	s.ErrorIs(err, validationErr)
}

// --- Error propagation tests ---

func (s *ServiceSettingsSuite) TestRepoErrorOnSave() {
	acct := validSlackAccount()
	repo := &mockServiceConfigRepo{
		upsertSlackFn: func(ctx context.Context, a *repository.SlackAccount) error {
			return fmt.Errorf("database connection lost")
		},
	}
	mgr := &mockWatcherManager{}
	factoryCalled := false
	factory := func(accountType string, accountID uuid.UUID) error {
		factoryCalled = true
		return nil
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.SaveSlackAccount(context.Background(), acct)

	s.Error(err)
	s.Contains(err.Error(), "database connection lost")
	s.False(factoryCalled, "factory should NOT be called when repo returns error")
}

func (s *ServiceSettingsSuite) TestFactoryError() {
	acct := validSlackAccount()
	upsertCalled := false
	repo := &mockServiceConfigRepo{
		upsertSlackFn: func(ctx context.Context, a *repository.SlackAccount) error {
			upsertCalled = true
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error {
		return fmt.Errorf("watcher creation failed")
	}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.SaveSlackAccount(context.Background(), acct)

	s.Error(err)
	s.Contains(err.Error(), "watcher creation failed")
	s.True(upsertCalled, "repo upsert should be called even if factory will fail")
}

func (s *ServiceSettingsSuite) TestRepoErrorOnDelete() {
	acct := validSlackAccount()
	acct.WorkspaceID = "TERRORDEL"
	repo := &mockServiceConfigRepo{
		getSlackFn: func(ctx context.Context, id uuid.UUID) (*repository.SlackAccount, error) {
			return acct, nil
		},
		deleteSlackFn: func(ctx context.Context, id uuid.UUID) error {
			return fmt.Errorf("delete failed")
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error { return nil }

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.DeleteSlackAccount(context.Background(), acct.ID)

	s.Error(err)
	s.Contains(err.Error(), "delete failed")
}

// --- Calendar Account Presenter Tests ---

func (s *ServiceSettingsSuite) TestListCalendarAccounts() {
	acct1 := validCalendarAccount()
	acct2 := validCalendarAccount()
	repo := &mockServiceConfigRepo{
		listCalendarFn: func(ctx context.Context) ([]*repository.CalendarAccount, error) {
			return []*repository.CalendarAccount{acct1, acct2}, nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error { return nil }

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	accounts, err := p.ListCalendarAccounts(context.Background())

	s.Require().NoError(err)
	s.Len(accounts, 2)
	s.Equal(acct1.ID, accounts[0].ID)
	s.Equal(acct2.ID, accounts[1].ID)
}

func (s *ServiceSettingsSuite) TestListCalendarAccountsEmpty() {
	repo := &mockServiceConfigRepo{
		listCalendarFn: func(ctx context.Context) ([]*repository.CalendarAccount, error) {
			return []*repository.CalendarAccount{}, nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error { return nil }

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	accounts, err := p.ListCalendarAccounts(context.Background())

	s.Require().NoError(err)
	s.Empty(accounts)
}

func (s *ServiceSettingsSuite) TestSaveNewCalendarAccount() {
	acct := validCalendarAccount()
	var upsertedAcct *repository.CalendarAccount
	repo := &mockServiceConfigRepo{
		upsertCalendarFn: func(ctx context.Context, a *repository.CalendarAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error { return nil }

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.SaveCalendarAccount(context.Background(), acct)

	s.Require().NoError(err)
	s.Equal(acct, upsertedAcct)
}

func (s *ServiceSettingsSuite) TestEditCalendarAccount() {
	acct := validCalendarAccount()
	acct.Name = "Updated Calendar"
	var upsertedAcct *repository.CalendarAccount
	repo := &mockServiceConfigRepo{
		upsertCalendarFn: func(ctx context.Context, a *repository.CalendarAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error { return nil }

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.EditCalendarAccount(context.Background(), acct, "Old Calendar")

	s.Require().NoError(err)
	s.Equal(acct, upsertedAcct)
}

func (s *ServiceSettingsSuite) TestDeleteCalendarAccount() {
	acct := validCalendarAccount()
	var deletedID uuid.UUID
	repo := &mockServiceConfigRepo{
		getCalendarFn: func(ctx context.Context, id uuid.UUID) (*repository.CalendarAccount, error) {
			return acct, nil
		},
		deleteCalendarFn: func(ctx context.Context, id uuid.UUID) error {
			deletedID = id
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error { return nil }

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.DeleteCalendarAccount(context.Background(), acct.ID)

	s.Require().NoError(err)
	s.Equal(acct.ID, deletedID)
}

func (s *ServiceSettingsSuite) TestToggleCalendarEnable() {
	acct := validCalendarAccount()
	acct.Enabled = false
	var upsertedAcct *repository.CalendarAccount
	repo := &mockServiceConfigRepo{
		getCalendarFn: func(ctx context.Context, id uuid.UUID) (*repository.CalendarAccount, error) {
			return acct, nil
		},
		upsertCalendarFn: func(ctx context.Context, a *repository.CalendarAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error { return nil }

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.ToggleCalendarAccount(context.Background(), acct.ID, true)

	s.Require().NoError(err)
	s.True(upsertedAcct.Enabled)
}

func (s *ServiceSettingsSuite) TestToggleCalendarDisable() {
	acct := validCalendarAccount()
	acct.Enabled = true
	var upsertedAcct *repository.CalendarAccount
	repo := &mockServiceConfigRepo{
		getCalendarFn: func(ctx context.Context, id uuid.UUID) (*repository.CalendarAccount, error) {
			return acct, nil
		},
		upsertCalendarFn: func(ctx context.Context, a *repository.CalendarAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}
	factory := func(accountType string, accountID uuid.UUID) error { return nil }

	p := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	err := p.ToggleCalendarAccount(context.Background(), acct.ID, false)

	s.Require().NoError(err)
	s.False(upsertedAcct.Enabled)
}
