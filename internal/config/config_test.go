package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	suite.Suite
}

func TestConfig(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}

// ---------------------------------------------------------------------------
// 1. TestLoadValidConfig — full round-trip parse of every field
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestLoadValidConfig() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/data/messages.db"

[orchestrator.router]
importance_threshold = 8
confidence_threshold = 0.9
buffer_size_per_source = 50

[ollama]
host = "192.168.1.10"
port = 11435
inference_model = "llama3"
embedding_model = "mxbai-embed-large"
timeout_seconds = 30

[notification]
audio_enabled = false

[gui]
window_width = 1920
window_height = 1080

[logging]
log_level = "debug"
log_dir = "/var/log/cue"
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	// Database
	s.Equal("/data/messages.db", cfg.Database.Path)

	// Orchestrator / Router
	s.Equal(8, cfg.Orchestrator.Router.ImportanceThreshold)
	s.InDelta(0.9, cfg.Orchestrator.Router.ConfidenceThreshold, 0.001)
	s.Equal(50, cfg.Orchestrator.Router.BufferSizePerSource)

	// Ollama
	s.Equal("192.168.1.10", cfg.Ollama.Host)
	s.Equal(11435, cfg.Ollama.Port)
	s.Equal("llama3", cfg.Ollama.InferenceModel)
	s.Equal("mxbai-embed-large", cfg.Ollama.EmbeddingModel)
	s.Equal(30, cfg.Ollama.TimeoutSeconds)

	// Notification
	s.False(cfg.Notification.AudioEnabled)

	// GUI
	s.Equal(1920, cfg.GUI.WindowWidth)
	s.Equal(1080, cfg.GUI.WindowHeight)

	// Logging
	s.Equal("debug", cfg.Logging.LogLevel)
	s.Equal("/var/log/cue", cfg.Logging.LogDir)
}

// ---------------------------------------------------------------------------
// 2. TestCreateDefaultConfigIfMissing — auto-create with sane defaults
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestCreateDefaultConfigIfMissing() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "subdir", "config.toml")

	// File must not exist yet.
	_, err := os.Stat(cfgPath)
	s.True(os.IsNotExist(err))

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	// File should now exist on disk.
	_, err = os.Stat(cfgPath)
	s.NoError(err)

	// Verify every default value matches CLAUDE.md specification.
	home, err := os.UserHomeDir()
	s.Require().NoError(err)
	s.Equal(filepath.Join(home, ".cue", "messages.db"), cfg.Database.Path)

	s.Equal(7, cfg.Orchestrator.Router.ImportanceThreshold)
	s.InDelta(0.8, cfg.Orchestrator.Router.ConfidenceThreshold, 0.001)
	s.Equal(100, cfg.Orchestrator.Router.BufferSizePerSource)

	s.Equal("localhost", cfg.Ollama.Host)
	s.Equal(11434, cfg.Ollama.Port)
	s.Equal("neural-chat", cfg.Ollama.InferenceModel)
	s.Equal("nomic-embed-text", cfg.Ollama.EmbeddingModel)
	s.Equal(10, cfg.Ollama.TimeoutSeconds)

	s.True(cfg.Notification.AudioEnabled)

	s.Equal(1200, cfg.GUI.WindowWidth)
	s.Equal(800, cfg.GUI.WindowHeight)

	s.Equal("info", cfg.Logging.LogLevel)
	s.Empty(cfg.Logging.LogDir)
}

// ---------------------------------------------------------------------------
// 3. TestValidateRequiredFields — missing/invalid required values
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestValidateRequiredFields() {
	tests := []struct {
		name   string
		toml   string
		errMsg string // substring expected in error
	}{
		{
			name: "empty database path",
			toml: `
[database]
path = ""

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10
`,
			errMsg: "database.path",
		},
		{
			name: "empty ollama host",
			toml: `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = ""
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10
`,
			errMsg: "ollama.host",
		},
		{
			name: "zero ollama port",
			toml: `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 0
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10
`,
			errMsg: "ollama.port",
		},
		{
			name: "missing inference model",
			toml: `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = ""
embedding_model = "nomic-embed-text"
timeout_seconds = 10
`,
			errMsg: "ollama.inference_model",
		},
		{
			name: "missing embedding model",
			toml: `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = ""
timeout_seconds = 10
`,
			errMsg: "ollama.embedding_model",
		},
		{
			name: "zero gui window_width",
			toml: `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[gui]
window_width = 0
window_height = 800
`,
			errMsg: "gui.window_width",
		},
		{
			name: "zero gui window_height",
			toml: `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[gui]
window_width = 1200
window_height = 0
`,
			errMsg: "gui.window_height",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			dir := s.T().TempDir()
			cfgPath := filepath.Join(dir, "config.toml")
			err := os.WriteFile(cfgPath, []byte(tc.toml), 0644)
			s.Require().NoError(err)

			cfg, err := config.Load(cfgPath)
			if err != nil {
				// Load itself may reject the config.
				s.Contains(err.Error(), tc.errMsg)
				return
			}

			// If Load succeeded, explicit Validate must catch it.
			err = cfg.Validate()
			s.Require().Error(err, "expected validation error for: %s", tc.name)
			s.Contains(err.Error(), tc.errMsg)
		})
	}
}

