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

- [ ] Audio tab shows two sliders: "Notification Volume" and "Timer Volume"
- [ ] Timer Volume slider range 0–100, step 1
- [ ] Timer Volume label updates live during drag
- [ ] Timer Volume is independent of Notification Volume
- [ ] Timer volume setting is applied to timer alert playback
