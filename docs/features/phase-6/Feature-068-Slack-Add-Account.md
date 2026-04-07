# Feature 068 — Slack Settings Add Account Callback Is Noop

**Phase:** Phase-6-Feature-068
**Type:** Bugfix
**Severity:** Medium
**Depends on:** Feature 060
**Status:** Done

## Problem

In the Slack settings tab, the "Add Account" button does nothing. Clicking it has no visible effect — no form appears, no account is created.

## Root Cause

`settings_view.go:114`: `newAccountTab("Slack", func() {})` — the `onAdd` callback is an empty function literal. Feature 060 implemented the tab structure but left the Add Account callbacks as noops.

## Fix

Replaced the static `newAccountTab` call with a dynamic content-switching pattern identical to the Email tab (Feature 067):

1. Created `createSlackAccountForm()` function that builds an inline form with Entry widgets for Bot Token, Workspace ID, and Poll Interval (seconds).
2. Replaced the Slack tab construction with a `slackAccountList` VBox, `slackAddBtn` button, and `buildSlackListContent` helper.
3. Wired `slackAddBtn.OnTapped` to swap tab content to the Slack account form.
4. Form validates all fields required and poll interval is numeric.
5. On valid save: creates `repository.SlackAccount` with `uuid.New()`, `Enabled: true`, calls `ssp.SaveSlackAccount()`, then restores the account list view.
6. Cancel button restores the account list view.

## Files Changed

- `internal/ui/settings_view.go` — added `createSlackAccountForm()`, replaced noop Slack tab with dynamic content switching

## API / Integration Points

- `presenter.ServiceSettingsPresenter.SaveSlackAccount()` — validates token, workspace ID, poll interval; persists via `ServiceConfigRepository.UpsertSlackAccount()`; starts watcher via factory function
- `repository.SlackAccount` — ID, Enabled, Token, WorkspaceID, PollIntervalSeconds

## Error Handling

- Empty required fields → "All fields are required" validation label
- Non-numeric poll interval → "Poll interval must be a number" validation label
- Presenter save failure → "Error: <message>" label (covers token/workspace validation and DB errors)

## Test Coverage

| Test | Type | File |
|---|---|---|
| `TestSlackAddAccountShowsFormFields` | UI acceptance | `tests/ui/settings_acceptance_test.go` |
| `TestSlackAddAccountValidationShowsError` | UI acceptance | `tests/ui/settings_acceptance_test.go` |
| `TestSlackAddAccountSaveWithValidDataReplacesForm` | UI acceptance | `tests/ui/settings_acceptance_test.go` |
| `TestSlackAddAccountShowsFormFields` | Unit interaction | `internal/ui/settings_interaction_test.go` |
| `TestSlackAddAccountValidationShowsError` | Unit interaction | `internal/ui/settings_interaction_test.go` |
| `TestSlackAddAccountSaveWithValidDataReplacesForm` | Unit interaction | `internal/ui/settings_interaction_test.go` |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED (UI) | orchestrator | manual | — | 675b9af |
| RED | Test Designer | ~31s | ~27,877 | b985d06 |
| GREEN | Implementer | ~8.4s | ~29,632 | f24713c |
| RED | Test Designer | ~25s | ~25,559 | 4953551 |
| RED | Test Designer | ~26s | ~26,035 | 4953551 |
