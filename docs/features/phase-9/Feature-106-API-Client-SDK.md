# Feature 106: API Client SDK

**Phase:** Phase-9-Feature-106
**Status:** Done
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
  client.go            # APIClient base (HTTP client, server URL, token, helpers, TOKEN_ISSUED auto-retry)
  errors.go            # Structured error types (APIError, error code constants)
  auth.go              # AuthClient interface + adapter (pairing, token management)
  messages.go          # MessageClient interface + adapter (messages + notifications)
  activity.go          # ActivityClient (WebSocket event stream + replay)
  feedback.go          # FeedbackClient interface + adapter (buffer operations)
  tasks.go             # TaskClient interface + adapter (task CRUD)
  rules.go             # RulesClient interface + adapter
  service_config.go    # ServiceConfigClient interface + adapter (Slack, Email, Calendar accounts)
  schedule.go          # ScheduleClient interface + adapter (active + date-based schedules, generate)
  timer.go             # TimerClient interface + adapter (get state)
```

Rationale: Mirrors the server handler file layout. Each file contains one interface and its concrete adapter type. All adapters share the base `APIClient` for HTTP transport.

Note: The server has no queue endpoint, so no `queue.go` exists. The server's "planner" concerns split naturally into `tasks.go` (task CRUD from `/api/v1/tasks`), `schedule.go` (schedule state from `/api/v1/planner/*`), and `timer.go` (`/api/v1/timer`), matching the handler layout.

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

The server has handler-level interfaces like `TodoServicer`, `RulesManager`, `ServiceManager`, etc. The SDK provides matching client types (`TaskClient`, `RulesClient`, `ServiceConfigClient`) that map 1:1 to server endpoints.

Feature 107 (Fyne Client Re-wire) composes these SDK types to satisfy presenter interface requirements. For example, the presenter's `BufferReviewer` combines methods from `FeedbackClient` and `MessageClient`.

This design keeps the SDK framework-agnostic and usable by any consumer, not just Fyne.

### 4. Grouped Interfaces Per Domain

**Decision: The SDK defines one interface per domain (e.g., `MessageClient`, `RulesClient`).**

Each domain file exports an interface and a concrete struct implementing it. Consumers get ready-made interfaces for mocking in tests. Nothing prevents consumers from defining narrower interfaces per Go convention.

### 5. JSON Field Naming

**Decision: `snake_case` matching the actual server response format, explicit DTO types.**

Every handler in `internal/server/handler/` uses `snake_case` JSON tags (e.g., `source_account`, `created_at`, `importance_score`, `elapsed_fraction`). SDK DTO types mirror this exactly. The Feature 096 ADR mentioned camelCase aspirationally, but the implementation landed on snake_case throughout — cleanup to a single convention is deferred.

Each adapter file defines its own request/response DTO types. Domain types from `internal/repository/` are not reused directly — the SDK has its own types to avoid coupling consumers to internal packages.

### 6. Structured Error Types

**Decision: `*APIError` type with code and message. Server returns flat `{"error": "message"}` strings; the SDK derives the code from the HTTP status.**

The server's `writeJSONError` helper (in `internal/server/handler/notification.go`) emits a flat error shape:
```json
{"error": "message describing what went wrong"}
```

There is no structured `error.code` field on non-auth endpoints. The SDK's `parseErrorResponse` reads the `error` string as `Message` and derives `Code` from the HTTP status code.

```go
type APIError struct {
    StatusCode int    // HTTP status code
    Code       string // Derived from StatusCode (e.g., "NOT_FOUND" for 404)
    Message    string // Server's flat "error" string
}

func (e *APIError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

Error code constants (mapped from HTTP status):
```go
const (
    ErrCodeValidation   = "VALIDATION_ERROR" // 400
    ErrCodeUnauthorized = "UNAUTHORIZED"     // 401
    ErrCodeNotFound     = "NOT_FOUND"        // 404
    ErrCodeConflict     = "CONFLICT"         // 409
    ErrCodeServerError  = "SERVER_ERROR"     // 5xx
    ErrCodeTokenIssued  = "TOKEN_ISSUED"     // Special: 401 with nested auth error body (see §7)
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
- `InitiatePairing(ctx, label) (PairSession, error)` — `POST /api/v1/auth/pair`, returns `{request_id, code}`
- `PollPairing(ctx, requestID) (PairResult, error)` — `GET /api/v1/auth/pair/{id}`, returns `{status, token}`
- `ApprovePairing(ctx, requestID) error` — `POST /api/v1/auth/pair/{id}/approve` (authenticated)
- `DenyPairing(ctx, requestID) error` — `POST /api/v1/auth/pair/{id}/deny` (authenticated)
- `ListTokens(ctx) ([]TokenInfo, error)` — `GET /api/v1/auth/tokens`
- `UpdateTokenLabel(ctx, id, label) error` — `PUT /api/v1/auth/tokens/{id}`
- `RevokeToken(ctx, id) error` — `DELETE /api/v1/auth/tokens/{id}`

There is no `/auth/logout` endpoint per Feature 108's decision — `DELETE /auth/tokens/{id}` covers revocation.

The pairing initiate response contains `{request_id, code}` only — no `expires_at` field. The 60-second expiry is enforced server-side; clients poll until the status changes to `approved`, `denied`, or `expired`.

For the first-client flow, the `APIClient` helpers detect the TOKEN_ISSUED shape on a 401 response, store the token, and retry the original request automatically. The actual response body shape is:
```json
{
  "error": {"code": "TOKEN_ISSUED", "message": "..."},
  "token": "<plaintext-bearer-token>"
}
```
Note the nested `error` object — this differs from the flat error shape used elsewhere. This is handled internally by the `APIClient` — callers don't need to manage it.

### 8. WebSocket Event Stream

**Decision: Automatic reconnect with hardcoded exponential backoff. Manual replay for missed events.**

The `ActivityClient` adapter maintains a persistent WebSocket connection to `/api/v1/websocket/events?token=<token>`. On disconnect:
1. Close the old connection
2. Backoff: 1s, 2s, 4s, 8s, ... capped at 30s
3. Reconnect and resume delivering events to the channel
4. Reset backoff on successful reconnect

**No automatic replay.** When the connection drops and reconnects, there may be a gap in the event sequence. The `Events() <-chan EventEnvelope` channel exposes sequence numbers; consumers who care about gaps can call `Replay(ctx, sinceSeq)` to fetch missed events via `GET /api/v1/events?since={seq}`.

The replay endpoint returns a `ReplayResponse` mirroring the server's `HistoryResponse`:
```json
{
  "events": [ {"seq": 42, "type": "activity", "timestamp": "...", "data": {...}}, ... ],
  "truncated": false,
  "oldest_seq": 1,
  "latest_seq": 42
}
```
`truncated` is `true` when requested `sinceSeq` is older than the server's ring buffer (500 events) retained.

The `Events()` channel is never closed — it just stops receiving during disconnects. The `LastSeq() uint64` method returns the highest sequence number received.

Event envelopes use `json.RawMessage` for the `data` field so consumers can unmarshal type-specific payloads (`ActivityData`, `AlertData`, `TimerTickData`, `TimerBlockCompleteData`) on demand.

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
| `AuthClient` | `authAdapter` | `POST /auth/pair`, `GET /auth/pair/{id}`, `POST .../approve`, `POST .../deny`, `GET /auth/tokens`, `PUT /auth/tokens/{id}`, `DELETE /auth/tokens/{id}` |
| `MessageClient` | `messageAdapter` | `GET /messages`, `GET /messages/{id}`, `GET /notifications`, `GET /notifications/{id}`, `POST /notifications/{id}/resolve`, `POST /notifications/{id}/dismiss` |
| `FeedbackClient` | `feedbackAdapter` | `GET /buffer`, `GET /buffer/{id}`, `GET /buffer/stats`, `POST /buffer/{id}/rate`, `DELETE /buffer/{id}` |
| `TaskClient` | `taskAdapter` | `GET/POST /tasks`, `GET/PUT/DELETE /tasks/{id}` |
| `ScheduleClient` | `scheduleAdapter` | `GET/DELETE /planner/active`, `GET/PUT/DELETE /planner/{date}`, `POST /planner/generate` |
| `ServiceConfigClient` | `serviceConfigAdapter` | `CRUD /services/slack/{id}`, `CRUD /services/email/{id}`, `CRUD /services/calendar/{id}`, `POST .../toggle`, `GET /services/status` |
| `RulesClient` | `rulesAdapter` | `GET/POST /rules`, `GET/PUT/PATCH/DELETE /rules/{id}` |
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
- `Task`, `CreateTaskRequest`, `UpdateTaskRequest`
- `Schedule`, `ScheduleBlock`, `GenerateRequest`, `GenerateResponse`, `ScheduleOption`
- `SlackAccount`, `EmailAccount`, `CalendarAccount`, `ServiceStatus`
- `RoutingRule`, `PatchRuleRequest`
- `TimerState`
- `EventEnvelope` (with `json.RawMessage` `data`), `ReplayResponse`
- `TokenInfo`, `PairSession`, `PairResult`
- `APIError`

All types use `snake_case` JSON tags matching the server's `internal/server/handler/*.go` DTOs exactly.

## TDD Behaviors

Structured as **12 grouped micro-loops** (RED → GREEN → REFACTOR per loop). Loops are ordered by dependency: foundation first, then auth (needed by all authenticated calls), then domain adapters.

| Loop | Adapter | Behaviors Covered | Test Strategy |
|---|---|---|---|
| 1 | `client.go` | `New(serverURL)`, `SetToken`, `Health(ctx)`, HTTP helpers set auth header + content-type | `httptest.NewServer` returning 200/503 |
| 2 | `errors.go` | `APIError` parsing from flat `{"error":"msg"}` responses, HTTP status → code mapping, `errors.As` | Mock server returning 4xx/5xx |
| 3 | `auth.go` | InitiatePairing, PollPairing, ApprovePairing, DenyPairing, ListTokens, UpdateTokenLabel, RevokeToken | Mock server per endpoint |
| 4 | `client.go` (mod) | First-client auto-token: 401 with nested TOKEN_ISSUED body → store token + retry once | Mock server issuing token on first request |
| 5 | `messages.go` | List/Get messages, List/Get/Resolve/Dismiss notifications | Mock server returning canned JSON |
| 6 | `feedback.go` | List/Get buffered, BufferStats, RateBuffered, DeleteBuffered | Mock server per endpoint |
| 7 | `tasks.go` | Task CRUD (ListTasks, CreateTask, GetTask, UpdateTask, DeleteTask) | Mock CRUD |
| 8 | `rules.go` | Rules CRUD + PatchRule (priority/enabled) | Mock CRUD + PATCH |
| 9 | `service_config.go` | Slack/Email/Calendar CRUD + toggle + ServiceStatus (grouped, shared helper) | Mock per-type CRUD |
| 10 | `schedule.go` + `timer.go` | ActiveSchedule, Delete/Get/Put/Delete by date, GenerateSchedules, GetTimerState | Mock with date paths |
| 11 | `activity.go` | WebSocket Connect, Events channel, LastSeq, Replay via GET /events?since= | `httptest.NewServer` with WS upgrade + HTTP replay |
| 12 | `activity.go` (mod) | Reconnection with exponential backoff (1s → 30s cap, reset on success, context-cancellable) | Mock server that disconnects then accepts |

Total distinct behaviors: 15 (11 HTTP adapter groups, 1 WS connect/receive/replay, 1 WS reconnect, 1 error parsing, 1 first-client auto-token).

## Dependencies

- `github.com/coder/websocket` — WebSocket client (same library server uses; chosen in Feature 096 ADR)
- No other new dependencies. Uses `net/http`, `encoding/json`, `crypto/sha256` from stdlib.

## Test Coverage

All adapter methods tested against `httptest.NewServer` mock servers. Tests verify:
- Correct HTTP method and path
- Authorization header present with token
- Request body marshaling (snake_case JSON)
- Response unmarshaling to SDK types
- Structured error handling for 4xx/5xx responses
- WebSocket connection lifecycle and event delivery
- Reconnection behavior on disconnect
- First-client token auto-issue flow

Target: >= 80% coverage on `pkg/client/`.

**Shipped: 86.5% statement coverage across 93 table-driven tests** (confirmed via `go test -cover ./pkg/client/...`).

## Wiring

The SDK is a standalone package. It has no consumer in the composition root — `cmd/cue-server/main.go` is the server; `cmd/cue/main.go` still builds against `internal/*` directly. Feature 107 (Fyne Client Re-wire) is the first consumer and will construct an `APIClient` + adapters to satisfy presenter interfaces.

Standalone-SDK wiring verification:
- No leftover `ErrNotImplemented` references in production code (confirmed via `grep -rn 'ErrNotImplemented' pkg/client/ --include='*.go' | grep -v '_test.go'`).
- The sentinel was removed in the chore commit once all stubs were replaced.
- `just security` and `just vulncheck` report zero issues for the new package.

## Success Criteria

- ✅ All server endpoints except the (absent) queue endpoint have corresponding SDK methods
- ✅ SDK types are self-contained — no imports from `internal/`
- ✅ Structured errors preserve status-code-derived codes and server messages
- ✅ WebSocket reconnects automatically with exponential backoff
- ✅ First-client auth flow is transparent to callers
- ✅ Pairing flow provides a clear 4-method API for UI integration
- ✅ `pkg/client/` is importable by external consumers
- ✅ All tests pass with 86.5% coverage (≥ 80% target)

## TDD Agent Stats

All twelve micro-loops used the full RED → GREEN → REFACTOR agent-team pipeline with context isolation (see `.claude/agents/`). Loops 1, 4, 6, 7, 8, 10 required no refactor commit (implementer output was already clean).

| Implementation Phase | TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Phase-9-Feature-106 (Step 0 doc fix) | DOCS | main | inline | inline | 5a5e85f |
| Phase-9-Feature-106 (L1 APIClient + Health) | RED | go-test-designer | ~65s | ~42,459 | 69f990a |
| Phase-9-Feature-106 (L1 APIClient + Health) | GREEN | go-implementer | ~49s | ~35,095 | 92620b1 |
| Phase-9-Feature-106 (L1 APIClient + Health) | REFACTOR | go-refactorer | ~25s | ~23,667 | — |
| Phase-9-Feature-106 (L2 APIError) | RED | go-test-designer | ~98s | ~40,455 | e1f4f7e |
| Phase-9-Feature-106 (L2 APIError) | GREEN | go-implementer | ~64s | ~36,934 | e88adf0 |
| Phase-9-Feature-106 (L2 APIError) | REFACTOR | go-refactorer | ~63s | ~26,992 | 86e3899 |
| Phase-9-Feature-106 (L3 AuthClient) | RED | go-test-designer | ~104s | ~52,839 | 0633359 |
| Phase-9-Feature-106 (L3 AuthClient) | GREEN | go-implementer | ~85s | ~47,912 | 9421064 |
| Phase-9-Feature-106 (L3 AuthClient) | REFACTOR | go-refactorer | ~74s | ~33,616 | 82f0920 |
| Phase-9-Feature-106 (L4 TOKEN_ISSUED retry) | RED | go-test-designer | ~106s | ~55,093 | cd06b78 |
| Phase-9-Feature-106 (L4 TOKEN_ISSUED retry) | GREEN | go-implementer | ~72s | ~45,950 | 5f20cae |
| Phase-9-Feature-106 (L4 TOKEN_ISSUED retry) | REFACTOR | go-refactorer | ~29s | ~26,292 | — |
| Phase-9-Feature-106 (L5 MessageClient) | RED | go-test-designer | ~151s | ~54,850 | 6585993 |
| Phase-9-Feature-106 (L5 MessageClient) | GREEN | go-implementer | ~67s | ~51,273 | 0f3fed1 |
| Phase-9-Feature-106 (L5 MessageClient) | REFACTOR | go-refactorer | ~69s | ~34,238 | 2429ff3 |
| Phase-9-Feature-106 (L6 FeedbackClient) | RED | go-test-designer | ~142s | ~62,221 | 8b5e811 |
| Phase-9-Feature-106 (L6 FeedbackClient) | GREEN | go-implementer | ~68s | ~52,068 | 3f22494 |
| Phase-9-Feature-106 (L6 FeedbackClient) | REFACTOR | go-refactorer | ~32s | ~30,540 | — |
| Phase-9-Feature-106 (L7 TaskClient) | RED | go-test-designer | ~138s | ~66,201 | 4b2d6db |
| Phase-9-Feature-106 (L7 TaskClient) | GREEN | go-implementer | ~59s | ~51,084 | 775829c |
| Phase-9-Feature-106 (L7 TaskClient) | REFACTOR | go-refactorer | ~26s | ~26,513 | — |
| Phase-9-Feature-106 (L8 RulesClient) | RED | go-test-designer | ~178s | ~70,332 | 1451d15 |
| Phase-9-Feature-106 (L8 RulesClient) | GREEN | go-implementer | ~61s | ~52,308 | 4fc2324 |
| Phase-9-Feature-106 (L8 RulesClient) | REFACTOR | go-refactorer | ~27s | ~27,086 | — |
| Phase-9-Feature-106 (L9 ServiceConfigClient) | RED | go-test-designer | ~226s | ~88,091 | ade7b90 |
| Phase-9-Feature-106 (L9 ServiceConfigClient) | GREEN | go-implementer | ~106s | ~63,427 | 10588d2 |
| Phase-9-Feature-106 (L9 ServiceConfigClient) | REFACTOR | go-refactorer | ~176s | ~40,754 | 7aa71cb |
| Phase-9-Feature-106 (L10 Schedule + Timer) | RED | go-test-designer | ~171s | ~70,202 | aeaa41f |
| Phase-9-Feature-106 (L10 Schedule + Timer) | GREEN | go-implementer | ~72s | ~57,141 | 85c03ce |
| Phase-9-Feature-106 (L10 Schedule + Timer) | REFACTOR | go-refactorer | ~22s | ~28,516 | — |
| Phase-9-Feature-106 (L11 ActivityClient + Replay) | RED | go-test-designer | ~164s | ~71,807 | 500ef8f |
| Phase-9-Feature-106 (L11 ActivityClient + Replay) | GREEN | go-implementer | ~80s | ~57,871 | ab824c4 |
| Phase-9-Feature-106 (L11 ActivityClient + Replay) | REFACTOR | go-refactorer | ~55s | ~34,046 | 492d7de |
| Phase-9-Feature-106 (L12 WS Reconnection) | RED | go-test-designer | ~210s | ~58,937 | 3c7bfe1 |
| Phase-9-Feature-106 (L12 WS Reconnection) | GREEN | go-implementer | ~1409s | ~69,654 | 75b8680 |
| Phase-9-Feature-106 (L12 WS Reconnection) | REFACTOR | go-refactorer | ~47s | ~31,035 | a47b16c |
| Phase-9-Feature-106 (wiring cleanup) | CHORE | main | inline | inline | 2916b2a |

Notable: L12 GREEN took ~23 minutes in-loop due to a WebSocket close deadlock caught by the reconnection tests (`coder/websocket.Conn.Close` waits on `readMu` which `readLoop` holds during `Read`). Resolved by switching `Close` to use `CloseNow()` instead of the graceful close handshake.
