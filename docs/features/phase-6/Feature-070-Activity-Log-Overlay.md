# Feature 070 — Activity Log Drawer Uses Split Instead of Overlay

**Phase:** Phase-6-Feature-070
**Type:** Bugfix
**Severity:** High
**Depends on:** Feature 019

## Problem

The Activity Log button is supposed to trigger the activity log overlaying the character window with a semi-transparent (30% transparency) view of activity log entries. Instead, the current implementation uses a `container.NewVSplit` that pushes the character area up (60/40 split), hiding part of it rather than overlaying it.

## Root Cause

`activity_log_drawer.go:66-68`:
```go
split := container.NewVSplit(character, d.drawerBox)
split.Offset = 0.6
```

This creates a vertical split that shares space with the character, rather than overlaying the character with a semi-transparent background.

## Fix

1. Replace the VSplit layout with a Stack layout.
2. When the drawer is closed: show only the toggle button anchored at the bottom of the character area.
3. When the drawer is open: overlay the character area with:
   - A semi-transparent background rectangle (30% opacity / 70% transparent — `RGBA(0, 0, 0, 77)`)
   - The activity log list on top of the background
   - The close button at the top of the overlay
4. Use `container.NewStack()` to layer character → overlay.
5. The character widget remains fully rendered underneath; the overlay sits on top.

## Files to Change

- `internal/ui/activity_log_drawer.go` — replace VSplit with Stack overlay
- `internal/ui/activity_log_drawer_test.go` — update tests for overlay behavior (if applicable)

## Acceptance Criteria

- [ ] Activity log button toggles an overlay on top of the character area
- [ ] Overlay has a semi-transparent dark background (30% transparency)
- [ ] Activity log entries are visible and readable over the overlay
- [ ] Character widget is visible underneath the overlay (not pushed/hidden)
- [ ] Close button dismisses the overlay, restoring full character view
- [ ] Drawer occupies ~40% of character area height when open (from bottom)
