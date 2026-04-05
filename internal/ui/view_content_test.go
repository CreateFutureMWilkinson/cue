package ui_test

import (
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// ViewContentSuite verifies that each center view (Character, Plan, Wizard,
// Settings) exposes the correct child widget when active.
type ViewContentSuite struct {
	suite.Suite
}

func TestViewContent(t *testing.T) {
	suite.Run(t, new(ViewContentSuite))
}

// TestCharacterViewContentIsNotNil verifies the default center content
// (ViewCharacter) is not nil.
func (s *ViewContentSuite) TestCharacterViewContentIsNotNil() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	content := mw.CenterContent()
	s.NotNil(content, "CenterContent() for ViewCharacter should not be nil")
}

// TestPlanViewContentIsPlaceholderLabel verifies navigating to ViewPlan
// produces a *widget.Label with text "Plan".
func (s *ViewContentSuite) TestPlanViewContentIsPlaceholderLabel() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	router.NavigateTo(ui.ViewPlan)

	content := mw.CenterContent()
	s.Require().NotNil(content, "CenterContent() for ViewPlan should not be nil")

	lbl, ok := content.(*widget.Label)
	s.Require().True(ok, "ViewPlan content should be *widget.Label, got %T", content)
	s.Equal("Plan", lbl.Text)
}

// TestWizardViewContentIsPlaceholderLabel verifies navigating to ViewWizard
// produces a *widget.Label with text "Wizard".
func (s *ViewContentSuite) TestWizardViewContentIsPlaceholderLabel() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	router.NavigateTo(ui.ViewWizard)

	content := mw.CenterContent()
	s.Require().NotNil(content, "CenterContent() for ViewWizard should not be nil")

	lbl, ok := content.(*widget.Label)
	s.Require().True(ok, "ViewWizard content should be *widget.Label, got %T", content)
	s.Equal("Wizard", lbl.Text)
}

// TestSettingsViewWithNilPresentersIsPlaceholderLabel verifies that when
// SettingsPresenter and ServiceSettingsPresenter are nil, ViewSettings
// falls back to a *widget.Label with text "Settings".
func (s *ViewContentSuite) TestSettingsViewWithNilPresentersIsPlaceholderLabel() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()
	mw := newTestMainWindow(fyneApp, router)

	router.NavigateTo(ui.ViewSettings)

	content := mw.CenterContent()
	s.Require().NotNil(content, "CenterContent() for ViewSettings should not be nil")

	lbl, ok := content.(*widget.Label)
	s.Require().True(ok, "ViewSettings content with nil presenters should be *widget.Label, got %T", content)
	s.Equal("Settings", lbl.Text)
}

// TestSettingsViewWithPresentersContainsTabs verifies that when real
// SettingsPresenter and ServiceSettingsPresenter are provided, the
// ViewSettings content is a *container.AppTabs with 4 tabs.
func (s *ViewContentSuite) TestSettingsViewWithPresentersContainsTabs() {
	fyneApp := test.NewApp()
	router := ui.NewCenterViewRouter()

	// Create a SettingsPresenter with a mock volume controller.
	sp, err := presenter.NewSettingsPresenter(&stubVolumeController{}, 50)
	s.Require().NoError(err)

	// Create a ServiceSettingsPresenter with nil deps — we only need the
	// pointer to be non-nil so the window constructs a SettingsView.
	ssp := presenter.NewServiceSettingsPresenter(nil, nil, nil)

	cfg := config.GUIConfig{
		WindowWidth:  1200,
		WindowHeight: 800,
	}
	mw := ui.NewMainWindow(
		fyneApp,
		cfg,
		(*presenter.NotificationPresenter)(nil),
		(*presenter.ActivityPresenter)(nil),
		(*presenter.FeedbackPresenter)(nil),
		(*presenter.AppPresenter)(nil),
		sp,
		ssp,
		config.OllamaConfig{},
		nil, // characterWidget
		router,
	)

	router.NavigateTo(ui.ViewSettings)

	content := mw.CenterContent()
	s.Require().NotNil(content, "CenterContent() for ViewSettings with presenters should not be nil")

	tabs, ok := content.(*container.AppTabs)
	s.Require().True(ok, "ViewSettings content with presenters should be *container.AppTabs, got %T", content)
	s.Equal(4, len(tabs.Items), "SettingsView should have 4 tabs")

	expectedNames := []string{"Slack", "Email", "Audio", "Ollama"}
	for i, item := range tabs.Items {
		s.Equal(expectedNames[i], item.Text, "tab %d name mismatch", i)
	}
}
