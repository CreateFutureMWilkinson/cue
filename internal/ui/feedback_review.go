package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// showFeedbackReview opens a modal window for reviewing buffered messages.
func showFeedbackReview(fp *presenter.FeedbackPresenter, app fyne.App) {
	ctx := context.Background()
	if err := fp.Load(ctx); err != nil {
		return
	}

	win := app.NewWindow("Feedback Review")
	win.Resize(fyne.NewSize(600, 400))

	counterLabel := widget.NewLabel("")
	contentLabel := widget.NewLabel("")
	contentLabel.Wrapping = fyne.TextWrapWord
	detailLabel := widget.NewLabel("")
	notesEntry := widget.NewMultiLineEntry()
	notesEntry.SetPlaceHolder("Optional notes...")

	var refresh func()
	refresh = func() {
		if !fp.HasCurrent() {
			counterLabel.SetText("No more messages")
			contentLabel.SetText("")
			detailLabel.SetText("")
			return
		}
		item := fp.Current()
		counterLabel.SetText(fp.Counter() + " buffered messages reviewed")
		contentLabel.SetText(item.Content)
		detailLabel.SetText(fmt.Sprintf("Source: %s | Sender: %s | Channel: %s | IS: %.1f | CS: %.2f",
			item.Source, item.Sender, item.Channel, item.ImportanceScore, item.ConfidenceScore))
		notesEntry.SetText("")
	}

	// Rating buttons 0-10.
	ratingButtons := make([]fyne.CanvasObject, 11)
	for i := range 11 {
		rating := i
		ratingButtons[i] = widget.NewButton(fmt.Sprintf("%d", rating), func() {
			var feedback *string
			if text := notesEntry.Text; text != "" {
				feedback = &text
			}
			_ = fp.SaveRating(ctx, rating, feedback)
			refresh()
		})
	}
	ratingRow := container.NewHBox(ratingButtons...)

	skipBtn := widget.NewButton("Skip", func() {
		fp.Skip()
		refresh()
	})

	deleteBtn := widget.NewButton("Delete", func() {
		_ = fp.Delete(ctx)
		refresh()
	})

	actionRow := container.NewHBox(skipBtn, deleteBtn)

	content := container.NewVBox(
		counterLabel,
		detailLabel,
		contentLabel,
		widget.NewLabel("Rate (0-10):"),
		ratingRow,
		notesEntry,
		container.NewHBox(widget.NewLabel("Actions:"), actionRow),
	)

	win.SetContent(container.NewVScroll(content))
	refresh()
	win.Show()
}
