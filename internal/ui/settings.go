package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

const (
	settingsWindowWidth  = 400
	settingsWindowHeight = 200
)

// showSettings opens a standalone settings panel with a volume slider.
func showSettings(sp *presenter.SettingsPresenter, app fyne.App) {
	win := app.NewWindow("Settings")
	win.Resize(fyne.NewSize(settingsWindowWidth, settingsWindowHeight))

	volumeLabel := widget.NewLabel(fmt.Sprintf("Volume: %d", sp.Volume()))

	volumeSlider := widget.NewSlider(0, 100)
	volumeSlider.Value = float64(sp.Volume())
	volumeSlider.Step = 1
	volumeSlider.OnChanged = func(val float64) {
		sp.SetVolume(int(val))
		volumeLabel.SetText(fmt.Sprintf("Volume: %d", sp.Volume()))
	}

	content := container.NewVBox(
		widget.NewLabel("Audio Settings"),
		volumeLabel,
		volumeSlider,
	)

	win.SetContent(content)
	win.Show()
}
