package ui_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Mocks ---

type mockPlannerCallbacks struct {
	mock.Mock
	stepChangeCallback func(presenter.WizardStep)
}

func (m *mockPlannerCallbacks) SetOnStepChange(fn func(presenter.WizardStep)) {
	m.Called(fn)
	m.stepChangeCallback = fn
}

func (m *mockPlannerCallbacks) HasActivePlan() bool {
	return m.Called().Bool(0)
}

func (m *mockPlannerCallbacks) LoadExistingPlan(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *mockPlannerCallbacks) CompleteCurrentTask(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *mockPlannerCallbacks) AbandonPlan(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *mockPlannerCallbacks) ActiveSchedule() *presenter.ActiveScheduleState {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*presenter.ActiveScheduleState)
}

type mockFocusRailCallbacks struct {
	mock.Mock
	doneCallback func()
}

func (m *mockFocusRailCallbacks) SetActivePlan(active bool) {
	m.Called(active)
}

func (m *mockFocusRailCallbacks) SetCurrentTask(task string) {
	m.Called(task)
}

func (m *mockFocusRailCallbacks) SetOnDone(fn func()) {
	m.Called(fn)
	m.doneCallback = fn
}

type mockRefreshableView struct {
	mock.Mock
}

func (m *mockRefreshableView) Refresh() {
	m.Called()
}

type mockViewNavigator struct {
	mock.Mock
}

func (m *mockViewNavigator) NavigateTo(view ui.CenterView) {
	m.Called(view)
}

// --- Suite ---

type AppBinderSuite struct {
	suite.Suite
	plannerP    *mockPlannerCallbacks
	focusRail   *mockFocusRailCallbacks
	wizardView  *mockRefreshableView
	plannerView *mockRefreshableView
	viewRouter  *mockViewNavigator
}

func TestAppBinder(t *testing.T) {
	suite.Run(t, new(AppBinderSuite))
}

func (s *AppBinderSuite) SetupTest() {
	s.plannerP = new(mockPlannerCallbacks)
	s.focusRail = new(mockFocusRailCallbacks)
	s.wizardView = new(mockRefreshableView)
	s.plannerView = new(mockRefreshableView)
	s.viewRouter = new(mockViewNavigator)
}

// --- Constructor Validation ---

func (s *AppBinderSuite) TestAppBinderNilPlannerReturnsError() {
	_, err := ui.NewAppBinder(nil, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Error(err)
}

func (s *AppBinderSuite) TestAppBinderNilFocusRailReturnsError() {
	_, err := ui.NewAppBinder(s.plannerP, nil, s.wizardView, s.plannerView, s.viewRouter)
	s.Error(err)
}

func (s *AppBinderSuite) TestAppBinderNilWizardViewReturnsError() {
	_, err := ui.NewAppBinder(s.plannerP, s.focusRail, nil, s.plannerView, s.viewRouter)
	s.Error(err)
}

func (s *AppBinderSuite) TestAppBinderNilPlannerViewReturnsError() {
	_, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, nil, s.viewRouter)
	s.Error(err)
}

func (s *AppBinderSuite) TestAppBinderNilViewRouterReturnsError() {
	_, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, nil)
	s.Error(err)
}

// --- Bind wiring ---

func (s *AppBinderSuite) TestBindWiresDoneToCompleteCurrentTask() {
	s.plannerP.On("SetOnStepChange", mock.AnythingOfType("func(presenter.WizardStep)")).Return()
	s.focusRail.On("SetOnDone", mock.AnythingOfType("func()")).Return()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	binder.Bind()

	// Invoke the done callback that was captured
	s.Require().NotNil(s.focusRail.doneCallback, "Bind should have wired a Done callback")

	s.plannerP.On("CompleteCurrentTask", mock.Anything).Return(nil)
	s.focusRail.doneCallback()

	s.plannerP.AssertCalled(s.T(), "CompleteCurrentTask", mock.Anything)
}

func (s *AppBinderSuite) TestBindDoneCallbackDoesNotPanicOnError() {
	s.plannerP.On("SetOnStepChange", mock.AnythingOfType("func(presenter.WizardStep)")).Return()
	s.focusRail.On("SetOnDone", mock.AnythingOfType("func()")).Return()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	binder.Bind()

	s.Require().NotNil(s.focusRail.doneCallback, "Bind should have wired a Done callback")

	s.plannerP.On("CompleteCurrentTask", mock.Anything).Return(errors.New("task completion failed"))

	s.NotPanics(func() {
		s.focusRail.doneCallback()
	})

	s.plannerP.AssertCalled(s.T(), "CompleteCurrentTask", mock.Anything)
}

