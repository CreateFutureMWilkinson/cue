# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
with the addition of a **Breaking** section for backwards-incompatible changes
that would otherwise appear under **Changed**. Entries under **Breaking** trigger
a major version bump in automated release recommendation logic.

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security

- **Path traversal protection** — `filterSupportedAudioFiles` rejects filenames resolving outside `AudioDir`; `BeepPlayer` uses `os.OpenRoot` (Go 1.24+) for kernel-level path scoping (gosec G304, Phase-3-Feature-014-Hotfix-A)
- **Cryptographic RNG for audio selection** — Replaced `math/rand/v2` with `crypto/rand` in audio file selection (gosec G404, Phase-3-Feature-014-Hotfix-A)
- **Dependency bumps** — `golang.org/x/image` v0.24.0→v0.38.0 (GO-2026-4815 TIFF OOM), `golang.org/x/net` v0.35.0→v0.45.0 (GO-2026-4441, GO-2026-4440, GO-2025-3595, GO-2025-3503) (Phase-3-Feature-014-Hotfix-A)
- **Explicit db.Close() error handling** — All SQLite repository constructors now use `_ = db.Close()` instead of bare `db.Close()` in error paths, resolving 6 gosec G104 findings (Phase-2-Feature-015-Hotfix-A)
- **Safe integer conversion in color capture** — Added `& 0xFF` bitmask to `uint32→uint8` color channel conversion in `ShutdownAnimator.captureFairyState`, resolving 4 gosec G115 (CWE-190) integer overflow findings (Phase-3-Feature-030-Hotfix-A)

### Breaking

- **Alert service API change** — `NewAlertService` now takes 4 args (cfg, beeper, filesystem, player); `PlayStartup` and `PlayShutdown` removed (Phase-1-Feature-012)
- **AppPresenter API change** — `NewAppPresenter` now takes 3 args (removed alerter parameter); presenter `Alerter` interface removed (Phase-1-Feature-012)
- **NewMainWindow API change** — Now accepts `*CenterViewRouter` as 8th argument; layout changed from two-pane HSplit to three-column (10%/60%/30%); "Review Buffered" button removed from bottom border (Phase-1-Feature-016, Phase-1-Feature-012, Phase-3-Feature-014)

### Fixed

- **First-run database crash** — `config.Load()` now calls `expandPaths()` on the default config, expanding `~/` to the user's home directory before SQLite opens the database. Previously, first-run users hit error 14 (SQLITE_CANTOPEN) because the literal `~/.cue/messages.db` path was passed to SQLite (Phase-1-Feature-001-Hotfix-A)
- **Wayland thread-safety in UAT harness** — Extracted `FPSLoop` type with injectable callback; FPS label updates now go through `fyne.Do()` instead of direct `SetText` from a background goroutine. Fixes `Error in Fyne call thread` on Wayland (Phase-3-Feature-024-Hotfix-A)
- **Character animations not playing** — `FairyCharacter.TransitionTo()` now starts/stops the appropriate state animator (idle, startup, working, notify, error, shutdown) instead of only updating a hidden indicator circle. `SetPosition()` and `SetGlowIntensity()` now trigger container refresh and glow layer alpha updates. Added production `WallClock` implementation of the `Clock` interface, `Close()` method on the `Character` interface for clean goroutine shutdown. 13 new tests (Phase-3-Feature-025-Hotfix-A)

### Added

