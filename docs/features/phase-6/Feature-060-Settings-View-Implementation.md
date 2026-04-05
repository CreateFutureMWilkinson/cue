# Feature 060: Settings View Implementation

**Phase:** Phase-6-Feature-060
**Type:** Bugfix
**Severity:** Medium
**Status:** Done
**Packages:** `internal/ui/`
**Related:** Feature 037 (Settings UI), Feature 036 (Settings Presenter), Feature 052 (Automated UI Testing)

---

## Bug Description

The Settings view has four tabs (Slack, Email, Audio, Ollama) but all contained only `widget.NewLabel` placeholder text. No form controls, volume sliders, or account management UI existed despite the `SettingsPresenter` and `ServiceSettingsPresenter` being fully implemented and passed to the view constructor.

## Root Cause

Settings UI (Feature 037) built the tab structure but left form content as stubs. The presenters provide all necessary data access methods — the UI rendering was never completed.

## Fix Summary

Replaced all four placeholder tabs with real widget content:

1. **Audio tab** — Title label, notification volume label with live percentage, `widget.Slider` (0–100, step 1) wired to `SettingsPresenter.SetVolume()` and live-updating label text.
2. **Ollama tab** — Read-only display of `OllamaConfig` fields: host, port, inference model, embedding model, timeout.
3. **Slack tab** — Border layout with title, scrollable account list area, and "Add Account" button.
4. **Email tab** — Same layout as Slack using shared `newAccountTab` helper.

## Design Decisions

- **Shared `newAccountTab` helper** — Slack and Email tabs follow an identical border layout pattern (title top, Add button bottom, scrollable list center). Extracted to `newAccountTab(title, onAdd)` to eliminate duplication.
- **Slider over Entry** — Volume uses `widget.Slider` per UiSpec.md for immediate visual feedback.
- **Read-only Ollama tab** — Ollama config is display-only since it comes from `config.toml` and requires app restart to change.

## API

### New unexported function

- `newAccountTab(title string, onAdd func()) *container.TabItem` — Creates a tab with border layout: title label (top), Add Account button (bottom), scrollable VBox (center).

### Modified constructor

- `NewSettingsView(sp, ssp, ollamaCfg)` — Now uses `sp.Volume()`, `sp.SetVolume()` for audio slider, and `ollamaCfg` fields for Ollama display.

## Error Handling

No new error paths. Volume is clamped 0–100 by the presenter. Account operations delegate to presenter which handles validation.

## Test Coverage

| Test | Type | Asserts |
|---|---|---|
| `TestAudioTabContainsVolumeSlider` | Structural | Slider exists with Min=0, Max=100, Step=1 |
| `TestAudioSliderOnChangedUpdatesVolumeLabel` | Interaction | Label text updates to "Notification Volume: 75%" |
| `TestAudioSliderOnChangedCallsPresenterSetVolume` | Interaction | `sp.Volume()` returns 75 after OnChanged(75) |
| `TestOllamaTabDisplaysConfigFields` | Structural | Labels contain "localhost" and "neural-chat" |
| `TestSlackTabContainsAddButton` | Structural | Button with text "Add Account" exists |
| `TestEmailTabContainsAddButton` | Structural | Button with text "Add Account" exists |

Plus 4 pre-existing structural tests remain green.

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED (behavior 1) | Test Designer | ~38s | ~24,000 | 1df1fd4 |
| GREEN (behavior 1) | Implementer | ~24s | ~22,000 | c592e47 |
| RED (behavior 2) | Test Designer | ~63s | ~29,000 | 3acabd1 |
| GREEN (behavior 2) | orchestrator | manual | — | 6b318c6 |
| RED (behavior 3) | Test Designer | ~46s | ~24,000 | 87c33e3 |
| GREEN (behavior 3) | orchestrator | manual | — | d46732d |
| RED (behavior 4) | Test Designer | ~26s | ~23,000 | 5d672ed |
| GREEN (behavior 4) | orchestrator | manual | — | 5575038 |
| RED (behavior 5) | Test Designer | ~23s | ~23,000 | 2057d93 |
| GREEN (behavior 5) | orchestrator | manual | — | 9c8abe6 |
| REFACTOR | Refactorer | ~41s | ~22,000 | 779e6bf |
