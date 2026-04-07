# Feature 075: WASM Character Plugins

**Phase:** Phase-7-Feature-075
**Type:** Feature
**Severity:** N/A (new capability)
**Status:** Planned
**Packages:** `internal/ui/character/`, `internal/config/`, `cmd/cue/`, CI/CD pipelines
**Related:** Feature 014 (Character System), Feature 024 (Character UAT Harness), Feature 041 (Character Package Restructure)

---

## Problem

The fairy character is compiled directly into the main `cue` binary. Every user gets the fairy whether they want it or not, and the binary includes ~22 Go source files of fairy-specific code, embedded PNG assets, and 6 state animator implementations. As more characters are added, binary size grows linearly with the number of characters — even though a user typically uses only one at a time.

The current architecture also requires all character code to live in-tree: third-party characters are impossible without forking the repository.

## Goal

Characters become **WASM plugins** loaded at runtime from a user-configurable directory. The main binary ships with no character code (only the `"none"` no-op fallback). Users download character `.wasm` files and drop them into `~/.cue/characters/`. The binary stays small, characters are portable across OS/arch, and third parties can create characters without forking.

## Architecture

### Overview

```
~/.cue/
  characters/
    fairy.wasm          ← compiled character plugin
    dragon.wasm         ← another character plugin
    fairy/
      jar_back.png      ← character assets (loaded by plugin via host API)
      jar_front.png
    dragon/
      sprite.png
```

```
                    ┌──────────────────────┐
                    │     cmd/cue/main.go  │
                    │                      │
                    │  character.Create()  │
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │  character registry   │
                    │                      │
                    │  "none" → NoOp       │
                    │  "fairy" → WASMHost  │
                    │  "dragon" → WASMHost │
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │  WASMCharacterHost   │
                    │  (wazero runtime)    │
                    │                      │
                    │  Loads .wasm module  │
                    │  Exposes host API    │
                    │  Bridges to Fyne     │
                    └──────────────────────┘
```

### Runtime: wazero

