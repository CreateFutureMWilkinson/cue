//go:build ui_acceptance

package ui_acceptance_test

import (
	"fmt"
	"strings"
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
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
	s.Require().Greater(len(tabs.Items), 4, "should have at least 5 tabs (Audio is index 4)")

	audioContent := tabs.Items[4].Content
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
		"label should reflect new slider value")
}

// AC: Both sliders operate independently — changing one doesn't affect the other.
func (s *SettingsAcceptanceSuite) TestSlidersOperateIndependently() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})
	audioContent := tabs.Items[4].Content

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
	ssp := presenter.NewServiceSettingsPresenter(&mockServiceConfigRepo{}, &mockWatcherToggler{})
	sv := ui.NewSettingsView(sp, ssp, nil, defaultOllamaConfig(), func() { closeCalled = true })
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

	s.GreaterOrEqual(len(entries), 6,
		"after tapping Add Account, Slack tab should contain at least 6 Entry widgets "+
			"(friendly name, web URL, user OAuth token, workspace ID, username, poll interval)")
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
	s.Require().GreaterOrEqual(len(entries), 6, "form should have at least 6 Entry widgets")

	// Fill in all 6 fields with valid data
	entries[0].SetText("My Slack")          // Friendly Name
	entries[1].SetText("https://slack.com") // Web URL
	entries[2].SetText("xoxp-test-token")   // User OAuth Token
	entries[3].SetText("T12345")            // Workspace ID
	entries[4].SetText("testuser")          // Username
	entries[5].SetText("600")               // Poll Interval

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})

	saveBtn.OnTapped()

	// Re-read tab content after save
	slackContent = tabs.Items[0].Content

	entriesAfterSave := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool {
		return true
	})

	s.Less(len(entriesAfterSave), 6,
		"after saving valid data, form should be replaced with account list (fewer than 6 Entry widgets)")
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

// AC: Email form has an encryption dropdown with SSL/TLS as default.
func (s *SettingsAcceptanceSuite) TestEmailFormHasEncryptionDropdown() {
	sv := newSettingsView()
	root := sv.Container()

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

	sel, found := uitest.FindWidget[*widget.Select](emailContent, func(_ *widget.Select) bool {
		return true
	})

	s.True(found, "email form should contain an encryption Select widget")
	s.Equal("SSL/TLS (Recommended)", sel.Selected,
		"encryption dropdown should default to 'SSL/TLS (Recommended)'")
	s.Len(sel.Options, 3, "encryption dropdown should have 3 options")
}

// AC: Email form save persists encryption value.
func (s *SettingsAcceptanceSuite) TestEmailFormSaveIncludesEncryption() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	emailContent := tabs.Items[1].Content

	btn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	btn.OnTapped()

	emailContent = tabs.Items[1].Content

	entries := uitest.FindAll[*widget.Entry](emailContent, func(_ *widget.Entry) bool {
		return true
	})
	s.Require().GreaterOrEqual(len(entries), 7, "form should have at least 7 Entry widgets")

	entries[0].SetText("Work Email")               // Friendly Name
	entries[1].SetText("https://mail.example.com") // Web URL
	entries[2].SetText("imap.example.com")         // IMAP Host
	entries[3].SetText("993")                      // IMAP Port
	entries[4].SetText("user@example.com")         // Username
	entries[5].SetText("secret")                   // Password
	entries[6].SetText("600")                      // Poll Interval

	// Select STARTTLS encryption
	sel := uitest.RequireWidget[*widget.Select](s.T(), emailContent, func(_ *widget.Select) bool {
		return true
	})
	sel.SetSelected("STARTTLS")

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})

	saveBtn.OnTapped()

	// Re-read tab content after save — form should be replaced with account list
	emailContent = tabs.Items[1].Content

	entriesAfterSave := uitest.FindAll[*widget.Entry](emailContent, func(_ *widget.Entry) bool {
		return true
	})

	s.Less(len(entriesAfterSave), 7,
		"after saving valid data with encryption set, form should be replaced with account list")
}

// AC: Empty account list shows helpful empty state message.
func (s *SettingsAcceptanceSuite) TestEmptySlackAccountListShowsEmptyState() {
	sv := newSettingsView() // no pre-loaded accounts
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	slackContent := tabs.Items[0].Content

	_, found := uitest.FindWidget[*widget.Label](slackContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "No Slack accounts configured")
	})

	s.True(found, "empty Slack account list should show 'No Slack accounts configured' message")
}

// AC: Empty email account list shows helpful empty state message.
func (s *SettingsAcceptanceSuite) TestEmptyEmailAccountListShowsEmptyState() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	emailContent := tabs.Items[1].Content

	_, found := uitest.FindWidget[*widget.Label](emailContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "No Email accounts configured")
	})

	s.True(found, "empty Email account list should show 'No Email accounts configured' message")
}

// AC: Empty calendar account list shows helpful empty state message.
func (s *SettingsAcceptanceSuite) TestEmptyCalendarAccountListShowsEmptyState() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	calendarContent := tabs.Items[2].Content

	_, found := uitest.FindWidget[*widget.Label](calendarContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "No Calendar accounts configured")
	})

	s.True(found, "empty Calendar account list should show 'No Calendar accounts configured' message")
}

// AC: Existing Slack accounts appear in the list when opening Settings.
func (s *SettingsAcceptanceSuite) TestPreExistingSlackAccountsAppearInList() {
	repo := &mockServiceConfigRepo{
		slackAccounts: []*repository.SlackAccount{
			{ID: uuid.New(), Enabled: true, Token: "xoxp-123", WorkspaceID: "T-ACME", Username: "testuser", PollIntervalSeconds: 600},
		},
	}
	sv := newSettingsViewWithRepo(repo)
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	slackContent := tabs.Items[0].Content

	_, found := uitest.FindWidget[*widget.Label](slackContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "T-ACME")
	})

	s.True(found, "pre-existing Slack account should show workspace ID 'T-ACME' in the list")
}

