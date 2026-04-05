# Feature 060A: Settings Exit Control

**Phase:** Phase-6-Feature-060A
**Type:** Enhancement (Hotfix)
**Severity:** Low
**Status:** Done
**Packages:** `internal/ui/`
**Related:** Feature 060 (Settings View Implementation), Feature 055 (Focus Rail Wiring)

---

## Problem

The Settings view (center column) has no exit control within the view itself to return to the main character view. Users must rely on the Back button in the FocusRail (left column) to navigate away. This is non-obvious — the Back button is small and in a different column, which can be missed especially by ADHD users who may lose spatial context.

## Current Navigation State

When `ViewSettings` is active, the FocusRail shows:
- **Plan** button (visible) — navigates to Plan view
- **Back** button (visible) — navigates to ViewCharacter
- **Settings** button (hidden) — already in settings

The Back button works but is in the left column, not within the settings view content area itself.

## Proposed Fix

Add a "Close" or "Done" button within the `SettingsView` container that navigates back to `ViewCharacter`. This provides an in-context exit affordance that is discoverable without scanning the left rail.

### Implementation Options

**Option A: Close button at bottom of settings tabs**
- Add a `widget.Button("Done", func() { router.NavigateTo(ViewCharacter) })` at the bottom of the settings view
- Requires passing the `CenterViewRouter` (or a `func()` callback) to `NewSettingsView`

**Option B: Close button in tab bar area**
- Place button alongside the tabs using a border layout (tabs top, close button top-right)
- More discoverable but trickier layout

**Recommended:** Option A — simpler, consistent with how other views work.

### Constructor Change

```go
func NewSettingsView(
    sp *presenter.SettingsPresenter,
    ssp *presenter.ServiceSettingsPresenter,
    ollamaCfg config.OllamaConfig,
    onClose func(),  // NEW: callback to navigate back to main view
) *SettingsView
```

## Test Strategy

- RED: Assert `SettingsView` container contains a Button with text "Done" (or "Close")
- RED: Assert tapping the button calls the `onClose` callback
- GREEN: Add the button and wire the callback
- REFACTOR: Clean up if needed

## Implementation

Implemented Option A. Added `onClose func()` as a 4th parameter to `NewSettingsView`. The constructor wraps the existing `AppTabs` in a `container.NewBorder` layout with a "Done" button at the bottom. In `window.go`, the callback navigates to `ViewCharacter` via the `CenterViewRouter`.

## Test Coverage

| Test | Type | Asserts |
|---|---|---|
| `TestSettingsViewContainsDoneButton` | Structural | `*widget.Button` with text "Done" found in container tree |
| `TestDoneButtonCallsOnClose` | Interaction | Tapping Done button invokes the `onClose` callback |

Plus all 14 pre-existing settings tests remain green.

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | ~68s | ~31,000 | 8451534 |
| GREEN | Implementer | ~83s | ~28,000 | 4c7f1cb |
| REFACTOR | orchestrator | manual | — | 26c0a86 |

## Acceptance Criteria

- [x] Settings view contains a visible "Done" button
- [x] Tapping "Done" navigates back to ViewCharacter
- [x] Existing FocusRail Back button continues to work
- [x] All existing settings tests remain green
