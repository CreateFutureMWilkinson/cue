# Roadmap

Implementation status for all phases and features. Feature docs live in `docs/features/phase-N/`.

## Phase 1 — Smart Routing + Feedback Buffer + UI

| # | Component | Status | Doc |
|---|---|---|---|
| 001 | Config loading + validation | Done | [Feature-001](features/phase-1/Feature-001-Config.md) |
| 001A | Tilde expansion in config paths | Done | [Feature-001A](features/phase-1/Feature-001A-Tilde-Expansion.md) |
| 002 | Message data model (SQLite) | Done | [Feature-002](features/phase-1/Feature-002-SQLite-Repository.md) |
| 003 | Deterministic routing rules | Done | [Feature-003](features/phase-1/Feature-003-Deterministic-Routing.md) |
| 004 | Ollama client + scoring | Done | [Feature-004](features/phase-1/Feature-004-Ollama-Client.md) |
| 005 | Slack watcher | Done | [Feature-005](features/phase-1/Feature-005-Slack-Watcher.md) |
| 006 | Email watcher | Done | [Feature-006](features/phase-1/Feature-006-Email-Watcher.md) |
| 007 | Router orchestration | Done | [Feature-007](features/phase-1/Feature-007-Orchestrator.md) |
| 008 | Vector integration (chromem-go) | Done | [Feature-008](features/phase-1/Feature-008-Vector-Store.md) |
| 009 | Feedback buffer | Done | [Feature-009](features/phase-1/Feature-009-Feedback-Buffer.md) |
| 010 | Audio alerts | Done | [Feature-010](features/phase-1/Feature-010-Audio-Alerts.md) |
| 011 | Fyne GUI | Done | [Feature-011](features/phase-1/Feature-011-Fyne-GUI.md) |
| 012 | Configurable audio alerts (amendment) | Done | [Feature-012](features/phase-1/Feature-012-Configurable-Audio-Alerts.md) |
| 013 | gopxl/beep audio player (amendment) | Done | [Feature-013](features/phase-1/Feature-013-Beep-Player.md) |
| 016 | Three-column layout + center view router | Done | [Feature-016](features/phase-1/Feature-016-Three-Column-Layout.md) |
| 017 | Focus rail (timer ring shell, navigation) | Done | [Feature-017](features/phase-1/Feature-017-Focus-Rail.md) |
| 017A | Countdown timer renderer (45-segment ring) | Done | [Feature-017A](features/phase-1/Feature-017A-Timer-Renderer.md) |
| 018 | Notification panel redesign (color-coded cards) | Done | [Feature-018](features/phase-1/Feature-018-Notification-Panel-Redesign.md) |
| 018A | Notification card visual rendering | Done | [Feature-018A](features/phase-1/Feature-018A-Notification-Card-Rendering.md) |
| 019 | Activity log drawer | Done | [Feature-019](features/phase-1/Feature-019-Activity-Log-Drawer.md) |

## Phase 2 — Day Planner + Timer

| # | Component | Status | Doc |
|---|---|---|---|
| 015 | Todo list (CRUD, categories, SQLite) | Done | [Feature-015](features/phase-2/Feature-015-Todo-List.md) |
| 015A | Gosec G104 unhandled db.Close() | Done | [Feature-015A](features/phase-2/Feature-015A-Gosec-Unhandled-Close.md) |
| 020 | Calendar adapter (ICS-over-HTTP) | Done | [Feature-020](features/phase-2/Feature-020-Calendar-Adapter.md) |
| 021 | Day planner (scheduling engine, Pomodoro) | Done | [Feature-021](features/phase-2/Feature-021-Day-Planner.md) |
| 022 | Planner UI (wizard pane, countdown timer) | Done | [Feature-022](features/phase-2/Feature-022-Planner-UI.md) |
| 022A | Center view router wiring | Done | [Feature-022A](features/phase-2/Feature-022A-Planner-UI-Views.md) |
| 022B | Plan view (schedule tree + no-plan state) | Done | [Feature-022B](features/phase-2/Feature-022B-Plan-View.md) |
| 022C | Todo list view + task detail modal | Done | [Feature-022C](features/phase-2/Feature-022C-Todo-List-View.md) |
| 022D | Day planner wizard steps 1-4 | Done | [Feature-022D](features/phase-2/Feature-022D-Wizard-Steps.md) |
| 022E | Timer tick loop + presenter/view binding | Done | [Feature-022E](features/phase-2/Feature-022E-Timer-Wiring.md) |
| 022F | Gosec G104/G404 security fixes | Done | [Feature-022F](features/phase-2/Feature-022F-Gosec-Security-Fixes.md) |
| 023 | Planner audio alerts (timer sounds, volume) | Done | [Feature-023](features/phase-2/Feature-023-Planner-Audio-Alerts.md) |

## Phase 3 — Animations

