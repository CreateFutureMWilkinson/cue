# Feature 054: Audio Alert Wiring

**Phase:** Phase-6-Feature-054
**Type:** Bugfix
**Severity:** Critical
**Status:** Planned
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

`cmd/cue/main.go:175-190` — `alertSvc` is created and configured but never passed to the orchestrator or connected via callback. The orchestrator has no reference to the alert service.

## Proposed Fix

Wire the alert service into the orchestrator's post-routing pipeline. The orchestrator already iterates over routed messages to insert them — add a callback or direct dependency that triggers `alertSvc.PlayNotification()` for each message with status NOTIFIED.

Two approaches:

**Option A (callback):** Add an `OnNotified func()` callback to the orchestrator. Set it to `alertSvc.PlayNotification` in `main.go`.

**Option B (interface injection):** Define a minimal `Alerter` interface (`PlayNotification()`) and inject it into the orchestrator constructor.

Option A is simpler and avoids adding a new interface for a single method.

## Test Strategy

- RED: Test that the orchestrator calls the `OnNotified` callback when a message is routed as NOTIFIED
- RED: Test that the callback is NOT called for BUFFERED or IGNORED messages
- GREEN: Implement the callback wiring
- REFACTOR: Clean up
