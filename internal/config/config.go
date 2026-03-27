package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config represents the application configuration loaded from a TOML file.
type Config struct {
	Database     DatabaseConfig     `toml:"database"`
	Slack        SlackConfig        `toml:"slack"`
	Email        EmailConfig        `toml:"email"`
	Orchestrator OrchestratorConfig `toml:"orchestrator"`
	Ollama       OllamaConfig       `toml:"ollama"`
	Notification NotificationConfig `toml:"notification"`
	GUI          GUIConfig          `toml:"gui"`
	Logging      LoggingConfig      `toml:"logging"`
}

type DatabaseConfig struct {
	Path string `toml:"path"`
}

type SlackConfig struct {
	Enabled             bool   `toml:"enabled"`
	BotToken            string `toml:"bot_token"`
	WorkspaceID         string `toml:"workspace_id"`
	PollIntervalSeconds int    `toml:"poll_interval_seconds"`
}

type EmailConfig struct {
	Enabled             bool   `toml:"enabled"`
	IMAPHost            string `toml:"imap_host"`
	IMAPPort            int    `toml:"imap_port"`
	Username            string `toml:"username"`
	PasswordEnv         string `toml:"password_env"`
	PollIntervalSeconds int    `toml:"poll_interval_seconds"`
}

type OrchestratorConfig struct {
	Router RouterConfig `toml:"router"`
}

type RouterConfig struct {
	ImportanceThreshold int     `toml:"importance_threshold"`
	ConfidenceThreshold float64 `toml:"confidence_threshold"`
	BufferSizePerSource int     `toml:"buffer_size_per_source"`
}

type OllamaConfig struct {
	Host           string `toml:"host"`
	Port           int    `toml:"port"`
	InferenceModel string `toml:"inference_model"`
	EmbeddingModel string `toml:"embedding_model"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
}

type NotificationConfig struct {
	AudioEnabled bool `toml:"audio_enabled"`
	BatchProcess bool `toml:"batch_process"`
}

type GUIConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type LoggingConfig struct {
	LogLevel string `toml:"log_level"`
	LogDir   string `toml:"log_dir"`
}

// defaultConfig returns a Config populated with the default values from the spec.
func defaultConfig() *Config {
	return &Config{
		Database: DatabaseConfig{
			Path: "~/.cue/messages.db",
		},
		Slack: SlackConfig{
			Enabled:             true,
			PollIntervalSeconds: 600,
		},
		Email: EmailConfig{
			Enabled:             true,
			IMAPHost:            "imap.gmail.com",
			IMAPPort:            993,
			PasswordEnv:         "CUE_EMAIL_PASSWORD",
			PollIntervalSeconds: 600,
		},
		Orchestrator: OrchestratorConfig{
			Router: RouterConfig{
				ImportanceThreshold: 7,
				ConfidenceThreshold: 0.8,
				BufferSizePerSource: 100,
			},
		},
		Ollama: OllamaConfig{
			Host:           "localhost",
			Port:           11434,
			InferenceModel: "neural-chat",
			EmbeddingModel: "nomic-embed-text",
			TimeoutSeconds: 10,
		},
		Notification: NotificationConfig{
			AudioEnabled: true,
			BatchProcess: true,
		},
		GUI: GUIConfig{
			Host: "localhost",
			Port: 8080,
		},
		Logging: LoggingConfig{
			LogLevel: "info",
		},
	}
}

const defaultTOML = `[database]
path = "~/.cue/messages.db"

[slack]
enabled = true
bot_token = ""
workspace_id = ""
poll_interval_seconds = 600

[email]
enabled = true
imap_host = "imap.gmail.com"
imap_port = 993
username = ""
password_env = "CUE_EMAIL_PASSWORD"
poll_interval_seconds = 600

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

[notification]
audio_enabled = true
batch_process = true

[gui]
host = "localhost"
port = 8080

[logging]
log_level = "info"
log_dir = ""
`

// Load reads the TOML config at the given path. If the file does not exist,
// it creates the parent directories, writes a default config, and returns it.
func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(filepath.Dir(path), 0755); mkErr != nil {
			return nil, fmt.Errorf("creating config directory: %w", mkErr)
		}
		if wErr := os.WriteFile(path, []byte(defaultTOML), 0644); wErr != nil {
			return nil, fmt.Errorf("writing default config: %w", wErr)
		}
		cfg := defaultConfig()
		return cfg, nil
	}

	cfg := &Config{}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}

	expandPaths(cfg)
	return cfg, nil
}

// expandPaths replaces leading ~/ with the user's home directory in path fields.
func expandPaths(cfg *Config) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	cfg.Database.Path = expandTilde(cfg.Database.Path, home)
	cfg.Logging.LogDir = expandTilde(cfg.Logging.LogDir, home)
}

func expandTilde(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// Validate checks that required configuration fields are set correctly.
func (c *Config) Validate() error {
	if c.Database.Path == "" {
		return fmt.Errorf("database.path must not be empty")
	}
	if c.Ollama.Host == "" {
		return fmt.Errorf("ollama.host must not be empty")
	}
	if c.Ollama.Port <= 0 {
		return fmt.Errorf("ollama.port must be greater than 0")
	}
	if c.Ollama.InferenceModel == "" {
		return fmt.Errorf("ollama.inference_model must not be empty")
	}
	if c.Ollama.EmbeddingModel == "" {
		return fmt.Errorf("ollama.embedding_model must not be empty")
	}
	if c.Slack.PollIntervalSeconds < 0 {
		return fmt.Errorf("slack.poll_interval_seconds must not be negative")
	}
	if c.Email.PollIntervalSeconds < 0 {
		return fmt.Errorf("email.poll_interval_seconds must not be negative")
	}
	if c.GUI.Port <= 0 {
		return fmt.Errorf("gui.port must be greater than 0")
	}
	return nil
}
