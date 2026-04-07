package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// ActivityLogDrawer wraps the activity log list in a toggle-able drawer
// that sits at the bottom of the character area.
type ActivityLogDrawer struct {
	open           bool
	toggleBtn      *widget.Button
	logList        *widget.List
	drawerBox      *fyne.Container
	character      fyne.CanvasObject
	stackContainer *fyne.Container
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
	if d.stackContainer != nil && d.character != nil {
		if d.open {
			overlay := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 77})
			d.stackContainer.Objects = []fyne.CanvasObject{d.character, overlay, d.drawerBox}
		} else {
			d.stackContainer.Objects = []fyne.CanvasObject{d.character, d.drawerBox}
		}
		d.stackContainer.Refresh()
	}
}

// Container returns the standalone drawer as a canvas object.
func (d *ActivityLogDrawer) Container() fyne.CanvasObject {
	return d.drawerBox
}

// ContainerWithCharacter returns a Stack with the character widget behind
// and the drawer overlaid on top. If character is nil, a placeholder is used.
func (d *ActivityLogDrawer) ContainerWithCharacter(character fyne.CanvasObject) fyne.CanvasObject {
	if character == nil {
		character = widget.NewLabel("")
	}
	d.character = character
	if d.open {
		overlay := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 77})
		d.stackContainer = container.NewStack(character, overlay, d.drawerBox)
	} else {
		d.stackContainer = container.NewStack(character, d.drawerBox)
	}
	return d.stackContainer
}
