package ui_test

import (
	"context"
	"strings"
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
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

	s.sv = ui.NewSettingsView(sp, ssp, nil, config.OllamaConfig{
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

	s.Require().Greater(len(tabs.Items), 4, "should have at least 5 tabs (Audio is index 4)")
	audioContent := tabs.Items[4].Content

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
	audioContent := tabs.Items[4].Content

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

	s.Require().Greater(len(tabs.Items), 5, "should have at least 6 tabs (Ollama is index 5)")
	ollamaContent := tabs.Items[5].Content

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
		nil,
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
	s.Require().GreaterOrEqual(len(entries), 7, "form should have at least 7 Entry widgets")

	// Fill in all 7 fields with valid data
	entries[0].SetText("Work Email")              // Friendly Name
	entries[1].SetText("https://mail.google.com") // Web URL
	entries[2].SetText("imap.example.com")        // IMAP Host
	entries[3].SetText("993")                     // IMAP Port
	entries[4].SetText("user@example.com")        // Username
	entries[5].SetText("secret")                  // Password
	entries[6].SetText("600")                     // Poll Interval

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})

	saveBtn.OnTapped()

	// Re-read tab content after save
	emailContent = tabs.Items[1].Content

	entriesAfterSave := uitest.FindAll[*widget.Entry](emailContent, func(_ *widget.Entry) bool {
		return true
	})

	s.Less(len(entriesAfterSave), 7,
		"after saving valid data, form should be replaced with account list (fewer than 7 Entry widgets)")
}

func (s *SettingsInteractionSuite) TestSlackAddAccountShowsFormFields() {
	root := s.sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	slackContent := tabs.Items[0].Content
	btn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	btn.OnTapped()
	slackContent = tabs.Items[0].Content
	entries := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool { return true })
	s.GreaterOrEqual(len(entries), 4, "after tapping Add Account, Slack tab should contain at least 4 Entry widgets (user OAuth token, workspace ID, username, poll interval)")
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
	audioContent := tabs.Items[4].Content

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

	s.Require().Greater(len(tabs.Items), 4, "should have at least 5 tabs (Audio is index 4)")
	audioContent := tabs.Items[4].Content

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

func (s *SettingsInteractionSuite) TestCalendarAddAccountShowsFormFields() {
	root := s.sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	calendarContent := tabs.Items[2].Content
	btn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	btn.OnTapped()
	calendarContent = tabs.Items[2].Content
	entries := uitest.FindAll[*widget.Entry](calendarContent, func(_ *widget.Entry) bool { return true })
	s.GreaterOrEqual(len(entries), 3, "after tapping Add Account, Calendar tab should contain at least 3 Entry widgets (Name, ICS URL, Poll Interval)")

	_, foundSave := uitest.FindWidget[*widget.Button](calendarContent, func(b *widget.Button) bool { return b.Text == "Save" })
	s.True(foundSave, "Calendar form should contain a 'Save' button")

	_, foundCancel := uitest.FindWidget[*widget.Button](calendarContent, func(b *widget.Button) bool { return b.Text == "Cancel" })
	s.True(foundCancel, "Calendar form should contain a 'Cancel' button")
}

func (s *SettingsInteractionSuite) TestCalendarAddAccountValidationShowsError() {
	root := s.sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	calendarContent := tabs.Items[2].Content
	addBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	addBtn.OnTapped()
	calendarContent = tabs.Items[2].Content
	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool { return b.Text == "Save" })
	saveBtn.OnTapped()
	calendarContent = tabs.Items[2].Content
	_, found := uitest.FindWidget[*widget.Label](calendarContent, func(l *widget.Label) bool {
		return strings.Contains(strings.ToLower(l.Text), "required") || strings.Contains(strings.ToLower(l.Text), "error")
	})
	s.True(found, "after tapping Save with empty fields, a validation error label should appear in the form")
}