| # | Component | Status | Doc |
|---|---|---|---|
| 014 | Character animation system | Done | [Feature-014](features/phase-3/Feature-014-Character-System.md) |
| 014A | Security hardening (gosec, CVEs) | Done | [Feature-014A](features/phase-3/Feature-014A-Security-Hardening.md) |
| 014B | Fairy animator integration | Done | [Feature-014B](features/phase-3/Feature-014B-Animator-Integration.md) |
| 024 | Character UAT harness | Done | [Feature-024](features/phase-3/Feature-024-Character-UAT-Harness.md) |
| 024A | Wayland thread-safety in UAT harness | Done | [Feature-024A](features/phase-3/Feature-024A-Wayland-Thread-Safety.md) |
| 024B | Fairy refresh thread safety | Done | [Feature-024B](features/phase-3/Feature-024B-Fairy-Refresh-Thread-Safety.md) |
| 024C | refreshFunc wiring (movement animations) | Done | [Feature-024C](features/phase-3/Feature-024C-RefreshFunc-Wiring.md) |
| 025 | Jar rendering (SVG layers, fairy body/glow) | Done | [Feature-025](features/phase-3/Feature-025-Jar-Rendering.md) |
| 025A | Animator wiring in FairyCharacter | Done | [Feature-025A](features/phase-3/Feature-025A-Animator-Wiring.md) |
| 026 | Fairy idle state (breathing glow) | Done | [Feature-026](features/phase-3/Feature-026-Fairy-Idle-State.md) |
| 027 | Fairy working state (pseudo-random drift) | Done | [Feature-027](features/phase-3/Feature-027-Fairy-Working-State.md) |
| 028 | Fairy notification state (erratic dart) | Done | [Feature-028](features/phase-3/Feature-028-Fairy-Notification-State.md) |
| 029 | Fairy error state (centered vibrate) | Done | [Feature-029](features/phase-3/Feature-029-Fairy-Error-State.md) |
| 030 | Fairy lifecycle states (startup/shutdown) | Done | [Feature-030](features/phase-3/Feature-030-Fairy-Lifecycle-States.md) |
| 030B | Shutdown animator deadlock fix | Done | [Feature-030B](features/phase-3/Feature-030B-Shutdown-Deadlock-Fix.md) |

## Phase 4 — Dynamic Service Config + Settings UI

| # | Component | Status | Depends on | Doc |
|---|---|---|---|---|
| 031 | ServiceConfig repository interface | Done | — | [Feature-031](features/phase-4/Feature-031-ServiceConfig-Repository.md) |
| 031A | Encrypted credential storage | Done | 031, 032 | [Feature-031A](features/phase-4/Feature-031A-Encrypted-Credential-Storage.md) |
| 031B | Key file path traversal fix (G304) | Done | 031A | [Feature-031B](features/phase-4/Feature-031B-Key-File-Path-Traversal.md) |
| 032 | SQLite ServiceConfig implementation | Done | 031 | [Feature-032](features/phase-4/Feature-032-SQLite-ServiceConfig.md) |
| 033 | Watcher config decoupling | Done | — | [Feature-033](features/phase-4/Feature-033-Watcher-Config-Decoupling.md) |
| 034 | Dynamic watcher management | Done | — | [Feature-034](features/phase-4/Feature-034-Dynamic-Watcher-Management.md) |
| 035 | TOML config slimming | Done | 033 | [Feature-035](features/phase-4/Feature-035-TOML-Config-Slimming.md) |
| 036 | Settings presenter expansion | Done | 031, 032, 034 | [Feature-036](features/phase-4/Feature-036-Settings-Presenter.md) |
| 037 | Settings UI expansion | Done | 036 | [Feature-037](features/phase-4/Feature-037-Settings-UI.md) |
| 038 | Main wiring update | Done | 031–037 | [Feature-038](features/phase-4/Feature-038-Main-Wiring.md) |
| 039 | Ollama model validation on startup | Done | — | [Feature-039](features/phase-4/Feature-039-Ollama-Model-Validation.md) |
| 040 | Example config generation CLI | Done | 035 | [Feature-040](features/phase-4/Feature-040-Example-Config-Generation.md) |
| 041 | Character package restructure | Done | — | [Feature-041](features/phase-4/Feature-041-Character-Package-Restructure.md) |

## Phase 5 — Wiring (Scaffold Completion + Real API Clients)

| # | Component | Status | Depends on | Doc |
|---|---|---|---|---|
| 042 | Vector-assisted routing | Done | 043, 044 | [Feature-042](features/phase-5/Feature-042-Vector-Assisted-Routing.md) |
| 043 | chromem-go vector database | Done | 044 | [Feature-043](features/phase-5/Feature-043-Chromem-Go-Vector-Database.md) |
| 044 | Ollama scorer wiring | Done | 039 | [Feature-044](features/phase-5/Feature-044-Ollama-Scorer-Wiring.md) |
| 045 | Slack API client | Done | 038 | [Feature-045](features/phase-5/Feature-045-Slack-API-Client.md) |
| 046 | IMAP email client | Done | 038 | [Feature-046](features/phase-5/Feature-046-IMAP-Email-Client.md) |
| 047 | MessageType SQLite persistence | Done | — | [Feature-047](features/phase-5/Feature-047-MessageType-Persistence.md) |
| 048 | Unused config field wiring | Done | — | [Feature-048](features/phase-5/Feature-048-Config-Field-Wiring.md) |
| 049 | MessageRepository QueryByID | Done | — | [Feature-049](features/phase-5/Feature-049-MessageRepository-QueryByID.md) |
| 050 | Resize refresh | Done | — | [Feature-050](features/phase-5/Feature-050-Resize-Refresh.md) |
