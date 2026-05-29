package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	chromem "github.com/rengensheng/chromem-go"

	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"
	"github.com/CreateFutureMWilkinson/cue/internal/secret"
	"github.com/CreateFutureMWilkinson/cue/internal/service/buffer"
	"github.com/CreateFutureMWilkinson/cue/internal/service/calendar"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
	"github.com/CreateFutureMWilkinson/cue/internal/service/orchestrator"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/CreateFutureMWilkinson/cue/internal/service/servicemanager"
	todosvc "github.com/CreateFutureMWilkinson/cue/internal/service/todo"
	"github.com/CreateFutureMWilkinson/cue/internal/service/validation"
	"github.com/CreateFutureMWilkinson/cue/internal/service/vector"
	"github.com/CreateFutureMWilkinson/cue/internal/service/watcher"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// dbAccessor is implemented by repository types that expose their underlying database connection.
type dbAccessor interface {
	DB() *sql.DB
}

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

	// HTTP is the headless HTTP/WebSocket server surface. It exposes REST and WebSocket
	// APIs for GUI clients. Started by the caller via HTTP.Start().
	HTTP *Server

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
	// Ticker broadcasts timer events at 0.2Hz while a schedule is active
	Ticker *Ticker

	shutdownOnce sync.Once
	shutdownErr  error
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

	// Open auth token repository on the shared database connection.
	var authTokenRepo *sqlite.SQLiteAuthTokenRepository
	if accessor, ok := msgRepo.(dbAccessor); ok {
		authTokenRepo, err = sqlite.NewSQLiteAuthTokenRepository(accessor.DB())
		if err != nil {
			return nil, fmt.Errorf("opening auth token repository: %w", err)
		}
	}

	ollamaClient, vectorStore, rulesEngine, err := constructServices(ctx, cfg, ruleRepo)
	if err != nil {
		return nil, err
	}

	hub, alerter, orch, queueProcessor, eventCh, err := startOrchestration(ctx, cfg, msgRepo, queueRepo, rulesEngine, ollamaClient, serviceConfigRepo)
	if err != nil {
		return nil, err
	}

	taskRepo, err := sqlite.NewSQLiteTaskRepository(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("opening task repository: %w", err)
	}

	categoryRepo, err := sqlite.NewSQLiteCategoryRepository(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("opening category repository: %w", err)
	}

	taskEstimator := planner.NewOllamaTaskEstimator(ollamaClient)
	todoSvc, err := todosvc.NewService(taskRepo, categoryRepo, taskEstimator)
	if err != nil {
		return nil, fmt.Errorf("creating todo service: %w", err)
	}

	scheduleRepo, err := sqlite.NewSQLiteScheduleRepository(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("opening schedule repository: %w", err)
	}

	calProvider := calendar.CalendarProvider(calendar.NewNoopCalendarProvider())
	plannerClock := wallClock{}
	plannerEngine, err := planner.NewPlanner(cfg.Planner, taskEstimator, plannerClock)
	if err != nil {
		return nil, fmt.Errorf("creating planner engine: %w", err)
	}

	watcherFactory := createWatcherFactory(orch, serviceConfigRepo)
	svcMgr, err := servicemanager.NewServiceManager(
		serviceConfigRepo,
		orch,
		watcherFactory,
		msgRepo,
		servicemanager.WithSlackValidator(validation.NewSlackAPIValidator()),
		servicemanager.WithEmailValidator(validation.NewIMAPValidator()),
		servicemanager.WithCalendarValidator(validation.NewICSValidator()),
	)
	if err != nil {
		return nil, fmt.Errorf("creating service manager: %w", err)
	}

	httpSrv, err := constructHTTPServer(cfg, msgRepo, vectorStore, hub, todoSvc, scheduleRepo, plannerEngine, calProvider, svcMgr, ruleRepo, queueRepo, orch, authTokenRepo)
	if err != nil {
		return nil, err
	}

	// Create and start the timer ticker, then attach it to the server
	// so schedule-mutation handlers can notify it of changes.
	ticker := NewTicker(scheduleRepo, hub, plannerClock, cfg.Planner.WorkdayStart, cfg.Planner.WorkdayEnd)
	ticker.Start(ctx)
	httpSrv.SetTicker(ticker)

	return &Composition{
		HTTP:              httpSrv,
		Ticker:            ticker,
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

// constructHTTPServer builds the HTTP/WebSocket server surface with shared hub.
func constructHTTPServer(cfg config.Config, msgRepo repository.MessageRepository, vectorStore *vector.ChromemVectorStore, hub *Hub, todoSvc *todosvc.Service, scheduleRepo repository.ScheduleRepository, plannerEngine *planner.Planner, calProvider calendar.CalendarProvider, svcMgr *servicemanager.ServiceManager, ruleRepo repository.RoutingRuleRepository, queueRepo repository.QueueRepository, orch *orchestrator.Orchestrator, authTokenRepo *sqlite.SQLiteAuthTokenRepository) (*Server, error) {
	bufSvc, err := buffer.NewBufferService(msgRepo, vectorStore)
	if err != nil {
		return nil, fmt.Errorf("creating buffer service: %w", err)
	}

	rulesPresenter := presenter.NewRulesPresenter(ruleRepo, queueRepo, cfg.Orchestrator.Router.QueueWarningThreshold, presenter.WithReloader(func() {
		ctx := context.Background()
		rules, err := ruleRepo.ListRules(ctx)
		if err != nil {
			slog.Warn("failed to reload routing rules", "error", err)
			return
		}
		orch.ReloadRules(rules)
	}))

	deps := Deps{
		Messages:          msgRepo,
		Buffer:            bufSvc,
		Hub:               hub,
		Todos:             todoSvc,
		Categories:        todoSvc,
		EffectiveEstimate: todosvc.EffectiveEstimate,
		Schedules:         scheduleRepo,
		ScheduleGenerator: plannerEngine,
		Calendar:          calProvider,
		Rules:             rulesPresenter,
		Services:          svcMgr,
	}
	if authTokenRepo != nil {
		deps.AuthTokens = authTokenRepo
		deps.AuthTokenManager = authTokenRepo
	}
	return New(cfg.Server, deps)
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
	serviceConfigRepo repository.ServiceConfigRepository,
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

	registerWatchersFromDB(ctx, orch, serviceConfigRepo)

	// Start the event publisher goroutine
	go publishEventsToHub(eventCh, hub)

	return hub, alerter, orch, queueProcessor, eventCh, nil
}

// publishEventsToHub is a goroutine that ranges over the orchestrator activity
// event channel and publishes each event to the hub for broadcast to WebSocket clients.
//
// This goroutine is the sole consumer of eventCh; the orchestrator and queue processor
// are the producers. The goroutine exits naturally when eventCh is closed (during shutdown).
func publishEventsToHub(eventCh <-chan orchestrator.ActivityEvent, hub *Hub) {
	for ev := range eventCh {
		hub.Publish(ActivityData{
			Source:  ev.Source,
			Message: ev.Message,
			IsError: ev.IsError,
		})
	}
}

// registerWatchersFromDB queries enabled Slack and Email accounts from the
// ServiceConfigRepository and registers watchers with the orchestrator.
// Errors during account listing, API client creation, or watcher construction
// are logged and skipped to allow partial success.
func registerWatchersFromDB(ctx context.Context, orch *orchestrator.Orchestrator, serviceConfigRepo repository.ServiceConfigRepository) {
	slackAccounts, err := serviceConfigRepo.ListSlackAccounts(ctx)
	if err != nil {
		slog.Warn("failed to list slack accounts", "source", "slack", "error", err)
	} else {
		for _, acct := range slackAccounts {
			if !acct.Enabled {
				continue
			}
			slackAPI, err := watcher.NewSlackWebClient(acct.Token)
			if err != nil {
				slog.Warn("failed to create slack API client", "source", "slack", "account_id", acct.WorkspaceID, "error", err)
				continue
			}
			sw, err := watcher.NewSlackWatcher(slackAPI, watcher.SlackWatcherConfig{WorkspaceID: acct.WorkspaceID, WebURL: acct.WebURL})
			if err != nil {
				slog.Warn("failed to create slack watcher", "source", "slack", "account_id", acct.WorkspaceID, "error", err)
				continue
			}
			orch.AddWatcher("slack:"+acct.WorkspaceID, sw)
		}
	}

	emailAccounts, err := serviceConfigRepo.ListEmailAccounts(ctx)
	if err != nil {
		slog.Warn("failed to list email accounts", "source", "email", "error", err)
	} else {
		for _, acct := range emailAccounts {
			if !acct.Enabled {
				continue
			}
			emailAPI, err := watcher.NewIMAPClient(acct.IMAPHost, acct.IMAPPort, acct.Username, acct.Password, acct.Encryption)
			if err != nil {
				slog.Warn("failed to create IMAP client", "source", "email", "account_id", acct.Username, "error", err)
				continue
			}
			ew, err := watcher.NewEmailWatcher(emailAPI, watcher.EmailWatcherConfig{Username: acct.Username, WebURL: acct.WebURL})
			if err != nil {
				slog.Warn("failed to create email watcher", "source", "email", "account_id", acct.Username, "error", err)
				continue
			}
			orch.AddWatcher("email:"+acct.Username, ew)
		}
	}
}

// createWatcherFactory returns a WatcherFactory closure that creates and registers
// watchers with the orchestrator for the given account type and ID.
func createWatcherFactory(orch *orchestrator.Orchestrator, repo repository.ServiceConfigRepository) servicemanager.WatcherFactory {
	return func(accountType string, accountID uuid.UUID) error {
		ctx := context.Background()
		switch accountType {
		case "slack":
			acct, err := repo.GetSlackAccount(ctx, accountID)
			if err != nil {
				return fmt.Errorf("getting slack account: %w", err)
			}
			slackAPI, err := watcher.NewSlackWebClient(acct.Token)
			if err != nil {
				return fmt.Errorf("creating slack API client: %w", err)
			}
			sw, err := watcher.NewSlackWatcher(slackAPI, watcher.SlackWatcherConfig{WorkspaceID: acct.WorkspaceID, WebURL: acct.WebURL})
			if err != nil {
				return fmt.Errorf("creating slack watcher: %w", err)
			}
			orch.AddWatcher("slack:"+acct.WorkspaceID, sw)
		case "email":
			acct, err := repo.GetEmailAccount(ctx, accountID)
			if err != nil {
				return fmt.Errorf("getting email account: %w", err)
			}
			emailAPI, err := watcher.NewIMAPClient(acct.IMAPHost, acct.IMAPPort, acct.Username, acct.Password, acct.Encryption)
			if err != nil {
				return fmt.Errorf("creating IMAP client: %w", err)
			}
			ew, err := watcher.NewEmailWatcher(emailAPI, watcher.EmailWatcherConfig{Username: acct.Username, WebURL: acct.WebURL})
			if err != nil {
				return fmt.Errorf("creating email watcher: %w", err)
			}
			orch.AddWatcher("email:"+acct.Username, ew)
		default:
			return fmt.Errorf("unknown account type: %s", accountType)
		}
		return nil
	}
}

// Shutdown performs an ordered shutdown of all composition components:
// 1. Stop orchestrator (waits for in-flight polls)
// 2. Stop queue processor
// 3. Close event channel (stops hub publisher goroutine)
// 4. Close shared database connection
//
// Shutdown is idempotent and logs progress. The first call executes the sequence;
// subsequent calls return the cached result immediately.
func (c *Composition) Shutdown(ctx context.Context) error {
	c.shutdownOnce.Do(func() {
		slog.Info("shutdown initiated")

		slog.Info("stopping orchestrator", "note", "waiting for in-flight polls to complete")
		if err := c.Orchestrator.Stop(); err != nil {
			slog.Error("failed to stop orchestrator", "error", err)
		}

		slog.Info("stopping queue processor")
		c.QueueProcessor.Stop()

		if c.Ticker != nil {
			slog.Info("stopping ticker")
			c.Ticker.Stop()
		}

		slog.Info("closing event channel")
		close(c.EventCh)

		slog.Info("shutting down HTTP server")
		if err := c.HTTP.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("failed to shutdown HTTP", "error", err)
			c.shutdownErr = fmt.Errorf("shutting down HTTP server: %w", err)
		}

		slog.Info("closing database")
		if accessor, ok := c.MessageRepo.(dbAccessor); ok {
			if err := accessor.DB().Close(); err != nil {
				slog.Error("failed to close database", "error", err)
				c.shutdownErr = fmt.Errorf("closing database: %w", err)
				return
			}
		}

		slog.Info("shutdown complete")
	})
	return c.shutdownErr
}

// wallClock implements planner.Clock using the real system clock.
type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }
