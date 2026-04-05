# UI Design Guide — Designing for Claude Implementation

This guide documents how to create UI designs that Claude can understand and implement, and how to validate the results with acceptance tests. It covers design formats, the Excalidraw MCP integration, and the Fyne testing strategy.

---

## 1. Design Formats (Ranked by Effectiveness)

### Tier 1: Structured Markdown + ASCII Wireframes (Primary)

**This is what Claude implements from.** The `UI-SPEC.md` file is the authoritative design artifact. It contains:

- **ASCII wireframes** using box-drawing characters (`┌ ─ ┐ │ └ ┘ ├ ┤ ┬ ┴ ┼`) for layout structure
- **Design tokens** (colors, typography, spacing) in tables
- **Component specs** with widget types, data formats, and interaction tables
- **Acceptance criteria** as checkboxes that map to tests

ASCII wireframes are 10–50x more token-efficient than images and parse unambiguously. Claude reads them as layout structure, not decoration.

**How to write effective wireframes:**

```
┌─────────────────────────┬───────────────────────┐
│  Left Pane (50%)        │  Right Pane (50%)     │
│                         │                       │
│  [widget.List]          │  [widget.List]        │
│  - Row 1                │  - Entry 1            │
│  - Row 2                │  - Entry 2            │
│                         │                       │
├─────────────────────────┴───────────────────────┤
│              [ Button Label ]                    │
└─────────────────────────────────────────────────┘
```

Rules:
- Annotate container types (HSplit, VBox, Border) in comments or headers
- Show representative data in the wireframe, not lorem ipsum
- Note percentages/sizes for splits
- Label widget types where non-obvious

### Tier 2: Excalidraw Diagrams (Supplementary Visual Reference)

Use the Excalidraw MCP server to create visual wireframes for collaborative iteration. These supplement the markdown spec — they don't replace it.

**Best for:** Layout exploration, discussing changes visually, progressive reveal of complex flows.

### Tier 3: Screenshots / Images (Reference Only)

Claude can read screenshots (multimodal vision), but they're token-heavy and imprecise for exact values. Use them when replicating an existing UI or showing a reference design to match.

### Tier 4: HTML/CSS Mockups (Not Recommended for Fyne)

Functional for web apps but translates poorly to Fyne widget code. Skip this for Cue.

---

## 2. Excalidraw MCP Integration

Cue has access to an Excalidraw MCP server with five tools for creating, saving, and reading back diagrams.

### Available Tools

| Tool                  | Purpose                                                    |
|-----------------------|------------------------------------------------------------|
| `read_me`             | Returns element format reference, color palette, usage tips |
| `create_view`         | Renders a diagram from a JSON element array                |
| `export_to_excalidraw`| Uploads diagram to excalidraw.com, returns shareable URL   |
| `save_checkpoint`     | Saves current diagram state (including user edits)         |
| `read_checkpoint`     | Reads back a previously saved checkpoint                   |

### Element Format

Diagrams are JSON arrays of Excalidraw elements:

```json
[
  {
    "type": "rectangle",
    "id": "pane-left",
    "x": 0, "y": 0,
    "width": 400, "height": 600,
    "label": { "text": "Notification Queue" }
  },
  {
    "type": "rectangle",
    "id": "pane-right",
    "x": 420, "y": 0,
    "width": 400, "height": 600,
    "label": { "text": "Activity Log" }
  },
  {
    "type": "arrow",
    "id": "flow-1",
    "x": 200, "y": 620,
    "width": 200, "height": 0,
    "startBinding": { "elementId": "pane-left", "fixedPoint": [0.5, 1] },
    "endBinding": { "elementId": "pane-right", "fixedPoint": [0.5, 1] }
  }
]
```

**Supported types:** `rectangle`, `ellipse`, `diamond`, `text`, `arrow`

**Pseudo-elements:** `cameraUpdate` (viewport control), `delete` (remove by ID), `restoreCheckpoint` (load saved state)

### Color Palette