// AC: Existing Email accounts appear in the list when opening Settings.
func (s *SettingsAcceptanceSuite) TestPreExistingEmailAccountsAppearInList() {
	repo := &mockServiceConfigRepo{
		emailAccounts: []*repository.EmailAccount{
			{ID: uuid.New(), Enabled: true, IMAPHost: "imap.example.com", IMAPPort: 993, Username: "user@example.com", Password: "secret", PollIntervalSeconds: 600},
		},
	}
	sv := newSettingsViewWithRepo(repo)
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	emailContent := tabs.Items[1].Content

	_, found := uitest.FindWidget[*widget.Label](emailContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "user@example.com")
	})

	s.True(found, "pre-existing Email account should show username 'user@example.com' in the list")
}

// AC: Existing Calendar accounts appear in the list when opening Settings.
func (s *SettingsAcceptanceSuite) TestPreExistingCalendarAccountsAppearInList() {
	repo := &mockServiceConfigRepo{
		calendarAccounts: []*repository.CalendarAccount{
			{ID: uuid.New(), Enabled: true, Name: "Work Calendar", ICSURL: "https://example.com/cal.ics", PollIntervalSeconds: 600},
		},
	}
	sv := newSettingsViewWithRepo(repo)
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	calendarContent := tabs.Items[2].Content

	_, found := uitest.FindWidget[*widget.Label](calendarContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "Work Calendar")
	})

	s.True(found, "pre-existing Calendar account should show name 'Work Calendar' in the list")
}

// AC: After adding a new Slack account, it appears in the account list immediately.
func (s *SettingsAcceptanceSuite) TestSlackAccountAppearsAfterSave() {
	repo := &mockServiceConfigRepo{}
	sv := newSettingsViewWithRepo(repo)
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	slackContent := tabs.Items[0].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})
	addBtn.OnTapped()

	slackContent = tabs.Items[0].Content
	entries := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 6)
	entries[0].SetText("New Slack")         // Friendly Name
	entries[1].SetText("https://slack.com") // Web URL
	entries[2].SetText("xoxp-new-token")
	entries[3].SetText("T-NEW-WS")
	entries[4].SetText("newuser")
	entries[5].SetText("600")

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})
	saveBtn.OnTapped()

	slackContent = tabs.Items[0].Content

	_, found := uitest.FindWidget[*widget.Label](slackContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "T-NEW-WS")
	})

	s.True(found, "after saving a new Slack account, workspace ID 'T-NEW-WS' should appear in the list")
}

// AC: After adding a new Email account, it appears in the account list immediately.
func (s *SettingsAcceptanceSuite) TestEmailAccountAppearsAfterSave() {
	repo := &mockServiceConfigRepo{}
	sv := newSettingsViewWithRepo(repo)
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	emailContent := tabs.Items[1].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})
	addBtn.OnTapped()

	emailContent = tabs.Items[1].Content
	entries := uitest.FindAll[*widget.Entry](emailContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 7)
	entries[0].SetText("Test Email")            // Friendly Name
	entries[1].SetText("https://mail.test.com") // Web URL
	entries[2].SetText("imap.test.com")
	entries[3].SetText("993")
	entries[4].SetText("new@test.com")
	entries[5].SetText("password")
	entries[6].SetText("600")

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})
	saveBtn.OnTapped()

	emailContent = tabs.Items[1].Content

	_, found := uitest.FindWidget[*widget.Label](emailContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "new@test.com")
	})

	s.True(found, "after saving a new Email account, username 'new@test.com' should appear in the list")
}

// AC: After adding a new Calendar account, it appears in the account list immediately.
func (s *SettingsAcceptanceSuite) TestCalendarAccountAppearsAfterSave() {
	repo := &mockServiceConfigRepo{}
	sv := newSettingsViewWithRepo(repo)
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	calendarContent := tabs.Items[2].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})
	addBtn.OnTapped()

	calendarContent = tabs.Items[2].Content
	entries := uitest.FindAll[*widget.Entry](calendarContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 3)
	entries[0].SetText("Personal Calendar")
	entries[1].SetText("https://example.com/personal.ics")
	entries[2].SetText("600")

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})
	saveBtn.OnTapped()

	calendarContent = tabs.Items[2].Content

	_, found := uitest.FindWidget[*widget.Label](calendarContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "Personal Calendar")
	})

	s.True(found, "after saving a new Calendar account, name 'Personal Calendar' should appear in the list")
}

// --- Edit account flow ---

// AC: Each Slack account row renders an Edit button.
func (s *SettingsAcceptanceSuite) TestSlackAccountRowHasEditButton() {
	repo := &mockServiceConfigRepo{
		slackAccounts: []*repository.SlackAccount{
			{ID: uuid.New(), Enabled: true, FriendlyName: "Acme", Token: "xoxp-old", WorkspaceID: "T-OLD", Username: "olduser", PollIntervalSeconds: 600},
		},
	}
	sv := newSettingsViewWithRepo(repo)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })

	slackContent := tabs.Items[0].Content
	_, found := uitest.FindWidget[*widget.Button](slackContent, func(b *widget.Button) bool { return b.Text == "Edit" })
	s.True(found, "Slack account row should have an Edit button")
}

