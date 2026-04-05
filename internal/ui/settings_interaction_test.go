package ui_test

import (
	"strings"
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
	sp *presenter.SettingsPresenter
}

func TestSettingsInteraction(t *testing.T) {
	suite.Run(t, new(SettingsInteractionSuite))
}

func (s *SettingsInteractionSuite) SetupTest() {
	vc := &stubVolumeController{}
	sp, err := presenter.NewSettingsPresenter(vc, 50)
	s.Require().NoError(err)
	s.sp = sp

	repo := &stubServiceConfigRepo{}
	mgr := &stubWatcherRemover{}
	factory := func(_ string, _ uuid.UUID) error { return nil }
	ssp := presenter.NewServiceSettingsPresenter(repo, mgr, factory)

	s.sv = ui.NewSettingsView(sp, ssp, config.OllamaConfig{
		Host:           "localhost",
		Port:           11434,
		InferenceModel: "neural-chat",
		EmbeddingModel: "nomic-embed-text",
		TimeoutSeconds: 10,
	}, func() {})
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

func (s *SettingsInteractionSuite) TestAudioSliderOnChangedUpdatesVolumeLabel() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})
	audioContent := tabs.Items[2].Content

	slider := uitest.RequireWidget[*widget.Slider](s.T(), audioContent, func(_ *widget.Slider) bool {
		return true
	})

	s.Require().NotNil(slider.OnChanged, "slider.OnChanged should be wired")

	slider.OnChanged(75)

	lbl := uitest.RequireWidget[*widget.Label](s.T(), audioContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "Notification Volume:")
	})

	s.Equal("Notification Volume: 75%", lbl.Text,
		"volume label should reflect the new slider value")
}

func (s *SettingsInteractionSuite) TestOllamaTabDisplaysConfigFields() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	s.Require().Greater(len(tabs.Items), 3, "should have at least 4 tabs (Ollama is index 3)")
	ollamaContent := tabs.Items[3].Content

	_, foundHost := uitest.FindWidget[*widget.Label](ollamaContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "localhost")
	})
	s.True(foundHost, "Ollama tab should contain a label with the configured host 'localhost'")

	_, foundModel := uitest.FindWidget[*widget.Label](ollamaContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "neural-chat")
	})
	s.True(foundModel, "Ollama tab should contain a label with the configured inference model 'neural-chat'")
}

func (s *SettingsInteractionSuite) TestSlackTabContainsAddButton() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	slackContent := tabs.Items[0].Content

	_, found := uitest.FindWidget[*widget.Button](slackContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	s.True(found, "Slack tab should contain an 'Add Account' button")
}

func (s *SettingsInteractionSuite) TestEmailTabContainsAddButton() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	emailContent := tabs.Items[1].Content

	_, found := uitest.FindWidget[*widget.Button](emailContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	s.True(found, "Email tab should contain an 'Add Account' button")
}

func (s *SettingsInteractionSuite) TestSettingsViewContainsDoneButton() {
	root := s.sv.Container()

	_, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Done"
	})

	s.True(found, "settings view should contain a 'Done' button")
}

func (s *SettingsInteractionSuite) TestDoneButtonCallsOnClose() {
	closeCalled := false
	sv := ui.NewSettingsView(s.sp,
		presenter.NewServiceSettingsPresenter(&stubServiceConfigRepo{}, &stubWatcherRemover{}, func(_ string, _ uuid.UUID) error { return nil }),
		config.OllamaConfig{},
		func() { closeCalled = true },
	)
	root := sv.Container()

	btn := uitest.RequireWidget[*widget.Button](s.T(), root, func(b *widget.Button) bool {
		return b.Text == "Done"
	})

	btn.OnTapped()

	s.True(closeCalled, "tapping Done button should invoke onClose callback")
}

func (s *SettingsInteractionSuite) TestAudioSliderOnChangedCallsPresenterSetVolume() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})
	audioContent := tabs.Items[2].Content

	slider := uitest.RequireWidget[*widget.Slider](s.T(), audioContent, func(_ *widget.Slider) bool {
		return true
	})

	s.Require().NotNil(slider.OnChanged, "slider.OnChanged should be wired")

	slider.OnChanged(75)

	s.Equal(75, s.sp.Volume(),
		"presenter volume should be updated to 75 after slider change")
}
