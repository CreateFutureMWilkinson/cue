# Feature 069 — Audio Settings Missing Timer Volume Slider

**Phase:** Phase-6-Feature-069
**Type:** Bugfix
**Severity:** Medium
**Depends on:** Feature 060

## Problem

The Audio settings tab has a "Notification Volume" slider but no "Timer Volume" slider. The UiSpec (`docs/guides/UiSpec.md:946-949`) defines two independent sliders: one for notification alerts and one for timer (plan phase end) alerts. Only the notification slider was implemented.

## Root Cause

`settings_view.go:55-59` builds `audioContent` with only the notification volume slider. No timer volume slider is created. The `SettingsPresenter` only manages a single `VolumeController` — there is no second volume controller for timer alerts.

## Fix

1. Add a `TimerVolumeController` interface or extend `SettingsPresenter` to support a second volume channel.
2. Add `TimerVolume()` / `SetTimerVolume()` methods to `SettingsPresenter`.
3. Add a second slider ("Timer Volume") to the Audio tab in `settings_view.go`.
4. Wire the timer slider's `OnChanged` to `sp.SetTimerVolume()`.
5. The timer volume controls the `TimerAlerter` volume used by `TimerPresenter`.

## Files to Change

- `internal/ui/presenter/settings_presenter.go` — add timer volume state + controller
- `internal/ui/presenter/settings_presenter_test.go` — test timer volume methods
- `internal/ui/settings_view.go` — add timer volume slider to Audio tab
- `internal/ui/settings_view_test.go` — verify two sliders exist (if applicable)

## Acceptance Criteria

- [x] Audio tab shows two sliders: "Notification Volume" and "Timer Volume"
- [x] Timer Volume slider range 0–100, step 1
- [x] Timer Volume label updates live during drag
- [x] Timer Volume is independent of Notification Volume
- [x] Timer volume setting is applied to timer alert playback

## Implementation

### Behavior 1: SettingsPresenter timer volume support

Extended `NewSettingsPresenter` to accept a second `VolumeController` parameter and initial timer volume value. Added `TimerVolume() int` and `SetTimerVolume(int)` methods with 0-100 clamping, mirroring the existing notification volume API. Updated all callers across tests and `cmd/cue/main.go`.

### Behavior 2: Timer Volume slider in Audio tab

Added a "Timer Volume" label and slider to the Audio tab in `settings_view.go`, positioned below the existing Notification Volume slider. The slider is wired to `sp.SetTimerVolume()` with live label updates showing the current percentage value, matching the Notification Volume slider pattern.

### Bug fix: UI acceptance test tab indices

The Bug069 UI acceptance tests used index 2 (Calendar) instead of index 3 (Audio) when selecting the Audio tab. Corrected to index 3 after Feature 065 inserted the Calendar tab at position 2.

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED (presenter) | Test Designer | ~35s | ~30,000 | 95a36a5 |
| GREEN (presenter) | Implementer | ~30s | ~28,000 | 27970e8 |
| RED (view) | Test Designer | ~35s | ~30,000 | 5c8ca36 |
| GREEN (view) | Implementer | ~30s | ~28,000 | fe03d20 |
| FIX (tab index) | orchestrator | ~10s | ~5,000 | a1bc3a4 |
