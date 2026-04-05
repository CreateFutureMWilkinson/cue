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

// newTestMainWindow is a helper that creates a MainWindow with a given
// CenterViewRouter and nil presenters (sufficient for layout tests).
func newTestMainWindow(router *ui.CenterViewRouter) *ui.MainWindow {
	cfg := config.GUIConfig{
		WindowWidth:  1200,
		WindowHeight: 800,
	}
	return ui.NewMainWindow(
		cfg,
		(*presenter.NotificationPresenter)(nil),
		(*presenter.ActivityPresenter)(nil),
		(*presenter.FeedbackPresenter)(nil),
		(*presenter.AppPresenter)(nil),
		(*presenter.SettingsPresenter)(nil),
		nil, // characterWidget
		router,
	)
}

func (s *ThreeColumnLayoutSuite) TestNewMainWindowAcceptsCenterViewRouter() {
	// This test verifies the NewMainWindow signature accepts a *CenterViewRouter.
	// It is a compile-time contract test — if the parameter is missing, this
	// file will not compile.
	//
	// We pass nil for all presenter dependencies because we are only testing
	// that the function signature is correct, not that it produces a working
	// window. The function should still return a non-nil *MainWindow.
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(router)
	s.NotNil(mw, "NewMainWindow should return a non-nil *MainWindow")
}

func (s *ThreeColumnLayoutSuite) TestCenterViewDefaultsToCharacterContent() {
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(router)

	content := mw.CenterContent()
	s.NotNil(content, "CenterContent should return a non-nil canvas object on startup")

	// The router should default to ViewCharacter.
	s.Equal(ui.ViewCharacter, router.CurrentView(),
		"CenterViewRouter should default to ViewCharacter")
}

func (s *ThreeColumnLayoutSuite) TestNavigateToPlanSwapsCenterContent() {
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(router)

	originalContent := mw.CenterContent()
	s.NotNil(originalContent)

	router.NavigateTo(ui.ViewPlan)

	newContent := mw.CenterContent()
	s.NotNil(newContent, "CenterContent should not be nil after navigating to ViewPlan")
	s.NotEqual(originalContent, newContent,
		"Navigating to ViewPlan should swap the center pane to different content")
}

func (s *ThreeColumnLayoutSuite) TestNavigateToWizardSwapsCenterContent() {
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(router)

	originalContent := mw.CenterContent()
	s.NotNil(originalContent)

	router.NavigateTo(ui.ViewWizard)

	newContent := mw.CenterContent()
	s.NotNil(newContent, "CenterContent should not be nil after navigating to ViewWizard")
	s.NotEqual(originalContent, newContent,
		"Navigating to ViewWizard should swap the center pane to different content")
}

func (s *ThreeColumnLayoutSuite) TestNavigateToSettingsSwapsCenterContent() {
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(router)

	originalContent := mw.CenterContent()
	s.NotNil(originalContent)

	router.NavigateTo(ui.ViewSettings)

	newContent := mw.CenterContent()
	s.NotNil(newContent, "CenterContent should not be nil after navigating to ViewSettings")
	s.NotEqual(originalContent, newContent,
		"Navigating to ViewSettings should swap the center pane to different content")
}

func (s *ThreeColumnLayoutSuite) TestNavigateBackToCharacterRestoresContent() {
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(router)

	characterContent := mw.CenterContent()
	s.NotNil(characterContent)

	// Navigate away to Plan view.
	router.NavigateTo(ui.ViewPlan)
	planContent := mw.CenterContent()
	s.NotEqual(characterContent, planContent,
		"Plan content should differ from character content")

	// Navigate back to Character view.
	router.NavigateTo(ui.ViewCharacter)
	restoredContent := mw.CenterContent()
	s.NotNil(restoredContent, "CenterContent should not be nil after returning to ViewCharacter")
	s.NotEqual(planContent, restoredContent,
		"Returning to ViewCharacter should swap away from plan content")
}
