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
| 011A | Injectable Fyne dependencies | Done | [Feature-011A](features/phase-1/Feature-011A-Injectable-Fyne-Dependencies.md) |
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
| 024 | Character UAT harness | Replaced by Feature 076 | [Feature-024](features/phase-3/Feature-024-Character-UAT-Harness.md) |
| 024A | Wayland thread-safety in UAT harness | Done | [Feature-024A](features/phase-3/Feature-024A-Wayland-Thread-Safety.md) |
| 024B | Fairy refresh thread safety | Done | [Feature-024B](features/phase-3/Feature-024B-Fairy-Refresh-Thread-Safety.md) |
| 024C | refreshFunc wiring (movement animations) | Done | [Feature-024C](features/phase-3/Feature-024C-RefreshFunc-Wiring.md) |
| 024D | Direct refresh removal (thread-safety fix) | Done | [Feature-024D](features/phase-3/Feature-024D-Direct-Refresh-Removal.md) |
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
| 051 | Fairy interior bounds | Done | — | [Feature-051](features/phase-5/Feature-051-Fairy-Interior-Bounds.md) |

## Phase 6 — The Big Clean (Bugfixes + Test Infrastructure)

| # | Component | Type | Severity | Status | Depends on | Doc |
|---|---|---|---|---|---|---|
| 052 | Automated UI testing framework | Enhancement | — | Done | — | [Feature-052](features/phase-6/Feature-052-Automated-UI-Testing.md) |
| 053 | Email mention detection never triggers IS=8 | Bugfix | Critical | Done | — | [Feature-053](features/phase-6/Feature-053-Email-Mention-Detection.md) |
| 054 | Audio alerts never fire on NOTIFIED | Bugfix | Critical | Done | — | [Feature-054](features/phase-6/Feature-054-Audio-Alert-Wiring.md) |
| 055 | Focus rail is placeholder label | Bugfix | High | Done | 052 | [Feature-055](features/phase-6/Feature-055-Focus-Rail-Wiring.md) |
| 056 | Plan/Wizard views are placeholder labels | Bugfix | High | Done | 052 | [Feature-056](features/phase-6/Feature-056-Plan-Wizard-Wiring.md) |
| 057 | Notification panel ignores color-coded cards | Bugfix | High | Done | 052 | [Feature-057](features/phase-6/Feature-057-Notification-Card-Rendering.md) |
| 058 | Vector score advisor not wired in main.go | Bugfix | Medium | Done | — | [Feature-058](features/phase-6/Feature-058-Vector-Advisor-Wiring.md) |
| 059 | Feedback review modal never callable | Bugfix | Medium | Done | 055 | [Feature-059](features/phase-6/Feature-059-Feedback-Review-Wiring.md) |
| 060 | Settings view tabs are all stubs | Bugfix | Medium | Done | 052 | [Feature-060](features/phase-6/Feature-060-Settings-View-Implementation.md) |
| 060A | Settings view missing exit control | Enhancement | Low | Done | 060 | [Feature-060A](features/phase-6/Feature-060A-Settings-Exit-Control.md) |
| 061 | Database insert errors silently swallowed | Bugfix | Low | Done | — | [Feature-061](features/phase-6/Feature-061-Insert-Error-Logging.md) |
| 062 | Notification list not refreshed after resolve | Bugfix | Low | Done | — | [Feature-062](features/phase-6/Feature-062-Notification-Refresh-After-Resolve.md) |
| 063 | PlannerView missing horizontal split + todo widget tree | Bugfix | High | Done | 052 | [Feature-063](features/phase-6/Feature-063-PlannerView-Widget-Tree.md) |
| 064 | WizardView missing step widget rendering | Bugfix | High | Done | 052 | [Feature-064](features/phase-6/Feature-064-WizardView-Widget-Tree.md) |
| 074 | Failing UI acceptance tests for bugs 065-073 | Enhancement | — | Done | 052 | [Feature-074](features/phase-6/Feature-074-Bugfix-UI-Acceptance-Tests.md) |
| 065 | Settings view missing Calendar tab | Bugfix | Medium | Done | 060, 074 | [Feature-065](features/phase-6/Feature-065-Calendar-Settings-Tab.md) |
| 066 | PlannerView no-plan content not rendered | Bugfix | High | Done | 063, 071 | [Feature-066](features/phase-6/Feature-066-PlannerView-Content-Rendering.md) |
| 067 | Email settings Add Account callback is noop | Bugfix | Medium | Done | 060 | [Feature-067](features/phase-6/Feature-067-Email-Add-Account.md) |
| 068 | Slack settings Add Account callback is noop | Bugfix | Medium | Done | 060 | [Feature-068](features/phase-6/Feature-068-Slack-Add-Account.md) |
| 069 | Audio settings missing Timer Volume slider | Bugfix | Medium | Done | 060 | [Feature-069](features/phase-6/Feature-069-Timer-Volume-Slider.md) |
| 070 | Activity log drawer uses split instead of overlay | Bugfix | High | Done | 019 | [Feature-070](features/phase-6/Feature-070-Activity-Log-Overlay.md) |
| 071 | Planner subsystem not wired in main.go | Bugfix | Critical | Done | — | [Feature-071](features/phase-6/Feature-071-Planner-Subsystem-Wiring.md) |
| 072 | Wizard step 3 Up/Down reorder buttons are noops | Bugfix | Medium | Done | 071 | [Feature-072](features/phase-6/Feature-072-Wizard-Reorder-Buttons.md) |
| 073 | PlannerView navigation buttons not wired | Bugfix | High | Done | 071 | [Feature-073](features/phase-6/Feature-073-PlannerView-Button-Wiring.md) |
| 070A | Activity log button fills entire center panel | Bugfix | Medium | Done | 070 | [Feature-070A](features/phase-6/Feature-070A-Activity-Log-Button-Layout.md) |
| 065A | Calendar Add Account form is noop | Bugfix | Medium | Done | 065 | [Feature-065A](features/phase-6/Feature-065A-Calendar-Add-Account-Form.md) |
| 067A | Email encryption setting missing | Bugfix | High | Done | 067 | [Feature-067A](features/phase-6/Feature-067A-Email-Encryption-Setting.md) |
| 068A | Slack user authentication (bot→user token) | Bugfix | High | Done | 068 | [Feature-068A](features/phase-6/Feature-068A-Slack-User-Authentication.md) |
| 077 | Plan > Plan my day does nothing | Bugfix | High | Done | 056, 071 | [Feature-077](features/phase-6/Feature-077-Plan-My-Day-Noop.md) |
| 078 | Added service accounts don't appear in UI | Bugfix | Critical | Done | 065, 067, 068 | [Feature-078](features/phase-6/Feature-078-Account-List-Not-Populated.md) |
| 079 | Credential/calendar validation on save | Enhancement | Medium | Done | 065, 067, 068 | [Feature-079](features/phase-6/Feature-079-Credential-Validation-On-Save.md) |
| 080 | Default poll intervals per service type | Bugfix | Low | Done | — | [Feature-080](features/phase-6/Feature-080-Default-Poll-Intervals.md) |
| 081 | Slack token setup instructions in settings | Enhancement | Low | Planned | 068 | [Feature-081](features/phase-6/Feature-081-Slack-Token-Instructions.md) |
| 082 | Graceful shutdown (SIGINT + clean exit) | Bugfix | Critical | Done | — | [Feature-082](features/phase-6/Feature-082-Graceful-Shutdown.md) |
| 083 | Fyne call thread safety violations | Bugfix | High | Planned | — | [Feature-083](features/phase-6/Feature-083-Fyne-Thread-Safety.md) |

## Phase 7 — WASM Character Plugins

| # | Component | Type | Severity | Status | Depends on | Doc |
|---|---|---|---|---|---|---|
| 075 | WASM character plugins | Feature | — | Done | 014, 041 | [Feature-075](features/phase-7/Feature-075-WASM-Character-Plugins.md) |
| 076 | Integrated character UAT mode | Refactor | — | Done | 075 | [Feature-076](features/phase-7/Feature-076-Integrated-Character-UAT.md) |
| 076A | UAT: activity log, initial char, motion | Bugfix | High | Done | 076 | [Feature-076A](features/phase-7/Feature-076A-UAT-Bugs.md) |
