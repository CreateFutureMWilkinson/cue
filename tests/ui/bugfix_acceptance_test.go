//go:build ui_acceptance

package ui_acceptance_test

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/uitest"
)

// =============================================================================
// Bug 065 — Settings view missing Calendar tab
// =============================================================================

type Bug065Suite struct {
	suite.Suite
}

func TestBug065(t *testing.T) {
	suite.Run(t, new(Bug065Suite))
}

// AC: Settings view should have 5 tabs (Slack, Email, Calendar, Audio, Ollama).
func (s *Bug065Suite) TestSettingsViewHasFiveTabs() {
	sv := newSettingsView()
	s.Equal(5, sv.TabCount(),
		"Bug 065: settings view should have 5 tabs (Slack, Email, Calendar, Audio, Ollama)")
}

// AC: Tab order is Slack, Email, Calendar, Audio, Ollama.
func (s *Bug065Suite) TestSettingsViewTabOrder() {
	sv := newSettingsView()
	expected := []string{"Slack", "Email", "Calendar", "Audio", "Ollama"}
	s.Equal(expected, sv.TabNames(),
		"Bug 065: tab order should be Slack, Email, Calendar, Audio, Ollama")
}

// AC: Calendar tab has an "Add Account" button.
func (s *Bug065Suite) TestCalendarTabHasAddAccountButton() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	// Calendar should be at index 2 (after Slack, Email).
	s.Require().GreaterOrEqual(len(tabs.Items), 3, "should have at least 3 tabs")
	calContent := tabs.Items[2].Content

	_, found := uitest.FindWidget[*widget.Button](calContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})
	s.True(found,
		"Bug 065: Calendar tab should have an 'Add Account' button")
}

// =============================================================================
// Bug 066 — PlannerView no-plan content not rendered in widget tree
// =============================================================================

type Bug066Suite struct {
	suite.Suite
}

func TestBug066(t *testing.T) {
	suite.Run(t, new(Bug066Suite))
}

// AC: No-plan state shows placeholder text visible in the widget tree.
func (s *Bug066Suite) TestPlaceholderTextRenderedInWidgetTree() {
	router := ui.NewCenterViewRouter()
	vm := &stubPlannerTimerVM{hasActivePlan: false}
	pv := ui.NewPlannerView(vm, vm, router, vm)
	root := pv.Container()

	placeholderText := pv.PlaceholderText()
	s.Require().NotEmpty(placeholderText, "placeholder text field should be set")

	// The placeholder text must actually appear as a Label in the widget tree,
	// not just stored in a field.
	_, found := uitest.FindWidget[*widget.Label](root, func(l *widget.Label) bool {
		return l.Text == placeholderText
	})
	s.True(found,
		"Bug 066: placeholder text %q should be rendered as a Label in the widget tree", placeholderText)
}

// AC: "Plan My Day" button tap navigates to ViewWizard.
func (s *Bug066Suite) TestPlanMyDayButtonNavigatesToWizard() {
	router := ui.NewCenterViewRouter()
	vm := &stubPlannerTimerVM{hasActivePlan: false}
	pv := ui.NewPlannerView(vm, vm, router, vm)
	root := pv.Container()

	btn, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Plan My Day"
	})
	s.Require().True(found, "should find 'Plan My Day' button in widget tree")

	btn.OnTapped()

	s.Equal(ui.ViewWizard, router.CurrentView(),
		"Bug 066: tapping 'Plan My Day' should navigate to ViewWizard")
}

// =============================================================================
// Bug 067 — Email settings Add Account callback is noop
// =============================================================================

type Bug067Suite struct {
	suite.Suite
}

func TestBug067(t *testing.T) {
	suite.Run(t, new(Bug067Suite))
}

