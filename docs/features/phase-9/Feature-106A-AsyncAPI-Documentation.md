# Feature 106A: AsyncAPI Documentation

**Phase:** Phase-9-Feature-106A
**Status:** Planning
**Depends on:** Features 097-105 (server APIs)
**Artifacts:** `docs/api/asyncapi.yaml`

---

## Overview

Create a machine-readable AsyncAPI 3.0 specification document that describes every endpoint exposed by `cue-server`. This gives alternative UI developers (web frontend, TUI, mobile app, CLI scripts) a complete API contract covering both REST operations and WebSocket event streams.

The specification is hand-authored YAML checked into the repository — not auto-generated from code annotations. This keeps the Go codebase free of documentation framework dependencies while providing a single authoritative reference for all API consumers, including the client SDK (Feature 106).

## Rationale

Phase 9 Overview (section 5 — Documentation) identified the need for a machine-readable API contract. AsyncAPI 3.0 was chosen over OpenAPI 3.1 because:

1. **WebSocket-native** — AsyncAPI models pub/sub channels and message payloads as first-class concepts. OpenAPI treats WebSocket as a second-class citizen (callback + webhook workarounds).
2. **REST + async in one spec** — AsyncAPI 3.0 supports request/reply operations alongside event channels, so one document covers the full API surface.
3. **Tooling ecosystem** — AsyncAPI Studio for editing, `@asyncapi/generator` for HTML docs, and `@asyncapi/modelina` for client codegen if needed later.
4. **Superset alignment** — AsyncAPI 3.0 reuses JSON Schema for payload definitions, making schemas portable to OpenAPI consumers.

## Scope

### Endpoints to Document

#### REST Operations (request/reply)

| Group | Endpoints | Source Feature |
|-------|-----------|----------------|
| Health | `GET /health`, `GET /health/ready`, `GET /api/v1/health`, `GET /api/v1/health/ready` | 097 |
| Messages | `GET /api/v1/messages`, `GET /api/v1/messages/{id}` | 098 |
| Notifications | `GET /api/v1/notifications`, `GET /api/v1/notifications/{id}`, `POST .../resolve`, `POST .../dismiss` | 098 |
| Feedback Buffer | `GET /api/v1/buffer`, `GET /api/v1/buffer/{id}`, `GET .../stats`, `POST .../rate`, `DELETE .../` | 100 |
| Tasks | `GET /api/v1/tasks`, `POST /api/v1/tasks`, `GET/PUT/DELETE /api/v1/tasks/{id}` | 101A |
| Day Planner | `GET /api/v1/planner/active`, `DELETE .../active`, `POST .../generate`, `GET/PUT/DELETE /api/v1/planner/{date}` | 101 |
| Service Config (Slack) | `GET/POST /api/v1/services/slack`, `GET/PUT/DELETE .../slack/{id}`, `POST .../toggle` | 102 |
| Service Config (Email) | `GET/POST /api/v1/services/email`, `GET/PUT/DELETE .../email/{id}`, `POST .../toggle` | 102 |
| Service Config (Calendar) | `GET/POST /api/v1/services/calendar`, `GET/PUT/DELETE .../calendar/{id}`, `POST .../toggle` | 102 |
| Service Status | `GET /api/v1/services/status` | 102 |
| Routing Rules | `GET/POST /api/v1/rules`, `GET/PUT/DELETE /api/v1/rules/{id}`, `POST .../reorder` | 103 |
| Timer | `GET /api/v1/timer`, `POST .../start`, `POST .../stop`, `POST .../skip` | 104 |
| Settings | `GET/PATCH /api/v1/settings`, `GET/PUT .../notification-volume`, `GET/PUT .../timer-volume` | 105 |
| Events (REST replay) | `GET /api/v1/events?since={seq}` | 099 |

#### WebSocket Channels (pub/sub)

| Channel | Direction | Message Types | Source Feature |
|---------|-----------|---------------|----------------|
| `/api/v1/websocket/events` | server-to-client | `activity`, `notification`, `timer_tick`, `timer_block_complete` | 099, 104 |

### Schemas to Define

Reusable JSON Schema components for all request/response payloads:

- `Message` — full message event model
- `NotificationSummary` / `NotificationDetail` — notification list item and detail
- `BufferedMessage` / `BufferStats` — feedback buffer models
- `Task` / `TaskCreateRequest` / `TaskUpdateRequest` — todo CRUD models
- `Schedule` / `ScheduleOption` / `GenerateRequest` — planner models
- `SlackAccount` / `EmailAccount` / `CalendarAccount` — service config models (with masked credentials)
- `ServiceStatus` — watcher registration state
- `RoutingRule` / `RuleCreateRequest` — routing rule models
- `TimerState` — timer status model
- `Settings` / `VolumeLevel` — settings models
- `EventEnvelope` — WebSocket event wrapper (`seq`, `type`, `timestamp`, `data`, `dropped_since_last`)
- `ActivityEvent` / `NotificationEvent` / `TimerTickEvent` — event payload types
- `PaginatedResponse` — generic pagination wrapper (`items`, `total`, `offset`, `limit`)
- `ErrorResponse` — standard error envelope (`error.code`, `error.message`)

