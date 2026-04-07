package uat_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uat"
)

// NoOpPresentersSuite verifies that the no-op presenter types satisfy their
// respective ui interfaces and return expected zero values.
type NoOpPresentersSuite struct {
	suite.Suite
}

func TestNoOpPresenters(t *testing.T) {
	suite.Run(t, new(NoOpPresentersSuite))
}

// Compile-time interface satisfaction checks.
var (
	_ ui.PlannerViewModel = (*uat.NoOpPlannerVM)(nil)
	_ ui.TimerViewModel   = (*uat.NoOpTimerVM)(nil)
	_ ui.WizardViewModel  = (*uat.NoOpWizardVM)(nil)
)

func (s *NoOpPresentersSuite) TestNoOpPlannerVMSatisfiesInterface() {
	vm := &uat.NoOpPlannerVM{}

	s.False(vm.HasActivePlan())
	s.Nil(vm.AvailableTasks())
	s.Nil(vm.FocusSchedule())
	s.Nil(vm.RecoverySchedule())
	s.Nil(vm.ActiveSchedule())
	s.Equal(0, int(vm.CurrentStep()))
}

func (s *NoOpPresentersSuite) TestNoOpTimerVMSatisfiesInterface() {
	vm := &uat.NoOpTimerVM{}

	s.False(vm.IsRunning())
	s.Equal(0, vm.ActiveSegment())
	s.Equal(float64(0), vm.ElapsedFraction())
	s.False(vm.IsFlashVisible())
	s.Empty(vm.CurrentTaskName())
}

func (s *NoOpPresentersSuite) TestNoOpWizardVMSatisfiesInterface() {
	vm := &uat.NoOpWizardVM{}

	s.Equal(0, vm.SelectedCount())
	s.Nil(vm.AvailableTasks())
	s.Nil(vm.Estimates())
	s.Nil(vm.FocusSchedule())
	s.Nil(vm.RecoverySchedule())
	s.Equal(0, int(vm.CurrentStep()))
}
