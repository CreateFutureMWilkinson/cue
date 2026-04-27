# Feature 108: TOFU Pairing Authentication

**Phase:** Phase-9-Feature-108
**Status:** Planning
**Depends on:** Feature 097 (server infrastructure), Feature 099 (WebSocket event stream)
**Blocks:** Feature 106 (API Client SDK), Feature 106A (AsyncAPI Documentation), Feature 107 (Fyne Client Re-wire)
**Package:** `internal/server/auth/`

---

## Overview

Implement Trust-On-First-Use (TOFU) authentication for `cue-server`. The first client to connect when no tokens exist is automatically issued a long-lived bearer token. Subsequent clients trigger a pairing prompt pushed to all already-connected clients via WebSocket; the user approves or denies. This secures the server without requiring manual credential setup, which aligns with Cue's local-first, low-friction philosophy.

The authentication model was specified in Feature 096 (Server Protocol Selection, Decision 3). This feature implements that specification.

## Design Decisions

### 1. Token Storage — SQLite Table

**Decision: `auth_tokens` table in the existing SQLite database.**

```sql
CREATE TABLE auth_tokens (
    id          TEXT PRIMARY KEY,           -- UUID
    label       TEXT NOT NULL DEFAULT '',   -- User-assigned device name ("desktop", "phone")
    token_hash  TEXT NOT NULL UNIQUE,       -- SHA-256 hash of the bearer token (never store plaintext)
    created_at  TEXT NOT NULL,              -- RFC 3339
    last_seen   TEXT NOT NULL,              -- RFC 3339, updated on each authenticated request
    revoked     INTEGER NOT NULL DEFAULT 0  -- 0 = active, 1 = revoked
);
```

Tokens are hashed before storage — the plaintext bearer token is returned exactly once at creation time. `last_seen` is updated at most once per minute to avoid write amplification on every request.

### 2. Auth Middleware

**Decision: stdlib `http.Handler` middleware wrapping all `/api/v1/` routes.**

```
Request → AuthMiddleware → Route Handler
                ↓
          Check Authorization header
          "Bearer <token>"
                ↓
          Hash token, lookup in auth_tokens
                ↓
          If valid + not revoked → update last_seen, proceed
          If invalid/missing → check if pairing or first-client flow
```

**Exempt routes** (no auth required):
- `GET /health`, `GET /health/ready` — load balancer probes
- `GET /api/v1/health`, `GET /api/v1/health/ready` — client health checks
- `POST /api/v1/auth/pair` — pairing request endpoint (by definition, unauthenticated)

### 3. First-Client Auto-Issue

**Decision: If zero non-revoked tokens exist, the next request to any authenticated endpoint auto-issues a token.**

Flow:
1. Client sends request with no `Authorization` header (or invalid token)
2. Middleware checks `auth_tokens` table: zero active tokens exist
3. Server generates a new token, hashes and stores it, responds with `401` + a JSON body containing:
   ```json
   {
     "error": {
       "code": "TOKEN_ISSUED",
       "message": "First client — token auto-issued"
     },
     "token": "<plaintext-bearer-token>"
   }
   ```
4. Client stores the token and retries the original request

**Why 401 instead of 200?** The original request was not authenticated — returning 200 with the requested data plus a surprise token would conflate two concerns. The client must explicitly acknowledge the token and retry. This also means every successful response has a predictable shape (no extra `token` field).

### 4. Pairing Flow (Subsequent Clients)

**Decision: WebSocket-based pairing prompt with 60-second timeout. Pairing requests are stored in-memory only (not persisted to SQLite) — they have a 60s lifetime and are lost on server restart, which is acceptable.**

Flow:
1. Unauthenticated client sends `POST /api/v1/auth/pair` with an optional `label` (device name)
2. Server generates a 6-digit numeric pairing code (0-9 only, displayed to user on both ends)
3. Server pushes a `pairing_request` event to all authenticated WebSocket clients:
   ```json
   {
     "type": "pairing_request",
     "data": {
       "request_id": "<uuid>",
       "label": "phone",
       "code": "472859",
       "expires_at": "2026-04-23T12:01:00Z"
     }
   }
   ```
4. The requesting client receives a `202 Accepted` response with the same `request_id` and `code`, then polls `GET /api/v1/auth/pair/{request_id}` for the result
5. An authenticated client approves via `POST /api/v1/auth/pair/{request_id}/approve` or denies via `POST /api/v1/auth/pair/{request_id}/deny`
6. On approval: server generates a new token, responds to the next poll with:
   ```json
   {
     "status": "approved",
     "token": "<plaintext-bearer-token>"
   }
   ```
7. On denial or timeout (60s): next poll returns `{"status": "denied"}` or `{"status": "expired"}`

**Why polling instead of WebSocket for the requesting client?** The requesting client doesn't have a token yet, so it can't authenticate a WebSocket connection. A simple poll loop on a single endpoint is simpler than an unauthenticated-but-scoped WebSocket channel.

### 5. Token Revocation