// AC: Tapping "Add Account" in Email tab triggers a visible effect (form or dialog).
func (s *Bug067Suite) TestEmailAddAccountCallbackIsWired() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	// Email tab is at index 1.
	s.Require().GreaterOrEqual(len(tabs.Items), 2)
	emailContent := tabs.Items[1].Content

	btn := uitest.RequireWidget[*widget.Button](s.T(), emailContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	// Tap the button — afterward, the email tab content should have changed
	// (e.g., form fields appeared). We check for Entry widgets that weren't
	// there before.
	entriesBefore := uitest.FindAll[*widget.Entry](emailContent, func(_ *widget.Entry) bool {
		return true
	})

	btn.OnTapped()

	// Re-read the tab content after tap — the tree may have been rebuilt.
	emailContentAfter := tabs.Items[1].Content
	entriesAfter := uitest.FindAll[*widget.Entry](emailContentAfter, func(_ *widget.Entry) bool {
		return true
	})

	s.Greater(len(entriesAfter), len(entriesBefore),
		"Bug 067: tapping Add Account in Email tab should add form entry fields")
}

// =============================================================================
// Bug 068 — Slack settings Add Account callback is noop
// =============================================================================

type Bug068Suite struct {
	suite.Suite
}

func TestBug068(t *testing.T) {
	suite.Run(t, new(Bug068Suite))
}

// AC: Tapping "Add Account" in Slack tab triggers a visible effect (form or dialog).
func (s *Bug068Suite) TestSlackAddAccountCallbackIsWired() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})

	// Slack tab is at index 0.
	s.Require().GreaterOrEqual(len(tabs.Items), 1)
	slackContent := tabs.Items[0].Content

	btn := uitest.RequireWidget[*widget.Button](s.T(), slackContent, func(b *widget.Button) bool {
		return b.Text == "Add Account"
	})

	entriesBefore := uitest.FindAll[*widget.Entry](slackContent, func(_ *widget.Entry) bool {
		return true
	})

	btn.OnTapped()

	slackContentAfter := tabs.Items[0].Content
	entriesAfter := uitest.FindAll[*widget.Entry](slackContentAfter, func(_ *widget.Entry) bool {
		return true
	})

	s.Greater(len(entriesAfter), len(entriesBefore),
		"Bug 068: tapping Add Account in Slack tab should add form entry fields")
}

// =============================================================================
// Bug 069 — Audio settings missing Timer Volume slider
// =============================================================================

type Bug069Suite struct {
	suite.Suite
}

func TestBug069(t *testing.T) {
	suite.Run(t, new(Bug069Suite))
}

// AC: Audio tab has two sliders (Notification Volume + Timer Volume).
func (s *Bug069Suite) TestAudioTabHasTwoSliders() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})
	s.Require().GreaterOrEqual(len(tabs.Items), 4)
	audioContent := tabs.Items[3].Content

	sliders := uitest.FindAll[*widget.Slider](audioContent, func(_ *widget.Slider) bool {
		return true
	})

	s.Equal(2, len(sliders),
		"Bug 069: Audio tab should have exactly 2 sliders (Notification + Timer)")
}

// AC: Timer volume label exists.
func (s *Bug069Suite) TestAudioTabHasTimerVolumeLabel() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})
	audioContent := tabs.Items[3].Content

	_, found := uitest.FindWidget[*widget.Label](audioContent, func(l *widget.Label) bool {
		return strings.Contains(l.Text, "Timer Volume")
	})

	s.True(found,
		"Bug 069: Audio tab should have a 'Timer Volume' label")
}

// AC: Timer volume label updates live during drag.
func (s *Bug069Suite) TestTimerVolumeSliderUpdatesLabel() {
	sv := newSettingsView()
	root := sv.Container()

	tabs := uitest.RequireWidget[*container.AppTabs](s.T(), root, func(_ *container.AppTabs) bool {
		return true
	})
	audioContent := tabs.Items[3].Content

	sliders := uitest.FindAll[*widget.Slider](audioContent, func(_ *widget.Slider) bool {
		return true
	})
	s.Require().Equal(2, len(sliders), "need 2 sliders to test timer volume")

	// The second slider should be the timer volume slider.
	timerSlider := sliders[1]
	s.Require().NotNil(timerSlider.OnChanged, "timer slider should have OnChanged wired")
	timerSlider.OnChanged(60)

	_, found := uitest.FindWidget[*widget.Label](audioContent, func(l *widget.Label) bool {
		return l.Text == "Timer Volume: 60%"
	})
	s.True(found,
		"Bug 069: dragging timer slider to 60 should update label to 'Timer Volume: 60%%'")
}

// =============================================================================
// Bug 070 — Activity log drawer uses split instead of overlay
// =============================================================================

