package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func init() {
	newFyneApp = func() fyne.App { return test.NewApp() }
}
