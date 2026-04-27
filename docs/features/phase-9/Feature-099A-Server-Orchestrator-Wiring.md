# Feature 099A: Server Orchestrator Wiring

**Phase:** Phase-9-Feature-099A
**Status:** Planning (not started)
**Package:** `cmd/cue-server/`, `internal/server/` (composition root)
**Depends on:** Feature 097 (Server Infrastructure), Feature 099 (Activity Event Stream)
**Blocks:** Feature 102 (Service Configuration API), Feature 107 (Fyne Client Re-wire)

---

## Overview

Relocate the orchestrator (and everything it depends on — watchers, queue processor, rules engine, Ollama client, buffer service, vector store) from `cmd/cue/main.go` into `cmd/cue-server/main.go`, and wire the orchestrator's activity event channel into the WebSocket hub's `Publish` method added in Feature 099.

This feature closes a gap left by Feature 097, which scaffolded the `cue-server` binary but explicitly punted the orchestrator / service / watcher wiring to "Feature 098+". That wiring never landed in 098 (Messages), 099 (Event Stream protocol), or anywhere else. Feature 102 (Service Configuration API) assumes the orchestrator is already running inside `cue-server` — it can't be built until 099A ships.

## Motivation

Three concrete consumers need this relocation:

1. **Feature 099 Event Stream** — the hub exists and can broadcast, but no events flow through it until a publisher is attached. After 099A, `orchestrator.ActivityEvent`s reach WebSocket clients end-to-end.
2. **Feature 102 Service Configuration API** — adding/removing a Slack or Email account must register/deregister a watcher with the orchestrator at runtime. The handler needs a live orchestrator reference.
3. **Feature 107 Fyne Client Re-wire** — turns `cmd/cue` into a thin client. All state (DB, orchestrator, services, watchers) must live in `cue-server` before 107 can proceed.

## Current State (as of Feature 099)

| Component | `cmd/cue` (GUI) | `cmd/cue-server` (headless) |
|---|---|---|
| Config load + validate | ✅ | ✅ |
| SQLite message repo | ✅ | ✅ (added in 098) |
| SQLite other repos (rules, accounts, todos, etc.) | ✅ | ❌ |
| Ollama client | ✅ | ❌ |
| Buffer service | ✅ | ❌ |
| Vector store | ✅ | ❌ |
| Queue processor | ✅ | ❌ |
| Rules engine | ✅ | ❌ |
| Orchestrator | ✅ | ❌ |
| Watcher construction from DB | ✅ | ❌ |
| `hub.Publish` calls | — | ❌ (hub exists, no publisher) |
| HTTP handlers (health, messages, notifications) | — | ✅ |
| WebSocket upgrade handler | — | ✅ (added in 099) |

## Scope

### In Scope

1. **Move repository construction** — all repos currently built in `cmd/cue/main.go` (messages, rules, service accounts, todos, etc.) move to `cmd/cue-server/main.go`. `cmd/cue` loses the imports.
2. **Move service construction** — ollama client, buffer service, vector store, queue processor, rules engine.
3. **Move orchestrator construction + `Start()` / `Stop()` lifecycle**.
4. **Move watcher construction from DB** (`buildWatchersFromDB` helper) — it belongs wherever the orchestrator lives.
5. **Wire orchestrator → hub** — replace the current `bridgeEvents` fan-out with one that also calls `srv.Hub().Publish(...)` for every event. `cmd/cue` keeps the presenter fan-out it still needs (until Feature 107 replaces it entirely).
6. **Graceful shutdown ordering** — in `cue-server`: stop orchestrator, drain queue, close repos, shut down HTTP server, close hub subscribers. Extend the existing SIGINT/SIGTERM handler.
7. **Split config validation** — `cue-server` starts requiring `[database]`, `[ollama]`, `[orchestrator]` to be valid. `cmd/cue` keeps validating its own sections (for now; tightened further in Feature 107).

### Out of Scope

