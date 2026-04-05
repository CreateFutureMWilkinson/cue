# Feature 037: Settings UI Expansion

**Phase:** Phase-4-Feature-037
**Status:** Planned
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

1. **Cog icon button in the focus rail** — always visible, below the Plan button. Tapping navigates to `ViewSettings`. When settings is active, the cog is replaced by a Back button (same pattern as Plan view hiding the Plan button and showing Back).
2. **Menu bar → Cue → Settings** — existing menu item, but now calls `viewRouter.NavigateTo(ViewSettings)` instead of opening a popup.

### Tabbed Layout in Center Area

`container.NewAppTabs` with one tab per config category:

| Tab | Contents |
|---|---|
| Slack | Account list + add/edit/delete |
| Email | Account list + add/edit/delete |
| Audio | Volume slider (existing functionality) |
| Ollama | Display configured models (read-only from TOML, informational) |

Tabs are ordered by frequency of use — service account management first, then audio, then informational.

### Account List Pattern

Each service tab shows a scrollable list of configured accounts. Each row displays:
- Service identifier (workspace ID for Slack, username for Email)
- Enabled/disabled toggle
- Edit and Delete buttons

An "Add Account" button at the bottom opens an inline form or a modal dialog.

### Form Dialogs

Adding/editing an account opens a Fyne dialog with form fields:

**Slack form fields:**
- Bot Token (password entry)
- Workspace ID (text entry)
- Poll Interval seconds (numeric entry, default 600)
- Enabled (checkbox, default true)

**Email form fields:**
- IMAP Host (text entry, default "imap.gmail.com")
- IMAP Port (numeric entry, default 993)
- Username (text entry)
- Password Env Var (text entry, default "CUE_EMAIL_PASSWORD")
- Poll Interval seconds (numeric entry, default 600)
- Enabled (checkbox, default true)

## Layout

### Overall (Settings Active in Center Area)

```
┌──────────────────────────────────────────────────────────────────┐
│  Cue  [Settings] [About] [Quit]                        Menu Bar │
├──────┬───────────────────────────────────────────┬───────────────┤
│      │  ┌─────┬─────┬───────┬────────┐          │  Notifs (4)   │
│  ◯   │  │Slack│Email│ Audio │ Ollama │          │───────────────│
│ 18m  │  └─────┴─────┴───────┴────────┘          │ [9] #alerts   │
│      │                                           │  Added to...  │
│ Write│  Slack Accounts                           │───────────────│
│ report│ ─────────────────────────────────        │ [8.5] Inbox   │
│      │  T0ABC  workspace-1  [On]  [Edit][Del]   │  Server down  │
│[Done]│  T0DEF  workspace-2  [Off] [Edit][Del]   │───────────────│
│      │                                           │ [8] #general  │
│      │                                           │  @user deploy │
│[Plan]│                                           │───────────────│
│      │            [+ Add Slack Account]          │ [7.2] #team   │
│[Back]│                                           │  Review Q1... │
│  ⚙   │                                           │  [◀ expand]   │
├──────┴───────────────────────────────────────────┴───────────────┤
```

Focus rail: 10% width (unchanged)
Settings center area: 60% width (replaces character area)
Notifications: 30% width (unchanged)

### Focus Rail Changes

When `ViewSettings` is active:
- **Cog button hidden** (already in settings)
- **Back button shown** (returns to ViewCharacter)
- Plan button remains visible

When other views are active:
- **Cog button visible** below Plan button

```
Focus Rail (default)     Focus Rail (settings active)
┌──────┐                 ┌──────┐
│  ◯   │                 │  ◯   │
│ 18m  │                 │ 18m  │
│      │                 │      │
│ Task │                 │ Task │
│[Done]│                 │[Done]│
│      │                 │      │
│[Plan]│                 │[Plan]│
│      │                 │[Back]│
│  ⚙   │                 │      │
└──────┘                 └──────┘
```

### Slack Tab

```
┌─────────────────────────────────────────────────┐
│  Slack Accounts                                  │
│─────────────────────────────────────────────────│
│  T0ABC123  workspace-1    [Enabled]  [Edit][Del] │
│  T0DEF456  workspace-2    [Disabled] [Edit][Del] │
│                                                   │
│                                                   │
│                          [+ Add Slack Account]    │
└─────────────────────────────────────────────────┘
```

### Email Tab

