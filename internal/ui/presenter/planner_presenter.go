package presenter

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/calendar"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
)

// TodoQuerier abstracts todo query operations needed by the planner presenter.
type TodoQuerier interface {
	QueryIncomplete(ctx context.Context) ([]*repository.Todo, error)
	Insert(ctx context.Context, todo *repository.Todo) error
	Update(ctx context.Context, todo *repository.Todo) error
	Complete(ctx context.Context, id uuid.UUID, completedAt time.Time) error
}

// CategoryQuerier abstracts category query operations needed by the planner presenter.
type CategoryQuerier interface {
	QueryAll(ctx context.Context) ([]*repository.Category, error)
}

// ScheduleGenerator abstracts schedule generation for the planner presenter.
type ScheduleGenerator interface {
	GenerateSchedules(ctx context.Context, tasks []planner.TaskEstimate, events []calendar.CalendarEvent, targetDate time.Time) (*planner.DaySchedule, *planner.DaySchedule, error)
}

// WizardStep represents the current step in the day planner wizard.
type WizardStep int

const (
	StepIdle       WizardStep = iota
	StepTaskSelect            // User selects tasks to plan
	StepEstimates             // Show/edit pomodoro estimates
	StepPriority              // Reorder task priority
	StepSchedule              // Preview generated schedules
	StepActive                // Active schedule in progress
)

// TodoRow is the view model for a todo item in the task selection step.
type TodoRow struct {
	ID         uuid.UUID
	Title      string
	Priority   int
	DueDate    *time.Time
	Categories []repository.Category
	Selected   bool
}

// TaskEstimateRow is the view model for a task estimate in the estimates step.
type TaskEstimateRow struct {
	TodoID         uuid.UUID
	Title          string
	EstimatedPomos int
	UserOverride   *int
	EffectivePomos int
}

// EstimateSummary summarizes the pomodoro estimates for selected tasks.
type EstimateSummary struct {
	TotalPomos      int
	AvailableBlocks int
	Overloaded      bool
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
	cal       calendar.CalendarProvider
	generator ScheduleGenerator
	estimator planner.TaskEstimator
	schedRepo repository.ScheduleRepository
	clock     planner.Clock

	// Wizard state
	step         WizardStep
	tasks        []TodoRow
	descriptions map[uuid.UUID]string // todo ID -> description
	estimates    []TaskEstimateRow

	// Generated schedules
	focusSchedule    *planner.DaySchedule
	recoverySchedule *planner.DaySchedule

	// Active schedule state
	activeScheduleID uuid.UUID
	activeBlocks     []TimeBlockPreview
	activeIndex      int
}

// NewPlannerPresenter creates a new PlannerPresenter, validating all dependencies.
func NewPlannerPresenter(
	todos TodoQuerier,
	cats CategoryQuerier,
	cal calendar.CalendarProvider,
	generator ScheduleGenerator,
	estimator planner.TaskEstimator,
	schedRepo repository.ScheduleRepository,
	clock planner.Clock,
) (*PlannerPresenter, error) {
	if todos == nil {
		return nil, fmt.Errorf("todos must not be nil")
	}
	if cats == nil {
		return nil, fmt.Errorf("categories must not be nil")
	}
	if cal == nil {
		return nil, fmt.Errorf("calendar must not be nil")
	}
	if generator == nil {
		return nil, fmt.Errorf("generator must not be nil")
	}
	if estimator == nil {
		return nil, fmt.Errorf("estimator must not be nil")
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
		cal:       cal,
		generator: generator,
		estimator: estimator,
		schedRepo: schedRepo,
		clock:     clock,
		step:      StepIdle,
	}, nil
}

// === Wizard Navigation ===

// CurrentStep returns the current wizard step.
func (p *PlannerPresenter) CurrentStep() WizardStep {
	return p.step
}

// StartPlanning loads incomplete todos and transitions to StepTaskSelect.
func (p *PlannerPresenter) StartPlanning(ctx context.Context) error {
	todos, err := p.todos.QueryIncomplete(ctx)
	if err != nil {
		return fmt.Errorf("loading incomplete todos: %w", err)
	}
	p.tasks = make([]TodoRow, len(todos))
	p.descriptions = make(map[uuid.UUID]string)
	for i, t := range todos {
		p.tasks[i] = todoToRow(t)
		p.descriptions[t.ID] = t.Description
	}
	p.step = StepTaskSelect
	return nil
}

