# Feature 018-Hotfix-A: Notification Card Visual Rendering

**Phase:** Phase-1-Feature-018-Hotfix-A
**Status:** Done
**Package:** `internal/ui/`
**Parent:** Feature 018 (Notification Panel Redesign)

---

## Overview

Feature 018 delivered the `NotificationCard` view model and `NotificationPresenter` with full color calculation logic (importance-based background colors, opacity scaling, relative timestamps). However, the actual Fyne view layer in `notification_pane.go` uses simplified label-based rendering rather than styled card widgets matching the UI-SPEC.

The presenter correctly computes:
- Background colors: red (#ffc9c9 for IS>=9), orange (#ffd8a8 for IS>=8), blue (#dbe4ff for IS<8)
- Opacity scaling: 0.2-0.4 linear with IS
- Relative timestamps
- Message previews with truncation

But the view doesn't apply these computed styles to the rendered cards.

## Issues to Fix

### 1. Collapsed Card Rendering

**UI-SPEC requirement (collapsed, 30% width):**
```
│ [9] #alerts   │
│  Added to...  │
```

Each card should show:
- Color-coded IS badge (background color from NotificationCard.BackgroundColor)
- Channel name
- Message preview (truncated)
- Source icon or label

**Current:** Cards rendered as plain text labels without background color or badge styling.

**Fix:** Create a custom `notificationCardWidget` that renders:
- A colored rectangle background using the card's `BackgroundColor` RGBA
- IS badge as a colored label or mini rectangle
- Channel + preview text
- Proper padding and spacing per design tokens

### 2. Expanded Card Rendering

**UI-SPEC requirement (expanded, 90% width):**
```
│  [9.0]  slack  #alerts   bot         2m ago   [Dismiss] │
│         You were added to #alerts                        │
```

Each expanded card should show:
- Full IS score with one decimal
- Source label
- Channel name
- Sender
- Relative timestamp
- Dismiss button
- Full message preview (second line)

**Current:** Expanded state uses the same simplified rendering.

**Fix:** Create an expanded card layout with all fields from `NotificationCard`:
- `FormattedScore` (already computed by presenter)
- `Source`, `Channel`, `Sender`
- `RelativeTime` (already computed by presenter)
- `MessagePreview`
- Dismiss button calling `presenter.DismissMessage(card.ID)`

### 3. Detail Dialog Styling

**UI-SPEC requirement:**
- Modal dialog showing full message content
- IS and CS scores displayed
- Reasoning text
- Resolve button

**Current status needs verification** — check if the detail dialog uses card styling.

## Files

| File | Action |
|---|---|
| `internal/ui/notification_pane.go` | Modify — replace label-based cards with styled card widgets using NotificationCard view model colors |
| `internal/ui/notification_pane_test.go` | Modify — add tests for card background colors and expanded layout |

## Test Coverage

### Presenter Cards Suite (`notification_presenter_test.go`)
- `TestCardsReturnsNotificationCards` — Cards() returns correct count and content
- `TestCardsHasCorrectColors` — IS>=9 gets red (#ffc9c9/#ef4444), IS<8 gets blue (#dbe4ff/#4a9eed)
- `TestCardsHasRelativeTime` — Cards have non-empty RelativeTime strings
- `TestSelectReturnsReasoning` — Select() populates Reasoning from repository message

### Notification Pane Suite (`notification_pane_test.go`)
- `TestCollapsedCardShowsBadgeText` — Collapsed cards expose IS and channel via Cards()
- `TestExpandedCardShowsFullScore` — Expanded cards expose IS, source, sender via Cards()
- `TestExpandedCardShowsDismissAction` — Dismiss by card ID removes card from list
- `TestDetailDialogShowsReasoning` — Select() detail includes reasoning text

**Total new tests: 8**

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Cards + Reasoning | RED | Test Designer | 252s | 35,591 | 0305024 |
| Cards + Reasoning | GREEN | Implementer | 48s | 23,302 | 738e36f |
| Cards + Reasoning | REFACTOR | Refactorer | 195s | 30,732 | 4667be4 |