```
┌─────────────────────────────────────────────────┐
│  Email Accounts                                  │
│─────────────────────────────────────────────────│
│  user@gmail.com    imap.gmail.com  [Enabled]     │
│                                    [Edit][Del]   │
│  work@company.com  imap.company.co [Disabled]    │
│                                    [Edit][Del]   │
│                                                   │
│                          [+ Add Email Account]    │
└─────────────────────────────────────────────────┘
```

### Add/Edit Dialog

```
┌──────────────────────────────────┐
│  Add Slack Account               │
│──────────────────────────────────│
│  Bot Token:    [••••••••••••]    │
│  Workspace ID: [T0ABC123    ]    │
│  Poll Interval: [600        ] s  │
│  Enabled:      [x]              │
│──────────────────────────────────│
│            [Cancel]  [Save]      │
└──────────────────────────────────┘
```

### Empty State (No Accounts Configured)

```
┌─────────────────────────────────────────────────┐
│  Slack Accounts                                  │
│─────────────────────────────────────────────────│
│                                                   │
│         No Slack accounts configured.            │
│         Add an account to start monitoring       │
│         your Slack workspace.                     │
│                                                   │
│                          [+ Add Slack Account]    │
└─────────────────────────────────────────────────┘
```

## Code Changes

### CenterViewRouter

Add `ViewSettings` to the view enum:

```go
const (
    ViewCharacter CenterView = iota
    ViewPlan
    ViewWizard
    ViewSettings  // new
)
```

### FocusRail

Add a cog button (`⚙` or Fyne's `theme.SettingsIcon()`):

```go
rail.settingsBtn = widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
    router.NavigateTo(ViewSettings)
})
```

Update `applyViewState`:
```go
case ViewSettings:
    r.planBtn.Show()
    r.backBtn.Show()
    r.settingsBtn.Hide()
```

### Window

- Remove `showSettings()` popup call from menu item
- Menu item calls `viewRouter.NavigateTo(ViewSettings)` instead
- Build settings center pane (tabbed container) and wire it into the view router's content switching
- Delete `settings.go` popup implementation (replaced by center view)

### Settings View Widget

New `SettingsView` struct that builds the tabbed container:

```go
type SettingsView struct {
    tabs      *container.AppTabs
    container fyne.CanvasObject
}

func NewSettingsView(
    sp *presenter.SettingsPresenter,
    ssp *presenter.ServiceSettingsPresenter,
) *SettingsView
```

## Error Handling

| Scenario | UI Behavior |
|---|---|
| Validation error from presenter | Show error dialog with field-specific message |
| Repository error | Show error dialog: "Failed to save account: ..." |
| Watcher start failure | Show warning: "Account saved but watcher failed to start" |
| Delete confirmation | Show confirm dialog before deleting |

## Integration Points

- **`CenterViewRouter`** (`center_view_router.go`): Add `ViewSettings` enum value
- **`FocusRail`** (`focus_rail.go`): Add cog button, update `applyViewState`
- **`MainWindow`** (`window.go`): Wire settings view into center area, update menu item
- **Feature 036** (Settings Presenter): All CRUD operations go through `ServiceSettingsPresenter`
- **Existing `SettingsPresenter`**: Audio tab continues to use existing presenter
- **Feature 038** (Main Wiring): Pass both presenters to `NewMainWindow`

## Test Coverage

- `CenterViewRouter`: `NavigateTo(ViewSettings)` fires callback
- `FocusRail`: cog button visible by default, hidden when `ViewSettings` active
- `FocusRail`: Back button shown when `ViewSettings` active
- Settings view creates with four tabs (Slack, Email, Audio, Ollama)
- Slack tab shows account list from presenter
- Email tab shows account list from presenter
- Audio tab shows volume slider
- Add account button triggers form dialog
- Save triggers presenter SaveSlackAccount/SaveEmailAccount
- Delete triggers confirmation then presenter delete
- Edit populates form with existing values
- Validation error shows error dialog
- Empty state (no accounts) shows helpful message
- Menu item navigates to ViewSettings (not popup)

## Files

| File | Action |
|---|---|
| `internal/ui/center_view_router.go` | Modify — add `ViewSettings` |
| `internal/ui/center_view_router_test.go` | Modify — add ViewSettings test |
| `internal/ui/focus_rail.go` | Modify — add cog button, update applyViewState |
| `internal/ui/focus_rail_test.go` | Modify — cog button visibility tests |
| `internal/ui/settings_view.go` | **New** — `SettingsView` tabbed center-area widget |
| `internal/ui/settings.go` | **Delete** — popup replaced by center view |
| `internal/ui/window.go` | Modify — wire settings view into center area, update menu handler |
| `internal/ui/window_layout_test.go` | Modify — update for new view |
