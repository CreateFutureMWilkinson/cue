package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
)

type ExampleConfigSuite struct {
	suite.Suite
}

func TestExampleConfig(t *testing.T) {
	suite.Run(t, new(ExampleConfigSuite))
}

// --- ExampleTOML() tests ---

func (s *ExampleConfigSuite) TestExampleTOMLReturnsValidTOML() {
	output := config.ExampleTOML()
	s.NotEmpty(output)

	var parsed map[string]any
	_, err := toml.Decode(output, &parsed)
	s.NoError(err, "ExampleTOML output must be valid TOML")
}

func (s *ExampleConfigSuite) TestExampleTOMLMatchesDefaultDatabasePath() {
	output := config.ExampleTOML()

	var cfg config.Config
	_, err := toml.Decode(output, &cfg)
	s.Require().NoError(err)

	s.Equal("~/.cue/messages.db", cfg.Database.Path)
}

func (s *ExampleConfigSuite) TestExampleTOMLMatchesDefaultOllamaValues() {
	output := config.ExampleTOML()

	var cfg config.Config
	_, err := toml.Decode(output, &cfg)
	s.Require().NoError(err)

	s.Equal("localhost", cfg.Ollama.Host)
	s.Equal(11434, cfg.Ollama.Port)
	s.Equal("neural-chat", cfg.Ollama.InferenceModel)
	s.Equal("nomic-embed-text", cfg.Ollama.EmbeddingModel)
	s.Equal(10, cfg.Ollama.TimeoutSeconds)
}

func (s *ExampleConfigSuite) TestExampleTOMLMatchesDefaultRouterValues() {
	output := config.ExampleTOML()

	var cfg config.Config
	_, err := toml.Decode(output, &cfg)
	s.Require().NoError(err)

	s.Equal(7, cfg.Orchestrator.Router.ImportanceThreshold)
	s.InDelta(0.8, cfg.Orchestrator.Router.ConfidenceThreshold, 0.001)
	s.Equal(100, cfg.Orchestrator.Router.BufferSizePerSource)
	s.Equal(50, cfg.Orchestrator.Router.QueueWarningThreshold)
	s.False(cfg.Orchestrator.Router.CalibrationEnabled)
	s.InDelta(0.75, cfg.Orchestrator.Router.CalibrationSimilarityThreshold, 0.001)
	s.Equal(5, cfg.Orchestrator.Router.CalibrationMaxExamples)
	s.Equal(600, cfg.Orchestrator.PollIntervalSeconds)
	s.Equal(10, cfg.Orchestrator.OllamaCooldownSeconds)
}

func (s *ExampleConfigSuite) TestExampleTOMLMatchesDefaultNotificationValues() {
	output := config.ExampleTOML()

	var cfg config.Config
	_, err := toml.Decode(output, &cfg)
	s.Require().NoError(err)

	s.True(cfg.Notification.AudioEnabled)
	s.Equal(2, cfg.Notification.AudioCooldownSeconds)
	s.Equal(100, cfg.Notification.AudioVolume)
	s.Equal(1000, cfg.Notification.FallbackFrequency)
	s.Equal(200, cfg.Notification.FallbackDurationMs)
}

func (s *ExampleConfigSuite) TestExampleTOMLMatchesDefaultGUIValues() {
	output := config.ExampleTOML()

	var cfg config.Config
	_, err := toml.Decode(output, &cfg)
	s.Require().NoError(err)

	s.Equal(1200, cfg.GUI.WindowWidth)
	s.Equal(800, cfg.GUI.WindowHeight)
	s.Equal("none", cfg.GUI.Character)
	s.Equal("~/.cue/characters", cfg.GUI.CharacterDir)
}

func (s *ExampleConfigSuite) TestExampleTOMLMatchesDefaultLoggingValues() {
	output := config.ExampleTOML()

	var cfg config.Config
	_, err := toml.Decode(output, &cfg)
	s.Require().NoError(err)

	s.Equal("info", cfg.Logging.LogLevel)
	s.Empty(cfg.Logging.LogDir)
}

func (s *ExampleConfigSuite) TestExampleTOMLMatchesDefaultPlannerValues() {
	output := config.ExampleTOML()

	var cfg config.Config
	_, err := toml.Decode(output, &cfg)
	s.Require().NoError(err)

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
	s.Equal(75, cfg.Planner.TimerVolume)
}

