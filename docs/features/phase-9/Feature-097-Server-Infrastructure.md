# Feature 097: Server Infrastructure + Composition Root

**Phase:** Phase-9-Feature-097
**Status:** Planning
**Package:** `cmd/cue-server/`, `internal/server/`

---

## Overview

Create the `cue-server` binary — a headless entry point that wires the same repositories, services, and watchers as the GUI binary but exposes them over HTTP/WebSocket instead of Fyne. This is the foundation that all subsequent API features build on.

## Design Decisions to Make

### Composition Root Strategy

The GUI binary (`cmd/cue/main.go`, 642 lines) wires repositories, services, watchers, presenters, and Fyne UI. The server binary needs the same repositories, services, and watchers but replaces presenters+Fyne with HTTP handlers+WebSocket.

**Decision: Option B — Independent composition roots.** Each binary has its own `main.go` that wires everything from scratch. The duplication is small (~100-150 lines) and the binaries serve fundamentally different purposes (Fyne GUI vs HTTP/WebSocket). If drift becomes a real problem after 2-3 features, extract shared wiring then — with concrete knowledge of what actually overlaps.

### Server Lifecycle

**Decision: Startup and shutdown sequence:**

1. Load config from `~/.cue/config.toml` (same as GUI)
2. Open SQLite database, run migrations
3. Initialize repositories
4. Initialize services (orchestrator, buffer, planner, queue processor, vector store)
5. Create and start watchers (Slack, Email) based on enabled service configs
6. Start the orchestrator polling loop
7. Bind HTTP server to configured host:port
8. Start WebSocket event broadcaster
9. Block until SIGINT/SIGTERM
10. Graceful shutdown: stop orchestrator, drain connections, close DB

**Decision: Restart-only for core config, hot-reload for watcher/service accounts.** Core config (port, DB path, Ollama settings, thresholds) requires a server restart. Watcher configuration (adding/removing/updating Slack workspaces and email accounts) is hot-reloaded — service accounts are stored in the database (managed via Settings UI), so the server watches for DB changes and starts/stops watchers accordingly. No restart needed for service account changes.

### HTTP Server Configuration

New config section needed in `config.toml`:

```toml
[server]
host = "0.0.0.0"      # Bind address (all interfaces — TOFU auth is the trust boundary)
port = 7130            # HTTP + WebSocket port
read_timeout_seconds = 30
write_timeout_seconds = 30
```

**Decisions:**
- **Port:** 7130.
- **Host:** `0.0.0.0` (all interfaces). TOFU token auth (Feature 096 Decision 3) is the trust boundary, not network binding. LAN access is needed for mobile/alternative UIs.
- **Config file:** Same `~/.cue/config.toml` for both binaries. Server ignores `[gui]` section, GUI ignores `[server]` section.

### Request/Response Conventions

Establish these once, follow everywhere:

**Decisions:**
- **Field naming:** `camelCase` (JavaScript/frontend convention — primary consumers are web and mobile UIs).
- **Date format:** RFC 3339 (`2026-04-10T14:30:00Z`).
- **UUID format:** Standard hyphenated (`550e8400-e29b-41d4-a716-446655440000`).
- **Null handling:** Omit null fields (`omitempty`).
- **Error response shape:** Structured — `{"error": {"code": "NOT_FOUND", "message": "Message not found"}}`. Codes allow programmatic error handling without string parsing.

### Middleware Stack

**Question: What middleware does the server need from day one?**

| Middleware | Purpose | Day One? |
|---|---|---|
| Request logging | Debug, audit | Yes |
| Panic recovery | Don't crash on handler panics | Yes |
| CORS | Cross-origin web UIs | Yes (configurable origins) |
| Request ID | Trace requests through logs | Yes |
| Content-Type enforcement | Reject non-JSON bodies | Yes |
| Authentication | TOFU + bearer tokens (Feature 096 Decision 3) | Yes |
| Rate limiting | Prevent abuse | No (local-only service) |
| Compression | Response gzip | No (local network, small payloads) |

### Health & Diagnostics

**Question: What operational endpoints?**

- `GET /health` — Is the server running? Returns 200 + `{"status": "ok"}`.
- `GET /health/ready` — Are all subsystems initialized? (DB connected, orchestrator running, Ollama reachable). Returns 200 or 503 with details.
- `GET /api/v1/status` — Runtime info: uptime, active watchers, queue depth, last poll times.

These are valuable for any UI that needs to show connection/system status.

## Behaviors to Implement

1. **Binary skeleton** — `cmd/cue-server/main.go` that loads config, opens DB, starts HTTP server, handles signals for graceful shutdown.
2. **Config extension** — Add `[server]` section to config with host, port, timeouts. Validate in `config.Validate()`.
3. **HTTP server setup** — Router with middleware stack (logging, recovery, CORS, request ID). Mounts under `/api/v1/`.
4. **Health endpoints** — `/health` and `/health/ready` with subsystem checks.
5. **WebSocket hub** — Central broadcaster that accepts subscriber connections and fans out events. Handles client connect/disconnect/reconnect gracefully.
6. **Graceful shutdown** — SIGINT/SIGTERM triggers ordered shutdown: stop accepting connections, drain in-flight requests, stop orchestrator, close DB.

## Error Handling

| Scenario | Behavior |
|---|---|
| Config file missing | Create with defaults (same as GUI) |
| Port already in use | Fatal error with clear message |
| DB migration fails | Fatal error, do not start server |
| Ollama unreachable at startup | Warn, start anyway (fallback scoring works) |
| Client WebSocket disconnect | Remove from subscriber list, log, continue |
| Handler panic | Recovery middleware returns 500, logs stack trace |

## Testing Considerations

- Server startup/shutdown: Use `httptest.Server` or start real server on ephemeral port in tests.
- WebSocket hub: Test with multiple concurrent subscribers, verify broadcast delivery and clean disconnect.
- Health endpoints: Test ready check with and without functioning subsystems.
- Integration: Start full server against in-memory SQLite, verify end-to-end request flow.

## Dependencies

- HTTP router: `net/http` stdlib or `github.com/go-chi/chi/v5` (pure Go, confirm no CGO)
- WebSocket: `nhooyr.io/websocket` (pure Go, confirm no CGO)
- Signal handling: `os/signal` stdlib
- No Fyne dependency — the server binary must not import any Fyne packages

## Decisions Summary

1. **Composition root:** Independent `main.go` files (Option B). No shared wiring package.
2. **Config reload:** Restart-only for core config. Hot-reload for watcher/service accounts (DB-driven).
3. **Port:** 7130.
4. **JSON fields:** `camelCase`.
5. **Error shape:** Structured — `{"error": {"code": "...", "message": "..."}}`.
6. **Authentication:** TOFU with pairing flow, bearer tokens (Feature 096 Decision 3).
7. **Module:** Same Go module as `cmd/cue/`. Single `go.mod`.
8. **Host binding:** `0.0.0.0` (all interfaces). TOFU auth is the trust boundary.
