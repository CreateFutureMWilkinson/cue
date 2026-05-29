# Feature 114: Character Animation Wiring

**Phase:** Phase-9-Feature-114
**Status:** Planning
**Depends on:** Feature 099 (Activity Event Stream), Feature 107 (Fyne Client Re-wire)
**Blocks:** —
**Packages:** `internal/ui/presenter/`, `internal/ui/character/`, `internal/ui/character/fairy/`, `cmd/cue/adapters/`, `cmd/cue/main.go`

---

## Overview

The `FairyCharacter` plugin defines six animation states (`StateIdle`, `StateStarting`, `StateWorking`, `StateNotifying`, `StateError`, `StateShuttingDown`), each with a working animator. In production today only `StateWorking` and `StateError` ever fire, and the idle breathing animation is never visible. Three of the six states (`StateStarting`, `StateNotifying`, `StateShuttingDown`) are inert despite having functional animators behind them.

This feature rewires `CharacterPresenter` and the lifecycle of `cmd/cue/main.go` so that every state surfaces in the UI under the conditions the user actually cares about.

---

## Problem Statement

`internal/ui/presenter/character_presenter.go:70-78` maps activity events to states using two checks: `IsError → StateError`, `strings.Contains(message, "NOTIFIED") → StateNotifying`, else `StateWorking`. A 5-second decay timer (`character_presenter.go:88-90`) returns the character to `StateIdle`.

Concrete defects:

1. **`StateNotifying` never fires.** The orchestrator emits `"rules: N notified, M ignored, …"` (lowercase `notified`); the presenter's `strings.Contains` is case-sensitive on `"NOTIFIED"`. The `alert` envelope, which already publishes on every NOTIFIED message (`internal/server/alerter.go:28`), is not consumed by the presenter at all.
2. **`StateStarting` never fires.** `FairyCharacter` is *constructed* in `StateStarting` (`internal/ui/character/fairy/fairy.go:107`) but no animator is attached at construction — `TransitionTo` is what builds and starts the animator. Nothing in `cmd/cue/main.go` calls `TransitionTo(StateStarting)` after creating the character.
3. **`StateIdle` is rarely visible.** Two compounding causes: (a) the bootstrap path never reaches Idle because `StateStarting`'s completion callback (which would chain to `TransitionTo(StateIdle)`, see `fairy.go:178-180`) never runs since the startup animator is never started; (b) the queue-depth heartbeat event (`internal/service/orchestrator/orchestrator.go:232-234`) fires on every orchestrator tick and currently maps to `StateWorking`, resetting the 5s decay timer indefinitely under normal load.
4. **`StateShuttingDown` never fires.** `FairyCharacter.Shutdown()` exists (`fairy.go:250-261`) and returns a done channel for the shutdown animator, but it is not part of the `Character` interface and is never called during app teardown.

`StateWorking` is fine but coarse — it fires only after a fetch succeeds, not while the watcher is actively polling or while the queue processor is mid-Ollama call. Tightening that is **out of scope** for this feature; tracked separately.

---

## Locked Decisions

### 1. Envelope-typed presenter (Q1a)

`CharacterPresenter` will subscribe to two independent sources from `cmd/cue/adapters/activity.go`:

- `ActivitySource` — for activity envelopes (`fetched`, `rules`, `import`, errors).
- `AlertSource` — new, for alert envelopes (`alerter.go` publishes one per NOTIFIED message).

Rationale:
- Matches the on-the-wire envelope split (`internal/server/hub.go`).
- Keeps the alert path independent of activity for future consumers (the audio alerter already consumes alerts separately on its own subscription — that path is unchanged).
- No tagged-union DTO needed in the presenter package.

The string-sniffing for `"NOTIFIED"` is removed entirely. Notifying is driven by the alert envelope only.

State mapping after rewrite:

| Trigger | State |
|---|---|
| `alert` envelope received | `StateNotifying` |
| `activity` envelope, `IsError=true` | `StateError` |
| `activity` envelope, otherwise (after heartbeat filter) | `StateWorking` |
| 5s decay (no event) | `StateIdle` |

### 2. Heartbeat filter (Q2)

Activity envelopes matching either of the following are ignored by `CharacterPresenter` — they neither change state nor reset the decay timer:

- `Source == "queue"` AND message starts with `"Ollama queue depth"` (matches both the warning variant `"⚠ Ollama queue depth: N — consider …"` and the plain variant `"Ollama queue depth: N"`).
- `Source == "system"` AND message equals `"No watchers configured"`.

Import-progress events (`"importing X: N messages..."`, `"import complete: N new from X"`) are **not** filtered — they map to `StateWorking` like any other activity. Imports are real work and the user should see the fairy animate during them.

The filter lives in `CharacterPresenter` (not in the orchestrator) so that other consumers — the activity-log drawer in particular — continue to receive every event.

### 3. Boot-time `StateStarting`

In `cmd/cue/main.go`, immediately after `character.Create(...)` and `SetRefreshHook(...)`, call `char.TransitionTo(character.StateStarting)`. The `StartupAnimator`'s completion callback (`fairy.go:178-180`) already chains to `TransitionTo(StateIdle)`, so this single call fixes both the visible startup animation and the missing initial idle pulse.

### 4. `Character.Shutdown()` on the interface

Add `Shutdown() <-chan struct{}` to the `Character` interface in `internal/ui/character/character.go`. Implementations:

