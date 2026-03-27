# Cue project commands

# Build the binary
build:
    @mkdir -p _build
    go build -o _build/cue ./cmd/cue

# Run tests with short output
test:
    go test -count=1 ./...

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
run:
    go run ./cmd/cue

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

# Clean build artifacts
clean:
    rm -rf _build/
