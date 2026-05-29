package ui_test

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// --- Mock WizardViewModel ---

type mockWizardViewModel struct {
	mock.Mock
}

func (m *mockWizardViewModel) CurrentStep() presenter.WizardStep {
	args := m.Called()
	return args.Get(0).(presenter.WizardStep)
}

func (m *mockWizardViewModel) FocusSchedule() *presenter.SchedulePreview {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*presenter.SchedulePreview)
}

func (m *mockWizardViewModel) RecoverySchedule() *presenter.SchedulePreview {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*presenter.SchedulePreview)
}

func (m *mockWizardViewModel) StartPlanning(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockWizardViewModel) PreviousStep() {
	m.Called()
}

func (m *mockWizardViewModel) SelectSchedule(ctx context.Context, strategy string) error {
	args := m.Called(ctx, strategy)
	return args.Error(0)
}

// --- Suite ---

type WizardViewSuite struct {
	suite.Suite
	vm     *mockWizardViewModel
	router *ui.CenterViewRouter
}

func TestWizardView(t *testing.T) {
	suite.Run(t, new(WizardViewSuite))
}

func (s *WizardViewSuite) SetupTest() {
	s.vm = new(mockWizardViewModel)
	s.router = ui.NewCenterViewRouter()
}

// --- Helper sample data ---

func sampleFocusSchedule() *presenter.SchedulePreview {
	return &presenter.SchedulePreview{
		Strategy:       "focus-maximized",
		TotalFocusTime: 3 * time.Hour,
		BreakCount:     4,
		Blocks: []presenter.TimeBlockPreview{
			{Type: "focus", TaskName: "Write tests"},
			{Type: "short_break"},
			{Type: "focus", TaskName: "Deploy fix"},
		},
	}
}

func sampleRecoverySchedule() *presenter.SchedulePreview {
	return &presenter.SchedulePreview{
		Strategy:       "recovery-balanced",
		TotalFocusTime: 2 * time.Hour,
		BreakCount:     6,
		Blocks: []presenter.TimeBlockPreview{
			{Type: "focus", TaskName: "Write tests"},
			{Type: "long_break"},
			{Type: "focus", TaskName: "Deploy fix"},
		},
	}
}

func (s *WizardViewSuite) setupIdleDefaults() {
	s.vm.On("CurrentStep").Return(presenter.StepIdle).Maybe()
	s.vm.On("FocusSchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
	s.vm.On("RecoverySchedule").Return((*presenter.SchedulePreview)(nil)).Maybe()
}

func (s *WizardViewSuite) setupScheduleDefaults() {
	s.vm.On("CurrentStep").Return(presenter.StepSchedule).Maybe()
	s.vm.On("FocusSchedule").Return(sampleFocusSchedule()).Maybe()
	s.vm.On("RecoverySchedule").Return(sampleRecoverySchedule()).Maybe()
}

// --- Constructor ---

func (s *WizardViewSuite) TestNewWizardViewReturnsNonNil() {
	s.setupIdleDefaults()
	view := ui.NewWizardView(s.vm, s.router)
	s.NotNil(view)
}

func (s *WizardViewSuite) TestContainerReturnsNonNil() {
	s.setupIdleDefaults()
	view := ui.NewWizardView(s.vm, s.router)
	s.NotNil(view.Container())
}

// --- StepIdle ---

func (s *WizardViewSuite) TestIdleStateRendersContent() {
	s.setupIdleDefaults()
	view := ui.NewWizardView(s.vm, s.router)
	s.Greater(len(view.Container().Objects), 0,
		"StepIdle should render visible content in the wizard container")
}

func (s *WizardViewSuite) TestIdleStateShowsPromptLabel() {
	s.setupIdleDefaults()
	view := ui.NewWizardView(s.vm, s.router)
	_, found := uitest.FindWidget[*widget.Label](view.Container(), func(l *widget.Label) bool {
		return l.Text != ""
	})
	s.True(found, "StepIdle should show a prompt label guiding the user")
}

// --- StepSchedule ---

func (s *WizardViewSuite) TestScheduleShowsTwoCards() {
	s.setupScheduleDefaults()
	view := ui.NewWizardView(s.vm, s.router)
	s.Equal(2, view.ScheduleCards())
}

func (s *WizardViewSuite) TestScheduleFocusCardStrategy() {
	s.setupScheduleDefaults()
	view := ui.NewWizardView(s.vm, s.router)
	s.Equal("focus-maximized", view.FocusCardStrategy())
}

func (s *WizardViewSuite) TestScheduleRecoveryCardStrategy() {
	s.setupScheduleDefaults()
	view := ui.NewWizardView(s.vm, s.router)
	s.Equal("recovery-balanced", view.RecoveryCardStrategy())
}

func (s *WizardViewSuite) TestScheduleFocusCardStats() {
	s.setupScheduleDefaults()
	view := ui.NewWizardView(s.vm, s.router)
	stats := view.FocusCardStats()
	s.Equal(2, stats.FocusBlocks)
	s.Equal(4, stats.Breaks)
	s.Equal("3h0m", stats.TotalTime)
}

func (s *WizardViewSuite) TestScheduleRecoveryCardStats() {
	s.setupScheduleDefaults()
	view := ui.NewWizardView(s.vm, s.router)
	stats := view.RecoveryCardStats()
	s.Equal(2, stats.FocusBlocks)
	s.Equal(6, stats.Breaks)
	s.Equal("2h0m", stats.TotalTime)
}

func (s *WizardViewSuite) TestScheduleHasSelectButtons() {
	s.setupScheduleDefaults()
	view := ui.NewWizardView(s.vm, s.router)

	_, foundFocus := uitest.FindWidget[*widget.Button](view.Container(), func(b *widget.Button) bool {
		return b.Text == "Select focus-maximized"
	})
	_, foundRecovery := uitest.FindWidget[*widget.Button](view.Container(), func(b *widget.Button) bool {
		return b.Text == "Select recovery-balanced"
	})
	s.True(foundFocus)
	s.True(foundRecovery)
}

func (s *WizardViewSuite) TestScheduleHasBackButton() {
	s.setupScheduleDefaults()
	view := ui.NewWizardView(s.vm, s.router)

	_, found := uitest.FindWidget[*widget.Button](view.Container(), func(b *widget.Button) bool {
		return b.Text == "Back"
	})
	s.True(found)
}

func (s *WizardViewSuite) TestScheduleSelectButtonCallsSelectSchedule() {
	s.setupScheduleDefaults()
	s.vm.On("SelectSchedule", mock.Anything, "focus-maximized").Return(nil).Once()

	view := ui.NewWizardView(s.vm, s.router)

	btn, found := uitest.FindWidget[*widget.Button](view.Container(), func(b *widget.Button) bool {
		return b.Text == "Select focus-maximized"
	})
	s.Require().True(found)
	btn.OnTapped()

	s.vm.AssertCalled(s.T(), "SelectSchedule", mock.Anything, "focus-maximized")
}

func (s *WizardViewSuite) TestScheduleBackButtonCallsPreviousStep() {
	s.setupScheduleDefaults()
	s.vm.On("PreviousStep").Once()

	view := ui.NewWizardView(s.vm, s.router)

	btn, found := uitest.FindWidget[*widget.Button](view.Container(), func(b *widget.Button) bool {
		return b.Text == "Back"
	})
	s.Require().True(found)
	btn.OnTapped()

	s.vm.AssertCalled(s.T(), "PreviousStep")
}
