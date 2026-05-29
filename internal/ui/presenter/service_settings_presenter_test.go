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

// --- Mock AccountWatcherToggler ---

type togglerCall struct {
	kind    string // "slack" | "email" | "calendar"
	id      uuid.UUID
	enabled bool
}

type mockWatcherManager struct {
	calls []togglerCall
	// err is returned from every Set call when non-nil. Tests can set
	// it to verify error paths through the presenter.
	err error
}

func (m *mockWatcherManager) SetSlackEnabled(_ context.Context, id uuid.UUID, enabled bool) error {
	m.calls = append(m.calls, togglerCall{kind: "slack", id: id, enabled: enabled})
	return m.err
}

func (m *mockWatcherManager) SetEmailEnabled(_ context.Context, id uuid.UUID, enabled bool) error {
	m.calls = append(m.calls, togglerCall{kind: "email", id: id, enabled: enabled})
	return m.err
}

func (m *mockWatcherManager) SetCalendarEnabled(_ context.Context, id uuid.UUID, enabled bool) error {
	m.calls = append(m.calls, togglerCall{kind: "calendar", id: id, enabled: enabled})
	return m.err
}

// addCalls / removeCalls return the legacy projections expected by the
// existing assertions. The naming is preserved for diff economy:
// addCalls is "starts" (enabled=true) and removeCalls is "stops"
// (enabled=false). Both return strings of the form "<kind>:<id>" so
// existing tests that compared natural keys ("slack:T12345") still
// have a check they can perform on the recorded ID.
func (m *mockWatcherManager) addCalls() []togglerCall {
	out := make([]togglerCall, 0, len(m.calls))
	for _, c := range m.calls {
		if c.enabled {
			out = append(out, c)
		}
	}
	return out
}

func (m *mockWatcherManager) removeCalls() []togglerCall {
	out := make([]togglerCall, 0, len(m.calls))
	for _, c := range m.calls {
		if !c.enabled {
			out = append(out, c)
		}
	}
	return out
}

// --- Mock SlackValidator ---

type mockSlackValidator struct {
	validateFn func(ctx context.Context, token string) error
}

func (m *mockSlackValidator) ValidateSlack(ctx context.Context, token string) error {
	return m.validateFn(ctx, token)
}

// --- Mock EmailValidator ---

type mockEmailValidator struct {
	validateFn func(ctx context.Context, host string, port int, username, password, encryption string) error
}

func (m *mockEmailValidator) ValidateEmail(ctx context.Context, host string, port int, username, password, encryption string) error {
	return m.validateFn(ctx, host, port, username, password, encryption)
}

// --- Mock CalendarValidator ---

type mockCalendarValidator struct {
	validateFn func(ctx context.Context, url string) error
}

func (m *mockCalendarValidator) ValidateCalendar(ctx context.Context, url string) error {
	return m.validateFn(ctx, url)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.SaveSlackAccount(context.Background(), acct)

	s.Require().NoError(err)
	s.Equal(acct, upsertedAcct)
	s.Require().Len(mgr.addCalls(), 1, "Save must start the watcher")
	s.Equal("slack", mgr.addCalls()[0].kind)
	s.Equal(acct.ID, mgr.addCalls()[0].id)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.SaveEmailAccount(context.Background(), acct)

	s.Require().NoError(err)
	s.Equal(acct, upsertedAcct)
	s.Require().Len(mgr.addCalls(), 1)
	s.Equal("email", mgr.addCalls()[0].kind)
	s.Equal(acct.ID, mgr.addCalls()[0].id)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.EditSlackAccount(context.Background(), acct, "T00001")

	s.Require().NoError(err)
	s.Equal(acct, upsertedAcct)
	// Edit re-asserts the watcher state via SetSlackEnabled(true);
	// the legacy "stop old watcher first" step is server-side now.
	s.Require().Len(mgr.addCalls(), 1)
	s.Equal("slack", mgr.addCalls()[0].kind)
	s.Equal(acct.ID, mgr.addCalls()[0].id)
	s.Empty(mgr.removeCalls())
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.EditEmailAccount(context.Background(), acct, "old@gmail.com")

	s.Require().NoError(err)
	s.Equal(acct, upsertedAcct)
	s.Require().Len(mgr.addCalls(), 1)
	s.Equal("email", mgr.addCalls()[0].kind)
	s.Equal(acct.ID, mgr.addCalls()[0].id)
	s.Empty(mgr.removeCalls())
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.DeleteSlackAccount(context.Background(), acct.ID)

	s.Require().NoError(err)
	s.Equal(acct.ID, deletedID)
	s.Require().Len(mgr.removeCalls(), 1, "Delete must stop the watcher")
	s.Equal("slack", mgr.removeCalls()[0].kind)
	s.Equal(acct.ID, mgr.removeCalls()[0].id)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.DeleteEmailAccount(context.Background(), acct.ID)

	s.Require().NoError(err)
	s.Equal(acct.ID, deletedID)
	s.Require().Len(mgr.removeCalls(), 1)
	s.Equal("email", mgr.removeCalls()[0].kind)
	s.Equal(acct.ID, mgr.removeCalls()[0].id)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.ToggleSlackAccount(context.Background(), acct.ID, true)

	s.Require().NoError(err)
	s.True(upsertedAcct.Enabled)
	s.Require().Len(mgr.addCalls(), 1)
	s.Equal("slack", mgr.addCalls()[0].kind)
	s.Equal(acct.ID, mgr.addCalls()[0].id)
	s.Empty(mgr.removeCalls())
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.ToggleSlackAccount(context.Background(), acct.ID, false)

	s.Require().NoError(err)
	s.False(upsertedAcct.Enabled)
	s.Require().Len(mgr.removeCalls(), 1)
	s.Equal("slack", mgr.removeCalls()[0].kind)
	s.Equal(acct.ID, mgr.removeCalls()[0].id)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.ToggleEmailAccount(context.Background(), acct.ID, true)

	s.Require().NoError(err)
	s.True(upsertedAcct.Enabled)
	s.Require().Len(mgr.addCalls(), 1)
	s.Equal("email", mgr.addCalls()[0].kind)
	s.Equal(acct.ID, mgr.addCalls()[0].id)
	s.Empty(mgr.removeCalls())
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.ToggleEmailAccount(context.Background(), acct.ID, false)

	s.Require().NoError(err)
	s.False(upsertedAcct.Enabled)
	s.Require().Len(mgr.removeCalls(), 1)
	s.Equal("email", mgr.removeCalls()[0].kind)
	s.Equal(acct.ID, mgr.removeCalls()[0].id)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.SaveSlackAccount(context.Background(), acct)

	s.Error(err)
	s.Contains(err.Error(), "workspace")
}