func (s *AppBinderSuite) TestBindWiresStepChangeToWizardRefresh() {
	s.plannerP.On("SetOnStepChange", mock.AnythingOfType("func(presenter.WizardStep)")).Return()
	s.focusRail.On("SetOnDone", mock.AnythingOfType("func()")).Return()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	binder.Bind()

	// Invoke the step change callback with a non-Active step
	s.Require().NotNil(s.plannerP.stepChangeCallback, "Bind should have wired a StepChange callback")

	s.wizardView.On("Refresh").Return()
	s.plannerP.stepChangeCallback(presenter.StepEstimates)

	s.wizardView.AssertCalled(s.T(), "Refresh")
}

func (s *AppBinderSuite) TestBindWiresStepActiveToNavigatePlan() {
	s.plannerP.On("SetOnStepChange", mock.AnythingOfType("func(presenter.WizardStep)")).Return()
	s.focusRail.On("SetOnDone", mock.AnythingOfType("func()")).Return()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	binder.Bind()

	s.Require().NotNil(s.plannerP.stepChangeCallback)

	s.wizardView.On("Refresh").Return()
	s.plannerView.On("Refresh").Return()
	s.viewRouter.On("NavigateTo", ui.ViewPlan).Return()
	s.focusRail.On("SetActivePlan", true).Return()

	s.plannerP.stepChangeCallback(presenter.StepActive)

	s.viewRouter.AssertCalled(s.T(), "NavigateTo", ui.ViewPlan)
	s.focusRail.AssertCalled(s.T(), "SetActivePlan", true)
}

func (s *AppBinderSuite) TestBindWiresStepIdleToDeactivatePlan() {
	s.plannerP.On("SetOnStepChange", mock.AnythingOfType("func(presenter.WizardStep)")).Return()
	s.focusRail.On("SetOnDone", mock.AnythingOfType("func()")).Return()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	binder.Bind()

	s.Require().NotNil(s.plannerP.stepChangeCallback)

	s.wizardView.On("Refresh").Return()
	s.focusRail.On("SetActivePlan", false).Return()

	s.plannerP.stepChangeCallback(presenter.StepIdle)

	s.focusRail.AssertCalled(s.T(), "SetActivePlan", false)
}

// --- AutoLoad ---

func (s *AppBinderSuite) TestAutoLoadWithExistingPlan() {
	s.plannerP.On("SetOnStepChange", mock.AnythingOfType("func(presenter.WizardStep)")).Return()
	s.focusRail.On("SetOnDone", mock.AnythingOfType("func()")).Return()
	s.plannerP.On("LoadExistingPlan", mock.Anything).Return(nil)
	s.plannerP.On("HasActivePlan").Return(true)
	s.focusRail.On("SetActivePlan", true).Return()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	err = binder.AutoLoad(context.Background())
	s.Require().NoError(err)

	s.focusRail.AssertCalled(s.T(), "SetActivePlan", true)
}

func (s *AppBinderSuite) TestAutoLoadWithNoPlan() {
	s.plannerP.On("SetOnStepChange", mock.AnythingOfType("func(presenter.WizardStep)")).Return()
	s.focusRail.On("SetOnDone", mock.AnythingOfType("func()")).Return()
	s.plannerP.On("LoadExistingPlan", mock.Anything).Return(nil)
	s.plannerP.On("HasActivePlan").Return(false)
	s.focusRail.On("SetActivePlan", false).Return()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	err = binder.AutoLoad(context.Background())
	s.Require().NoError(err)

	s.plannerP.AssertCalled(s.T(), "LoadExistingPlan", mock.Anything)
	s.plannerP.AssertCalled(s.T(), "HasActivePlan")
}

func (s *AppBinderSuite) TestAutoLoadError() {
	s.plannerP.On("SetOnStepChange", mock.AnythingOfType("func(presenter.WizardStep)")).Return()
	s.focusRail.On("SetOnDone", mock.AnythingOfType("func()")).Return()
	s.plannerP.On("LoadExistingPlan", mock.Anything).Return(errors.New("db error"))

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	err = binder.AutoLoad(context.Background())
	s.Error(err)
}
