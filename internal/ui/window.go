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

const refreshInterval = 30 * time.Second

// MainWindow holds the Fyne application and its primary window.
type MainWindow struct {
	fyneApp fyne.App
	window  fyne.Window
	appP    *presenter.AppPresenter
	notifP  *presenter.NotificationPresenter
}

// NewMainWindow creates the main application window with three-pane layout.
// The optional characterWidget, if non-nil, is displayed below the activity log.
func NewMainWindow(
	cfg config.GUIConfig,
	np *presenter.NotificationPresenter,
	ap *presenter.ActivityPresenter,
	fp *presenter.FeedbackPresenter,
	appP *presenter.AppPresenter,
	sp *presenter.SettingsPresenter,
	characterWidget fyne.CanvasObject,
) *MainWindow {
	fyneApp := app.New()
	win := fyneApp.NewWindow("Cue")
	win.Resize(fyne.NewSize(float32(cfg.WindowWidth), float32(cfg.WindowHeight)))

	notifList := newNotificationPane(np, win)
	activityList := newActivityLog(ap)

	var rightPane fyne.CanvasObject
	if characterWidget != nil {
		rightPane = container.NewBorder(nil, characterWidget, nil, nil, activityList)
	} else {
		rightPane = activityList
	}

	topSplit := container.NewHSplit(notifList, rightPane)
	topSplit.SetOffset(0.5)

	reviewBtn := widget.NewButton("Review Buffered", func() {
		showFeedbackReview(fp, fyneApp)
	})

	mainLayout := container.NewBorder(nil, reviewBtn, nil, nil, topSplit)
	win.SetContent(mainLayout)

	// Menu
	settingsItem := fyne.NewMenuItem("Settings", func() {
		showSettings(sp, fyneApp)
	})
	aboutItem := fyne.NewMenuItem("About", func() {
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "Cue",
			Content: "Cue - ADHD-friendly productivity assistant",
		})
	})
	quitItem := fyne.NewMenuItem("Quit", func() {
		fyneApp.Quit()
	})
	win.SetMainMenu(fyne.NewMainMenu(
		fyne.NewMenu("Cue", settingsItem, aboutItem, quitItem),
	))

	return &MainWindow{
		fyneApp: fyneApp,
		window:  win,
		appP:    appP,
		notifP:  np,
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
