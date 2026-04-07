package ui

import (
	"context"
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// SettingsView provides a tabbed settings interface for the center column.
type SettingsView struct {
	tabs      *container.AppTabs
	container fyne.CanvasObject
}

// createEmailAccountForm creates the form UI for adding a new email account.
// onSaved is called after a successful save to restore the account list view.
func createEmailAccountForm(ssp *presenter.ServiceSettingsPresenter, onSaved func()) *fyne.Container {
	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("IMAP Host")
	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("IMAP Port")
	userEntry := widget.NewEntry()
	userEntry.SetPlaceHolder("Username")
	passEntry := widget.NewEntry()
	passEntry.SetPlaceHolder("Password")
	passEntry.Password = true
	encryptionSelect := widget.NewSelect(
		[]string{"SSL/TLS (Recommended)", "STARTTLS", "None"},
		nil,
	)
	encryptionSelect.SetSelected("SSL/TLS (Recommended)")
	pollEntry := widget.NewEntry()
	pollEntry.SetPlaceHolder("Poll Interval (seconds)")
	pollEntry.SetText(strconv.Itoa(presenter.DefaultPollInterval("email")))

	errorLabel := widget.NewLabel("")
	errorLabel.Hide()

	saveBtn := widget.NewButton("Save", nil)
	cancelBtn := widget.NewButton("Cancel", func() {})

	saveBtn.OnTapped = func() {
		if hostEntry.Text == "" || portEntry.Text == "" || userEntry.Text == "" || passEntry.Text == "" || pollEntry.Text == "" {
			errorLabel.SetText("All fields are required")
			errorLabel.Show()
			return
		}
		port, err := strconv.Atoi(portEntry.Text)
		if err != nil {
			errorLabel.SetText("IMAP port must be a number")
			errorLabel.Show()
			return
		}
		poll, err := strconv.Atoi(pollEntry.Text)
		if err != nil {
			errorLabel.SetText("Poll interval must be a number")
			errorLabel.Show()
			return
		}
		encMap := map[string]string{
			"SSL/TLS (Recommended)": "ssl_tls",
			"STARTTLS":              "starttls",
			"None":                  "none",
		}
		encryption := encMap[encryptionSelect.Selected]
		if encryption == "" {
			encryption = "ssl_tls"
		}
		acct := &repository.EmailAccount{
			ID:                  uuid.New(),
			Enabled:             true,
			IMAPHost:            hostEntry.Text,
			IMAPPort:            port,
			Username:            userEntry.Text,
			Password:            passEntry.Text,
			Encryption:          encryption,
			PollIntervalSeconds: poll,
		}
		errorLabel.SetText("Validating...")
		errorLabel.Show()
		if err := ssp.SaveEmailAccount(context.Background(), acct); err != nil {
			errorLabel.SetText(fmt.Sprintf("Error: %s", err))
			errorLabel.Show()
			return
		}
		errorLabel.Hide()
		onSaved()
	}

	return container.NewVBox(
		widget.NewLabel("Add Email Account"),
		hostEntry,
		portEntry,
		userEntry,
		passEntry,
		encryptionSelect,
		pollEntry,
		errorLabel,
		container.NewHBox(saveBtn, cancelBtn),
	)
}

// createSlackAccountForm creates the form UI for adding a new Slack account.
// onSaved is called after a successful save to restore the account list view.
func createSlackAccountForm(ssp *presenter.ServiceSettingsPresenter, onSaved func()) *fyne.Container {
	tokenEntry := widget.NewEntry()
	tokenEntry.SetPlaceHolder("User OAuth Token (xoxp-...)")
	tokenEntry.Password = true
	workspaceEntry := widget.NewEntry()
	workspaceEntry.SetPlaceHolder("Workspace ID")
	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Your Slack Username (@handle)")
	pollEntry := widget.NewEntry()
	pollEntry.SetPlaceHolder("Poll Interval (seconds)")
	pollEntry.SetText(strconv.Itoa(presenter.DefaultPollInterval("slack")))

	errorLabel := widget.NewLabel("")
	errorLabel.Hide()

	saveBtn := widget.NewButton("Save", nil)
	cancelBtn := widget.NewButton("Cancel", func() {
		onSaved()
	})

	saveBtn.OnTapped = func() {
		if tokenEntry.Text == "" || workspaceEntry.Text == "" || usernameEntry.Text == "" || pollEntry.Text == "" {
			errorLabel.SetText("All fields are required")
			errorLabel.Show()
			return
		}
		poll, err := strconv.Atoi(pollEntry.Text)
		if err != nil {
			errorLabel.SetText("Poll interval must be a number")
			errorLabel.Show()
			return
		}
		acct := &repository.SlackAccount{
			ID:                  uuid.New(),
			Enabled:             true,
			Token:               tokenEntry.Text,
			WorkspaceID:         workspaceEntry.Text,
			Username:            usernameEntry.Text,
			PollIntervalSeconds: poll,
		}
		errorLabel.SetText("Validating...")
		errorLabel.Show()
		if err := ssp.SaveSlackAccount(context.Background(), acct); err != nil {
			errorLabel.SetText(fmt.Sprintf("Error: %s", err))
			errorLabel.Show()
			return
		}
		errorLabel.Hide()
		onSaved()
	}

	return container.NewVBox(
		widget.NewLabel("Add Slack Account"),
		tokenEntry,
		workspaceEntry,
		usernameEntry,
		pollEntry,
		errorLabel,
		container.NewHBox(saveBtn, cancelBtn),
	)
}

// createCalendarAccountForm creates the form UI for adding a new calendar account.
// onSaved is called after a successful save to restore the account list view.
func createCalendarAccountForm(ssp *presenter.ServiceSettingsPresenter, onSaved func()) *fyne.Container {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Account Name")
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("ICS Calendar URL")
	pollEntry := widget.NewEntry()
	pollEntry.SetPlaceHolder("Poll Interval (seconds)")
	pollEntry.SetText(strconv.Itoa(presenter.DefaultPollInterval("calendar")))

	errorLabel := widget.NewLabel("")
	errorLabel.Hide()

	saveBtn := widget.NewButton("Save", nil)
	cancelBtn := widget.NewButton("Cancel", func() {
		onSaved()
	})

	saveBtn.OnTapped = func() {
		if nameEntry.Text == "" || urlEntry.Text == "" || pollEntry.Text == "" {
			errorLabel.SetText("All fields are required")
			errorLabel.Show()
			return
		}
		poll, err := strconv.Atoi(pollEntry.Text)
		if err != nil {
			errorLabel.SetText("Poll interval must be a number")
			errorLabel.Show()
			return
		}
		acct := &repository.CalendarAccount{
			ID:                  uuid.New(),
			Enabled:             true,
			Name:                nameEntry.Text,
			ICSURL:              urlEntry.Text,
			PollIntervalSeconds: poll,
		}
		errorLabel.SetText("Validating...")
		errorLabel.Show()
		if err := ssp.SaveCalendarAccount(context.Background(), acct); err != nil {
			errorLabel.SetText(fmt.Sprintf("Error: %s", err))
			errorLabel.Show()
			return
		}
		errorLabel.Hide()
		onSaved()
	}

	return container.NewVBox(
		widget.NewLabel("Add Calendar Account"),
		nameEntry,
		urlEntry,
		pollEntry,
		errorLabel,
		container.NewHBox(saveBtn, cancelBtn),
	)
}

// refreshAccountList clears the VBox and repopulates it from the presenter.
// Shows an empty state message if no accounts exist.
func refreshAccountList(list *fyne.Container, emptyMsg string, items []fyne.CanvasObject) {
	list.RemoveAll()
	if len(items) == 0 {
		list.Add(widget.NewLabel(emptyMsg))
	} else {
		for _, item := range items {
			list.Add(item)
		}
	}
}

// listAccountWidgets queries the presenter and returns label widgets for each account.
// Returns nil on error or if no accounts exist. Safe to call with nil presenter.
func listAccountWidgets(ssp *presenter.ServiceSettingsPresenter, accountType string) (widgets []fyne.CanvasObject) {
	if ssp == nil {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			widgets = nil
		}
	}()
	switch accountType {
	case "slack":
		accts, err := ssp.ListSlackAccounts(context.Background())
		if err != nil {
			return nil
		}
		for _, a := range accts {
			widgets = append(widgets, widget.NewLabel(fmt.Sprintf("Slack: %s (@%s)", a.WorkspaceID, a.Username)))
		}
	case "email":
		accts, err := ssp.ListEmailAccounts(context.Background())
		if err != nil {
			return nil
		}
		for _, a := range accts {
			widgets = append(widgets, widget.NewLabel(fmt.Sprintf("Email: %s (%s:%d)", a.Username, a.IMAPHost, a.IMAPPort)))
		}
	case "calendar":
		accts, err := ssp.ListCalendarAccounts(context.Background())
		if err != nil {
			return nil
		}
		for _, a := range accts {
			widgets = append(widgets, widget.NewLabel(fmt.Sprintf("Calendar: %s", a.Name)))
		}
	}
	return widgets
}

