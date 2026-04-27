# Feature 096: Server Protocol Selection

**Phase:** Phase-9-Feature-096
**Status:** Planning
**Package:** N/A (architectural decision record)

---

## Overview

Before building the cue-server binary, the team must choose the transport protocol(s) that will underpin all API communication. This decision affects every subsequent feature in Phase 9 and constrains what alternative UIs are practical to build. This document compares the four candidate approaches and makes a recommendation.

## Candidates

### 1. REST + Polling

**How it works:** Standard HTTP endpoints returning JSON. Clients poll at intervals for updates (notifications, timer state, activity events).

**Strengths:**
- Universally understood; trivial to consume from any language or platform
- Stateless server — easy to reason about, test, and debug
- Works through every proxy, firewall, and CDN
- `curl`-debuggable from day one
- Excellent Go library ecosystem (Chi, Echo, stdlib `net/http`)

**Weaknesses:**
- Polling wastes bandwidth and adds latency for real-time data (activity events, timer ticks)
- Chatty for the planner wizard — each step requires a round trip
- No server-push: client can't know about new notifications until next poll
- Poll interval is a latency/load tradeoff with no good universal answer

**Best fit:** Admin/config CRUD, infrequent reads, simple clients, CLI tooling.

### 2. REST + WebSocket Hybrid

**How it works:** REST endpoints for request/response operations (CRUD, planner steps). A WebSocket connection for server-pushed events (activity log, timer ticks, new notifications).

**Strengths:**
- REST for CRUD keeps things simple and cacheable
- WebSocket for events eliminates polling latency entirely
- Well-supported in browsers, mobile, and terminal UIs
- Matches Cue's existing event architecture (fan-out channels map directly to WS broadcast)
- Go stdlib + `nhooyr.io/websocket` or `gorilla/websocket` are mature

**Weaknesses:**
- Two protocols to implement, test, and document
- WebSocket connections are stateful — need reconnection logic, heartbeats
- Slightly more complex client implementation than pure REST
- WebSocket doesn't work through some corporate proxies (rare but real)

**Best fit:** Applications with both CRUD operations AND real-time event streams — which is exactly Cue's profile.

### 3. gRPC + Server Streaming

**How it works:** Protocol Buffers define the API contract. Unary RPCs for CRUD, server-streaming RPCs for events.

**Strengths:**
- Strongly typed contract (`.proto` files) with codegen for Go, TypeScript, Dart, etc.
- Server streaming is first-class — no bolt-on WebSocket needed
- Binary encoding is compact and fast
- Bidirectional streaming available if needed later
- Built-in deadline/cancellation via context

**Weaknesses:**
- Not browser-native — requires grpc-web proxy or Connect protocol for web UIs
- Harder to debug (`grpcurl` exists but is less intuitive than `curl`)
- Heavier dependency footprint (`google.golang.org/grpc`, protoc toolchain)
- Proto file maintenance is overhead for a small team
- Mobile support varies (good on Android/iOS via official libs, awkward in Flutter/React Native)

**Best fit:** Service-to-service communication, polyglot teams with codegen pipelines, high-throughput systems.

### 4. GraphQL

**How it works:** Single endpoint, clients specify exactly the data shape they need. Subscriptions (over WebSocket) for real-time events.

**Strengths:**
- Clients fetch exactly what they need — no over/under-fetching
- Single endpoint simplifies routing
- Subscriptions cover real-time use case
- Self-documenting schema with introspection
- Good fit if multiple UIs need very different data shapes

**Weaknesses:**
- Significant server complexity (resolver graph, N+1 query problem, authorization per field)
- Go GraphQL libraries (`gqlgen`, `graphql-go`) are less mature than REST/gRPC tooling
- Overkill for Cue's data model — entities are simple, relationships are shallow
- Subscription implementations in Go are less battle-tested
- Harder to cache than REST
- Steeper learning curve for contributors

**Best fit:** APIs serving many diverse clients with varying data needs, complex entity graphs.