// ---------------------------------------------------------------------------
// 4. TestValidateFieldTypes — TOML type mismatches
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestValidateFieldTypes() {
	tests := []struct {
		name string
		toml string
	}{
		{
			name: "port as string",
			toml: `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = "not-a-number"
`,
		},
		{
			name: "confidence_threshold as string",
			toml: `
[orchestrator.router]
confidence_threshold = "high"
`,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			dir := s.T().TempDir()
			cfgPath := filepath.Join(dir, "config.toml")
			err := os.WriteFile(cfgPath, []byte(tc.toml), 0644)
			s.Require().NoError(err)

			_, err = config.Load(cfgPath)
			s.Error(err, "expected type mismatch error for: %s", tc.name)
		})
	}
}

// ---------------------------------------------------------------------------
// 5. TestExpandHomePath — tilde expansion in path fields
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestExpandHomePath() {
	if runtime.GOOS == "windows" {
		s.T().Skip("tilde expansion test targets Unix-like systems")
	}

	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "~/.cue/messages.db"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[logging]
log_level = "info"
log_dir = "~/logs/cue"
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)

	home, err := os.UserHomeDir()
	s.Require().NoError(err)

	// Database path must be expanded.
	s.Equal(filepath.Join(home, ".cue", "messages.db"), cfg.Database.Path)
	s.NotContains(cfg.Database.Path, "~")

	// Logging dir must be expanded.
	s.Equal(filepath.Join(home, "logs", "cue"), cfg.Logging.LogDir)
	s.NotContains(cfg.Logging.LogDir, "~")
}

// ---------------------------------------------------------------------------
// 5b. TestExpandHomePath_NoTilde — paths without ~ are unchanged
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// 6. TestLoadValidConfigWithAudioFields — round-trip parse of new audio fields
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestLoadValidConfigWithAudioFields() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	// Create a real directory to use as audio_dir.
	audioDir := filepath.Join(dir, "sounds")
	err := os.MkdirAll(audioDir, 0750)
	s.Require().NoError(err)

	tomlContent := `
[database]
path = "/data/messages.db"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[notification]
audio_enabled = true
batch_process = true
audio_dir = "` + audioDir + `"
audio_cooldown_seconds = 5
audio_volume = 75
fallback_frequency = 440
fallback_duration_ms = 500
`
	err = os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Equal(audioDir, cfg.Notification.AudioDir)
	s.Equal(5, cfg.Notification.AudioCooldownSeconds)
	s.Equal(75, cfg.Notification.AudioVolume)
	s.Equal(440, cfg.Notification.FallbackFrequency)
	s.Equal(500, cfg.Notification.FallbackDurationMs)
}

// ---------------------------------------------------------------------------
// 7. TestCreateDefaultConfigAudioDefaults — default values for new audio fields
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestCreateDefaultConfigAudioDefaults() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "nonexistent", "config.toml")

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Empty(cfg.Notification.AudioDir)
	s.Equal(2, cfg.Notification.AudioCooldownSeconds)
	s.Equal(100, cfg.Notification.AudioVolume)
	s.Equal(1000, cfg.Notification.FallbackFrequency)
	s.Equal(200, cfg.Notification.FallbackDurationMs)
}

// ---------------------------------------------------------------------------
// 8. TestValidateAudioDirNotExist — non-existent audio_dir must fail
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestValidateAudioDirNotExist() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[notification]
audio_dir = "/nonexistent/path/to/sounds"
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		s.Contains(err.Error(), "notification.audio_dir")
		return
	}

	err = cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "notification.audio_dir")
}

