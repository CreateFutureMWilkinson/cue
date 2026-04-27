package server_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"
	"github.com/CreateFutureMWilkinson/cue/internal/secret"
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

// TestNewCompositionStartsOrchestrator verifies that NewComposition populates
// Hub, Alerter, Orchestrator, QueueProcessor, and EventCh (B5 orchestration wiring).
func (s *CompositionSuite) TestNewCompositionStartsOrchestrator() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	cfg := minimalConfig(dbPath)

	comp, err := server.NewComposition(context.Background(), cfg)
	s.Require().NoError(err, "NewComposition should not return an error")
	s.Require().NotNil(comp, "NewComposition should return a non-nil Composition")

	s.NotNil(comp.Hub, "Hub should be populated")
	s.NotNil(comp.Alerter, "Alerter should be populated")
	s.NotNil(comp.Orchestrator, "Orchestrator should be populated")
	s.NotNil(comp.QueueProcessor, "QueueProcessor should be populated")
	s.NotNil(comp.EventCh, "EventCh should be populated")
}

// TestNewCompositionBuildsWatchersFromDB verifies that NewComposition reads
// enabled Slack and Email accounts from the ServiceConfigRepository and
// registers corresponding watchers on the orchestrator.
func (s *CompositionSuite) TestNewCompositionBuildsWatchersFromDB() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	cfg := minimalConfig(dbPath)

	// Seed the DB directly via a raw SQLiteMessageRepository + ServiceConfigRepo,
	// bypassing Composition so we avoid leaking orchestrator goroutines twice.
	msgRepo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err, "opening message repo for seeding")

	keyfilePath := filepath.Join(tmpDir, "secret.key")
	encryptor, err := secret.NewKeyFileEncryptor(keyfilePath)
	s.Require().NoError(err, "creating encryptor for seeding")

	serviceConfigRepo, err := sqlite.NewSQLiteServiceConfigRepository(msgRepo.DB(), encryptor)
	s.Require().NoError(err, "opening service config repo for seeding")

	ctx := context.Background()

	err = serviceConfigRepo.UpsertSlackAccount(ctx, &repository.SlackAccount{
		ID:          uuid.New(),
		Enabled:     true,
		Token:       "xoxb-test",
		WorkspaceID: "T999",
	})
	s.Require().NoError(err, "upserting slack account")

	err = serviceConfigRepo.UpsertEmailAccount(ctx, &repository.EmailAccount{
		ID:         uuid.New(),
		Enabled:    true,
		IMAPHost:   "imap.example.com",
		IMAPPort:   993,
		Username:   "test@example.com",
		Password:   "pw",
		Encryption: "ssl",
	})
	s.Require().NoError(err, "upserting email account")

	// Close the seeding DB connection so the Composition can open its own.
	err = msgRepo.DB().Close()
	s.Require().NoError(err, "closing seeded message repo")

	// Now construct a Composition against the same DB — it should read
	// the seeded accounts and register watchers.
	comp, err := server.NewComposition(ctx, cfg)
	s.Require().NoError(err, "NewComposition should not return an error")
	s.Require().NotNil(comp, "NewComposition should return a non-nil Composition")

	names := comp.Orchestrator.ListWatcherNames()
	s.Require().Len(names, 2, "expected exactly 2 watchers registered")
	s.Contains(names, "slack:T999", "expected slack watcher registered with workspace ID")
	s.Contains(names, "email:test@example.com", "expected email watcher registered with username")
}
