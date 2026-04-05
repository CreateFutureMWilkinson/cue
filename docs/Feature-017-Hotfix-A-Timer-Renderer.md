# Feature 017-Hotfix-A: Countdown Timer Renderer

**Phase:** Phase-1-Feature-017-Hotfix-A
**Status:** Planned
**Package:** `internal/ui/`
**Parent:** Feature 017 (Focus Rail + Countdown Timer)

---

## Overview

The `CountdownTimer` widget (Feature 017) has complete segment calculation logic (`Segments()`, `SetProgress()`, `Reset()`) and 14 passing tests, but its `CreateRenderer` returns a stub that draws only a transparent rectangle. The timer ring is never actually painted on screen.

This hotfix implements the real `countdownTimerRenderer` that draws 45 line segments radiating inward from the outer edge of a ring, with future/elapsed coloring and 1Hz flash on the current segment, per the UI-SPEC.md countdown timer specification.

## Design Decisions

### Canvas Lines for Segments

Each segment is a `canvas.Line` positioned using trigonometry from the UI-SPEC rendering pseudocode. Lines radiate **inward** from the outer edge of the ring:

```
outerX = centerX + radius * sin(angle)
outerY = centerY - radius * cos(angle)
innerX = centerX + (radius - length) * sin(angle)
innerY = centerY - (radius - length) * cos(angle)
```

12 o'clock (0°/360°) is at the top. Segments start at 8° and go clockwise.

### Responsive Sizing

The ring radius scales to fit the widget's allocated size rather than using a fixed pixel value. The design token `timer-ring-radius` (120px) is the canonical full-size spec, but the focus rail version scales to fit the 10% rail width (~40-50px radius at 1200w window). The renderer computes:

```go
radius = min(size.Width, size.Height) / 2 * 0.9  // 90% of available space
```

Line lengths scale proportionally from the short/medium/long constants relative to the 120px canonical radius.

### Segment Colors

From UI-SPEC design tokens:
- **Future segments**: `#FFCE1B` (full alpha) — `timer-line` token
- **Elapsed segments**: `RGBA(255, 206, 27, 64)` — `timer-line-dim` token
- **Current segment**: Same as future but with 1Hz flash (500ms on/off toggle)

The existing `SegmentInfo.Color` field from `Segments()` already provides the correct color per segment. The renderer uses these directly.

### 1Hz Flash

The current segment (first non-elapsed segment) flashes at 1Hz — visible for 500ms, hidden for 500ms. The `TimerPresenter.IsFlashVisible()` already implements this logic. The renderer needs a `SetFlashVisible(bool)` method that the UI tick loop calls to toggle the current segment's visibility.

### Line Width

UI-SPEC doesn't specify line width explicitly. Use `StrokeWidth = 2.0` for standard segments and `StrokeWidth = 3.0` for cardinal/diagonal segments to give them visual weight matching their longer length.

## API

### Existing (Unchanged)

```go
// CountdownTimer — all existing methods stay as-is
func NewCountdownTimer() *CountdownTimer
func (t *CountdownTimer) Segments() []SegmentInfo
func (t *CountdownTimer) SetProgress(p float64)
func (t *CountdownTimer) Reset()
func (t *CountdownTimer) MinSize() fyne.Size
func (t *CountdownTimer) CreateRenderer() fyne.WidgetRenderer  // ← this changes internally
```

### New

```go
// SetFlashVisible controls the 1Hz flash state of the current segment.
// Called by the UI tick loop based on TimerPresenter.IsFlashVisible().
func (t *CountdownTimer) SetFlashVisible(visible bool)
```

### Renderer Implementation

```go
type countdownTimerRenderer struct {
    timer *CountdownTimer
    lines [45]*canvas.Line
}

func (r *countdownTimerRenderer) Layout(size fyne.Size)
// Computes center, radius, and positions all 45 lines using trig

func (r *countdownTimerRenderer) Refresh()
// Reads timer.Segments(), updates each line's color and visibility
// Handles flash state for the current segment

func (r *countdownTimerRenderer) Objects() []fyne.CanvasObject
// Returns the 45 canvas.Line objects

func (r *countdownTimerRenderer) MinSize() fyne.Size
func (r *countdownTimerRenderer) Destroy()
```

## Rendering Geometry

Per UI-SPEC section "Countdown Timer Widget → Rendering":

```
For each of 45 segments (i = 0..44):
    angle = (i + 1) * 8°           // segment 0 at 8°, segment 44 at 360°
    angleRad = angle * π / 180

    // Line length by tier
    cardinal (0°, 90°, 180°, 270°): 3× short    = 36px @ full scale
    diagonal (45°, 135°, 225°, 315°): 2× short   = 24px @ full scale
    regular: 1× short                             = 12px @ full scale

    // Scale factor for current widget size
    scale = radius / canonicalRadius  // canonicalRadius = 120px

    // Line endpoints (inward from outer edge)
    outerX = centerX + radius * sin(angleRad)
    outerY = centerY - radius * cos(angleRad)
    innerX = centerX + (radius - length*scale) * sin(angleRad)
    innerY = centerY - (radius - length*scale) * cos(angleRad)
```

Depletion order: clockwise starting at 8° (first segment), 360°/0° (12 o'clock) is last.

## Error Handling

No error conditions — this is pure rendering. If the widget size is zero, `Layout` is a no-op.

## Integration Points

- **`CountdownTimer`** (`countdown_timer.go`): Replace stub renderer with real implementation
- **`FocusRail`** (`focus_rail.go`): Timer is already wired into the rail — once the renderer draws, it will be visible when `SetActivePlan(true)` is called
- **`TimerPresenter`** (`presenter/timer_presenter.go`): Already computes `ElapsedFraction()`, `IsFlashVisible()`, `ActiveSegment()` — the UI tick loop needs to call `timer.SetProgress(fraction)` and `timer.SetFlashVisible(visible)` on each tick

## Test Coverage

Existing 14 tests remain unchanged (they test segment data, not rendering).

New renderer tests:

- `CreateRenderer` returns non-nil renderer with 45 line objects
- `Objects()` returns exactly 45 `canvas.Line` instances
- `Layout` positions lines within allocated size (verify endpoints are within bounds)
- `Layout` with zero size is a no-op (no panic)
- `Layout` scales proportionally — larger size = larger radius
- `Refresh` at 0% progress — all 45 lines have future color
- `Refresh` at 100% progress — all 45 lines have elapsed color
- `Refresh` at 50% progress — ~22 elapsed, ~23 future
- Cardinal lines (90°, 180°, 270°, 360°) are longer than regular lines
- Diagonal lines (45°, 135°, 225°, 315°) are medium length
- `SetFlashVisible(false)` hides the current segment
- `SetFlashVisible(true)` shows the current segment
- `MinSize` returns positive dimensions

## Files

| File | Action |
|---|---|
| `internal/ui/countdown_timer.go` | Modify — replace stub renderer with real line-drawing implementation, add `SetFlashVisible` |
| `internal/ui/countdown_timer_test.go` | Modify — add renderer tests |
