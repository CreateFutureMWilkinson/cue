# Feature 016: Three-Column Layout

**Phase:** Phase-1-Feature-016
**Status:** Planned
**Packages:** `internal/ui/`

---

## Overview

Restructure the main window from its current layout to the three-column layout defined in `docs/UI-SPEC.md`. The window is divided into a focus rail (10% width), character area (60% width), and notification panel (30% width). A center view router manages which content occupies the character area column (Character view, Plan view, or Wizard). This is the foundational UI change that all subsequent UI features depend on.

## Design Decisions

- **Percentage-based column widths** — focus rail 10%, character area 60%, notifications 30%. These proportions hold at the default 1200x800 window size and scale with resizing.
- **Center view router as a state machine** — a `CenterViewRouter` manages which view occupies the center 60% column. Only one view is active at a time: Character (default), Plan, or Wizard. The focus rail and notification column remain unchanged across all center views.
- **Focus rail is a shell in this feature** — the rail column is created with placeholder content. Full focus rail widgets (timer ring, task name, navigation buttons) are implemented in Feature 017.
- **Notification panel retains current behavior** — the notification pane is moved into the right column but keeps its existing widget structure. The redesign (color-coded cards, collapsed/expanded states) is Feature 018.
- **Character area hosts the character widget** — the existing character widget (Feature 014) is placed in the center area. The activity log drawer conversion is Feature 019.
- **No tab bar** — navigation between views is handled by buttons in the focus rail and contextual controls on each pane, per UI-SPEC.md.

## API

### CenterViewRouter

```go
type CenterView int

const (
    ViewCharacter CenterView = iota  // Default: fairy + activity log drawer
    ViewPlan                          // Plan overview + todo list (Feature 022)
    ViewWizard                        // Day planner wizard (Feature 022)
)

type CenterViewRouter struct {
    currentView CenterView
    onViewChange func(CenterView)
}

func NewCenterViewRouter() *CenterViewRouter
func (r *CenterViewRouter) CurrentView() CenterView
func (r *CenterViewRouter) NavigateTo(view CenterView)
func (r *CenterViewRouter) SetOnViewChange(fn func(CenterView))
```

### Updated MainWindow

```go
// NewMainWindow now creates the three-column layout.
// The center area content is swapped by the CenterViewRouter.
func NewMainWindow(
    app fyne.App,
    notificationPresenter NotificationPresenter,
    activityPresenter ActivityPresenter,
    feedbackPresenter FeedbackPresenter,
    settingsPresenter SettingsPresenter,
    characterWidget fyne.CanvasObject,
    viewRouter *CenterViewRouter,
) fyne.Window
```

## Layout

```
┌──────────────────────────────────────────────────────────────────┐
│  Cue  [Settings] [About] [Quit]                        Menu Bar │
├──────┬───────────────────────────────────────────┬───────────────┤
│      │                                           │               │
│      │                                           │  Notification │
│      │                                           │  Panel        │
│ Focus│         Character Area                    │  (existing    │
│ Rail │         (center view)                     │   widgets)    │
│      │                                           │               │
│      │                                           │               │
│      │                                           │               │
│      │                                           │               │
├──────┴───────────────────────────────────────────┴───────────────┤

  Focus rail: 10% width
  Character area: 60% width
  Notifications: 30% width
```

## Error Handling

| Scenario | Behavior |
|---|---|
| Invalid view navigation | Log warning, remain on current view |
| Character widget nil | Show empty placeholder in center area |
| Window resize below minimum | Fyne handles minimum size constraints |

## Integration Points

- **Fyne GUI (Feature 011):** Replaces the existing window layout. All existing presenters are reused.
- **Character System (Feature 014):** Character widget placed in center area as default view.
- **Focus Rail (Feature 017):** Focus rail column created here; populated with widgets in Feature 017.
- **Notification Panel Redesign (Feature 018):** Notification column created here; redesigned in Feature 018.
- **Activity Log Drawer (Feature 019):** Activity log moved into center area as drawer in Feature 019.
- **Planner UI (Feature 022):** Plan and Wizard views registered with the CenterViewRouter.

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `ui` | `CenterViewRouterSuite` | Constructor defaults to ViewCharacter, NavigateTo changes current view, callback fires on view change, invalid view handling |
| `ui` | `ThreeColumnLayoutSuite` | Layout creates three columns, column proportions correct, center area swaps on view change, menu bar present |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| CenterViewRouter | RED | Test Designer | — | — | — |
| CenterViewRouter | GREEN | Implementer | — | — | — |
| CenterViewRouter | REFACTOR | Refactorer | — | — | — |
| Layout | RED | Test Designer | — | — | — |
| Layout | GREEN | Implementer | — | — | — |
| Layout | REFACTOR | Refactorer | — | — | — |
