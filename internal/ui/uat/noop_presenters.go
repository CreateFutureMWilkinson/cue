package uat

import (
	"context"

	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// NoOpPlannerVM is a no-op implementation of ui.PlannerViewModel.
// All methods return zero values. Used in UAT mode where no real planner is active.
type NoOpPlannerVM struct{}

// CurrentStep returns StepIdle (zero value).
func (n *NoOpPlannerVM) CurrentStep() presenter.WizardStep { return 0 }

// HasActivePlan returns false.
func (n *NoOpPlannerVM) HasActivePlan() bool { return false }

// ActiveSchedule returns nil.
func (n *NoOpPlannerVM) ActiveSchedule() *presenter.ActiveScheduleState { return nil }

// CurrentFocusTask returns (nil, nil).
func (n *NoOpPlannerVM) CurrentFocusTask(_ context.Context) (*presenter.TodoRow, error) {
	return nil, nil
}

// NoOpTimerVM is a no-op implementation of ui.TimerViewModel.
// All methods return zero values. Used in UAT mode where no real timer is active.
type NoOpTimerVM struct{}

// IsRunning returns false.
func (n *NoOpTimerVM) IsRunning() bool { return false }

// ActiveSegment returns 0.
func (n *NoOpTimerVM) ActiveSegment() int { return 0 }

// ElapsedFraction returns 0.
func (n *NoOpTimerVM) ElapsedFraction() float64 { return 0 }

// IsFlashVisible returns false.
func (n *NoOpTimerVM) IsFlashVisible() bool { return false }

// CurrentTaskName returns empty string.
func (n *NoOpTimerVM) CurrentTaskName() string { return "" }

// BlockType returns the zero value of planner.BlockType.
func (n *NoOpTimerVM) BlockType() planner.BlockType { return 0 }

// NoOpWizardVM is a no-op implementation of ui.WizardViewModel.
// All methods return zero values. Used in UAT mode where no real wizard is active.
type NoOpWizardVM struct{}

// CurrentStep returns StepIdle (zero value).
func (n *NoOpWizardVM) CurrentStep() presenter.WizardStep { return 0 }

// FocusSchedule returns nil.
func (n *NoOpWizardVM) FocusSchedule() *presenter.SchedulePreview { return nil }

// RecoverySchedule returns nil.
func (n *NoOpWizardVM) RecoverySchedule() *presenter.SchedulePreview { return nil }

// StartPlanning is a no-op and returns nil.
func (n *NoOpWizardVM) StartPlanning(_ context.Context) error { return nil }

// PreviousStep is a no-op.
func (n *NoOpWizardVM) PreviousStep() {}

// SelectSchedule is a no-op and returns nil.
func (n *NoOpWizardVM) SelectSchedule(_ context.Context, _ string) error { return nil }
