# Feature 105: Settings API

**Phase:** Phase-9-Feature-105
**Status:** Planning
**Package:** `internal/server/handler/`

---

## Overview

Expose runtime settings over REST. Currently this covers audio volume controls. This is the simplest API surface but establishes the pattern for future settings categories.

## Endpoints

### Get All Settings

```
GET /api/v1/settings
```

**Response:**
```json
{
  "notification_volume": 80,
  "timer_volume": 70,
  "server_audio_enabled": false
}
```

### Update Settings

```
PATCH /api/v1/settings
```

**Request (partial update):**
```json
{
  "notification_volume": 60
}
```

Only provided fields are updated. Returns the full settings object after update.

### Get Notification Volume

```
GET /api/v1/settings/notification-volume
```

**Response:**
```json
{"value": 80}
```

### Set Notification Volume

```
PUT /api/v1/settings/notification-volume
```

**Request:**
```json
{"value": 60}
```

Validation: 0-100 integer.

### Get Timer Volume

```
GET /api/v1/settings/timer-volume
```

### Set Timer Volume

```
PUT /api/v1/settings/timer-volume
```

Same pattern as notification volume.

## Design Decisions to Make

### Settings Persistence

**Question: Where do settings changes get persisted?**

Volume is currently a runtime-only setting managed by `SettingsPresenter` calling `VolumeController.SetVolume()`. There's no persistence — volume resets on restart.

Options:
- **Runtime only**: Same as GUI. Volume resets on restart. Simple.
- **Persist to config.toml**: Write changes back to the TOML file. Survives restarts. But modifying config files at runtime is risky (formatting loss, race conditions with manual edits).
- **Persist to SQLite**: Settings table in the database. Clean separation from config file. Easy to query.

**Recommendation:** Persist to SQLite for v1. The config file is the source of *initial* settings. Runtime changes go to a settings table. On startup, DB settings override config defaults if present.

### Future Settings Categories

The settings API should be designed to accommodate future additions without breaking changes:

- Routing thresholds (importance_threshold, confidence_threshold)
- Poll intervals
- Ollama model selection
- Logging level

**Question: Should the API expose these now as read-only, or wait until they're mutable?**

**Recommendation:** Wait. Only expose settings that can actually be changed. Read-only settings that come from config.toml can be exposed in a future `GET /api/v1/config` read-only endpoint if clients need them.

### Validation

| Setting | Type | Range | Default |
|---|---|---|---|
| `notification_volume` | int | 0-100 | 80 |
| `timer_volume` | int | 0-100 | 70 |
| `server_audio_enabled` | bool | true/false | false |

## Behaviors to Implement

1. **Get all settings handler** — Aggregate current values from volume controllers + config.
2. **Update settings handler** — Partial update, validate ranges, apply to controllers.
3. **Individual get/set handlers** — Per-setting endpoints for simple clients.
4. **Settings persistence** — SQLite settings table, load on startup, save on change.

## Questions Summary

1. Persist settings to SQLite, config.toml, or runtime-only?
2. Expose read-only config values (thresholds, intervals) via the settings API?
3. Should settings changes broadcast a WebSocket event so other connected clients update?
4. Should there be a "reset to defaults" endpoint?