// === Task Management ===

// AvailableTasks returns the current list of todo rows for selection.
func (p *PlannerPresenter) AvailableTasks() []TodoRow {
	result := make([]TodoRow, len(p.tasks))
	copy(result, p.tasks)
	return result
}

// SelectTask sets the selection state of a task by ID.
func (p *PlannerPresenter) SelectTask(id uuid.UUID, selected bool) {
	if taskIndex := p.findTaskIndex(id); taskIndex >= 0 {
		p.tasks[taskIndex].Selected = selected
	}
}

// AddTask creates a new todo via the repository and adds it to the available list.
func (p *PlannerPresenter) AddTask(ctx context.Context, title string, priority int) error {
	todo := &repository.Todo{
		ID:        uuid.New(),
		Title:     title,
		Priority:  priority,
		CreatedAt: p.clock.Now(),
	}
	if err := p.todos.Insert(ctx, todo); err != nil {
		return fmt.Errorf("inserting task: %w", err)
	}
	p.tasks = append(p.tasks, todoToRow(todo))
	return nil
}

// === Estimation Management ===

// NextStep advances the wizard to the next step.
func (p *PlannerPresenter) NextStep(ctx context.Context) error {
	switch p.step {
	case StepTaskSelect:
		return p.nextFromTaskSelect(ctx)
	case StepEstimates:
		p.step = StepPriority
		return nil
	case StepPriority:
		return p.nextFromPriority(ctx)
	case StepSchedule:
		return fmt.Errorf("cannot advance from schedule step: use SelectSchedule")
	default:
		return fmt.Errorf("cannot advance from step %d", p.step)
	}
}

// PreviousStep moves the wizard back one step.
func (p *PlannerPresenter) PreviousStep() {
	switch p.step {
	case StepTaskSelect:
		p.step = StepIdle
	case StepEstimates:
		p.step = StepTaskSelect
	case StepPriority:
		p.step = StepEstimates
	case StepSchedule:
		p.step = StepPriority
	}
}

// Estimates returns the current task estimate rows.
func (p *PlannerPresenter) Estimates() []TaskEstimateRow {
	result := make([]TaskEstimateRow, len(p.estimates))
	copy(result, p.estimates)
	return result
}

// OverrideEstimate sets a user override for a task's pomodoro estimate.
func (p *PlannerPresenter) OverrideEstimate(todoID uuid.UUID, pomos int) {
	if estimateIndex := p.findEstimateIndex(todoID); estimateIndex >= 0 {
		override := pomos
		p.estimates[estimateIndex].UserOverride = &override
		p.estimates[estimateIndex].EffectivePomos = pomos
	}
}

// EstimateSummary returns the summary of pomodoro estimates.
func (p *PlannerPresenter) EstimateSummary() EstimateSummary {
	total := 0
	for _, e := range p.estimates {
		total += e.EffectivePomos
	}
	// Default workday: 09:00-17:00 = 8 hours = 480 minutes
	// Default pomodoro: 25 minutes
	available := 480 / 25 // = 19
	return EstimateSummary{
		TotalPomos:      total,
		AvailableBlocks: available,
		Overloaded:      total > available,
	}
}

// ReorderTask moves a task estimate from one index to another.
func (p *PlannerPresenter) ReorderTask(from, to int) {
	if !p.isValidEstimateIndex(from) || !p.isValidEstimateIndex(to) {
		return
	}
	item := p.estimates[from]
	// Remove from old position
	p.estimates = append(p.estimates[:from], p.estimates[from+1:]...)
	// Insert at new position
	p.estimates = append(p.estimates[:to], append([]TaskEstimateRow{item}, p.estimates[to:]...)...)
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
	todoID := p.findTodoIDByTaskName(block.TaskName)
	if todoID == uuid.Nil {
		return nil // Task not found, silently continue
	}

	if err := p.todos.Complete(ctx, todoID, p.clock.Now()); err != nil {
		return fmt.Errorf("completing task: %w", err)
	}
	return nil
}