func (s *SettingsInteractionSuite) TestCalendarAddAccountNonNumericPollShowsError() {
	root := s.sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	calendarContent := tabs.Items[2].Content
	addBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	addBtn.OnTapped()
	calendarContent = tabs.Items[2].Content
	entries := uitest.FindAll[*widget.Entry](calendarContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 3, "form should have at least 3 Entry widgets")
	entries[0].SetText("Test")                        // Name
	entries[1].SetText("https://example.com/cal.ics") // ICS URL
	entries[2].SetText("abc")                         // Poll Interval (non-numeric)
	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool { return b.Text == "Save" })
	saveBtn.OnTapped()
	calendarContent = tabs.Items[2].Content
	_, found := uitest.FindWidget[*widget.Label](calendarContent, func(l *widget.Label) bool {
		return strings.Contains(strings.ToLower(l.Text), "number") || strings.Contains(strings.ToLower(l.Text), "error")
	})
	s.True(found, "after tapping Save with non-numeric poll interval, a validation error label should appear in the form")
}

func (s *SettingsInteractionSuite) TestSlackAddAccountSaveWithValidDataReplacesForm() {
	root := s.sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	slackContent := tabs.Items[0].Content
	addBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	addBtn.OnTapped()
	slackContent = tabs.Items[0].Content
	entries := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 6, "form should have at least 6 Entry widgets")
	entries[0].SetText("My Slack")          // Friendly Name
	entries[1].SetText("https://slack.com") // Web URL
	entries[2].SetText("xoxp-test-token")   // User OAuth Token
	entries[3].SetText("T12345")            // Workspace ID
	entries[4].SetText("testuser")          // Username
	entries[5].SetText("600")               // Poll Interval
	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool { return b.Text == "Save" })
	saveBtn.OnTapped()
	slackContent = tabs.Items[0].Content
	entriesAfterSave := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool { return true })
	s.Less(len(entriesAfterSave), 6, "after saving valid data, form should be replaced with account list (fewer than 6 Entry widgets)")
}

func (s *SettingsInteractionSuite) TestCalendarAddAccountSaveWithValidDataReplacesForm() {
	root := s.sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	calContent := tabs.Items[2].Content
	addBtn := uitest.RequireWidget[*widget.Button](s.T(), calContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	addBtn.OnTapped()
	calContent = tabs.Items[2].Content
	entries := uitest.FindAll[*widget.Entry](calContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 3, "form should have at least 3 Entry widgets")
	entries[0].SetText("Work Calendar")               // Name
	entries[1].SetText("https://example.com/cal.ics") // ICS URL
	entries[2].SetText("600")                         // Poll Interval
	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), calContent, func(b *widget.Button) bool { return b.Text == "Save" })
	saveBtn.OnTapped()
	calContent = tabs.Items[2].Content
	entriesAfterSave := uitest.FindAll[*widget.Entry](calContent, func(_ *widget.Entry) bool { return true })
	s.Less(len(entriesAfterSave), 3, "after saving valid data, form should be replaced with account list (fewer than 3 Entry widgets)")
}

func (s *SettingsInteractionSuite) TestEmailFormHasEncryptionDropdown() {
	root := s.sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	emailContent := tabs.Items[1].Content
	btn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	btn.OnTapped()
	emailContent = tabs.Items[1].Content

	sel, found := uitest.FindWidget[*widget.Select](emailContent, func(_ *widget.Select) bool { return true })
	s.Require().True(found, "email form should contain an encryption Select widget")
	s.Equal("SSL/TLS (Recommended)", sel.Selected, "encryption dropdown should default to 'SSL/TLS (Recommended)'")
	s.Equal([]string{"SSL/TLS (Recommended)", "STARTTLS", "None"}, sel.Options, "encryption dropdown should have correct options")
}

func (s *SettingsInteractionSuite) TestCalendarAddAccountCancelReturnsToList() {
	root := s.sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	calContent := tabs.Items[2].Content
	addBtn := uitest.RequireWidget[*widget.Button](s.T(), calContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	addBtn.OnTapped()
	calContent = tabs.Items[2].Content
	cancelBtn := uitest.RequireWidget[*widget.Button](s.T(), calContent, func(b *widget.Button) bool { return b.Text == "Cancel" })
	cancelBtn.OnTapped()
	calContent = tabs.Items[2].Content
	_, found := uitest.FindWidget[*widget.Button](calContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	s.True(found, "after tapping Cancel, calendar tab should show the account list with 'Add Account' button")
}

func (s *SettingsInteractionSuite) TestSlackTabShowsEmptyStateWhenNoAccounts() {
	root := s.sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	slackContent := tabs.Items[0].Content

	_, found := uitest.FindWidget[*widget.Label](slackContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "No Slack accounts configured")
	})

	s.True(found, "Slack tab should contain a label with 'No Slack accounts configured' when no accounts exist")
}

