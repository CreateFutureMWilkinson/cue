# Feature 067 — Email Settings Add Account Callback Is Noop

**Phase:** Phase-6-Feature-067
**Type:** Bugfix
**Severity:** Medium
**Depends on:** Feature 060

## Problem

In the Email settings tab, the "Add Account" button does nothing. Clicking it has no visible effect — no form appears, no account is created.

## Root Cause

`settings_view.go:42`: `newAccountTab("Email", func() {})` — the `onAdd` callback is an empty function literal. Feature 060 implemented the tab structure but left the Add Account callbacks as noops.

## Fix

1. Wire the Email tab's `onAdd` callback to open an account creation form.
2. The form should collect: IMAP host, IMAP port, username, password, poll interval.
3. On submit, call `ssp.SaveEmailAccount()` with a new `repository.EmailAccount`.
4. On success, refresh the account list to show the new entry.
5. Validate required fields (host, port, username, password, poll interval > 0) before calling the presenter.

## Files to Change

- `internal/ui/settings_view.go` — implement email add account form and callback

## Acceptance Criteria

- [ ] Clicking "Add Account" in the Email tab opens a form
- [ ] Form has fields for IMAP host, port, username, password, poll interval
- [ ] Submitting with valid data creates an email account via the presenter
- [ ] Empty required fields show validation feedback
- [ ] Account list refreshes after successful creation