// NewSettingsView creates a SettingsView with tabs for Slack, Email, Calendar, Audio, and Ollama.
// The onClose callback is invoked when the user taps the Done button to exit settings.
func NewSettingsView(
	sp *presenter.SettingsPresenter,
	ssp *presenter.ServiceSettingsPresenter,
	ollamaCfg config.OllamaConfig,
	onClose func(),
) *SettingsView {
	// Slack tab with dynamic content switching between account list and add form
	slackAccountList := container.NewVBox()
	slackAddBtn := widget.NewButton("Add Account", nil)

	refreshSlack := func() {
		refreshAccountList(slackAccountList, "No Slack accounts configured. Tap Add Account to get started.", listAccountWidgets(ssp, "slack"))
	}

	buildSlackListContent := func() fyne.CanvasObject {
		refreshSlack()
		return container.NewBorder(
			widget.NewLabel("Slack Accounts"),
			slackAddBtn,
			nil, nil,
			container.NewVScroll(slackAccountList),
		)
	}

	slackTab := container.NewTabItem("Slack", buildSlackListContent())

	slackAddBtn.OnTapped = func() {
		slackTab.Content = createSlackAccountForm(ssp, func() {
			slackTab.Content = buildSlackListContent()
		})
	}

	// Email tab with dynamic content switching between account list and add form
	emailAccountList := container.NewVBox()
	emailAddBtn := widget.NewButton("Add Account", nil)

	refreshEmail := func() {
		refreshAccountList(emailAccountList, "No Email accounts configured. Tap Add Account to get started.", listAccountWidgets(ssp, "email"))
	}

	buildEmailListContent := func() fyne.CanvasObject {
		refreshEmail()
		return container.NewBorder(
			widget.NewLabel("Email Accounts"),
			emailAddBtn,
			nil, nil,
			container.NewVScroll(emailAccountList),
		)
	}

	emailTab := container.NewTabItem("Email", buildEmailListContent())

	emailAddBtn.OnTapped = func() {
		emailTab.Content = createEmailAccountForm(ssp, func() {
			emailTab.Content = buildEmailListContent()
		})
	}
	// Calendar tab with dynamic content switching between account list and add form
	calendarAccountList := container.NewVBox()
	calendarAddBtn := widget.NewButton("Add Account", nil)

	refreshCalendar := func() {
		refreshAccountList(calendarAccountList, "No Calendar accounts configured. Tap Add Account to get started.", listAccountWidgets(ssp, "calendar"))
	}

	buildCalendarListContent := func() fyne.CanvasObject {
		refreshCalendar()
		return container.NewBorder(
			widget.NewLabel("Calendar Accounts"),
			calendarAddBtn,
			nil, nil,
			container.NewVScroll(calendarAccountList),
		)
	}

	calendarTab := container.NewTabItem("Calendar", buildCalendarListContent())

	calendarAddBtn.OnTapped = func() {
		calendarTab.Content = createCalendarAccountForm(ssp, func() {
			calendarTab.Content = buildCalendarListContent()
		})
	}
	volumeLabel := widget.NewLabel(fmt.Sprintf("Notification Volume: %d%%", sp.Volume()))
	volumeSlider := &widget.Slider{
		Min:   0,
		Max:   100,
		Step:  1,
		Value: float64(sp.Volume()),
		OnChanged: func(v float64) {
			vol := int(v)
			sp.SetVolume(vol)
			volumeLabel.SetText(fmt.Sprintf("Notification Volume: %d%%", vol))
		},
	}
	timerVolumeLabel := widget.NewLabel(fmt.Sprintf("Timer Volume: %d%%", sp.TimerVolume()))
	timerVolumeSlider := &widget.Slider{
		Min:   0,
		Max:   100,
		Step:  1,
		Value: float64(sp.TimerVolume()),
		OnChanged: func(v float64) {
			vol := int(v)
			sp.SetTimerVolume(vol)
			timerVolumeLabel.SetText(fmt.Sprintf("Timer Volume: %d%%", vol))
		},
	}
	audioContent := container.NewVBox(
		widget.NewLabel("Audio Settings"),
		volumeLabel,
		volumeSlider,
		timerVolumeLabel,
		timerVolumeSlider,
	)
	audioTab := container.NewTabItem("Audio", audioContent)
	ollamaContent := container.NewVBox(
		widget.NewLabel("Ollama Settings"),
		widget.NewLabel(fmt.Sprintf("Host: %s", ollamaCfg.Host)),
		widget.NewLabel(fmt.Sprintf("Port: %d", ollamaCfg.Port)),
		widget.NewLabel(fmt.Sprintf("Inference Model: %s", ollamaCfg.InferenceModel)),
		widget.NewLabel(fmt.Sprintf("Embedding Model: %s", ollamaCfg.EmbeddingModel)),
		widget.NewLabel(fmt.Sprintf("Timeout: %ds", ollamaCfg.TimeoutSeconds)),
	)
	ollamaTab := container.NewTabItem("Ollama", ollamaContent)

	tabs := container.NewAppTabs(slackTab, emailTab, calendarTab, audioTab, ollamaTab)

	doneBtn := widget.NewButton("Done", onClose)

	sv := &SettingsView{
		tabs:      tabs,
		container: container.NewBorder(nil, doneBtn, nil, nil, tabs),
	}
	return sv
}

// Container returns the Fyne canvas object for the settings view.
func (sv *SettingsView) Container() fyne.CanvasObject {
	return sv.container
}

// TabCount returns the number of tabs in the settings view.
func (sv *SettingsView) TabCount() int {
	return len(sv.tabs.Items)
}

// TabNames returns the names of all tabs in order.
func (sv *SettingsView) TabNames() []string {
	names := make([]string, len(sv.tabs.Items))
	for i, tab := range sv.tabs.Items {
		names[i] = tab.Text
	}
	return names
}
