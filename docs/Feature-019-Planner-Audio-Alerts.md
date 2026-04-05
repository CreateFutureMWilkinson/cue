# Feature 019: Planner Audio Alerts

**Phase:** Phase-2-Feature-019
**Status:** Planned
**Packages:** `internal/alert/`, `internal/service/planner/`

---

## Overview

Audio alerts for the day planner's timer system. Plays a distinct sound when a focus or break block ends, with meeting-aware suppression. A single configurable audio file path in `config.toml` with a fallback beep that has a different tonality from the existing notification alert. Missed alerts during meetings are routed to the notification queue for later review — no replay of sounds after meetings end. The timer alert has its own independent volume control, allowing the user to elevate timer alert volume above or below the notification alert volume.

## Design Decisions

- **Single audio file, not a directory** — unlike notification alerts (which can randomly select from a directory), the timer-end sound is a single file. Consistency matters for a recurring timer signal; the user should always hear the same sound.
- **Distinct fallback beep tonality** — the timer fallback beep uses a different frequency/duration than the notification fallback. This allows the user to distinguish timer-end from message notification by sound alone, even without configured audio files.
- **Meeting suppression at the timer level** — the `TimerPresenter` knows the current block type. When `BlockType == BlockMeeting`, it signals the alerter to suppress sound. This keeps the suppression logic in the presenter, not the alert service.
- **Missed alerts routed to notification queue** — during meetings, if a timer block ends, a notification event is created and pushed to the existing notification queue (same as NOTIFIED messages). The user reviews missed alerts from the queue after the meeting. No sound replay.
- **Extends existing alert infrastructure** — new `TimerAlerter` interface wraps the existing `AlertService` patterns (file playback with beep fallback, async goroutine, mutex for thread safety). Does not modify the existing `Alerter` interface to avoid breaking Phase 1 contracts.
- **No cooldown for timer alerts** — unlike notification alerts (which have a 2-second mute), timer-end sounds play immediately. The scheduling engine ensures blocks are always >5 minutes apart, providing natural spacing.
- **Independent volume control** — timer alerts have their own volume slider (0–100), separate from the notification alert volume. This allows the user to set timer alerts louder (e.g., to cut through headphones during focus) or quieter without affecting notification volume. Persisted in config as `planner.timer_volume`.

## Config

```toml
[planner]
timer_sound = ""        # path to audio file; empty = use fallback beep
timer_volume = 75       # 0–100, independent of notification volume
```

The `timer_sound` and `timer_volume` fields are added to the existing `[planner]` config section (Feature 017). When `timer_sound` is empty or unset, the fallback beep is used. `timer_volume` defaults to 75 and is adjustable at runtime via the settings UI.

## API

### TimerAlerter Interface

```go
type TimerAlerter interface {
    // PlayTimerEnd plays the timer-end sound. If during a meeting (suppressed=true),
    // returns a MissedAlert instead of playing sound.
    PlayTimerEnd(ctx context.Context, suppressed bool) (*MissedAlert, error)

    // SetVolume sets the timer alert volume (0–100). Clamped to range.
    SetVolume(volume int)

    // Volume returns the current timer alert volume (0–100).
    Volume() int
}
```

### MissedAlert

```go
type MissedAlert struct {
    BlockType BlockType
    TaskName  string
    Time      time.Time
    Message   string  // e.g., "Focus block ended: Write report"
}
```

### Constructor

```go
func NewTimerAlertService(
    soundPath string,        // empty = fallback beep
    volume int,              // initial volume 0–100
    beeper Beeper,
    fileSystem FileSystem,
    audioPlayer AudioPlayer,
) (*TimerAlertService, error)
```

### Fallback Beep Specification

```go
// Notification fallback (existing): short high-pitched beep
// Timer fallback (new): longer, lower-pitched tone — distinct from notification

const (
    timerBeepFrequency = 440.0  // Hz (A4, lower than notification)
    timerBeepDuration  = 800    // ms (longer than notification)
)
```

## Interaction Flow

### Normal Focus/Break Block End

```
Timer block ends
  → TimerPresenter.SetOnBlockComplete fires
  → Check BlockType:
    - If Meeting: call PlayTimerEnd(ctx, suppressed=true)
      → No sound played
      → MissedAlert returned
      → Route to notification queue
    - If Focus/Break: call PlayTimerEnd(ctx, suppressed=false)
      → Play configured sound file (or fallback beep)
      → Return nil MissedAlert
```

### Meeting Block Handling

```
Meeting starts (BlockMeeting active)
  → Timer runs silently (no intermediate alerts)
  → On meeting block end: no timer-end sound
  → If any timer events were missed during meeting:
    → Each creates a MissedAlert
    → Routed to notification queue
  → After meeting: user checks notification queue
  → No sound replay of missed alerts
```

## Error Handling

| Scenario | Behavior |
|---|---|
| Sound file not found | Fall back to beep, log warning |
| Sound file unreadable | Fall back to beep, log warning |
| Audio playback failure | Fall back to beep, log warning |
| Beep failure | Log error, continue silently (non-fatal) |
| Suppressed alert (meeting) | Return MissedAlert, no sound, no error |
| Empty sound path in config | Use fallback beep (not an error) |
| Volume out of range | Clamped to [0, 100] silently |
| Volume at 0 | No sound played (effectively muted) |

## Integration Points

- **Planner UI (Feature 018):** `TimerPresenter` calls `TimerAlerter.PlayTimerEnd` on block completion, passing suppression flag based on current block type.
- **Notification Queue (Feature 011):** Missed alerts during meetings are converted to notification-queue-compatible events and displayed in the existing notification pane.
- **Alert Service (Feature 010):** Reuses `Beeper`, `FileSystem`, `AudioPlayer` interfaces from the existing alert package. Does not modify the `Alerter` interface.
- **Config (Feature 001):** `timer_sound` field in `[planner]` section, validated as optional file path.

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `alert` | `TimerAlertServiceSuite` | Constructor validation, play with configured file, play with fallback beep, file not found fallback, suppressed (meeting) returns MissedAlert, unsuppressed plays sound, beep frequency/duration distinct from notification, nil beeper/filesystem handling, volume set/get, volume clamping (0–100), volume 0 mutes, volume applied to file playback, volume applied to fallback beep |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Timer Alert | RED | Test Designer | — | — | — |
| Timer Alert | GREEN | Implementer | — | — | — |
| Timer Alert | REFACTOR | Refactorer | — | — | — |
