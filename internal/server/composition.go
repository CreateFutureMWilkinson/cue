package server

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	chromem "github.com/rengensheng/chromem-go"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"
	"github.com/CreateFutureMWilkinson/cue/internal/secret"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
	"github.com/CreateFutureMWilkinson/cue/internal/service/vector"
)

// ErrCompositionNotImplemented is returned by stub methods that are not yet implemented.
var ErrCompositionNotImplemented = errors.New("not implemented")

// Composition holds all long-lived components wired into cue-server.
// All repositories share a single SQLite database connection for consistency.
type Composition struct {
	// MessageRepo stores messages received from Slack and Email sources
	MessageRepo repository.MessageRepository
	// QueueRepo stores the FIFO routing queue for batch processing
	QueueRepo repository.QueueRepository
	// RuleRepo stores deterministic routing rules with regex patterns
	RuleRepo repository.RoutingRuleRepository
	// ServiceConfigRepo stores encrypted service credentials and config
	ServiceConfigRepo repository.ServiceConfigRepository

	// OllamaClient connects to the local Ollama instance for LLM scoring
	OllamaClient *decisionengine.OllamaClient
	// VectorStore provides chromem-go backed vector embeddings
	VectorStore *vector.ChromemVectorStore
	// RulesEngine holds compiled deterministic routing rules
	RulesEngine *decisionengine.RulesEngine
}

// NewComposition opens all repositories, constructs services, wires the
// orchestrator, and returns a ready-to-use Composition.
//
// All repositories share a single SQLite database connection opened by the
// message repository. Service configuration is encrypted using a keyfile
// stored alongside the database.
func NewComposition(ctx context.Context, cfg config.Config) (*Composition, error) {
	// Open the primary message repository, which owns the SQLite connection
	msgRepo, err := sqlite.NewSQLiteMessageRepository(cfg.Database.Path, cfg.Orchestrator.Router.BufferSizePerSource)
	if err != nil {
		return nil, fmt.Errorf("opening message repository: %w", err)
	}

	// Open remaining repositories sharing the same database connection
	queueRepo, err := sqlite.NewSQLiteQueueRepository(msgRepo.DB())
	if err != nil {
		return nil, fmt.Errorf("opening queue repository: %w", err)
	}

	ruleRepo, err := sqlite.NewSQLiteRoutingRuleRepository(msgRepo.DB())
	if err != nil {
		return nil, fmt.Errorf("opening routing rule repository: %w", err)
	}

	// Initialize encryption for service configuration storage
	keyfilePath := filepath.Join(filepath.Dir(cfg.Database.Path), "secret.key")
	encryptor, err := secret.NewKeyFileEncryptor(keyfilePath)
	if err != nil {
		return nil, fmt.Errorf("opening encryptor: %w", err)
	}

	serviceConfigRepo, err := sqlite.NewSQLiteServiceConfigRepository(msgRepo.DB(), encryptor)
	if err != nil {
		return nil, fmt.Errorf("opening service config repository: %w", err)
	}

	// Construct Ollama client for LLM scoring
	ollamaURL := fmt.Sprintf("http://%s:%d", cfg.Ollama.Host, cfg.Ollama.Port)
	ollamaClient, err := decisionengine.NewOllamaClient(
		ollamaURL,
		cfg.Ollama.InferenceModel,
		time.Duration(cfg.Ollama.TimeoutSeconds)*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("creating ollama client: %w", err)
	}

	// Construct chromem-go vector store for persistent embeddings
	vectorPath := filepath.Join(filepath.Dir(cfg.Database.Path), "vectors")
	ollamaEmbFn := chromem.NewEmbeddingFuncOllama(cfg.Ollama.EmbeddingModel, ollamaURL+"/api")
	vectorStore, err := vector.NewChromemVectorStore(vectorPath, vector.EmbeddingFunc(ollamaEmbFn), cfg.Ollama.EmbeddingModel)
	if err != nil {
		return nil, fmt.Errorf("creating vector store: %w", err)
	}

	// Load routing rules and construct rules engine
	ruleList, err := ruleRepo.ListRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading routing rules: %w", err)
	}
	rulesEngine := decisionengine.NewRulesEngine(ruleList)

	return &Composition{
		MessageRepo:       msgRepo,
		QueueRepo:         queueRepo,
		RuleRepo:          ruleRepo,
		ServiceConfigRepo: serviceConfigRepo,
		OllamaClient:      ollamaClient,
		VectorStore:       vectorStore,
		RulesEngine:       rulesEngine,
	}, nil
}

// Shutdown performs an ordered shutdown of all composition components.
func (c *Composition) Shutdown(ctx context.Context) error {
	return ErrCompositionNotImplemented
}
