# Feature 106: API Client SDK

**Phase:** Phase-9-Feature-106
**Status:** Planning
**Depends on:** Features 096-105 (server APIs)
**Package:** `internal/client/`

---

## Overview

Create an API client SDK package (`internal/client/`) that provides HTTP/WebSocket-backed implementations of every interface the Fyne UI presenters depend on. This enables the Fyne app to be re-wired as a thin client connecting to `cue-server` (Feature 107) without modifying any presenter or view code.

The existing presenters already depend on clean interfaces (defined in `internal/ui/presenter/interfaces.go`, `internal/repository/`, and `internal/service/`). This feature creates adapter types that implement those same interfaces by making REST calls and WebSocket subscriptions to the server's `/api/v1/` endpoints.

## Design Decisions

### Package Structure

**Decision: Single `internal/client/` package, one file per adapter domain.**

```
internal/client/
  client.go            # APIClient base (HTTP client, server URL, auth token, helpers)
  messages.go          # MessageQuerier + MessageUpdater
  activity.go          # ActivitySource (WebSocket)
  feedback.go          # BufferReviewer
  planner.go           # TodoQuerier, CategoryQuerier, ScheduleGenerator, TaskEstimator
  schedule.go          # ScheduleRepository
  service_config.go    # ServiceConfigRepository
  rules.go             # RoutingRuleRepository
  queue.go             # QueueRepository
  watchers.go          # WatcherRemover + WatcherFactory
  settings.go          # VolumeController
```

Rationale: Mirrors the server handler file layout. Each file contains one or two closely related adapter types. All adapters share the base `APIClient` for HTTP transport.

### Transport Layer

**Decision: Thin wrappers over `net/http` + `github.com/coder/websocket`.**

No generated client code or third-party REST client libraries. The API surface is small enough that hand-written adapters are simpler and easier to maintain. Each adapter method is 10-20 lines: build URL, marshal request, call HTTP, unmarshal response.

Helper methods on APIClient:
- `get(ctx, path) ([]byte, error)`
- `post(ctx, path, body) ([]byte, error)`
- `put(ctx, path, body) ([]byte, error)`
- `delete(ctx, path) error`

These handle auth headers, content-type, status code checking, and structured error responses.

### JSON Field Naming

**Decision: `camelCase` per Feature 096 ADR.** Adapter code must marshal/unmarshal using `camelCase` JSON tags matching the server's response format. Domain types (`repository.Message`, etc.) use Go-style fields internally. The adapter layer handles the translation.

Two approaches:
- **Option A:** Use `json:"camelCase"` struct tags on DTO types in `internal/client/`, convert to/from domain types.
- **Option B:** Use a JSON naming convention library to auto-convert.

**Decision: Option A — explicit DTO types.** Clearer, no magic, easy to test. Each adapter file defines its own request/response DTOs.

### Authentication

**Decision: Bearer token from TOFU pairing (Feature 096).** The APIClient stores a token string, sent as `Authorization: Bearer <token>` on every request. Token acquisition (first-client auto-issue, subsequent-client pairing) is handled by a separate `Authenticate()` method on APIClient, called during startup.

For initial development/testing, the token can be empty (server in dev mode accepts unauthenticated requests).

### Error Handling

**Decision: Map HTTP status codes to Go errors.**

| HTTP Status | Go Error |
|---|---|
| 200-299 | nil |
| 404 | Domain-specific "not found" (e.g., message not found) |
| 409 | Conflict (concurrent update) |
| 422 | Validation error (bad input) |
| 500+ | `fmt.Errorf("server error: %s", body.Error.Message)` |
| Network error | Wrapped with context: `fmt.Errorf("GET /api/v1/...: %w", err)` |

### WebSocket Reconnection

**Decision: Automatic reconnect with exponential backoff.**

The `ActivitySource` adapter maintains a persistent WebSocket connection to `/api/v1/ws/events`. On disconnect:
1. Close the old connection
2. Backoff: 1s, 2s, 4s, 8s, ... capped at 30s
3. Reconnect and resume delivering events to the channel
4. Reset backoff on successful reconnect

The `Events() <-chan ActivityEvent` channel is never closed — it just stops receiving during disconnects. Presenters already handle empty/stale data gracefully.

## Interface to Adapter Mapping

### Server-backed adapters (HTTP)

