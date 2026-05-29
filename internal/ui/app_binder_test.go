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

func (m *mockPlannerCallbacks) StartPlanning(ctx context.Context) error {
	return m.Called(ctx).Error(0)
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

func (m *mockPlannerCallbacks) NextStep(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *mockPlannerCallbacks) PreviousStep() {
	m.Called()
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
	backCallback func()
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

func (m *mockFocusRailCallbacks) SetOnBack(fn func()) {
	m.Called(fn)
	m.backCallback = fn
}

type mockRefreshableView struct {
	mock.Mock
}

func (m *mockRefreshableView) Refresh() {
	m.Called()
}

type mockPlannerViewBindable struct {
	mock.Mock
	planMyDayCallback    func()
	nextCallback         func()
	backCallback         func()
	completeTaskCallback func()
	abandonPlanCallback  func()
}

func (m *mockPlannerViewBindable) Refresh() {
	m.Called()
}

func (m *mockPlannerViewBindable) SetOnPlanMyDay(fn func()) {
	m.Called(fn)
	m.planMyDayCallback = fn
}

func (m *mockPlannerViewBindable) SetOnNext(fn func()) {
	m.Called(fn)
	m.nextCallback = fn
}

func (m *mockPlannerViewBindable) SetOnBack(fn func()) {
	m.Called(fn)
	m.backCallback = fn
}

func (m *mockPlannerViewBindable) SetOnCompleteTask(fn func()) {
	m.Called(fn)
	m.completeTaskCallback = fn
}

func (m *mockPlannerViewBindable) SetOnAbandonPlan(fn func()) {
	m.Called(fn)
	m.abandonPlanCallback = fn
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
	plannerView *mockPlannerViewBindable
	viewRouter  *mockViewNavigator
}

func TestAppBinder(t *testing.T) {
	suite.Run(t, new(AppBinderSuite))
}

func (s *AppBinderSuite) SetupTest() {
	s.plannerP = new(mockPlannerCallbacks)
	s.focusRail = new(mockFocusRailCallbacks)
	s.wizardView = new(mockRefreshableView)
	s.plannerView = new(mockPlannerViewBindable)
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

// expectBindCalls sets up mock expectations for all calls made by Bind().
func (s *AppBinderSuite) expectBindCalls() {
	s.plannerP.On("SetOnStepChange", mock.AnythingOfType("func(presenter.WizardStep)")).Return()
	s.focusRail.On("SetOnDone", mock.AnythingOfType("func()")).Return()
	s.focusRail.On("SetOnBack", mock.AnythingOfType("func()")).Return()
	s.plannerView.On("SetOnPlanMyDay", mock.AnythingOfType("func()")).Return()
	s.plannerView.On("SetOnNext", mock.AnythingOfType("func()")).Return()
	s.plannerView.On("SetOnBack", mock.AnythingOfType("func()")).Return()
	s.plannerView.On("SetOnCompleteTask", mock.AnythingOfType("func()")).Return()
	s.plannerView.On("SetOnAbandonPlan", mock.AnythingOfType("func()")).Return()
}

// --- Bind wiring ---

func (s *AppBinderSuite) TestBindWiresDoneToCompleteCurrentTask() {
	s.expectBindCalls()

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
	s.expectBindCalls()

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
	s.expectBindCalls()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	binder.Bind()

	// Invoke the step change callback with a non-Active step
	s.Require().NotNil(s.plannerP.stepChangeCallback, "Bind should have wired a StepChange callback")

	s.wizardView.On("Refresh").Return()
	s.plannerP.stepChangeCallback(presenter.StepSchedule)

	s.wizardView.AssertCalled(s.T(), "Refresh")
}

func (s *AppBinderSuite) TestBindWiresStepActiveToNavigatePlan() {
	s.expectBindCalls()

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
	s.expectBindCalls()

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

// --- Bug 073: Bind wires planner view buttons ---

func (s *AppBinderSuite) TestBindWiresNextButtonToPresenterNextStep() {
	s.expectBindCalls()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	binder.Bind()

	s.Require().NotNil(s.plannerView.nextCallback, "Bind should wire a Next callback on plannerView")

	s.plannerP.On("NextStep", mock.Anything).Return(nil)
	s.plannerView.nextCallback()

	s.plannerP.AssertCalled(s.T(), "NextStep", mock.Anything)
}

func (s *AppBinderSuite) TestBindWiresBackButtonToPresenterPreviousStep() {
	s.expectBindCalls()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	binder.Bind()

	s.Require().NotNil(s.plannerView.backCallback, "Bind should wire a Back callback on plannerView")

	s.plannerP.On("PreviousStep").Return()
	s.plannerView.backCallback()

	s.plannerP.AssertCalled(s.T(), "PreviousStep")
}

func (s *AppBinderSuite) TestBindWiresCompleteTaskButtonToPresenterCompleteCurrentTask() {
	s.expectBindCalls()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	binder.Bind()

	s.Require().NotNil(s.plannerView.completeTaskCallback, "Bind should wire a CompleteTask callback on plannerView")

	s.plannerP.On("CompleteCurrentTask", mock.Anything).Return(nil)
	s.plannerView.completeTaskCallback()

	s.plannerP.AssertCalled(s.T(), "CompleteCurrentTask", mock.Anything)
}

func (s *AppBinderSuite) TestBindWiresPlanMyDayToStartPlanningAndNavigate() {
	s.expectBindCalls()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	binder.Bind()

	s.Require().NotNil(s.plannerView.planMyDayCallback, "Bind should wire a PlanMyDay callback on plannerView")

	s.plannerP.On("StartPlanning", mock.Anything).Return(nil)
	s.viewRouter.On("NavigateTo", ui.ViewWizard).Return()

	s.plannerView.planMyDayCallback()

	s.plannerP.AssertCalled(s.T(), "StartPlanning", mock.Anything)
	s.viewRouter.AssertCalled(s.T(), "NavigateTo", ui.ViewWizard)
}

func (s *AppBinderSuite) TestBindStepChangeCallbackUsesUIScheduler() {
	s.expectBindCalls()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	schedulerCalled := false
	binder.SetUIScheduler(func(fn func()) {
		schedulerCalled = true
		fn()
	})

	binder.Bind()

	s.Require().NotNil(s.plannerP.stepChangeCallback, "Bind should have wired a StepChange callback")

	// Set up expectations for StepActive path
	s.wizardView.On("Refresh").Return()
	s.plannerView.On("Refresh").Return()
	s.viewRouter.On("NavigateTo", ui.ViewPlan).Return()
	s.focusRail.On("SetActivePlan", true).Return()

	s.plannerP.stepChangeCallback(presenter.StepActive)

	s.True(schedulerCalled, "step-change callback should dispatch view mutations through the UIScheduler")
}

func (s *AppBinderSuite) TestBindWiresAbandonButtonToPresenterAbandonPlan() {
	s.expectBindCalls()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	binder.Bind()

	s.Require().NotNil(s.plannerView.abandonPlanCallback, "Bind should wire an AbandonPlan callback on plannerView")

	s.plannerP.On("AbandonPlan", mock.Anything).Return(nil)
	s.plannerView.abandonPlanCallback()

	s.plannerP.AssertCalled(s.T(), "AbandonPlan", mock.Anything)
}

func (s *AppBinderSuite) TestBindWiresFocusRailBackToPreviousStep() {
	s.expectBindCalls()

	binder, err := ui.NewAppBinder(s.plannerP, s.focusRail, s.wizardView, s.plannerView, s.viewRouter)
	s.Require().NoError(err)

	binder.Bind()

	s.Require().NotNil(s.focusRail.backCallback, "Bind should wire a Back callback on focusRail")

	s.plannerP.On("PreviousStep").Return()
	s.focusRail.backCallback()

	s.plannerP.AssertCalled(s.T(), "PreviousStep")
}
