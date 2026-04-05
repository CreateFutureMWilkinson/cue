# Feature 057: Notification Card Color Rendering

**Phase:** Phase-6-Feature-057
**Type:** Bugfix
**Severity:** High
**Status:** Done
**Packages:** `internal/ui/`
**Related:** Feature 018 (Notification Panel Redesign), Feature 018A (Notification Card Rendering)

---

## Overview

Fixed the notification panel to render color-coded `NotificationCard` widgets instead of plain-text labels. Added expand/collapse toggle button and per-card dismiss functionality.

## Bug Description

The notification panel rendered plain text labels instead of the color-coded `NotificationCard` widgets. The `NotificationCard` struct with proper importance-based coloring existed in `presenter/notification_card.go` and worked correctly, but `notification_pane.go` ignored it and rendered simple text via the presenter's `Messages()` method instead of `Cards()`.

## Design Decisions

- **Card rendering in list:** Replaced `widget.List` `CreateItem`/`UpdateItem` to use `canvas.Rectangle` backgrounds with card colors and badge colors from the presenter's `Cards()` method instead of plain `widget.Label` via `Messages()`.
- **RenderCard/RenderExpandedCard methods:** Exposed card rendering as testable public methods, enabling structural assertions without needing to introspect Fyne's virtual list internals.
- **cardAt helper:** Extracted shared bounds-checking and card lookup into a private helper to reduce duplication between collapsed and expanded card rendering.

## API

### NotificationPanel (modified)

| Method | Description |
|---|---|
| `RenderCard(index int) fyne.CanvasObject` | Returns collapsed card: Rectangle bg + badge, channel, preview, sender, time |
| `RenderExpandedCard(index int) fyne.CanvasObject` | Returns expanded card: full score, source, channel, sender, time, Dismiss button, word-wrapped content |

### List rendering (modified)

- `CreateItem` now returns `container.NewStack(bg, content)` with placeholder rectangles and labels
- `UpdateItem` populates from `presenter.Cards()` with proper colors
- Length function uses `Cards()` instead of `Messages()`

### Header (modified)

- Header now contains "Notifications" label + "◀ expand" toggle button in HBox
- Toggle button wired to `presenter.ToggleExpanded()`

## Error Handling

- Out-of-range card indices return `nil` (safe for Fyne list recycling)
- Dismiss errors are silently dropped (non-blocking UI operation)

## Test Coverage

| Test | Behavior |
|---|---|
| `TestCardRenderingUsesColoredElements` | RenderCard returns container with canvas.Rectangle |
| `TestPanelContainsExpandToggleButton` | Panel contains button with "expand" text |
| `TestExpandedCardContainsDismissButton` | RenderExpandedCard contains "Dismiss" button |

All existing tests (expand/collapse state, detail dialog, resolve, dismiss via presenter) continue to pass.

## TDD Agent Stats

| TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|
| RED (behavior 1) | Test Designer | ~51s | ~29,000 | 4cad526 |
| GREEN (behavior 1) | Implementer | ~25s | ~22,000 | 70a5fb0 |
| REFACTOR (behavior 1) | Refactorer | ~44s | ~24,000 | be1a5dc |
| RED (behavior 2) | Test Designer | ~55s | ~27,000 | 3021988 |
| GREEN (behavior 2) | Implementer | ~38s | ~25,000 | a38f3ed |
| RED (behavior 3) | Test Designer | ~43s | ~26,000 | b9c7876 |
| GREEN (behavior 3) | Implementer | ~41s | ~23,000 | dc5770a |
| REFACTOR (behavior 3) | Refactorer | manual | — | f60109f |