- Thinning `cmd/cue` into an HTTP client (that's Feature 107).
- Changing the activity event shape, hub API, or WebSocket protocol (locked in Feature 099).
- Any new REST endpoints (102 onward).
- Hot-reload of service accounts (part of 102's behaviour, not here).

## Open Design Questions

These need answers before TDD loops start. They're collected here rather than in the Feature 099 doc because they only concern the relocation, not the protocol.

1. **Transitional dual-wiring.** During the relocation, does `cmd/cue` still have its own orchestrator (running alongside the one in `cue-server`), or does it start depending on `cue-server` immediately? The cleanest path is probably: the orchestrator moves in one commit, and `cmd/cue` temporarily gains a no-op `ActivitySource` so its presenters still compile until Feature 107 replaces them with WebSocket adapters. Alternative: `cmd/cue` continues to create its own orchestrator until 107 lands, and 099A's hub publisher wiring is duplicated in both binaries via a shared helper. Decision deferred.
2. **How does `cmd/cue` get activity events during the transition?** If the orchestrator leaves `cmd/cue`, presenter fan-out needs a new source. Options: (a) temporary in-process no-op; (b) `cmd/cue` connects to its own `cue-server` over WebSocket (prematurely bringing 107 forward); (c) keep the orchestrator in both binaries with duplicated wiring.
3. **Single DB or two DB handles?** If both binaries run and both open the SQLite DB, WAL contention is possible. Mitigation: `cmd/cue` should not open the DB at all after 099A; it relies on `cue-server`'s HTTP APIs for reads, even transiently via temporary adapters.
4. **Startup ordering.** Does `cue-server` need to be running before `cmd/cue` starts, or can they start in any order? Feature 107 assumes server-first-then-client; 099A should make that the supported topology.
5. **Graceful shutdown interleaving.** `cue-server` now owns the orchestrator's `Stop()` — what's the shutdown order relative to draining HTTP requests and closing WebSocket connections? Stopping the orchestrator first means in-flight polls complete before HTTP shutdown; this is probably correct but worth confirming.
6. **Testing strategy.** Integration tests that exercise orchestrator → hub → WebSocket require a real `cue-server` subprocess or an in-process `httptest.NewServer` wrapping the full composition. Decide which, and whether `just test` or `just test-ui` is the right gate.

## Proposed Behaviors (sketch)

To be refined after the open questions above are answered. Rough outline:

1. `cue-server` main opens all repos currently owned by `cmd/cue`.
2. `cue-server` main constructs ollama client, buffer service, vector store, queue processor, rules engine.
3. `cue-server` main constructs orchestrator and starts it.
4. `cue-server` main wires `orchEventCh` → `hub.Publish` for every event (type `"activity"`, data = `ActivityData{Source, Message, IsError}`).
5. `cue-server` main calls `buildWatchersFromDB` against the live orchestrator.
6. `cue-server` graceful shutdown: stop orchestrator → drain queue → shut HTTP → close WebSocket connections → close repos.
7. `cmd/cue` loses the moved construction code. Transitional: either a no-op `ActivitySource` stub or (depending on Q1) keeps a parallel orchestrator.
8. End-to-end integration test: start `cue-server` in-process, connect a WebSocket client, trigger an orchestrator event, assert the client receives a correctly-shaped envelope.

## Acceptance Criteria

- `cue-server` can be started standalone and runs the orchestrator loop.
- WebSocket clients connected to `cue-server` receive activity envelopes in real time.
- `cmd/cue` still builds and runs (it may have a transient no-op activity source until Feature 107 lands).
- No DB is opened by both binaries simultaneously.
- `just test` and `just test-ui` remain green.

## Risks

- **Large diff.** This is a substantial refactor touching both binaries and most of the composition root. Should be landed on a dedicated branch with careful review.
- **Transitional UX regression.** If `cmd/cue` temporarily loses its live activity stream during the window between 099A and 107, the user may see a stale activity log. Mitigate by making 107 the next feature after 099A, or by accepting the gap as documented.
- **Shutdown complexity.** The shutdown order matters — wrong ordering leaks goroutines or loses in-flight events. Needs explicit test coverage.
