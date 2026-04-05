# Feature 018: Notification Panel Redesign

**Phase:** Phase-1-Feature-018
**Status:** Done
**Packages:** `internal/ui/`, `internal/ui/presenter/`

---

## Overview

Redesign the notification panel from simple list rows to color-coded cards with two states: collapsed (30% width, compact cards) and expanded (90% width replacing the character area, full cards with dismiss buttons). Cards are colored by importance score. Clicking any card opens a detail dialog modal showing full message content, scores, and a Resolve button. The expand/collapse toggle drives center area visibility and focus rail Review button state.

## Design Decisions

- **Two-state panel (collapsed/expanded)** — collapsed is the default at 30% width showing compact cards. Expanded takes 90% width by hiding the character area. This avoids a separate "notification detail view" and keeps the three-column layout intact.
- **Color-coded cards by importance** — IS >= 9 gets red tint (`#ffc9c9`), IS >= 8 gets orange tint (`#ffd8a8`), IS < 8 gets blue tint (`#dbe4ff`). Badge colors match. Provides instant visual priority scanning.
- **Card background opacity fades with lower IS** — IS 9 = 40% opacity, IS 7 = 20% opacity. Subtle visual hierarchy within each color tier.
- **Detail dialog is a modal** — blocks main window interaction. Shows full message content without truncation, IS, CS, timestamp, and Resolve button. Same dialog accessed from both collapsed and expanded states.
- **Dismiss button only in expanded state** — quick triage without opening the dialog. Marks message as Resolved and removes from list.
- **Expand/collapse drives other component state** — expanding hides the character area and shows the Review button in the focus rail. Collapsing restores both.
- **30-second refresh interval** — notification list refreshes on a 30s tick or on manual action (dismiss/resolve).

## API

### Updated NotificationPresenter

```go
// Extended methods on the existing NotificationPresenter

func (p *NotificationPresenter) IsExpanded() bool
func (p *NotificationPresenter) ToggleExpanded()
func (p *NotificationPresenter) SetOnExpandedChange(fn func(bool))
func (p *NotificationPresenter) DismissMessage(ctx context.Context, id uuid.UUID) error
```

### NotificationCard (View Model)

```go
type NotificationCard struct {
    ID              uuid.UUID
    ImportanceScore float64
    Source          string
    Channel         string
    Sender          string
    MessagePreview  string      // truncated for collapsed view
    FullContent     string      // for detail dialog
    ConfidenceScore float64
    CreatedAt       time.Time
    RelativeTime    string      // "2m ago", "12m ago"
    CardColor       color.Color // based on IS tier
    BadgeColor      color.Color // based on IS tier
    Opacity         float64     // 0.2–0.4 based on IS
}
```

## Layout

### Collapsed State (30% width)

```
┌───────────────┐
│ Notifs (4)    │
│───────────────│
│ [9] #alerts   │
│  Added to...  │
│  bot  2m ago  │
│───────────────│
│ [8.5] Inbox   │
│  Server down  │
│  alice  5m    │
│───────────────│
│  [◀ expand]   │
└───────────────┘
```

### Expanded State (90% width)

```
┌─────────────────────────────────────────────────────────────┐
│  Notifications (4)                               [collapse ▶]│
│─────────────────────────────────────────────────────────────│
│  [9.0]  slack  #alerts   bot         2m ago       [Dismiss] │
│         You were added to #alerts                            │
│─────────────────────────────────────────────────────────────│
│  [8.5]  email  Inbox     alice@ex    5m ago       [Dismiss] │
│         URGENT: Server down, need immediate action           │
└─────────────────────────────────────────────────────────────┘
```

### Detail Dialog (Modal)

```
┌──────────────────────────────────────┐
│  Message Detail                       │
│                                       │
│  Importance Score: 8.5                │
│  Confidence Score: 0.92               │
│  Created: 2026-03-28 14:32:05         │
│                                       │
│  ┌────────────────────────────────┐   │
│  │ Full message content here,     │   │
│  │ word-wrapped, no truncation.   │   │
│  └────────────────────────────────┘   │
│                                       │
│              [ Resolve ]              │
└──────────────────────────────────────┘
```

### Card Color by Importance

| Importance | Card Background | Badge Color |
|---|---|---|
| IS >= 9 | `#ffc9c9` (light red) | `#ef4444` (red) |
| IS >= 8 | `#ffd8a8` (light orange) | `#f59e0b` (amber) |
| IS < 8 | `#dbe4ff` (light blue) | `#4a9eed` (blue) |

## Interactions

| Action | State | Behavior |
|---|---|---|
| Click expand toggle | Collapsed | Expand to 90%, hide character, show Review in focus rail |
| Click collapse toggle | Expanded | Collapse to 30%, restore character, hide Review |
| Click compact card | Collapsed | Open detail dialog modal |
| Click expanded card | Expanded | Open detail dialog modal |
| Click Dismiss | Expanded | Mark as Resolved, remove from list |
| Click Resolve (dialog) | Either | Mark as Resolved, remove from list, close dialog |
| List refresh | Either | 30s tick or manual action |

## Error Handling

| Scenario | Behavior |
|---|---|
| Dismiss fails (DB error) | Log error, keep card in list, show error in activity log |
| No notifications | Show empty state text in panel |
| Resolve fails (DB error) | Log error, keep dialog open, show error message |

## Integration Points

- **Three-Column Layout (Feature 016):** Notification panel occupies the right 30% column. Expansion takes over the character area's 60%.
- **Focus Rail (Feature 017):** Expand/collapse state drives Review button visibility via `SetOnExpandedChange` callback.
- **Existing NotificationPresenter (Feature 011):** Extended with expand/collapse and dismiss methods. Existing query/resolve logic reused.
- **UI-SPEC.md:** Authoritative reference for card format, color tokens, and interaction behavior.

## Test Coverage Summary

| Package | Suite | Tests |
|---|---|---|
| `presenter` | `NotificationPresenterExpandSuite` | 8 — default collapsed, toggle expands, toggle collapse, expanded change callback fires, dismiss marks resolved, dismiss removes from list, dismiss unknown ID error, dismiss updater error propagates |
| `presenter` | `NotificationCardSuite` | 10 — IS >= 9 red card/badge, IS >= 8 orange card/badge, IS < 8 blue card/badge, opacity scales with IS, relative time (minutes/hours/just now), message preview truncated, card fields populated, empty messages |
| `ui` | `NotificationPaneSuite` | 7 — panel not nil, default collapsed, toggle expand, toggle collapse, container not nil, detail dialog content, resolve removes message |

**Total new tests: 25**

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Presenter Expand | RED | Test Designer | 164s | 26,503 | 9069a68 |
| Presenter Expand | GREEN | Implementer | 34s | 22,503 | 098f77c |
| Presenter Expand | REFACTOR | Refactorer | 54s | 31,995 | 7246a44 |
| Card View | RED | Test Designer | 63s | 22,635 | dd31874 |
| Card View | GREEN | Implementer | 40s | 23,636 | d6d4055 |
| Card View | REFACTOR | Refactorer | 102s | 37,573 | 1175b57 |
| Detail Dialog | RED | Test Designer | 61s | 28,659 | f2aed38 |
| Detail Dialog | GREEN | Implementer | 34s | 22,055 | 3317a8d |
| Detail Dialog | REFACTOR | Refactorer | 93s | 26,540 | ce71403 |
