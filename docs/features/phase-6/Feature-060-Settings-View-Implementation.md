# Feature 060: Settings View Implementation

**Phase:** Phase-6-Feature-060
**Type:** Bugfix
**Severity:** Medium
**Status:** Planned
**Packages:** `internal/ui/`
**Related:** Feature 037 (Settings UI), Feature 036 (Settings Presenter)

---

## Bug Description

The Settings view has four tabs (Slack, Email, Audio, Ollama) but all contain only `widget.NewLabel` placeholder text. No form controls, volume sliders, or account management UI exists despite the `SettingsPresenter` being fully implemented and passed to the view constructor.

## Expected Behavior

Per UiSpec.md and Feature 037:
- **Slack tab:** List of configured Slack accounts with add/edit/remove
- **Email tab:** List of configured email accounts with add/edit/remove
- **Audio tab:** Volume slider, notification sound toggle, sound file selection
- **Ollama tab:** Host/port configuration, model selection, connection test

## Actual Behavior

`settings_view.go` — All four tabs render `widget.NewLabel("Slack Accounts")`, `widget.NewLabel("Email Accounts")`, etc. The `SettingsPresenter` (sp) and `ServiceSettingsPresenter` (ssp) are accepted by the constructor but never used to populate forms.

## Root Cause

Settings UI (Feature 037) built the tab structure but left form content as stubs. The presenter provides all necessary data access methods — the UI rendering was never completed.

## Proposed Fix

Implement the four tab contents using the data and methods already available on `SettingsPresenter` and `ServiceSettingsPresenter`:

1. **Audio tab** (simplest, highest impact): Volume slider bound to `sp.Volume()` / `sp.SetVolume()`, notification toggle
2. **Ollama tab:** Host/port entries, model dropdown, test connection button
3. **Slack tab:** Account list with add/edit/delete using `ssp`
4. **Email tab:** Account list with add/edit/delete using `ssp`

## Test Strategy

- RED: Structural test asserting Audio tab contains a slider widget, not a label
- RED: Interaction test — move volume slider, verify presenter receives updated value
- GREEN: Implement tab contents one at a time
- REFACTOR: Extract shared account list pattern for Slack/Email tabs
