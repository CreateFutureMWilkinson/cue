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
	sp, err := presenter.NewSettingsPresenter(vc, 50, &stubVolumeController{}, 50)
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
	s.NotNil(tabs, "the AppTabs should be found within the Container() tree")
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

	s.Require().Greater(len(tabs.Items), 3, "should have at least 4 tabs (Audio is index 3)")
	audioContent := tabs.Items[3].Content

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
	audioContent := tabs.Items[3].Content

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

	s.Require().Greater(len(tabs.Items), 4, "should have at least 5 tabs (Ollama is index 4)")
	ollamaContent := tabs.Items[4].Content

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

func (s *SettingsInteractionSuite) TestEmailAddAccountShowsFormFields() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	emailContent := tabs.Items[1].Content

	btn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	btn.OnTapped()

	// Re-read tab content after tap
	emailContent = tabs.Items[1].Content

	entries := uitest.FindAll[*widget.Entry](emailContent, func(_ *widget.Entry) bool {
		return true
	})

	s.GreaterOrEqual(len(entries), 5,
		"after tapping Add Account, email tab should contain at least 5 Entry widgets "+
			"(IMAP host, port, username, password, poll interval)")
}

func (s *SettingsInteractionSuite) TestEmailAddAccountValidationShowsError() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	emailContent := tabs.Items[1].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	addBtn.OnTapped()

	// Re-read tab content after tap (form is now shown)
	emailContent = tabs.Items[1].Content

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})

	// Tap Save with all entries empty
	saveBtn.OnTapped()

	// Re-read tab content after save tap (validation error should appear)
	emailContent = tabs.Items[1].Content

	_, found := uitest.FindWidget[*widget.Label](emailContent, func(l *widget.Label) bool {
		return strings.Contains(strings.ToLower(l.Text), "required") ||
			strings.Contains(strings.ToLower(l.Text), "error")
	})

	s.True(found, "after tapping Save with empty fields, a validation error label should appear in the form")
}

func (s *SettingsInteractionSuite) TestEmailAddAccountSaveWithValidDataReplacesForm() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	emailContent := tabs.Items[1].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	addBtn.OnTapped()

	// Re-read tab content after tap (form is now shown)
	emailContent = tabs.Items[1].Content

	entries := uitest.FindAll[*widget.Entry](emailContent, func(_ *widget.Entry) bool {
		return true
	})
	s.Require().GreaterOrEqual(len(entries), 5, "form should have at least 5 Entry widgets")

	// Fill in all 5 fields with valid data
	entries[0].SetText("imap.example.com") // IMAP Host
	entries[1].SetText("993")              // IMAP Port
	entries[2].SetText("user@example.com") // Username
	entries[3].SetText("secret")           // Password
	entries[4].SetText("600")              // Poll Interval

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})

	saveBtn.OnTapped()

	// Re-read tab content after save
	emailContent = tabs.Items[1].Content

	entriesAfterSave := uitest.FindAll[*widget.Entry](emailContent, func(_ *widget.Entry) bool {
		return true
	})

	s.Less(len(entriesAfterSave), 5,
		"after saving valid data, form should be replaced with account list (fewer than 5 Entry widgets)")
}

func (s *SettingsInteractionSuite) TestSlackAddAccountShowsFormFields() {
	root := s.sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	slackContent := tabs.Items[0].Content
	btn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	btn.OnTapped()
	slackContent = tabs.Items[0].Content
	entries := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool { return true })
	s.GreaterOrEqual(len(entries), 3, "after tapping Add Account, Slack tab should contain at least 3 Entry widgets (bot token, workspace ID, poll interval)")
}

func (s *SettingsInteractionSuite) TestSlackAddAccountValidationShowsError() {
	root := s.sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	slackContent := tabs.Items[0].Content
	addBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	addBtn.OnTapped()
	slackContent = tabs.Items[0].Content
	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool { return b.Text == "Save" })
	saveBtn.OnTapped()
	slackContent = tabs.Items[0].Content
	_, found := uitest.FindWidget[*widget.Label](slackContent, func(l *widget.Label) bool {
		return strings.Contains(strings.ToLower(l.Text), "required") || strings.Contains(strings.ToLower(l.Text), "error")
	})
	s.True(found, "after tapping Save with empty fields, a validation error label should appear in the form")
}

func (s *SettingsInteractionSuite) TestAudioSliderOnChangedCallsPresenterSetVolume() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})
	audioContent := tabs.Items[3].Content

	slider := uitest.RequireWidget[*widget.Slider](s.T(), audioContent, func(_ *widget.Slider) bool {
		return true
	})

	s.Require().NotNil(slider.OnChanged, "slider.OnChanged should be wired")

	slider.OnChanged(75)

	s.Equal(75, s.sp.Volume(),
		"presenter volume should be updated to 75 after slider change")
}

func (s *SettingsInteractionSuite) TestTimerVolumeSlider() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	s.Require().Greater(len(tabs.Items), 3, "should have at least 4 tabs (Audio is index 3)")
	audioContent := tabs.Items[3].Content

	// There should be 2 sliders: notification volume and timer volume
	sliders := uitest.FindAll[*widget.Slider](audioContent, func(_ *widget.Slider) bool {
		return true
	})
	s.Equal(2, len(sliders), "Audio tab should contain exactly 2 sliders (notification + timer)")

	// Find the timer volume label
	timerLabel, found := uitest.FindWidget[*widget.Label](audioContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "Timer Volume:")
	})
	s.Require().True(found, "Audio tab should contain a 'Timer Volume:' label")

	// Find the timer slider (second slider)
	timerSlider := sliders[1]
	s.Equal(float64(0), timerSlider.Min, "timer slider Min should be 0")
	s.Equal(float64(100), timerSlider.Max, "timer slider Max should be 100")
	s.Equal(float64(1), timerSlider.Step, "timer slider Step should be 1")

	// Simulate dragging the timer slider
	s.Require().NotNil(timerSlider.OnChanged, "timer slider OnChanged should be wired")
	timerSlider.OnChanged(70)

	s.Equal("Timer Volume: 70%", timerLabel.Text,
		"timer volume label should update to reflect new slider value")
}

func (s *SettingsInteractionSuite) TestSlackAddAccountSaveWithValidDataReplacesForm() {
	root := s.sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	slackContent := tabs.Items[0].Content
	addBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	addBtn.OnTapped()
	slackContent = tabs.Items[0].Content
	entries := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 3, "form should have at least 3 Entry widgets")
	entries[0].SetText("xoxp-test-token") // Bot Token
	entries[1].SetText("T12345")          // Workspace ID
	entries[2].SetText("600")             // Poll Interval
	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool { return b.Text == "Save" })
	saveBtn.OnTapped()
	slackContent = tabs.Items[0].Content
	entriesAfterSave := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool { return true })
	s.Less(len(entriesAfterSave), 3, "after saving valid data, form should be replaced with account list (fewer than 3 Entry widgets)")
}
