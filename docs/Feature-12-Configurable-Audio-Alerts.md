# Feature 12: Configurable Audio Alerts

**Phase:** Phase-1-Feature-12 (amendment)
**Status:** Done
**Packages:** `internal/alert/`, `internal/config/`, `internal/ui/presenter/`, `internal/ui/`, `cmd/cue/`

---

## Overview

Replaces the hardcoded beeep-only alert system with configurable file-based audio playback. The alert service now picks a random MP3/WAV/OGG file from a user-configured directory and plays it asynchronously. Falls back to a configurable beeep tone when no audio files are available. Startup and shutdown sounds are removed entirely. A new settings panel provides runtime volume control via a slider.

## Design Decisions

### File Playback with Fallback

The alert service tries file playback first (random file from `audio_dir`), falling back to `beeep` in these cases:
- `audio_dir` is empty (unconfigured)
- Directory contains no supported files (.mp3, .wav, .ogg)
- `AudioPlayer` is nil (not wired yet)
- `AudioPlayer.PlayFile` returns an error
- `FileSystem.ReadDir` returns an error

This ensures users always hear something on NOTIFIED messages, even with misconfiguration.

### Async Playback

File playback runs in a goroutine (fire-and-forget) to avoid blocking the routing pipeline. If the goroutine's playback fails, it synchronously calls the beeep fallback. The beeep fallback itself is synchronous since it's fast (~200ms).

### Interface-Based Testability

Three interfaces enable full mock coverage without real audio or filesystem:
- `Beeper` — wraps `beeep.Beep` for fallback tones
- `FileSystem` — wraps `os.ReadDir` for directory listing
- `AudioPlayer` — abstracts file playback (not yet wired to a real implementation)

Production `osFileSystem` is defined in `cmd/cue/main.go`. The `AudioPlayer` is nil for now (always uses beeep fallback) until `gopxl/beep` is integrated.

### Configurable Cooldown and Fallback Tone

Previously hardcoded values are now in `config.toml`:
- `audio_cooldown_seconds` (default 2) — minimum time between alerts
- `fallback_frequency` (default 1000 Hz) — beeep tone frequency
- `fallback_duration_ms` (default 200) — beeep tone duration

### Runtime Volume Control

`SettingsPresenter` holds the current volume (0-100) and delegates to the `AlertService` via the `VolumeController` interface. The UI settings panel exposes a Fyne slider. Volume clamping (0-100) happens at both the presenter and alert service layers.

### Removal of Startup/Shutdown Sounds

`PlayStartup` and `PlayShutdown` were removed from both the alert service and the presenter `Alerter` interface. The `AppPresenter` no longer depends on an alerter at all — its constructor takes 3 args instead of 4.

## Config Changes

```toml
[notification]
audio_enabled = true
audio_dir = ""                # empty = beeep fallback
audio_cooldown_seconds = 2
audio_volume = 100            # 0-100
fallback_frequency = 1000     # Hz
fallback_duration_ms = 200    # ms
```

Validation:
- `audio_dir` non-empty but doesn't exist on disk = error
- `audio_cooldown_seconds` >= 0
- `audio_volume` 0-100
- `fallback_frequency` > 0
- `fallback_duration_ms` > 0
- `audio_dir` supports tilde expansion (`~/sounds` -> `/home/user/sounds`)

## API Changes

### Alert Service
- Constructor: `NewAlertService(cfg, beeper, fs, player)` (was 2 args, now 4)
- Added: `SetVolume(int)`, `FileSystem` interface, `AudioPlayer` interface
- Removed: `PlayStartup()`, `PlayShutdown()`

### Presenter
- `NewAppPresenter` takes 3 args (removed alerter)
- Added: `VolumeController` interface, `SettingsPresenter`
- Removed: `Alerter` interface (from presenter package)

### UI
- `NewMainWindow` takes `SettingsPresenter` as 6th arg
- Added: Settings menu item, standalone settings panel with volume slider

## Test Coverage

| Area | Tests |
|---|---|
| Config fields (parse, defaults, validation, tilde) | 10 |
| Alert service (playback, fallback, cooldown, volume, errors) | 22 |
| Settings presenter (volume, clamping, delegation) | 6 |
| App presenter (updated for 3-arg constructor) | 6 |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED (config) | test-designer | 173s | 30862 | c847cff |
| GREEN (config) | implementer | 139s | 42365 | 55d1d40 |
| REFACTOR (config) | refactorer | 126s | 33309 | 5ea6f12 |
| RED (alert) | test-designer | 82s | 28401 | c5abb89 |
| GREEN (alert) | implementer | 50s | 29182 | 3eeddc1 |
| REFACTOR (alert) | refactorer | 56s | 22903 | 92204a6 |
| RED (presenter) | test-designer | 54s | 26691 | 4b612eb |
| GREEN (presenter) | implementer | 53s | 27827 | 1a136cc |
| REFACTOR (presenter) | refactorer | 38s | 19687 | 3bd8eea |
