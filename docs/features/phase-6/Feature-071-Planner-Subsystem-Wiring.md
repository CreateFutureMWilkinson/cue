# Feature 071 — Planner Subsystem Not Wired in main.go

**Phase:** Phase-6-Feature-071
**Type:** Bugfix
**Severity:** Critical
**Status:** Done
**Packages:** `cmd/cue/`, `internal/ui/`, `internal/service/decisionengine/`, `internal/service/calendar/`, `internal/alert/`, `internal/ui/presenter/`
**Depends on:** —

---

## Overview

The entire Phase 2 day planner subsystem was implemented and tested (Features 015-023) but never instantiated in the composition root (`cmd/cue/main.go`). The `NewMainWindow` call passed `nil, nil, nil` for `plannerVM`, `timerVM`, and `wizardVM`, leaving the Plan view, Wizard view, countdown timer, AppBinder, and todo list completely inert.

## Root Cause

`main.go` created all Phase 1 dependencies (orchestrator, router, notifications, feedback, alerts, character) but never created the Phase 2 dependencies:

- No `sqlite.NewSQLiteTodoRepository()` call
- No `sqlite.NewSQLiteScheduleRepository()` call
- No `sqlite.NewSQLiteCategoryRepository()` call
- No `calendar.NewProvider()` or equivalent
- No `planner.NewGenerator()` or `planner.NewEstimator()` calls
- No `presenter.NewPlannerPresenter()` call
- No `presenter.NewTimerPresenter()` call
- No `ui.NewTimerLoop()` call
- No `ui.NewAppBinder()` call
- No `planner.NewRealClock()` call

## Fix Description

Six behaviors were implemented across multiple packages to bridge the gap:

### 1. PlannerPresenter.SelectedCount()

Added missing method to satisfy the `WizardViewModel` interface. Counts the number of selected tasks in the planner's task list by iterating the task slice and tallying those marked as selected.

### 2. OllamaClient.Generate()

Added raw prompt-to-response method satisfying the `planner.OllamaGenerator` interface. The planner engine needs to call Ollama with freeform prompts (e.g., schedule generation) rather than the structured scoring prompt used by `Score()`. Refactored `Score()` and `Generate()` to share a common `sendRequest()` helper that handles HTTP transport, timeout, and error mapping.

### 3. NoopCalendarProvider

New noop implementation in the `calendar` package for when no calendar accounts are configured. Returns an empty event slice and `nil` error, allowing the planner to function without calendar integration. Used as the default when `len(calendarAccounts) == 0`.

### 4. TimerAlerterAdapter

New adapter in the `alert` package bridging `TimerAlertService.PlayTimerEnd()` to `presenter.TimerAlerter.PlayBlockComplete()`. The timer presenter expects a `TimerAlerter` interface with `PlayBlockComplete()`, while the existing alert infrastructure exposes `PlayTimerEnd()`. The adapter provides the glue without modifying either side.

### 5. MainWindow.PlannerViewRef() / WizardViewRef()

New accessor methods on `MainWindow` exposing the `PlannerView` and `WizardView` as `RefreshableView` interfaces. Required by `AppBinder` to trigger view refreshes when planner state changes (step transitions, plan load, timer tick).

### 6. main.go Wiring

The composition root now creates all Phase 2 dependencies in order:

1. `TodoRepository`, `CategoryRepository`, `ScheduleRepository` from the existing SQLite DB connection
2. `CalendarProvider` — noop when no calendar accounts configured, ICS adapter otherwise
3. `planner.NewGenerator()`, `planner.NewEstimator()`, `planner.NewRealClock()`
4. `presenter.NewPlannerPresenter()` with all dependencies
5. `presenter.NewTimerPresenter()` with clock and timer alerter adapter
6. Real presenters passed to `NewMainWindow` instead of `nil`
7. `ui.NewAppBinder()` created after window, with planner presenter, focus rail, wizard view, planner view, and view router; `Bind()` and `AutoLoad()` called
8. `ui.NewTimerLoop()` created and started

A `#nosec G704` annotation was added for the calendar HTTP client's `http.Get` call, which gosec flags but is safe here since the URL comes from encrypted config storage.

## Design Decisions

