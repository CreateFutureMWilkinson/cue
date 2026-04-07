package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// FocusRail provides the persistent left column with a countdown timer,
// task information, and navigation buttons (Plan, Back, Done, Review, Settings).
type FocusRail struct {
	router      *CenterViewRouter
	planBtn     *widget.Button
	backBtn     *widget.Button
	doneBtn     *widget.Button
	reviewBtn   *widget.Button
	settingsBtn *widget.Button
	taskLabel   *widget.Label
	timer       *CountdownTimer
	onBack      func()
	onDone      func()
	onReview    func()
}

// NewFocusRail creates a focus rail bound to the given center view router.
func NewFocusRail(router *CenterViewRouter) *FocusRail {
	rail := &FocusRail{
		router:    router,
		taskLabel: widget.NewLabel(""),
		timer:     NewCountdownTimer(),
	}

	rail.planBtn = widget.NewButton("Plan", func() {
		router.NavigateTo(ViewPlan)
	})

	rail.backBtn = widget.NewButton("Back", func() {
		if rail.onBack != nil {
			rail.onBack()
		}
		router.NavigateTo(ViewCharacter)
	})

	rail.doneBtn = widget.NewButton("Done", func() {
		if rail.onDone != nil {
			rail.onDone()
		}
	})

	rail.reviewBtn = widget.NewButton("Review", func() {
		if rail.onReview != nil {
			rail.onReview()
		}
	})

	rail.settingsBtn = widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		router.NavigateTo(ViewSettings)
	})

	// Set initial visibility based on current router state.
	rail.applyViewState(router.CurrentView())

	// Hide plan-dependent widgets by default.
	rail.timer.Hide()
	rail.taskLabel.Hide()
	rail.doneBtn.Hide()

	// Review hidden by default.
	rail.reviewBtn.Hide()

	// Listen for router view changes.
	router.SetOnViewChange(func(view CenterView) {
		rail.applyViewState(view)
	})

	return rail
}

// applyViewState updates button visibility based on the active view.
func (r *FocusRail) applyViewState(view CenterView) {
	switch view {
	case ViewCharacter:
		r.planBtn.Show()
		r.backBtn.Hide()
		r.settingsBtn.Show()
	case ViewPlan, ViewWizard:
		r.planBtn.Hide()
		r.backBtn.Show()
		r.settingsBtn.Hide()
	case ViewSettings:
		r.planBtn.Show()
		r.backBtn.Show()
		r.settingsBtn.Hide()
	}
}

// PlanButton returns the Plan navigation button.
func (r *FocusRail) PlanButton() *widget.Button { return r.planBtn }

// BackButton returns the Back navigation button.
func (r *FocusRail) BackButton() *widget.Button { return r.backBtn }

// DoneButton returns the Done button for completing the current task.
func (r *FocusRail) DoneButton() *widget.Button { return r.doneBtn }

// ReviewButton returns the Review button for feedback review.
func (r *FocusRail) ReviewButton() *widget.Button { return r.reviewBtn }

// SettingsButton returns the Settings navigation button.
func (r *FocusRail) SettingsButton() *widget.Button { return r.settingsBtn }

// TaskLabel returns the label displaying the current task name.
func (r *FocusRail) TaskLabel() *widget.Label { return r.taskLabel }

// Timer returns the countdown timer widget.
func (r *FocusRail) Timer() *CountdownTimer { return r.timer }

// SetActivePlan shows or hides the timer, task label, and Done button.
func (r *FocusRail) SetActivePlan(active bool) {
	if active {
		r.timer.Show()
		r.taskLabel.Show()
		r.doneBtn.Show()
	} else {
		r.timer.Hide()
		r.taskLabel.Hide()
		r.doneBtn.Hide()
	}
}

// SetCurrentTask updates the task label text.
func (r *FocusRail) SetCurrentTask(task string) {
	r.taskLabel.SetText(task)
}

// SetNotificationsExpanded shows or hides the Review button.
func (r *FocusRail) SetNotificationsExpanded(expanded bool) {
	if expanded {
		r.reviewBtn.Show()
	} else {
		r.reviewBtn.Hide()
	}
}

// SetOnBack registers a callback invoked when the Back button is tapped,
// before navigation occurs.
func (r *FocusRail) SetOnBack(fn func()) {
	r.onBack = fn
}

// SetOnDone registers a callback invoked when the Done button is tapped.
func (r *FocusRail) SetOnDone(fn func()) {
	r.onDone = fn
}

// SetOnReview registers a callback invoked when the Review button is tapped.
func (r *FocusRail) SetOnReview(fn func()) {
	r.onReview = fn
}

// Container returns a *fyne.Container with all FocusRail widgets in a VBox layout.
func (r *FocusRail) Container() *fyne.Container {
	return container.NewVBox(
		r.timer,
		r.taskLabel,
		r.planBtn,
		r.backBtn,
		r.doneBtn,
		r.reviewBtn,
		r.settingsBtn,
	)
}