| Color   | Hex       | Pastel Fill   |
|---------|-----------|---------------|
| Blue    | `#4a9eed` | `#a5d8ff`     |
| Amber   | `#f59e0b` | `#fff3bf`     |
| Green   | `#22c55e` | `#b2f2bb`     |
| Red     | `#ef4444` | `#ffc9c9`     |
| Purple  | `#8b5cf6` | `#d0bfff`     |
| Pink    | `#ec4899` | `#eebefa`     |
| Cyan    | `#06b6d4` | `#c3fae8`     |
| Lime    | `#84cc16` | (none)        |

### Design Iteration Workflow

The round-trip workflow for collaborative design:

```
1. Claude creates wireframe          →  create_view(elements)
                                          ↓ returns checkpointId
2. User edits in Excalidraw editor   →  (fullscreen edit mode)
                                          ↓
3. Claude reads edits back           →  read_checkpoint(id)
                                          ↓ returns modified JSON
4. Claude interprets layout changes  →  Updates UI-SPEC.md + Fyne code
                                          ↓
5. Repeat as needed                  →  save_checkpoint(id) to preserve state
```

**Step-by-step usage:**

1. Ask Claude to draw the UI layout:
   > "Draw a wireframe of the notification queue pane with 3 example rows"

2. Claude calls `create_view` with Excalidraw elements. The diagram renders in the Excalidraw viewer.

3. Open fullscreen mode and rearrange elements, resize panes, add annotations.

4. Ask Claude to read the changes:
   > "Read back the checkpoint and update the UI spec to match"

5. Claude calls `read_checkpoint`, interprets the geometric layout (positions, sizes, labels), and updates `UI-SPEC.md` accordingly.

### Constraints

