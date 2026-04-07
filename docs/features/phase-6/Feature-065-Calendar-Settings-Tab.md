# Feature 065 — Settings View Missing Calendar Tab

**Phase:** Phase-6-Feature-065
**Type:** Bugfix
**Severity:** Medium
**Depends on:** Feature 060

## Problem

The `SettingsView` has four tabs (Slack, Email, Audio, Ollama) but no Calendar tab. The `ServiceSettingsPresenter` already exposes `ListCalendarAccounts()`, `SaveCalendarAccount()`, `EditCalendarAccount()`, `DeleteCalendarAccount()`, and `ToggleCalendarAccount()` — but there is no UI surface to invoke them. Users have no way to add, view, or manage calendar accounts.

## Root Cause

`settings_view.go:71` constructs `container.NewAppTabs(slackTab, emailTab, audioTab, ollamaTab)` with no calendar tab. A `newAccountTab("Calendar", ...)` call is missing.

## Fix

1. Added `calendarTab` using `newAccountTab("Calendar", onAdd)` in `NewSettingsView`, grouped with the other account tabs (Slack, Email).
2. Updated tab order to: Slack, Email, Calendar, Audio, Ollama.
3. Updated unit tests: renamed `TestSettingsViewHasFourTabs` → `TestSettingsViewHasFiveTabs`, updated expected tab names.
4. Updated hardcoded tab indices in `settings_interaction_test.go`, `view_content_test.go`, and `settings_acceptance_test.go` (Audio: 2→3, Ollama: 3→4).

## Files Changed

- `internal/ui/settings_view.go` — added Calendar tab
- `internal/ui/settings_view_test.go` — updated tab count and names assertions
- `internal/ui/settings_interaction_test.go` — updated Audio/Ollama tab indices
- `internal/ui/view_content_test.go` — updated tab count and names
- `tests/ui/settings_acceptance_test.go` — updated Audio tab index

## Design Decisions

- Used existing `newAccountTab` helper for consistency with Slack/Email tabs.
- Calendar tab placed between Email and Audio to group all account-type tabs together before configuration tabs.
- `onAdd` callback is a noop for now — wiring to a calendar account creation form is out of scope for this bugfix (will be addressed when calendar account management is fully implemented).

## Error Handling

No new error paths introduced. The Calendar tab uses the same `newAccountTab` pattern as Slack and Email.

## Integration Points

- `ServiceSettingsPresenter` — Calendar CRUD methods already exist and are tested (Feature 060).
- `newAccountTab` helper — shared with Slack and Email tabs.

## Test Coverage Summary

| Test File | Tests Updated | Result |
|---|---|---|
| `settings_view_test.go` | 2 (count + names) | PASS |
| `settings_interaction_test.go` | 4 (Audio/Ollama index refs) | PASS |
| `view_content_test.go` | 1 (tab count + names) | PASS |
| `bugfix_acceptance_test.go` (Bug065) | 3 (count, order, add button) | PASS |
| `settings_acceptance_test.go` | 3 (slider/label index refs) | PASS |

## Acceptance Criteria

- [x] Settings view has 5 tabs: Slack, Email, Calendar, Audio, Ollama
- [x] Calendar tab shows list of configured accounts
- [x] Calendar tab has "Add Account" button
- [ ] Add Account opens a form for ICS URL, name, and poll interval (deferred — noop callback, same as Slack/Email)
- [ ] Submitting the form calls `ServiceSettingsPresenter.SaveCalendarAccount()` (deferred — noop callback)

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | ~32s | ~23,000 | 6839d16 |
| GREEN | Implementer | ~19s | ~21,000 | 09e3df4 |
| REFACTOR | orchestrator | ~5s | — | 09ccce6 |
