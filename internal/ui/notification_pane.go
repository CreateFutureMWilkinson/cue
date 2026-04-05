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

// NotificationPanel is the redesigned notification panel widget.
type NotificationPanel struct {
	presenter *presenter.NotificationPresenter
	window    fyne.Window
	root      fyne.CanvasObject
}

// NewNotificationPanel creates a new notification panel.
func NewNotificationPanel(np *presenter.NotificationPresenter, win fyne.Window) *NotificationPanel {
	// Create the notification list widget
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

	// Add click handler for detail dialog
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

	header := widget.NewLabel("Notifications")
	root := container.NewBorder(header, nil, nil, nil, list)

	return &NotificationPanel{
		presenter: np,
		window:    win,
		root:      root,
	}
}

// IsExpanded delegates to the presenter's IsExpanded.
func (p *NotificationPanel) IsExpanded() bool {
	return p.presenter.IsExpanded()
}

// ToggleExpand delegates to the presenter's ToggleExpanded.
func (p *NotificationPanel) ToggleExpand() {
	p.presenter.ToggleExpanded()
}

// Container returns the root canvas object for embedding in layouts.
func (p *NotificationPanel) Container() fyne.CanvasObject {
	return p.root
}

// RenderCard returns the rendered card widget for the notification at the given index.
func (p *NotificationPanel) RenderCard(index int) fyne.CanvasObject {
	return nil // stub — not implemented
}
