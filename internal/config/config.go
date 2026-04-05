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
	AudioEnabled         bool   `toml:"audio_enabled"`
	BatchProcess         bool   `toml:"batch_process"`
	AudioDir             string `toml:"audio_dir"`
	AudioCooldownSeconds int    `toml:"audio_cooldown_seconds"`
	AudioVolume          int    `toml:"audio_volume"`
	FallbackFrequency    int    `toml:"fallback_frequency"`
	FallbackDurationMs   int    `toml:"fallback_duration_ms"`
}

// notificationAudioConfigured returns true if any audio-specific notification
// field has been set to a non-zero value, indicating the section was configured.
func (n NotificationConfig) notificationAudioConfigured() bool {
	return n.AudioDir != "" ||
		n.AudioCooldownSeconds != 0 ||
		n.AudioVolume != 0 ||
		n.FallbackFrequency != 0 ||
		n.FallbackDurationMs != 0
}

type GUIConfig struct {
	WindowWidth  int `toml:"window_width"`
	WindowHeight int `toml:"window_height"`
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
			PasswordEnv:         "CUE_EMAIL_PASSWORD", // #nosec G101 -- env var name, not a credential
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
			AudioEnabled:         true,
			BatchProcess:         true,
			AudioDir:             "",
			AudioCooldownSeconds: 2,
			AudioVolume:          100,
			FallbackFrequency:    1000,
			FallbackDurationMs:   200,
		},
		GUI: GUIConfig{
			WindowWidth:  1200,
			WindowHeight: 800,
		},
		Logging: LoggingConfig{
			LogLevel: "info",
		},
	}
}

// generateDefaultTOML creates the default TOML content from the default config.
func generateDefaultTOML() (string, error) {
	cfg := defaultConfig()
	var buf strings.Builder
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(cfg); err != nil {
		return "", fmt.Errorf("generating default TOML: %w", err)
	}
	return buf.String(), nil
}

// Load reads the TOML config at the given path. If the file does not exist,
// it creates the parent directories, writes a default config, and returns it.
func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(filepath.Dir(path), 0750); mkErr != nil {
			return nil, fmt.Errorf("creating config directory: %w", mkErr)
		}

		defaultTOML, genErr := generateDefaultTOML()
		if genErr != nil {
			return nil, fmt.Errorf("generating default config: %w", genErr)
		}

		if wErr := os.WriteFile(path, []byte(defaultTOML), 0600); wErr != nil {
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
	cfg.Notification.AudioDir = expandTilde(cfg.Notification.AudioDir, home)
}

func expandTilde(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// validationRule represents a single validation rule.
type validationRule struct {
	check    func(*Config) bool
	errorMsg string
}

// Validate checks that required configuration fields are set correctly.
func (c *Config) Validate() error {
	rules := []validationRule{
		{func(cfg *Config) bool { return cfg.Database.Path != "" }, "database.path must not be empty"},
		{func(cfg *Config) bool { return cfg.Ollama.Host != "" }, "ollama.host must not be empty"},
		{func(cfg *Config) bool { return cfg.Ollama.Port > 0 }, "ollama.port must be greater than 0"},
		{func(cfg *Config) bool { return cfg.Ollama.InferenceModel != "" }, "ollama.inference_model must not be empty"},
		{func(cfg *Config) bool { return cfg.Ollama.EmbeddingModel != "" }, "ollama.embedding_model must not be empty"},
		{func(cfg *Config) bool { return cfg.Slack.PollIntervalSeconds >= 0 }, "slack.poll_interval_seconds must not be negative"},
		{func(cfg *Config) bool { return cfg.Email.PollIntervalSeconds >= 0 }, "email.poll_interval_seconds must not be negative"},
		{func(cfg *Config) bool {
			if cfg.Notification.AudioDir == "" {
				return true
			}
			_, err := os.Stat(cfg.Notification.AudioDir)
			return err == nil
		}, "notification.audio_dir must be a valid directory"},
		{func(cfg *Config) bool { return cfg.Notification.AudioCooldownSeconds >= 0 }, "notification.audio_cooldown_seconds must not be negative"},
		{func(cfg *Config) bool {
			return cfg.Notification.AudioVolume >= 0 && cfg.Notification.AudioVolume <= 100
		}, "notification.audio_volume must be between 0 and 100"},
		{func(cfg *Config) bool {
			if !cfg.Notification.notificationAudioConfigured() {
				return true
			}
			return cfg.Notification.FallbackFrequency > 0
		}, "notification.fallback_frequency must be greater than 0"},
		{func(cfg *Config) bool {
			if !cfg.Notification.notificationAudioConfigured() {
				return true
			}
			return cfg.Notification.FallbackDurationMs > 0
		}, "notification.fallback_duration_ms must be greater than 0"},
		{func(cfg *Config) bool {
			if cfg.GUI.WindowWidth == 0 && cfg.GUI.WindowHeight == 0 {
				return true
			}
			return cfg.GUI.WindowWidth > 0
		}, "gui.window_width must be greater than 0"},
		{func(cfg *Config) bool {
			if cfg.GUI.WindowWidth == 0 && cfg.GUI.WindowHeight == 0 {
				return true
			}
			return cfg.GUI.WindowHeight > 0
		}, "gui.window_height must be greater than 0"},
	}

	for _, rule := range rules {
		if !rule.check(c) {
			return fmt.Errorf("%s", rule.errorMsg)
		}
	}
	return nil
}
