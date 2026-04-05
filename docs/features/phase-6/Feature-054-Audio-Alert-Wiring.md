# Feature 054: Audio Alert Wiring

**Phase:** Phase-6-Feature-054
**Type:** Bugfix
**Severity:** Critical
**Status:** Done
**Packages:** `internal/service/orchestrator/`, `cmd/cue/`
**Related:** Feature 010 (Audio Alerts), Feature 012 (Configurable Audio Alerts), Feature 007 (Orchestrator)

---

## Bug Description

Audio alerts never fire when messages are routed as NOTIFIED. The `AlertService` is fully implemented and instantiated at startup in `main.go`, but nothing calls `alertSvc.PlayNotification()` when the orchestrator routes a message to NOTIFIED status.

## Expected Behavior

When the orchestrator routes a message and the router returns status=NOTIFIED, an audio alert should play (subject to the cooldown/mute window already implemented in `AlertService`).

## Actual Behavior

The `AlertService` instance exists but is completely inert at runtime. No code path invokes `PlayNotification()`.

## Root Cause

`cmd/cue/main.go` — `alertSvc` was created and configured but never passed to the orchestrator or connected via callback. The orchestrator had no reference to the alert service.

## Fix (Option B — Interface Injection)

Chose Option B: defined a minimal `Alerter` interface in the orchestrator package and injected the `AlertService` via the constructor.

### Design

1. **`Alerter` interface** (`orchestrator.go:27-30`) — single method `PlayNotification(ctx context.Context) error`
2. **Constructor injection** — `NewOrchestrator` accepts an `Alerter` parameter (nullable for backwards compatibility)
3. **Post-routing trigger** (`orchestrator.go:163-167`) — after counting routed messages, if `notified > 0` and alerter is non-nil, calls `PlayNotification`
4. **`main.go` wiring** — `alertSvc` passed directly to `NewOrchestrator`

### Error Handling

- Alert errors are non-fatal: logged as activity events, do not abort the poll cycle
- Nil alerter is safe: guarded by nil check, no panic

## Test Coverage

| Test | Behavior |
|---|---|
| `TestPollCycleTriggersAlertOnNotified` | Alert fires when batch contains NOTIFIED messages |
| `TestPollCycleNoAlertOnBufferedOnly` | Alert does NOT fire for BUFFERED-only batches |
| `TestPollCycleAlertErrorNonFatal` | Alert error does not crash or abort poll cycle |
| `TestPollCycleNilAlerterSafe` | Nil alerter does not panic on NOTIFIED messages |

All tests pass. Coverage includes the happy path, negative case, error path, and nil-safety.

## TDD Agent Stats

| TDD Phase | Agent | Commit |
|---|---|---|
| RED | Test Designer | `129c97b` |
| GREEN | Implementer | `a03bfa4` |
| REFACTOR | Refactorer | `5e71a54` |
