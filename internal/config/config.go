package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Config represents the application configuration loaded from a TOML file.
type Config struct {
	Database     DatabaseConfig     `toml:"database"`
	Orchestrator OrchestratorConfig `toml:"orchestrator"`
	Ollama       OllamaConfig       `toml:"ollama"`
	Notification NotificationConfig `toml:"notification"`
	GUI          GUIConfig          `toml:"gui"`
	Logging      LoggingConfig      `toml:"logging"`
	Planner      PlannerConfig      `toml:"planner"`
}

type PlannerConfig struct {
	WorkdayStart           string `toml:"workday_start"`
	WorkdayEnd             string `toml:"workday_end"`
	PlanningCutoff         string `toml:"planning_cutoff"`
	PomodoroMinutes        int    `toml:"pomodoro_minutes"`
	ShortBreakMinutes      int    `toml:"short_break_minutes"`
	LongBreakMinutes       int    `toml:"long_break_minutes"`
	LongBreakAfterCycles   int    `toml:"long_break_after_cycles"`
	MeetingMergeGapMinutes int    `toml:"meeting_merge_gap_minutes"`
	LunchWindowStart       string `toml:"lunch_window_start"`
	LunchWindowEnd         string `toml:"lunch_window_end"`
	TimerSound             string `toml:"timer_sound"`
	TimerVolume            int    `toml:"timer_volume"`
}

// isConfigured returns true if any planner field has been explicitly set.
func (p PlannerConfig) isConfigured() bool {
	defaultCfg := PlannerConfig{}
	return p != defaultCfg
}