// AbandonPlan deletes the active schedule and returns to StepIdle.
func (p *PlannerPresenter) AbandonPlan(ctx context.Context) error {
	if err := p.schedRepo.Delete(ctx, p.activeScheduleID); err != nil {
		return fmt.Errorf("deleting schedule: %w", err)
	}
	p.step = StepIdle
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
	return nil
}

// === Internal Helpers ===

// Task index helpers
func (p *PlannerPresenter) findTaskIndex(id uuid.UUID) int {
	for i := range p.tasks {
		if p.tasks[i].ID == id {
			return i
		}
	}
	return -1
}

func (p *PlannerPresenter) findEstimateIndex(todoID uuid.UUID) int {
	for i := range p.estimates {
		if p.estimates[i].TodoID == todoID {
			return i
		}
	}
	return -1
}

func (p *PlannerPresenter) isValidEstimateIndex(index int) bool {
	return index >= 0 && index < len(p.estimates)
}

// Date helpers
func (p *PlannerPresenter) todayDate() time.Time {
	now := p.clock.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// Wizard step transition helpers

func (p *PlannerPresenter) nextFromTaskSelect(ctx context.Context) error {
	selected := p.selectedTasks()
	if len(selected) == 0 {
		return fmt.Errorf("no tasks selected")
	}

	estimates := make([]TaskEstimateRow, 0, len(selected))
	for _, t := range selected {
		desc := p.descriptions[t.ID]
		pomos, err := p.estimator.EstimatePomodoros(ctx, t.Title, desc)
		if err != nil {
			pomos = 1 // fallback
		}
		estimates = append(estimates, TaskEstimateRow{
			TodoID:         t.ID,
			Title:          t.Title,
			EstimatedPomos: pomos,
			EffectivePomos: pomos,
		})
	}

	p.estimates = estimates
	p.step = StepEstimates
	return nil
}

func (p *PlannerPresenter) nextFromPriority(ctx context.Context) error {
	date := p.todayDate()

	// Fetch calendar events, gracefully handling failures
	events := p.fetchCalendarEventsOrEmpty(ctx, date)

	// Build task estimates for the generator
	tasks := p.buildTaskEstimates()

	focus, recovery, err := p.generator.GenerateSchedules(ctx, tasks, events, date)
	if err != nil {
		return fmt.Errorf("generating schedules: %w", err)
	}

	p.focusSchedule = focus
	p.recoverySchedule = recovery
	p.step = StepSchedule
	return nil
}

func (p *PlannerPresenter) selectedTasks() []TodoRow {
	var selected []TodoRow
	for _, t := range p.tasks {
		if t.Selected {
			selected = append(selected, t)
		}
	}
	return selected
}

// Additional helper methods
func (p *PlannerPresenter) isValidActiveIndex() bool {
	return p.activeIndex >= 0 && p.activeIndex < len(p.activeBlocks)
}

func (p *PlannerPresenter) findTodoIDByTaskName(taskName string) uuid.UUID {
	for _, e := range p.estimates {
		if e.Title == taskName {
			return e.TodoID
		}
	}
	return uuid.Nil
}

func (p *PlannerPresenter) fetchCalendarEventsOrEmpty(ctx context.Context, date time.Time) []calendar.CalendarEvent {
	events, err := p.cal.FetchEvents(ctx, date)
	if err != nil {
		return []calendar.CalendarEvent{}
	}
	return events
}

func (p *PlannerPresenter) buildTaskEstimates() []planner.TaskEstimate {
	tasks := make([]planner.TaskEstimate, len(p.estimates))
	for i, e := range p.estimates {
		tasks[i] = planner.TaskEstimate{
			TodoID:         e.TodoID,
			Title:          e.Title,
			EstimatedPomos: e.EstimatedPomos,
			UserOverride:   e.UserOverride,
		}
	}
	return tasks
}

// Model conversion helpers

func todoToRow(t *repository.Todo) TodoRow {
	return TodoRow{
		ID:         t.ID,
		Title:      t.Title,
		Priority:   t.Priority,
		DueDate:    t.DueDate,
		Categories: t.Categories,
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
