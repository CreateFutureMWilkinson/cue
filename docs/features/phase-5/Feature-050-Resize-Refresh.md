# Feature 050: Resize Refresh

**Phase:** Phase-5-Feature-050
**Status:** Done
**Packages:** `internal/ui/character/fairy/`, `cmd/character-uat/`

---

## Overview

When the Fyne application window is resized, the fairy character's jar PNG images and animation circles do not re-render at the new size. This feature fixes the resize refresh behaviour so that all visual elements update correctly on window resize, and ensures the character-uat harness embeds the character widget identically to the main application.

## Root Cause

Two issues contribute to the broken resize behaviour:

1. **Jar images not refreshed on layout:** In `fairyJarLayout.positionJarLayers`, the jar `canvas.Image` objects are resized via `Resize()` but never explicitly `Refresh()`-ed. Fyne's `canvas.Image` caches rendered pixels and requires an explicit `Refresh()` call to re-render at a new size.

2. **UAT embedding divergence:** The character-uat harness wraps the character widget in `container.NewCenter()`, which constrains the widget to its minimum size and centres it. The main application uses a space-filling `VSplit` layout instead. This means the character widget in the UAT does not fill available space and does not exercise the same resize path as production.

## Design

### Behavior 1: Jar and Circle Refresh on Layout

Add `Refresh()` calls to jar images after `Resize()` in `positionJarLayers`. Both the main application and UAT harness benefit automatically since they consume the character widget through the `character.Character` interface's `Widget()` method.

**Changed file:** `internal/ui/character/fairy/layout.go`

### Behavior 2: UAT Embedding Parity

Replace `container.NewCenter(w.charContainer)` with `container.NewStack(w.charContainer)` in the UAT harness so the character widget fills the available left-panel space, matching how the main application embeds it.

**Changed file:** `cmd/character-uat/uat_window.go`

## Architectural Notes

- Neither behaviour references the `fairy` package directly from outside its own package. The fix in `layout.go` is internal to the fairy implementation. The UAT fix operates on the `fyne.CanvasObject` returned by the `character.Character.Widget()` interface method.
- The `fairy` package import exists only in composition roots (`cmd/cue-uat/main.go` for registration, test files for setup) — this is preserved.

## Test Strategy

- **Behavior 1:** Test that after resizing the fairy container to a new size, jar images report the updated size (verifying layout ran) and a layout refresh hook confirms `Refresh()` was called.
- **Behavior 2:** Test that the UAT left panel uses `container.NewStack` (space-filling) rather than `container.NewCenter` (min-size constrained).

## Error Handling

No new error paths introduced. Layout changes are purely visual.

## Integration Points

- `internal/ui/character/fairy/layout.go` — shared by all consumers of `fairy.Widget()`
- `cmd/character-uat/uat_window.go` — UAT harness layout
- `internal/ui/window.go` — main application layout (unchanged, already correct)

## TDD Agent Stats

| Implementation Phase | TDD Phase | Agent | Duration | Tokens | Commit |
|---|---|---|---|---|---|
| Phase-5-Feature-050 | RED | Test Designer | ~53s | ~32,000 | cc9ec3b |
| Phase-5-Feature-050 | GREEN | Implementer | ~57s | ~31,000 | 29c5387 |
| Phase-5-Feature-050 | REFACTOR | Refactorer | ~35s | ~28,000 | (no changes) |
| Phase-5-Feature-050 | RED | Test Designer | ~60s | ~27,000 | 38a2549 |
| Phase-5-Feature-050 | GREEN | Implementer | ~46s | ~26,000 | 4077ffb |
| Phase-5-Feature-050 | REFACTOR | Refactorer | ~34s | ~24,000 | (no changes) |
