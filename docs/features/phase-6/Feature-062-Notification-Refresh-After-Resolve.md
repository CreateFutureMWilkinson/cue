# Feature 062: Notification Refresh After Resolve

**Phase:** Phase-6-Feature-062
**Type:** Bugfix
**Severity:** Low
**Status:** Done
**Packages:** `internal/ui/`
**Related:** Feature 018 (Notification Panel Redesign)

---

## Bug Description

After resolving a notification via the detail dialog, the notification list was not refreshed. The resolved item remained visible until the next poll cycle or manual refresh.

## Root Cause

The detail dialog's Resolve button called `list.Refresh()` directly but did not close the dialog. When the user dismissed the dialog via the "Close" button, no list refresh was triggered. This meant the list showed stale data after the dialog closed.

## Fix

1. **Dialog auto-close on resolve** — The Resolve button now calls `d.Hide()` to close the dialog after resolving the notification.
2. **`SetOnClosed` refresh** — `d.SetOnClosed(func() { list.Refresh() })` ensures the list refreshes whenever the dialog closes, whether via Resolve or the Close button.
3. **`CardCount()` accessor** — New method on `NotificationPanel` exposes the current card count for testability.

## API

### NotificationPanel

```go
// CardCount returns the number of notification cards currently displayed.
func (p *NotificationPanel) CardCount() int
```

## Error Handling

No new error paths. The existing error handling for `np.Resolve()` is unchanged.

## Integration Points

- `NotificationPanel` → `NotificationPresenter.Cards()` (card count)
- `NotificationPanel` → `dialog.Dialog.SetOnClosed()` (refresh trigger)

## Test Coverage

| Test | Verifies |
|---|---|
| `TestListRefreshesAfterResolve` | Card count goes from 2 → 1 after resolving one notification |

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED | Test Designer | ~42s | ~29,400 | 7dda5b7 |
| GREEN | Implementer | ~53s | ~27,200 | ee4e8af |
| REFACTOR | Refactorer | ~62s | ~28,300 | 8b36f01 |
