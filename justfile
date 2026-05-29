# Cue project commands

# Platform detection
os := os()
arch := arch()

# CGO package lists by distro family
_deb_pkgs := "libasound2-dev libgl-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev libwayland-dev libxkbcommon-dev wayland-protocols pkg-config"
_rpm_pkgs := "alsa-lib-devel mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel wayland-devel libxkbcommon-devel wayland-protocols-devel pkgconf-pkg-config"
_arch_pkgs := "alsa-lib mesa libxcursor libxrandr libxinerama libxi libxxf86vm wayland wayland-protocols libxkbcommon pkgconf"

# Detect Linux distro family
_distro := if os == "linux" {
    `if [ -f /etc/os-release ]; then . /etc/os-release; case "${ID:-} ${ID_LIKE:-}" in *arch*) echo arch;; *debian* | *ubuntu*) echo deb;; *fedora* | *rhel*) echo rpm;; *suse*) echo rpm;; *) echo unknown;; esac; else echo unknown; fi`
} else {
    ""
}

# Detect display server (Linux only)
_display_server := if os == "linux" {
    `if [ -n "${WAYLAND_DISPLAY:-}" ]; then echo wayland; elif [ -n "${DISPLAY:-}" ]; then echo x11; else echo none; fi`
} else if os == "macos" {
    "cocoa"
} else {
    "unknown"
}

# Check whether Wayland dev headers are available at build time
_wayland_build := if os == "linux" {
    `pkg-config --exists wayland-client 2>/dev/null && echo ok || echo missing`
} else {
    "n/a"
}

# Check whether X11 dev headers are available at build time
_x11_build := if os == "linux" {
    `pkg-config --exists xcursor xrandr xinerama xi xxf86vm 2>/dev/null && echo ok || echo missing`
} else {
    "n/a"
}

# Check whether CGO build dependencies are installed
_check_deps := if os == "macos" {
    `xcode-select -p >/dev/null 2>&1 && echo ok || echo missing`
} else if os == "linux" {
    `pkg-config --exists alsa gl xcursor xrandr xinerama xi xxf86vm wayland-client xkbcommon 2>/dev/null && echo ok || echo missing`
} else {
    "unknown"
}

# Build the binary
build: clean
    {{ if _check_deps == "missing" { "@ echo 'WARNING: Build dependencies not found. Run just deps to see install instructions.'" } else { "" } }}
    {{ if os == "linux" { if _wayland_build == "missing" { "@ echo 'WARNING: Wayland headers not found — binary will only support X11. Run just deps to install.'" } else { "" } } else { "" } }}
    {{ if os == "linux" { if _x11_build == "missing" { "@ echo 'WARNING: X11 headers not found — binary will only support Wayland. Run just deps to install.'" } else { "" } } else { "" } }}
    @mkdir -p _build
    CGO_ENABLED=1 go build -o _build/cue ./cmd/cue

# Run tests with short output
test:
    go test -count=1 ./...

# Run UI acceptance tests (headless)
test-ui:
    go test -count=1 -tags=ui_acceptance ./tests/ui/...

# Run tests with verbose output
test-verbose:
    go test -count=1 -v ./...

# Run tests with coverage report
test-coverage:
    go test -count=1 -coverprofile=_build/coverage.out ./...
    go tool cover -html=_build/coverage.out -o _build/coverage.html
    @echo "Coverage report: _build/coverage.html"

# Watch for changes and re-run tests
watch:
    find . -name '*.go' | entr -c just test

# Run the application
run: build
    ./_build/cue

# Format all Go code
fmt:
    go fmt ./...

# Lint: check formatting + vet
lint:
    @test -z "$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)
    go vet ./...

# Tidy modules
tidy:
    go mod tidy && go mod verify

# Security scan
security:
    gosec ./...

# Vulnerability check
vulncheck:
    govulncheck ./...

# Run the character UAT harness via main binary
run-uat: build
    ./_build/cue uat

# Build model benchmark tool
bench:
    @mkdir -p _build
    go build -o _build/cue-bench ./cmd/cue-bench/

# Build WASM character plugins
build-plugins:
    @mkdir -p _build/characters
    GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o _build/characters/echo.wasm ./cmd/echo-plugin

build-server:
    {{ if _check_deps == "missing" { "@ echo 'WARNING: Build dependencies not found. Run just deps to see install instructions.'" } else { "" } }}
    {{ if os == "linux" { if _wayland_build == "missing" { "@ echo 'WARNING: Wayland headers not found — binary will only support X11. Run just deps to install.'" } else { "" } } else { "" } }}
    {{ if os == "linux" { if _x11_build == "missing" { "@ echo 'WARNING: X11 headers not found — binary will only support Wayland. Run just deps to install.'" } else { "" } } else { "" } }}
    @mkdir -p _build
    CGO_ENABLED=1 go build -o _build/cue-server ./cmd/cue-server
# Build both binaries and plugins for current platform
build-all: build build-plugins build-server

# Show required system packages for current platform
deps:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Platform:       {{os}}/{{arch}}"
    {{ if os == "macos" { 'echo "Distro:         macOS"' } else if os == "linux" { 'echo "Distro:         ' + _distro + '"' } else { 'echo "Distro:         unknown"' } }}
    echo "Display server: {{_display_server}}"
    {{ if os == "linux" {
        'echo ""; echo "Build-time backends:"; echo "  X11:     ' + (if _x11_build == "ok" { "✓ headers found" } else { "✗ headers missing" }) + '"; echo "  Wayland: ' + (if _wayland_build == "ok" { "✓ headers found" } else { "✗ headers missing" }) + '"'
    } else { "" } }}
    echo ""
    {{ if os == "macos" {
        'if xcode-select -p >/dev/null 2>&1; then echo "✓ Xcode Command Line Tools already installed"; else echo "Install Xcode Command Line Tools:"; echo "  xcode-select --install"; fi'
    } else if _distro == "deb" {
        'echo "Install build dependencies:"; echo "  sudo apt-get install -y ' + _deb_pkgs + '"'
    } else if _distro == "rpm" {
        'echo "Install build dependencies:"; echo "  sudo dnf install -y ' + _rpm_pkgs + '"'
    } else if _distro == "arch" {
        'echo "Install build dependencies:"; echo "  sudo pacman -S --needed ' + _arch_pkgs + '"'
    } else {
        'echo "Unknown platform. See docs/guides/Building.md for package lists."'
    } }}
    {{ if os == "linux" {
        'echo ""; echo "Note: Both X11 and Wayland backends are compiled into the same binary."; echo "GLFW auto-detects the display server at runtime via WAYLAND_DISPLAY."; echo "See docs/guides/Building.md for details on forcing a specific backend."'
    } else { "" } }}

# Create a local goreleaser snapshot (no publish)
release-snapshot:
    goreleaser release --snapshot --clean

# Check that a package compiles without producing a binary
check *args='./...':
    go build {{args}}

# Run tests for a specific package (e.g. just test-pkg ./internal/service/decisionengine/)
test-pkg *args:
    go test -count=1 -v {{args}}

# Generate OpenAPI 3.1 spec from swaggo/swag annotations
api-gen:
    @mkdir -p docs/api
    go tool swag init \
        --v3.1 \
        --generalInfo server.go \
        --dir internal/server,internal/server/handler \
        --parseInternal \
        --output docs/api \
        --outputTypes yaml \
        --quiet
    @mv docs/api/swagger.yaml docs/api/openapi.yaml

# Validate the generated OpenAPI 3.1 spec
api-lint:
    go run ./scripts/api-lint docs/api/openapi.yaml

# Clean build artifacts
clean:
    rm -rf _build/
