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
	friendlyNameEntry := widget.NewEntry()
	friendlyNameEntry.SetPlaceHolder("Friendly Name")
	webURLEntry := widget.NewEntry()
	webURLEntry.SetPlaceHolder("Web URL")
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
	cancelBtn := widget.NewButton("Cancel", func() {
		onSaved()
	})

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
			Enabled:             true,
			FriendlyName:        friendlyNameEntry.Text,
			WebURL:              webURLEntry.Text,
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
		friendlyNameEntry,
		webURLEntry,
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
	friendlyNameEntry := widget.NewEntry()
	friendlyNameEntry.SetPlaceHolder("Friendly Name")
	webURLEntry := widget.NewEntry()
	webURLEntry.SetPlaceHolder("Web URL")
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
			Enabled:             true,
			FriendlyName:        friendlyNameEntry.Text,
			WebURL:              webURLEntry.Text,
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

	tokenInstructions := widget.NewAccordion(
		widget.NewAccordionItem("How to get a token", container.NewVBox(
			widget.NewLabel("1. Go to https://api.slack.com/apps"),
			widget.NewLabel("2. Create a new app (or select existing) for your workspace"),
			widget.NewLabel("3. Add the following User Token Scopes under OAuth & Permissions:"),
			widget.NewLabel("   - channels:history"),
			widget.NewLabel("   - channels:read"),
			widget.NewLabel("   - groups:history"),
			widget.NewLabel("   - groups:read"),
			widget.NewLabel("   - im:history"),
			widget.NewLabel("   - im:read"),
			widget.NewLabel("   - mpim:history"),
			widget.NewLabel("   - mpim:read"),
			widget.NewLabel("   - users:read"),
			widget.NewLabel("4. Install the app to your workspace"),
			widget.NewLabel("5. Copy the User OAuth Token (starts with xoxp-)"),
		)),
	)

	return container.NewVBox(
		widget.NewLabel("Add Slack Account"),
		tokenInstructions,
		friendlyNameEntry,
		webURLEntry,
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
			id := a.ID // capture for closure
			label := widget.NewLabel(fmt.Sprintf("Slack: %s (@%s)", a.WorkspaceID, a.Username))
			deleteBtn := widget.NewButton("Delete", func() {
				_ = ssp.DeleteSlackAccount(context.Background(), id)
			})
			widgets = append(widgets, container.NewHBox(label, deleteBtn))
		}
	case "email":
		accts, err := ssp.ListEmailAccounts(context.Background())
		if err != nil {
			return nil
		}
		for _, a := range accts {
			id := a.ID // capture for closure
			label := widget.NewLabel(fmt.Sprintf("Email: %s (%s:%d)", a.Username, a.IMAPHost, a.IMAPPort))
			deleteBtn := widget.NewButton("Delete", func() {
				_ = ssp.DeleteEmailAccount(context.Background(), id)
			})
			widgets = append(widgets, container.NewHBox(label, deleteBtn))
		}
	case "calendar":
		accts, err := ssp.ListCalendarAccounts(context.Background())
		if err != nil {
			return nil
		}
		for _, a := range accts {
			id := a.ID // capture for closure
			label := widget.NewLabel(fmt.Sprintf("Calendar: %s", a.Name))
			deleteBtn := widget.NewButton("Delete", func() {
				_ = ssp.DeleteCalendarAccount(context.Background(), id)
			})
			widgets = append(widgets, container.NewHBox(label, deleteBtn))
		}
	}
	return widgets
}

// buildAddRuleForm creates the form UI for adding a new routing rule.
func buildAddRuleForm(rp *presenter.RulesPresenter, rulesTab *container.TabItem, buildList func() fyne.CanvasObject) fyne.CanvasObject {
	sourceTypeMap := map[string]string{"Email": "email", "Slack": "slack"}
	actionMap := map[string]string{"Notified": "notified", "Ignored": "ignored"}

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Rule Name (optional)")

	sourceTypeSelect := widget.NewSelect([]string{"Email", "Slack"}, nil)

	channelPatternEntry := widget.NewEntry()
	channelPatternEntry.SetPlaceHolder("Channel Pattern (regex, optional)")

	contentPatternEntry := widget.NewEntry()
	contentPatternEntry.SetPlaceHolder("Content Pattern (regex, optional)")

	messageTypeEntry := widget.NewEntry()
	messageTypeEntry.SetPlaceHolder("Message Type (e.g. channel_join, optional)")

	actionSelect := widget.NewSelect([]string{"Notified", "Ignored"}, nil)

	errorLabel := widget.NewLabel("")
	errorLabel.Hide()

	saveBtn := widget.NewButton("Save", nil)
	cancelBtn := widget.NewButton("Cancel", func() {
		rulesTab.Content = buildList()
	})

	saveBtn.OnTapped = func() {
		sourceType := sourceTypeMap[sourceTypeSelect.Selected]
		action := actionMap[actionSelect.Selected]

		rules, _ := rp.ListRules(context.Background())

		rule := &repository.RoutingRule{
			ID:             uuid.New(),
			Name:           nameEntry.Text,
			Priority:       len(rules),
			SourceType:     sourceType,
			ChannelPattern: channelPatternEntry.Text,
			ContentPattern: contentPatternEntry.Text,
			MessageType:    messageTypeEntry.Text,
			Action:         action,
			Enabled:        true,
		}

		if err := rp.SaveRule(context.Background(), rule); err != nil {
			errorLabel.SetText(fmt.Sprintf("Error: %s", err))
			errorLabel.Show()
			return
		}
		errorLabel.Hide()
		rulesTab.Content = buildList()
	}

	return container.NewVBox(
		widget.NewLabel("Add Rule"),
		nameEntry,
		sourceTypeSelect,
		channelPatternEntry,
		contentPatternEntry,
		messageTypeEntry,
		actionSelect,
		errorLabel,
		container.NewHBox(saveBtn, cancelBtn),
	)
}

