package presenter

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
)

// TodoQuerier abstracts todo query operations needed by the planner presenter.
type TodoQuerier interface {
	QueryFiltered(ctx context.Context, filter repository.TaskFilter) ([]*repository.Task, int, error)
	Insert(ctx context.Context, todo *repository.Task) error
	Update(ctx context.Context, todo *repository.Task) error
	Complete(ctx context.Context, id uuid.UUID, completedAt time.Time) error
}

// CategoryQuerier abstracts category query operations needed by the planner presenter.
//
// Updated for Feature 109: QueryAll takes a withCounts flag and returns
// CategoryWithCount so the planner can show task totals alongside each
// category. The presenter currently only needs the name_key list and
// passes withCounts=false.
type CategoryQuerier interface {
	QueryAll(ctx context.Context, withCounts bool) ([]*repository.CategoryWithCount, error)
}

// ScheduleGenerator abstracts schedule generation for the planner
// presenter. Feature 107: the signature collapsed from
// (ctx, tasks, events, date) to (ctx, date). Tasks are managed in the
// todo list view, not selected per plan; calendar events are fetched
// server-side. The generator returns a focus-maximized schedule and a
// recovery-balanced schedule for the same date.
type ScheduleGenerator interface {
	GenerateSchedules(ctx context.Context, targetDate time.Time) (*planner.DaySchedule, *planner.DaySchedule, error)
}

// WizardStep represents the current step in the day planner wizard.
//
// Feature 107 WP5 collapsed the wizard to a schedule-generation flow:
// StepIdle → StepSchedule → StepActive. Todos are managed in the Plan
// view's todo list (a separate surface); the wizard never edits them.
type WizardStep int

const (
	StepIdle WizardStep = iota
	StepSchedule
	StepActive
)

// TodoRow is the view model for a todo item.
type TodoRow struct {
	ID         uuid.UUID
	Title      string
	Priority   int
	DueDate    *time.Time
	Categories []repository.Category
	Selected   bool
}

// TaskEstimateRow is the view model for a task estimate.
//
// Retained as a public type for downstream consumers; the wizard no
// longer uses it after Feature 107 WP5.
type TaskEstimateRow struct {
	TodoID         uuid.UUID
	Title          string
	EstimatedPomos int
	UserOverride   *int
	EffectivePomos int
}

// SchedulePreview is the view model for a generated schedule.
type SchedulePreview struct {
	Strategy       string
	TotalFocusTime time.Duration
	BreakCount     int
	Blocks         []TimeBlockPreview
}

// TimeBlockPreview is the view model for a single time block.
type TimeBlockPreview struct {
	Start    time.Time
	End      time.Time
	Type     string
	TaskName string
	TaskID   *uuid.UUID
}

// ActiveScheduleState is the view model for an active schedule.
type ActiveScheduleState struct {
	Blocks       []TimeBlockPreview
	CurrentIndex int
	CurrentBlock *TimeBlockPreview
}

// PlannerPresenter manages the day planner wizard state and view models.
type PlannerPresenter struct {
	// Dependencies
	todos     TodoQuerier
	cats      CategoryQuerier
	generator ScheduleGenerator
	schedRepo repository.ScheduleRepository
	clock     planner.Clock

	// Wizard state
	step WizardStep

	// Generated schedules
	focusSchedule    *planner.DaySchedule
	recoverySchedule *planner.DaySchedule

	// Active schedule state
	activeScheduleID uuid.UUID
	activeBlocks     []TimeBlockPreview
	activeIndex      int

	// Step change callback
	onStepChange func(WizardStep)
}

// NewPlannerPresenter creates a new PlannerPresenter, validating all
// dependencies.
func NewPlannerPresenter(
	todos TodoQuerier,
	cats CategoryQuerier,
	generator ScheduleGenerator,
	schedRepo repository.ScheduleRepository,
	clock planner.Clock,
) (*PlannerPresenter, error) {
	if todos == nil {
		return nil, fmt.Errorf("todos must not be nil")
	}
	if cats == nil {
		return nil, fmt.Errorf("categories must not be nil")
	}
	if generator == nil {
		return nil, fmt.Errorf("generator must not be nil")
	}
	if schedRepo == nil {
		return nil, fmt.Errorf("schedRepo must not be nil")
	}
	if clock == nil {
		return nil, fmt.Errorf("clock must not be nil")
	}
	return &PlannerPresenter{
		todos:     todos,
		cats:      cats,
		generator: generator,
		schedRepo: schedRepo,
		clock:     clock,
		step:      StepIdle,
	}, nil
}

// SetOnStepChange registers a callback that fires whenever the wizard step changes.
func (p *PlannerPresenter) SetOnStepChange(fn func(WizardStep)) {
	p.onStepChange = fn
}

func (p *PlannerPresenter) fireStepChange() {
	if p.onStepChange != nil {
		p.onStepChange(p.step)
	}
}

// === Wizard Navigation ===