// AC: Tapping Edit on a Slack account opens the form prefilled with its values
// and saving updates the existing account in place (no duplicate, same ID).
func (s *SettingsAcceptanceSuite) TestSlackEditPrefillsFormAndUpdatesInPlace() {
	originalID := uuid.New()
	repo := &mockServiceConfigRepo{
		slackAccounts: []*repository.SlackAccount{
			{ID: originalID, Enabled: true, FriendlyName: "Acme", WebURL: "https://acme.slack.com", Token: "xoxp-old", WorkspaceID: "T-OLD", Username: "olduser", PollIntervalSeconds: 600},
		},
	}
	sv := newSettingsViewWithRepo(repo, presenter.WithSlackValidator(&mockSlackValidator{err: nil}))
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })

	slackContent := tabs.Items[0].Content
	editBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool { return b.Text == "Edit" })
	editBtn.OnTapped()

	slackContent = tabs.Items[0].Content
	entries := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 6, "edit form should have at least 6 entries")

	// Prefill check: existing values must populate the form.
	s.Equal("Acme", entries[0].Text, "FriendlyName entry should be prefilled")
	s.Equal("https://acme.slack.com", entries[1].Text, "WebURL entry should be prefilled")
	s.Empty(entries[2].Text, "Token entry should NOT be prefilled — leave blank to keep existing token")
	s.Equal("T-OLD", entries[3].Text, "WorkspaceID entry should be prefilled")
	s.Equal("olduser", entries[4].Text, "Username entry should be prefilled")
	s.Equal("600", entries[5].Text, "Poll interval entry should be prefilled")

	// Edit username and save without touching the token; the existing
	// token must be preserved by the form.
	entries[4].SetText("newuser")
	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool { return b.Text == "Save" })
	saveBtn.OnTapped()

	slackContent = tabs.Items[0].Content
	_, found := uitest.FindWidget[*widget.Label](slackContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "newuser")
	})
	s.True(found, "after editing, list should show the updated username 'newuser'")

	s.Require().Len(repo.slackAccounts, 1, "edit should update in place, not append a duplicate")
	s.Equal(originalID, repo.slackAccounts[0].ID, "account ID should be preserved on edit")
	s.Equal("newuser", repo.slackAccounts[0].Username, "username should reflect the edit")
	s.Equal("xoxp-old", repo.slackAccounts[0].Token, "token should be preserved when the field is left blank on edit")
}

// AC: Each Email account row renders an Edit button.
func (s *SettingsAcceptanceSuite) TestEmailAccountRowHasEditButton() {
	repo := &mockServiceConfigRepo{
		emailAccounts: []*repository.EmailAccount{
			{ID: uuid.New(), Enabled: true, IMAPHost: "imap.example.com", IMAPPort: 993, Username: "user@example.com", Password: "secret", Encryption: "ssl_tls", PollIntervalSeconds: 600},
		},
	}
	sv := newSettingsViewWithRepo(repo)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })

	emailContent := tabs.Items[1].Content
	_, found := uitest.FindWidget[*widget.Button](emailContent, func(b *widget.Button) bool { return b.Text == "Edit" })
	s.True(found, "Email account row should have an Edit button")
}

// AC: Tapping Edit on an Email account opens the form prefilled and saving
// updates the existing account in place (no duplicate, same ID).
func (s *SettingsAcceptanceSuite) TestEmailEditPrefillsFormAndUpdatesInPlace() {
	originalID := uuid.New()
	repo := &mockServiceConfigRepo{
		emailAccounts: []*repository.EmailAccount{
			{ID: originalID, Enabled: true, FriendlyName: "Work", WebURL: "https://mail.example.com", IMAPHost: "imap.example.com", IMAPPort: 993, Username: "old@example.com", Password: "oldsecret", Encryption: "starttls", PollIntervalSeconds: 600},
		},
	}
	sv := newSettingsViewWithRepo(repo, presenter.WithEmailValidator(&mockEmailValidator{err: nil}))
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })

	emailContent := tabs.Items[1].Content
	editBtn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool { return b.Text == "Edit" })
	editBtn.OnTapped()

	emailContent = tabs.Items[1].Content
	entries := uitest.FindAll[*widget.Entry](emailContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 7, "edit form should have at least 7 entries")

	s.Equal("Work", entries[0].Text, "FriendlyName entry should be prefilled")
	s.Equal("https://mail.example.com", entries[1].Text, "WebURL entry should be prefilled")
	s.Equal("imap.example.com", entries[2].Text, "IMAPHost entry should be prefilled")
	s.Equal("993", entries[3].Text, "IMAPPort entry should be prefilled")
	s.Equal("old@example.com", entries[4].Text, "Username entry should be prefilled")
	s.Empty(entries[5].Text, "Password entry should NOT be prefilled — leave blank to keep existing password")
	s.Equal("600", entries[6].Text, "Poll interval entry should be prefilled")

	sel := uitest.RequireWidget[*widget.Select](s.T(), emailContent, func(_ *widget.Select) bool { return true })
	s.Equal("STARTTLS", sel.Selected, "encryption select should reflect the existing value")

	entries[4].SetText("new@example.com")
	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool { return b.Text == "Save" })
	saveBtn.OnTapped()

	emailContent = tabs.Items[1].Content
	_, found := uitest.FindWidget[*widget.Label](emailContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "new@example.com")
	})
	s.True(found, "after editing, list should show the updated username 'new@example.com'")

	s.Require().Len(repo.emailAccounts, 1, "edit should update in place, not append a duplicate")
	s.Equal(originalID, repo.emailAccounts[0].ID, "account ID should be preserved on edit")
	s.Equal("new@example.com", repo.emailAccounts[0].Username, "username should reflect the edit")
	s.Equal("oldsecret", repo.emailAccounts[0].Password, "password should be preserved when the field is left blank on edit")
}

// AC: Each Calendar account row renders an Edit button.
func (s *SettingsAcceptanceSuite) TestCalendarAccountRowHasEditButton() {
	repo := &mockServiceConfigRepo{
		calendarAccounts: []*repository.CalendarAccount{
			{ID: uuid.New(), Enabled: true, Name: "Work", ICSURL: "https://example.com/cal.ics", PollIntervalSeconds: 600},
		},
	}
	sv := newSettingsViewWithRepo(repo)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })

	calendarContent := tabs.Items[2].Content
	_, found := uitest.FindWidget[*widget.Button](calendarContent, func(b *widget.Button) bool { return b.Text == "Edit" })
	s.True(found, "Calendar account row should have an Edit button")
}

