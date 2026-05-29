# Feature 113: Client Layer Simplification

**Phase:** Phase-9-Feature-113
**Status:** Planning
**Depends on:** Feature 107 (Fyne Client Re-wire), Feature 111 (Sidecar Supervisor), Feature 112 (UI Single-Instance Lock)
**Blocks:** Phase 9 close-out
**Packages:** `pkg/client/`, `cmd/cue/adapters/`, `internal/ui/presenter/`, `internal/ui/`

---

## Overview

Phase 9 grew the client side incrementally — one work-package at a time, one resource at a time. The result is correct and tested but structurally repetitive: every endpoint owns a hand-written client file, a hand-written adapter, and a hand-written presenter, each with its own test suite. Across the three layers the UI runs to ~9.5k LOC of production code and ~18.5k LOC of tests, and the bulk of that volume is plumbing rather than behaviour.

This feature collapses the plumbing without touching the genuinely complex parts of the UI (Fyne layout, WASM character host, countdown timer widget, focus rail, drawer/modal interactions). The goal is a smaller, easier-to-extend client surface that retains identical observable behaviour.

There are no live deployments and no external SDK consumers of `pkg/client/`. Old shapes are deleted in the same commit they're replaced. No deprecation period.

---

## Problem Statement

After WP1–WP13 of Feature 107, the client stack has three layers, each duplicated per resource:

```
pkg/client/{resource}.go               ── one file per endpoint group (12 pairs)
cmd/cue/adapters/{resource}.go         ── one file per endpoint group (8 pairs)
internal/ui/presenter/{resource}_presenter.go  ── one file per view (12 pairs)
```

Per-file LOC counts (production + test):

| Layer | Files | Prod LOC | Test LOC |
|---|---|---|---|
| `pkg/client/` | 12 resources | ~1.6k | ~3.2k |
| `cmd/cue/adapters/` | 8 resources | ~1.5k | ~1.6k |
| `internal/ui/presenter/` | 12 presenters | ~2.2k | ~4.6k |

The shapes inside each layer are near-identical:

- **`pkg/client/`** — every file constructs a request, calls `client.do()`, decodes a JSON envelope, maps HTTP errors to typed errors. The error-mapping was already centralised in WP13 (`pkg/client/errors.go`); the per-resource files now exist mostly to hold one or two CRUD methods over the shared core.
- **`cmd/cue/adapters/`** — every file wraps a `pkg/client` type behind a UI-facing interface, translating SDK DTOs into presenter inputs. Several adapters do nothing more than rename fields and pass values through.
- **`internal/ui/presenter/`** — several presenters (settings, service-settings, rules, feedback) marshal adapter output directly into a view-model with no real decision logic. The `*_test.go` suites for these presenters mostly exercise pass-through.

The cost is felt every time a new endpoint or view is added: WP1–WP13 each touched all three layers, each touch came with a full RED→GREEN→REFACTOR cycle, and the test scaffolding scaled linearly. The complexity is **not** in the UI — it's in the layered repetition between the UI and the wire.

---

## Locked Decisions

### 1. Scope: transport, adapter, and pass-through presenters only

In scope:

- `pkg/client/` per-resource files.
- `cmd/cue/adapters/` per-resource files.
- Presenters with no behaviour beyond DTO→view-model translation.

Out of scope (do not touch):

- Fyne window, layout, view-router, drawer, modal, or any widget code.
- `internal/ui/character/` (WASM host).
- `internal/ui/countdown_timer.go`, `focus_rail.go`, `timer_loop.go`.
- Presenters that hold real coordination logic: `notification_presenter`, `planner_presenter`, `timer_presenter`, `activity_presenter`, `app_presenter`, `character_presenter`. These stay as-is.

### 2. Collapse `pkg/client/` to a generic typed core

Replace the per-resource files with a small generic core plus thin endpoint definitions:

