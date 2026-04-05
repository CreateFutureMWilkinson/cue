package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// newNotificationPane creates a List widget displaying notification rows.
func newNotificationPane(np *presenter.NotificationPresenter, win fyne.Window) *widget.List {
	list := widget.NewList(
		func() int {
			return len(np.Messages())
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			rows := np.Messages()
			if id >= len(rows) {
				return
			}
			row := rows[id]
			obj.(*widget.Label).SetText(
				fmt.Sprintf("[%s] %s | %s | %s", row.Source, row.Sender, row.Channel, row.Preview),
			)
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		detail, err := np.Select(id)
		if err != nil {
			return
		}

		resolveBtn := widget.NewButton("Resolve", func() {
			_ = np.Resolve(context.Background(), detail.ID)
			list.UnselectAll()
			list.Refresh()
		})

		content := container.NewVBox(
			widget.NewLabel(fmt.Sprintf("IS: %.1f  CS: %.2f", detail.ImportanceScore, detail.ConfidenceScore)),
			widget.NewLabel(fmt.Sprintf("Created: %s", detail.CreatedAt.Format("2006-01-02 15:04:05"))),
			widget.NewLabel(detail.Content),
			resolveBtn,
		)

		d := dialog.NewCustom("Notification Detail", "Close", content, win)
		d.Show()
		list.UnselectAll()
	}

	return list
}
