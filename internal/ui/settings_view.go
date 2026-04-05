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

// NewSettingsView creates a SettingsView with tabs for Slack, Email, Audio, and Ollama.
func NewSettingsView(
	sp *presenter.SettingsPresenter,
	ssp *presenter.ServiceSettingsPresenter,
	ollamaCfg config.OllamaConfig,
) *SettingsView {
	slackAccountList := container.NewVBox()
	addSlackBtn := widget.NewButton("Add Account", func() {})
	slackContent := container.NewBorder(
		widget.NewLabel("Slack Accounts"),
		addSlackBtn,
		nil, nil,
		container.NewVScroll(slackAccountList),
	)
	slackTab := container.NewTabItem("Slack", slackContent)
	emailTab := container.NewTabItem("Email", widget.NewLabel("Email Accounts"))
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
