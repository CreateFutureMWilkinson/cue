package ui

import (
	"context"
	"fmt"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// PlannerCallbacks abstracts the planner presenter for binding.
type PlannerCallbacks interface {
	SetOnStepChange(func(presenter.WizardStep))
	StartPlanning(ctx context.Context) error
	HasActivePlan() bool
	LoadExistingPlan(ctx context.Context) error
	CompleteCurrentTask(ctx context.Context) error
	NextStep(ctx context.Context) error
	PreviousStep()
	AbandonPlan(ctx context.Context) error
}

// FocusRailCallbacks abstracts the focus rail for binding.
type FocusRailCallbacks interface {
	SetActivePlan(active bool)
	SetCurrentTask(task string)
	SetOnDone(fn func())
}

// RefreshableView abstracts a view that can be refreshed.
type RefreshableView interface {
	Refresh()
}

// PlannerViewBindable abstracts a planner view that can be refreshed
// and have button callbacks wired.
type PlannerViewBindable interface {
	RefreshableView
	SetOnPlanMyDay(fn func())
	SetOnNext(fn func())
	SetOnBack(fn func())
	SetOnCompleteTask(fn func())
	SetOnAbandonPlan(fn func())
}

// ViewNavigator abstracts the center view router for binding.
type ViewNavigator interface {
	NavigateTo(view CenterView)
}

// UIScheduler dispatches a function to the UI thread.
// In production this is fyne.Do; in tests it can be a synchronous wrapper.
type UIScheduler func(func())

// AppBinder wires presenter callbacks to view updates.
// It coordinates between the planner presenter, focus rail, views, and router
// to keep the UI synchronized with the underlying application state.
type AppBinder struct {
	plannerP    PlannerCallbacks
	focusRail   FocusRailCallbacks
	wizardView  RefreshableView
	plannerView PlannerViewBindable
	viewRouter  ViewNavigator
	doFunc      UIScheduler
}

// SetUIScheduler sets the function used to dispatch view mutations to the UI thread.
// In production, pass fyne.Do. If not set, view mutations are called directly.
func (b *AppBinder) SetUIScheduler(fn UIScheduler) {
	b.doFunc = fn
}

// scheduleUI dispatches fn through the UIScheduler if one is set,
// otherwise calls fn directly on the current goroutine.
func (b *AppBinder) scheduleUI(fn func()) {
	if b.doFunc != nil {
		b.doFunc(fn)
	} else {
		fn()
	}
}

// NewAppBinder creates a new AppBinder, validating all dependencies.
func NewAppBinder(
	plannerP PlannerCallbacks,
	focusRail FocusRailCallbacks,
	wizardView RefreshableView,
	plannerView PlannerViewBindable,
	viewRouter ViewNavigator,
) (*AppBinder, error) {
	if plannerP == nil {
		return nil, fmt.Errorf("plannerP must not be nil")
	}
	if focusRail == nil {
		return nil, fmt.Errorf("focusRail must not be nil")
	}
	if wizardView == nil {
		return nil, fmt.Errorf("wizardView must not be nil")
	}
	if plannerView == nil {
		return nil, fmt.Errorf("plannerView must not be nil")
	}
	if viewRouter == nil {
		return nil, fmt.Errorf("viewRouter must not be nil")
	}
	return &AppBinder{
		plannerP:    plannerP,
		focusRail:   focusRail,
		wizardView:  wizardView,
		plannerView: plannerView,
		viewRouter:  viewRouter,
	}, nil
}

// Bind wires all presenter callbacks to view updates.
func (b *AppBinder) Bind() {
	b.plannerP.SetOnStepChange(func(step presenter.WizardStep) {
		b.scheduleUI(func() {
			b.wizardView.Refresh()
			switch step {
			case presenter.StepActive:
				b.plannerView.Refresh()
				b.viewRouter.NavigateTo(ViewPlan)
				b.focusRail.SetActivePlan(true)
			case presenter.StepIdle:
				b.focusRail.SetActivePlan(false)
			}
		})
	})

	b.plannerView.SetOnPlanMyDay(func() {
		_ = b.plannerP.StartPlanning(context.Background())
		b.viewRouter.NavigateTo(ViewWizard)
	})

	b.focusRail.SetOnDone(func() {
		// Intentionally discard error - UI callback should not panic on task completion failure
		_ = b.plannerP.CompleteCurrentTask(context.Background())
	})

	b.plannerView.SetOnNext(func() {
		_ = b.plannerP.NextStep(context.Background())
	})

	b.plannerView.SetOnBack(func() {
		b.plannerP.PreviousStep()
	})

	b.plannerView.SetOnCompleteTask(func() {
		_ = b.plannerP.CompleteCurrentTask(context.Background())
	})

	b.plannerView.SetOnAbandonPlan(func() {
		_ = b.plannerP.AbandonPlan(context.Background())
	})
}

// AutoLoad loads an existing plan and updates focus rail state.
func (b *AppBinder) AutoLoad(ctx context.Context) error {
	if err := b.plannerP.LoadExistingPlan(ctx); err != nil {
		return fmt.Errorf("loading existing plan: %w", err)
	}
	b.focusRail.SetActivePlan(b.plannerP.HasActivePlan())
	return nil
}
