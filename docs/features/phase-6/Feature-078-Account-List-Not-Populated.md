# Feature 078 — Added Service Accounts Don't Appear in UI

| Field | Value |
|---|---|
| Phase | 6 |
| Type | Bugfix |
| Severity | Critical |
| Status | Planned |
| Depends on | 065, 067, 068 |
| UI Tests | Yes |

## Problem

Adding Calendar, Email, or Slack accounts via Settings creates database records successfully, but nothing appears in the account list UI. The lists remain permanently empty.

## Root Cause Analysis

The account list containers (`slackAccountList`, `emailAccountList`, `calendarAccountList`) in `settings_view.go` are created as empty `container.NewVBox()` and **never populated with data**:

- Line 248: `slackAccountList := container.NewVBox()` — created empty
- Line 269: `emailAccountList := container.NewVBox()` — created empty
- Line 291: `calendarAccountList := container.NewVBox()` — created empty

The `ServiceSettingsPresenter` has working methods:
- `ListSlackAccounts(ctx)`
- `ListEmailAccounts(ctx)`
- `ListCalendarAccounts(ctx)`

But these are **never called** from the UI layer — not on initial load, and not after saving a new account. The `onSaved()` callback resets to the list view but rebuilds the same empty container.

## Required Changes

1. **Initial load**: When `NewSettingsView` is created (or when each tab is first shown), call the presenter's `List*Accounts()` methods and populate the VBox with account summary widgets
2. **Refresh after save**: After `Save*Account()` succeeds, re-query the account list and rebuild the VBox contents
3. **Account display**: Each account entry should show key identifying info (e.g., workspace name for Slack, email address for Email, calendar name for Calendar) and an edit/delete control
4. **Empty state**: When no accounts exist, show a helpful message (e.g., "No Slack accounts configured. Tap Add Account to get started.")

## Acceptance Criteria

- Existing accounts appear in the list when opening Settings
- After adding a new account, it appears in the list immediately
- After editing an account, the list reflects the change
- Empty state message shown when no accounts exist

## UI Test Coverage

- UI acceptance test: add an account, verify it appears in the account list
- UI acceptance test: open settings with pre-existing accounts, verify they are listed
