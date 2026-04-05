package ui_test

import (
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// --- SettingsInteractionSuite ---

type SettingsInteractionSuite struct {
	suite.Suite
	sv *ui.SettingsView
}

func TestSettingsInteraction(t *testing.T) {
	suite.Run(t, new(SettingsInteractionSuite))
}

func (s *SettingsInteractionSuite) SetupTest() {
	vc := &stubVolumeController{}
	sp, err := presenter.NewSettingsPresenter(vc, 50)
	s.Require().NoError(err)

	repo := &stubServiceConfigRepo{}
	mgr := &stubWatcherRemover{}
	factory := func(_ string, _ uuid.UUID) error { return nil }
	ssp := presenter.NewServiceSettingsPresenter(repo, mgr, factory)

	s.sv = ui.NewSettingsView(sp, ssp, config.OllamaConfig{})
}

func (s *SettingsInteractionSuite) TestSettingsViewContainsAppTabs() {
	root := s.sv.Container()

	tabs, found := uitest.FindWidget[*container.AppTabs](root, func(_ *container.AppTabs) bool {
		return true
	})

	s.True(found, "Container() tree should contain an AppTabs widget")
	s.Equal(root, tabs, "the AppTabs should be the Container() itself")
}

func (s *SettingsInteractionSuite) TestSettingsViewDefaultsToFirstTab() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	s.Equal(0, tabs.SelectedIndex(), "default selected tab should be index 0 (Slack)")
}

func (s *SettingsInteractionSuite) TestSettingsViewTabContentContainsLabel() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})
	firstTabContent := tabs.Items[0].Content

	lbl, found := uitest.FindWidget[*widget.Label](firstTabContent, func(l *widget.Label) bool {
		return l.Text == "Slack Accounts"
	})

	s.True(found, "first tab content should contain a Label with text 'Slack Accounts'")
	s.Equal("Slack Accounts", lbl.Text)
}

func (s *SettingsInteractionSuite) TestAudioTabContainsVolumeSlider() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	s.Require().Greater(len(tabs.Items), 2, "should have at least 3 tabs (Audio is index 2)")
	audioContent := tabs.Items[2].Content

	slider, found := uitest.FindWidget[*widget.Slider](audioContent, func(sl *widget.Slider) bool {
		return true
	})

	s.Require().True(found, "Audio tab content should contain a widget.Slider")
	s.Equal(float64(0), slider.Min, "slider Min should be 0")
	s.Equal(float64(100), slider.Max, "slider Max should be 100")
	s.Equal(float64(1), slider.Step, "slider Step should be 1")
}

func (s *SettingsInteractionSuite) TestSettingsViewEachTabHasContent() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	for i, item := range tabs.Items {
		s.NotNilf(item.Content, "tab %d (%s) should have non-nil Content", i, item.Text)
	}
}