## Design Decisions

### 1. Hand-Authored YAML

**Decision: Check in a single `docs/api/asyncapi.yaml` maintained by hand.**

Alternatives considered:
- **Code annotations** (swaggo, go-swagger) — adds build-time dependencies, clutters handler code with comment directives, poor WebSocket support.
- **Auto-generated from tests** — fragile, hard to review diffs, doesn't capture intent or descriptions.

Hand-authored YAML means the spec is a deliberate design artifact, reviewed in PRs like any other documentation. Drift from implementation is caught by integration tests (see Testing section).

### 2. AsyncAPI 3.0 (not 2.x)

AsyncAPI 3.0 introduces the `operation` / `channel` / `message` separation that cleanly models both REST request/reply and WebSocket pub/sub. Version 2.x conflates channels with operations, making REST awkward.

### 3. File Location

`docs/api/asyncapi.yaml` — alongside other documentation, not in a generated output directory. The file is source-controlled and human-editable.

### 4. Schema Source of Truth

The AsyncAPI schemas are the **documentation** source of truth for API consumers. The Go structs in handler packages remain the **implementation** source of truth. Consistency between the two is verified by integration tests that validate response payloads against the AsyncAPI schemas.

### 5. Planned vs Implemented Endpoints

All endpoints (implemented and planned) are included in the spec. Planned endpoints that are not yet implemented are tagged with `x-status: planned` in the operation metadata. This gives SDK and alternative UI developers a complete picture of the API surface for planning purposes.

## Spec Structure

```yaml
asyncapi: 3.0.0
info:
  title: Cue Server API
  version: 1.0.0
  description: Local-first ADHD productivity assistant — REST + WebSocket API
  license:
    name: AGPL-3.0

servers:
  local:
    host: localhost:9400
    protocol: http
    description: Default local server

channels:
  # REST channels (one per resource group)
  health: ...
  messages: ...
  notifications: ...
  buffer: ...
  tasks: ...
  planner: ...
  services-slack: ...
  services-email: ...
  services-calendar: ...
  services-status: ...
  rules: ...
  timer: ...
  settings: ...
  events: ...
  # WebSocket channel
  event-stream: ...

operations:
  # One operation per HTTP method + path combination
  listMessages: ...
  getMessage: ...
  # ... etc

components:
  schemas:
    Message: ...
    ErrorResponse: ...
    PaginatedResponse: ...
    # ... all reusable schemas
  messages:
    ActivityEvent: ...
    NotificationEvent: ...
    # ... all message types
```

## Testing

### Schema Validation Tests

Integration tests that:
1. Start `cue-server` on an ephemeral port
2. Make API calls to each endpoint
3. Parse `docs/api/asyncapi.yaml` and extract the relevant response schema
4. Validate the actual JSON response against the schema

This catches drift between the spec and implementation without adding annotations to production code.

### Spec Lint

A `just` command to validate the AsyncAPI YAML:
```bash
just api-lint  # asyncapi validate docs/api/asyncapi.yaml
```

This requires the `@asyncapi/cli` tool installed (documented in contributing guide, optional for Go-only contributors).

## HTML Documentation Generation

Optional — not part of the core feature, but enabled by it:

```bash
just api-docs  # asyncapi generate fromTemplate docs/api/asyncapi.yaml @asyncapi/html-template -o _build/api-docs/
```

Generated HTML is placed in `_build/` (not committed). Useful for developers building alternative UIs.

## Implementation Notes

1. Start with implemented endpoints (Features 097-102) — these have concrete response shapes to reference.
2. Add planned endpoints (Features 103-105) with `x-status: planned` tags.
3. WebSocket event envelope schema comes from Feature 099's `EventEnvelope` struct.
4. Use `$ref` extensively — shared pagination, error responses, and common fields should be defined once in `components/schemas/`.
5. The `PaginatedResponse` wrapper uses a generic pattern: `items` array with `$ref` to the item schema, plus `total`, `offset`, `limit` fields.

## Success Criteria

- Single `docs/api/asyncapi.yaml` covers all REST and WebSocket endpoints
- Valid AsyncAPI 3.0 — passes `asyncapi validate`
- Schemas match actual Go response structs (verified by integration tests)
- Planned endpoints clearly marked with `x-status: planned`
- HTML docs generate cleanly from the spec