- **Fairy lifecycle states** — Startup and shutdown one-shot animations for the fairy character. Startup: fairy wakes from dormant (#004900, glow 0.1) to idle (#006100, glow 0.5) over 1.5s with Hermite smoothstep easing. Shutdown: captures current state and interpolates to dormant (#004900, glow 0.15) over 1.5s with done-channel completion signaling for graceful app close. `EaseInOut` function, `StartupAnimator` with onComplete callback, `ShutdownAnimator` with `Done()` channel. 29 tests across 3 suites (Phase-3-Feature-030)
- **Fairy error state** — Centered vibration animation for actionable errors: horizontal oscillation at 15 Hz with ±4% amplitude around jar center (0.5, 0.5), rapid 2 Hz glow pulse (0.4–0.9 intensity), near-notification body color `#00B800`. Immediate snap-to-center entry with no transition. `ErrorAnimator` with `ErrorPosition`/`ErrorGlowIntensity` functions using shared `glowIntensity` helper. 16 tests (Phase-3-Feature-029)
- **Fairy notification state** — Erratic darting animation for the fairy character: random snap-to-position every 0.5 seconds within jar bounds, brightest green body color (#00C300), accelerated 1.5-second breathing glow cycle with elevated minimum (0.5–0.9), immediate entry transition with no interpolation. Shared `glowIntensity` helper extracted, frame interval constants consolidated. Tests cover dart timing, position bounds, color, glow cycle, and entry behavior (Phase-3-Feature-028)
- **Fairy working state** — Pseudo-random drift animation for the fairy character: layered sinusoidal movement (primary 4s circuit + secondary/tertiary noise at incommensurate frequencies) producing organic, never-repeating paths within jar bounds. 0.5-second entry transition interpolating position from idle floor (0.5, 1.0) to drift and color from idle `#006100` to working `#009200`. Breathing glow cycle maintained at same 3-second rate as idle. Shared idle constants extracted (`IdleOriginX`/`IdleOriginY`/`IdleBodyColor`), `lerpColor` utility consolidated. 11 tests (Phase-3-Feature-027)
- **Fairy idle state** — Breathing glow animation for the fairy character: sinusoidal glow oscillation between 0.3–0.8 intensity over a 3-second cycle at 30 FPS. `StateAnimator` interface, `Clock`/`Ticker` abstractions for testable animation, context-based goroutine lifecycle with synchronous stop. Position fixed at bottom-center (0.5, 1.0), body color `#006100`. 17 tests (Phase-3-Feature-026)
- **Jar rendering** — Three-layer fairy-in-a-jar composition: jar_back SVG (background), fairy body/glow circles (middle), jar_front SVG (foreground). Body circle sized at 10% of jar width, glow at 25% with 8 concentric translucent layers. Normalized position (0.0–1.0) with clamping, glow intensity control, custom Fyne layout for proportional resizing. Initial state: dark green `#006100` at bottom-center (0.5, 1.0). 14 tests (Phase-3-Feature-025)
- **Character UAT harness** — Standalone binary (`just uat` / `just run-uat`) for visually testing character animations. 800x600 Fyne window with 60/40 split: character display area and controls/diagnostics panel. Character dropdown discovers registered characters via registry, 6 state trigger buttons (Idle/Starting/Working/Notifying/Error/Shutdown), live FPS counter, and diagnostic labels. Thread-safe `FPSCounter` with tick-based measurement. Self-contained — no dependency on orchestrator, repository, or config. 11 tests across 2 suites (Phase-3-Feature-024)
- **Planner audio alerts** — `TimerAlertService` plays a configurable sound file (or distinct 440Hz/800ms fallback beep) when timer blocks complete. Meeting-aware suppression returns `MissedAlert` for notification queue routing instead of playing sound. Independent volume control (0–100, separate from notification volume). Config adds `timer_sound` and `timer_volume` to `[planner]` section. 18 tests (Phase-2-Feature-023)
- **Planner UI** — Wizard-style day planning pane with 5-step flow: task selection from todo repository, Ollama-powered Pomodoro estimation with user override and 1-pomo fallback, drag-to-reorder priority, dual schedule preview (focus-maximized vs recovery-balanced), and active schedule execution view. `PlannerPresenter` manages wizard state machine with graceful calendar/estimation failure handling. `TimerPresenter` drives 45-segment countdown ring with 1Hz flash toggle, segment progression (0–44), block-complete alerts (suppressed for meetings), and tick-driven callbacks. `PlannerView` Fyne component with step-based button visibility (Plan/Next/Back/Complete/Abandon). Interfaces `PlannerViewModel`/`TimerViewModel` for view–presenter decoupling. 64 tests across 3 suites (Phase-2-Feature-022)
- **Day planner** — Scheduling engine generating two candidate Pomodoro-style day plans: focus-maximized (minimal breaks, max contiguous focus) and recovery-balanced (post-meeting breaks, standard cycles). Meeting merging for <5min gaps, post-meeting recovery breaks (5min for ≤30min meetings, 20min for >30min), lunch-adjacent break placement, task assignment by priority with overflow detection. `PlannerConfig` in `[planner]` TOML section with workday hours, break timing, and planning cutoff. `OllamaTaskEstimator` for Pomodoro count inference with 1-pomo fallback. `ScheduleRepository` with SQLite persistence (save/load/delete, date-based overwrite). Planning horizon auto-switches to next working day after cutoff. 46 tests across 4 suites (Phase-2-Feature-021)
- **Calendar adapter** — Provider-agnostic calendar integration with ICS-over-HTTP adapter for Google Calendar secret links. `CalendarProvider` interface with `FetchEvents(ctx, date)` returning `[]CalendarEvent`. ICS parsing via `arran4/golang-ical`, date filtering, all-day event detection (`VALUE=DATE`), multi-day event spanning. Constructor validates URL, HTTP client, and timeout. 17 tests (Phase-2-Feature-020)
- **Activity log drawer** — Pull-up drawer at bottom of character area, hidden by default with toggle button. Opens to ~40% height via VSplit, showing real-time system event feed with `[HH:MM:SS] Source: Message` formatting. Error entries in red, normal in white. Integrated into MainWindow center pane via `ContainerWithCharacter`. 7 tests (Phase-1-Feature-019)
- **Notification panel redesign** — Color-coded notification cards with three importance tiers: red (IS>=9, `#ffc9c9`), orange (IS>=8, `#ffd8a8`), blue (IS<8, `#dbe4ff`). `NotificationCard` view model with linear opacity scaling (0.2–0.4), relative timestamps, and message previews. `NotificationPanel` widget with expand/collapse state, dismiss support, and detail dialog modal. `NotificationPresenter` extended with `IsExpanded`/`ToggleExpanded`/`SetOnExpandedChange`/`DismissMessage`. 25 tests across 3 suites (Phase-1-Feature-018)
- **Focus rail + countdown timer** — Persistent left column with navigation buttons (Plan/Back/Done/Review) driven by CenterViewRouter state, task label, and custom 45-segment countdown timer ring widget. Timer uses 8-degree segment intervals with cardinal (36px), diagonal (24px), and regular (12px) line lengths, yellow #FFCE1B future color with dimmed alpha=64 elapsed state. Plan-dependent widgets hidden until active plan exists. 33 tests (14 timer, 19 rail) (Phase-1-Feature-017)
- **Three-column layout** — Restructured Fyne GUI from two-pane HSplit to three-column layout: focus rail (10%), character/center area (60%), notification panel (30%). `CenterViewRouter` state machine controls center column content with `ViewCharacter`/`ViewPlan`/`ViewWizard` states and view-change callbacks. Headless test support via replaceable Fyne app factory (Phase-1-Feature-016)
- **Todo list** — CRUD operations for todos with user-defined categories, many-to-many category associations via junction table, priority ordering, optional due dates, markdown descriptions, completion tracking. TodoRepository and CategoryRepository interfaces with SQLite implementations using WAL mode and cascade deletes (Phase-2-Feature-015)
- **Character animation system** — Pluggable character abstraction with state machine (Idle/Starting/Working/Notifying/Error/ShuttingDown), registry pattern, NoOp and Fairy implementations, CharacterPresenter consuming activity events with auto-decay, configurable via `gui.character` in config.toml (Phase-3-Feature-014)
- **gopxl/beep audio player** — Real AudioPlayer implementation using gopxl/beep/v2 for MP3/WAV/OGG playback with lazy speaker init, automatic resampling, and logarithmic volume mapping (Phase-1-Feature-013)
- **Configurable audio alerts** — Random file playback from user-configured directory (MP3/WAV/OGG), async playback, beeep fallback when no files available, configurable cooldown and fallback tone, runtime volume control via settings panel (Phase-1-Feature-012)
- **Audio config fields** — `audio_dir`, `audio_cooldown_seconds`, `audio_volume`, `fallback_frequency`, `fallback_duration_ms` in `[notification]` section with validation and tilde expansion (Phase-1-Feature-012)
- **Settings panel** — Standalone Fyne settings window with volume slider (0-100), accessible from menu bar (Phase-1-Feature-012)
- **SettingsPresenter** — Runtime volume control with VolumeController interface and 0-100 clamping (Phase-1-Feature-012)
- **Config loading and validation** — TOML-based configuration at `~/.cue/config.toml` with safe defaults, auto-creation on first run, tilde expansion, and table-driven validation (Phase-1-Feature-001)
- **SQLite message repository** — Pure Go SQLite storage (`modernc.org/sqlite`) with WAL mode, FIFO eviction (100 messages per source), upsert by MessageID, and full CRUD operations (Phase-1-Feature-002)
- **Deterministic routing rules** — Decision engine router with channel_join (IS=9) and @mention (IS=8) deterministic rules, Scorer interface for LLM evaluation, configurable threshold-based routing (NOTIFIED/BUFFERED/IGNORED), and safe fallback on scorer failure (Phase-1-Feature-003)
- **Ollama client scoring** — HTTP client implementing the Scorer interface for local Ollama LLM inference, with JSON prompt construction, markdown code block extraction, configurable timeout, and graceful error handling (Phase-1-Feature-004)
- **Slack watcher polling** — Polls Slack channels for new messages, detects channel joins (IS=9), includes thread context for replies, tracks per-channel timestamps to avoid reprocessing (Phase-1-Feature-005)
- **Email watcher polling** — Polls IMAP inbox for new messages, extracts sender/subject/folder/body, detects @mentions in To/CC/BCC (case-insensitive), tracks last-seen UID to avoid reprocessing (Phase-1-Feature-006)
- **Router orchestration** — Coordinates watchers, router, and repository in batch polling loops with per-source goroutines, immediate first poll, configurable intervals, activity event emission, graceful error handling (individual store errors don't abort batch), and idempotent shutdown (Phase-1-Feature-007)
- **Vector store with cosine similarity** — In-memory vector storage with pluggable embedding function, cosine similarity search (topN), message ID association for feedback linking, and zero-denominator handling (Phase-1-Feature-008)
- **Feedback buffer service** — Review workflow for buffered messages (IS >= 7, CS < 0.8) with oldest-first retrieval, user rating (0-10) with optional notes, message deletion, optional vector embedding on save for learning loop, and graceful embedding failure handling (Phase-1-Feature-009)
- **Audio alerts** — Cross-platform audio notifications using `beeep` with sharp ping on NOTIFIED messages (1000 Hz), startup chime (600 Hz), shutdown tone (400 Hz), 2-second cooldown to prevent spam, configurable on/off via config, and non-fatal error handling (Phase-1-Feature-010)
- **Fyne GUI** — Desktop GUI with presenter/view architecture: notification queue (NOTIFIED messages newest-first with 15-char truncation), real-time activity log (ring buffer, error highlighting), feedback buffer review (0-10 rating buttons, skip, delete, counter), app lifecycle management (startup/shutdown alerts), and `cmd/cue/main.go` composition root wiring all Phase 1 components (Phase-1-Feature-011)
- `MessageType` field on `Message` struct for distinguishing message event types
- Agent team TDD workflow with test-designer, implementer, and refactorer agents
- Agent log tracking duration and token usage per TDD phase
- Validation pipelines

### Changed

- **GUIConfig** — Replaced web-oriented `Host`/`Port` fields with Fyne-relevant `WindowWidth`/`WindowHeight` (defaults: 1200x800) (Phase-1-Feature-011)

### Removed

- **Startup and shutdown sounds** — `PlayStartup` and `PlayShutdown` removed from alert service and presenter Alerter interface (Phase-1-Feature-012)

### Fixed