func (s *ExampleConfigSuite) TestExampleTOMLContainsHeaderComment() {
	output := config.ExampleTOML()
	s.Contains(output, "Cue Configuration")
}

func (s *ExampleConfigSuite) TestExampleTOMLContainsOllamaModelPullInstructions() {
	output := config.ExampleTOML()
	s.Contains(output, "ollama pull neural-chat")
	s.Contains(output, "ollama pull nomic-embed-text")
}

func (s *ExampleConfigSuite) TestExampleTOMLContainsSectionHeaders() {
	output := config.ExampleTOML()
	s.Contains(output, "[database]")
	s.Contains(output, "[ollama]")
	s.Contains(output, "[notification]")
	s.Contains(output, "[gui]")
	s.Contains(output, "[logging]")
	s.Contains(output, "[planner]")
	s.Contains(output, "[orchestrator]")
}

func (s *ExampleConfigSuite) TestExampleTOMLDoesNotContainSlackSection() {
	output := config.ExampleTOML()
	s.NotContains(output, "[slack]")
}

func (s *ExampleConfigSuite) TestExampleTOMLDoesNotContainEmailSection() {
	output := config.ExampleTOML()
	s.NotContains(output, "[email]")
}

// --- WriteExampleConfig() tests ---

func (s *ExampleConfigSuite) TestWriteExampleConfigCreatesNewFile() {
	dir := s.T().TempDir()
	path := filepath.Join(dir, "config.toml")

	err := config.WriteExampleConfig(path, false)
	s.NoError(err)

	content, readErr := os.ReadFile(path)
	s.NoError(readErr)
	s.NotEmpty(content)
}

func (s *ExampleConfigSuite) TestWriteExampleConfigContentMatchesExampleTOML() {
	dir := s.T().TempDir()
	path := filepath.Join(dir, "config.toml")

	err := config.WriteExampleConfig(path, false)
	s.Require().NoError(err)

	content, readErr := os.ReadFile(path)
	s.Require().NoError(readErr)

	s.Equal(config.ExampleTOML(), string(content))
}

func (s *ExampleConfigSuite) TestWriteExampleConfigErrorsIfFileExistsAndForceIsFalse() {
	dir := s.T().TempDir()
	path := filepath.Join(dir, "config.toml")

	err := os.WriteFile(path, []byte("existing content"), 0600)
	s.Require().NoError(err)

	writeErr := config.WriteExampleConfig(path, false)
	s.Error(writeErr)
	s.True(strings.Contains(writeErr.Error(), "already exists"),
		"error message should contain 'already exists', got: %s", writeErr.Error())

	// Verify original content was not modified.
	content, _ := os.ReadFile(path)
	s.Equal("existing content", string(content))
}

func (s *ExampleConfigSuite) TestWriteExampleConfigOverwritesIfFileExistsAndForceIsTrue() {
	dir := s.T().TempDir()
	path := filepath.Join(dir, "config.toml")

	err := os.WriteFile(path, []byte("old content"), 0600)
	s.Require().NoError(err)

	writeErr := config.WriteExampleConfig(path, true)
	s.NoError(writeErr)

	content, readErr := os.ReadFile(path)
	s.Require().NoError(readErr)
	s.Equal(config.ExampleTOML(), string(content))
}

func (s *ExampleConfigSuite) TestWriteExampleConfigCreatesParentDirectories() {
	dir := s.T().TempDir()
	path := filepath.Join(dir, "sub", "dir", "config.toml")

	err := config.WriteExampleConfig(path, false)
	s.NoError(err)

	content, readErr := os.ReadFile(path)
	s.NoError(readErr)
	s.Equal(config.ExampleTOML(), string(content))
}

// --- Feature 048: BatchProcess removal ---

func (s *ExampleConfigSuite) TestExampleTOMLOmitsBatchProcess() {
	output := config.ExampleTOML()
	s.NotContains(output, "batch_process", "ExampleTOML should not contain batch_process after removal")
}

// --- Feature 097: Server section ---

func (s *ExampleConfigSuite) TestExampleTOMLContainsServerSection() {
	output := config.ExampleTOML()
	s.Contains(output, "[server]", "ExampleTOML should contain a [server] section")
}

func (s *ExampleConfigSuite) TestExampleTOMLServerSectionIsCommentedOut() {
	output := config.ExampleTOML()
	s.Contains(output, "# [server]", "server section should be commented out by default")
	s.Contains(output, "# host =", "server host should be commented out by default")
	s.Contains(output, "# port =", "server port should be commented out by default")
}