| Interface | Defined In | Adapter Type | Endpoints |
|---|---|---|---|
| `presenter.MessageQuerier` | `presenter/interfaces.go` | `MessageAdapter` | `GET /notifications`, `GET /messages` |
| `presenter.MessageUpdater` | `presenter/interfaces.go` | `MessageAdapter` | `POST /notifications/{id}/resolve`, `PUT /messages/{id}` |
| `presenter.BufferReviewer` | `presenter/interfaces.go` | `FeedbackAdapter` | `GET/POST/DELETE /buffer/*` |
| `presenter.VolumeController` | `presenter/interfaces.go` | `SettingsAdapter` | `PUT /settings/volume` |
| `presenter.TodoQuerier` | `presenter/planner_presenter.go` | `PlannerAdapter` | `GET/POST/PUT /planner/todos` |
| `presenter.CategoryQuerier` | `presenter/planner_presenter.go` | `PlannerAdapter` | `GET /planner/categories` |
| `presenter.ScheduleGenerator` | `presenter/planner_presenter.go` | `PlannerAdapter` | `POST /planner/generate` |
| `planner.TaskEstimator` | `service/planner/planner.go` | `PlannerAdapter` | `POST /planner/estimate` |
| `calendar.CalendarProvider` | `service/calendar/calendar.go` | `PlannerAdapter` | `GET /planner/calendar` |
| `repository.ScheduleRepository` | `repository/schedule.go` | `ScheduleAdapter` | `GET/POST/DELETE /planner/schedules` |
| `repository.ServiceConfigRepository` | `repository/service_config.go` | `ServiceConfigAdapter` | `CRUD /services/{type}/{id}` |
| `presenter.WatcherRemover` | `presenter/service_settings_presenter.go` | `WatcherAdapter` | `DELETE /watchers/{name}` |
| `presenter.WatcherFactory` | `presenter/service_settings_presenter.go` | `WatcherAdapter` | `POST /watchers` |
| `repository.RoutingRuleRepository` | `repository/routing_rule.go` | `RulesAdapter` | `CRUD /rules/{id}` |
| `repository.QueueRepository` | `repository/queue.go` | `QueueAdapter` | `GET/POST /queue` |

### Server-backed adapters (WebSocket)

| Interface | Defined In | Adapter Type | Endpoint |
|---|---|---|---|
| `presenter.ActivitySource` | `presenter/interfaces.go` | `ActivityAdapter` | `WS /ws/events` |

### Client-side only (no adapter needed)

| Interface | Defined In | Reason |
|---|---|---|
| `planner.Clock` | `service/planner/planner.go` | Local `time.Now()` — no server round-trip |
| `character.Character` | `ui/character/character.go` | Fyne widget — animations are purely visual |
| `presenter.TimerAlerter` | `presenter/timer_presenter.go` | Local audio playback via beep/beeep |

## TDD Behaviors

| # | Behavior | Adapter | Test Strategy |
|---|---|---|---|
| 1 | APIClient creation + health check | `client.go` | `httptest.NewServer` returning 200/503 |
| 2 | MessageQuerier.QueryByStatus | `messages.go` | Mock server returns JSON message list |
| 3 | MessageUpdater.Update | `messages.go` | Mock server accepts POST, verify request body |
| 4 | ActivitySource.Events (WebSocket) | `activity.go` | `httptest.NewServer` with WS upgrade, push events |
| 5 | BufferReviewer (all 4 methods) | `feedback.go` | Mock server for each endpoint |
| 6 | TodoQuerier + CategoryQuerier | `planner.go` | Mock CRUD responses |
| 7 | ScheduleGenerator + TaskEstimator | `planner.go` | Mock POST compute responses |
| 8 | ScheduleRepository (Save/Load/Delete) | `schedule.go` | Mock CRUD responses |
| 9 | ServiceConfigRepository (full CRUD) | `service_config.go` | Mock per-type CRUD |
| 10 | WatcherRemover + WatcherFactory | `watchers.go` | Mock POST/DELETE |
| 11 | RoutingRuleRepository + QueueRepository | `rules.go`, `queue.go` | Mock CRUD + depth query |
| 12 | VolumeController.SetVolume | `settings.go` | Mock PUT |

## Dependencies

- `github.com/coder/websocket` — WebSocket client (same library server uses; chosen in Feature 096 ADR)

## Test Coverage

All adapter methods tested against `httptest.NewServer` mock servers. Tests verify:
- Correct HTTP method and path
- Request body marshaling (camelCase JSON)
- Response unmarshaling to domain types
- Error handling for 4xx/5xx responses
- WebSocket connection lifecycle and event delivery
- Reconnection behavior on disconnect

Target: >= 80% coverage on `internal/client/`.