```go
// pkg/client/resource.go
type Resource[T any] struct {
    client *Client
    base   string  // e.g. "/api/v1/todo/tasks"
}

func (r *Resource[T]) Get(ctx context.Context, id string) (*T, error)
func (r *Resource[T]) List(ctx context.Context, query url.Values) ([]T, error)
func (r *Resource[T]) Create(ctx context.Context, body any) (*T, error)
func (r *Resource[T]) Update(ctx context.Context, id string, body any) (*T, error)
func (r *Resource[T]) Delete(ctx context.Context, id string) error
```

Per-resource files reduce to:

- The DTO type(s).
- A `func NewTaskClient(c *Client) *Resource[Task]` constructor that fixes the path.
- Any non-CRUD verb (e.g. `Tasks.Complete(id)`) as a free-standing method on a thin wrapper type.

The error mapping in `pkg/client/errors.go` (delivered in WP13) is the single classifier; the generic core calls it for every response. No per-resource error tables.

The activity stream (`pkg/client/activity.go`) is **not** generic — it's a WebSocket fan-out, structurally different from CRUD. It stays bespoke.

### 3. Collapse `cmd/cue/adapters/` where adapters are pure translation

Three categories of adapter exist today; treat each differently:

| Category | Examples | Disposition |
|---|---|---|
| Pure rename + pass-through | `feedback`, `messages`, `rules`, `service_config` | **Delete the adapter.** Presenters consume `pkg/client` types directly. |
| DTO reshape (one-to-many or many-to-one) | `tasks` (Category embed), `schedule` (tree shape) | **Keep.** These earn their place. |
| Fan-out / event translation | `activity` | **Keep.** WebSocket → typed channel mapping is non-trivial. |

Removing the pure-translation adapters takes ~700 LOC production + ~660 LOC tests out of `cmd/cue/adapters/`.

### 4. Collapse pass-through presenters

A presenter is "pass-through" if it satisfies all of:

1. Holds no internal state beyond cached last-fetched data.
2. Does not coordinate between two or more data sources.
3. Has no event-fan-out, no debouncing, no scheduling.

By that definition: `settings_presenter`, `service_settings_presenter`, `rules_presenter`, `feedback_presenter` are pass-through.

For each: delete the presenter and its test suite. Wire the view directly to the `pkg/client` resource (or to its surviving adapter). View-model translation that was happening in the presenter moves into a small pure function in the view file (e.g. `formatRuleForDisplay(r client.Rule) ruleRow`), tested in the view's existing `*_interaction_test.go`.

This removes ~700 LOC production + ~1.9k LOC tests from `internal/ui/presenter/`.

### 5. No behavioural change

This is a structural refactor. The full `just test && just test-ui` suite must remain green throughout. A behaviour-changing diff (new feature, bug fix found in passing) is rejected — open a separate feature for it. The only acceptable "change" is the deletion of tests whose subject (a presenter or adapter) ceases to exist; the behaviour those tests covered is re-asserted by the existing view-level interaction tests.

### 6. Test re-anchoring rule

When deleting a presenter or adapter:

1. Identify each unit test's assertion.
2. If an equivalent assertion exists at the view-interaction or client-level test, delete the unit test outright.
3. If no equivalent exists, **first** add the assertion at the surviving layer (RED), then delete the unit test in the same loop's GREEN.

Net test LOC must drop, but no behavioural assertion is lost.

### 7. Imports stay acyclic

The Go module's package dependency graph must remain acyclic per CLAUDE.md §14. Specifically:

- `internal/ui/` may import `pkg/client/` directly once adapters are removed.
- `internal/ui/presenter/` may import `pkg/client/` directly.
- `pkg/client/` imports nothing from `internal/`.

This is already the shape; the refactor preserves it.

---

## TDD Loop Plan

Per `CLAUDE.md` §13: each loop = RED (test-designer) → GREEN (implementer) → REFACTOR (refactorer), three commits, `just fmt` last before each commit. Agent teams used throughout. UI tests (`just test-ui`) are the outer gate per the UI Feature Workflow — they must stay green at the end of every loop.