type Bug070Suite struct {
	suite.Suite
}

func TestBug070(t *testing.T) {
	suite.Run(t, new(Bug070Suite))
}

// AC: When open, activity log overlays the character area (not a VSplit).
func (s *Bug070Suite) TestOpenDrawerUsesStackNotSplit() {
	ap := s.newActivityPresenter()
	drawer := ui.NewActivityLogDrawer(ap)
	charWidget := widget.NewLabel("Character")
	root := drawer.ContainerWithCharacter(charWidget)

	drawer.ToggleOpen()

	// After opening, the container should NOT contain a Split — it should use
	// a Stack layout where the log overlays the character.
	_, hasSplit := uitest.FindWidget[*container.Split](root, func(_ *container.Split) bool {
		return true
	})
	s.False(hasSplit,
		"Bug 070: open activity log should overlay character via Stack, not use a Split")
}

// AC: Overlay has a semi-transparent dark background.
func (s *Bug070Suite) TestOpenDrawerHasSemiTransparentBackground() {
	ap := s.newActivityPresenter()
	drawer := ui.NewActivityLogDrawer(ap)
	charWidget := widget.NewLabel("Character")
	root := drawer.ContainerWithCharacter(charWidget)

	drawer.ToggleOpen()

	// Look for a canvas.Rectangle with non-zero alpha < 255 (semi-transparent).
	_, found := uitest.FindWidget[*canvas.Rectangle](root, func(r *canvas.Rectangle) bool {
		if r.FillColor == nil {
			return false
		}
		_, _, _, a := r.FillColor.RGBA()
		// Semi-transparent: alpha > 0 and alpha < full opacity (0xFFFF).
		return a > 0 && a < 0xFFFF
	})
	s.True(found,
		"Bug 070: open activity log overlay should have a semi-transparent background rectangle")
}

func (s *Bug070Suite) newActivityPresenter() *presenter.ActivityPresenter {
	source := newMockActivitySource()
	ap, err := presenter.NewActivityPresenter(source, 500)
	s.Require().NoError(err)
	return ap
}

// =============================================================================
// Bug 072 — Wizard step 3 Up/Down reorder buttons are noops
// =============================================================================

type Bug072Suite struct {
	suite.Suite
}

func TestBug072(t *testing.T) {
	suite.Run(t, new(Bug072Suite))
}

// AC: Tapping "Up" calls ReorderTask on the view model.
func (s *Bug072Suite) TestUpButtonCallsReorderTask() {
	router := ui.NewCenterViewRouter()
	vm := &trackingWizardVM{
		stubWizardVM: stubWizardVM{
			step: presenter.StepPriority,
			tasks: []presenter.TodoRow{
				{Title: "Task A", Priority: 1},
				{Title: "Task B", Priority: 2},
			},
			estimates: []presenter.TaskEstimateRow{
				{Title: "Task A", EstimatedPomos: 2, EffectivePomos: 2},
				{Title: "Task B", EstimatedPomos: 3, EffectivePomos: 3},
			},
		},
	}
	wv := ui.NewWizardView(vm, router)
	root := wv.Container()

	btn, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Up" && !b.Disabled()
	})
	s.Require().True(found, "step 3 should have an enabled 'Up' button")

	btn.OnTapped()

	s.True(vm.reorderCalled,
		"Bug 072: tapping 'Up' should call ReorderTask on the view model")
}

// AC: Tapping "Down" calls ReorderTask on the view model.
func (s *Bug072Suite) TestDownButtonCallsReorderTask() {
	router := ui.NewCenterViewRouter()
	vm := &trackingWizardVM{
		stubWizardVM: stubWizardVM{
			step: presenter.StepPriority,
			tasks: []presenter.TodoRow{
				{Title: "Task A", Priority: 1},
				{Title: "Task B", Priority: 2},
			},
			estimates: []presenter.TaskEstimateRow{
				{Title: "Task A", EstimatedPomos: 2, EffectivePomos: 2},
				{Title: "Task B", EstimatedPomos: 3, EffectivePomos: 3},
			},
		},
	}
	wv := ui.NewWizardView(vm, router)
	root := wv.Container()

	btn, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Down" && !b.Disabled()
	})
	s.Require().True(found, "step 3 should have an enabled 'Down' button")

	btn.OnTapped()

	s.True(vm.reorderCalled,
		"Bug 072: tapping 'Down' should call ReorderTask on the view model")
}

