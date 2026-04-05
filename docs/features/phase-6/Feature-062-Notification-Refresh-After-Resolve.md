# Feature 062: Notification Refresh After Resolve

**Phase:** Phase-6-Feature-062
**Type:** Bugfix
**Severity:** Low
**Status:** Planned
**Packages:** `internal/ui/`
**Related:** Feature 018 (Notification Panel Redesign)

---

## Bug Description

After resolving a notification via the detail dialog, the notification list is not refreshed. The resolved item remains visible until the next poll cycle or manual refresh.

## Expected Behavior

Closing the detail dialog after a Resolve action should immediately refresh the notification list to reflect the updated state.

## Actual Behavior

`notification_pane.go:46` — After `np.Select(id)` returns and the detail dialog closes, no `Refresh()` or data reload is triggered on the list widget.

## Root Cause

The detail dialog's dismiss/resolve handler does not call back to refresh the parent list.

## Proposed Fix

After the detail dialog closes (whether via Resolve or dismiss), trigger a refresh on the notification list widget. This can be done by calling the presenter's refresh method and then `list.Refresh()` in the dialog's close callback.

## Test Strategy

- RED: Interaction test — resolve a notification via detail dialog, verify the list item count decreases
- GREEN: Add refresh call after dialog close
- REFACTOR: Clean up
