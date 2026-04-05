# Feature 10: Audio Alerts

**Phase:** Phase-1-Feature-10
**Status:** Done
**Package:** `internal/alert/`

---

## Overview

Cross-platform audio alert service using `github.com/gen2brain/beeep`. Plays sounds when messages are routed as NOTIFIED (sharp ping), on system startup (subtle chime), and on shutdown (lower tone). Includes a 2-second cooldown to prevent notification spam during batch processing. Integrates with the orchestrator as an optional dependency.

## Design Decisions

### Beeper Interface for Testability

The `beeep.Beep` function is a package-level call with no struct to mock. The `Beeper` interface wraps this, allowing tests to inject a `mockBeeper` that captures call arguments. Production code uses `NewBeeepBeeper()` to get the real implementation.

### Injectable Clock for Cooldown Testing

Rather than using `time.Sleep` in tests (flaky) or exposing internal state, the service accepts a `now` function (default `time.Now`) that tests override via `SetNowFunc`. This lets cooldown tests run deterministically without real time delays.

### Batch-Level Alerting

The orchestrator calls `PlayNotification` once per watcher batch that contains at least one NOTIFIED message, not once per message. This prevents audio spam when a batch contains multiple notifications — the 2-second cooldown provides additional protection.

### Optional Orchestrator Dependency

The `Alerter` interface is defined in the orchestrator package (consumer-side). The orchestrator accepts a nil alerter (no audio), matching the pattern used for optional dependencies elsewhere (e.g., `VectorEmbedder` in the buffer service).

### Non-Fatal Alert Errors

Alert failures never crash the system. The orchestrator logs alert errors as non-error activity events and continues processing. This aligns with the spec: "Low-priority background errors do NOT alert; only show in activity log."

## API

### Constructor

```go
func NewAlertService(cfg AlertConfig, beeper Beeper) (*AlertService, error)
```

`beeper` required (error if nil). `cfg.AudioEnabled` gates all sound output.

### Methods

```go
// Sharp ping (1000 Hz, 200ms) with 2-second cooldown. No-op when disabled.
func (a *AlertService) PlayNotification(ctx context.Context) error

// Subtle chime (600 Hz, 150ms). No-op when disabled.
func (a *AlertService) PlayStartup(ctx context.Context) error

// Lower tone (400 Hz, 300ms). No-op when disabled.
func (a *AlertService) PlayShutdown(ctx context.Context) error

// Inject clock function for testing cooldown behavior.
func (a *AlertService) SetNowFunc(fn func() time.Time)

// Create production beeper wrapping beeep.Beep.
func NewBeeepBeeper() Beeper
```

## Sound Parameters

| Alert | Frequency | Duration | Cooldown |
|---|---|---|---|
| Notification (NOTIFIED) | 1000 Hz | 200ms | 2 seconds |
| Startup | 600 Hz | 150ms | None |
| Shutdown | 400 Hz | 300ms | None |

## Error Handling

| Error | Action |
|---|---|
| Beeper returns error | Propagated to caller (orchestrator logs, continues) |
| Audio disabled in config | All methods return nil immediately |
| Within cooldown period | PlayNotification returns nil (skipped) |
| Nil alerter in orchestrator | No alert call, no panic |

## Integration Points

- **Orchestrator** (`internal/service/orchestrator/`) — Calls `Alerter.PlayNotification` after routing if any messages are NOTIFIED
- **Config** (`internal/config/`) — `NotificationConfig.AudioEnabled` controls on/off
- **Future: `cmd/cue/main.go`** — Will wire `NewAlertService(AlertConfig{AudioEnabled: cfg.Notification.AudioEnabled}, NewBeeepBeeper())` and pass to orchestrator

## Test Coverage Summary

15 tests total (11 alert + 4 orchestrator integration):
- Constructor validation (nil beeper rejected, valid deps accepted)
- PlayNotification: enabled/disabled, cooldown skip, cooldown expiry, beeper error propagation
- PlayStartup: enabled/disabled with correct frequency/duration
- PlayShutdown: enabled/disabled with correct frequency/duration
- Orchestrator: alert on NOTIFIED, no alert on BUFFERED-only, alert error non-fatal, nil alerter safe

## TDD Agent Stats

| TDD Phase | Agent | Commit |
|---|---|---|
| RED | Test Designer | 129c97b |
| GREEN | Implementer | a03bfa4 |
| REFACTOR | Refactorer | 5cce4eb |
