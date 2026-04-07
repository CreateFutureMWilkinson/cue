# Feature 078 — Added Service Accounts Don't Appear in UI

| Field | Value |
|---|---|
| Phase | 6 |
| Type | Bugfix |
| Severity | Critical |
| Status | Done |
| Depends on | 065, 067, 068 |
| UI Tests | Yes |

## Problem

Adding Calendar, Email, or Slack accounts via Settings creates database records successfully, but nothing appears in the account list UI. The lists remain permanently empty.

## Root Cause Analysis

The account list containers (`slackAccountList`, `emailAccountList`, `calendarAccountList`) in `settings_view.go` are created as empty `container.NewVBox()` and **never populated with data**.

The `ServiceSettingsPresenter` has working methods (`ListSlackAccounts`, `ListEmailAccounts`, `ListCalendarAccounts`) but these were **never called** from the UI layer — not on initial load, and not after saving a new account. The `onSaved()` callback reset to the list view but rebuilt the same empty container.

## Solution

### New Functions

- **`refreshAccountList(list, emptyMsg, items)`** — clears a VBox and repopulates it. Shows an empty state message label when no accounts exist.
- **`listAccountWidgets(ssp, accountType)`** — queries the presenter for accounts of the given type and returns label widgets. Handles nil presenter and nil internal deps gracefully via defer/recover.

### Integration Points

- **Initial load:** `buildXxxListContent()` now calls `refreshXxx()` which queries the presenter before building the container.
- **Refresh after save:** The `onSaved` callback in each form already calls `buildXxxListContent()`, which now triggers a refresh — so newly saved accounts appear immediately.
- **Empty state:** When no accounts exist, shows "No X accounts configured. Tap Add Account to get started."

### Account Display Format

| Type | Format | Example |
|---|---|---|
| Slack | `Slack: {WorkspaceID} (@{Username})` | `Slack: T-ACME (@alice)` |
| Email | `Email: {Username} ({Host}:{Port})` | `Email: user@example.com (imap.gmail.com:993)` |
| Calendar | `Calendar: {Name}` | `Calendar: Work Calendar` |

### Supporting Change

Added `*container.Scroll` traversal to `uitest.children()` so `FindWidget` can discover labels inside scrollable account lists.

## Test Coverage

### UI Acceptance Tests (9 new)

- Empty state messages for Slack, Email, Calendar tabs
- Pre-existing accounts appear on initial load (all 3 types)
- Newly saved accounts appear after save (all 3 types)

### Unit Tests (1 new)

- `TestSlackTabShowsEmptyStateWhenNoAccounts` — verifies empty state label in Slack tab

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| UI Tests | Test Designer (orchestrator) | ~3m | ~15,000 | 20f51a3 |
| RED | Test Designer | ~25s | ~35,000 | 100efbd |
| GREEN | Implementer (orchestrator) | ~5m | ~20,000 | 7881eea |
| REFACTOR | Refactorer | ~90s | ~34,000 | fd1ff6a |
