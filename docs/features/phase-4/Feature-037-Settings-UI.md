# Feature 037: Settings UI Expansion

**Phase:** Phase-4-Feature-037
**Status:** Done
**Package:** `internal/ui/`
**Depends on:** Feature 036

---

## Overview

Replace the current popup settings window with a full center-area view, following the same pattern as Plan and Wizard views. Settings occupies the center 60% column and is accessed via a cog icon in the focus rail and a menu bar item. A tabbed layout gives each configuration category its own view.

## Design Decisions

### Center View, Not Popup

The current settings popup (`showSettings()`) is replaced. Settings becomes `ViewSettings` in the `CenterViewRouter`, taking over the center column exactly like `ViewPlan` and `ViewWizard` do. The focus rail and notification panel remain visible while settings is open.

This provides more screen real estate for account management forms and keeps the UI pattern consistent — all major views live in the center area.

### Access: Cog Icon + Menu Item

Settings is accessed two ways:

1. **Cog icon button in the focus rail** — always visible in Character view, below the Review button. Tapping navigates to `ViewSettings`. When settings is active, the cog is hidden and a Back button is shown (same pattern as Plan/Wizard views).
2. **Menu bar → Cue → Settings** — existing menu item (wiring to `viewRouter.NavigateTo(ViewSettings)` deferred to Feature 038).

### Tabbed Layout in Center Area

`container.NewAppTabs` with one tab per config category:

| Tab | Contents |
|---|---|
| Slack | Account list + add/edit/delete (placeholder) |
| Email | Account list + add/edit/delete (placeholder) |
| Audio | Volume slider (existing functionality, placeholder) |
| Ollama | Display configured models (read-only from TOML, placeholder) |

Tabs are ordered by frequency of use — service account management first, then audio, then informational. Tab contents are placeholder labels; full CRUD forms are wired in Feature 038 (Main Wiring).

### Focus Rail Button Visibility

| View | Plan | Back | Settings (cog) |
|---|---|---|---|
| Character | Visible | Hidden | Visible |
| Plan/Wizard | Hidden | Visible | Hidden |
| Settings | Visible | Visible | Hidden |

## Code Changes

### CenterViewRouter (`center_view_router.go`)

Added `ViewSettings` constant to the CenterView enum:

```go
const (
    ViewCharacter CenterView = iota
    ViewPlan
    ViewWizard
    ViewSettings
)
```

### FocusRail (`focus_rail.go`)

- Added `settingsBtn *widget.Button` field
- Created button with `widget.NewButtonWithIcon("", theme.SettingsIcon(), ...)` navigating to `ViewSettings`
- Added `SettingsButton()` accessor
- Updated `applyViewState` to handle `ViewSettings` — shows Plan+Back, hides cog
- Updated `Container()` to include settings button

### SettingsView (`settings_view.go` — new file)

New `SettingsView` struct with:
- `NewSettingsView(sp, ssp, ollamaCfg)` constructor accepting SettingsPresenter, ServiceSettingsPresenter, and OllamaConfig
- `Container()` returning the tabbed container as `fyne.CanvasObject`
- `TabCount()` and `TabNames()` for test introspection
- Four tabs: Slack, Email, Audio, Ollama (placeholder content)

### Window (`window.go`)

Added `ViewSettings: widget.NewLabel("Settings")` to the `viewContents` map so navigation to ViewSettings swaps center content.

## Error Handling

| Scenario | UI Behavior |
|---|---|
| Validation error from presenter | Show error dialog with field-specific message (Feature 038) |
| Repository error | Show error dialog: "Failed to save account: ..." (Feature 038) |
| Delete confirmation | Show confirm dialog before deleting (Feature 038) |

## Integration Points

- **`CenterViewRouter`** (`center_view_router.go`): `ViewSettings` enum value
- **`FocusRail`** (`focus_rail.go`): Cog button, `applyViewState` handling
- **`MainWindow`** (`window.go`): Settings placeholder in center area
- **Feature 036** (Settings Presenter): `ServiceSettingsPresenter` accepted by `NewSettingsView`
- **Feature 038** (Main Wiring): Full wiring of presenters to SettingsView, menu item redirect

## Test Coverage

| Test | File |
|---|---|
| `NavigateTo(ViewSettings)` fires callback | `center_view_router_test.go` |
| Settings button visible by default | `focus_rail_test.go` |
| Settings button hidden in Settings view | `focus_rail_test.go` |
| Back button visible in Settings view | `focus_rail_test.go` |
| Plan button visible in Settings view | `focus_rail_test.go` |
| Settings button navigates to ViewSettings | `focus_rail_test.go` |
| Settings button hidden in Plan view | `focus_rail_test.go` |
| Container includes settings button | `focus_rail_test.go` |
| NewSettingsView returns non-nil | `settings_view_test.go` |
| Container returns non-nil | `settings_view_test.go` |
| Has four tabs | `settings_view_test.go` |
| Tab names correct and ordered | `settings_view_test.go` |
| NavigateTo(ViewSettings) swaps center content | `window_layout_test.go` |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | ~89s | ~47,800 | 5c52ace |
| GREEN | Implementer | ~95s | ~35,400 | 18e6782 |
| REFACTOR | Refactorer | manual | — | 1240e55 |

## Files

| File | Action |
|---|---|
| `internal/ui/center_view_router.go` | Modified — added `ViewSettings` |
| `internal/ui/center_view_router_test.go` | Modified — added ViewSettings test |
| `internal/ui/focus_rail.go` | Modified — added cog button, updated applyViewState |
| `internal/ui/focus_rail_test.go` | Modified — 7 new cog button tests |
| `internal/ui/settings_view.go` | **New** — `SettingsView` tabbed center-area widget |
| `internal/ui/settings_view_test.go` | **New** — 4 SettingsView tests with stubs |
| `internal/ui/window.go` | Modified — added ViewSettings to viewContents |
| `internal/ui/window_layout_test.go` | Modified — added ViewSettings navigation test |
