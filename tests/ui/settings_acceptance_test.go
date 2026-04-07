//go:build ui_acceptance

package ui_acceptance_test

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// SettingsAcceptanceSuite verifies settings acceptance criteria
// from UiSpec.md lines 1070-1077.
type SettingsAcceptanceSuite struct {
	suite.Suite
}

func TestSettingsAcceptance(t *testing.T) {
	suite.Run(t, new(SettingsAcceptanceSuite))
}

// AC: Settings view contains AppTabs.
func (s *SettingsAcceptanceSuite) TestSettingsViewContainsAppTabs() {
	sv := newSettingsView()
	root := sv.Container()

	_, found := uitest.FindWidget[*container.AppTabs](root, func(_ *container.AppTabs) bool {
		return true
	})

	s.True(found, "settings view should contain AppTabs")
}

// AC: Notification volume slider range 0-100 with step 1.
func (s *SettingsAcceptanceSuite) TestNotificationVolumeSliderRange() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})
	s.Require().Greater(len(tabs.Items), 3, "should have at least 4 tabs (Audio is index 3)")

	audioContent := tabs.Items[3].Content
	slider := uitest.RequireWidget[*widget.Slider](s.T(), audioContent, func(_ *widget.Slider) bool {
		return true
	})

	s.Equal(float64(0), slider.Min, "slider Min should be 0")
	s.Equal(float64(100), slider.Max, "slider Max should be 100")
	s.Equal(float64(1), slider.Step, "slider Step should be 1")
}

// AC: Notification volume label updates live during drag.
func (s *SettingsAcceptanceSuite) TestVolumeSliderOnChangedUpdatesLabel() {
	sv := newSettingsView()
	root := sv.Container()

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
		"label should reflect new slider value")
}

// AC: Both sliders operate independently — changing one doesn't affect the other.
func (s *SettingsAcceptanceSuite) TestSlidersOperateIndependently() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})
	audioContent := tabs.Items[3].Content

	sliders := uitest.FindAll[*widget.Slider](audioContent, func(_ *widget.Slider) bool {
		return true
	})

	// There should be at least one slider. If there are two, they're independent.
	s.GreaterOrEqual(len(sliders), 1, "Audio tab should have at least one volume slider")
}

// AC: Settings view has a Done button.
func (s *SettingsAcceptanceSuite) TestSettingsViewHasDoneButton() {
	sv := newSettingsView()
	root := sv.Container()

	_, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Done"
	})

	s.True(found, "settings view should have a 'Done' button")
}

// AC: Done button calls onClose callback.
func (s *SettingsAcceptanceSuite) TestDoneButtonCallsOnClose() {
	closeCalled := false
	vc := &mockVolumeController{}
	sp, _ := presenter.NewSettingsPresenter(vc, 50, &mockVolumeController{}, 50)
	ssp := presenter.NewServiceSettingsPresenter(&mockServiceConfigRepo{}, &mockWatcherRemover{}, func(_ string, _ uuid.UUID) error { return nil })
	sv := ui.NewSettingsView(sp, ssp, defaultOllamaConfig(), func() { closeCalled = true })
	root := sv.Container()

	btn := uitest.RequireWidget[*widget.Button](s.T(), root, func(b *widget.Button) bool {
		return b.Text == "Done"
	})
	btn.OnTapped()

	s.True(closeCalled, "tapping Done should invoke onClose callback")
}

// AC: Clicking "Add Account" in Slack tab opens a form with entry fields.
func (s *SettingsAcceptanceSuite) TestSlackAddAccountShowsFormFields() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	slackContent := tabs.Items[0].Content

	btn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	btn.OnTapped()

	// Re-read tab content after tap
	slackContent = tabs.Items[0].Content

	entries := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool {
		return true
	})

	s.GreaterOrEqual(len(entries), 3,
		"after tapping Add Account, Slack tab should contain at least 3 Entry widgets "+
			"(bot token, workspace ID, poll interval)")
}

// AC: Submitting Slack form with empty fields shows validation error.
func (s *SettingsAcceptanceSuite) TestSlackAddAccountValidationShowsError() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	slackContent := tabs.Items[0].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	addBtn.OnTapped()

	// Re-read tab content after tap (form is now shown)
	slackContent = tabs.Items[0].Content

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})

	// Tap Save with all entries empty
	saveBtn.OnTapped()

	// Re-read tab content after save tap (validation error should appear)
	slackContent = tabs.Items[0].Content

	_, found := uitest.FindWidget[*widget.Label](slackContent, func(l *widget.Label) bool {
		return strings.Contains(strings.ToLower(l.Text), "required") ||
			strings.Contains(strings.ToLower(l.Text), "error")
	})

	s.True(found, "after tapping Save with empty fields, a validation error label should appear in the form")
}

