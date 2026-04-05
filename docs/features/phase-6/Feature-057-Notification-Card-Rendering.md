# Feature 057: Notification Card Color Rendering

**Phase:** Phase-6-Feature-057
**Type:** Bugfix
**Severity:** High
**Status:** Planned
**Packages:** `internal/ui/`
**Related:** Feature 018 (Notification Panel Redesign), Feature 018A (Notification Card Rendering)

---

## Bug Description

The notification panel renders plain text labels instead of the color-coded `NotificationCard` widgets. The `NotificationCard` struct with proper importance-based coloring exists in `presenter/notification_card.go` and works correctly, but `notification_pane.go` ignores it and renders simple text via the presenter's `Messages()` method instead of `Cards()`.

Additionally, the notification panel has no expand/collapse toggle and no dismiss functionality.

## Expected Behavior

Per UiSpec.md:
- Collapsed (30% width): Color-coded cards with importance badge, channel, message preview, sender, relative time, background opacity scaling with importance
- Expanded (90% width): Full cards replacing the character area, with source/channel/sender/time/dismiss in a single row and full message preview below
- Toggle button to switch between collapsed and expanded states

## Actual Behavior

All notifications render as identical plain-text list items. No color differentiation by importance. No expand/collapse. No dismiss button.

## Root Cause

`notification_pane.go` — The list widget's `CreateItem`/`UpdateItem` callbacks render `widget.Label` with truncated text. The `NotificationCard` type and `BuildNotificationCards()` function (which apply correct colors per importance tier) are never used in the rendering pipeline.

## Proposed Fix

1. Replace the `widget.List` rendering to use `NotificationCard` widgets with the color scheme from `presenter/notification_card.go`
2. Add an expand/collapse toggle button to the notification panel header
3. Wire expand state to resize the notification column (30% ↔ 90%) and show/hide the character area
4. Add dismiss button per card that calls `Resolve()` on the presenter

## Test Strategy

- RED: Structural test asserting notification list items contain colored card elements, not plain labels
- RED: Interaction test — tap expand toggle, verify panel width state changes
- RED: Interaction test — tap dismiss on a card, verify notification is resolved
- GREEN: Implement card rendering, expand/collapse, dismiss
- REFACTOR: Remove unused plain-text rendering code
