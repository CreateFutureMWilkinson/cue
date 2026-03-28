package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// newActivityLog creates a List widget for the activity log and hooks the
// presenter's OnUpdate callback to refresh it.
func newActivityLog(ap *presenter.ActivityPresenter) *widget.List {
	list := widget.NewList(
		func() int {
			return len(ap.Entries())
		},
		func() fyne.CanvasObject {
			return canvas.NewText("template", color.White)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			entries := ap.Entries()
			if id >= len(entries) {
				return
			}
			entry := entries[id]
			text := obj.(*canvas.Text)
			text.Text = fmt.Sprintf("[%s] %s: %s",
				entry.Timestamp.Format("15:04:05"), entry.Source, entry.Message)
			if entry.IsError {
				text.Color = color.RGBA{R: 255, G: 80, B: 80, A: 255}
			} else {
				text.Color = color.White
			}
			text.Refresh()
		},
	)

	ap.SetOnUpdate(func() {
		list.Refresh()
	})

	return list
}
