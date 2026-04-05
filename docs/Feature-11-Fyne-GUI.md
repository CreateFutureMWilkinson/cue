# Feature 11: Fyne GUI

**Phase:** Phase-1-Feature-11
**Status:** Done
**Packages:** `internal/ui/`, `internal/ui/presenter/`, `cmd/cue/`

---

## Overview

Desktop GUI using Fyne v2 with a presenter/view architecture for testability. Three-pane layout: notification queue (NOTIFIED messages, newest-first), activity log (real-time system events), and feedback buffer review (buffered messages with 0-10 rating). Includes `cmd/cue/main.go` as the composition root wiring all Phase 1 components together. GUIConfig updated from web-oriented Host/Port to Fyne-relevant WindowWidth/WindowHeight.

## Design Decisions

### Presenter/View Separation

All business logic lives in `internal/ui/presenter/` with zero Fyne imports. Presenters are pure Go, fully testable at 80%+ coverage. The view layer (`internal/ui/`) is thin Fyne widget wiring with no unit tests. This MVP pattern is the standard approach for testing GUI applications — presenters are the unit under test, views are manual-test territory.

### Consumer-Side Interfaces

The presenter package defines its own narrow interfaces (`MessageQuerier`, `MessageUpdater`, `BufferReviewer`, `ActivitySource`, `Alerter`) rather than importing concrete types from other packages. This follows the existing codebase convention (e.g., buffer package defines its own `MessageRepository` subset).

### Local ActivityEvent Type

`presenter.ActivityEvent` mirrors `orchestrator.ActivityEvent` to avoid a direct import dependency. The composition root bridges the two via a simple goroutine adapter. This keeps the dependency graph acyclic.

### Polling vs Push for Notifications

The notification pane refreshes on a 30-second `time.Ticker`. This is simpler and more robust than push-based updates for Phase 1. The activity log is push-based via the `OnUpdate` callback since events arrive asynchronously.

### Optional Alerter

`AppPresenter` accepts a nil `Alerter` (audio disabled), matching the pattern used throughout the codebase for optional dependencies.

### GUIConfig Change

Replaced `Host`/`Port` (web-server oriented) with `WindowWidth`/`WindowHeight` (Fyne desktop). Defaults: 1200x800.

## API

### Presenters

```go
// NotificationPresenter — notification queue logic
func NewNotificationPresenter(querier MessageQuerier, updater MessageUpdater) (*NotificationPresenter, error)
func (p *NotificationPresenter) Refresh(ctx context.Context) error
func (p *NotificationPresenter) Messages() []NotificationRow
func (p *NotificationPresenter) Select(index int) (*NotificationDetail, error)
func (p *NotificationPresenter) Resolve(ctx context.Context, id uuid.UUID) error

// ActivityPresenter — activity log event consumption
func NewActivityPresenter(source ActivitySource, maxEntries int) (*ActivityPresenter, error)
func (p *ActivityPresenter) Start(ctx context.Context)
func (p *ActivityPresenter) Entries() []ActivityEntry
func (p *ActivityPresenter) Stop()
func (p *ActivityPresenter) SetOnUpdate(fn func())

// FeedbackPresenter — feedback buffer review workflow
func NewFeedbackPresenter(reviewer BufferReviewer) (*FeedbackPresenter, error)
func (p *FeedbackPresenter) Load(ctx context.Context) error
func (p *FeedbackPresenter) Current() *FeedbackItem
func (p *FeedbackPresenter) Counter() string
func (p *FeedbackPresenter) SaveRating(ctx context.Context, rating int, feedback *string) error
func (p *FeedbackPresenter) Skip()
func (p *FeedbackPresenter) Delete(ctx context.Context) error
func (p *FeedbackPresenter) HasCurrent() bool

// AppPresenter — app lifecycle
func NewAppPresenter(notification, activity, feedback, alerter) (*AppPresenter, error)
func (p *AppPresenter) Start(ctx context.Context) error
func (p *AppPresenter) Shutdown(ctx context.Context) error
```

### View Layer

```go
// MainWindow — Fyne application window
func NewMainWindow(cfg GUIConfig, np, ap, fp, appP) *MainWindow
func (w *MainWindow) Run()
```

## Error Handling

| Error | Action |
|---|---|
| Repository query fails | Propagated to caller, notification list unchanged |
| Buffer reviewer fails | Propagated to caller, review state unchanged |
| Resolve updater fails | In-memory state rolled back, error returned |
| Activity source closes | Goroutine exits cleanly |
| Alerter returns error | Propagated but non-fatal (logged by orchestrator) |
| Nil alerter | All alert calls safely skipped |

## Integration Points

- **Repository** (`internal/repository/`) — `MessageQuerier` and `MessageUpdater` for notification pane
- **Buffer** (`internal/service/buffer/`) — `BufferReviewer` for feedback review pane
- **Orchestrator** (`internal/service/orchestrator/`) — `ActivityEvent` channel bridged to presenter
- **Alert** (`internal/alert/`) — `Alerter` for startup/shutdown sounds
- **Config** (`internal/config/`) — `GUIConfig.WindowWidth/WindowHeight` for window sizing

## Test Coverage Summary

55 tests total across 4 presenter suites:
- **NotificationPresenter** (15 tests): constructor validation, refresh/sort, truncation (15ch/80ch), select/expand, resolve with rollback, error propagation
- **ActivityPresenter** (11 tests): constructor validation, event consumption, newest-first ordering, OnUpdate callback, stop, ring buffer cap, error flag, timestamp
- **FeedbackPresenter** (18 tests): constructor validation, load, current/counter, save rating, skip, delete, boundary cases, counter updates
- **AppPresenter** (11 tests): constructor validation, start (alerts, activity start, notification refresh), shutdown (alerts, activity stop), nil alerter safety

## TDD Agent Stats

### Sub-feature 11a: Notification Presenter

| TDD Phase | Agent | Commit |
|---|---|---|
| RED | Test Designer | 9c793d4 |
| GREEN | Implementer | a88a0bb |
| REFACTOR | Refactorer | fe817ad |

### Sub-feature 11b: Activity Presenter

| TDD Phase | Agent | Commit |
|---|---|---|
| RED | Test Designer | 8440c09 |
| GREEN | Implementer | 57f65a4 |
| REFACTOR | Refactorer | cca74df |

### Sub-feature 11c: Feedback Presenter

| TDD Phase | Agent | Commit |
|---|---|---|
| RED | Test Designer | 3f7a811 |
| GREEN | Implementer | 332ce24 |
| REFACTOR | Refactorer | 6e9aa02 |

### Sub-feature 11d: App Presenter + GUIConfig + Views + main.go

| TDD Phase | Agent | Commit |
|---|---|---|
| RED | Test Designer | 77ec1dd |
| GREEN | Implementer | 6d17653 |
| REFACTOR | Refactorer | 9b48440 |
