package ui

import (
	"log/slog"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// openWebURL returns a callback that opens the given URL in the user's default
// browser via the Fyne application. Empty URLs are ignored so a notification
// card without a deep link is a no-op rather than an error.
func openWebURL(app fyne.App) func(string) {
	return func(raw string) {
		if raw == "" {
			return
		}
		u, err := url.Parse(raw)
		if err != nil {
			slog.Warn("notification web_url parse failed", "url", raw, "error", err)
			return
		}
		if err := app.OpenURL(u); err != nil {
			slog.Warn("notification OpenURL failed", "url", raw, "error", err)
		}
	}
}

const (
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
	focusRail    *FocusRail
	centerStack  *fyne.Container
	viewContents map[CenterView]fyne.CanvasObject
	plannerView  *PlannerView
	wizardView   *WizardView
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
	rp *presenter.RulesPresenter,
	ollamaCfg config.OllamaConfig,
	characterWidget fyne.CanvasObject,
	viewRouter *CenterViewRouter,
	plannerVM PlannerViewModel,
	timerVM TimerViewModel,
	wizardVM WizardViewModel,
	rightPanelOverride fyne.CanvasObject, // replaces notification panel when non-nil
) *MainWindow {
	win := fyneApp.NewWindow("Cue")
	win.Resize(fyne.NewSize(float32(cfg.WindowWidth), float32(cfg.WindowHeight)))

	// Focus rail (left 10%) — timer ring, task info, and navigation buttons.
	var focusRailWidget fyne.CanvasObject
	var fr *FocusRail
	if viewRouter != nil {
		fr = NewFocusRail(viewRouter)
		if fp != nil {
			fr.SetOnReview(func() {
				showFeedbackReview(fp, fyneApp)
			})
		}
		if np != nil {
			fr.SetNotificationsExpanded(np.IsExpanded())
			np.SetOnExpandedChange(fr.SetNotificationsExpanded)
		}
		focusRailWidget = fr.Container()
	} else {
		focusRailWidget = widget.NewLabel("Focus")
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
		sv := NewSettingsView(sp, ssp, rp, ollamaCfg, func() {
			if viewRouter != nil {
				viewRouter.NavigateTo(ViewCharacter)
			}
		})
		settingsContent = sv.Container()
	} else {
		settingsContent = widget.NewLabel("Settings")
	}

	// Build planner view content.
	var planContent fyne.CanvasObject
	var pv *PlannerView
	if plannerVM != nil && timerVM != nil {
		pv = NewPlannerView(plannerVM, timerVM, viewRouter, nil)
		planContent = pv.Container()
	} else {
		planContent = widget.NewLabel("Plan")
	}

	// Build wizard view content.
	var wizardContent fyne.CanvasObject
	var wv *WizardView
	if wizardVM != nil {
		wv = NewWizardView(wizardVM, viewRouter)
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
	// When rightPanelOverride is provided, it replaces the notification panel entirely.
	var notifPane fyne.CanvasObject
	var notifPanel *NotificationPanel
	if rightPanelOverride != nil {
		notifPane = rightPanelOverride
	} else if np != nil {
		notifPanel = NewNotificationPanel(np, win)
		notifPanel.SetOnNotificationClick(openWebURL(fyneApp))
		notifPane = notifPanel.Container()
	} else {
		notifPane = widget.NewLabel("")
	}

	// Three-column layout using nested HSplit containers.
	innerSplit := container.NewHSplit(centerStack, notifPane)
	innerSplit.SetOffset(innerSplitOffset)

	outerSplit := container.NewHSplit(focusRailWidget, innerSplit)
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
		focusRail:    fr,
		centerStack:  centerStack,
		viewContents: viewContents,
		plannerView:  pv,
		wizardView:   wv,
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
// Wrapped in fyne.Do to ensure thread safety when called from non-UI goroutines.
func (m *MainWindow) switchCenterView(view CenterView) {
	if content, exists := m.viewContents[view]; exists {
		fyne.Do(func() {
			m.centerStack.Objects = []fyne.CanvasObject{content}
			m.centerStack.Refresh()
		})
	}
}

// SetCharacterWidget replaces the character content in the center column and
// refreshes the view if it is currently active.
func (m *MainWindow) SetCharacterWidget(w fyne.CanvasObject) {
	m.viewContents[ViewCharacter] = w
	if m.viewRouter != nil && m.viewRouter.CurrentView() == ViewCharacter {
		fyne.Do(func() {
			m.centerStack.Objects = []fyne.CanvasObject{w}
			m.centerStack.Refresh()
		})
	}
}

// FocusRail returns the focus rail component.
func (m *MainWindow) FocusRail() *FocusRail { return m.focusRail }

// PlannerViewRef returns the PlannerView as a PlannerViewBindable, or nil.
func (m *MainWindow) PlannerViewRef() PlannerViewBindable {
	if m.plannerView == nil {
		return nil
	}
	return m.plannerView
}

// WizardViewRef returns the WizardView as a RefreshableView, or nil.
func (m *MainWindow) WizardViewRef() RefreshableView {
	if m.wizardView == nil {
		return nil
	}
	return m.wizardView
}

// Run shows the window and starts the Fyne event loop. Blocks until quit.
func (m *MainWindow) Run() {
	m.window.ShowAndRun()
}

// Show displays the window without starting the Fyne event loop. The
// caller is responsible for driving the loop via app.Run() — used when
// the cue ui boot flow needs the loop running BEFORE the main window
// (e.g. to show a Retry/Quit dialog on a transient boot window).
func (m *MainWindow) Show() {
	m.window.Show()
}

// SetCloseIntercept registers a callback that runs when the user closes
// the main window. Fyne treats the boot window as the master, so the
// app does not quit by default when the main window is closed; the
// caller installs an intercept that calls fyneApp.Quit to drive
// shutdown.
func (m *MainWindow) SetCloseIntercept(fn func()) {
	m.window.SetCloseIntercept(fn)
}
