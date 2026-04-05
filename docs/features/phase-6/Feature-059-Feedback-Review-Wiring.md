# Feature 059: Feedback Review Modal Wiring

**Phase:** Phase-6-Feature-059
**Type:** Bugfix
**Severity:** Medium
**Status:** Done
**Packages:** `internal/ui/`
**Related:** Feature 009 (Feedback Buffer), Feature 017 (Focus Rail), Feature 055 (Focus Rail Wiring)

---

## Overview

Wires the previously dead-code `showFeedbackReview()` function to the focus rail's Review button, and connects the notification panel's expand/collapse state to the Review button's visibility.

## Bug Description

The `showFeedbackReview()` function existed in `feedback_review.go` but nothing in the UI invoked it. The Review button in the focus rail had an `onReview` callback slot but it was never set.

## Root Cause

Two missing wiring connections in `NewMainWindow`:
1. `FocusRail.SetOnReview` was never called with a callback to invoke `showFeedbackReview`
2. `NotificationPresenter.SetOnExpandedChange` was never connected to `FocusRail.SetNotificationsExpanded`

## Design Decisions

- **Accessor pattern:** Added `MainWindow.FocusRail()` accessor to expose the focus rail for test assertions, consistent with the existing `Content()` and `CenterContent()` pattern.
- **Initial state sync:** `SetNotificationsExpanded(np.IsExpanded())` is called at construction time to ensure the Review button matches the presenter's state, not just future changes.
- **Nil-safe guards:** Both wirings are guarded by nil checks (`fp != nil`, `np != nil`) so tests passing nil presenters continue to work.

## API

No new public API. Two wiring connections added inside `NewMainWindow`:

```go
// Review button → feedback modal
fr.SetOnReview(func() { showFeedbackReview(fp, fyneApp) })

// Notification expand → review button visibility
fr.SetNotificationsExpanded(np.IsExpanded())
np.SetOnExpandedChange(fr.SetNotificationsExpanded)
```

## Error Handling

- Nil `FeedbackPresenter`: Review callback is not set (button does nothing)
- Nil `NotificationPresenter`: Expand/collapse wiring is skipped

## Integration Points

- `internal/ui/window.go` — NewMainWindow wiring
- `internal/ui/focus_rail.go` — SetOnReview, SetNotificationsExpanded (existing)
- `internal/ui/feedback_review.go` — showFeedbackReview (existing)
- `internal/ui/presenter/notification_presenter.go` — SetOnExpandedChange (existing)

## Test Coverage

| Test | Description |
|---|---|
| `TestReviewButtonCallbackWiredWhenFeedbackPresenterProvided` | Verifies FocusRail() accessor returns non-nil and Review button has a callback |
| `TestReviewButtonVisibleWhenNotificationsExpanded` | Verifies Review button shows after ToggleExpanded |
| `TestReviewButtonHiddenWhenNotificationsCollapsed` | Verifies Review button hides after expand+collapse |

## TDD Agent Stats

| TDD Phase | Agent | Commit |
|---|---|---|
| RED (behavior 1) | Test Designer | 528bc40 |
| GREEN (behavior 1) | Implementer | 1518d5f |
| RED (behavior 2) | Test Designer | 880c1f9 |
| GREEN (behavior 2) | Implementer | 2fe1d13 |
