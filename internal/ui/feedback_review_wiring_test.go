package ui_test

import (
	"context"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// stubMessageQuerier satisfies presenter.MessageQuerier for wiring tests.
type stubMessageQuerier struct{}

func (s *stubMessageQuerier) QueryByStatus(_ context.Context, _ string) ([]*repository.Message, error) {
	return nil, nil
}

// stubMessageUpdater satisfies presenter.MessageUpdater for wiring tests.
type stubMessageUpdater struct{}

func (s *stubMessageUpdater) Update(_ context.Context, _ *repository.Message) error {
	return nil
}

// stubBufferReviewer is a minimal mock satisfying presenter.BufferReviewer
// for wiring tests that need a real *presenter.FeedbackPresenter.
type stubBufferReviewer struct{}

func (s *stubBufferReviewer) GetBufferedMessages(_ context.Context) ([]*repository.Message, error) {
	return nil, nil
}

func (s *stubBufferReviewer) CountBuffered(_ context.Context) (int, error) {
	return 0, nil
}

func (s *stubBufferReviewer) SaveRating(_ context.Context, _ uuid.UUID, _ int, _ *string) error {
	return nil
}

func (s *stubBufferReviewer) DeleteMessage(_ context.Context, _ uuid.UUID) error {
	return nil
}

// FeedbackReviewWiringSuite tests that the MainWindow wires the focus rail's
// Review button callback when a FeedbackPresenter is provided.
type FeedbackReviewWiringSuite struct {
	suite.Suite
}

func TestFeedbackReviewWiring(t *testing.T) {
	suite.Run(t, new(FeedbackReviewWiringSuite))
}

func (s *FeedbackReviewWiringSuite) TestReviewButtonCallbackWiredWhenFeedbackPresenterProvided() {
	app := test.NewApp()
	defer app.Quit()

	router := ui.NewCenterViewRouter()

	fp, err := presenter.NewFeedbackPresenter(&stubBufferReviewer{})
	s.Require().NoError(err)

	cfg := config.GUIConfig{
		WindowWidth:  1200,
		WindowHeight: 800,
	}

	mw := ui.NewMainWindow(
		app,
		cfg,
		(*presenter.NotificationPresenter)(nil),
		(*presenter.ActivityPresenter)(nil),
		fp,
		(*presenter.AppPresenter)(nil),
		(*presenter.SettingsPresenter)(nil),
		(*presenter.ServiceSettingsPresenter)(nil),
		config.OllamaConfig{},
		nil, // characterWidget
		router,
		nil, // plannerVM
		nil, // timerVM
		nil, // wizardVM
		nil, // rightPanelOverride
	)

	s.Require().NotNil(mw.FocusRail(), "FocusRail() should return the focus rail instance")
	s.NotNil(mw.FocusRail().ReviewButton().OnTapped,
		"Review button should have a callback wired when FeedbackPresenter is provided")
}

func (s *FeedbackReviewWiringSuite) TestReviewButtonVisibleWhenNotificationsExpanded() {
	app := test.NewApp()
	defer app.Quit()

	router := ui.NewCenterViewRouter()

	np, err := presenter.NewNotificationPresenter(&stubMessageQuerier{}, &stubMessageUpdater{})
	s.Require().NoError(err)

	fp, err := presenter.NewFeedbackPresenter(&stubBufferReviewer{})
	s.Require().NoError(err)

	cfg := config.GUIConfig{
		WindowWidth:  1200,
		WindowHeight: 800,
	}

	mw := ui.NewMainWindow(
		app,
		cfg,
		np,
		(*presenter.ActivityPresenter)(nil),
		fp,
		(*presenter.AppPresenter)(nil),
		(*presenter.SettingsPresenter)(nil),
		(*presenter.ServiceSettingsPresenter)(nil),
		config.OllamaConfig{},
		nil, // characterWidget
		router,
		nil, // plannerVM
		nil, // timerVM
		nil, // wizardVM
		nil, // rightPanelOverride
	)

	// Review button should be hidden by default.
	s.Require().NotNil(mw.FocusRail())
	s.False(mw.FocusRail().ReviewButton().Visible(),
		"Review button should be hidden before notifications are expanded")

	// Expand notifications — review button should become visible.
	np.ToggleExpanded()

	s.True(mw.FocusRail().ReviewButton().Visible(),
		"Review button should be visible after notifications are expanded")
}

func (s *FeedbackReviewWiringSuite) TestReviewButtonHiddenWhenNotificationsCollapsed() {
	app := test.NewApp()
	defer app.Quit()

	router := ui.NewCenterViewRouter()

	np, err := presenter.NewNotificationPresenter(&stubMessageQuerier{}, &stubMessageUpdater{})
	s.Require().NoError(err)

	fp, err := presenter.NewFeedbackPresenter(&stubBufferReviewer{})
	s.Require().NoError(err)

	cfg := config.GUIConfig{
		WindowWidth:  1200,
		WindowHeight: 800,
	}

	mw := ui.NewMainWindow(
		app,
		cfg,
		np,
		(*presenter.ActivityPresenter)(nil),
		fp,
		(*presenter.AppPresenter)(nil),
		(*presenter.SettingsPresenter)(nil),
		(*presenter.ServiceSettingsPresenter)(nil),
		config.OllamaConfig{},
		nil, // characterWidget
		router,
		nil, // plannerVM
		nil, // timerVM
		nil, // wizardVM
		nil, // rightPanelOverride
	)

	s.Require().NotNil(mw.FocusRail())

	// Expand then collapse — review button should hide again.
	np.ToggleExpanded()
	np.ToggleExpanded()

	s.False(mw.FocusRail().ReviewButton().Visible(),
		"Review button should be hidden after notifications are collapsed")
}