### Adapter pattern for TimerAlerter

Rather than modifying the existing `TimerAlertService` interface to match what the presenter expects, a thin adapter was created. This preserves the alert package's API stability and keeps the coupling between packages at the interface level.

### Noop calendar provider

A dedicated noop type was preferred over `nil` checks throughout the planner engine. The engine can unconditionally call `provider.Events()` without guarding against nil, simplifying the hot path.

### sendRequest() extraction in OllamaClient

The `Score()` and `Generate()` methods shared identical HTTP transport logic (request construction, timeout, response reading, error wrapping). Extracting `sendRequest()` eliminated duplication and made both methods easier to test independently.

## Files Changed

| File | Change |
|---|---|
| `internal/ui/presenter/planner_presenter.go` | Added `SelectedCount()` method |
| `internal/service/decisionengine/ollama_client.go` | Added `Generate()` method, extracted `sendRequest()` helper |
| `internal/service/calendar/noop.go` | New file — `NoopCalendarProvider` |
| `internal/alert/timer_alerter_adapter.go` | New file — `TimerAlerterAdapter` |
| `internal/ui/window.go` | Added `PlannerViewRef()`, `WizardViewRef()` accessors |
| `cmd/cue/main.go` | Wired all Phase 2 planner dependencies |

## Error Handling

- Calendar provider errors during startup are fatal (fail-fast)
- Timer alerter adapter delegates errors to the underlying `TimerAlertService`
- Repository construction errors from `NewSQLiteTodoRepository`, `NewSQLiteCategoryRepository`, `NewSQLiteScheduleRepository` are fatal

## Integration Points

- **Planner engine** (`internal/service/planner/`) — receives repositories, calendar provider, generator, estimator, clock
- **Presenters** (`internal/ui/presenter/`) — `PlannerPresenter` and `TimerPresenter` drive the UI
- **AppBinder** (`internal/ui/`) — connects presenter state changes to view refreshes
- **TimerLoop** (`internal/ui/`) — ticks the countdown timer at 1 Hz
- **MainWindow** (`internal/ui/`) — receives real presenters instead of nil

## Test Coverage

| Test | Package | Behavior |
|---|---|---|
| `TestPlannerPresenter_SelectedCount` | `presenter_test` | Counts selected tasks correctly |
| `TestOllamaClient_Generate` | `decisionengine_test` | Raw prompt generates response via Ollama |
| `TestOllamaClient_sendRequest` | `decisionengine_test` | Shared HTTP helper handles errors |
| `TestNoopCalendarProvider` | `calendar_test` | Returns empty events, nil error |
| `TestTimerAlerterAdapter` | `alert_test` | Delegates PlayBlockComplete to PlayTimerEnd |
| `TestMainWindow_PlannerViewRef` | `ui_test` | Returns non-nil RefreshableView when planner wired |
| `TestMainWindow_WizardViewRef` | `ui_test` | Returns non-nil RefreshableView when wizard wired |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED (SelectedCount) | Test Designer | ~35s | ~27,000 | d9d72ba |
| GREEN (SelectedCount) | Implementer | ~30s | ~25,000 | d2f116d |
| RED (Generate) | Test Designer | ~40s | ~28,000 | 2223ce3 |
| GREEN (Generate) | Implementer | ~45s | ~30,000 | f655e7c |
| REFACTOR (sendRequest) | Refactorer | ~50s | ~32,000 | b9275a8 |
| RED (NoopCalendar) | Test Designer | ~30s | ~25,000 | 6a0da4c |
| GREEN (NoopCalendar) | Implementer | ~35s | ~27,000 | 55cc15d |
| RED (TimerAlerter) | Test Designer | ~30s | ~25,000 | 193c88a |
| GREEN (TimerAlerter) | Implementer | ~35s | ~28,000 | 8081312 |
| RED (ViewRefs) | Test Designer | ~40s | ~29,000 | dc23f45 |
| GREEN (ViewRefs) | Implementer | ~45s | ~31,000 | bb5d4ad |
| GREEN (main wiring) | Implementer | ~60s | ~35,000 | 69b8410 |
| REFACTOR (nosec) | orchestrator | ~30s | ~25,000 | 49d3da9 |
