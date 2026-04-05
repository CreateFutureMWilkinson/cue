package ui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// newFyneApp is the factory function for creating a Fyne application.
// Tests can replace this with test.NewApp to avoid requiring a display.
var newFyneApp = func() fyne.App { return app.New() }

const (
	// refreshInterval is how often the UI refreshes notification content.
	refreshInterval = 30 * time.Second

	// outerSplitOffset positions the focus rail at ~10% width.
	outerSplitOffset = 0.1
	// innerSplitOffset positions the center area at ~67% of the remaining space
	// (60% of total / 90% remaining ≈ 0.667).
	innerSplitOffset = 0.667
)

// MainWindow holds the Fyne application and its primary window.
type MainWindow struct {
	fyneApp    fyne.App
	window     fyne.Window
	appP       *presenter.AppPresenter
	notifP     *presenter.NotificationPresenter
	viewRouter *CenterViewRouter
}

// NewMainWindow creates the main application window with a three-column layout:
// focus rail (10%), character/center area (60%), and notification panel (30%).
// The optional characterWidget, if non-nil, is displayed in the center column.
// The viewRouter controls center-column view switching.
func NewMainWindow(
	cfg config.GUIConfig,
	np *presenter.NotificationPresenter,
	ap *presenter.ActivityPresenter,
	fp *presenter.FeedbackPresenter,
	appP *presenter.AppPresenter,
	sp *presenter.SettingsPresenter,
	characterWidget fyne.CanvasObject,
	viewRouter *CenterViewRouter,
) *MainWindow {
	fyneApp := newFyneApp()
	win := fyneApp.NewWindow("Cue")
	win.Resize(fyne.NewSize(float32(cfg.WindowWidth), float32(cfg.WindowHeight)))

	// Focus rail (left 10%) — placeholder for now.
	focusRail := widget.NewLabel("Focus")

	// Center area (60%) — character widget + activity log.
	var centerPane fyne.CanvasObject
	if ap != nil {
		activityList := newActivityLog(ap)
		if characterWidget != nil {
			centerPane = container.NewBorder(nil, characterWidget, nil, nil, activityList)
		} else {
			centerPane = activityList
		}
	} else if characterWidget != nil {
		centerPane = characterWidget
	} else {
		centerPane = widget.NewLabel("")
	}

	// Notification panel (right 30%).
	var notifPane fyne.CanvasObject
	if np != nil {
		notifPane = newNotificationPane(np, win)
	} else {
		notifPane = widget.NewLabel("")
	}

	// Three-column layout using nested HSplit containers.
	innerSplit := container.NewHSplit(centerPane, notifPane)
	innerSplit.SetOffset(innerSplitOffset)

	outerSplit := container.NewHSplit(focusRail, innerSplit)
	outerSplit.SetOffset(outerSplitOffset)

	win.SetContent(outerSplit)

	// Menu — guard against nil presenters.
	menuItems := make([]*fyne.MenuItem, 0, 3)
	if sp != nil {
		menuItems = append(menuItems, fyne.NewMenuItem("Settings", func() {
			showSettings(sp, fyneApp)
		}))
	}
	menuItems = append(menuItems, fyne.NewMenuItem("About", func() {
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "Cue",
			Content: "Cue - ADHD-friendly productivity assistant",
		})
	}))
	menuItems = append(menuItems, fyne.NewMenuItem("Quit", func() {
		fyneApp.Quit()
	}))
	win.SetMainMenu(fyne.NewMainMenu(
		fyne.NewMenu("Cue", menuItems...),
	))

	return &MainWindow{
		fyneApp:    fyneApp,
		window:     win,
		appP:       appP,
		notifP:     np,
		viewRouter: viewRouter,
	}
}

// Run shows the window and starts the Fyne event loop. Blocks until quit.
func (m *MainWindow) Run() {
	// Periodic notification refresh.
	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			m.window.Canvas().Refresh(m.window.Canvas().Content())
		}
	}()

	m.window.ShowAndRun()
}
