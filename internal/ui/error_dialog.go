package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/uierror"
)

// ShowAdapterError renders a presenter-facing error in a Fyne dialog
// after classifying it through internal/uierror. ErrCodeUnauthorized
// surfaces as a "Restart and re-pair" custom-confirm; every other
// error kind renders as a standard information dialog.
//
// Per Feature 107 Decision 13 / Position A: adapters return raw
// wrapped errors; this helper is the single boundary at which they
// are classified for the user. UI callbacks call it on terminal error
// paths.
//
// nil errors are silently dropped so call sites can do the simple
// `ShowAdapterError(win, err)` after every presenter call without
// the early-return check.
func ShowAdapterError(parent fyne.Window, err error) {
	if err == nil || parent == nil {
		return
	}
	d := uierror.Classify(err)
	if d.ActionRetryRePair {
		dlg := dialog.NewCustomConfirm(
			d.Title,
			"Restart and re-pair",
			"Dismiss",
			widget.NewLabel(d.Body),
			func(restart bool) {
				if restart {
					// The user accepted; quit the app so they can
					// relaunch and the auth bootstrap re-runs.
					parent.Close()
				}
			},
			parent,
		)
		dlg.Show()
		return
	}
	dialog.NewInformation(d.Title, d.Body, parent).Show()
}
