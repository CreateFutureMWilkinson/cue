package ui

import (
	"context"
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
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
			return len(np.Cards())
		},
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			badge := canvas.NewRectangle(color.Transparent)
			badge.SetMinSize(fyne.NewSize(8, 8))
			badgeLabel := widget.NewLabel("")
			channelLabel := widget.NewLabel("")
			previewLabel := widget.NewLabel("")
			senderLabel := widget.NewLabel("")
			timeLabel := widget.NewLabel("")
			content := container.NewVBox(
				container.NewHBox(badge, badgeLabel, channelLabel),
				previewLabel,
				container.NewHBox(senderLabel, timeLabel),
			)
			return container.NewStack(bg, content)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			cards := np.Cards()
			if id >= len(cards) {
				return
			}
			card := cards[id]
			stack := obj.(*fyne.Container)
			bg := stack.Objects[0].(*canvas.Rectangle)
			content := stack.Objects[1].(*fyne.Container)
			row1 := content.Objects[0].(*fyne.Container)
			badge := row1.Objects[0].(*canvas.Rectangle)
			badgeLabel := row1.Objects[1].(*widget.Label)
			channelLabel := row1.Objects[2].(*widget.Label)
			previewLabel := content.Objects[1].(*widget.Label)
			row3 := content.Objects[2].(*fyne.Container)
			senderLabel := row3.Objects[0].(*widget.Label)
			timeLabel := row3.Objects[1].(*widget.Label)

			bg.FillColor = card.CardColor
			bg.Refresh()
			badge.FillColor = card.BadgeColor
			badge.Refresh()
			badgeLabel.SetText(fmt.Sprintf("[%.0f]", card.ImportanceScore))
			channelLabel.SetText(card.Channel)
			previewLabel.SetText(card.MessagePreview)
			senderLabel.SetText(card.Sender)
			timeLabel.SetText(card.RelativeTime)
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

	headerLabel := widget.NewLabel("Notifications")
	expandBtn := widget.NewButton("◀ expand", func() {
		np.ToggleExpanded()
	})
	header := container.NewHBox(headerLabel, expandBtn)
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

// CardCount returns the number of notification cards currently displayed.
func (p *NotificationPanel) CardCount() int {
	return 0
}

// cardAt returns the notification card at the given index, or nil if out of range.
func (p *NotificationPanel) cardAt(index int) *presenter.NotificationCard {
	cards := p.presenter.Cards()
	if index < 0 || index >= len(cards) {
		return nil
	}
	return &cards[index]
}

// RenderCard returns the rendered collapsed card widget for the notification at the given index.
func (p *NotificationPanel) RenderCard(index int) fyne.CanvasObject {
	card := p.cardAt(index)
	if card == nil {
		return nil
	}
	bg := canvas.NewRectangle(card.CardColor)
	badge := canvas.NewRectangle(card.BadgeColor)
	badge.SetMinSize(fyne.NewSize(8, 8))
	badgeLabel := widget.NewLabel(fmt.Sprintf("[%.0f]", card.ImportanceScore))
	channelLabel := widget.NewLabel(card.Channel)
	previewLabel := widget.NewLabel(card.MessagePreview)
	senderLabel := widget.NewLabel(card.Sender)
	timeLabel := widget.NewLabel(card.RelativeTime)
	content := container.NewVBox(
		container.NewHBox(badge, badgeLabel, channelLabel),
		previewLabel,
		container.NewHBox(senderLabel, timeLabel),
	)
	return container.NewStack(bg, content)
}

// RenderExpandedCard returns the rendered expanded card widget for the notification at the given index.
func (p *NotificationPanel) RenderExpandedCard(index int) fyne.CanvasObject {
	card := p.cardAt(index)
	if card == nil {
		return nil
	}
	bg := canvas.NewRectangle(card.CardColor)
	badgeLabel := widget.NewLabel(fmt.Sprintf("[%.1f]", card.ImportanceScore))
	sourceLabel := widget.NewLabel(card.Source)
	channelLabel := widget.NewLabel(card.Channel)
	senderLabel := widget.NewLabel(card.Sender)
	timeLabel := widget.NewLabel(card.RelativeTime)
	dismissBtn := widget.NewButton("Dismiss", func() {
		_ = p.presenter.DismissMessage(context.Background(), card.ID)
	})
	previewLabel := widget.NewLabel(card.FullContent)
	previewLabel.Wrapping = fyne.TextWrapWord
	row1 := container.NewHBox(badgeLabel, sourceLabel, channelLabel, senderLabel, timeLabel, dismissBtn)
	content := container.NewVBox(row1, previewLabel)
	return container.NewStack(bg, content)
}
