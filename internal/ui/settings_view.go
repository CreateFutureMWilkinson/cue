package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// SettingsView provides a tabbed settings interface for the center column.
type SettingsView struct {
	tabs      *container.AppTabs
	container fyne.CanvasObject
}

// newAccountTab creates a tab with a list of accounts and an "Add Account" button.
func newAccountTab(title string, onAdd func()) *container.TabItem {
	accountList := container.NewVBox()
	addBtn := widget.NewButton("Add Account", onAdd)
	content := container.NewBorder(
		widget.NewLabel(title+" Accounts"),
		addBtn,
		nil, nil,
		container.NewVScroll(accountList),
	)
	return container.NewTabItem(title, content)
}

// NewSettingsView creates a SettingsView with tabs for Slack, Email, Calendar, Audio, and Ollama.
// The onClose callback is invoked when the user taps the Done button to exit settings.
func NewSettingsView(
	sp *presenter.SettingsPresenter,
	ssp *presenter.ServiceSettingsPresenter,
	ollamaCfg config.OllamaConfig,
	onClose func(),
) *SettingsView {
	slackTab := newAccountTab("Slack", func() {})
	emailAddBtn := widget.NewButton("Add Account", nil)
	emailTab := container.NewTabItem("Email", container.NewBorder(
		widget.NewLabel("Email Accounts"),
		emailAddBtn,
		nil, nil,
		container.NewVScroll(container.NewVBox()),
	))
	emailAddBtn.OnTapped = func() {
		hostEntry := widget.NewEntry()
		hostEntry.SetPlaceHolder("IMAP Host")
		portEntry := widget.NewEntry()
		portEntry.SetPlaceHolder("IMAP Port")
		userEntry := widget.NewEntry()
		userEntry.SetPlaceHolder("Username")
		passEntry := widget.NewEntry()
		passEntry.SetPlaceHolder("Password")
		passEntry.Password = true
		pollEntry := widget.NewEntry()
		pollEntry.SetPlaceHolder("Poll Interval (seconds)")

		saveBtn := widget.NewButton("Save", func() {})
		cancelBtn := widget.NewButton("Cancel", func() {})

		emailTab.Content = container.NewVBox(
			widget.NewLabel("Add Email Account"),
			hostEntry,
			portEntry,
			userEntry,
			passEntry,
			pollEntry,
			container.NewHBox(saveBtn, cancelBtn),
		)
	}
	calendarTab := newAccountTab("Calendar", func() {})
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
	audioContent := container.NewVBox(
		widget.NewLabel("Audio Settings"),
		volumeLabel,
		volumeSlider,
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