// CurrentStep returns the current wizard step.
func (p *PlannerPresenter) CurrentStep() WizardStep {
	return p.step
}

// StartPlanning runs the schedule generator for today and transitions
// to StepSchedule. The caller can then inspect FocusSchedule /
// RecoverySchedule and pick one via SelectSchedule.
func (p *PlannerPresenter) StartPlanning(ctx context.Context) error {
	focus, recovery, err := p.generator.GenerateSchedules(ctx, p.todayDate())
	if err != nil {
		return fmt.Errorf("generating schedules: %w", err)
	}
	p.focusSchedule = focus
	p.recoverySchedule = recovery
	p.step = StepSchedule
	p.fireStepChange()
	return nil
}

// NextStep is a no-op that returns an error for StepSchedule (use
// SelectSchedule instead) and for StepActive. It exists to satisfy the
// PlannerCallbacks contract; the wizard's only forward transition is
// SelectSchedule.
func (p *PlannerPresenter) NextStep(_ context.Context) error {
	switch p.step {
	case StepIdle:
		return nil
	case StepSchedule:
		return fmt.Errorf("cannot advance from schedule step: use SelectSchedule")
	default:
		return fmt.Errorf("cannot advance from step %d", p.step)
	}
}

// PreviousStep moves the wizard back. From StepSchedule it returns to
// StepIdle; from any other step it is a no-op.
func (p *PlannerPresenter) PreviousStep() {
	if p.step == StepSchedule {
		p.step = StepIdle
		p.fireStepChange()
	}
}

// === Schedule Management ===

// FocusSchedule returns the focus-maximized schedule preview.
func (p *PlannerPresenter) FocusSchedule() *SchedulePreview {
	if p.focusSchedule == nil {
		return nil
	}
	return dayScheduleToPreview(p.focusSchedule)
}

// RecoverySchedule returns the recovery-balanced schedule preview.
func (p *PlannerPresenter) RecoverySchedule() *SchedulePreview {
	if p.recoverySchedule == nil {
		return nil
	}
	return dayScheduleToPreview(p.recoverySchedule)
}

// SelectSchedule saves the chosen schedule and transitions to StepActive.
func (p *PlannerPresenter) SelectSchedule(ctx context.Context, strategy string) error {
	var chosen *planner.DaySchedule
	switch strategy {
	case "focus-maximized":
		chosen = p.focusSchedule
	case "recovery-balanced":
		chosen = p.recoverySchedule
	default:
		return fmt.Errorf("unknown strategy: %s", strategy)
	}
	if chosen == nil {
		return fmt.Errorf("no schedule available for strategy: %s", strategy)
	}

	repoSchedule := &repository.Schedule{
		ID:        chosen.ID,
		Date:      p.todayDate(),
		Strategy:  strategy,
		Blocks:    timeBlocksToRepoBlocks(chosen.Blocks),
		CreatedAt: p.clock.Now(),
	}
	if err := p.schedRepo.Save(ctx, repoSchedule); err != nil {
		return fmt.Errorf("saving schedule: %w", err)
	}

	p.activeScheduleID = repoSchedule.ID
	p.activeBlocks = timeBlocksToPreview(chosen.Blocks)
	p.activeIndex = 0
	p.step = StepActive
	p.fireStepChange()
	return nil
}

// === Active Plan Management ===

// ActiveSchedule returns the current active schedule state.
func (p *PlannerPresenter) ActiveSchedule() *ActiveScheduleState {
	if p.step != StepActive {
		return nil
	}
	state := &ActiveScheduleState{
		Blocks:       p.activeBlocks,
		CurrentIndex: p.activeIndex,
	}
	if p.activeIndex >= 0 && p.activeIndex < len(p.activeBlocks) {
		block := p.activeBlocks[p.activeIndex]
		state.CurrentBlock = &block
	}
	return state
}

// CompleteCurrentTask marks the current focus block's task as completed.
func (p *PlannerPresenter) CompleteCurrentTask(ctx context.Context) error {
	if p.step != StepActive {
		return fmt.Errorf("no active plan")
	}
	if !p.isValidActiveIndex() {
		return fmt.Errorf("no current block")
	}

	block := p.activeBlocks[p.activeIndex]
	if block.TaskID == nil {
		return nil // No task associated with this block; silently skip.
	}
	if err := p.todos.Complete(ctx, *block.TaskID, p.clock.Now()); err != nil {
		return fmt.Errorf("completing task: %w", err)
	}
	return nil
}

// AbandonPlan deletes the active schedule and returns to StepIdle.
func (p *PlannerPresenter) AbandonPlan(ctx context.Context) error {
	if err := p.schedRepo.Delete(ctx, p.todayDate()); err != nil {
		return fmt.Errorf("deleting schedule: %w", err)
	}
	p.step = StepIdle
	p.fireStepChange()
	p.activeBlocks = nil
	p.activeIndex = 0
	p.activeScheduleID = uuid.Nil
	return nil
}

