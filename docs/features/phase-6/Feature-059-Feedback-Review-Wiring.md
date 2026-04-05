# Feature 059: Feedback Review Modal Wiring

**Phase:** Phase-6-Feature-059
**Type:** Bugfix
**Severity:** Medium
**Status:** Planned
**Packages:** `internal/ui/`
**Related:** Feature 009 (Feedback Buffer), Feature 017 (Focus Rail)

---

## Bug Description

The `showFeedbackReview()` function exists in `feedback_review.go` but nothing in the UI invokes it. The Review button in the focus rail (which is itself unwired — see Feature 055) should trigger the feedback review modal when the notification panel is expanded.

## Expected Behavior

Per UiSpec.md: When the notification panel is expanded, a Review button appears in the focus rail. Tapping it opens the feedback review modal where the user can rate buffered messages 0–10 with optional notes.

## Actual Behavior

`showFeedbackReview()` is dead code. No UI element triggers it.

## Root Cause

The feedback review function was built but never connected to any button's `OnTapped` callback. The focus rail (which should contain the Review button) is also unwired (Feature 055).

## Proposed Fix

After Feature 055 (focus rail wiring) is complete:
1. Connect the Review button's `OnTapped` callback to `showFeedbackReview()`
2. Ensure the Review button visibility is tied to notification panel expansion state
3. Pass required presenter dependencies for feedback data access

**Depends on:** Feature 055 (Focus Rail Wiring)

## Test Strategy

- RED: Interaction test — with notifications expanded, verify Review button is visible
- RED: Interaction test — tap Review button, verify feedback modal opens
- GREEN: Wire callback
- REFACTOR: Clean up