[wazero](https://github.com/tetratelabs/wazero) is a pure-Go WebAssembly runtime with zero dependencies. It requires no CGO — aligning with Cue's pure-Go constraint (the sole CGO exception remains audio via `gopxl/beep`).

Key properties:
- **Pure Go** — no C dependencies, no `libwasmtime`, no system WASM runtime
- **WASI support** — plugins can do basic I/O (read their own asset files) via sandboxed filesystem access
- **Sandboxed** — plugins cannot access host filesystem, network, or memory outside their sandbox
- **Cross-platform** — Linux, macOS, Windows (same `.wasm` file works everywhere)
- **Ahead-of-time compilation** — wazero can AOT compile `.wasm` to native code for near-native performance

### Host API

The host (Cue) exposes functions to the WASM guest that bridge to the Fyne rendering layer. The plugin never touches Fyne directly — it communicates via a narrow, stable ABI.

#### Host → Guest (calls into the plugin)

| Function | Signature | Purpose |
|---|---|---|
| `character_name` | `() → string` | Returns the character's display name |
| `character_init` | `(jar_width, jar_height f64)` | Called once after load with available render area |
| `character_transition` | `(state i32)` | Notify plugin of state change (uses `CharacterState` enum) |
| `character_tick` | `(elapsed_ms i64) → bool` | Called every frame (~33ms). Returns true if visuals changed. |
| `character_close` | `()` | Cleanup signal before unload |

#### Guest → Host (plugin calls back into Cue)

| Function | Signature | Purpose |
|---|---|---|
| `host_set_circle` | `(id i32, x, y, radius f64, r, g, b, a u8)` | Create or update a colored circle |
| `host_remove_circle` | `(id i32)` | Remove a circle |
| `host_set_image` | `(id i32, x, y, w, h f64, asset_path string)` | Place an image from the character's asset directory |
| `host_remove_image` | `(id i32)` | Remove an image |
| `host_log` | `(level i32, msg string)` | Emit a log message to Cue's activity log |

This keeps the ABI minimal: circles (body, glow layers) and images (jar, sprites). The host manages z-ordering by ID (lower IDs render first). All coordinates are normalized 0.0–1.0, matching the existing fairy position system.

### WASMCharacterHost

New type in `internal/ui/character/` that implements the `Character` interface by delegating to a loaded WASM module:

```go
type WASMCharacterHost struct {
    name     string
    runtime  wazero.Runtime
    module   api.Module
    widget   fyne.CanvasObject
    objects  map[int32]fyne.CanvasObject // circles + images managed by plugin
    assetDir string                      // ~/.cue/characters/<name>/
    ticker   *time.Ticker
    mu       sync.Mutex
}
```

**Lifecycle:**
1. `LoadCharacterPlugin(wasmPath, assetDir string) (*WASMCharacterHost, error)` — instantiates wazero runtime, compiles module, injects host functions, calls `character_init`
2. `TransitionTo(state)` — calls `character_transition(state)` in the guest
3. Internal tick loop calls `character_tick(elapsed)` at 30 FPS; if it returns true, refreshes the Fyne widget
4. `Close()` — calls `character_close()`, stops ticker, closes wazero runtime

### Plugin Discovery and Registration

On startup, `cmd/cue/main.go` scans `~/.cue/characters/` for `*.wasm` files. For each file, it registers a factory in the character registry:

```go
pluginDir := filepath.Join(homeDir, ".cue", "characters")
entries, _ := os.ReadDir(pluginDir)
for _, entry := range entries {
    if filepath.Ext(entry.Name()) == ".wasm" {
        name := strings.TrimSuffix(entry.Name(), ".wasm")
        wasmPath := filepath.Join(pluginDir, entry.Name())
        assetDir := filepath.Join(pluginDir, name)
        character.Register(name, func() character.Character {
            host, err := character.LoadCharacterPlugin(wasmPath, assetDir)
            if err != nil {
                log.Printf("warning: failed to load character %q: %v", name, err)
                return character.NewNoOpCharacter()
            }
            return host
        })
    }
}
```

### Config Changes

**Remove** `gui.character` from TOML and `GUIConfig` struct. Replace with:

```toml
[gui]
window_width = 1200
window_height = 800
character_plugin = "fairy"   # name of .wasm file (without extension), or "none"
character_dir = "~/.cue/characters"  # optional override, defaults to ~/.cue/characters
```

The `"none"` value skips plugin loading entirely and uses the built-in `NoOpCharacter`. If the named plugin is not found, fall back to `"none"` with a log warning (same behavior as today).

### Fairy Migration

The existing fairy implementation moves from compiled-in code to a WASM plugin:

1. **New build target**: `cmd/fairy-plugin/` — a TinyGo (or standard Go + WASI) main package that compiles to `fairy.wasm`
2. **Fairy logic stays Go** — the animators, position math, glow calculations remain in Go, compiled to WASM
3. **Rendering via host API** — instead of creating Fyne circles directly, the fairy calls `host_set_circle` and `host_set_image`
4. **Assets** — `jar_back.png` and `jar_front.png` move to `~/.cue/characters/fairy/` (or bundled in a release archive)
5. **Release archives** include `fairy.wasm` + assets alongside the `cue` binary

The fairy character package (`internal/ui/character/fairy/`) is removed from the main binary's import graph. The `character.Register("fairy", ...)` call in `main.go` is removed — fairy registers automatically via plugin discovery.

### Compilation

Character plugins can be compiled with either:

**TinyGo** (smaller output, better WASM support):
```bash
tinygo build -o fairy.wasm -target=wasi ./cmd/fairy-plugin/
```

**Standard Go** (larger output, full stdlib):
```bash
GOOS=wasip1 GOARCH=wasm go build -o fairy.wasm ./cmd/fairy-plugin/
```

TinyGo is preferred for smaller `.wasm` files (~100KB vs ~5MB). The choice is a build-time concern — the host loads any valid WASM module regardless of how it was compiled.

## CI/CD Changes

### CI Pipeline (`ci.yml`)

No matrix expansion needed for characters — WASM is architecture-independent. Add a step to the `build-test` job:

```yaml
- name: Build character plugins (WASM)
  run: |
    set -euo pipefail
    mkdir -p _build/characters/fairy
    tinygo build -o _build/characters/fairy.wasm -target=wasi ./cmd/fairy-plugin/
    cp cmd/fairy-plugin/assets/* _build/characters/fairy/
```

**Key point:** `.wasm` is cross-platform — one build produces one artifact that works on all OS/arch combinations. No matrix expansion. This is the primary advantage over the `.so` approach, which would have required OS x arch x character matrix explosion.

### Release Pipeline (`release.yml`)

Add WASM build step before goreleaser. Include `fairy.wasm` + assets in release archives:

```yaml
- name: Build character plugins
  run: |
    mkdir -p dist/characters/fairy
    tinygo build -o dist/characters/fairy.wasm -target=wasi ./cmd/fairy-plugin/
    cp cmd/fairy-plugin/assets/* dist/characters/fairy/
```

Update `.goreleaser.yml` to include the characters directory in archives:

```yaml
archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - src: dist/characters/**/*
        dst: characters/
        strip_parent: true
```

### No Matrix Explosion

The current release matrix is:

| Job | OS | Arch | Artifacts |
|---|---|---|---|
| `release-linux` | linux | amd64, arm64 | `cue`, `cue-uat` |
| `release-darwin` | darwin | arm64 | `cue`, `cue-uat` |

With WASM plugins, this stays **exactly the same**. The `.wasm` files are built once and included in every platform archive. Compare this to the `.so` approach which would have required `OS x arch x character` = `3 x 2 x N` build combinations.

### Character UAT Harness

The existing `cmd/cue-uat` (character UAT) should also load WASM plugins for visual testing. Update it to scan the same `characters/` directory.

## Security Considerations

- **Sandboxing**: wazero runs WASM modules in a sandbox. Plugins cannot access the filesystem except through explicit WASI pre-opens (limited to their own asset directory).
- **No network access**: The host API does not expose network functions. Plugins cannot make HTTP calls or open sockets.
- **Resource limits**: wazero supports memory limits per module. Set a reasonable cap (e.g., 64MB) to prevent runaway plugins.
- **No code execution**: Plugins can only draw circles and images. They cannot execute shell commands, read arbitrary files, or interact with other system components.
- **Asset path validation**: `host_set_image` must validate that the requested path is within the plugin's own asset directory (prevent path traversal).

## Dependencies

| Dependency | Version | Purpose | Pure Go |
|---|---|---|---|
| `github.com/tetratelabs/wazero` | latest | WASM runtime | Yes |

**Build-time** (for compiling character plugins):
| Tool | Purpose |
|---|---|
| `tinygo` | Compile Go character code to WASM (preferred, smaller output) |
| `go` (1.21+) | Alternative: `GOOS=wasip1 GOARCH=wasm go build` (larger output) |

## Test Strategy

### Behaviors

1. **Plugin discovery** — scanning `~/.cue/characters/` finds `.wasm` files and registers them
2. **WASMCharacterHost implements Character** — Name(), TransitionTo(), CurrentState(), Widget(), Close() all work
3. **Host API: circles** — `host_set_circle` / `host_remove_circle` create/update/remove Fyne circles
4. **Host API: images** — `host_set_image` / `host_remove_image` create/update/remove Fyne images with path validation
5. **Tick loop** — `character_tick` called at 30 FPS, widget refreshed only when guest returns true
6. **State transitions** — `character_transition` called on TransitionTo, guest receives correct state enum
7. **Fallback** — missing/broken plugin falls back to NoOpCharacter with log warning
8. **Asset sandboxing** — `host_set_image` rejects paths outside the plugin's asset directory
9. **Config** — `gui.character_plugin` parsed, `gui.character` removed, defaults to `"none"`
10. **Fairy WASM parity** — fairy.wasm produces identical visual output to the current compiled-in fairy (verified via UAT harness)

### Migration Tests

- Existing fairy unit tests should be adapted to test the fairy WASM plugin end-to-end (load fairy.wasm → transition states → verify host API calls)
- Registry tests updated: fairy no longer auto-registered at compile time, registered via plugin discovery

## Files to Create

| File | Purpose |
|---|---|
| `internal/ui/character/wasm_host.go` | `WASMCharacterHost` — loads and bridges WASM plugins |
| `internal/ui/character/wasm_host_test.go` | Tests for WASM host |
| `internal/ui/character/discovery.go` | Plugin directory scanner + auto-registration |
| `internal/ui/character/discovery_test.go` | Tests for plugin discovery |
| `cmd/fairy-plugin/main.go` | Fairy character compiled to WASM |
| `cmd/fairy-plugin/animators.go` | Fairy state animators (ported from internal/ui/character/fairy/) |
| `cmd/fairy-plugin/assets/` | `jar_back.png`, `jar_front.png` (moved from fairy/) |

## Files to Change

| File | Change |
|---|---|
| `internal/config/config.go` | Replace `Character string` with `CharacterPlugin string` + `CharacterDir string` in `GUIConfig` |
| `internal/config/config_test.go` | Update config parsing tests |
| `cmd/cue/main.go` | Remove compiled-in fairy registration, add plugin discovery call |
| `cmd/cue-uat/main.go` | Same — use plugin discovery |
| `.goreleaser.yml` | Include `characters/` directory in release archives |
| `.github/workflows/ci.yml` | Add WASM build step |
| `.github/workflows/release.yml` | Add WASM build step |
| `.gitea/workflows/ci.yml` | Add WASM build step |
| `.gitea/workflows/release.yml` | Add WASM build step |
| `justfile` | Add `just build-characters` target |
| `docs/guides/CharacterDevelopmentGuide.md` | Rewrite for WASM plugin authoring |

## Files to Remove

| File | Reason |
|---|---|
| `internal/ui/character/fairy/` (entire package) | Fairy moves to `cmd/fairy-plugin/` as a WASM plugin |

## Acceptance Criteria

- [ ] `wazero` added as dependency, no CGO required
- [ ] `WASMCharacterHost` implements `Character` interface
- [ ] Host API supports circles and images with normalized coordinates
- [ ] Plugin discovery scans `~/.cue/characters/*.wasm` and registers factories
- [ ] `gui.character_plugin` replaces `gui.character` in config.toml
- [ ] Missing plugin falls back to `"none"` with log warning
- [ ] Fairy compiles to `fairy.wasm` via TinyGo or standard Go
- [ ] `fairy.wasm` produces visually identical output to compiled-in fairy
- [ ] Asset paths validated against plugin's own directory (no path traversal)
- [ ] CI builds `.wasm` plugins as part of the pipeline
- [ ] Release archives include `characters/` directory with `fairy.wasm` + assets
- [ ] No build matrix expansion — `.wasm` files are platform-independent
- [ ] All existing tests pass (with fairy tests migrated to WASM plugin tests)
- [ ] Character Development Guide updated for WASM plugin authoring
- [ ] Main binary size reduced (no fairy code compiled in)