// =============================================================================
// Bug 073 — PlannerView navigation buttons not wired
// =============================================================================

type Bug073Suite struct {
	suite.Suite
}

func TestBug073(t *testing.T) {
	suite.Run(t, new(Bug073Suite))
}

// AC: "Next" button calls through to its wired callback.
func (s *Bug073Suite) TestNextButtonCallsCallback() {
	router := ui.NewCenterViewRouter()
	vm := &stubPlannerTimerVM{step: presenter.StepTaskSelect}
	pv := ui.NewPlannerView(vm, vm, router, vm)

	called := false
	pv.SetOnNext(func() { called = true })

	btn := pv.NextButton()
	s.Require().True(btn.Visible(), "Next button should be visible in StepTaskSelect")
	btn.OnTapped()

	s.True(called, "Bug 073: tapping 'Next' should invoke the wired callback")
}

// AC: "Back" button calls through to its wired callback.
func (s *Bug073Suite) TestBackButtonCallsCallback() {
	router := ui.NewCenterViewRouter()
	vm := &stubPlannerTimerVM{step: presenter.StepEstimates}
	pv := ui.NewPlannerView(vm, vm, router, vm)

	called := false
	pv.SetOnBack(func() { called = true })

	btn := pv.BackButton()
	s.Require().True(btn.Visible(), "Back button should be visible in StepEstimates")
	btn.OnTapped()

	s.True(called, "Bug 073: tapping 'Back' should invoke the wired callback")
}

// AC: "Complete Task" button calls through to its wired callback.
func (s *Bug073Suite) TestCompleteTaskButtonCallsCallback() {
	router := ui.NewCenterViewRouter()
	vm := &stubPlannerTimerVM{
		step:          presenter.StepActive,
		hasActivePlan: true,
		activeSchedule: &presenter.ActiveScheduleState{
			Blocks:       nil,
			CurrentIndex: 0,
		},
	}
	pv := ui.NewPlannerView(vm, vm, router, vm)

	called := false
	pv.SetOnCompleteTask(func() { called = true })

	btn := pv.CompleteTaskButton()
	s.Require().True(btn.Visible(), "Complete Task button should be visible in StepActive")
	btn.OnTapped()

	s.True(called, "Bug 073: tapping 'Complete Task' should invoke the wired callback")
}

// AC: "Abandon Plan" button calls through to its wired callback.
func (s *Bug073Suite) TestAbandonButtonCallsCallback() {
	router := ui.NewCenterViewRouter()
	vm := &stubPlannerTimerVM{
		step:          presenter.StepActive,
		hasActivePlan: true,
		activeSchedule: &presenter.ActiveScheduleState{
			Blocks:       nil,
			CurrentIndex: 0,
		},
	}
	pv := ui.NewPlannerView(vm, vm, router, vm)

	called := false
	pv.SetOnAbandonPlan(func() { called = true })

	btn := pv.AbandonButton()
	s.Require().True(btn.Visible(), "Abandon button should be visible in StepActive")
	btn.OnTapped()

	s.True(called, "Bug 073: tapping 'Abandon Plan' should invoke the wired callback")
}

// AC: All four buttons handle nil callback gracefully (no panics).
func (s *Bug073Suite) TestButtonsDoNotPanicWithoutCallbacks() {
	router := ui.NewCenterViewRouter()
	vm := &stubPlannerTimerVM{
		step:          presenter.StepActive,
		hasActivePlan: true,
		activeSchedule: &presenter.ActiveScheduleState{
			Blocks:       nil,
			CurrentIndex: 0,
		},
	}
	pv := ui.NewPlannerView(vm, vm, router, vm)

	s.NotPanics(func() { pv.AbandonButton().OnTapped() },
		"Abandon button should not panic without a wired callback")
	s.NotPanics(func() { pv.CompleteTaskButton().OnTapped() },
		"Complete Task button should not panic without a wired callback")
}

// =============================================================================
// Tracking mocks — extend stubs to record calls
// =============================================================================

