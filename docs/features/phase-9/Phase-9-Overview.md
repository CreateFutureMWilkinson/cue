# Phase 9: Headless Server Mode

## Goal

Create a `cue-server` binary that runs Cue headless — no GUI, no Fyne dependency. All functionality is exposed over REST + WebSocket, enabling alternative UIs (web frontend, TUI, mobile app, CLI scripts) to connect and interact with Cue.

## Design Principles

1. **Same services, different consumer.** The server wires the same repositories, orchestrator, watchers, planner, and buffer as the GUI binary. Only the presentation layer changes.
2. **Stateless where possible.** REST endpoints are stateless request/response. Server-side state is limited to the planner wizard session and the timer.
3. **Real-time via WebSocket.** Activity events, timer ticks, and notification arrivals are pushed to connected clients over a single multiplexed WebSocket connection.
4. **Local-first.** The server binds to localhost by default. No authentication in v1 (configurable for LAN access later).
5. **No Fyne dependency.** The server binary must not import Fyne packages. This ensures it builds and runs on headless systems.

## Protocol Decision

**REST + WebSocket Hybrid.** See Feature 096 for the full comparison of REST, gRPC, GraphQL, and the rationale for this choice.

## Feature Map

| # | Feature | Type | Complexity | Dependencies |
|---|---|---|---|---|
| 096 | [Protocol Selection](Feature-096-Server-Protocol-Selection.md) | ADR | N/A | None |
| 097 | [Server Infrastructure](Feature-097-Server-Infrastructure.md) | Infrastructure | Medium | 096 |
| 098 | [Message & Notification API](Feature-098-Message-Notification-API.md) | REST | Low-Medium | 097 |
| 099 | [Activity Event Stream](Feature-099-Activity-Event-Stream.md) | WebSocket | Medium | 097 |
| 099A | [Server Orchestrator Wiring](Feature-099A-Server-Orchestrator-Wiring.md) | Refactor | Medium | 097, 099 |
| 100 | [Feedback Buffer API](Feature-100-Feedback-Buffer-API.md) | REST | Low | 097 |
| 101 | [Day Planner API](Feature-101-Day-Planner-API.md) | REST + State | High | 097 |
| 102 | [Service Configuration API](Feature-102-Service-Configuration-API.md) | REST | Medium | 097 |
| 103 | [Routing Rules API](Feature-103-Routing-Rules-API.md) | REST | Low | 097 |
| 104 | [Timer API](Feature-104-Timer-API.md) | REST + WS | Medium | 097, 099 |
| 105 | ~~Settings API~~ (removed — redundant with 102) | — | — | — |
| 108 | [TOFU Pairing](Feature-108-TOFU-Pairing.md) | Auth | Medium | 097, 099 |
| 106 | [API Client SDK](Feature-106-API-Client-SDK.md) | Client | High | 096-104, 108 |
| 106A | [AsyncAPI Documentation](Feature-106A-AsyncAPI-Documentation.md) | Documentation | Low | 097-104, 108 |
| 109 | [Todo Domain Restructure](Feature-109-Todo-Domain-Restructure.md) | Refactor | Medium | 101A, 102, 106 |
| 110 | [TOFU Client Bootstrap](Feature-110-TOFU-Client-Bootstrap.md) | Auth Library | Low | 106, 108 |
| 107 | [Fyne Client Re-wire](Feature-107-Fyne-Client-Rewire.md) | Client | High | 106, 108, 109, 110 |
| 111 | [Sidecar Supervisor](Feature-111-Sidecar-Supervisor.md) | Process Mgmt | Medium | 107, 110 |
| 112 | [UI Single-Instance Lock](Feature-112-UI-Single-Instance-Lock.md) | Process Mgmt | Low | 107 |

## Suggested Implementation Order

```
097 (infrastructure) ─────────────────────┐
    ↓                                     │
098 (notifications) ──┐                   │
099 (event stream) ───┤ can be parallel   │
100 (buffer) ─────────┘                   │
    ↓                                     │
099A (server orchestrator wiring) ─── unblocks 102, 104, 107
    ↓                                     │
103 (rules) ──────────┐                   │
102 (config) ─────────┘ can be parallel   │
    ↓                                     │
101 (planner) ────────── most complex     │
104 (timer) ──────────── depends on 099   │
    ↓                                     │
108 (TOFU pairing) ───── depends on 097, 099
    ↓                                     │
106 (client SDK) ─────── depends on all + 108
106A (AsyncAPI docs) ──── can parallel 106 │
    ↓                                     │
109 (todo restructure) ─ depends on 101A, 102, 106
    ↓                                     │
110 (TOFU client bootstrap) ── depends on 106, 108
    ↓                                     │
107 (Fyne re-wire) ───── depends on 106, 108, 109, 110
    ↓                                     │
111 (sidecar) ───── depends on 107, 110   │
112 (UI lock) ───── depends on 107  ── 111 + 112 can be parallel
```

**Critical path:** 097 → 098/099/100 → 101. The planner API depends on the most infrastructure and is the most complex — save it for last when patterns are established.

## Cross-Cutting Questions

These questions affect multiple features and should be resolved before implementation begins:

### 1. Authentication Model
Even for localhost, should the server require a bearer token? This matters if:
- A mobile UI connects over LAN
- Multiple users share a machine
- A web UI runs on a different port (CORS + auth)

**Recommendation:** No auth for v1 (localhost-only binding). Add optional token auth in a follow-up when LAN access is needed.

### 2. API Versioning
`/api/v1/...` from day one. Cheap insurance against breaking changes.

### 3. JSON Conventions
Decide once, enforce everywhere:
- Field names: `snake_case`
- Dates: RFC 3339
- UUIDs: Standard hyphenated
- Nulls: Omit (`omitempty`)
- Errors: `{"error": {"code": "NOT_FOUND", "message": "..."}}`

### 4. Testing Strategy
Each feature needs:
- Unit tests for handlers (mock repositories/services)
- Integration tests against real SQLite + HTTP server on ephemeral port
- WebSocket tests with real connections

### 5. Documentation
Consider auto-generating OpenAPI/Swagger from handler definitions or annotations. This gives alternative UI developers a machine-readable API contract.

## Not In Scope (Future)

- Todo CRUD API (separate feature — planner reads todos but doesn't manage them)
- Calendar event preview API
- Webhook/callback support (server pushes to external URL)
- Multi-user / authentication
- TLS termination (use a reverse proxy)
- API rate limiting