// HasActivePlan returns true if there is an active schedule.
func (p *PlannerPresenter) HasActivePlan() bool {
	return p.step == StepActive
}

// LoadExistingPlan loads a schedule for today from the repository.
func (p *PlannerPresenter) LoadExistingPlan(ctx context.Context) error {
	date := p.todayDate()

	schedule, err := p.schedRepo.LoadByDate(ctx, date)
	if err != nil {
		return fmt.Errorf("loading schedule: %w", err)
	}
	if schedule == nil {
		return nil
	}

	p.activeScheduleID = schedule.ID
	p.activeBlocks = repoBlocksToPreview(schedule.Blocks)
	p.activeIndex = 0
	p.step = StepActive
	p.fireStepChange()
	return nil
}

// === Internal Helpers ===

// Date helpers
func (p *PlannerPresenter) todayDate() time.Time {
	now := p.clock.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func (p *PlannerPresenter) isValidActiveIndex() bool {
	return p.activeIndex >= 0 && p.activeIndex < len(p.activeBlocks)
}

// CurrentFocusTask returns the highest-priority pending todo, or
// (nil, nil) when the user has no incomplete todos. Feature 107: the
// active-schedule view consumes this as a single-task hint rendered
// alongside each focus block.
//
// Highest priority = lowest Priority integer (the ordering convention
// used throughout the todo views). Ties break by earliest CreatedAt
// to keep the choice stable across calls.
func (p *PlannerPresenter) CurrentFocusTask(ctx context.Context) (*TodoRow, error) {
	tasks, _, err := p.todos.QueryFiltered(ctx, repository.TaskFilter{Status: "incomplete"})
	if err != nil {
		return nil, fmt.Errorf("loading incomplete todos: %w", err)
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].Priority != tasks[j].Priority {
			return tasks[i].Priority < tasks[j].Priority
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	row := todoToRow(tasks[0])
	return &row, nil
}

// Model conversion helpers

func todoToRow(t *repository.Task) TodoRow {
	return TodoRow{
		ID:       t.ID,
		Title:    t.Title,
		Priority: t.Priority,
		DueDate:  t.DueDate,
		// TODO(feat-109 Loop 7): populate from t.CategoryKey via the
		// CategoryRepository so views can render the display name.
		Categories: nil,
		Selected:   false,
	}
}

func blockTypeString(bt planner.BlockType) string {
	switch bt {
	case planner.BlockFocus:
		return "focus"
	case planner.BlockShortBreak:
		return "short_break"
	case planner.BlockLongBreak:
		return "long_break"
	case planner.BlockMeeting:
		return "meeting"
	default:
		return "unknown"
	}
}

func repoBlockTypeString(bt repository.ScheduleBlockType) string {
	switch bt {
	case repository.ScheduleBlockFocus:
		return "focus"
	case repository.ScheduleBlockShortBreak:
		return "short_break"
	case repository.ScheduleBlockLongBreak:
		return "long_break"
	case repository.ScheduleBlockMeeting:
		return "meeting"
	default:
		return "unknown"
	}
}

func dayScheduleToPreview(ds *planner.DaySchedule) *SchedulePreview {
	preview := &SchedulePreview{
		Strategy: ds.Strategy,
		Blocks:   timeBlocksToPreview(ds.Blocks),
	}
	for _, b := range ds.Blocks {
		if b.Type == planner.BlockFocus {
			preview.TotalFocusTime += b.End.Sub(b.Start)
		}
		if b.Type == planner.BlockShortBreak || b.Type == planner.BlockLongBreak {
			preview.BreakCount++
		}
	}
	return preview
}

func timeBlocksToPreview(blocks []planner.TimeBlock) []TimeBlockPreview {
	result := make([]TimeBlockPreview, len(blocks))
	for i, b := range blocks {
		result[i] = TimeBlockPreview{
			Start:    b.Start,
			End:      b.End,
			Type:     blockTypeString(b.Type),
			TaskName: b.TaskName,
			TaskID:   b.TaskID,
		}
	}
	return result
}

func repoBlocksToPreview(blocks []repository.ScheduleBlock) []TimeBlockPreview {
	result := make([]TimeBlockPreview, len(blocks))
	for i, b := range blocks {
		result[i] = TimeBlockPreview{
			Start:    b.Start,
			End:      b.End,
			Type:     repoBlockTypeString(b.Type),
			TaskName: b.TaskName,
			TaskID:   b.TaskID,
		}
	}
	return result
}

func timeBlocksToRepoBlocks(blocks []planner.TimeBlock) []repository.ScheduleBlock {
	result := make([]repository.ScheduleBlock, len(blocks))
	for i, b := range blocks {
		result[i] = repository.ScheduleBlock{
			Start:    b.Start,
			End:      b.End,
			Type:     repository.ScheduleBlockType(b.Type),
			TaskID:   b.TaskID,
			TaskName: b.TaskName,
		}
	}
	return result
}
