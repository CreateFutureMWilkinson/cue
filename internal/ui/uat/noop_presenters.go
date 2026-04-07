package uat

import (
	"context"

	"github.com/google/uuid"

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

// AvailableTasks returns nil.
func (n *NoOpPlannerVM) AvailableTasks() []presenter.TodoRow { return nil }

// Estimates returns nil.
func (n *NoOpPlannerVM) Estimates() []presenter.TaskEstimateRow { return nil }

// EstimateSummary returns a zero-value EstimateSummary.
func (n *NoOpPlannerVM) EstimateSummary() presenter.EstimateSummary {
	return presenter.EstimateSummary{}
}

// FocusSchedule returns nil.
func (n *NoOpPlannerVM) FocusSchedule() *presenter.SchedulePreview { return nil }

// RecoverySchedule returns nil.
func (n *NoOpPlannerVM) RecoverySchedule() *presenter.SchedulePreview { return nil }

// ActiveSchedule returns nil.
func (n *NoOpPlannerVM) ActiveSchedule() *presenter.ActiveScheduleState { return nil }

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

// AvailableTasks returns nil.
func (n *NoOpWizardVM) AvailableTasks() []presenter.TodoRow { return nil }

// Estimates returns nil.
func (n *NoOpWizardVM) Estimates() []presenter.TaskEstimateRow { return nil }

// EstimateSummary returns a zero-value EstimateSummary.
func (n *NoOpWizardVM) EstimateSummary() presenter.EstimateSummary {
	return presenter.EstimateSummary{}
}

// FocusSchedule returns nil.
func (n *NoOpWizardVM) FocusSchedule() *presenter.SchedulePreview { return nil }

// RecoverySchedule returns nil.
func (n *NoOpWizardVM) RecoverySchedule() *presenter.SchedulePreview { return nil }

// SelectTask is a no-op.
func (n *NoOpWizardVM) SelectTask(_ uuid.UUID, _ bool) {}

// AddTask is a no-op and returns nil.
func (n *NoOpWizardVM) AddTask(_ context.Context, _ string, _ int) error { return nil }

// NextStep is a no-op and returns nil.
func (n *NoOpWizardVM) NextStep(_ context.Context) error { return nil }

// PreviousStep is a no-op.
func (n *NoOpWizardVM) PreviousStep() {}

// OverrideEstimate is a no-op.
func (n *NoOpWizardVM) OverrideEstimate(_ uuid.UUID, _ int) {}

// ReorderTask is a no-op.
func (n *NoOpWizardVM) ReorderTask(_, _ int) {}

// SelectSchedule is a no-op and returns nil.
func (n *NoOpWizardVM) SelectSchedule(_ context.Context, _ string) error { return nil }

// SelectedCount returns 0.
func (n *NoOpWizardVM) SelectedCount() int { return 0 }