- No emoji in text labels (won't render)
- Camera views must be 4:3 ratio
- Minimum font: 14px annotations, 16px body, 20px titles
- Element IDs must be unique and never reused after deletion
- Diagrams are hand-drawn style — use for layout intent, not pixel precision

---

## 3. Fyne UI Testing Strategy

### Testing Tiers

The project uses four tiers of UI testing, in order of priority:

#### Tier 1: Presenter Logic Tests (Highest Value)

Pure Go tests on presenter structs. No Fyne dependency, fast, deterministic. These already exist for all presenters.

```go
func (s *NotificationSuite) TestTruncation() {
    // Test that a 30-char sender is truncated to 15
    s.Equal("JohnDoe________", row.Sender)
}
```

**When to use:** Always. Every UI behavior that can be tested as data transformation should be tested here.

#### Tier 2: Widget Behavior Tests

Test Fyne widgets headlessly using `fyne.io/fyne/v2/test`. Simulate user interactions and assert on widget state.

```go
func (s *NotificationPaneSuite) TestClickRowOpensDetail() {
    app := test.NewTempApp(s.T())
    w := test.NewTempWindow(s.T(), pane)

    test.Tap(listItem)

    s.True(detailDialog.Visible())
    s.Contains(detailLabel.Text, "Importance Score:")
}
```

**Key `fyne.io/fyne/v2/test` functions:**

| Function                      | Purpose                              |
|-------------------------------|--------------------------------------|
| `test.NewTempApp(t)`          | In-memory app, auto-cleanup         |
| `test.NewTempWindow(t, c)`    | Headless window, auto-cleanup       |
| `test.Tap(obj)`               | Simulate mouse click                |
| `test.DoubleTap(obj)`         | Simulate double-click               |
| `test.Type(obj, "text")`      | Simulate keyboard input             |
| `test.Drag(c, pos, dx, dy)`   | Simulate drag gesture               |
| `test.Scroll(c, pos, dx, dy)` | Simulate scroll                     |
| `test.WidgetRenderer(w)`      | Access internal renderer for inspection |
| `test.LaidOutObjects(o)`      | Get all recursively laid-out children |

**When to use:** For interaction-driven behavior (tap opens dialog, slider updates label, button triggers action).

#### Tier 3: Markup Golden File Tests

Compare rendered widget trees against XML golden files. Human-readable diffs, stable across platforms.

```go
func (s *LayoutSuite) TestMainWindowStructure() {
    app := test.NewTempApp(s.T())
    w := test.NewTempWindow(s.T(), mainContent)

    test.AssertRendersToMarkup(s.T(), "main_window.xml", w.Canvas())
}
```

**Golden file workflow:**

1. Write test with `test.AssertRendersToMarkup(t, "path/name.xml", canvas)`
2. First run **fails** — actual output written to `testdata/failed/path/name.xml`
3. Review output; if correct, copy to `testdata/path/name.xml` as golden master
4. Subsequent runs compare against master; failures write new output to `testdata/failed/`

**When to use:** For validating layout structure and widget composition. Good for catching regressions in the three-pane layout, notification list structure, and dialog contents.

#### Tier 4: Image Golden File Tests (Use Sparingly)

Pixel-level PNG comparison. Brittle across environments (font rendering, DPI differences).

```go
func (s *VisualSuite) TestErrorTextColor() {
    test.AssertRendersToImage(s.T(), "error_entry.png", canvas)
}
```

**When to use:** Only for specific visual states where pixel accuracy matters (e.g., error text color). Run in a controlled environment with fixed theme and scale factor.

### Testing Pyramid

```
         /\
        /  \  Tier 4: Image golden (few)
       /    \
      /──────\  Tier 3: Markup golden (some)
     /        \
    /──────────\  Tier 2: Widget behavior (many)
   /            \
  /──────────────\  Tier 1: Presenter logic (most)
```

### What NOT to Use

- **Godog/BDD:** Adds indirection without benefit for a single-developer project. Fyne's test package already provides the building blocks.
- **External visual regression tools** (Percy, Applitools, BackstopJS): All web-focused, none support native Go desktop apps.
- **Manual visual inspection as the only validation:** Always pair with at least Tier 1 + Tier 2 automated tests.

---

## 4. Design-to-Implementation Pipeline

The full workflow from design to tested code:

```
 ┌─────────────────────────────────────────────────────────┐
 │  1. DESIGN                                               │
 │     Sketch layout (Excalidraw, paper, or mental model)   │
 │     Write/update UI-SPEC.md (ASCII wireframes + tokens)  │
 │     Optionally create Excalidraw diagram for iteration    │
 └─────────────────────┬───────────────────────────────────┘
                       ▼
 ┌─────────────────────────────────────────────────────────┐
 │  2. RED — Test Designer Agent                            │
 │     Read UI-SPEC.md acceptance criteria                   │
 │     Write failing tests:                                  │
 │       - Tier 1: Presenter logic tests                     │
 │       - Tier 2: Widget behavior tests (fyne/test)         │
 │       - Tier 3: Markup golden files (if layout-critical)  │
 │     Commit: test(ui): failing tests for ...               │
 └─────────────────────┬───────────────────────────────────┘
                       ▼
 ┌─────────────────────────────────────────────────────────┐
 │  3. GREEN — Implementer Agent                            │
 │     Read failing tests as specification                   │
 │     Build Fyne widgets to pass tests                      │
 │     Commit: feat(ui): implement ... [tests pass]          │
 └─────────────────────┬───────────────────────────────────┘
                       ▼
 ┌─────────────────────────────────────────────────────────┐
 │  4. REFACTOR — Refactorer Agent                          │
 │     Improve code quality, extract helpers                 │
 │     All tests stay green                                  │
 │     Commit: refactor(ui): improve ...                     │
 └─────────────────────┬───────────────────────────────────┘
                       ▼
 ┌─────────────────────────────────────────────────────────┐
 │  5. VALIDATE                                             │
 │     just fmt && just lint && just tidy && just test       │
 │     Visual spot-check if needed                           │
 └─────────────────────────────────────────────────────────┘
```

---

## 5. References

- [UI-SPEC.md](UI-SPEC.md) — Authoritative UI specification for Cue
- [Fyne test package](https://pkg.go.dev/fyne.io/fyne/v2/test) — Headless testing API
- [Fyne testing guide](https://docs.fyne.io/started/testing/) — Official testing documentation
- [Mockdown](https://www.mockdown.design/about) — ASCII wireframe editor with AI export
- [BareMinimum](https://bareminimum.design) — AI-friendly wireframe generator
- [DESIGN.md pattern](https://designmd.ai/what-is-design-md) — Design system specification format
- [ASCII wireframe pipeline](https://www.nathanonn.com/codex-plans-with-ascii-wireframes-%E2%86%92-claude-code-builds-%E2%86%92-codex-reviews/) — 97% first-pass success rate workflow