// NewSettingsView creates a SettingsView with tabs for Slack, Email, Calendar, Rules, Audio, and Ollama.
// The onClose callback is invoked when the user taps the Done button to exit settings.
func NewSettingsView(
	sp *presenter.SettingsPresenter,
	ssp *presenter.ServiceSettingsPresenter,
	rp *presenter.RulesPresenter,
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

	// Rules tab with dynamic content switching between rule list and add form
	rulesTab := container.NewTabItem("Rules", widget.NewLabel("Loading..."))

	var buildRulesListContent func() fyne.CanvasObject
	buildRulesListContent = func() fyne.CanvasObject {
		if rp == nil {
			return container.NewVBox(widget.NewLabel("Rules"))
		}
		ruleList := container.NewVBox()

		// Queue depth label at top
		depth, _ := rp.QueueDepth(context.Background())
		threshold := rp.QueueWarningThreshold()
		var queueLabel *widget.Label
		if depth > threshold {
			queueLabel = widget.NewLabel(fmt.Sprintf("\u26a0 Ollama queue: %d pending \u2014 consider adding more rules", depth))
		} else {
			queueLabel = widget.NewLabel(fmt.Sprintf("Ollama queue: %d pending", depth))
		}

		rules, _ := rp.ListRules(context.Background())
		if len(rules) == 0 {
			ruleList.Add(widget.NewLabel("No routing rules configured"))
		} else {
			for idx, rule := range rules {
				ruleIdx := idx
				r := rule
				// Summary label
				actionStr := r.Action
				parts := []string{r.SourceType}
				if r.ChannelPattern != "" {
					parts = append(parts, "ch:"+r.ChannelPattern)
				}
				if r.ContentPattern != "" {
					parts = append(parts, "ct:"+r.ContentPattern)
				}
				if r.MessageType != "" {
					parts = append(parts, "mt:"+r.MessageType)
				}
				label := r.Name
				if label == "" {
					label = fmt.Sprintf("[%s]", parts[0])
					if len(parts) > 1 {
						for _, p := range parts[1:] {
							label += " " + p
						}
					}
				}
				summary := fmt.Sprintf("%s \u2192 %s", label, actionStr)
				summaryLabel := widget.NewLabel(summary)

				// Enabled checkbox
				enabledCheck := widget.NewCheck("", func(checked bool) {
					_ = rp.ToggleRule(context.Background(), r.ID, checked)
				})
				enabledCheck.Checked = r.Enabled

				// Up button
				upBtn := widget.NewButton("Up", func() {
					_ = rp.ReorderRule(context.Background(), r.ID, ruleIdx-1)
					rulesTab.Content = buildRulesListContent()
				})
				if ruleIdx == 0 {
					upBtn.Disable()
				}

				// Down button
				downBtn := widget.NewButton("Down", func() {
					_ = rp.ReorderRule(context.Background(), r.ID, ruleIdx+1)
					rulesTab.Content = buildRulesListContent()
				})
				if ruleIdx == len(rules)-1 {
					downBtn.Disable()
				}

				// Delete button
				deleteBtn := widget.NewButton("Delete", func() {
					_ = rp.DeleteRule(context.Background(), r.ID)
					rulesTab.Content = buildRulesListContent()
				})

				row := container.NewHBox(summaryLabel, enabledCheck, upBtn, downBtn, deleteBtn)
				ruleList.Add(row)
			}
		}

		addBtn := widget.NewButton("Add Rule", func() {
			rulesTab.Content = buildAddRuleForm(rp, rulesTab, buildRulesListContent)
		})

		return container.NewBorder(
			queueLabel,
			addBtn,
			nil, nil,
			container.NewVScroll(ruleList),
		)
	}

	rulesTab.Content = buildRulesListContent()

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

	tabs := container.NewAppTabs(slackTab, emailTab, calendarTab, rulesTab, audioTab, ollamaTab)

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
