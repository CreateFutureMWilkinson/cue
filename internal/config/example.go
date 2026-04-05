package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ExampleTOML returns a handwritten, annotated TOML configuration template
// with all default values matching defaultConfig().
func ExampleTOML() string {
	return `# Cue Configuration
# ==================
# This file controls local application settings.
# Service accounts (Slack, Email) are configured in the Settings UI
# and stored in the application database.

# ─── Database ───────────────────────────────────────────────
[database]
# Path to the SQLite database file.
# Supports ~ for home directory.
path = "~/.cue/messages.db"

# ─── Ollama (Local LLM) ────────────────────────────────────
[ollama]
# Ollama server connection details.
# Models must be pulled locally before use:
#   ollama pull neural-chat
#   ollama pull nomic-embed-text
host = "localhost"
port = 11434

# Model used for message importance scoring.
inference_model = "neural-chat"

# Model used for generating vector embeddings.
embedding_model = "nomic-embed-text"

# Timeout in seconds for Ollama API calls.
# Messages exceeding this are scored IS=7, CS=0.0 (BUFFERED).
timeout_seconds = 10

# ─── Routing Thresholds ────────────────────────────────────
[orchestrator]
# How often (seconds) to poll all configured watchers.
poll_interval_seconds = 600

[orchestrator.router]
# Messages scoring >= importance AND >= confidence → NOTIFIED.
# Messages scoring >= importance AND < confidence → BUFFERED.
# All others → IGNORED.
importance_threshold = 7
confidence_threshold = 0.8

# Maximum messages retained per source (FIFO eviction).
buffer_size_per_source = 100

# ─── Vector-Assisted Routing ─────────────────────────────────
# When enabled, historical user feedback influences future scoring.
# The router queries the vector store for similar previously-rated
# messages and adjusts importance scores accordingly.
vector_enabled = false

# Minimum cosine similarity to consider a match (0.0-1.0).
vector_similarity_threshold = 0.75

# Maximum number of similar messages to consider.
vector_top_n = 5

# How aggressively to adjust scores (0.0-1.0).
# 0.0 = no adjustment, 1.0 = full adjustment.
vector_damping_factor = 0.5

# ─── Notifications & Audio ─────────────────────────────────
[notification]
# Master toggle for audio alerts.
audio_enabled = true

# Process messages in batches (recommended).
batch_process = true

# Directory containing custom notification sounds.
# Leave empty to use system fallback beep.
audio_dir = ""

# Minimum seconds between consecutive audio alerts.
audio_cooldown_seconds = 2

# Volume level (0-100).
audio_volume = 100

# Fallback beep frequency in Hz (used when no audio files available).
fallback_frequency = 1000

# Fallback beep duration in milliseconds.
fallback_duration_ms = 200

# ─── GUI ────────────────────────────────────────────────────
[gui]
# Initial window dimensions.
window_width = 1200
window_height = 800

# Character displayed in the center area.
# Options: "none", "fairy"
character = "none"

# ─── Logging ────────────────────────────────────────────────
[logging]
# Log level: "debug", "info", "warn", "error"
log_level = "info"

# Directory for log files. Empty = stderr only.
log_dir = ""

# ─── Day Planner ───────────────────────────────────────────
[planner]
# Working hours (HH:MM format).
workday_start = "09:00"
workday_end = "17:00"

# Stop generating new pomodoro blocks after this time.
planning_cutoff = "16:00"

# Pomodoro cycle durations (minutes).
pomodoro_minutes = 25
short_break_minutes = 5
long_break_minutes = 20
long_break_after_cycles = 4

# Merge meetings closer than this gap (minutes).
meeting_merge_gap_minutes = 5

# Lunch window — planner avoids scheduling here.
lunch_window_start = "12:00"
lunch_window_end = "14:00"

# Timer completion sound file path. Empty = system beep.
timer_sound = ""

# Timer alert volume (0-100).
timer_volume = 75
`
}

// WriteExampleConfig writes the example TOML configuration to the given path.
// It creates parent directories if needed. If the file already exists and force
// is false, it returns an error. If force is true, it overwrites the file.
func WriteExampleConfig(path string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config file already exists: %s", path)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("creating parent directories: %w", err)
	}

	if err := os.WriteFile(path, []byte(ExampleTOML()), 0600); err != nil {
		return fmt.Errorf("writing example config: %w", err)
	}

	return nil
}