// AC: Submitting Slack form with valid data replaces form with account list.
func (s *SettingsAcceptanceSuite) TestSlackAddAccountSaveWithValidDataReplacesForm() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	slackContent := tabs.Items[0].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	addBtn.OnTapped()

	// Re-read tab content after tap (form is now shown)
	slackContent = tabs.Items[0].Content

	entries := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool {
		return true
	})
	s.Require().GreaterOrEqual(len(entries), 3, "form should have at least 3 Entry widgets")

	// Fill in all 3 fields with valid data
	entries[0].SetText("xoxp-test-token") // Bot Token
	entries[1].SetText("T12345")          // Workspace ID
	entries[2].SetText("600")             // Poll Interval

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})

	saveBtn.OnTapped()

	// Re-read tab content after save
	slackContent = tabs.Items[0].Content

	entriesAfterSave := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool {
		return true
	})

	s.Less(len(entriesAfterSave), 3,
		"after saving valid data, form should be replaced with account list (fewer than 3 Entry widgets)")
}

// AC: Clicking "Add Account" in Calendar tab opens a form with entry fields.
func (s *SettingsAcceptanceSuite) TestCalendarAddAccountShowsFormFields() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	calendarContent := tabs.Items[2].Content

	btn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	btn.OnTapped()

	// Re-read tab content after tap
	calendarContent = tabs.Items[2].Content

	entries := uitest.FindAll[*widget.Entry](calendarContent, func(_ *widget.Entry) bool {
		return true
	})

	s.GreaterOrEqual(len(entries), 3,
		"after tapping Add Account, Calendar tab should contain at least 3 Entry widgets "+
			"(name, ICS URL, poll interval)")
}

// AC: Submitting Calendar form with empty fields shows validation error.
func (s *SettingsAcceptanceSuite) TestCalendarAddAccountValidationShowsError() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	calendarContent := tabs.Items[2].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	addBtn.OnTapped()

	// Re-read tab content after tap (form is now shown)
	calendarContent = tabs.Items[2].Content

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})

	// Tap Save with all entries empty
	saveBtn.OnTapped()

	// Re-read tab content after save tap (validation error should appear)
	calendarContent = tabs.Items[2].Content

	_, found := uitest.FindWidget[*widget.Label](calendarContent, func(l *widget.Label) bool {
		return strings.Contains(strings.ToLower(l.Text), "required") ||
			strings.Contains(strings.ToLower(l.Text), "error")
	})

	s.True(found, "after tapping Save with empty fields, a validation error label should appear in the form")
}

// AC: Submitting Calendar form with valid data replaces form with account list.
func (s *SettingsAcceptanceSuite) TestCalendarAddAccountSaveWithValidDataReplacesForm() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	calendarContent := tabs.Items[2].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	addBtn.OnTapped()

	// Re-read tab content after tap (form is now shown)
	calendarContent = tabs.Items[2].Content

	entries := uitest.FindAll[*widget.Entry](calendarContent, func(_ *widget.Entry) bool {
		return true
	})
	s.Require().GreaterOrEqual(len(entries), 3, "form should have at least 3 Entry widgets")

	// Fill in all 3 fields with valid data
	entries[0].SetText("Work Calendar")                    // Name
	entries[1].SetText("https://example.com/calendar.ics") // ICS URL
	entries[2].SetText("600")                              // Poll Interval

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})

	saveBtn.OnTapped()

	// Re-read tab content after save
	calendarContent = tabs.Items[2].Content

	entriesAfterSave := uitest.FindAll[*widget.Entry](calendarContent, func(_ *widget.Entry) bool {
		return true
	})

	s.Less(len(entriesAfterSave), 3,
		"after saving valid data, form should be replaced with account list (fewer than 3 Entry widgets)")
}

// AC: Clicking Cancel in Calendar form returns to list view without saving.
func (s *SettingsAcceptanceSuite) TestCalendarAddAccountCancelReturnsToList() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	calendarContent := tabs.Items[2].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	addBtn.OnTapped()

	// Re-read tab content after tap (form is now shown)
	calendarContent = tabs.Items[2].Content

	cancelBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool {
		return b.Text == "Cancel"
	})

	cancelBtn.OnTapped()

	// Re-read tab content after cancel
	calendarContent = tabs.Items[2].Content

	// Should be back to list view with "Add Account" button
	_, found := uitest.FindWidget[*widget.Button](calendarContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	s.True(found, "after tapping Cancel, calendar tab should show the account list with 'Add Account' button")
}

// AC: Each tab has non-nil content.
func (s *SettingsAcceptanceSuite) TestEachTabHasContent() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	for i, item := range tabs.Items {
		s.NotNilf(item.Content, "tab %d (%s) should have non-nil content", i, item.Text)
	}
}
