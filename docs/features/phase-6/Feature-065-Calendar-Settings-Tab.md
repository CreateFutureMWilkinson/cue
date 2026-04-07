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

1. Add a `calendarTab` using `newAccountTab("Calendar", onAdd)` in `NewSettingsView`.
2. Wire the `onAdd` callback to open a calendar account creation form (ICS URL + name + poll interval), calling `ssp.SaveCalendarAccount()` on submit.
3. Update tab order to: Slack, Email, Calendar, Audio, Ollama.
4. Update `TestSettingsViewHasFourTabs` → 5 tabs, update `TestSettingsViewTabNames` expected order.

## Files to Change

- `internal/ui/settings_view.go` — add Calendar tab
- `internal/ui/settings_view_test.go` — update tab count and names assertions

## Acceptance Criteria

- [ ] Settings view has 5 tabs: Slack, Email, Calendar, Audio, Ollama
- [ ] Calendar tab shows list of configured accounts
- [ ] Calendar tab has "Add Account" button
- [ ] Add Account opens a form for ICS URL, name, and poll interval
- [ ] Submitting the form calls `ServiceSettingsPresenter.SaveCalendarAccount()`