| # | Loop | Scope |
|---|---|---|
| 1 | Generic `Resource[T]` core | Add `pkg/client/resource.go` with the generic CRUD core. Tests (`resource_test.go`) cover happy path + every error class via the centralised classifier. No existing files removed yet. |
| 2 | Migrate simple resources to generic core | Convert `feedback.go`, `messages.go`, `rules.go`, `service_config.go`, `categories.go`, `auth.go`, `timer.go` (CRUD parts only) to thin constructors over `Resource[T]`. Delete the per-resource error-mapping that the centralised classifier already covers. Tests retain DTO assertions; transport assertions move to `resource_test.go`. |
| 3 | Migrate complex resources | `tasks.go` (Category embed shaping), `schedule.go` (tree DTO). These keep small bespoke methods alongside `Resource[T]` for the parts that don't fit CRUD. |
| 4 | Delete pure-translation adapters | Remove `cmd/cue/adapters/{feedback,messages,rules,service_config}.go` + tests. Update `cmd/cue/main.go` and the four corresponding views to consume `pkg/client` types directly. View-interaction tests gain any assertion previously held by adapter tests. |
| 5 | Delete pass-through presenters: settings + service-settings | Remove `settings_presenter.go`, `service_settings_presenter.go` + tests. Inline view-model formatting into `settings_view.go` / the corresponding view file as pure functions. View-interaction tests cover format outputs. |
| 6 | Delete pass-through presenters: rules + feedback | Same pattern as Loop 5 for `rules_presenter.go` and `feedback_presenter.go`. |
| 7 | Surviving adapters review | Audit `cmd/cue/adapters/{tasks,schedule,activity}.go` for any code paths that became unreachable after Loops 2–6. Delete dead code. No interface changes. |
| 8 | Composition root cleanup | `cmd/cue/main.go`: remove constructor calls for deleted adapters/presenters; renumber constructor argument lists; verify the dependency graph is still acyclic. |
| 9 | Documentation + Roadmap | CHANGELOG entry under `### Changed` (this is breaking only for hypothetical external `pkg/client` consumers, which we don't have — note that explicitly). README's architecture sketch updated to drop the adapter/presenter rectangles where they no longer exist. Roadmap row → Done. Agent log updated. |

9 loops total, ~4 working days. The work is mechanical but spans many files; loop sizing prioritises a single layer per loop so that test failures localise cleanly.

---

## Wiring Verification

After Loop 9, before security checks:

1. `grep -rn "ErrNotImplemented" internal/ pkg/ cmd/` (non-test) — empty.
2. `find pkg/client -name '*.go' -not -name '*_test.go' | xargs wc -l` — total prod LOC dropped by ≥ 40%.
3. `find cmd/cue/adapters -name '*.go' -not -name '*_test.go' | xargs wc -l` — total prod LOC dropped by ≥ 40%.
4. `find internal/ui/presenter -name '*.go' -not -name '*_test.go' | xargs wc -l` — total prod LOC dropped by ≥ 25%.
5. `grep -rn "presenter\.\(Settings\|ServiceSettings\|Rules\|Feedback\)Presenter" internal/ cmd/` — empty.
6. `grep -rn "adapters\.\(Feedback\|Messages\|Rules\|ServiceConfig\)" internal/ cmd/` — empty.
7. `cmd/cue` boots against a running `cmd/cue-server`; manual smoke test of every view: notifications render, rules CRUD, feedback rate, settings save, schedule view loads, activity drawer streams, timer ticks. Identical behaviour to pre-refactor.
8. `just test && just test-ui && just security && just vulncheck` all green.
9. `just lint && just tidy` clean.

---

## Acceptance Criteria

- `pkg/client/` exposes a generic `Resource[T]` core; per-resource files are thin DTO + path declarations.
- `cmd/cue/adapters/` contains only the three adapters that earn their place: `tasks`, `schedule`, `activity`.
- Pass-through presenters (`settings`, `service_settings`, `rules`, `feedback`) are removed; their views consume `pkg/client` directly.
- Total UI-stack production LOC reduced by ≥ 25% relative to the post-WP13 baseline. Test LOC reduced commensurately, with no loss of behavioural assertions (re-anchored to view-interaction or client-level tests).
- `just test`, `just test-ui`, `just security`, `just vulncheck`, `just lint`, `just tidy` all green.
- Manual smoke test confirms identical UX across all views.

---

## Risk Areas

1. **Generics churn (Loop 1).** Go generics on a `Client` core that decodes typed responses requires care around `nil` body, empty-list decoding, and the existing error classifier. Mitigation: Loop 1 lands `Resource[T]` + tests with no migrations; Loops 2–3 only flip call sites once the core is solid.
2. **Test loss without re-anchoring.** Deleting a presenter test could silently drop a behaviour assertion. Mitigation: Decision 6's re-anchoring rule is enforced — every deleted unit test must have an equivalent assertion at a surviving layer **before** removal. Reviewer checks the diff for asymmetric deletions.
3. **Hidden coupling in adapters.** A "pass-through" adapter may turn out to hold a subtle field rename or default-value rule. Mitigation: Loop 4's RED phase reads each adapter end-to-end and writes view-interaction tests for any non-trivial translation before the adapter is deleted.
4. **Composition root regressions.** `cmd/cue/main.go` is touched in Loop 8. Mitigation: rely on `composition_test.go` (already in `internal/ui/`) and a manual smoke test before security checks.
5. **Scope creep into stable presenters.** Loops are tempted to also "tidy" `notification_presenter` or `planner_presenter`. Mitigation: Decision 1's out-of-scope list is binding. Any cleanup of those presenters is a separate feature.

---

## Estimate

- Removed: ~700 LOC adapters + ~700 LOC pass-through presenters + ~600 LOC client files = ~2.0k production LOC.
- Removed tests: ~660 LOC adapter tests + ~1.9k LOC presenter tests + replaced client tests ≈ ~2.6k test LOC.
- Added: ~150 LOC generic `Resource[T]` core + ~250 LOC of re-anchored view-interaction tests.
- Net: −1.85k production LOC, −2.35k test LOC.
- Loops: 9, ~4 working days.

---

## Knock-on Effects

- Future endpoints add: one DTO struct + one `NewXClient` line in `pkg/client/`, plus view wiring. No new presenter, no new adapter.
- The Phase 9 close-out review (Roadmap "Phase-9 Overview") gains a "Simplification" line — the final shape of the client stack is what enters Phase 10.
- Sidecar supervisor (Feature 111) and single-instance lock (Feature 112) are unaffected; they touch process management, not the UI layers refactored here.

---

## TDD Agent Stats

| Loop | Phase    | Agent           | Commit |
|------|----------|-----------------|--------|
| 1    | Red      | test-designer   | TBD    |
| 1    | Green    | implementer     | TBD    |
| 1    | Refactor | refactorer      | TBD    |
| 2    | Red      | test-designer   | TBD    |
| 2    | Green    | implementer     | TBD    |
| 2    | Refactor | refactorer      | TBD    |
| 3    | Red      | test-designer   | TBD    |
| 3    | Green    | implementer     | TBD    |
| 3    | Refactor | refactorer      | TBD    |
| 4    | Red      | test-designer   | TBD    |
| 4    | Green    | implementer     | TBD    |
| 4    | Refactor | refactorer      | TBD    |
| 5    | Red      | test-designer   | TBD    |
| 5    | Green    | implementer     | TBD    |
| 5    | Refactor | refactorer      | TBD    |
| 6    | Red      | test-designer   | TBD    |
| 6    | Green    | implementer     | TBD    |
| 6    | Refactor | refactorer      | TBD    |
| 7    | Refactor | refactorer      | TBD    |
| 8    | Refactor | refactorer      | TBD    |
| —    | Wiring   | direct          | TBD    |
