package ui

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// WizardViewModel abstracts the wizard presenter for the WizardView.
//
// Feature 107 WP5 collapsed the wizard to a schedule-generation flow:
// StepIdle → StepSchedule → StepActive. The view never edits todos.
type WizardViewModel interface {
	CurrentStep() presenter.WizardStep
	FocusSchedule() *presenter.SchedulePreview
	RecoverySchedule() *presenter.SchedulePreview

	StartPlanning(ctx context.Context) error
	PreviousStep()
	SelectSchedule(ctx context.Context, strategy string) error
}

// WizardView renders the day planner wizard step content.
type WizardView struct {
	vm     WizardViewModel
	router *CenterViewRouter

	container *fyne.Container

	// Cached state from buildState
	scheduleCards        int
	focusCardStrategy    string
	recoveryCardStrategy string
	focusCardStats       ScheduleCardStats
	recoveryCardStats    ScheduleCardStats
}

// ScheduleCardStats holds the display stats for a schedule card.
type ScheduleCardStats struct {
	FocusBlocks int
	Breaks      int
	TotalTime   string
}

// NewWizardView creates a new WizardView bound to the given view model and router.
func NewWizardView(vm WizardViewModel, router *CenterViewRouter) *WizardView {
	v := &WizardView{
		vm:        vm,
		router:    router,
		container: container.NewVBox(),
	}
	v.buildState()
	v.renderContainer()
	return v
}

// Container returns the top-level Fyne container for the wizard view.
func (v *WizardView) Container() *fyne.Container {
	return v.container
}

// buildState reads the view model and populates cached card data.
func (v *WizardView) buildState() {
	v.scheduleCards = 0
	v.focusCardStrategy = ""
	v.recoveryCardStrategy = ""
	v.focusCardStats = ScheduleCardStats{}
	v.recoveryCardStats = ScheduleCardStats{}

	focus := v.vm.FocusSchedule()
	recovery := v.vm.RecoverySchedule()

	v.buildScheduleCard(focus, &v.focusCardStrategy, &v.focusCardStats)
	v.buildScheduleCard(recovery, &v.recoveryCardStrategy, &v.recoveryCardStats)
}

func (v *WizardView) buildScheduleCard(preview *presenter.SchedulePreview, strategy *string, stats *ScheduleCardStats) {
	if preview != nil {
		v.scheduleCards++
		*strategy = preview.Strategy
		*stats = buildCardStats(preview)
	}
}

func buildCardStats(preview *presenter.SchedulePreview) ScheduleCardStats {
	focusBlocks := 0
	for _, b := range preview.Blocks {
		if b.Type == "focus" {
			focusBlocks++
		}
	}
	return ScheduleCardStats{
		FocusBlocks: focusBlocks,
		Breaks:      preview.BreakCount,
		TotalTime:   formatDuration(preview.TotalFocusTime),
	}
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}

// ScheduleCards returns the number of schedule choice cards.
func (v *WizardView) ScheduleCards() int { return v.scheduleCards }

// FocusCardStrategy returns the strategy name on the focus card.
func (v *WizardView) FocusCardStrategy() string { return v.focusCardStrategy }

// RecoveryCardStrategy returns the strategy name on the recovery card.
func (v *WizardView) RecoveryCardStrategy() string { return v.recoveryCardStrategy }

// FocusCardStats returns the stats for the focus card.
func (v *WizardView) FocusCardStats() ScheduleCardStats { return v.focusCardStats }

// RecoveryCardStats returns the stats for the recovery card.
func (v *WizardView) RecoveryCardStats() ScheduleCardStats { return v.recoveryCardStats }

// renderContainer clears the container and dispatches to step-specific render methods.
func (v *WizardView) renderContainer() {
	v.container.Objects = nil

	switch v.vm.CurrentStep() {
	case presenter.StepIdle:
		v.renderIdle()
	case presenter.StepSchedule:
		v.renderSchedule()
	}

	v.container.Refresh()
}

// renderIdle renders the idle state prompt when no wizard step is active.
func (v *WizardView) renderIdle() {
	v.container.Objects = append(v.container.Objects,
		widget.NewLabel("Use \"Plan My Day\" to start planning your day."))
}

// renderSchedule renders the two schedule preview cards and the Back button.
func (v *WizardView) renderSchedule() {
	if v.focusCardStrategy != "" {
		strategy := v.focusCardStrategy
		v.container.Objects = append(v.container.Objects,
			widget.NewButton("Select "+strategy, func() {
				v.vm.SelectSchedule(context.Background(), strategy) // #nosec G104 -- GUI callback; error logged by presenter
			}))
	}
	if v.recoveryCardStrategy != "" {
		strategy := v.recoveryCardStrategy
		v.container.Objects = append(v.container.Objects,
			widget.NewButton("Select "+strategy, func() {
				v.vm.SelectSchedule(context.Background(), strategy) // #nosec G104 -- GUI callback; error logged by presenter
			}))
	}

	v.container.Objects = append(v.container.Objects,
		widget.NewButton("Back", func() {
			v.vm.PreviousStep()
			if v.router != nil {
				v.router.NavigateTo(ViewPlan)
			}
		}),
	)
}

// Refresh updates the wizard view from the current model state.
func (v *WizardView) Refresh() {
	v.buildState()
	v.renderContainer()
}
