# Feature 106A: API Documentation

**Phase:** Phase-9-Feature-106A
**Status:** Done
**Depends on:** Features 097–104, 108 (all Done)
**Artifacts:** `docs/api/openapi.yaml`, `docs/api/websocket.md`, `scripts/api-lint/`, `just api-gen`, `just api-lint`, `cue-server` route `GET /docs/api`

---

## Overview

Publish a developer-facing reference for the full `cue-server` API so alternative client authors (web frontend, TUI, mobile app, CLI scripts) can build against a single documented contract.

The REST surface is described by an **OpenAPI 3.1** specification generated from `swaggo/swag` v2 annotations placed on the existing `http.Handler` functions. The single WebSocket channel is documented in a hand-authored Markdown reference. An embedded Swagger UI served at `/docs/api` by the server binary gives developers an interactive browser view of the spec without requiring any Node or external tooling.

This feature supersedes the original AsyncAPI-based plan for 106A. The move to OpenAPI reflects the breadth of existing Go tooling (pure-Go validators, embedded UI assets, mature codegen in every major language) and the fact that developers building clients are already familiar with OpenAPI.

## Rationale

Phase 9 Overview (§5 — Documentation) identified the need for a machine-readable API contract. The initial plan chose AsyncAPI 3.0 for its native WebSocket support. On review we prefer OpenAPI 3.1 because:

1. **Pure-Go ecosystem** — `kin-openapi`, `libopenapi`, and `swaggest/swgui` cover parsing, validation, and UI serving without introducing a Node or npm dependency.
2. **Familiarity and codegen** — OpenAPI is the de facto REST spec format; every major language has a generator. Client authors pay a lower learning cost.
3. **Annotations keep docs next to code** — `swaggo/swag` v2 annotations live above each handler, so schema changes are reviewed in the same PR that changes behavior. This replaces the drift risk of a parallel hand-maintained YAML.
4. **WebSocket surface is tiny** — a single channel (`/api/v1/websocket/events`) with seven event types. A short Markdown reference is clearer than contorting OpenAPI callbacks/webhooks or maintaining a second AsyncAPI document.

## Scope

### REST — annotation coverage

All routes registered in `internal/server/server.go`. Grouped for incremental delivery:

| Group | Handler package / file | Routes |
|-------|------------------------|--------|
| Health | `internal/server/` | `GET /health`, `/health/ready`, `/api/v1/health`, `/api/v1/health/ready` |
| Messages + Notifications | `internal/server/handler/` | `GET /api/v1/messages`, `/messages/{id}`, `/notifications`, `/notifications/{id}`, `POST .../resolve`, `POST .../dismiss` |
| Feedback Buffer | `internal/server/handler/` | `GET /api/v1/buffer`, `/buffer/stats`, `/buffer/{id}`, `POST .../rate`, `DELETE .../` |
| Tasks | `internal/server/handler/` | `GET/POST /api/v1/tasks`, `GET/PUT/DELETE /api/v1/tasks/{id}` |
| Day Planner | `internal/server/handler/` | `GET/DELETE /api/v1/planner/active`, `POST .../generate`, `GET/PUT/DELETE /api/v1/planner/{date}` |
| Services | `internal/server/handler/` | Slack, Email, Calendar CRUD + toggle; `GET /api/v1/services/status` |
| Routing Rules | `internal/server/handler/` | `GET/POST /api/v1/rules`, `GET/PUT/DELETE /api/v1/rules/{id}`, `POST .../reorder` |
| Timer | `internal/server/handler/` | `GET /api/v1/timer` |
| Auth + Pairing | `internal/server/handler/` | `POST /api/v1/auth/pair`, `GET .../{id}`, `POST .../approve`, `POST .../deny`, `GET /api/v1/auth/tokens`, `PUT/DELETE .../tokens/{id}`, `POST /api/v1/auth/logout` |
| Events replay | `internal/server/handler/` | `GET /api/v1/events?since={seq}` |

### WebSocket — Markdown reference

Single channel `/api/v1/websocket/events` documented in `docs/api/websocket.md`:

- Connection handshake, auth header requirements, reconnect / resume-by-seq semantics
- `EventEnvelope` schema (`seq`, `type`, `timestamp`, `data`, `dropped_since_last`)
- Per-event-type payload reference with JSON examples: `activity`, `notification`, `alert`, `timer_tick`, `timer_block_complete`, `pairing_request`, `pairing_resolved`