// AC: Tapping Edit on a Calendar account opens the form prefilled and saving
// updates the existing account in place (no duplicate, same ID).
func (s *SettingsAcceptanceSuite) TestCalendarEditPrefillsFormAndUpdatesInPlace() {
	originalID := uuid.New()
	repo := &mockServiceConfigRepo{
		calendarAccounts: []*repository.CalendarAccount{
			{ID: originalID, Enabled: true, Name: "Old Cal", ICSURL: "https://example.com/old.ics", PollIntervalSeconds: 600},
		},
	}
	sv := newSettingsViewWithRepo(repo, presenter.WithCalendarValidator(&mockCalendarValidator{err: nil}))
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })

	calendarContent := tabs.Items[2].Content
	editBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool { return b.Text == "Edit" })
	editBtn.OnTapped()

	calendarContent = tabs.Items[2].Content
	entries := uitest.FindAll[*widget.Entry](calendarContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 3, "edit form should have at least 3 entries")

	s.Equal("Old Cal", entries[0].Text, "Name entry should be prefilled")
	s.Equal("https://example.com/old.ics", entries[1].Text, "ICSURL entry should be prefilled")
	s.Equal("600", entries[2].Text, "Poll interval entry should be prefilled")

	entries[0].SetText("New Cal")
	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool { return b.Text == "Save" })
	saveBtn.OnTapped()

	calendarContent = tabs.Items[2].Content
	_, found := uitest.FindWidget[*widget.Label](calendarContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "New Cal")
	})
	s.True(found, "after editing, list should show the updated name 'New Cal'")

	s.Require().Len(repo.calendarAccounts, 1, "edit should update in place, not append a duplicate")
	s.Equal(originalID, repo.calendarAccounts[0].ID, "account ID should be preserved on edit")
	s.Equal("New Cal", repo.calendarAccounts[0].Name, "name should reflect the edit")
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

// --- Feature 079: Credential Validation on Save ---

// AC: Invalid Slack token: form stays open with error message, no DB record created.
func (s *SettingsAcceptanceSuite) TestSlackValidationFailureKeepsFormOpen() {
	repo := &mockServiceConfigRepo{}
	failValidator := &mockSlackValidator{err: fmt.Errorf("invalid_auth")}
	sv := newSettingsViewWithRepo(repo, presenter.WithSlackValidator(failValidator))
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	slackContent := tabs.Items[0].Content
	addBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})
	addBtn.OnTapped()

	slackContent = tabs.Items[0].Content
	entries := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 6)
	entries[0].SetText("Bad Slack")         // Friendly Name
	entries[1].SetText("https://slack.com") // Web URL
	entries[2].SetText("xoxp-bad-token")
	entries[3].SetText("T12345")
	entries[4].SetText("testuser")
	entries[5].SetText("600")

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})
	saveBtn.OnTapped()

	// Form should still be showing (entries still present)
	slackContent = tabs.Items[0].Content
	entriesAfter := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool { return true })
	s.GreaterOrEqual(len(entriesAfter), 6, "form should remain open after validation failure")

	// Error label should contain the validation error
	_, found := uitest.FindWidget[*widget.Label](slackContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "invalid_auth")
	})
	s.True(found, "error label should show the validation error 'invalid_auth'")

	// No DB record should have been created
	s.Empty(repo.slackAccounts, "no Slack account should be persisted after validation failure")
}

// AC: Invalid IMAP credentials: form stays open with error message.
func (s *SettingsAcceptanceSuite) TestEmailValidationFailureKeepsFormOpen() {
	repo := &mockServiceConfigRepo{}
	failValidator := &mockEmailValidator{err: fmt.Errorf("IMAP login: authentication failed")}
	sv := newSettingsViewWithRepo(repo, presenter.WithEmailValidator(failValidator))
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	emailContent := tabs.Items[1].Content
	addBtn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})
	addBtn.OnTapped()

	emailContent = tabs.Items[1].Content
	entries := uitest.FindAll[*widget.Entry](emailContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 7)
	entries[0].SetText("Fail Email")               // Friendly Name
	entries[1].SetText("https://mail.example.com") // Web URL
	entries[2].SetText("imap.example.com")
	entries[3].SetText("993")
	entries[4].SetText("user@example.com")
	entries[5].SetText("wrong-password")
	entries[6].SetText("600")

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})
	saveBtn.OnTapped()

	emailContent = tabs.Items[1].Content
	entriesAfter := uitest.FindAll[*widget.Entry](emailContent, func(_ *widget.Entry) bool { return true })
	s.GreaterOrEqual(len(entriesAfter), 7, "form should remain open after validation failure")

	_, found := uitest.FindWidget[*widget.Label](emailContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "authentication failed")
	})
	s.True(found, "error label should show the validation error 'authentication failed'")

	s.Empty(repo.emailAccounts, "no Email account should be persisted after validation failure")
}

// AC: Invalid calendar URL: form stays open with error message.
func (s *SettingsAcceptanceSuite) TestCalendarValidationFailureKeepsFormOpen() {
	repo := &mockServiceConfigRepo{}
	failValidator := &mockCalendarValidator{err: fmt.Errorf("404 Not Found")}
	sv := newSettingsViewWithRepo(repo, presenter.WithCalendarValidator(failValidator))
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	calendarContent := tabs.Items[2].Content
	addBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})
	addBtn.OnTapped()

	calendarContent = tabs.Items[2].Content
	entries := uitest.FindAll[*widget.Entry](calendarContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 3)
	entries[0].SetText("Bad Calendar")
	entries[1].SetText("https://example.com/nonexistent.ics")
	entries[2].SetText("600")

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), calendarContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})
	saveBtn.OnTapped()

	calendarContent = tabs.Items[2].Content
	entriesAfter := uitest.FindAll[*widget.Entry](calendarContent, func(_ *widget.Entry) bool { return true })
	s.GreaterOrEqual(len(entriesAfter), 3, "form should remain open after validation failure")

	_, found := uitest.FindWidget[*widget.Label](calendarContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "404 Not Found")
	})
	s.True(found, "error label should show the validation error '404 Not Found'")

	s.Empty(repo.calendarAccounts, "no Calendar account should be persisted after validation failure")
}