**Decision: Soft-delete via `revoked` flag. No self-revocation guard.**

`DELETE /api/v1/auth/tokens/{id}` sets `revoked = 1`. The token remains in the table for audit purposes. Auth middleware rejects revoked tokens with `401`.

Any client can revoke any token (including its own). The UI is responsible for warning users about self-revocation if desired. No separate `/auth/logout` endpoint — `DELETE /auth/tokens/{id}` covers all cases.

### 5A. Auth Reset (Recovery)

**Decision: `--reset-auth` CLI flag on `cue-server`.**

`cue-server --reset-auth` deletes all rows from `auth_tokens` and exits. This is a direct database operation — the server does not start. This provides a recovery path when a user has lost access to all devices with valid tokens (e.g., single token on a device they no longer have).

### 6. WebSocket Authentication

**Decision: Token in query parameter on upgrade.**

```
GET /api/v1/websocket/events?token=<bearer-token>
```

**Why not a header?** Many WebSocket client implementations (browser `WebSocket` API, some mobile libs) don't support custom headers on the upgrade request. Query parameter is universally supported.

The token is validated during the HTTP upgrade handshake. If invalid, the server responds with `401` before upgrading — no WebSocket connection is established.

### 7. Dev Mode

**Decision: Single `auth_enabled` bool on `ServerConfig`, defaults to `true`.**

```toml
[server]
port = 9400
auth_enabled = true  # set to false for development/testing
```

When `auth_enabled = false`, the auth middleware passes all requests through without checking tokens. WebSocket connections also skip token validation. This is the current behavior and preserves backward compatibility for development.

**All existing tests continue to pass unchanged** — they run with auth disabled.

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/v1/auth/pair` | No | Initiate pairing request |
| `GET` | `/api/v1/auth/pair/{id}` | No | Poll pairing result |
| `POST` | `/api/v1/auth/pair/{id}/approve` | Yes | Approve pairing request |
| `POST` | `/api/v1/auth/pair/{id}/deny` | Yes | Deny pairing request |
| `GET` | `/api/v1/auth/tokens` | Yes | List all tokens (id, label, created, last_seen, revoked) |
| `PUT` | `/api/v1/auth/tokens/{id}` | Yes | Update token label |
| `DELETE` | `/api/v1/auth/tokens/{id}` | Yes | Revoke a token |

**CLI flag:**

| Flag | Description |
|------|-------------|
| `--reset-auth` | Delete all auth tokens from the database and exit. Recovery for lockout scenarios. |

## WebSocket Event Types

| Type | Direction | Payload |
|------|-----------|---------|
| `pairing_request` | server → client | `{request_id, label, code, expires_at}` |
| `pairing_resolved` | server → client | `{request_id, status}` — notifies connected clients when pairing completes |

## TDD Behaviors

| # | Behavior | Test Strategy |
|---|----------|---------------|
| 1 | Auth token repository CRUD (create, lookup by hash, list, revoke, update label, delete all) | SQLite test with `s.T().TempDir()` |
| 2 | Auth token repository CountActive (returns count of non-revoked tokens) | SQLite test |
| 3 | Auth token repository UpdateLastSeen | SQLite test |
| 4 | Auth middleware — valid token passes through | `httptest.NewServer` with middleware, mock repo |
| 5 | Auth middleware — missing/invalid/revoked token returns 401 | Same |
| 6 | Auth middleware — exempt routes bypass auth | Same (health + pair endpoints) |
| 7 | First-client auto-issue — zero active tokens, auto-issues new token | Mock repo returning count=0 |
| 8 | Auth middleware — last_seen throttle (per-token, once per minute) | Clock mock |
| 9 | Dev mode — auth disabled passes all requests | Config toggle |
| 10 | Pairing initiation — POST /auth/pair generates 6-digit code, stores in-memory, broadcasts to hub | Mock hub + in-memory store |
| 11 | Pairing poll — GET /auth/pair/{id} returns pending/approved/denied/expired | In-memory store |
| 12 | Pairing approval — POST /auth/pair/{id}/approve issues token, returns to poller | Mock repo + in-memory store |
| 13 | WebSocket auth — query param token validated on upgrade, rejected if invalid | WS test server |
| 14 | `--reset-auth` flag — deletes all tokens from DB and exits | Opens repo directly, calls DeleteAll |

## Dependencies

- No new external dependencies. Uses `crypto/sha256` (stdlib) for token hashing, `crypto/rand` for token generation.

## Success Criteria

- First client connecting to a fresh server receives a bearer token automatically
- Subsequent clients can pair via approval from an existing client
- All `/api/v1/` endpoints require authentication (except exempt routes)
- Revoked tokens are immediately rejected
- Existing tests pass unchanged with `auth_enabled = false`
- WebSocket connections require valid token in query parameter
- Token plaintext is never stored; only SHA-256 hashes persist
- `cue-server --reset-auth` deletes all tokens and exits (lockout recovery)
- Pairing codes are 6-digit numeric
- `last_seen` updated at most once per minute per token