### Schemas

swaggo annotations reference existing Go response structs, which become OpenAPI `components/schemas` entries automatically. No parallel schema definitions are maintained.

### UI serving

`cue-server` mounts `github.com/swaggest/swgui` at `GET /docs/api` (and the supporting asset routes). The handler reads the generated spec via `embed.FS` from `docs/api/openapi.yaml`. The page also links to the WebSocket Markdown reference (served as static content under `/docs/api/websocket`).

## Design Decisions

### 1. OpenAPI 3.1 for REST, Markdown for WebSocket
OpenAPI treats WebSocket poorly (callbacks/webhooks are awkward workarounds). Since the WebSocket surface is one channel with a handful of event types, a short hand-authored Markdown reference is clearer than forcing it into OpenAPI.

### 2. swaggo/swag v2 annotations over hand-authored YAML
Annotations live above the handler they describe. A handler change and its doc change land in the same diff, so reviewers catch drift naturally. The generated `docs/api/openapi.yaml` is committed so external consumers can read it without running `swag`.

**Cost accepted:** ~5–15 lines of `// @...` comments per handler file. swaggo v2 is currently pre-release; if it blocks on a real issue during annotation, fall back to hand-authored YAML in the same file layout.

### 3. Pure-Go validation via `kin-openapi`
`scripts/api-lint/main.go` loads `docs/api/openapi.yaml` through `github.com/getkin/kin-openapi/openapi3` and runs `Validate(ctx)`. Exit 0 on success, exit 1 with diagnostics on failure. No Node, no npm, no external binaries.

### 4. `swaggest/swgui` for embedded UI
Pure Go. Embeds Swagger UI v5 assets. Serves as an `http.Handler` mounted on the existing `ServeMux`. No build-time generation of HTML; the UI loads the YAML at request time.

### 5. No drift-vs-implementation integration tests
Per direction: the spec is a developer reference. Clients consume it as-is; if behavior drifts from the spec, clients re-check the code. Writing response-validation tests adds cost without matching the intent of the docs.

### 6. `docs/api/openapi.yaml` is committed
Even though it is generated, checking it in gives external developers a stable URL for their codegen pipelines and makes spec changes reviewable in PRs. `just api-gen` regenerates; CI should not regenerate silently.

### 7. `/docs/api` served from the main binary
Avoids a separate docs process. Developers hitting a running `cue-server` can discover the API in the browser with no extra tooling.

## Work Breakdown

The feature proceeds in five phases. TDD micro-loops apply to Phase A (tooling) and Phase D (UI wiring). Phases B and C are documentation work and use `docs(api): ...` commits directly.

### Phase A — Tooling (TDD)

**A.1 swag v2 integration** (`chore(api): add swag v2 tool`)
- Add `swag` v2 to `go.mod` via `tool` directive.
- Add `just api-gen` target that invokes swag against the handler packages and writes `docs/api/openapi.yaml`.
- No tests; this is a build-tooling chore.

**A.2 `api-lint` validator** (TDD micro-loop)
- Create `scripts/api-lint/` with `main.go` + `main_test.go`.
- RED: `test(api): failing test for openapi validator` — test that `Validate(path)` returns error on a malformed fixture spec.
- GREEN: `feat(api): implement openapi validator [tests pass]` — minimal implementation using `kin-openapi/openapi3.Loader` + `Validate`.
- REFACTOR: `refactor(api): cleanup openapi validator` — iff non-trivial.
- Add `just api-lint` target invoking `go run ./scripts/api-lint docs/api/openapi.yaml`.

### Phase B — Handler annotations (documentation commits)

For each group in the Scope table, one commit:

1. Add swaggo v2 annotations above every handler constructor / function in the group.
2. Run `just api-gen && just api-lint`.
3. Commit: `docs(api): annotate <group> endpoints`.

Order (smallest surface first, to validate the toolchain early): Health → Timer → Events replay → Messages+Notifications → Buffer → Tasks → Planner → Services → Rules → Auth+Pairing.

### Phase C — WebSocket reference (documentation commit)