func (s *SettingsInteractionSuite) TestEmailFormSavesMappedEncryptionValue() {
	// Create a fresh settings view with a capturing repo
	repo := &stubServiceConfigRepo{}
	mgr := &stubWatcherRemover{}
	factory := func(_ string, _ uuid.UUID) error { return nil }
	ssp := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	vc := &stubVolumeController{}
	sp, _ := presenter.NewSettingsPresenter(vc, 50, &stubVolumeController{}, 50)
	sv := ui.NewSettingsView(sp, ssp, nil, config.OllamaConfig{}, func() {})
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	emailContent := tabs.Items[1].Content
	btn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	btn.OnTapped()
	emailContent = tabs.Items[1].Content

	// Fill entries
	entries := uitest.FindAll[*widget.Entry](emailContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 7)
	entries[0].SetText("Work Email")              // Friendly Name
	entries[1].SetText("https://mail.google.com") // Web URL
	entries[2].SetText("imap.example.com")
	entries[3].SetText("993")
	entries[4].SetText("user@example.com")
	entries[5].SetText("secret")
	entries[6].SetText("600")

	// Select STARTTLS
	sel := uitest.RequireWidget[*widget.Select](s.T(), emailContent, func(_ *widget.Select) bool { return true })
	sel.SetSelected("STARTTLS")

	// Save
	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool { return b.Text == "Save" })
	saveBtn.OnTapped()

	s.Require().NotNil(repo.lastSavedEmail, "repo should have received a saved email account")
	s.Equal("starttls", repo.lastSavedEmail.Encryption, "encryption should be mapped from display value to stored value")
}

func (s *SettingsInteractionSuite) TestSlackFormHasFriendlyNameAndWebURLEntries() {
	root := s.sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	slackContent := tabs.Items[0].Content
	btn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	btn.OnTapped()
	slackContent = tabs.Items[0].Content

	_, foundFriendly := uitest.FindWidget[*widget.Entry](slackContent, func(e *widget.Entry) bool {
		return e.PlaceHolder == "Friendly Name"
	})
	s.True(foundFriendly, "Slack account form should contain an Entry with placeholder 'Friendly Name'")

	_, foundWebURL := uitest.FindWidget[*widget.Entry](slackContent, func(e *widget.Entry) bool {
		return e.PlaceHolder == "Web URL"
	})
	s.True(foundWebURL, "Slack account form should contain an Entry with placeholder 'Web URL'")
}

func (s *SettingsInteractionSuite) TestSlackFormTokenInstructionsAccordion() {
	root := s.sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	slackContent := tabs.Items[0].Content

	// Tap "Add Account" to show the form
	addBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool { return b.Text == "Add Account" })
	addBtn.OnTapped()
	slackContent = tabs.Items[0].Content

	// The form should contain an Accordion with token setup instructions
	accordion, found := uitest.FindWidget[*widget.Accordion](slackContent, func(_ *widget.Accordion) bool { return true })
	s.Require().True(found, "Slack account form should contain a widget.Accordion for token instructions")

	// Accordion should have at least 1 item with a title mentioning "token"
	s.Require().GreaterOrEqual(len(accordion.Items), 1, "accordion should have at least 1 item")

	var tokenItem *widget.AccordionItem
	for _, item := range accordion.Items {
		if strings.Contains(strings.ToLower(item.Title), "token") {
			tokenItem = item
			break
		}
	}
	s.Require().NotNil(tokenItem, "accordion should have an item with title containing 'token'")

	// The detail content should mention all 9 required OAuth scopes
	requiredScopes := []string{
		"channels:history",
		"channels:read",
		"groups:history",
		"groups:read",
		"im:history",
		"im:read",
		"mpim:history",
		"mpim:read",
		"users:read",
	}

	// Collect all text from labels and rich text widgets in the detail
	detail := tokenItem.Detail
	s.Require().NotNil(detail, "accordion item detail should not be nil")

	// Gather all label text from the detail content
	labels := uitest.FindAll[*widget.Label](detail, func(_ *widget.Label) bool { return true })
	richTexts := uitest.FindAll[*widget.RichText](detail, func(_ *widget.RichText) bool { return true })

	var allText string
	for _, l := range labels {
		allText += " " + l.Text
	}
	for _, rt := range richTexts {
		allText += " " + rt.String()
	}
	allText = strings.ToLower(allText)

	for _, scope := range requiredScopes {
		s.Contains(allText, scope,
			"accordion detail should mention required OAuth scope: %s", scope)
	}
}

