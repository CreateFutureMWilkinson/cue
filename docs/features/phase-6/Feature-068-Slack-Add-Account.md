# Feature 068 — Slack Settings Add Account Callback Is Noop

**Phase:** Phase-6-Feature-068
**Type:** Bugfix
**Severity:** Medium
**Depends on:** Feature 060

## Problem

In the Slack settings tab, the "Add Account" button does nothing. Clicking it has no visible effect — no form appears, no account is created.

## Root Cause

`settings_view.go:41`: `newAccountTab("Slack", func() {})` — the `onAdd` callback is an empty function literal. Feature 060 implemented the tab structure but left the Add Account callbacks as noops.

## Fix

1. Wire the Slack tab's `onAdd` callback to open an account creation form.
2. The form should collect: bot token, workspace ID, poll interval.
3. On submit, call `ssp.SaveSlackAccount()` with a new `repository.SlackAccount`.
4. On success, refresh the account list to show the new entry.
5. Validate required fields (token, workspace ID, poll interval > 0) before calling the presenter.

## Files to Change

- `internal/ui/settings_view.go` — implement slack add account form and callback

## Acceptance Criteria

- [ ] Clicking "Add Account" in the Slack tab opens a form
- [ ] Form has fields for bot token, workspace ID, poll interval
- [ ] Submitting with valid data creates a Slack account via the presenter
- [ ] Empty required fields show validation feedback
- [ ] Account list refreshes after successful creation