func (s *ServiceSettingsSuite) TestValidationSlackInvalidPollInterval() {
	acct := validSlackAccount()
	acct.PollIntervalSeconds = -1
	repo := &mockServiceConfigRepo{
		upsertSlackFn: func(ctx context.Context, a *repository.SlackAccount) error {
			s.Fail("upsert should not be called on validation failure")
			return nil
		},
	}
	mgr := &mockWatcherManager{}

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.SaveEmailAccount(context.Background(), acct)

	s.Error(err)
	s.Contains(err.Error(), "password")
}

func (s *ServiceSettingsSuite) TestValidationEmailInvalidPollInterval() {
	acct := validEmailAccount()
	acct.PollIntervalSeconds = -1
	repo := &mockServiceConfigRepo{
		upsertEmailFn: func(ctx context.Context, a *repository.EmailAccount) error {
			s.Fail("upsert should not be called on validation failure")
			return nil
		},
	}
	mgr := &mockWatcherManager{}

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr, presenter.WithSlackValidator(validator))
	err := p.SaveSlackAccount(context.Background(), acct)

	s.Error(err)
	s.ErrorIs(err, validationErr)
}

func (s *ServiceSettingsSuite) TestSaveEmailAccount_ValidationFailure() {
	acct := validEmailAccount()
	validationErr := fmt.Errorf("IMAP connection refused")
	validator := &mockEmailValidator{
		validateFn: func(ctx context.Context, host string, port int, username, password, encryption string) error {
			s.Equal(acct.IMAPHost, host)
			s.Equal(acct.IMAPPort, port)
			s.Equal(acct.Username, username)
			s.Equal(acct.Password, password)
			s.Equal(acct.Encryption, encryption)
			return validationErr
		},
	}
	repo := &mockServiceConfigRepo{
		upsertEmailFn: func(ctx context.Context, a *repository.EmailAccount) error {
			s.Fail("UpsertEmailAccount should not be called when credential validation fails")
			return nil
		},
	}
	mgr := &mockWatcherManager{}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, presenter.WithEmailValidator(validator))
	err := p.SaveEmailAccount(context.Background(), acct)

	s.Error(err)
	s.ErrorIs(err, validationErr)
}

func (s *ServiceSettingsSuite) TestSaveCalendarAccount_ValidationFailure() {
	acct := validCalendarAccount()
	validationErr := fmt.Errorf("calendar unreachable")
	validator := &mockCalendarValidator{
		validateFn: func(ctx context.Context, url string) error {
			s.Equal(acct.ICSURL, url)
			return validationErr
		},
	}
	repo := &mockServiceConfigRepo{
		upsertCalendarFn: func(ctx context.Context, a *repository.CalendarAccount) error {
			s.Fail("UpsertCalendarAccount should not be called when credential validation fails")
			return nil
		},
	}
	mgr := &mockWatcherManager{}

	p := presenter.NewServiceSettingsPresenter(repo, mgr, presenter.WithCalendarValidator(validator))
	err := p.SaveCalendarAccount(context.Background(), acct)

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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.SaveSlackAccount(context.Background(), acct)

	s.Error(err)
	s.Contains(err.Error(), "database connection lost")
	s.False(factoryCalled, "factory should NOT be called when repo returns error")
}

