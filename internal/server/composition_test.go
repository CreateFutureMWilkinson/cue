package server_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/server"
)

// CompositionSuite tests the server composition root.
type CompositionSuite struct {
	suite.Suite
}

func TestComposition(t *testing.T) { suite.Run(t, new(CompositionSuite)) }

func minimalConfig(dbPath string) config.Config {
	return config.Config{
		Database: config.DatabaseConfig{Path: dbPath},
		Orchestrator: config.OrchestratorConfig{
			PollIntervalSeconds: 60,
			Router: config.RouterConfig{
				ImportanceThreshold:   7,
				ConfidenceThreshold:   0.8,
				BufferSizePerSource:   100,
				QueueWarningThreshold: 50,
			},
		},
		Ollama: config.OllamaConfig{
			Host:           "localhost",
			Port:           11434,
			InferenceModel: "test",
			EmbeddingModel: "test",
			TimeoutSeconds: 10,
		},
		Server: config.ServerConfig{
			Host:                "127.0.0.1",
			Port:                0,
			ReadTimeoutSeconds:  30,
			WriteTimeoutSeconds: 30,
		},
	}
}

// TestNewCompositionOpensRepositories verifies that NewComposition returns a
// Composition with all four repository fields populated (non-nil).
func (s *CompositionSuite) TestNewCompositionOpensRepositories() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	cfg := minimalConfig(dbPath)

	comp, err := server.NewComposition(context.Background(), cfg)
	s.Require().NoError(err, "NewComposition should not return an error")
	s.Require().NotNil(comp, "NewComposition should return a non-nil Composition")

	s.NotNil(comp.MessageRepo, "MessageRepo should be populated")
	s.NotNil(comp.QueueRepo, "QueueRepo should be populated")
	s.NotNil(comp.RuleRepo, "RuleRepo should be populated")
	s.NotNil(comp.ServiceConfigRepo, "ServiceConfigRepo should be populated")
}

// TestNewCompositionConstructsServices verifies that NewComposition populates
// OllamaClient, VectorStore, and RulesEngine (B4 services).
func (s *CompositionSuite) TestNewCompositionConstructsServices() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	cfg := minimalConfig(dbPath)

	comp, err := server.NewComposition(context.Background(), cfg)
	s.Require().NoError(err, "NewComposition should not return an error")
	s.Require().NotNil(comp, "NewComposition should return a non-nil Composition")

	s.NotNil(comp.OllamaClient, "OllamaClient should be populated")
	s.NotNil(comp.VectorStore, "VectorStore should be populated")
	s.NotNil(comp.RulesEngine, "RulesEngine should be populated")
}