Hand-author `docs/api/websocket.md` covering connection, auth, envelope schema, per-event payload reference with JSON examples. Commit: `docs(api): websocket reference`.

### Phase D — UI route wiring (TDD micro-loop)

- RED: `test(server): failing test for /docs/api route` — suite test that `GET /docs/api` returns 200 with HTML containing Swagger UI markers, and that the embedded spec is reachable.
- GREEN: `feat(server): serve swagger UI at /docs/api [tests pass]` — mount `swaggest/swgui` handler on the ServeMux in `internal/server/server.go`; wire `embed.FS` for `docs/api/openapi.yaml`; add a static route for `docs/api/websocket.md`.
- REFACTOR: `refactor(server): cleanup docs route` — iff non-trivial.

### Phase E — Wiring verification + finalization

1. Grep for `ErrNotImplemented` in production code — none may remain.
2. `just api-gen && just api-lint` both pass.
3. Every route in `internal/server/server.go` has a matching annotated handler (spot-check grep).
4. `GET /docs/api` renders in the browser against a running server.
5. `just security && just vulncheck`.
6. `just fmt && just lint && just tidy` — commit `chore(api): fmt and lint`.
7. Post-feature docs commit: update this design doc with final stats, update `docs/agent-log.md`, `CHANGELOG.md`, and `README.md`. Commit: `docs(feat-106A): API docs shipping`.

## Testing

- **Validator** — unit tests in `scripts/api-lint/main_test.go` cover valid-spec pass, malformed-spec fail.
- **UI route** — suite test in `internal/server/server_test.go` verifies `GET /docs/api` returns Swagger UI HTML and that the spec asset is served.
- **No drift tests** — intentionally omitted per §Design Decision 5.

## Success Criteria

- `docs/api/openapi.yaml` is generated from annotations, committed, and valid OpenAPI 3.1 (`just api-lint` passes).
- `docs/api/websocket.md` covers the event stream channel and every event type with JSON examples.
- `GET /docs/api` on a running `cue-server` renders interactive Swagger UI and links to the WebSocket reference.
- `just api-gen` and `just api-lint` run with no external dependencies beyond Go.
- Every route registered in `internal/server/server.go` appears in the generated spec.
- Coverage gate ≥80% for any new Go code in `scripts/api-lint/` and the new server route wiring.

## Out of Scope

- Drift-vs-implementation integration tests (see §Design Decision 5).
- Handler framework migration (fuego / huma / goa). Future work if desired.
- Client codegen automation. The committed spec enables it; the repo does not run it.
- HTML export of the WebSocket reference — Markdown is sufficient.

## Final Stats

Feature shipped over 16 commits on `develop`:

| Phase | Commits | Notes |
|-------|---------|-------|
| A.1 swag tool | `fd4a7aa` | go.mod tool directive + `just api-gen`. |
| A.2 api-lint | `87cc81b` (RED) → `0a65dc7` (GREEN); refactor skipped | `kin-openapi/openapi3` validator; `just api-lint`. |
| B Health | `71085d2` | 4 routes + general API info block. |
| B Timer | `f7209e6` | 1 route. |
| B Events replay | `b6037b1` | 1 route. |
| B Messages + Notifications | `e8ac6ad` | 6 handlers. |
| B Buffer | `fafde08` | 5 handlers. |
| B Tasks | `1afdeff` | 5 handlers. |
| B Planner | `28ee0e9` | 4 handlers; `/active` aliases described in prose, not separately routed in spec (path-param mismatch with /active). |
| B Services | `9cfa470` | 19 handlers. |
| B Rules | `b5ea72c` | 6 handlers; PATCH covers reorder + toggle in lieu of a separate `/reorder`. |
| B Auth + Pairing | `21af1f8` | 7 handlers; `/auth/logout` not in current code base — omitted. |
| C WebSocket reference | `7223821` | `docs/api/websocket.md`. |
| D /docs/api route | `a71b323` (RED) → `b8097be` (GREEN) → `c6fdb9b` (REFACTOR) | `swaggest/swgui/v5emb`, embed package at `docs/api/embed.go`. |

Quality gates at completion: `just test` green, `just security` clean (0 issues), `just vulncheck` clean (0 vulnerabilities in called code), `just fmt && just lint && just tidy` clean.