## Recommendation: REST + WebSocket Hybrid

**Primary reasons:**

1. **Matches the architecture.** Cue already has two distinct data flow patterns: request/response (query messages, save rating, CRUD config) and event streams (activity log, timer ticks, notification arrivals). REST handles the first; WebSocket handles the second. Forcing both through one protocol means compromising on one.

2. **Lowest barrier for alternative UIs.** A web frontend, a terminal TUI, a mobile app, or even a simple shell script can all consume REST+WS with zero codegen or special tooling. This maximizes the audience for alternative UI development.

3. **Cue's data model is simple.** There are ~6 entity types with shallow relationships. GraphQL's flexibility and gRPC's type codegen don't pay for themselves at this scale.

4. **Event fan-out already exists.** The `bridgeEvents()` pattern in `main.go` already fans out `ActivityEvent` to multiple channel consumers. Adding a WebSocket broadcaster is a natural extension — not a new pattern.

5. **Debuggability matters for a small project.** Being able to `curl` an endpoint or open a WebSocket in browser devtools dramatically reduces the feedback loop during development.

**Libraries:**
- HTTP router: `net/http` stdlib (Go 1.22+ pattern matching)
- WebSocket: `github.com/coder/websocket` (context-aware, stdlib-aligned, pure Go)
- JSON: `encoding/json` stdlib (consider `github.com/go-json-experiment/json` if perf matters later)

## Decisions

1. **HTTP router: stdlib `net/http`**. Go 1.22+ pattern matching (`GET /api/v1/foo`) is sufficient. No Chi dependency — keeps the dependency tree minimal and aligns with the project's pure-Go preference. Middleware can be composed with stdlib `http.Handler` wrappers.

2. **WebSocket library: `coder/websocket`**. Actively maintained fork of `nhooyr.io/websocket`, context-aware, stdlib-aligned, pure Go. No CGO.

3. **Authentication model: TOFU with pairing flow.**
   - Server binds `0.0.0.0` (all interfaces).
   - **First client** connects with no tokens in the system → automatically issued a long-lived bearer token. Assumption: first client is on the same machine.
   - **Subsequent clients** connecting without a valid token trigger a **pairing prompt** pushed to all already-connected clients via WebSocket. User approves or denies.
   - **Approval** generates a new long-lived token for the requesting client. **Denial** (or 60-second timeout) rejects the connection. A new pairing request can always be initiated.
   - **Token storage:** SQLite table — supports multiple named tokens (e.g. "desktop", "phone"), creation timestamps, and revocation.
   - **Connected devices schema:** tracks all issued tokens with device label, created/last-seen timestamps, and revoked flag. UI will expose a "connected devices" view for token revocation (not necessarily Phase 9, but schema supports it from day one).
   - All authenticated requests use `Authorization: Bearer <token>` header. WebSocket auth via token query parameter on upgrade (headers not reliably supported by all WS clients).

4. **API versioning: `/api/v1/...` from day one.** Cheap insurance. All endpoints are prefixed with `/api/v1/`. WebSocket path: `/api/v1/ws/events`.

5. **CORS policy: reflect origin.** Server echoes back the request's `Origin` header in `Access-Control-Allow-Origin`. TOFU token auth is the trust boundary, not the origin. This keeps things simple and avoids maintaining an allow-list as clients connect from different hosts/ports.

6. **WebSocket and HTTP: single shared port.** One `server.port` config value. WebSocket upgrades on `/api/v1/ws/events`. Simplifies config, firewall rules, and client setup.

---

## Next Features

This decision unblocks:
- Feature 097: Server Infrastructure + Composition Root
- Feature 098: Message & Notification API
- Feature 099: Activity Event Stream
- Feature 100: Feedback Buffer API
- Feature 101: Day Planner API
- Feature 102: Service Configuration API
- Feature 103: Routing Rules API
- Feature 104: Timer API
- Feature 105: Settings API
