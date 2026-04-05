package ui_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Stubs for SettingsView dependencies ---

type stubVolumeController struct{}

func (s *stubVolumeController) SetVolume(_ int) {}

type stubServiceConfigRepo struct{}

func (s *stubServiceConfigRepo) ListSlackAccounts(_ context.Context) ([]*repository.SlackAccount, error) {
	return nil, nil
}
func (s *stubServiceConfigRepo) GetSlackAccount(_ context.Context, _ uuid.UUID) (*repository.SlackAccount, error) {
	return nil, nil
}
func (s *stubServiceConfigRepo) UpsertSlackAccount(_ context.Context, _ *repository.SlackAccount) error {
	return nil
}
func (s *stubServiceConfigRepo) DeleteSlackAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (s *stubServiceConfigRepo) ListEmailAccounts(_ context.Context) ([]*repository.EmailAccount, error) {
	return nil, nil
}
func (s *stubServiceConfigRepo) GetEmailAccount(_ context.Context, _ uuid.UUID) (*repository.EmailAccount, error) {
	return nil, nil
}
func (s *stubServiceConfigRepo) UpsertEmailAccount(_ context.Context, _ *repository.EmailAccount) error {
	return nil
}
func (s *stubServiceConfigRepo) DeleteEmailAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (s *stubServiceConfigRepo) ListCalendarAccounts(_ context.Context) ([]*repository.CalendarAccount, error) {
	return nil, nil
}
func (s *stubServiceConfigRepo) GetCalendarAccount(_ context.Context, _ uuid.UUID) (*repository.CalendarAccount, error) {
	return nil, nil
}
func (s *stubServiceConfigRepo) UpsertCalendarAccount(_ context.Context, _ *repository.CalendarAccount) error {
	return nil
}
func (s *stubServiceConfigRepo) DeleteCalendarAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}

type stubWatcherRemover struct{}

func (s *stubWatcherRemover) RemoveWatcher(_ string) {}

// --- SettingsViewSuite ---

type SettingsViewSuite struct {
	suite.Suite
	sp        *presenter.SettingsPresenter
	ssp       *presenter.ServiceSettingsPresenter
	ollamaCfg config.OllamaConfig
}

func TestSettingsView(t *testing.T) {
	suite.Run(t, new(SettingsViewSuite))
}

func (s *SettingsViewSuite) SetupTest() {
	vc := &stubVolumeController{}
	sp, err := presenter.NewSettingsPresenter(vc, 50)
	s.Require().NoError(err)
	s.sp = sp

	repo := &stubServiceConfigRepo{}
	mgr := &stubWatcherRemover{}
	factory := func(_ string, _ uuid.UUID) error { return nil }
	s.ssp = presenter.NewServiceSettingsPresenter(repo, mgr, factory)

	s.ollamaCfg = config.OllamaConfig{}
}

func (s *SettingsViewSuite) TestNewSettingsViewReturnsNonNil() {
	sv := ui.NewSettingsView(s.sp, s.ssp, s.ollamaCfg, func() {})

	s.NotNil(sv, "NewSettingsView should return a non-nil SettingsView")
}

func (s *SettingsViewSuite) TestSettingsViewContainerReturnsNonNil() {
	sv := ui.NewSettingsView(s.sp, s.ssp, s.ollamaCfg, func() {})

	container := sv.Container()

	s.NotNil(container, "Container() should return a non-nil fyne.CanvasObject")
}

func (s *SettingsViewSuite) TestSettingsViewHasFourTabs() {
	sv := ui.NewSettingsView(s.sp, s.ssp, s.ollamaCfg, func() {})

	tabs := sv.TabCount()

	s.Equal(4, tabs, "SettingsView should have exactly 4 tabs")
}

func (s *SettingsViewSuite) TestSettingsViewTabNames() {
	sv := ui.NewSettingsView(s.sp, s.ssp, s.ollamaCfg, func() {})

	expected := []string{"Slack", "Email", "Audio", "Ollama"}
	names := sv.TabNames()

	s.Equal(expected, names,
		"tab names should be Slack, Email, Audio, Ollama in that order")
}
