# Feature 076A — UAT Mode Bug Fixes

| Field | Value |
|---|---|
| Phase | 7 |
| Type | Bugfix |
| Severity | High |
| Status | Done |
| Depends on | 076 |
| UI Tests | No (unit tests only) |

## Problems

Three bugs in the integrated character UAT mode (Feature 076):

### Bug A: Missing activity log
UAT passed `nil` for the activity presenter, so `NewMainWindow` skipped creating the `ActivityLogDrawer`. The center column showed the character without the activity log overlay.

### Bug B: State buttons noop until dropdown used
`UATPanel.currentChar` started as `nil`. The initial character created in `runUAT()` was passed to `NewMainWindow` for display but never linked to the panel. State buttons checked `if p.currentChar != nil` before calling `TransitionTo`, so they silently did nothing.

### Bug C: Fairy motion broken (glow works, position doesn't)
Fyne's `container.Refresh()` only repaints without re-running the custom `fairyJarLayout.Layout()`. `SetGlowIntensity` modifies circle `FillColor` directly (picked up by repaint), but `SetPosition` stores normalized coordinates that only the layout reads when positioning circles via `Move()`. Without a layout re-run, position changes were invisible.

## Changes Made

### Bug A fix
- Created a channel-based `ActivitySource` in `uat.go`
- Created an `ActivityPresenter` and passed it to `NewMainWindow`
- Wired `UATPanel.SetOnStateChange` to publish events ("State -> Working") to the channel
- Startup animation also logs to activity log

### Bug B fix
- Added `SetInitialCharacter(ch, name)` to `UATPanel` — sets `currentChar`, updates labels, enables buttons
- Called from `runUAT()` immediately after panel creation

### Bug C fix
- Added `ForceRefresh()` to `FairyCharacter` — explicitly calls `Layout.Layout()` on the fairy's container with its current size, forcing circle repositioning
- Updated refresh hooks in both `uat.go` and `main.go` to call `ForceRefresh()` instead of `Widget().Refresh()`

## Test Coverage

- `TestSetInitialCharacterEnablesButtons` — verifies label update
- `TestSetInitialCharacterAllowsStateTrigger` — verifies buttons work after initial char set
- `TestStateTriggerPublishesActivityEvent` — verifies state change callback fires
- `TestForceRefreshTriggersLayout` — verifies circles reposition after SetPosition + ForceRefresh

## TDD Agent Stats

| Phase | Role | Commit |
|---|---|---|
| RED (initial char + events) | Test Designer | abdd613 |
| GREEN (initial char + events) | Implementer | 070bd7b |
| RED (ForceRefresh) | Test Designer | 46137d0 |
| GREEN (ForceRefresh) | Implementer | e6c9022 |
| GREEN (UAT wiring) | Implementer | 4b3077d |
