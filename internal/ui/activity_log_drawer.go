package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// ActivityLogDrawer wraps the activity log list in a toggle-able drawer
// that sits at the bottom of the character area.
type ActivityLogDrawer struct {
	open      bool
	toggleBtn *widget.Button
	logList   *widget.List
	drawerBox *fyne.Container
}

// NewActivityLogDrawer creates an ActivityLogDrawer using the given presenter.
// The drawer is hidden (closed) by default.
func NewActivityLogDrawer(ap *presenter.ActivityPresenter) *ActivityLogDrawer {
	d := &ActivityLogDrawer{}
	d.logList = newActivityLog(ap)

	d.toggleBtn = widget.NewButton("Activity Log", func() {
		d.ToggleOpen()
	})

	d.drawerBox = container.NewStack(d.toggleBtn)

	return d
}

// IsOpen returns whether the drawer is currently open.
func (d *ActivityLogDrawer) IsOpen() bool {
	return d.open
}

// ToggleOpen toggles the drawer between open and closed states.
func (d *ActivityLogDrawer) ToggleOpen() {
	d.open = !d.open
	if d.open {
		d.toggleBtn.SetText("close ▼")
		d.drawerBox.Objects = []fyne.CanvasObject{
			container.NewBorder(d.toggleBtn, nil, nil, nil, d.logList),
		}
	} else {
		d.toggleBtn.SetText("Activity Log")
		d.drawerBox.Objects = []fyne.CanvasObject{d.toggleBtn}
	}
	d.drawerBox.Refresh()
}

// Container returns the standalone drawer as a canvas object.
func (d *ActivityLogDrawer) Container() fyne.CanvasObject {
	return d.drawerBox
}

// ContainerWithCharacter returns a VSplit with the character widget on top
// and the drawer on the bottom. If character is nil, a placeholder is used.
func (d *ActivityLogDrawer) ContainerWithCharacter(character fyne.CanvasObject) fyne.CanvasObject {
	if character == nil {
		character = widget.NewLabel("")
	}
	split := container.NewVSplit(character, d.drawerBox)
	split.Offset = 0.6
	return split
}
