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
	"github.com/CreateFutureMWilkinson/cue/internal/service/orchestrator"
	"github.com/CreateFutureMWilkinson/cue/internal/service/vector"
)

// ErrCompositionNotImplemented is returned by stub methods that are not yet implemented.
var ErrCompositionNotImplemented = errors.New("not implemented")

// orchestratorEventBufferSize defines the buffer size for the activity event channel
// shared between orchestrator/queue processor and the hub publisher.
const orchestratorEventBufferSize = 100

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

	// Hub is the central event broadcaster for WebSocket clients
	Hub *Hub
	// Alerter broadcasts alert envelopes to connected WebSocket clients
	Alerter *HubAlerter
	// Orchestrator manages batch polling and routing of messages
	Orchestrator *orchestrator.Orchestrator
	// QueueProcessor processes Ollama scoring queue entries
	QueueProcessor *orchestrator.QueueProcessor
	// EventCh carries activity events from orchestrator to the hub publisher
	EventCh chan orchestrator.ActivityEvent
}

// NewComposition opens all repositories, constructs services (Ollama client,
// vector store, rules engine), and returns a ready-to-use Composition.
//
// All repositories share a single SQLite database connection opened by the
// message repository. Service configuration is encrypted using a keyfile
// stored alongside the database.
func NewComposition(ctx context.Context, cfg config.Config) (*Composition, error) {
	msgRepo, queueRepo, ruleRepo, serviceConfigRepo, err := openRepositories(cfg)
	if err != nil {
		return nil, err
	}

	ollamaClient, vectorStore, rulesEngine, err := constructServices(ctx, cfg, ruleRepo)
	if err != nil {
		return nil, err
	}

	hub, alerter, orch, queueProcessor, eventCh, err := startOrchestration(ctx, cfg, msgRepo, queueRepo, rulesEngine, ollamaClient)
	if err != nil {
		return nil, err
	}

	return &Composition{
		MessageRepo:       msgRepo,
		QueueRepo:         queueRepo,
		RuleRepo:          ruleRepo,
		ServiceConfigRepo: serviceConfigRepo,
		OllamaClient:      ollamaClient,
		VectorStore:       vectorStore,
		RulesEngine:       rulesEngine,
		Hub:               hub,
		Alerter:           alerter,
		Orchestrator:      orch,
		QueueProcessor:    queueProcessor,
		EventCh:           eventCh,
	}, nil
}

// openRepositories opens all SQLite repositories sharing a single database connection.
func openRepositories(cfg config.Config) (
	repository.MessageRepository,
	repository.QueueRepository,
	repository.RoutingRuleRepository,
	repository.ServiceConfigRepository,
	error,
) {
	// Open the primary message repository, which owns the SQLite connection
	msgRepo, err := sqlite.NewSQLiteMessageRepository(cfg.Database.Path, cfg.Orchestrator.Router.BufferSizePerSource)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("opening message repository: %w", err)
	}

	// Open remaining repositories sharing the same database connection
	queueRepo, err := sqlite.NewSQLiteQueueRepository(msgRepo.DB())
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("opening queue repository: %w", err)
	}

	ruleRepo, err := sqlite.NewSQLiteRoutingRuleRepository(msgRepo.DB())
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("opening routing rule repository: %w", err)
	}

	// Initialize encryption for service configuration storage
	keyfilePath := filepath.Join(filepath.Dir(cfg.Database.Path), "secret.key")
	encryptor, err := secret.NewKeyFileEncryptor(keyfilePath)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("opening encryptor: %w", err)
	}

	serviceConfigRepo, err := sqlite.NewSQLiteServiceConfigRepository(msgRepo.DB(), encryptor)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("opening service config repository: %w", err)
	}

	return msgRepo, queueRepo, ruleRepo, serviceConfigRepo, nil
}

// constructServices builds the Ollama client, vector store, and rules engine.
func constructServices(ctx context.Context, cfg config.Config, ruleRepo repository.RoutingRuleRepository) (
	*decisionengine.OllamaClient,
	*vector.ChromemVectorStore,
	*decisionengine.RulesEngine,
	error,
) {
	// Construct Ollama client for LLM scoring
	ollamaURL := fmt.Sprintf("http://%s:%d", cfg.Ollama.Host, cfg.Ollama.Port)
	ollamaClient, err := decisionengine.NewOllamaClient(
		ollamaURL,
		cfg.Ollama.InferenceModel,
		time.Duration(cfg.Ollama.TimeoutSeconds)*time.Second,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating ollama client: %w", err)
	}

	// Construct chromem-go vector store for persistent embeddings
	vectorPath := filepath.Join(filepath.Dir(cfg.Database.Path), "vectors")
	ollamaEmbFn := chromem.NewEmbeddingFuncOllama(cfg.Ollama.EmbeddingModel, ollamaURL+"/api")
	vectorStore, err := vector.NewChromemVectorStore(vectorPath, vector.EmbeddingFunc(ollamaEmbFn), cfg.Ollama.EmbeddingModel)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating vector store: %w", err)
	}

	// Load routing rules and construct rules engine
	ruleList, err := ruleRepo.ListRules(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading routing rules: %w", err)
	}
	rulesEngine := decisionengine.NewRulesEngine(ruleList)

	return ollamaClient, vectorStore, rulesEngine, nil
}

// startOrchestration creates the hub, alerter, orchestrator, and queue processor,
// then starts the orchestrator and queue processor.
func startOrchestration(
	ctx context.Context,
	cfg config.Config,
	msgRepo repository.MessageRepository,
	queueRepo repository.QueueRepository,
	rulesEngine *decisionengine.RulesEngine,
	ollamaClient *decisionengine.OllamaClient,
) (*Hub, *HubAlerter, *orchestrator.Orchestrator, *orchestrator.QueueProcessor, chan orchestrator.ActivityEvent, error) {
	hub := NewHub()
	alerter := NewHubAlerter(hub)

	eventCh := make(chan orchestrator.ActivityEvent, orchestratorEventBufferSize)

	orch, err := orchestrator.NewOrchestrator(
		orchestrator.OrchestratorConfig{
			PollIntervalSeconds:   cfg.Orchestrator.PollIntervalSeconds,
			QueueWarningThreshold: cfg.Orchestrator.Router.QueueWarningThreshold,
		},
		rulesEngine,
		queueRepo,
		msgRepo,
		nil, // watchers wired in B6
		eventCh,
		alerter,
	)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("creating orchestrator: %w", err)
	}

	queueProcessor, err := orchestrator.NewQueueProcessor(
		queueRepo,
		msgRepo,
		ollamaClient,
		alerter,
		eventCh,
		float64(cfg.Orchestrator.Router.ImportanceThreshold),
		cfg.Orchestrator.Router.ConfidenceThreshold,
		time.Duration(cfg.Orchestrator.OllamaCooldownSeconds)*time.Second,
	)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("creating queue processor: %w", err)
	}

	if err := orch.Start(ctx); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("starting orchestrator: %w", err)
	}
	queueProcessor.Start(ctx)

	// TODO(B6): register watchers from DB

	return hub, alerter, orch, queueProcessor, eventCh, nil
}

// Shutdown performs an ordered shutdown of all composition components.
func (c *Composition) Shutdown(ctx context.Context) error {
	return ErrCompositionNotImplemented
}