// trackingWizardVM wraps stubWizardVM to track ReorderTask calls.
type trackingWizardVM struct {
	stubWizardVM
	reorderCalled bool
	reorderFrom   int
	reorderTo     int
}

func (t *trackingWizardVM) ReorderTask(from, to int) {
	t.reorderCalled = true
	t.reorderFrom = from
	t.reorderTo = to
}

// =============================================================================
// Bug 070A — Activity log button fills entire center panel when closed
// =============================================================================

type Bug070ASuite struct {
	suite.Suite
}

func TestBug070A(t *testing.T) {
	suite.Run(t, new(Bug070ASuite))
}

func (s *Bug070ASuite) newActivityPresenter() *presenter.ActivityPresenter {
	source := newMockActivitySource()
	ap, err := presenter.NewActivityPresenter(source, 500)
	s.Require().NoError(err)
	return ap
}

// AC: When closed, the stackContainer should have a single child (a Border layout)
// rather than multiple children in a Stack (which would stretch both to fill).
// The bug: Stack(character, drawerBox) causes drawerBox to fill the entire panel.
// The fix: Stack(Border(nil, button, nil, nil, character)) — one child, button at bottom.
func (s *Bug070ASuite) TestClosedStateSingleChildInStack() {
	ap := s.newActivityPresenter()
	drawer := ui.NewActivityLogDrawer(ap)
	charWidget := widget.NewLabel("Character")
	root := drawer.ContainerWithCharacter(charWidget)

	topContainer, ok := root.(*fyne.Container)
	s.Require().True(ok, "root should be a *fyne.Container")

	// In the fixed layout, the top-level Stack should contain exactly 1 object
	// (a Border layout), not 2+ objects (character + drawerBox both filling via Stack).
	s.Equal(1, len(topContainer.Objects),
		"Bug 070A: closed state should have 1 child in the Stack (a Border wrapping character + button), not %d", len(topContainer.Objects))
}

// AC: When closed, button height matches natural widget.Button MinSize (not panel height).
func (s *Bug070ASuite) TestClosedButtonAtNaturalHeight() {
	ap := s.newActivityPresenter()
	drawer := ui.NewActivityLogDrawer(ap)
	charWidget := widget.NewLabel("Character")
	root := drawer.ContainerWithCharacter(charWidget)

	btn, found := uitest.FindWidget[*widget.Button](root, func(b *widget.Button) bool {
		return b.Text == "Activity Log"
	})
	s.Require().True(found, "should find Activity Log button")

	// A reference button gives us the natural MinSize height.
	refBtn := widget.NewButton("Reference", nil)
	s.Equal(refBtn.MinSize().Height, btn.MinSize().Height,
		"Bug 070A: closed toggle button MinSize height should match natural widget.Button height")
}

// AC: When opened then closed again, the single-child Border layout is restored.
func (s *Bug070ASuite) TestToggleBackToClosedRestoresSingleChild() {
	ap := s.newActivityPresenter()
	drawer := ui.NewActivityLogDrawer(ap)
	charWidget := widget.NewLabel("Character")
	root := drawer.ContainerWithCharacter(charWidget)

	drawer.ToggleOpen() // open
	drawer.ToggleOpen() // close again

	topContainer, ok := root.(*fyne.Container)
	s.Require().True(ok, "root should be a *fyne.Container")

	s.Equal(1, len(topContainer.Objects),
		"Bug 070A: after toggling back to closed, Stack should have 1 child (Border), not %d", len(topContainer.Objects))
}

// AC: Open state overlay still has semi-transparent background (regression check).
func (s *Bug070ASuite) TestOpenOverlayPreservedAfterFix() {
	ap := s.newActivityPresenter()
	drawer := ui.NewActivityLogDrawer(ap)
	charWidget := widget.NewLabel("Character")
	root := drawer.ContainerWithCharacter(charWidget)

	drawer.ToggleOpen()

	_, found := uitest.FindWidget[*canvas.Rectangle](root, func(r *canvas.Rectangle) bool {
		if r.FillColor == nil {
			return false
		}
		_, _, _, a := r.FillColor.RGBA()
		return a > 0 && a < 0xFFFF
	})
	s.True(found,
		"Bug 070A: open state should still have semi-transparent overlay after layout fix")
}
