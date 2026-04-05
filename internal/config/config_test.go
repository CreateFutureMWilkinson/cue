package config_test

import (
	"os"
	"path/filepath"
	"runtime"
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

[slack]
enabled = true
bot_token = "xoxb-test-token-123"
workspace_id = "T0001"
poll_interval_seconds = 300

[email]
enabled = false
imap_host = "imap.example.com"
imap_port = 993
username = "alice@example.com"
password_env = "MY_EMAIL_PW"
poll_interval_seconds = 120

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
batch_process = false

[gui]
host = "0.0.0.0"
port = 9090

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

	// Slack
	s.True(cfg.Slack.Enabled)
	s.Equal("xoxb-test-token-123", cfg.Slack.BotToken)
	s.Equal("T0001", cfg.Slack.WorkspaceID)
	s.Equal(300, cfg.Slack.PollIntervalSeconds)

	// Email
	s.False(cfg.Email.Enabled)
	s.Equal("imap.example.com", cfg.Email.IMAPHost)
	s.Equal(993, cfg.Email.IMAPPort)
	s.Equal("alice@example.com", cfg.Email.Username)
	s.Equal("MY_EMAIL_PW", cfg.Email.PasswordEnv)
	s.Equal(120, cfg.Email.PollIntervalSeconds)

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
	s.False(cfg.Notification.BatchProcess)

	// GUI
	s.Equal("0.0.0.0", cfg.GUI.Host)
	s.Equal(9090, cfg.GUI.Port)

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
	s.Equal("~/.cue/messages.db", cfg.Database.Path)

	s.True(cfg.Slack.Enabled)
	s.Empty(cfg.Slack.BotToken)
	s.Empty(cfg.Slack.WorkspaceID)
	s.Equal(600, cfg.Slack.PollIntervalSeconds)

	s.True(cfg.Email.Enabled)
	s.Equal("imap.gmail.com", cfg.Email.IMAPHost)
	s.Equal(993, cfg.Email.IMAPPort)
	s.Empty(cfg.Email.Username)
	s.Equal("CUE_EMAIL_PASSWORD", cfg.Email.PasswordEnv)
	s.Equal(600, cfg.Email.PollIntervalSeconds)

	s.Equal(7, cfg.Orchestrator.Router.ImportanceThreshold)
	s.InDelta(0.8, cfg.Orchestrator.Router.ConfidenceThreshold, 0.001)
	s.Equal(100, cfg.Orchestrator.Router.BufferSizePerSource)

	s.Equal("localhost", cfg.Ollama.Host)
	s.Equal(11434, cfg.Ollama.Port)
	s.Equal("neural-chat", cfg.Ollama.InferenceModel)
	s.Equal("nomic-embed-text", cfg.Ollama.EmbeddingModel)
	s.Equal(10, cfg.Ollama.TimeoutSeconds)

	s.True(cfg.Notification.AudioEnabled)
	s.True(cfg.Notification.BatchProcess)

	s.Equal("localhost", cfg.GUI.Host)
	s.Equal(8080, cfg.GUI.Port)

	s.Equal("info", cfg.Logging.LogLevel)
	s.Empty(cfg.Logging.LogDir)
}

// ---------------------------------------------------------------------------
// 3. TestValidateRequiredFields — missing/invalid required values
// ---------------------------------------------------------------------------

func (s *ConfigSuite) TestValidateRequiredFields() {
	tests := []struct {
		name    string
		toml    string
		errMsg  string // substring expected in error
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
			name: "zero gui port",
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
host = "localhost"
port = 0
`,
			errMsg: "gui.port",
		},
		{
			name: "negative poll interval slack",
			toml: `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[slack]
poll_interval_seconds = -1
`,
			errMsg: "slack.poll_interval_seconds",
		},
		{
			name: "negative poll interval email",
			toml: `
[database]
path = "/tmp/db.sqlite"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[email]
poll_interval_seconds = -5
`,
			errMsg: "email.poll_interval_seconds",
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
			name: "enabled as string instead of bool",
			toml: `
[slack]
enabled = "yes"
`,
		},
		{
			name: "confidence_threshold as string",
			toml: `
[orchestrator.router]
confidence_threshold = "high"
`,
		},
		{
			name: "poll_interval as float",
			toml: `
[slack]
poll_interval_seconds = "ten"
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

func (s *ConfigSuite) TestExpandHomePath_NoTilde() {
	dir := s.T().TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	tomlContent := `
[database]
path = "/absolute/path/messages.db"

[ollama]
host = "localhost"
port = 11434
inference_model = "neural-chat"
embedding_model = "nomic-embed-text"
timeout_seconds = 10

[logging]
log_level = "info"
log_dir = "/var/log/cue"
`
	err := os.WriteFile(cfgPath, []byte(tomlContent), 0644)
	s.Require().NoError(err)

	cfg, err := config.Load(cfgPath)
	s.Require().NoError(err)

	s.Equal("/absolute/path/messages.db", cfg.Database.Path)
	s.Equal("/var/log/cue", cfg.Logging.LogDir)
}