func (s *ServiceSettingsSuite) TestTogglerError() {
	acct := validSlackAccount()
	upsertCalled := false
	repo := &mockServiceConfigRepo{
		upsertSlackFn: func(ctx context.Context, a *repository.SlackAccount) error {
			upsertCalled = true
			return nil
		},
	}
	mgr := &mockWatcherManager{err: fmt.Errorf("watcher creation failed")}

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.SaveSlackAccount(context.Background(), acct)

	s.Error(err)
	s.Contains(err.Error(), "watcher creation failed")
	s.True(upsertCalled, "repo upsert must be called even if the toggler will fail")
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
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

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.ToggleCalendarAccount(context.Background(), acct.ID, false)

	s.Require().NoError(err)
	s.False(upsertedAcct.Enabled)
}

// --- Default poll interval tests ---

func (s *ServiceSettingsSuite) TestDefaultPollIntervalConstants() {
	s.Equal(60, presenter.DefaultSlackPollInterval, "Slack default poll interval should be 60 seconds")
	s.Equal(600, presenter.DefaultEmailPollInterval, "Email default poll interval should be 600 seconds")
	s.Equal(600, presenter.DefaultCalendarPollInterval, "Calendar default poll interval should be 600 seconds")
}

func (s *ServiceSettingsSuite) TestSaveSlackAccountAppliesDefaultPollInterval() {
	acct := validSlackAccount()
	acct.PollIntervalSeconds = 0 // zero means "use default"

	var upsertedAcct *repository.SlackAccount
	repo := &mockServiceConfigRepo{
		upsertSlackFn: func(ctx context.Context, a *repository.SlackAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.SaveSlackAccount(context.Background(), acct)

	s.Require().NoError(err, "SaveSlackAccount should not error when PollIntervalSeconds is 0")
	s.Require().NotNil(upsertedAcct, "upsert should have been called")
	s.Equal(60, upsertedAcct.PollIntervalSeconds, "PollIntervalSeconds should be set to Slack default (60)")
}

func (s *ServiceSettingsSuite) TestSaveEmailAccountAppliesDefaultPollInterval() {
	acct := validEmailAccount()
	acct.PollIntervalSeconds = 0 // zero means "use default"

	var upsertedAcct *repository.EmailAccount
	repo := &mockServiceConfigRepo{
		upsertEmailFn: func(ctx context.Context, a *repository.EmailAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.SaveEmailAccount(context.Background(), acct)

	s.Require().NoError(err, "SaveEmailAccount should not error when PollIntervalSeconds is 0")
	s.Require().NotNil(upsertedAcct, "upsert should have been called")
	s.Equal(600, upsertedAcct.PollIntervalSeconds, "PollIntervalSeconds should be set to Email default (600)")
}

func (s *ServiceSettingsSuite) TestSaveCalendarAccountAppliesDefaultPollInterval() {
	acct := &repository.CalendarAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Name:                "Test Cal",
		ICSURL:              "https://example.com/cal.ics",
		PollIntervalSeconds: 0, // zero means "use default"
	}

	var upsertedAcct *repository.CalendarAccount
	repo := &mockServiceConfigRepo{
		upsertCalendarFn: func(ctx context.Context, a *repository.CalendarAccount) error {
			upsertedAcct = a
			return nil
		},
	}
	mgr := &mockWatcherManager{}

	p := presenter.NewServiceSettingsPresenter(repo, mgr)
	err := p.SaveCalendarAccount(context.Background(), acct)

	s.Require().NoError(err, "SaveCalendarAccount should not error when PollIntervalSeconds is 0")
	s.Require().NotNil(upsertedAcct, "upsert should have been called")
	s.Equal(600, upsertedAcct.PollIntervalSeconds, "PollIntervalSeconds should be set to Calendar default (600)")
}

func (s *ServiceSettingsSuite) TestDefaultPollIntervalReturnsCorrectDefaults() {
	s.Equal(60, presenter.DefaultPollInterval("slack"), "slack should default to 60s")
	s.Equal(600, presenter.DefaultPollInterval("email"), "email should default to 600s")
	s.Equal(600, presenter.DefaultPollInterval("calendar"), "calendar should default to 600s")
	s.Equal(600, presenter.DefaultPollInterval("unknown"), "unknown service type should fall back to 600s")
}
