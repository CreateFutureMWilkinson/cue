# Building Cue

Cue targets the following platforms:

| Platform | Architecture | Display Server |
|---|---|---|
| Linux | amd64 | X11, Wayland |
| Linux | arm64 | X11, Wayland |
| macOS | arm64 (Apple Silicon) | Cocoa (native) |

## Prerequisites

### All Platforms

- **Go 1.26+** with `CGO_ENABLED=1`
- **just** task runner (`cargo install just` or via package manager)

### Linux (Debian/Ubuntu)

```bash
sudo apt-get install -y --no-install-recommends \
  libasound2-dev \
  libgl-dev \
  libxcursor-dev \
  libxrandr-dev \
  libxinerama-dev \
  libxi-dev \
  libxxf86vm-dev \
  libwayland-dev \
  libxkbcommon-dev \
  wayland-protocols \
  pkg-config
```

These provide:
- **ALSA** (`libasound2-dev`) — audio playback via ebitengine/oto
- **OpenGL** (`libgl-dev`) — Fyne GUI rendering
- **X11** (`libxcursor-dev`, `libxrandr-dev`, `libxinerama-dev`, `libxi-dev`, `libxxf86vm-dev`) — X11 display backend
- **Wayland** (`libwayland-dev`, `libxkbcommon-dev`, `wayland-protocols`) — Wayland display backend

On Arch Linux:
```bash
sudo pacman -S alsa-lib mesa libxcursor libxrandr libxinerama libxi libxxf86vm wayland wayland-protocols libxkbcommon pkgconf
```

On Fedora:
```bash
sudo dnf install alsa-lib-devel mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel wayland-devel libxkbcommon-devel wayland-protocols-devel pkgconf-pkg-config
```

### macOS (Apple Silicon)

```bash
xcode-select --install
```

Xcode Command Line Tools provide the C compiler and all required frameworks (CoreAudio, Cocoa, OpenGL).

## Building

```bash
# Build the main application
just build

# Build the character UAT harness
just uat

# Build both
just build-all

# Show required system packages
just deps
```

Binaries are placed in `_build/`.

## Display Server Selection (Linux)

Fyne uses GLFW, which auto-detects the display server at runtime:

- If `WAYLAND_DISPLAY` is set → uses Wayland backend
- Otherwise → uses X11 backend

Both backends are compiled into the same binary when the Wayland dev headers are present at build time. No build tags or environment variables are needed.

To force a specific backend:
```bash
# Force Wayland
FYNE_DRIVER=wayland ./cue

# Force X11 (e.g., under XWayland)
DISPLAY=:0 WAYLAND_DISPLAY= ./cue
```

## Cross-Compilation (Linux arm64 from amd64)

Building for linux/arm64 from an amd64 host requires a cross-compiler and arm64 dev libraries:

```bash
sudo dpkg --add-architecture arm64
sudo apt-get update
sudo apt-get install -y \
  gcc-aarch64-linux-gnu \
  g++-aarch64-linux-gnu \
  libasound2-dev:arm64 \
  libgl-dev:arm64 \
  libxcursor-dev:arm64 \
  libxrandr-dev:arm64 \
  libxinerama-dev:arm64 \
  libxi-dev:arm64 \
  libxxf86vm-dev:arm64 \
  libwayland-dev:arm64 \
  libxkbcommon-dev:arm64

CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ \
  CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
  go build -o _build/cue-linux-arm64 ./cmd/cue
```

## Release Builds

Release binaries for all platforms are produced via [goreleaser](https://goreleaser.com/) when a version tag is pushed.

To create a local snapshot (builds all targets, no publish):

```bash
just release-snapshot
```

This requires goreleaser v2+ installed locally.

## macOS Gatekeeper

Release binaries are **unsigned** (no Apple Developer codesigning or notarization). macOS Gatekeeper will block the binary on first launch.

**Workaround — remove quarantine flag:**
```bash
xattr -d com.apple.quarantine ./cue
```

**Alternative — allow via System Settings:**
1. Double-click `cue` — macOS blocks it
2. Open **System Settings → Privacy & Security**
3. Scroll to the "Security" section — you'll see "cue was blocked"
4. Click **Allow Anyway**
5. Double-click `cue` again — click **Open** in the confirmation dialog

This only needs to be done once per binary.

## Runtime Dependencies

At runtime, Cue needs:
- **Audio:** ALSA, PulseAudio, or PipeWire (Linux); CoreAudio (macOS — always available)
- **Display:** A running X11 or Wayland compositor (Linux); macOS desktop (always available)
- **Ollama:** Running locally on port 11434 with `neural-chat` and `nomic-embed-text` models
