package ui

import (
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

// NewSettingsView creates a SettingsView with tabs for Slack, Email, Audio, and Ollama.
func NewSettingsView(
	sp *presenter.SettingsPresenter,
	ssp *presenter.ServiceSettingsPresenter,
	ollamaCfg config.OllamaConfig,
) *SettingsView {
	slackTab := container.NewTabItem("Slack", widget.NewLabel("Slack Accounts"))
	emailTab := container.NewTabItem("Email", widget.NewLabel("Email Accounts"))
	audioTab := container.NewTabItem("Audio", widget.NewLabel("Audio Settings"))
	ollamaTab := container.NewTabItem("Ollama", widget.NewLabel("Ollama Settings"))

	tabs := container.NewAppTabs(slackTab, emailTab, audioTab, ollamaTab)

	sv := &SettingsView{
		tabs:      tabs,
		container: tabs,
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