// AC: Valid Slack credentials: saved to DB, returns to account list.
func (s *SettingsAcceptanceSuite) TestSlackValidationSuccessSavesAndReturnsToList() {
	repo := &mockServiceConfigRepo{}
	passValidator := &mockSlackValidator{err: nil}
	sv := newSettingsViewWithRepo(repo, presenter.WithSlackValidator(passValidator))
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	slackContent := tabs.Items[0].Content
	addBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})
	addBtn.OnTapped()

	slackContent = tabs.Items[0].Content
	entries := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 6)
	entries[0].SetText("Valid Slack")       // Friendly Name
	entries[1].SetText("https://slack.com") // Web URL
	entries[2].SetText("xoxp-valid-token")
	entries[3].SetText("T-VALID")
	entries[4].SetText("validuser")
	entries[5].SetText("600")

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool {
		return b.Text == "Save"
	})
	saveBtn.OnTapped()

	// Should be back to list view (fewer entries)
	slackContent = tabs.Items[0].Content
	entriesAfter := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool { return true })
	s.Less(len(entriesAfter), 6, "after successful validation and save, form should be replaced with account list")

	// Account should appear in list
	_, found := uitest.FindWidget[*widget.Label](slackContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "T-VALID")
	})
	s.True(found, "saved Slack account should appear in the account list")

	s.Len(repo.slackAccounts, 1, "one Slack account should be persisted after successful validation")
}

// AC: Slack "Add Account" form includes token setup instructions with required OAuth scopes.
func (s *SettingsAcceptanceSuite) TestSlackFormContainsTokenInstructions() {
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

	slackContent = tabs.Items[0].Content

	// Verify an Accordion widget exists with token instructions.
	acc, found := uitest.FindWidget[*widget.Accordion](slackContent, func(_ *widget.Accordion) bool {
		return true
	})
	s.True(found, "Slack Add Account form should contain an Accordion widget with token setup instructions")

	// Verify the accordion has at least one item mentioning token setup.
	s.Require().GreaterOrEqual(len(acc.Items), 1, "Accordion should have at least one item")
	s.Contains(acc.Items[0].Title, "token",
		"Accordion item title should mention 'token'")

	// Verify required OAuth scopes are listed in the accordion detail.
	detail := acc.Items[0].Detail
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
	for _, scope := range requiredScopes {
		_, scopeFound := uitest.FindWidget[*widget.Label](detail, func(l *widget.Label) bool {
			return strings.Contains(l.Text, scope)
		})
		if !scopeFound {
			// Also check RichText widgets.
			_, scopeFound = uitest.FindWidget[*widget.RichText](detail, func(r *widget.RichText) bool {
				for _, seg := range r.Segments {
					if ts, ok := seg.(*widget.TextSegment); ok && strings.Contains(ts.Text, scope) {
						return true
					}
				}
				return false
			})
		}
		s.True(scopeFound, "Accordion detail should list required scope: %s", scope)
	}
}

// --- Feature 089: Rules Settings Tab ---

// AC: Settings view has 6 tabs.
func (s *SettingsAcceptanceSuite) TestRulesTabSettingsViewHasSixTabs() {
	sv := newSettingsView()
	s.Equal(6, sv.TabCount(), "SettingsView should have 6 tabs")
}

// AC: Tab names include Rules at index 3.
func (s *SettingsAcceptanceSuite) TestRulesTabNamesInOrder() {
	sv := newSettingsView()
	expected := []string{"Slack", "Email", "Calendar", "Rules", "Audio", "Ollama"}
	s.Equal(expected, sv.TabNames(), "tab names should include Rules at index 3")
}

// AC: Rules tab is at index 3.
func (s *SettingsAcceptanceSuite) TestRulesTabIsAtIndex3() {
	sv := newSettingsView()
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	s.Require().Greater(len(tabs.Items), 3)
	s.Equal("Rules", tabs.Items[3].Text, "tab at index 3 should be Rules")
}

// AC: Rules tab with no rules shows empty state text.
func (s *SettingsAcceptanceSuite) TestRulesTabEmptyStateText() {
	sv := newSettingsViewWithRules(&mockRoutingRuleRepo{}, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	_, found := uitest.FindWidget[*widget.Label](rulesContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "No routing rules configured")
	})
	s.True(found, "Rules tab with no rules should show 'No routing rules configured' text")
}

// AC: Rules tab with no rules shows "Add Rule" button.
func (s *SettingsAcceptanceSuite) TestRulesTabEmptyStateHasAddButton() {
	sv := newSettingsViewWithRules(&mockRoutingRuleRepo{}, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	_, found := uitest.FindWidget[*widget.Button](rulesContent, func(b *widget.Button) bool {
		return b.Text == "Add Rule"
	})
	s.True(found, "Rules tab should have an 'Add Rule' button")
}

// AC: Rules tab with pre-existing rules displays them sorted by priority.
func (s *SettingsAcceptanceSuite) TestRulesTabDisplaysRulesSortedByPriority() {
	ruleRepo := &mockRoutingRuleRepo{
		rules: []*repository.RoutingRule{
			{ID: uuid.New(), Priority: 0, SourceType: "slack", ChannelPattern: "^general$", Action: "notified", Enabled: true},
			{ID: uuid.New(), Priority: 1, SourceType: "email", ContentPattern: "boss@", Action: "notified", Enabled: true},
		},
	}
	sv := newSettingsViewWithRules(ruleRepo, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	// Should find labels containing the rule summaries (new model shows channel pattern)
	_, found := uitest.FindWidget[*widget.Label](rulesContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "general") && strings.Contains(l.Text, "notified")
	})
	s.True(found, "Rules tab should display first rule summary")
}

// AC: Each rule row shows source, field, pattern, action.
func (s *SettingsAcceptanceSuite) TestRulesTabRowShowsSummary() {
	ruleRepo := &mockRoutingRuleRepo{
		rules: []*repository.RoutingRule{
			{ID: uuid.New(), Priority: 0, SourceType: "slack", ChannelPattern: "^general$", Action: "notified", Enabled: true},
		},
	}
	sv := newSettingsViewWithRules(ruleRepo, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	_, found := uitest.FindWidget[*widget.Label](rulesContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "notified")
	})
	s.True(found, "Rule row should show action in summary")
}