- `*FairyCharacter` — already has a method matching the signature (`fairy.go:250-261`); no code change beyond the interface assertion.
- `*NoopCharacter` (`internal/ui/character/noop.go`) — return a pre-closed channel.
- WASM host (`internal/ui/character/wasmhost/`) — call `plugin_transition_to(StateShuttingDown)` and return a closed channel (no plugin-side completion signal exists yet; deferred).

In `cmd/cue/main.go`'s shutdown sequence (the path that currently closes the orchestrator and `eventCh`), call `char.Shutdown()` and wait on its done channel **before** the Fyne window is torn down. A reasonable ceiling (e.g. 2s) on the wait is acceptable to avoid blocking shutdown if an animator misbehaves — confirm timing with the existing `ShutdownAnimator` duration when implementing.

### 5. Backwards-compat is not a concern

There are no external consumers of `CharacterPresenter` or the `Character` interface outside this repo. Old shapes are removed in the same commit they're replaced. No deprecation shims.

---

## Behaviour Decomposition (TDD micro-loops)

Each behaviour follows the standard RED → GREEN → REFACTOR cycle. UI acceptance tests are updated **before** any micro-loop begins, per the UI Feature Workflow in `CLAUDE.md`.

### 0. UI test update (pre-loop)

Update `cmd/cue/uat.go` (or whichever UI test file owns character assertions) to assert:

- After app boot, the character reaches `StateIdle` (via `StateStarting` → completion → Idle) without any external event.
- An `alert` envelope causes the character to transition to `StateNotifying`.
- A queue-depth activity envelope does NOT change state from Idle.
- An app-shutdown trigger transitions the character through `StateShuttingDown` before window teardown.

Commit: `test(ui): failing tests for character lifecycle states`

### 1. `CharacterPresenter` accepts an `AlertSource`

- New `AlertSource` interface in `internal/ui/presenter/interfaces.go` (`Events() <-chan AlertEvent`).
- `CharacterPresenter` constructor takes `(char, ActivitySource, AlertSource, decayDuration)`.
- The event loop selects on both channels.
- Alert event → `TransitionTo(StateNotifying)` + reset decay.

### 2. Remove `"NOTIFIED"` string sniffing

- `mapEventToState` becomes activity-only: `IsError → StateError`, else `StateWorking`.
- Remove the `notifiedKeyword` constant.

### 3. Heartbeat filter

- New unexported predicate `isHeartbeat(event ActivityEvent) bool`.
- `Source=="queue" && strings.HasPrefix(message, "Ollama queue depth")` → true.
- `Source=="system" && message == "No watchers configured"` → true.
- Filtered events: skip both the state transition AND the decay reset.

### 4. `ActivityAdapter` exposes `AlertSource`

- Add `Subscribe()` for the alert envelope path, mirroring the activity subscription.
- Existing audio-alerter wiring is unchanged (it has its own subscriber on the hub side; this is the UI-side adapter).

### 5. Boot wires `StateStarting`

- `cmd/cue/main.go`: after character creation and refresh-hook installation, call `TransitionTo(StateStarting)`.
- Implementer must verify the startup animator chains to Idle as expected (already covered by existing `startup_animator_test.go`).

### 6. `Character.Shutdown()` on interface + main wiring

- Add `Shutdown() <-chan struct{}` to `Character` interface.
- `NoopCharacter.Shutdown` returns a pre-closed channel.
- WASM host `Shutdown` triggers `StateShuttingDown` transition and returns closed channel.
- `cmd/cue/main.go` shutdown path: `<-char.Shutdown()` (with a `select` + `time.After` ceiling) before window close.

---

## Wiring Verification (per CLAUDE.md §17)

After all micro-loops, verify:

1. No `ErrNotImplemented` introduced in production code.
2. `CharacterPresenter` constructor is called with both `ActivitySource` and `AlertSource` in `cmd/cue/main.go`.
3. `ActivityAdapter` actually publishes alert envelopes to the new alert source (not just a defined-but-unsubscribed channel).
4. `char.Shutdown()` is called on the app's shutdown path and awaited.
5. `TransitionTo(StateStarting)` is reached on boot for both Fyne and headless UAT entry points.

Run `just test`, `just test-ui`, `just security`, `just vulncheck`, `just fmt`, `just lint`, `just tidy`.

---

## Test Coverage Targets

- `internal/ui/presenter/character_presenter_test.go` — cover all six branch outcomes (alert→Notifying, activity error→Error, activity normal→Working, heartbeat→ignored, decay→Idle, no-events→stays in initial state).
- `cmd/cue/adapters/activity_test.go` — cover alert envelope demultiplex into `AlertSource`.
- `internal/ui/character/noop_test.go` — `Shutdown` returns a closed channel.
- UI acceptance tests as per section 0 above.

Coverage gate: ≥ 80%.

---

## Out of Scope

- **Tightening `StateWorking`** to fire during in-flight polls and Ollama calls (currently fires on `"fetched"` post-hoc). Tracked as a follow-up; the orchestrator and queue processor would emit start-of-work events.
- **WASM-plugin shutdown completion signal.** Plugins currently have no `plugin_shutdown_done()` export; the host returns a pre-closed channel. A real signal is a separate effort if we adopt non-fairy plugin characters that need it.
- **Decay-window tuning.** 5s remains for now. If Working flashes feel too brief or too sticky after this work lands, revisit.

---

## TDD Agent Stats

(To be filled in during implementation — one row per RED/GREEN/REFACTOR phase.)

| Implementation Phase | TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| | | | | | |
