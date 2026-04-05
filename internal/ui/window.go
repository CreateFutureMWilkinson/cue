package ui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

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
	fyneApp      fyne.App
	window       fyne.Window
	appP         *presenter.AppPresenter
	notifP       *presenter.NotificationPresenter
	viewRouter   *CenterViewRouter
	notifPanel   *NotificationPanel
	centerStack  *fyne.Container
	viewContents map[CenterView]fyne.CanvasObject
}

// NewMainWindow creates the main application window with a three-column layout:
// focus rail (10%), character/center area (60%), and notification panel (30%).
// The optional characterWidget, if non-nil, is displayed in the center column.
// The viewRouter controls center-column view switching.
func NewMainWindow(
	fyneApp fyne.App,
	cfg config.GUIConfig,
	np *presenter.NotificationPresenter,
	ap *presenter.ActivityPresenter,
	fp *presenter.FeedbackPresenter,
	appP *presenter.AppPresenter,
	sp *presenter.SettingsPresenter,
	ssp *presenter.ServiceSettingsPresenter,
	ollamaCfg config.OllamaConfig,
	characterWidget fyne.CanvasObject,
	viewRouter *CenterViewRouter,
	plannerVM PlannerViewModel,
	timerVM TimerViewModel,
	wizardVM WizardViewModel,
) *MainWindow {
	win := fyneApp.NewWindow("Cue")
	win.Resize(fyne.NewSize(float32(cfg.WindowWidth), float32(cfg.WindowHeight)))

	// Focus rail (left 10%) — timer ring, task info, and navigation buttons.
	var focusRail fyne.CanvasObject
	if viewRouter != nil {
		fr := NewFocusRail(viewRouter)
		focusRail = fr.Container()
	} else {
		focusRail = widget.NewLabel("Focus")
	}

	// Center area (60%) — dynamically controlled by viewRouter.
	// Build content for each view; the router swaps between them.
	var characterContent fyne.CanvasObject
	if ap != nil {
		drawer := NewActivityLogDrawer(ap)
		characterContent = drawer.ContainerWithCharacter(characterWidget)
	} else if characterWidget != nil {
		characterContent = characterWidget
	} else {
		characterContent = widget.NewLabel("")
	}

	// Build settings view content.
	var settingsContent fyne.CanvasObject
	if sp != nil && ssp != nil {
		sv := NewSettingsView(sp, ssp, ollamaCfg)
		settingsContent = sv.Container()
	} else {
		settingsContent = widget.NewLabel("Settings")
	}

	// Build planner view content.
	var planContent fyne.CanvasObject
	if plannerVM != nil && timerVM != nil {
		pv := NewPlannerView(plannerVM, timerVM, viewRouter)
		planContent = pv.Container()
	} else {
		planContent = widget.NewLabel("Plan")
	}

	// Build wizard view content.
	var wizardContent fyne.CanvasObject
	if wizardVM != nil {
		wv := NewWizardView(wizardVM, viewRouter)
		wizardContent = wv.Container()
	} else {
		wizardContent = widget.NewLabel("Wizard")
	}

	// Map views to their content for lookup during navigation.
	viewContents := map[CenterView]fyne.CanvasObject{
		ViewCharacter: characterContent,
		ViewPlan:      planContent,
		ViewWizard:    wizardContent,
		ViewSettings:  settingsContent,
	}

	// Start with the character view.
	centerStack := container.NewStack(characterContent)

	// Notification panel (right 30%) — expandable from collapsed state.
	var notifPane fyne.CanvasObject
	var notifPanel *NotificationPanel
	if np != nil {
		notifPanel = NewNotificationPanel(np, win)
		notifPane = notifPanel.Container()
	} else {
		notifPane = widget.NewLabel("")
	}

	// Three-column layout using nested HSplit containers.
	innerSplit := container.NewHSplit(centerStack, notifPane)
	innerSplit.SetOffset(innerSplitOffset)

	outerSplit := container.NewHSplit(focusRail, innerSplit)
	outerSplit.SetOffset(outerSplitOffset)

	win.SetContent(outerSplit)

	// Menu bar — dynamically built based on available presenters.
	menuItems := make([]*fyne.MenuItem, 0, 3)
	if sp != nil && viewRouter != nil {
		menuItems = append(menuItems, fyne.NewMenuItem("Settings", func() {
			viewRouter.NavigateTo(ViewSettings)
		}))
	}
	menuItems = append(menuItems, fyne.NewMenuItem("About", func() {
		fyneApp.SendNotification(&fyne.Notification{
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

	mw := &MainWindow{
		fyneApp:      fyneApp,
		window:       win,
		appP:         appP,
		notifP:       np,
		viewRouter:   viewRouter,
		notifPanel:   notifPanel,
		centerStack:  centerStack,
		viewContents: viewContents,
	}

	// Register a view-change listener to swap center content on navigation.
	// Uses AddOnViewChange to avoid overwriting FocusRail's callback.
	if viewRouter != nil {
		viewRouter.AddOnViewChange(mw.switchCenterView)
	}

	return mw
}

// Content returns the top-level canvas object set as the window's content.
// This exposes the widget tree for structural testing.
func (m *MainWindow) Content() fyne.CanvasObject {
	return m.window.Content()
}

// CenterContent returns the canvas object currently displayed in the center column.
func (m *MainWindow) CenterContent() fyne.CanvasObject {
	if m.viewRouter == nil {
		return nil
	}
	return m.viewContents[m.viewRouter.CurrentView()]
}

// switchCenterView handles view switching by updating the center stack container.
func (m *MainWindow) switchCenterView(view CenterView) {
	if content, exists := m.viewContents[view]; exists {
		m.centerStack.Objects = []fyne.CanvasObject{content}
		m.centerStack.Refresh()
	}
}

// FocusRail returns the focus rail component for testing wiring.
func (m *MainWindow) FocusRail() *FocusRail { return nil }

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
