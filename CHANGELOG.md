# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
with the addition of a **Breaking** section for backwards-incompatible changes
that would otherwise appear under **Changed**. Entries under **Breaking** trigger
a major version bump in automated release recommendation logic.

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security

- **Path traversal protection** — `filterSupportedAudioFiles` rejects filenames resolving outside `AudioDir`; `BeepPlayer` uses `os.OpenRoot` (Go 1.24+) for kernel-level path scoping (gosec G304, Feature-014-hotfix-A)
- **Cryptographic RNG for audio selection** — Replaced `math/rand/v2` with `crypto/rand` in audio file selection (gosec G404, Feature-014-hotfix-A)
- **Dependency bumps** — `golang.org/x/image` v0.24.0→v0.38.0 (GO-2026-4815 TIFF OOM), `golang.org/x/net` v0.35.0→v0.45.0 (GO-2026-4441, GO-2026-4440, GO-2025-3595, GO-2025-3503) (Feature-014-hotfix-A)

### Breaking

- **Alert service API change** — `NewAlertService` now takes 4 args (cfg, beeper, filesystem, player); `PlayStartup` and `PlayShutdown` removed (Phase-1-Feature-12)
- **AppPresenter API change** — `NewAppPresenter` now takes 3 args (removed alerter parameter); presenter `Alerter` interface removed (Phase-1-Feature-12)
- **NewMainWindow API change** — Now accepts `SettingsPresenter` as 6th argument and `characterWidget fyne.CanvasObject` as 7th argument (Phase-1-Feature-12, Phase-3-Feature-14)

### Added

- **Character animation system** — Pluggable character abstraction with state machine (Idle/Starting/Working/Notifying/Error/ShuttingDown), registry pattern, NoOp and Fairy implementations, CharacterPresenter consuming activity events with auto-decay, configurable via `gui.character` in config.toml (Phase-3-Feature-14)
- **gopxl/beep audio player** — Real AudioPlayer implementation using gopxl/beep/v2 for MP3/WAV/OGG playback with lazy speaker init, automatic resampling, and logarithmic volume mapping (Phase-1-Feature-13)
- **Configurable audio alerts** — Random file playback from user-configured directory (MP3/WAV/OGG), async playback, beeep fallback when no files available, configurable cooldown and fallback tone, runtime volume control via settings panel (Phase-1-Feature-12)
- **Audio config fields** — `audio_dir`, `audio_cooldown_seconds`, `audio_volume`, `fallback_frequency`, `fallback_duration_ms` in `[notification]` section with validation and tilde expansion (Phase-1-Feature-12)
- **Settings panel** — Standalone Fyne settings window with volume slider (0-100), accessible from menu bar (Phase-1-Feature-12)
- **SettingsPresenter** — Runtime volume control with VolumeController interface and 0-100 clamping (Phase-1-Feature-12)
- **Config loading and validation** — TOML-based configuration at `~/.cue/config.toml` with safe defaults, auto-creation on first run, tilde expansion, and table-driven validation (Phase-1-Feature-1)
- **SQLite message repository** — Pure Go SQLite storage (`modernc.org/sqlite`) with WAL mode, FIFO eviction (100 messages per source), upsert by MessageID, and full CRUD operations (Phase-1-Feature-2)
- **Deterministic routing rules** — Decision engine router with channel_join (IS=9) and @mention (IS=8) deterministic rules, Scorer interface for LLM evaluation, configurable threshold-based routing (NOTIFIED/BUFFERED/IGNORED), and safe fallback on scorer failure (Phase-1-Feature-3)
- **Ollama client scoring** — HTTP client implementing the Scorer interface for local Ollama LLM inference, with JSON prompt construction, markdown code block extraction, configurable timeout, and graceful error handling (Phase-1-Feature-4)
- **Slack watcher polling** — Polls Slack channels for new messages, detects channel joins (IS=9), includes thread context for replies, tracks per-channel timestamps to avoid reprocessing (Phase-1-Feature-5)
- **Email watcher polling** — Polls IMAP inbox for new messages, extracts sender/subject/folder/body, detects @mentions in To/CC/BCC (case-insensitive), tracks last-seen UID to avoid reprocessing (Phase-1-Feature-6)
- **Router orchestration** — Coordinates watchers, router, and repository in batch polling loops with per-source goroutines, immediate first poll, configurable intervals, activity event emission, graceful error handling (individual store errors don't abort batch), and idempotent shutdown (Phase-1-Feature-7)
- **Vector store with cosine similarity** — In-memory vector storage with pluggable embedding function, cosine similarity search (topN), message ID association for feedback linking, and zero-denominator handling (Phase-1-Feature-8)
- **Feedback buffer service** — Review workflow for buffered messages (IS >= 7, CS < 0.8) with oldest-first retrieval, user rating (0-10) with optional notes, message deletion, optional vector embedding on save for learning loop, and graceful embedding failure handling (Phase-1-Feature-9)
- **Audio alerts** — Cross-platform audio notifications using `beeep` with sharp ping on NOTIFIED messages (1000 Hz), startup chime (600 Hz), shutdown tone (400 Hz), 2-second cooldown to prevent spam, configurable on/off via config, and non-fatal error handling (Phase-1-Feature-10)
- **Fyne GUI** — Desktop GUI with presenter/view architecture: notification queue (NOTIFIED messages newest-first with 15-char truncation), real-time activity log (ring buffer, error highlighting), feedback buffer review (0-10 rating buttons, skip, delete, counter), app lifecycle management (startup/shutdown alerts), and `cmd/cue/main.go` composition root wiring all Phase 1 components (Phase-1-Feature-11)
- `MessageType` field on `Message` struct for distinguishing message event types
- Agent team TDD workflow with test-designer, implementer, and refactorer agents
- Agent log tracking duration and token usage per TDD phase
- Validation pipelines

### Changed

- **GUIConfig** — Replaced web-oriented `Host`/`Port` fields with Fyne-relevant `WindowWidth`/`WindowHeight` (defaults: 1200x800) (Phase-1-Feature-11)

### Removed

- **Startup and shutdown sounds** — `PlayStartup` and `PlayShutdown` removed from alert service and presenter Alerter interface (Phase-1-Feature-12)

### Fixed
