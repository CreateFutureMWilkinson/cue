# Feature 021: Jar Rendering

**Phase:** Phase-3-Feature-021
**Status:** Planned
**Packages:** `internal/ui/character/`

---

## Overview

Replaces the current 40px colored circle fairy with a layered jar composition using SVG assets. The jar is rendered as two layers (back and front) with the fairy body and glow circles sandwiched between them, creating the illusion of a fairy trapped inside a glass jar. The fairy body (inner circle) is constrained to the jar interior bounds; the glow (outer circle) can overlap the jar edges. SVGs are used for crisp scaling at any size.

## Design Decisions

- **Three-layer rendering** — jar_back.svg (background), fairy circles (middle), jar_front.svg (foreground). Fyne's `container.NewStack` or `container.NewWithoutLayout` with explicit positioning achieves the z-ordering.
- **SVG for scaling** — SVG assets scale cleanly at any resolution. Loaded via Fyne's `canvas.NewImageFromResource` or `canvas.NewImageFromFile`. PNG fallbacks exist but SVGs are preferred.
- **Fairy body is 10% of jar width** — the inner solid circle diameter scales proportionally to the jar container width.
- **Fairy glow is 25% of jar width** — the outer translucent circle diameter. The glow can extend beyond jar interior bounds (overlaps jar edges) since it renders between the back and front SVG layers.
- **Jar interior bounds calculated from asset proportions** — the fairy body must stay within the glass area of the jar (excluding cork and rim). This is a fixed proportional region derived from the SVG artwork: approximately the middle 60% height (below cork, above jar base) and 70% width (inside glass walls).
- **Initial fairy color: #006100** — the darkest green, representing the idle/rest state. Glow is the same hue with radial alpha falloff.
- **Fairy position is managed by animation features** — this feature only establishes the rendering structure and a `SetPosition(x, y float64)` method. Position values are normalized (0.0–1.0) within the jar interior bounds.
- **Glow rendered as concentric translucent circles** — simulates radial gradient by drawing multiple concentric circles with decreasing alpha from center outward. Pure Fyne canvas primitives, no shader dependency.

## Asset Paths

```
build_assets/images/jar_back.svg    # Jar background (glass, rear wall)
build_assets/images/jar_front.svg   # Jar foreground (glass reflections, rim, cork)
build_assets/images/jar_back.png    # PNG fallback
build_assets/images/jar_front.png   # PNG fallback
```

## Rendering Stack

```
┌──────────────────────────────────────┐
│  jar_front.svg  (top layer, z=2)     │  ← Glass reflections, cork, rim
│                                      │
│    ┌ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┐    │
│    │  Glow circle (z=1)        │    │  ← Translucent green, can overlap edges
│    │    ┌─────────────┐        │    │
│    │    │ Body circle │        │    │  ← Solid green, constrained to interior
│    │    └─────────────┘        │    │
│    └ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┘    │
│                                      │
│  jar_back.svg  (bottom layer, z=0)   │  ← Jar body, back wall
└──────────────────────────────────────┘
```

## Jar Interior Bounds

The fairy body is constrained to the glass interior of the jar, excluding the cork and rim:

```
┌──────────────────┐
│     ████████     │  ← Cork (excluded)
│    ──────────    │  ← Rim (excluded)
│   ┌──────────┐   │
│   │          │   │  ← Interior bounds
│   │   Body   │   │     ~70% width
│   │ movement │   │     ~60% height
│   │  region  │   │     offset from top ~30%
│   │          │   │
│   └──────────┘   │
│                  │  ← Base (excluded)
└──────────────────┘
```

Proportional constants (derived from SVG artwork):
```go
const (
    jarInteriorLeft   = 0.15  // 15% from left edge
    jarInteriorRight  = 0.85  // 85% from left edge
    jarInteriorTop    = 0.30  // 30% from top (below cork+rim)
    jarInteriorBottom = 0.92  // 92% from top (above base)
)
```

## API

### FairyCharacter (Modified)

```go
// NewFairyCharacter now returns a jar-based fairy instead of a plain circle.
func NewFairyCharacter() *FairyCharacter

// SetPosition sets the fairy's position within the jar interior.
// x, y are normalized 0.0–1.0 within interior bounds.
// (0.0, 0.0) = top-left of interior; (1.0, 1.0) = bottom-right.
// (0.5, 1.0) = bottom-center (rest position on jar floor).
func (f *FairyCharacter) SetPosition(x, y float64)

// Position returns the current normalized position.
func (f *FairyCharacter) Position() (x, y float64)

// SetBodyColor sets the fairy body color (inner circle).
func (f *FairyCharacter) SetBodyColor(c color.Color)

// SetGlowIntensity sets the glow opacity multiplier (0.0–1.0).
func (f *FairyCharacter) SetGlowIntensity(intensity float64)

// Widget returns the layered jar + fairy container.
func (f *FairyCharacter) Widget() fyne.CanvasObject
```

### Glow Rendering

```go
const glowLayers = 8  // concentric circles for gradient simulation

// Glow is rendered as 8 concentric circles:
// - Innermost: same color as body, alpha ~128
// - Each successive ring: same hue, alpha decreasing by ~16
// - Outermost: alpha ~0 (fully transparent)
```

## Color Specification

| State | Body Color | Usage |
|---|---|---|
| Idle / Rest | `#006100` | Darkest green (this feature's default) |
| Working | Intermediate (Feature 023) | Brighter green |
| Notifying | `#00C300` | Brightest green (Feature 024) |
| Error | Near `#00C300` (Feature 025) | Close to notification |
| ShuttingDown | `#004900` | Darkest (Feature 026) |

This feature initializes the fairy at `#006100`. Color transitions are handled by subsequent features.

## Error Handling

| Scenario | Behavior |
|---|---|
| SVG file not found | Fall back to PNG; if PNG also missing, render jar as a plain rectangle outline |
| Position out of range (>1.0 or <0.0) | Clamped to [0.0, 1.0] |
| Glow intensity out of range | Clamped to [0.0, 1.0] |
| Widget resize | All proportions recalculate on `Resize()` — SVGs scale, circle sizes update |

## Integration Points

- **Character Interface (Feature 014):** `FairyCharacter` continues to implement `Character` interface. `Widget()` now returns the jar composition instead of a plain circle.
- **Main Window (Feature 011):** No changes needed — the character widget is placed via the existing `container.NewBorder` layout.
- **UAT Harness (Feature 020):** Fairy is testable via the existing registry. Jar rendering is immediately visible in the UAT window.
- **Subsequent Fairy Features (021–026):** All animation features call `SetPosition()`, `SetBodyColor()`, and `SetGlowIntensity()` to animate the fairy within the jar.

## Test Coverage Plan

| Package | Suite | Expected Tests |
|---|---|---|
| `character` | `FairyJarSuite` | Widget returns non-nil container, SVG layers present, body circle sized at 10% of jar width, glow circle sized at 25% of jar width, SetPosition clamping, Position round-trip, SetBodyColor applied, SetGlowIntensity clamping, glow layer count, initial color is #006100, initial position is bottom-center (0.5, 1.0) |

## TDD Agent Stats

| TDD Cycle | Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Jar Rendering | RED | Test Designer | — | — | — |
| Jar Rendering | GREEN | Implementer | — | — | — |
| Jar Rendering | REFACTOR | Refactorer | — | — | — |