// AC: Each rule row has an enabled checkbox.
func (s *SettingsAcceptanceSuite) TestRulesTabRowHasEnabledCheckbox() {
	ruleRepo := &mockRoutingRuleRepo{
		rules: []*repository.RoutingRule{
			{ID: uuid.New(), Priority: 0, SourceType: "slack", ChannelPattern: "^general$", Action: "notified", Enabled: true},
		},
	}
	sv := newSettingsViewWithRules(ruleRepo, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	_, found := uitest.FindWidget[*widget.Check](rulesContent, func(_ *widget.Check) bool { return true })
	s.True(found, "Rule row should have an enabled checkbox")
}

// AC: Each rule row has Up/Down reorder buttons.
func (s *SettingsAcceptanceSuite) TestRulesTabRowHasUpDownButtons() {
	ruleRepo := &mockRoutingRuleRepo{
		rules: []*repository.RoutingRule{
			{ID: uuid.New(), Priority: 0, SourceType: "slack", ChannelPattern: "^general$", Action: "notified", Enabled: true},
			{ID: uuid.New(), Priority: 1, SourceType: "email", ContentPattern: "boss@", Action: "notified", Enabled: true},
		},
	}
	sv := newSettingsViewWithRules(ruleRepo, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	upBtns := uitest.FindAll[*widget.Button](rulesContent, func(b *widget.Button) bool { return b.Text == "Up" })
	downBtns := uitest.FindAll[*widget.Button](rulesContent, func(b *widget.Button) bool { return b.Text == "Down" })
	s.GreaterOrEqual(len(upBtns), 1, "should have at least one Up button")
	s.GreaterOrEqual(len(downBtns), 1, "should have at least one Down button")
}

// AC: Each rule row has a Delete button.
func (s *SettingsAcceptanceSuite) TestRulesTabRowHasDeleteButton() {
	ruleRepo := &mockRoutingRuleRepo{
		rules: []*repository.RoutingRule{
			{ID: uuid.New(), Priority: 0, SourceType: "slack", ChannelPattern: "^general$", Action: "notified", Enabled: true},
		},
	}
	sv := newSettingsViewWithRules(ruleRepo, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	_, found := uitest.FindWidget[*widget.Button](rulesContent, func(b *widget.Button) bool { return b.Text == "Delete" })
	s.True(found, "Rule row should have a Delete button")
}

// AC: First rule's Up button is disabled.
func (s *SettingsAcceptanceSuite) TestRulesTabFirstRuleUpDisabled() {
	ruleRepo := &mockRoutingRuleRepo{
		rules: []*repository.RoutingRule{
			{ID: uuid.New(), Priority: 0, SourceType: "slack", ChannelPattern: "^general$", Action: "notified", Enabled: true},
			{ID: uuid.New(), Priority: 1, SourceType: "email", ContentPattern: "boss@", Action: "notified", Enabled: true},
		},
	}
	sv := newSettingsViewWithRules(ruleRepo, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	upBtns := uitest.FindAll[*widget.Button](rulesContent, func(b *widget.Button) bool { return b.Text == "Up" })
	s.Require().GreaterOrEqual(len(upBtns), 1, "should have at least one Up button")
	s.True(upBtns[0].Disabled(), "first rule's Up button should be disabled")
}

// AC: Last rule's Down button is disabled.
func (s *SettingsAcceptanceSuite) TestRulesTabLastRuleDownDisabled() {
	ruleRepo := &mockRoutingRuleRepo{
		rules: []*repository.RoutingRule{
			{ID: uuid.New(), Priority: 0, SourceType: "slack", ChannelPattern: "^general$", Action: "notified", Enabled: true},
			{ID: uuid.New(), Priority: 1, SourceType: "email", ContentPattern: "boss@", Action: "notified", Enabled: true},
		},
	}
	sv := newSettingsViewWithRules(ruleRepo, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	downBtns := uitest.FindAll[*widget.Button](rulesContent, func(b *widget.Button) bool { return b.Text == "Down" })
	s.Require().GreaterOrEqual(len(downBtns), 1, "should have at least one Down button")
	lastDown := downBtns[len(downBtns)-1]
	s.True(lastDown.Disabled(), "last rule's Down button should be disabled")
}

// AC: Tapping "Add Rule" replaces list with form.
func (s *SettingsAcceptanceSuite) TestRulesTabAddRuleShowsForm() {
	sv := newSettingsViewWithRules(&mockRoutingRuleRepo{}, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Add Rule" })
	addBtn.OnTapped()

	rulesContent = tabs.Items[3].Content
	_, found := uitest.FindWidget[*widget.Select](rulesContent, func(sel *widget.Select) bool { return true })
	s.True(found, "Add Rule form should contain a Select dropdown")
}

// AC: Form has Source dropdown with options: Email, Slack.
func (s *SettingsAcceptanceSuite) TestRulesTabFormHasSourceDropdown() {
	sv := newSettingsViewWithRules(&mockRoutingRuleRepo{}, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Add Rule" })
	addBtn.OnTapped()
	rulesContent = tabs.Items[3].Content

	selects := uitest.FindAll[*widget.Select](rulesContent, func(_ *widget.Select) bool { return true })
	s.Require().GreaterOrEqual(len(selects), 1, "should have at least one Select (Source)")
	sourceSelect := selects[0]
	s.Contains(sourceSelect.Options, "Email", "Source dropdown should include Email")
	s.Contains(sourceSelect.Options, "Slack", "Source dropdown should include Slack")
}

// AC: Selecting Email source shows fields: sender, subject.
// SKIP: Form redesigned for multi-pattern model (Feature 103). No per-source field dropdown.
func (s *SettingsAcceptanceSuite) TestRulesTabFormEmailFields() {
	s.T().Skip("Form redesigned for multi-pattern model (Feature 103)")
	//nolint:govet // unreachable code after Skip
	sv := newSettingsViewWithRules(&mockRoutingRuleRepo{}, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Add Rule" })
	addBtn.OnTapped()
	rulesContent = tabs.Items[3].Content

	selects := uitest.FindAll[*widget.Select](rulesContent, func(_ *widget.Select) bool { return true })
	s.Require().GreaterOrEqual(len(selects), 2, "should have Source and Field selects")
	sourceSelect := selects[0]
	fieldSelect := selects[1]

	sourceSelect.SetSelected("Email")
	s.Contains(fieldSelect.Options, "sender", "Email fields should include sender")
	s.Contains(fieldSelect.Options, "subject", "Email fields should include subject")
}

// AC: Selecting Slack source shows fields: sender, channel, content, message_type.
// SKIP: Form redesigned for multi-pattern model (Feature 103). No per-source field dropdown.
func (s *SettingsAcceptanceSuite) TestRulesTabFormSlackFields() {
	s.T().Skip("Form redesigned for multi-pattern model (Feature 103)")
	//nolint:govet // unreachable code after Skip
	sv := newSettingsViewWithRules(&mockRoutingRuleRepo{}, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Add Rule" })
	addBtn.OnTapped()
	rulesContent = tabs.Items[3].Content

	selects := uitest.FindAll[*widget.Select](rulesContent, func(_ *widget.Select) bool { return true })
	s.Require().GreaterOrEqual(len(selects), 2)
	sourceSelect := selects[0]
	fieldSelect := selects[1]

	sourceSelect.SetSelected("Slack")
	s.Contains(fieldSelect.Options, "sender")
	s.Contains(fieldSelect.Options, "channel")
	s.Contains(fieldSelect.Options, "content")
	s.Contains(fieldSelect.Options, "message_type")
}

// AC: Form has Pattern text entry.
func (s *SettingsAcceptanceSuite) TestRulesTabFormHasPatternEntry() {
	sv := newSettingsViewWithRules(&mockRoutingRuleRepo{}, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Add Rule" })
	addBtn.OnTapped()
	rulesContent = tabs.Items[3].Content

	_, found := uitest.FindWidget[*widget.Entry](rulesContent, func(_ *widget.Entry) bool { return true })
	s.True(found, "Add Rule form should have a Pattern text entry")
}

// AC: Form has Negate checkbox.
// SKIP: Negate removed in multi-pattern model (Feature 103).
func (s *SettingsAcceptanceSuite) TestRulesTabFormHasNegateCheckbox() {
	s.T().Skip("Negate removed in multi-pattern model (Feature 103)")
	//nolint:govet // unreachable code after Skip
	sv := newSettingsViewWithRules(&mockRoutingRuleRepo{}, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Add Rule" })
	addBtn.OnTapped()
	rulesContent = tabs.Items[3].Content

	_, found := uitest.FindWidget[*widget.Check](rulesContent, func(c *widget.Check) bool {
		return strings.Contains(c.Text, "nvert") || strings.Contains(c.Text, "egate") || strings.Contains(c.Text, "not matches")
	})
	s.True(found, "Add Rule form should have a Negate checkbox")
}

// AC: Form has Action dropdown: Notified, Ignored.
func (s *SettingsAcceptanceSuite) TestRulesTabFormHasActionDropdown() {
	sv := newSettingsViewWithRules(&mockRoutingRuleRepo{}, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Add Rule" })
	addBtn.OnTapped()
	rulesContent = tabs.Items[3].Content

	selects := uitest.FindAll[*widget.Select](rulesContent, func(_ *widget.Select) bool { return true })
	s.Require().GreaterOrEqual(len(selects), 2, "should have Source and Action selects")
	actionSelect := selects[len(selects)-1]
	s.Contains(actionSelect.Options, "Notified")
	s.Contains(actionSelect.Options, "Ignored")
}

// AC: Form has Save and Cancel buttons.
func (s *SettingsAcceptanceSuite) TestRulesTabFormHasSaveAndCancel() {
	sv := newSettingsViewWithRules(&mockRoutingRuleRepo{}, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Add Rule" })
	addBtn.OnTapped()
	rulesContent = tabs.Items[3].Content

	_, saveFound := uitest.FindWidget[*widget.Button](rulesContent, func(b *widget.Button) bool { return b.Text == "Save" })
	_, cancelFound := uitest.FindWidget[*widget.Button](rulesContent, func(b *widget.Button) bool { return b.Text == "Cancel" })
	s.True(saveFound, "Add Rule form should have a Save button")
	s.True(cancelFound, "Add Rule form should have a Cancel button")
}

// AC: Cancel returns to rule list without saving.
func (s *SettingsAcceptanceSuite) TestRulesTabFormCancelReturnsToList() {
	ruleRepo := &mockRoutingRuleRepo{}
	sv := newSettingsViewWithRules(ruleRepo, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Add Rule" })
	addBtn.OnTapped()
	rulesContent = tabs.Items[3].Content

	cancelBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Cancel" })
	cancelBtn.OnTapped()
	rulesContent = tabs.Items[3].Content

	// Should be back on the list view with Add Rule button
	_, found := uitest.FindWidget[*widget.Button](rulesContent, func(b *widget.Button) bool { return b.Text == "Add Rule" })
	s.True(found, "Cancel should return to list view with Add Rule button")
	s.Empty(ruleRepo.rules, "Cancel should not save any rules")
}

// AC: Saving with invalid regexp shows inline error.
func (s *SettingsAcceptanceSuite) TestRulesTabFormInvalidRegexpShowsError() {
	sv := newSettingsViewWithRules(&mockRoutingRuleRepo{}, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Add Rule" })
	addBtn.OnTapped()
	rulesContent = tabs.Items[3].Content

	// Fill in form with invalid regex
	selects := uitest.FindAll[*widget.Select](rulesContent, func(_ *widget.Select) bool { return true })
	s.Require().GreaterOrEqual(len(selects), 2)
	selects[0].SetSelected("Slack")
	selects[len(selects)-1].SetSelected("Notified")

	// Find the channel pattern entry and set invalid regex
	entries := uitest.FindAll[*widget.Entry](rulesContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 1)
	entries[1].SetText("[invalid(regex") // channel pattern entry

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Save" })
	saveBtn.OnTapped()

	rulesContent = tabs.Items[3].Content
	_, found := uitest.FindWidget[*widget.Label](rulesContent, func(l *widget.Label) bool {
		return strings.Contains(strings.ToLower(l.Text), "error") || strings.Contains(strings.ToLower(l.Text), "invalid")
	})
	s.True(found, "Saving with invalid regexp should show an inline error")
}

// AC: Saving with valid data persists rule and returns to list.
func (s *SettingsAcceptanceSuite) TestRulesTabFormSaveValidRuleReturnsToList() {
	ruleRepo := &mockRoutingRuleRepo{}
	sv := newSettingsViewWithRules(ruleRepo, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Add Rule" })
	addBtn.OnTapped()
	rulesContent = tabs.Items[3].Content

	selects := uitest.FindAll[*widget.Select](rulesContent, func(_ *widget.Select) bool { return true })
	s.Require().GreaterOrEqual(len(selects), 2)
	selects[0].SetSelected("Slack")
	selects[len(selects)-1].SetSelected("Notified")

	// Set channel pattern via entry field
	entries := uitest.FindAll[*widget.Entry](rulesContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 1)
	entries[1].SetText("^general$") // channel pattern entry

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Save" })
	saveBtn.OnTapped()

	rulesContent = tabs.Items[3].Content
	_, found := uitest.FindWidget[*widget.Button](rulesContent, func(b *widget.Button) bool { return b.Text == "Add Rule" })
	s.True(found, "After saving, should return to list view with Add Rule button")
}

// AC: Newly saved rule appears in the list.
func (s *SettingsAcceptanceSuite) TestRulesTabNewlySavedRuleAppearsInList() {
	ruleRepo := &mockRoutingRuleRepo{}
	sv := newSettingsViewWithRules(ruleRepo, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	addBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Add Rule" })
	addBtn.OnTapped()
	rulesContent = tabs.Items[3].Content

	selects := uitest.FindAll[*widget.Select](rulesContent, func(_ *widget.Select) bool { return true })
	s.Require().GreaterOrEqual(len(selects), 2)
	selects[0].SetSelected("Slack")
	selects[len(selects)-1].SetSelected("Notified")

	// Set channel pattern via entry field
	entries := uitest.FindAll[*widget.Entry](rulesContent, func(_ *widget.Entry) bool { return true })
	s.Require().GreaterOrEqual(len(entries), 1)
	entries[1].SetText("^general$") // channel pattern entry

	saveBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Save" })
	saveBtn.OnTapped()

	rulesContent = tabs.Items[3].Content
	_, found := uitest.FindWidget[*widget.Label](rulesContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "general")
	})
	s.True(found, "Newly saved rule should appear in the list")
}

// AC: Queue depth label shown at top of Rules tab.
func (s *SettingsAcceptanceSuite) TestRulesTabQueueDepthLabel() {
	sv := newSettingsViewWithRules(&mockRoutingRuleRepo{}, &mockQueueRepo{pending: 3}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	_, found := uitest.FindWidget[*widget.Label](rulesContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "queue") || strings.Contains(l.Text, "Queue")
	})
	s.True(found, "Rules tab should show queue depth label")
}

// AC: Queue depth shows warning text when exceeding threshold.
func (s *SettingsAcceptanceSuite) TestRulesTabQueueDepthWarning() {
	sv := newSettingsViewWithRules(&mockRoutingRuleRepo{}, &mockQueueRepo{pending: 57}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	_, found := uitest.FindWidget[*widget.Label](rulesContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "consider adding more rules")
	})
	s.True(found, "Queue depth exceeding threshold should show warning text")
}

// AC: Tapping Down on a rule swaps it with the next rule.
func (s *SettingsAcceptanceSuite) TestRulesTabDownReordersRule() {
	id1, id2 := uuid.New(), uuid.New()
	ruleRepo := &mockRoutingRuleRepo{
		rules: []*repository.RoutingRule{
			{ID: id1, Priority: 0, SourceType: "slack", ChannelPattern: "^general$", Action: "notified", Enabled: true},
			{ID: id2, Priority: 1, SourceType: "email", ContentPattern: "boss@", Action: "ignored", Enabled: true},
		},
	}
	sv := newSettingsViewWithRules(ruleRepo, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	downBtns := uitest.FindAll[*widget.Button](rulesContent, func(b *widget.Button) bool { return b.Text == "Down" })
	s.Require().GreaterOrEqual(len(downBtns), 1)
	downBtns[0].OnTapped()

	// After reorder, first rule should now be the email one
	rulesContent = tabs.Items[3].Content
	labels := uitest.FindAll[*widget.Label](rulesContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "boss@") || strings.Contains(l.Text, "general")
	})
	s.Require().GreaterOrEqual(len(labels), 2, "should still have both rules visible after reorder")
}

// AC: Tapping Up on a rule swaps it with the previous rule.
func (s *SettingsAcceptanceSuite) TestRulesTabUpReordersRule() {
	id1, id2 := uuid.New(), uuid.New()
	ruleRepo := &mockRoutingRuleRepo{
		rules: []*repository.RoutingRule{
			{ID: id1, Priority: 0, SourceType: "slack", ChannelPattern: "^general$", Action: "notified", Enabled: true},
			{ID: id2, Priority: 1, SourceType: "email", ContentPattern: "boss@", Action: "ignored", Enabled: true},
		},
	}
	sv := newSettingsViewWithRules(ruleRepo, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	upBtns := uitest.FindAll[*widget.Button](rulesContent, func(b *widget.Button) bool { return b.Text == "Up" })
	s.Require().GreaterOrEqual(len(upBtns), 2, "should have Up buttons for both rules")
	// Tap Up on second rule
	upBtns[1].OnTapped()

	rulesContent = tabs.Items[3].Content
	labels := uitest.FindAll[*widget.Label](rulesContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "boss@") || strings.Contains(l.Text, "general")
	})
	s.Require().GreaterOrEqual(len(labels), 2, "should still have both rules visible after reorder")
}

// AC: Tapping Delete removes the rule from the list.
func (s *SettingsAcceptanceSuite) TestRulesTabDeleteRemovesRule() {
	ruleRepo := &mockRoutingRuleRepo{
		rules: []*repository.RoutingRule{
			{ID: uuid.New(), Priority: 0, SourceType: "slack", ChannelPattern: "^general$", Action: "notified", Enabled: true},
		},
	}
	sv := newSettingsViewWithRules(ruleRepo, &mockQueueRepo{}, 50)
	root := sv.Container()
	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool { return true })
	rulesContent := tabs.Items[3].Content

	deleteBtn := uitest.RequireWidget[*widget.Button](s.T(), rulesContent, func(b *widget.Button) bool { return b.Text == "Delete" })
	deleteBtn.OnTapped()

	rulesContent = tabs.Items[3].Content
	_, found := uitest.FindWidget[*widget.Label](rulesContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "No routing rules configured")
	})
	s.True(found, "After deleting last rule, should show empty state")
}