func (s *SettingsInteractionSuite) TestSlackAccountListRendersDeleteButton() {
	// Create a stub repo that returns one Slack account
	repo := &stubServiceConfigRepoWithSlack{
		slackAccounts: []*repository.SlackAccount{
			{
				ID:          uuid.New(),
				Enabled:     true,
				Token:       "xoxp-test",
				WorkspaceID: "T12345",
				Username:    "testuser",
			},
		},
	}
	mgr := &stubWatcherRemover{}
	factory := func(_ string, _ uuid.UUID) error { return nil }
	ssp := presenter.NewServiceSettingsPresenter(repo, mgr, factory)
	vc := &stubVolumeController{}
	sp, _ := presenter.NewSettingsPresenter(vc, 50, &stubVolumeController{}, 50)
	sv := ui.NewSettingsView(sp, ssp, nil, config.OllamaConfig{}, func() {})
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	slackContent := tabs.Items[0].Content

	_, found := uitest.FindWidget[*widget.Button](slackContent, func(b *widget.Button) bool {
		return b.Text == "Delete"
	})

	s.True(found, "Slack account list should contain a 'Delete' button for each configured account")
}

// stubServiceConfigRepoWithSlack is a stub that returns pre-configured Slack accounts.
// All other account types return empty lists.
type stubServiceConfigRepoWithSlack struct {
	slackAccounts []*repository.SlackAccount
}

func (s *stubServiceConfigRepoWithSlack) ListSlackAccounts(_ context.Context) ([]*repository.SlackAccount, error) {
	return s.slackAccounts, nil
}
func (s *stubServiceConfigRepoWithSlack) GetSlackAccount(_ context.Context, _ uuid.UUID) (*repository.SlackAccount, error) {
	return nil, nil
}
func (s *stubServiceConfigRepoWithSlack) UpsertSlackAccount(_ context.Context, _ *repository.SlackAccount) error {
	return nil
}
func (s *stubServiceConfigRepoWithSlack) DeleteSlackAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (s *stubServiceConfigRepoWithSlack) ListEmailAccounts(_ context.Context) ([]*repository.EmailAccount, error) {
	return nil, nil
}
func (s *stubServiceConfigRepoWithSlack) GetEmailAccount(_ context.Context, _ uuid.UUID) (*repository.EmailAccount, error) {
	return nil, nil
}
func (s *stubServiceConfigRepoWithSlack) UpsertEmailAccount(_ context.Context, _ *repository.EmailAccount) error {
	return nil
}
func (s *stubServiceConfigRepoWithSlack) DeleteEmailAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (s *stubServiceConfigRepoWithSlack) ListCalendarAccounts(_ context.Context) ([]*repository.CalendarAccount, error) {
	return nil, nil
}
func (s *stubServiceConfigRepoWithSlack) GetCalendarAccount(_ context.Context, _ uuid.UUID) (*repository.CalendarAccount, error) {
	return nil, nil
}
func (s *stubServiceConfigRepoWithSlack) UpsertCalendarAccount(_ context.Context, _ *repository.CalendarAccount) error {
	return nil
}
func (s *stubServiceConfigRepoWithSlack) DeleteCalendarAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}
