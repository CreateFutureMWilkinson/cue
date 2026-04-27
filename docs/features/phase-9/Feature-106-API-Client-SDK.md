# Feature 106: API Client SDK

**Phase:** Phase-9-Feature-106
**Status:** Planning
**Depends on:** Features 096-104 (server APIs), Feature 108 (TOFU pairing)
**Package:** `pkg/client/`

---

## Overview

Create an API client SDK package (`pkg/client/`) that provides HTTP/WebSocket-backed types for every resource exposed by `cue-server`. This enables the Fyne app to be re-wired as a thin client (Feature 107) and supports alternative UIs (web frontend, TUI, mobile app, CLI scripts) connecting to the same server.

The SDK mirrors the server's API surface, not the Fyne presenter interfaces. Feature 107 is responsible for composing SDK types to satisfy presenter dependencies. This keeps the SDK decoupled from any specific UI framework.

The package lives under `pkg/client/` so it is importable by external consumers within or outside this module. If the SDK later needs to become a standalone module (`github.com/CreateFutureMWilkinson/cue-sdk`), extraction is straightforward.

## Design Decisions

### 1. Package Structure

**Decision: `pkg/client/` package, one file per adapter domain. Grouped interfaces per domain.**

```
pkg/client/
  client.go            # APIClient base (HTTP client, server URL, token, helpers)
  errors.go            # Structured error types (APIError, error code constants)
  messages.go          # MessageClient interface + adapter
  activity.go          # ActivityClient (WebSocket event stream)
  feedback.go          # FeedbackClient interface + adapter
  planner.go           # PlannerClient interface + adapter (todos, categories, schedule generation)
  schedule.go          # ScheduleClient interface + adapter
  service_config.go    # ServiceConfigClient interface + adapter (Slack, Email, Calendar accounts)
  rules.go             # RulesClient interface + adapter
  queue.go             # QueueClient interface + adapter
  timer.go             # TimerClient interface + adapter
  auth.go              # AuthClient interface + adapter (pairing, token management)
```

Rationale: Mirrors the server handler file layout. Each file contains one interface and its concrete adapter type. All adapters share the base `APIClient` for HTTP transport.

### 2. Transport Layer

**Decision: Thin wrappers over `net/http` + `github.com/coder/websocket`.**

No generated client code or third-party REST client libraries. The API surface is small enough that hand-written adapters are simpler and easier to maintain. Each adapter method is 10-20 lines: build URL, marshal request, call HTTP, unmarshal response.

Helper methods on APIClient:
- `get(ctx, path) ([]byte, error)`
- `post(ctx, path, body) ([]byte, error)`
- `put(ctx, path, body) ([]byte, error)`
- `patch(ctx, path, body) ([]byte, error)`
- `delete(ctx, path) error`

These handle auth headers (`Authorization: Bearer <token>`), content-type, status code checking, and structured error responses.

### 3. SDK Mirrors Server API, Not Presenter Interfaces

**Decision: Adapters align with the server's handler interfaces, not the Fyne presenter interfaces.**

The server has handler-level interfaces like `TodoServicer`, `RulesManager`, `ServiceManager`, etc. The SDK provides matching client types (`TodoClient`, `RulesClient`, `ServiceConfigClient`) that map 1:1 to server endpoints.

Feature 107 (Fyne Client Re-wire) composes these SDK types to satisfy presenter interface requirements. For example, the presenter's `BufferReviewer` combines methods from `FeedbackClient` and `MessageClient`.

This design keeps the SDK framework-agnostic and usable by any consumer, not just Fyne.

### 4. Grouped Interfaces Per Domain

**Decision: The SDK defines one interface per domain (e.g., `MessageClient`, `RulesClient`).**

Each domain file exports an interface and a concrete struct implementing it. Consumers get ready-made interfaces for mocking in tests. Nothing prevents consumers from defining narrower interfaces per Go convention.

### 5. JSON Field Naming

**Decision: `camelCase` per Feature 096 ADR, explicit DTO types.**

Adapter code marshals/unmarshals using `camelCase` JSON tags matching the server's response format. Each adapter file defines its own request/response DTO types. Domain types from `internal/repository/` are not reused directly — the SDK has its own types to avoid coupling consumers to internal packages.

### 6. Structured Error Types

**Decision: `*APIError` type with error code and message, preserving server error semantics.**