type DatabaseConfig struct {
	Path string `toml:"path"`
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
// field has been explicitly set, indicating the section was configured.
func (n NotificationConfig) notificationAudioConfigured() bool {
	return n.AudioDir != "" ||
		n.AudioCooldownSeconds != 0 ||
		n.AudioVolume != 0 ||
		n.FallbackFrequency != 0 ||
		n.FallbackDurationMs != 0
}

type GUIConfig struct {
	WindowWidth  int    `toml:"window_width"`
	WindowHeight int    `toml:"window_height"`
	Character    string `toml:"character"`
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
			Character:    "none",
		},
		Logging: LoggingConfig{
			LogLevel: "info",
		},
		Planner: PlannerConfig{
			WorkdayStart:           "09:00",
			WorkdayEnd:             "17:00",
			PlanningCutoff:         "16:00",
			PomodoroMinutes:        25,
			ShortBreakMinutes:      5,
			LongBreakMinutes:       20,
			LongBreakAfterCycles:   4,
			MeetingMergeGapMinutes: 5,
			LunchWindowStart:       "12:00",
			LunchWindowEnd:         "14:00",
			TimerSound:             "",
			TimerVolume:            75,
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
		expandPaths(cfg)
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
	cfg.Planner.TimerSound = expandTilde(cfg.Planner.TimerSound, home)
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

func conditionalRule(condition func(*Config) bool, check func(*Config) bool, msg string) validationRule {
	return validationRule{
		check: func(cfg *Config) bool {
			if !condition(cfg) {
				return true
			}
			return check(cfg)
		},
		errorMsg: msg,
	}
}

// Validate checks that required configuration fields are set correctly.
func (c *Config) Validate() error {
	guiConfigured := func(cfg *Config) bool {
		return cfg.GUI.WindowWidth != 0 || cfg.GUI.WindowHeight != 0
	}

	rules := []validationRule{
		{func(cfg *Config) bool { return cfg.Database.Path != "" }, "database.path must not be empty"},
		{func(cfg *Config) bool { return cfg.Ollama.Host != "" }, "ollama.host must not be empty"},
		{func(cfg *Config) bool { return cfg.Ollama.Port > 0 }, "ollama.port must be greater than 0"},
		{func(cfg *Config) bool { return cfg.Ollama.InferenceModel != "" }, "ollama.inference_model must not be empty"},
		{func(cfg *Config) bool { return cfg.Ollama.EmbeddingModel != "" }, "ollama.embedding_model must not be empty"},
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
		conditionalRule(
			func(cfg *Config) bool { return cfg.Notification.notificationAudioConfigured() },
			func(cfg *Config) bool { return cfg.Notification.FallbackFrequency > 0 },
			"notification.fallback_frequency must be greater than 0"),
		conditionalRule(
			func(cfg *Config) bool { return cfg.Notification.notificationAudioConfigured() },
			func(cfg *Config) bool { return cfg.Notification.FallbackDurationMs > 0 },
			"notification.fallback_duration_ms must be greater than 0"),
		conditionalRule(guiConfigured, func(cfg *Config) bool { return cfg.GUI.WindowWidth > 0 }, "gui.window_width must be greater than 0"),
		conditionalRule(guiConfigured, func(cfg *Config) bool { return cfg.GUI.WindowHeight > 0 }, "gui.window_height must be greater than 0"),
		conditionalRule(
			func(cfg *Config) bool { return cfg.Planner.isConfigured() },
			func(cfg *Config) bool { return cfg.Planner.PomodoroMinutes > 0 },
			"planner.pomodoro_minutes must be greater than 0"),
		conditionalRule(
			func(cfg *Config) bool { return cfg.Planner.isConfigured() },
			func(cfg *Config) bool { return cfg.Planner.ShortBreakMinutes > 0 },
			"planner.short_break_minutes must be greater than 0"),
		conditionalRule(
			func(cfg *Config) bool { return cfg.Planner.isConfigured() },
			func(cfg *Config) bool { return cfg.Planner.LongBreakMinutes > 0 },
			"planner.long_break_minutes must be greater than 0"),
		conditionalRule(
			func(cfg *Config) bool { return cfg.Planner.isConfigured() },
			func(cfg *Config) bool { return cfg.Planner.LongBreakAfterCycles > 0 },
			"planner.long_break_after_cycles must be greater than 0"),
		conditionalRule(
			func(cfg *Config) bool { return cfg.Planner.isConfigured() },
			func(cfg *Config) bool { return cfg.Planner.MeetingMergeGapMinutes > 0 },
			"planner.meeting_merge_gap_minutes must be greater than 0"),
		conditionalRule(
			func(cfg *Config) bool { return cfg.Planner.isConfigured() },
			func(cfg *Config) bool { return isValidTimeOfDay(cfg.Planner.WorkdayStart) },
			"planner.workday_start must be a valid HH:MM time"),
		conditionalRule(
			func(cfg *Config) bool { return cfg.Planner.isConfigured() },
			func(cfg *Config) bool { return isValidTimeOfDay(cfg.Planner.WorkdayEnd) },
			"planner.workday_end must be a valid HH:MM time"),
		conditionalRule(
			func(cfg *Config) bool { return cfg.Planner.isConfigured() },
			func(cfg *Config) bool { return isValidTimeOfDay(cfg.Planner.PlanningCutoff) },
			"planner.planning_cutoff must be a valid HH:MM time"),
		conditionalRule(
			func(cfg *Config) bool { return cfg.Planner.isConfigured() },
			func(cfg *Config) bool { return isValidTimeOfDay(cfg.Planner.LunchWindowStart) },
			"planner.lunch_window_start must be a valid HH:MM time"),
		conditionalRule(
			func(cfg *Config) bool { return cfg.Planner.isConfigured() },
			func(cfg *Config) bool { return isValidTimeOfDay(cfg.Planner.LunchWindowEnd) },
			"planner.lunch_window_end must be a valid HH:MM time"),
		conditionalRule(
			func(cfg *Config) bool { return cfg.Planner.isConfigured() },
			func(cfg *Config) bool {
				s, sErr := parseTimeOfDay(cfg.Planner.WorkdayStart)
				e, eErr := parseTimeOfDay(cfg.Planner.WorkdayEnd)
				if sErr != nil || eErr != nil {
					return true
				}
				return e.After(s)
			},
			"planner.workday_end must be after planner.workday_start"),
		conditionalRule(
			func(cfg *Config) bool { return cfg.Planner.isConfigured() },
			func(cfg *Config) bool {
				s, sErr := parseTimeOfDay(cfg.Planner.LunchWindowStart)
				e, eErr := parseTimeOfDay(cfg.Planner.LunchWindowEnd)
				if sErr != nil || eErr != nil {
					return true
				}
				return e.After(s)
			},
			"planner.lunch_window_end must be after planner.lunch_window_start"),
		conditionalRule(
			func(cfg *Config) bool { return cfg.Planner.isConfigured() },
			func(cfg *Config) bool {
				return cfg.Planner.TimerVolume >= 0 && cfg.Planner.TimerVolume <= 100
			},
			"planner.timer_volume must be between 0 and 100"),
	}

	for _, rule := range rules {
		if !rule.check(c) {
			return fmt.Errorf("%s", rule.errorMsg)
		}
	}
	return nil
}

// isValidTimeOfDay checks if a string is a valid HH:MM time format.
func isValidTimeOfDay(s string) bool {
	_, err := parseTimeOfDay(s)
	return err == nil
}

// parseTimeOfDay parses a "HH:MM" string into a time.Time on the zero date.
func parseTimeOfDay(s string) (time.Time, error) {
	return time.Parse("15:04", s)
}
