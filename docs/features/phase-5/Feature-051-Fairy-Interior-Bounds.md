# Feature 051: Fairy Interior Bounds

**Phase:** Phase-5-Feature-051
**Status:** Planned
**Packages:** `internal/ui/character/fairy/`

---

## Overview

The fairy character's animation currently uses the full container bounds (0.0–1.0 normalized coordinates) for movement. Because the jar images use `ImageFillContain`, the actual rendered jar occupies a subset of the container when the aspect ratio doesn't match. This causes the fairy to fly outside the visible jar area.

This feature constrains the fairy's movement and rendering to the jar image's visible interior region, accounting for both `ImageFillContain` letterboxing and the jar's glass walls/lid.

## Problem

Two coordinate mismatches exist:

1. **Letterboxing mismatch:** The jar image maintains its aspect ratio via `ImageFillContain`, so it may not fill the full container. The fairy's normalized coordinates map to the full container, placing the fairy in empty space outside the rendered jar.

2. **No interior region:** Even within the rendered jar image, the fairy should only move inside the glass — not overlap the jar walls, lid, or base. No interior bounds are currently enforced.

## Design

### Behavior 1: Jar Rendered Rect Calculation

Compute the actual pixel rectangle where the jar image renders within the container, replicating Fyne's internal `ImageFillContain` letterboxing logic. This uses `canvas.Image.Aspect()` and the container dimensions.

```go
func jarRenderedRect(containerW, containerH, imgAspect float32) (x, y, w, h float32) {
    containerAspect := containerW / containerH
    if containerAspect > imgAspect {
        // Container wider than jar — pillarboxed (gaps on sides)
        h = containerH
        w = containerH * imgAspect
        x = (containerW - w) / 2
    } else {
        // Container taller than jar — letterboxed (gaps top/bottom)
        w = containerW
        h = containerW / imgAspect
        y = (containerH - h) / 2
    }
    return
}
```

**Changed file:** `internal/ui/character/fairy/layout.go`

### Behavior 2: Interior Bounds Mapping

Define the jar's interior region as proportions of the rendered jar image:

| Edge | Proportion | Pixels (unscaled) | Description |
|---|---|---|---|
| Top | 0.2943 | 234px | Below the jar lid |
| Bottom | 0.9119 | 725px (from top) | Above the jar base |
| Left | 0.0693 | 26px | Inside left wall |
| Right | 0.9307 | 349px (from left) | Inside right wall |

The fairy's normalized position (0.0–1.0) maps into this interior region within the rendered jar rect:

```
pixelX = jarX + (interiorLeft + posX * (interiorRight - interiorLeft)) * jarW
pixelY = jarY + (interiorTop + posY * (interiorBottom - interiorTop)) * jarH
```

This replaces the current `posX * containerWidth` calculation in `positionCircle()`.

**Changed file:** `internal/ui/character/fairy/layout.go`

### Constants

```go
const (
    jarInteriorTop    = 0.2943
    jarInteriorBottom = 0.9119
    jarInteriorLeft   = 0.0693
    jarInteriorRight  = 0.9307
)
```

## Architectural Notes

- All changes are internal to the fairy package layout. No public API changes.
- Animators are unaffected — they continue to emit normalized 0.0–1.0 positions. The remapping happens entirely in the layout's circle positioning.
- The jar rendered rect is recalculated on every `Layout()` call, so resize is handled automatically.
- `ImageFillContain` letterboxing logic is replicated from Fyne's `internal/painter/software/draw.go` since Fyne does not expose rendered bounds publicly.

## Test Strategy

- **Behavior 1:** Test that `jarRenderedRect` correctly computes the rendered area for containers wider than the jar (pillarboxing), taller than the jar (letterboxing), and matching aspect ratio (no gaps).
- **Behavior 2:** Test that `positionCircle` places the fairy body and glow circles within the interior bounds of the computed jar rect. Verify at boundary positions: (0,0) maps to interior top-left, (1,1) maps to interior bottom-right, (0.5,0.5) maps to interior center.

## Error Handling

No new error paths. If `Aspect()` returns 0 (no image loaded), fall back to full-container positioning to avoid division by zero.

## Integration Points

- `internal/ui/character/fairy/layout.go` — sole changed file, shared by all consumers of `fairy.Widget()`
- All animators (`idle`, `working`, `notify`, `error`, `startup`, `shutdown`) benefit automatically without modification