// ---------------------------------------------------------------------------
// 9. TestValidateAudioDirExists — real directory passes validation
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestValidateAudioDirExists() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	audioDir := filepath.Join(dir, "sounds")
	err := os.MkdirAll(audioDir, 0750)
	s.Require().NoError(err)

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[notification]
audio_dir = "` + audioDir + `"
audio_cooldown_seconds = 2
audio_volume = 100
fallback_frequency = 1000
fallback_duration_ms = 200
`
	err = os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)

	err = cfg.Validate()
	s.NoError(err)
}

// ---------------------------------------------------------------------------
// 10. TestValidateAudioDirEmpty — empty audio_dir is allowed (fallback mode)
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestValidateAudioDirEmpty() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[notification]
audio_dir = ""
audio_cooldown_seconds = 2
audio_volume = 100
fallback_frequency = 1000
fallback_duration_ms = 200
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)

	err = cfg.Validate()
	s.NoError(err)
}

// ---------------------------------------------------------------------------
// 11. TestValidateAudioCooldownNegative — negative cooldown must fail
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestValidateAudioCooldownNegative() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[notification]
audio_cooldown_seconds = -1
audio_volume = 100
fallback_frequency = 1000
fallback_duration_ms = 200
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		s.Contains(err.Error(), "notification.audio_cooldown_seconds")
		return
	}

	err = cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "notification.audio_cooldown_seconds")
}

// ---------------------------------------------------------------------------
// 12. TestValidateAudioVolumeOutOfRange — volume < 0 and > 100 must fail
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestValidateAudioVolumeOutOfRange() {
	tests := []struct {
		name   string
		volume int
	}{
		{"volume below zero", -1},
		{"volume above 100", 101},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			dir := s.T().TempDir()
			cfgPath := filepath.Join(dir, "config.toml")

			tomlContent := fmt.Sprintf(`
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[notification]
audio_cooldown_seconds = 2
audio_volume = %d
fallback_frequency = 1000
fallback_duration_ms = 200
`, tc.volume)
			err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
			s.Require().NoError(err)

			cfg, err := config.Load(cfgPath)
			if err != nil {
				s.Contains(err.Error(), "notification.audio_volume")
				return
			}

			err = cfg.Validate()
			s.Require().Error(err, "expected validation error for %s", tc.name)
			s.Contains(err.Error(), "notification.audio_volume")
		})
	}
}

// ---------------------------------------------------------------------------
// 13. TestValidateFallbackFrequencyZero — zero frequency must fail
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestValidateFallbackFrequencyZero() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[notification]
audio_cooldown_seconds = 2
audio_volume = 100
fallback_frequency = 0
fallback_duration_ms = 200
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		s.Contains(err.Error(), "notification.fallback_frequency")
		return
	}

	err = cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "notification.fallback_frequency")
}

// ---------------------------------------------------------------------------
// 14. TestValidateFallbackDurationZero — zero duration must fail
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestValidateFallbackDurationZero() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[notification]
audio_cooldown_seconds = 2
audio_volume = 100
fallback_frequency = 1000
fallback_duration_ms = 0
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		s.Contains(err.Error(), "notification.fallback_duration_ms")
		return
	}

	err = cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "notification.fallback_duration_ms")
}

// ---------------------------------------------------------------------------
// 15. TestExpandHomePathAudioDir — tilde expansion for audio_dir
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestExpandHomePathAudioDir() {
	if runtime.GOOS == "windows" {
		s.T().Skip("tilde expansion test targets Unix-like systems")
	}

	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[notification]
audio_dir = "~/sounds"
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)

	home, err := os.UserHomeDir()
	s.Require().NoError(err)

	s.Equal(filepath.Join(home, "sounds"), cfg.Notification.AudioDir)
	s.NotContains(cfg.Notification.AudioDir, "~")
}

// ---------------------------------------------------------------------------
// 16. TestGUICharacterFieldParsesFromTOML — character field in [gui] section
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestGUICharacterFieldParsesFromTOML() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[gui]
window_width = 1200
window_height = 800
character = "fairy"
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Equal("fairy", cfg.GUI.Character)
}

// ---------------------------------------------------------------------------
// 17. TestGUICharacterDefaultsToNone — missing character defaults to "none"
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestGUICharacterDefaultsToNone() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "nonexistent", "config.toml")

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Equal("none", cfg.GUI.Character)
}

// ---------------------------------------------------------------------------
// 18. TestPlannerConfigDefaults — default config has planner defaults populated
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestPlannerConfigDefaults() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "nonexistent", "config.toml")

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Equal("09:00", cfg.Planner.WorkdayStart)
	s.Equal("17:00", cfg.Planner.WorkdayEnd)
	s.Equal("16:00", cfg.Planner.PlanningCutoff)
	s.Equal(25, cfg.Planner.PomodoroMinutes)
	s.Equal(5, cfg.Planner.ShortBreakMinutes)
	s.Equal(20, cfg.Planner.LongBreakMinutes)
	s.Equal(4, cfg.Planner.LongBreakAfterCycles)
	s.Equal(5, cfg.Planner.MeetingMergeGapMinutes)
	s.Equal("12:00", cfg.Planner.LunchWindowStart)
	s.Equal("14:00", cfg.Planner.LunchWindowEnd)
}

// ---------------------------------------------------------------------------
// 19. TestPlannerConfigParses — TOML with [planner] section parses correctly
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestPlannerConfigParses() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[planner]
workday_start = "08:00"
workday_end = "16:00"
planning_cutoff = "15:00"
pomodoro_minutes = 30
short_break_minutes = 10
long_break_minutes = 25
long_break_after_cycles = 3
meeting_merge_gap_minutes = 10
lunch_window_start = "11:30"
lunch_window_end = "13:30"
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Equal("08:00", cfg.Planner.WorkdayStart)
	s.Equal("16:00", cfg.Planner.WorkdayEnd)
	s.Equal("15:00", cfg.Planner.PlanningCutoff)
	s.Equal(30, cfg.Planner.PomodoroMinutes)
	s.Equal(10, cfg.Planner.ShortBreakMinutes)
	s.Equal(25, cfg.Planner.LongBreakMinutes)
	s.Equal(3, cfg.Planner.LongBreakAfterCycles)
	s.Equal(10, cfg.Planner.MeetingMergeGapMinutes)
	s.Equal("11:30", cfg.Planner.LunchWindowStart)
	s.Equal("13:30", cfg.Planner.LunchWindowEnd)
}

// ---------------------------------------------------------------------------
// 20. TestPlannerConfigValidation — invalid planner config fails validation
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestPlannerConfigValidation() {
	tests := []struct {
		name   string
		toml   string
		errMsg string
	}{
		{
			name: "workday_end before workday_start",
			toml: `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[planner]
