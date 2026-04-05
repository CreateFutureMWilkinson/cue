package ui_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// ThreeColumnLayoutSuite tests that the MainWindow API accepts the
// CenterViewRouter parameter required by the three-column layout.
type ThreeColumnLayoutSuite struct {
	suite.Suite
}

func TestThreeColumnLayout(t *testing.T) {
	suite.Run(t, new(ThreeColumnLayoutSuite))
}

func (s *ThreeColumnLayoutSuite) TestNewMainWindowAcceptsCenterViewRouter() {
	// This test verifies the NewMainWindow signature accepts a *CenterViewRouter.
	// It is a compile-time contract test — if the parameter is missing, this
	// file will not compile.
	//
	// We pass nil for all presenter dependencies because we are only testing
	// that the function signature is correct, not that it produces a working
	// window. The function should still return a non-nil *MainWindow.
	cfg := config.GUIConfig{
		WindowWidth:  1200,
		WindowHeight: 800,
	}

	router := ui.NewCenterViewRouter()

	// The new signature must accept *CenterViewRouter as a parameter.
	// The exact position in the parameter list is an implementation detail,
	// but the call must compile with the router argument.
	mw := ui.NewMainWindow(
		cfg,
		(*presenter.NotificationPresenter)(nil),
		(*presenter.ActivityPresenter)(nil),
		(*presenter.FeedbackPresenter)(nil),
		(*presenter.AppPresenter)(nil),
		(*presenter.SettingsPresenter)(nil),
		nil, // characterWidget
		router,
	)

	s.NotNil(mw, "NewMainWindow should return a non-nil *MainWindow")
}