```go
type APIError struct {
    StatusCode int    // HTTP status code
    Code       string // Server error code (e.g., "NOT_FOUND", "VALIDATION_ERROR")
    Message    string // Human-readable error message
}

func (e *APIError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

Error code constants:
```go
const (
    ErrCodeNotFound        = "NOT_FOUND"
    ErrCodeConflict        = "CONFLICT"
    ErrCodeValidation      = "VALIDATION_ERROR"
    ErrCodeServerError     = "SERVER_ERROR"
    ErrCodeTokenIssued     = "TOKEN_ISSUED"
    ErrCodeUnauthorized    = "UNAUTHORIZED"
)
```

Callers can type-assert `*APIError` to inspect the code programmatically:
```go
var apiErr *client.APIError
if errors.As(err, &apiErr) && apiErr.Code == client.ErrCodeNotFound {
    // handle not found
}
```

Network errors are wrapped with context: `fmt.Errorf("GET /api/v1/...: %w", err)`.

### 7. Authentication

**Decision: Bearer token from Feature 108 TOFU pairing.**

The `APIClient` stores a token string, sent as `Authorization: Bearer <token>` on every HTTP request and as a `?token=` query parameter on WebSocket upgrade.

Token acquisition uses the `AuthClient` interface:
- `Pair(ctx, label) (PairSession, error)` — initiates pairing, returns session with code and poll method
- `PollPairResult(ctx, requestID) (PairResult, error)` — polls for approval/denial
- `ListTokens(ctx) ([]TokenInfo, error)` — list connected devices
- `RevokeToken(ctx, id) error` — revoke a token
- `Logout(ctx) error` — revoke current session

For the first-client flow, the API helpers detect `TOKEN_ISSUED` in the 401 response, store the token, and retry automatically. This is handled internally by the `APIClient` — callers don't need to manage it.

### 8. WebSocket Event Stream

**Decision: Automatic reconnect with hardcoded exponential backoff. Manual replay for missed events.**

The `ActivityClient` adapter maintains a persistent WebSocket connection to `/api/v1/websocket/events?token=<token>`. On disconnect:
1. Close the old connection
2. Backoff: 1s, 2s, 4s, 8s, ... capped at 30s
3. Reconnect and resume delivering events to the channel
4. Reset backoff on successful reconnect

**No automatic replay.** When the connection drops and reconnects, there may be a gap in the event sequence. The `Events() <-chan EventEnvelope` channel exposes sequence numbers; consumers who care about gaps can call `Replay(ctx, sinceSeq)` to fetch missed events via `GET /api/v1/events?since={seq}`.

The `Events()` channel is never closed — it just stops receiving during disconnects. The `LastSeq() uint64` method returns the highest sequence number received.

### 9. VolumeController — Client-Local Only

**Decision: No SDK adapter for volume control.**

Audio playback and volume settings are client-local concerns. `VolumeController` stays in the Fyne app (or any other UI). There is no server endpoint for volume — Feature 105 (Settings API) was removed.

### 10. ActiveSchedule Convenience Wrapper

**Decision: `ScheduleClient` provides `ActiveSchedule(ctx)` alongside date-based methods.**

The server resolves `/api/v1/planner/active` to the current date's schedule. The SDK exposes this as a convenience method that callers can use instead of determining the active date themselves:

```go
type ScheduleClient interface {
    ActiveSchedule(ctx context.Context) (*Schedule, error)
    DeleteActiveSchedule(ctx context.Context) error
    GetSchedule(ctx context.Context, date time.Time) (*Schedule, error)
    PutSchedule(ctx context.Context, date time.Time, schedule *Schedule) error
    DeleteSchedule(ctx context.Context, date time.Time) error
}
```

## Interface to Endpoint Mapping

### HTTP Adapters

| SDK Interface | Adapter Type | Endpoints |
|---|---|---|
| `AuthClient` | `authAdapter` | `POST /auth/pair`, `GET /auth/pair/{id}`, `POST .../approve`, `POST .../deny`, `GET /auth/tokens`, `PUT /auth/tokens/{id}`, `DELETE /auth/tokens/{id}`, `POST /auth/logout` |
| `MessageClient` | `messageAdapter` | `GET /messages`, `GET /messages/{id}`, `GET /notifications`, `GET /notifications/{id}`, `POST /notifications/{id}/resolve`, `POST /notifications/{id}/dismiss` |
| `FeedbackClient` | `feedbackAdapter` | `GET /buffer`, `GET /buffer/{id}`, `GET /buffer/stats`, `POST /buffer/{id}/rate`, `DELETE /buffer/{id}` |
| `PlannerClient` | `plannerAdapter` | `GET/POST /tasks`, `GET/PUT/DELETE /tasks/{id}`, `POST /planner/generate` |
| `ScheduleClient` | `scheduleAdapter` | `GET/DELETE /planner/active`, `GET/PUT/DELETE /planner/{date}` |
| `ServiceConfigClient` | `serviceConfigAdapter` | `CRUD /services/slack/{id}`, `CRUD /services/email/{id}`, `CRUD /services/calendar/{id}`, `POST .../toggle`, `GET /services/status` |
| `RulesClient` | `rulesAdapter` | `GET/POST /rules`, `GET/PUT/PATCH/DELETE /rules/{id}` |
| `QueueClient` | `queueAdapter` | `GET /queue/depth` (if exposed), queue operations |
| `TimerClient` | `timerAdapter` | `GET /timer` |

All paths are prefixed with `/api/v1/`.

### WebSocket Adapter

| SDK Interface | Adapter Type | Endpoint |
|---|---|---|
| `ActivityClient` | `activityAdapter` | `WS /api/v1/websocket/events?token=<token>` |

### Client-Side Only (No Adapter Needed)

| Concern | Reason |
|---|---|
| Volume control | Local audio playback — no server round-trip |
| Character animations | Fyne widget — purely visual |
| Timer alerts (audio) | Local audio playback via beep/beeep |
| `time.Now()` clock | No server round-trip needed |

## SDK Types (Own Domain Types)

The SDK defines its own types rather than importing from `internal/`. This avoids coupling consumers to internal packages and ensures the SDK is self-contained.

Key types include:
- `Message`, `NotificationSummary`, `NotificationDetail`
- `BufferedMessage`, `BufferStats`
- `Task`, `Category`
- `Schedule`, `ScheduleOption`, `GenerateRequest`
- `SlackAccount`, `EmailAccount`, `CalendarAccount`, `ServiceStatus`
- `RoutingRule`
- `TimerState`
- `EventEnvelope`, `ActivityEvent`, `NotificationEvent`, `TimerTickEvent`, `PairingRequestEvent`
- `TokenInfo`, `PairSession`, `PairResult`
- `APIError`

## TDD Behaviors

| # | Behavior | Adapter | Test Strategy |
|---|---|---|---|
| 1 | APIClient creation + health check | `client.go` | `httptest.NewServer` returning 200/503 |
| 2 | Structured error parsing from server responses | `errors.go` | Mock server returning 4xx/5xx with error body |
| 3 | Auth header injection on all requests | `client.go` | Mock server verifying `Authorization` header |
| 4 | First-client auto-token flow (401 TOKEN_ISSUED → retry) | `auth.go` | Mock server issuing token on first request |
| 5 | Pairing initiation + polling | `auth.go` | Mock server with pair/poll endpoints |
| 6 | MessageClient.ListNotifications + GetNotification | `messages.go` | Mock server returns JSON |
| 7 | MessageClient.ResolveNotification + DismissNotification | `messages.go` | Mock server accepts POST |
| 8 | ActivityClient.Events (WebSocket connect + receive) | `activity.go` | `httptest.NewServer` with WS upgrade |
| 9 | ActivityClient reconnection with exponential backoff | `activity.go` | Mock server that disconnects then accepts |
| 10 | ActivityClient.Replay (manual event replay) | `activity.go` | Mock GET /events?since=N |
| 11 | FeedbackClient (list, get, stats, rate, delete) | `feedback.go` | Mock server per endpoint |
| 12 | PlannerClient (task CRUD, generate schedules) | `planner.go` | Mock CRUD + POST compute |
| 13 | ScheduleClient (active, get, put, delete by date) | `schedule.go` | Mock CRUD with date paths |
| 14 | ServiceConfigClient (Slack/Email/Calendar CRUD + toggle + status) | `service_config.go` | Mock per-type CRUD |
| 15 | RulesClient (CRUD + patch) | `rules.go` | Mock CRUD |
| 16 | QueueClient (depth query) | `queue.go` | Mock GET |
| 17 | TimerClient.GetState | `timer.go` | Mock GET |

## Dependencies

- `github.com/coder/websocket` — WebSocket client (same library server uses; chosen in Feature 096 ADR)
- No other new dependencies. Uses `net/http`, `encoding/json`, `crypto/sha256` from stdlib.

## Test Coverage

All adapter methods tested against `httptest.NewServer` mock servers. Tests verify:
- Correct HTTP method and path
- Authorization header present with token
- Request body marshaling (camelCase JSON)
- Response unmarshaling to SDK types
- Structured error handling for 4xx/5xx responses
- WebSocket connection lifecycle and event delivery
- Reconnection behavior on disconnect
- First-client token auto-issue flow

Target: >= 80% coverage on `pkg/client/`.

## Success Criteria

- All server endpoints have corresponding SDK methods
- SDK types are self-contained — no imports from `internal/`
- Structured errors preserve server error codes
- WebSocket reconnects automatically with exponential backoff
- First-client auth flow is transparent to callers
- Pairing flow provides clear API for UI integration
- `pkg/client/` is importable by external consumers
- All tests pass with >= 80% coverage