workday_start = "17:00"
workday_end = "09:00"
planning_cutoff = "16:00"
pomodoro_minutes = 25
short_break_minutes = 5
long_break_minutes = 20
long_break_after_cycles = 4
meeting_merge_gap_minutes = 5
lunch_window_start = "12:00"
lunch_window_end = "14:00"
`,
			errMsg: "planner.workday_end must be after",
		},
		{
			name: "zero pomodoro_minutes",
			toml: `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[planner]
workday_start = "09:00"
workday_end = "17:00"
planning_cutoff = "16:00"
pomodoro_minutes = 0
short_break_minutes = 5
long_break_minutes = 20
long_break_after_cycles = 4
meeting_merge_gap_minutes = 5
lunch_window_start = "12:00"
lunch_window_end = "14:00"
`,
			errMsg: "planner.pomodoro_minutes",
		},
		{
			name: "invalid workday_start format",
			toml: `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[planner]
workday_start = "nine"
workday_end = "17:00"
planning_cutoff = "16:00"
pomodoro_minutes = 25
short_break_minutes = 5
long_break_minutes = 20
long_break_after_cycles = 4
meeting_merge_gap_minutes = 5
lunch_window_start = "12:00"
lunch_window_end = "14:00"
`,
			errMsg: "planner.workday_start",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			dir := s.T().TempDir()
			cfgPath := filepath.Join(dir, "config.toml")
			err := os.WriteFile(cfgPath, []byte(tc.toml), 0644)
			s.Require().NoError(err)

			cfg, err := config.Load(cfgPath)
			if err != nil {
				s.Contains(err.Error(), tc.errMsg)
				return
			}

			err = cfg.Validate()
			s.Require().Error(err, "expected validation error for: %s", tc.name)
			s.Contains(err.Error(), tc.errMsg)
		})
	}
}

// ---------------------------------------------------------------------------
// 21. TestPlannerTimerSoundDefaults — default config has empty TimerSound and TimerVolume=75
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestPlannerTimerSoundDefaults() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "nonexistent", "config.toml")

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Empty(cfg.Planner.TimerSound, "default TimerSound should be empty (fallback beep)")
	s.Equal(75, cfg.Planner.TimerVolume, "default TimerVolume should be 75")
}

// ---------------------------------------------------------------------------
// 22. TestPlannerTimerVolumeValidation — timer_volume must be 0-100
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestPlannerTimerVolumeValidation() {
	tests := []struct {
		name   string
		volume int
	}{
		{"timer_volume below zero", -1},
		{"timer_volume above 100", 101},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			dir := s.T().TempDir()
			cfgPath := filepath.Join(dir, "config.toml")

			tomlContent := fmt.Sprintf(`
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[planner]
workday_start = "09:00"
workday_end = "17:00"
planning_cutoff = "16:00"
pomodoro_minutes = 25
short_break_minutes = 5
long_break_minutes = 20
long_break_after_cycles = 4
meeting_merge_gap_minutes = 5
lunch_window_start = "12:00"
lunch_window_end = "14:00"
timer_volume = %d
`, tc.volume)
			err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
			s.Require().NoError(err)

			cfg, err := config.Load(cfgPath)
			if err != nil {
				s.Contains(err.Error(), "planner.timer_volume")
				return
			}

			err = cfg.Validate()
			s.Require().Error(err, "expected validation error for %s", tc.name)
			s.Contains(err.Error(), "planner.timer_volume")
		})
	}
}

// ---------------------------------------------------------------------------
// 23. TestPlannerTimerSoundParsesFromTOML — timer_sound field parsed correctly
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestPlannerTimerSoundParsesFromTOML() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[planner]
workday_start = "09:00"
workday_end = "17:00"
planning_cutoff = "16:00"
pomodoro_minutes = 25
short_break_minutes = 5
long_break_minutes = 20
long_break_after_cycles = 4
meeting_merge_gap_minutes = 5
lunch_window_start = "12:00"
lunch_window_end = "14:00"
timer_sound = "/home/user/sounds/timer.wav"
timer_volume = 80
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Equal("/home/user/sounds/timer.wav", cfg.Planner.TimerSound)
	s.Equal(80, cfg.Planner.TimerVolume)
}

// ---------------------------------------------------------------------------
// 24. TestOrchestratorPollIntervalDefault — default poll_interval_seconds is 600
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestOrchestratorPollIntervalDefault() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "nonexistent", "config.toml")

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Equal(600, cfg.Orchestrator.PollIntervalSeconds,
		"default poll_interval_seconds should be 600")
}

// ---------------------------------------------------------------------------
// 25. TestOrchestratorPollIntervalParsesFromTOML — custom value parsed correctly
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestOrchestratorPollIntervalParsesFromTOML() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[orchestrator]
poll_interval_seconds = 300

[orchestrator.router]
importance_threshold = 7
confidence_threshold = 0.8
buffer_size_per_source = 100

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Equal(300, cfg.Orchestrator.PollIntervalSeconds,
		"poll_interval_seconds should be parsed from TOML")
}

// ---------------------------------------------------------------------------
// 26. TestValidateOrchestratorPollIntervalZero — zero poll interval must fail
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestValidateOrchestratorPollIntervalZero() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[orchestrator]
poll_interval_seconds = 0

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		s.Contains(err.Error(), "orchestrator.poll_interval_seconds")
		return
	}

	err = cfg.Validate()
	s.Require().Error(err, "expected validation error for zero poll_interval_seconds")
	s.Contains(err.Error(), "orchestrator.poll_interval_seconds")
}

// ---------------------------------------------------------------------------
// 27. TestValidateOrchestratorPollIntervalNegative — negative poll interval must fail
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestValidateOrchestratorPollIntervalNegative() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[orchestrator]
poll_interval_seconds = -10

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		s.Contains(err.Error(), "orchestrator.poll_interval_seconds")
		return
	}

	err = cfg.Validate()
	s.Require().Error(err, "expected validation error for negative poll_interval_seconds")
	s.Contains(err.Error(), "orchestrator.poll_interval_seconds")
}

// ---------------------------------------------------------------------------
// TestIgnoresUnknownSections — old TOML files with [slack]/[email] parse OK
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestIgnoresUnknownSections() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[slack]
enabled = true
bot_token = "xoxb-old-token"
workspace_id = "T0001"
poll_interval_seconds = 300

[email]
enabled = false
imap_host = "imap.example.com"
imap_port = 993
username = "alice@example.com"
password_env = "MY_EMAIL_PW"
poll_interval_seconds = 120
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err, "stale config with [slack]/[email] sections must parse without error")
	s.Require().NotNil(cfg)

	// Core fields still load correctly.
	s.Equal("/tmp/db.sqlite", cfg.Database.Path)
	s.Equal("localhost", cfg.Ollama.Host)
}

// ---------------------------------------------------------------------------
// TestDefaultTOMLHasNoSlackOrEmail — generated default has no [slack]/[email]
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestDefaultTOMLHasNoSlackOrEmail() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "subdir", "config.toml")

	// Load from non-existent path triggers default creation.
	_, err := config.Load(cfgPath)
	s.Require().NoError(err)

	// Read the generated file back from disk.
	data, err := os.ReadFile(cfgPath)
	s.Require().NoError(err)

	content := string(data)
	s.False(strings.Contains(content, "[slack]"),
		"default TOML must not contain [slack] section, got:\n%s", content)
	s.False(strings.Contains(content, "[email]"),
		"default TOML must not contain [email] section, got:\n%s", content)
}

// ---------------------------------------------------------------------------
// Feature 042: Vector routing config defaults and validation
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestVectorConfigDefaults() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)

	s.False(cfg.Orchestrator.Router.VectorEnabled, "vector_enabled should default to false")
	s.InDelta(0.75, cfg.Orchestrator.Router.VectorSimilarityThreshold, 0.001)
	s.Equal(5, cfg.Orchestrator.Router.VectorTopN)
	s.InDelta(0.5, cfg.Orchestrator.Router.VectorDampingFactor, 0.001)
}

func (s *ConfigSuite) TestVectorConfigValidation_DampingFactorTooHigh() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"

[orchestrator.router]
vector_enabled = true
vector_damping_factor = 1.5
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		s.Contains(err.Error(), "damping")
		return
	}

	err = cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "damping")
}

func (s *ConfigSuite) TestVectorConfigValidation_DampingFactorNegative() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"

[orchestrator.router]
vector_enabled = true
vector_damping_factor = -0.1
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		s.Contains(err.Error(), "damping")
		return
	}

	err = cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "damping")
}

func (s *ConfigSuite) TestVectorConfigValidation_TopNZero() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"

[orchestrator.router]
vector_enabled = true
vector_top_n = 0
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		s.Contains(err.Error(), "top_n")
		return
	}

	err = cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "top_n")
}

func (s *ConfigSuite) TestVectorConfigValidation_SimilarityThresholdOutOfRange() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"

[orchestrator.router]
vector_enabled = true
vector_similarity_threshold = 1.5
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		s.Contains(err.Error(), "similarity")
		return
	}

	err = cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "similarity")
}

func (s *ConfigSuite) TestVectorConfigValidation_SimilarityThresholdNegative() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"

[orchestrator.router]
vector_enabled = true
vector_similarity_threshold = -0.1
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		s.Contains(err.Error(), "similarity")
		return
	}

	err = cfg.Validate()
	s.Require().Error(err)
	s.Contains(err.Error(), "similarity")
}

// ---------------------------------------------------------------------------
// Feature 048: BatchProcess field removal — backward compatibility
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestBatchProcessFieldIgnored() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/data/messages.db"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[notification]
audio_enabled = true
batch_process = true
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.NoError(err, "TOML with batch_process should load without error (unknown keys ignored)")
	s.NotNil(cfg)
}

// ---------------------------------------------------------------------------
// Feature 069: CharacterDir config field
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestGUIConfigCharacterDirDefault() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "nonexistent", "config.toml")

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	home, err := os.UserHomeDir()
	s.Require().NoError(err)

	expected := filepath.Join(home, ".cue", "characters")
	s.Equal(expected, cfg.GUI.CharacterDir)
}

func (s *ConfigSuite) TestGUIConfigCharacterDirParsesFromTOML() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/data/messages.db"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[gui]
character_dir = "/custom/plugins"
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	s.Equal("/custom/plugins", cfg.GUI.CharacterDir)
}

func (s *ConfigSuite) TestExpandPathsExpandsCharacterDir() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/data/messages.db"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[gui]
character_dir = "~/my-characters"
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)
	s.Require().NotNil(cfg)

	home, err := os.UserHomeDir()
	s.Require().NoError(err)

	s.True(strings.HasPrefix(cfg.GUI.CharacterDir, home),
		"CharacterDir should start with home dir, got: %s", cfg.GUI.CharacterDir)
	s.NotContains(cfg.GUI.CharacterDir, "~")
}
