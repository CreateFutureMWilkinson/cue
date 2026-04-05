# Feature 017: Focus Rail

**Phase:** Phase-1-Feature-017
**Status:** Planned
**Packages:** `internal/ui/`, `internal/ui/presenter/`

---

## Overview

Persistent left column (10% width) showing a countdown timer ring placeholder, current task name, and navigation buttons. The focus rail is always visible regardless of center area state. Button visibility changes based on application state: Plan/Back buttons are mutually exclusive, Review button only appears when notifications are expanded. The timer ring and task name are only visible when an active plan exists. Full timer mechanics are implemented in Feature 022 (Planner UI); this feature creates the widget shell and visibility logic.

## Design Decisions

- **Timer ring is a placeholder in this feature** — the visual ring widget is created with static geometry (45 lines at 8° intervals, correct tiers) but no countdown logic. Full timer state management (flashing, depletion, block tracking) comes with Feature 022.
- **Visibility rules driven by presenter state** — `FocusRailPresenter` tracks which controls are visible based on: active plan existence, center view state, and notification expansion state. The view layer binds to these signals.
- **Back and Plan are mutually exclusive** — Plan button navigates to Plan view, Back button returns to character view. Only one is visible at a time, determined by the current center view.
- **Review button appears only in expanded notification state** — ties into the notification panel's expanded/collapsed toggle (Feature 018).
- **Timer ring scales to fit rail width** — approximately 40–50px radius at 1200w window. The `timer-ring-radius` design token (120px) applies to the full-size spec; this version scales proportionally.

## API

### FocusRailPresenter

```go
type FocusRailPresenter struct {
    viewRouter    *CenterViewRouter
    onStateChange func()
}

func NewFocusRailPresenter(
    viewRouter *CenterViewRouter,
) (*FocusRailPresenter, error)

// Visibility
func (p *FocusRailPresenter) TimerVisible() bool          // true when active plan exists
func (p *FocusRailPresenter) TaskNameVisible() bool        // true when active plan exists
func (p *FocusRailPresenter) DoneVisible() bool            // true when active plan exists
func (p *FocusRailPresenter) BackVisible() bool            // true when center is Plan or Wizard
func (p *FocusRailPresenter) PlanVisible() bool            // true when center is Character
func (p *FocusRailPresenter) ReviewVisible() bool          // true when notifications expanded

// State
func (p *FocusRailPresenter) CurrentTaskName() string
func (p *FocusRailPresenter) SetActivePlan(active bool)
func (p *FocusRailPresenter) SetNotificationsExpanded(expanded bool)
func (p *FocusRailPresenter) SetOnStateChange(fn func())

// Actions
func (p *FocusRailPresenter) OnDone()                      // marks current task done
func (p *FocusRailPresenter) OnBack()                      // navigates to character view
func (p *FocusRailPresenter) OnPlan()                      // navigates to plan view
func (p *FocusRailPresenter) OnReview()                    // opens feedback review
```

### Timer Ring Widget (Static Shell)

```go
type TimerRingWidget struct {
    widget.BaseWidget
    radius      float32
    segments    int          // 45
    lineShort   float32      // scaled from design token
}

func NewTimerRingWidget() *TimerRingWidget
func (w *TimerRingWidget) CreateRenderer() fyne.WidgetRenderer
```

## Layout

```
┌──────┐
│      │
│  ◯   │  ← Timer ring (static, 45 lines)
│      │
│      │
│ Task │  ← Current task name (word-wrapped)
│ name │
│      │
│[Done]│  ← Marks task done
│      │
│      │
│[Back]│  ← Returns to character view (mutually exclusive with Plan)
│[Plan]│  ← Opens Plan view (mutually exclusive with Back)
│[Review]│ ← Only when notifications expanded
└──────┘
```

### Control Visibility

| Control | Visible when |
|---|---|
| Timer ring | Active plan exists |
| Task name | Active plan exists |
| Done | Active plan exists |
| Back | Center area showing Plan view or Wizard |
| Plan | Center area showing Character |
| Review | Notifications expanded |

## Error Handling

| Scenario | Behavior |
|---|---|
| No active plan | Timer ring, task name, and Done hidden; only Plan button visible |
| View router nil | Constructor returns validation error |

## Integration Points

- **Three-Column Layout (Feature 016):** Focus rail occupies the left 10% column created by the layout.
- **CenterViewRouter (Feature 016):** Back/Plan button visibility driven by current center view state.
- **Notification Panel (Feature 018):** Review button visibility driven by notification expanded state.
- **Planner UI (Feature 022):** Timer ring populated with countdown logic; task name driven by planner state; Done button wired to task completion.

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `presenter` | `FocusRailPresenterSuite` | Constructor validation, default visibility (no plan: timer/task/done hidden, plan visible), with active plan (timer/task/done visible), back/plan mutual exclusivity, review visible only when expanded, state change callback fires, action delegates (done/back/plan/review) |
| `ui` | `TimerRingWidgetSuite` | Creates 45 line objects, cardinal lines 3x length, diagonal lines 2x length, standard lines 1x length, ring scales with size |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| FocusRailPresenter | RED | Test Designer | — | — | — |
| FocusRailPresenter | GREEN | Implementer | — | — | — |
| FocusRailPresenter | REFACTOR | Refactorer | — | — | — |
| TimerRingWidget | RED | Test Designer | — | — | — |
| TimerRingWidget | GREEN | Implementer | — | — | — |
| TimerRingWidget | REFACTOR | Refactorer | — | — | — |
