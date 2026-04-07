package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/google/uuid"
	chromem "github.com/rengensheng/chromem-go"
	"github.com/urfave/cli/v3"

	"github.com/CreateFutureMWilkinson/cue/internal/alert"
	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"
	"github.com/CreateFutureMWilkinson/cue/internal/secret"
	"github.com/CreateFutureMWilkinson/cue/internal/service/buffer"
	"github.com/CreateFutureMWilkinson/cue/internal/service/calendar"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
	"github.com/CreateFutureMWilkinson/cue/internal/service/orchestrator"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/CreateFutureMWilkinson/cue/internal/service/validation"
	"github.com/CreateFutureMWilkinson/cue/internal/service/vector"
	"github.com/CreateFutureMWilkinson/cue/internal/service/watcher"
	"github.com/CreateFutureMWilkinson/cue/internal/shutdown"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/fairy"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character/wasmhost"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

const (
	// configRelPath is the path to the config file relative to the user's home directory.
	configRelPath = ".cue/config.toml"

	// eventChannelBuffer is the capacity of the activity event channels.
	eventChannelBuffer = 100
)

func main() {
	app := &cli.Command{
		Name:  "cue",
		Usage: "ADHD-friendly productivity assistant",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return run()
		},
		Commands: []*cli.Command{
			configCommand(),
			uatCommand(),
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatalf("cue: %v", err)
	}
}

func configCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Configuration management",
		Commands: []*cli.Command{
			{
				Name:  "example",
				Usage: "Print an annotated example config.toml",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Write to file instead of stdout",
					},
					&cli.BoolFlag{
						Name:  "force",
						Usage: "Overwrite existing file",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					output := cmd.String("output")
					if output == "" {
						fmt.Print(config.ExampleTOML())
						return nil
					}
					return config.WriteExampleConfig(output, cmd.Bool("force"))
				},
			},
		},
	}
}

func run() error {
	// Load configuration.
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home directory: %w", err)
	}
	cfgPath := filepath.Join(home, configRelPath)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validating config: %w", err)
	}

	// Validate Ollama models are available locally.
	ctx := context.Background()
	ollamaURL := fmt.Sprintf("http://%s:%d", cfg.Ollama.Host, cfg.Ollama.Port)
	if err := decisionengine.ValidateOllamaModels(ctx, ollamaURL, []string{
		cfg.Ollama.InferenceModel,
		cfg.Ollama.EmbeddingModel,
	}); err != nil {
		return fmt.Errorf("ollama model validation: %w", err)
	}

	// Open SQLite database.
	repo, err := sqlite.NewSQLiteMessageRepository(cfg.Database.Path, cfg.Orchestrator.Router.BufferSizePerSource)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}

	// Create encryptor for credential storage.
	keyPath := filepath.Join(home, ".cue", "secret.key")
	enc, err := secret.NewKeyFileEncryptor(keyPath)
	if err != nil {
		return fmt.Errorf("creating encryptor: %w", err)
	}

	// Create service config repository (shares the same DB connection).
	serviceConfigRepo, err := sqlite.NewSQLiteServiceConfigRepository(repo.DB(), enc)
	if err != nil {
		return fmt.Errorf("creating service config repository: %w", err)
	}

	// Create Ollama scorer for LLM-based message importance evaluation.
	ollamaClient, err := decisionengine.NewOllamaClient(
		ollamaURL,
		cfg.Ollama.InferenceModel,
		time.Duration(cfg.Ollama.TimeoutSeconds)*time.Second,
	)
	if err != nil {
		return fmt.Errorf("creating ollama client: %w", err)
	}

	// Derive vector storage path from database path (sibling directory).
	vectorPath := filepath.Join(filepath.Dir(cfg.Database.Path), "vectors")

	// Create Ollama embedding function for vector store.
	ollamaEmbFn := chromem.NewEmbeddingFuncOllama(cfg.Ollama.EmbeddingModel, ollamaURL+"/api")

	// Create chromem-go vector store for persistent embeddings.
	vectorStore, err := vector.NewChromemVectorStore(vectorPath, vector.EmbeddingFunc(ollamaEmbFn))
	if err != nil {
		return fmt.Errorf("creating vector store: %w", err)
	}

	// Create queue repository (shares the same DB connection).
	queueRepo, err := sqlite.NewSQLiteQueueRepository(repo.DB())
	if err != nil {
		return fmt.Errorf("creating queue repository: %w", err)
	}

	// Create routing rule repository and build rules engine.
	ruleRepo, err := sqlite.NewSQLiteRoutingRuleRepository(repo.DB())
	if err != nil {
		return fmt.Errorf("creating routing rule repository: %w", err)
	}
	ruleList, err := ruleRepo.ListRules(ctx)
	if err != nil {
		return fmt.Errorf("loading routing rules: %w", err)
	}
	rulesEngine := decisionengine.NewRulesEngine(ruleList)

	// Create buffer service with vector embedder.
	bufferSvc, err := buffer.NewBufferService(repo, vectorStore)
	if err != nil {
		return fmt.Errorf("creating buffer service: %w", err)
	}

	// Create alert service.
	alertSvc, err := alert.NewAlertService(
		alert.AlertConfig{
			AudioEnabled:         cfg.Notification.AudioEnabled,
			AudioDir:             cfg.Notification.AudioDir,
			AudioCooldownSeconds: cfg.Notification.AudioCooldownSeconds,
			AudioVolume:          cfg.Notification.AudioVolume,
			FallbackFrequency:    cfg.Notification.FallbackFrequency,
			FallbackDurationMs:   cfg.Notification.FallbackDurationMs,
		},
		alert.NewBeeepBeeper(),
		&osFileSystem{},
		alert.NewBeepPlayer(cfg.Notification.AudioDir),
	)
	if err != nil {
		return fmt.Errorf("creating alert service: %w", err)
	}

	// Activity event channel bridges orchestrator -> presenter.
	orchEventCh := make(chan orchestrator.ActivityEvent, eventChannelBuffer)

	// Create orchestrator with zero watchers (populated dynamically from DB).
	orch, err := orchestrator.NewOrchestrator(
		orchestrator.OrchestratorConfig{
			PollIntervalSeconds: cfg.Orchestrator.PollIntervalSeconds,
		},
		rulesEngine,
		queueRepo,
		repo,
		nil,
		orchEventCh,
		alertSvc,
	)
	if err != nil {
		return fmt.Errorf("creating orchestrator: %w", err)
	}

	// Create queue processor for background Ollama scoring.
	queueProcessor, err := orchestrator.NewQueueProcessor(
		queueRepo,
		repo,
		ollamaClient,
		alertSvc,
		orchEventCh,
		float64(cfg.Orchestrator.Router.ImportanceThreshold),
		cfg.Orchestrator.Router.ConfidenceThreshold,
		time.Duration(cfg.Orchestrator.OllamaCooldownSeconds)*time.Second,
	)
	if err != nil {
		return fmt.Errorf("creating queue processor: %w", err)
	}

	// Build watchers from enabled service accounts in the DB.
	buildWatchersFromDB(ctx, serviceConfigRepo, orch)

	// Bridge channel: convert orchestrator events to presenter events (fan-out).
	presenterEventCh := make(chan presenter.ActivityEvent, eventChannelBuffer)
	charPresenterEventCh := make(chan presenter.ActivityEvent, eventChannelBuffer)
	go bridgeEvents(orchEventCh, presenterEventCh, charPresenterEventCh)

	// Create presenters.
	notifPresenter, err := presenter.NewNotificationPresenter(repo, repo)
	if err != nil {
		return fmt.Errorf("creating notification presenter: %w", err)
	}

	activityPresenter, err := presenter.NewActivityPresenter(
		&channelActivitySource{ch: presenterEventCh}, 500,
	)
	if err != nil {
		return fmt.Errorf("creating activity presenter: %w", err)
	}

	feedbackPresenter, err := presenter.NewFeedbackPresenter(bufferSvc)
	if err != nil {
		return fmt.Errorf("creating feedback presenter: %w", err)
	}

	appPresenter, err := presenter.NewAppPresenter(
		notifPresenter, activityPresenter, feedbackPresenter,
	)
	if err != nil {
		return fmt.Errorf("creating app presenter: %w", err)
	}

	// Create settings presenter for runtime volume control.
	settingsPresenter, err := presenter.NewSettingsPresenter(alertSvc, cfg.Notification.AudioVolume, alertSvc, cfg.Notification.AudioVolume)
	if err != nil {
		return fmt.Errorf("creating settings presenter: %w", err)
	}

	// Create watcher factory for runtime account management via Settings UI.
	watcherFactory := func(accountType string, accountID uuid.UUID) error {
		switch accountType {
		case "slack":
			acct, err := serviceConfigRepo.GetSlackAccount(ctx, accountID)
			if err != nil {
				return fmt.Errorf("querying slack account: %w", err)
			}
			slackAPI, err := watcher.NewSlackWebClient(acct.Token)
			if err != nil {
				return fmt.Errorf("creating slack API client: %w", err)
			}
			sw, err := watcher.NewSlackWatcher(slackAPI, watcher.SlackWatcherConfig{WorkspaceID: acct.WorkspaceID})
			if err != nil {
				return fmt.Errorf("creating slack watcher: %w", err)
			}
			orch.AddWatcher("slack:"+acct.WorkspaceID, sw)
		case "email":
			acct, err := serviceConfigRepo.GetEmailAccount(ctx, accountID)
			if err != nil {
				return fmt.Errorf("querying email account: %w", err)
			}
			emailAPI, err := watcher.NewIMAPClient(acct.IMAPHost, acct.IMAPPort, acct.Username, acct.Password, acct.Encryption)
			if err != nil {
				return fmt.Errorf("creating IMAP client: %w", err)
			}
			ew, err := watcher.NewEmailWatcher(emailAPI, watcher.EmailWatcherConfig{Username: acct.Username})
			if err != nil {
				return fmt.Errorf("creating email watcher: %w", err)
			}
			orch.AddWatcher("email:"+acct.Username, ew)
		default:
			return fmt.Errorf("unknown account type: %s", accountType)
		}
		return nil
	}

	// Create service settings presenter with credential validators.
	serviceSettingsPresenter := presenter.NewServiceSettingsPresenter(
		serviceConfigRepo, orch, watcherFactory,
		presenter.WithSlackValidator(validation.NewSlackAPIValidator()),
		presenter.WithEmailValidator(validation.NewIMAPValidator()),
		presenter.WithCalendarValidator(validation.NewICSValidator()),
	)

	// === Phase 2: Day Planner Subsystem ===

	// Create planner repositories (share the same DB path).
	todoRepo, err := sqlite.NewSQLiteTodoRepository(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("creating todo repository: %w", err)
	}

	categoryRepo, err := sqlite.NewSQLiteCategoryRepository(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("creating category repository: %w", err)
	}

	scheduleRepo, err := sqlite.NewSQLiteScheduleRepository(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("creating schedule repository: %w", err)
	}

	// Calendar provider: use noop when no calendar accounts are configured.
	var calProvider calendar.CalendarProvider
	calAccounts, calErr := serviceConfigRepo.ListCalendarAccounts(ctx)
	if calErr != nil || len(calAccounts) == 0 {
		calProvider = calendar.NewNoopCalendarProvider()
	} else {
		// Use first enabled calendar account's ICS URL.
		for _, acct := range calAccounts {
			if acct.Enabled {
				icsProvider, err := calendar.NewICSProvider(
					acct.ICSURL,
					&httpClient{},
					time.Duration(cfg.Ollama.TimeoutSeconds)*time.Second,
				)
				if err == nil {
					calProvider = icsProvider
					break
				}
				log.Printf("warning: failed to create calendar provider for %s: %v", acct.Name, err)
			}
		}
		if calProvider == nil {
			calProvider = calendar.NewNoopCalendarProvider()
		}
	}

	// Create planner clock, estimator, and engine.
	plannerClock := &wallClock{}

	taskEstimator := planner.NewOllamaTaskEstimator(ollamaClient)

	plannerEngine, err := planner.NewPlanner(cfg.Planner, taskEstimator, plannerClock)
	if err != nil {
		return fmt.Errorf("creating planner engine: %w", err)
	}

	// Create planner presenter.
	plannerPresenter, err := presenter.NewPlannerPresenter(
		todoRepo, categoryRepo, calProvider, plannerEngine, taskEstimator, scheduleRepo, plannerClock,
	)
	if err != nil {
		return fmt.Errorf("creating planner presenter: %w", err)
	}

	// Create timer alert service and adapter.
	timerAlertSvc, err := alert.NewTimerAlertService(
		cfg.Notification.AudioDir,
		cfg.Notification.AudioVolume,
		alert.NewBeeepBeeper(),
		&osFileSystem{},
		alert.NewBeepPlayer(cfg.Notification.AudioDir),
	)
	if err != nil {
		return fmt.Errorf("creating timer alert service: %w", err)
	}
	timerAlerter := alert.NewTimerAlerterAdapter(timerAlertSvc)

	// Create timer presenter.
	timerPresenter, err := presenter.NewTimerPresenter(plannerClock, timerAlerter)
	if err != nil {
		return fmt.Errorf("creating timer presenter: %w", err)
	}

	// Create character from config, with fallback to "none".
	character.Register("fairy", func() character.Character {
		f := fairy.NewFairyCharacter()
		// Coalesced refresh: multiple SetPosition/SetBodyColor/SetGlowIntensity calls
		// per frame mark dirty, but ForceRefresh runs at most once per 16ms via fyne.Do.
		var dirty atomic.Bool
		f.SetRefreshHook(func() {
			if dirty.CompareAndSwap(false, true) {
				fyne.Do(func() {
					dirty.Store(false)
					f.ForceRefresh()
				})
			}
		})
		return f
	})
	// Discover and register WASM character plugins from the configured directory.
	if err := wasmhost.RegisterDiscoveredPlugins(cfg.GUI.CharacterDir); err != nil {
		log.Printf("warning: WASM plugin discovery failed: %v", err)
	}

	charName := cfg.GUI.Character
	char, charErr := character.Create(charName)
	if charErr != nil {
		log.Printf("warning: character %q not found, falling back to %q", charName, character.NoneCharacterName)
		char, _ = character.Create(character.NoneCharacterName)
	}

	// Create character presenter, sharing activity events via fan-out.
	charPresenter, err := presenter.NewCharacterPresenter(
		char, &channelActivitySource{ch: charPresenterEventCh}, 5*time.Second,
	)
	if err != nil {
		return fmt.Errorf("creating character presenter: %w", err)
	}

	// Start orchestrator.
	if err := orch.Start(ctx); err != nil {
		return fmt.Errorf("starting orchestrator: %w", err)
	}

	// Start queue processor for background Ollama scoring.
	queueProcessor.Start(ctx)

	// Start app presenter.
	if err := appPresenter.Start(ctx); err != nil {
		return fmt.Errorf("starting app presenter: %w", err)
	}

	// Start character presenter.
	charPresenter.Start(ctx)

	// Create center view router for three-column layout.
	viewRouter := ui.NewCenterViewRouter()

	// Create Fyne app before any UI operations that require the event loop.
	fyneApp := app.New()

	// Trigger startup animation after the event loop starts (fyne.Do requires it).
	fyneApp.Lifecycle().SetOnStarted(func() {
		char.TransitionTo(character.StateStarting)
	})

	mainWindow := ui.NewMainWindow(fyneApp, cfg.GUI, notifPresenter, activityPresenter, feedbackPresenter, appPresenter, settingsPresenter, serviceSettingsPresenter, cfg.Ollama, char.Widget(), viewRouter, plannerPresenter, timerPresenter, plannerPresenter, nil)

	// Wire AppBinder to connect planner presenter callbacks to views.
	if pvRef := mainWindow.PlannerViewRef(); pvRef != nil {
		if wvRef := mainWindow.WizardViewRef(); wvRef != nil {
			appBinder, err := ui.NewAppBinder(plannerPresenter, mainWindow.FocusRail(), wvRef, pvRef, viewRouter)
			if err != nil {
				log.Printf("warning: failed to create app binder: %v", err)
			} else {
				appBinder.SetUIScheduler(fyne.Do)
				appBinder.Bind()
				if err := appBinder.AutoLoad(ctx); err != nil {
					log.Printf("warning: failed to auto-load plan: %v", err)
				}
			}
		}
	}

	// Create and start timer loop (1Hz tick driving countdown timer widget).
	timerLoop, err := ui.NewTimerLoop(timerPresenter, mainWindow.FocusRail().Timer(), mainWindow.FocusRail())
	if err != nil {
		log.Printf("warning: failed to create timer loop: %v", err)
	} else {
		timerLoop.SetUIScheduler(fyne.Do)
		timerLoop.Start(ctx)
		defer timerLoop.Stop()
	}

	// Install signal handler: SIGINT/SIGTERM → fyneApp.Quit().
	sigHandler := shutdown.NewSignalHandler(fyneApp.Quit)
	sigHandler.Start(ctx)

	mainWindow.Run()

	// Graceful shutdown with 5-second timeout guard.
	const shutdownTimeout = 5 * time.Second
	if err := shutdown.RunCleanup(shutdownTimeout, func() error {
		// Play shutdown animation if character supports it.
		type shutdownable interface {
			Shutdown() <-chan struct{}
		}
		if s, ok := char.(shutdownable); ok {
			<-s.Shutdown()
		} else {
			char.Close()
		}
		return nil
	}, func() error {
		charPresenter.Stop()
		return nil
	}, func() error {
		return appPresenter.Shutdown(ctx)
	}, func() error {
		queueProcessor.Stop()
		return nil
	}, func() error {
		return orch.Stop()
	}); err != nil {
		log.Printf("warning: shutdown cleanup: %v", err)
	}

	return nil
}

// bridgeEvents converts orchestrator.ActivityEvent to presenter.ActivityEvent,
// fanning out to all provided output channels.
func bridgeEvents(in <-chan orchestrator.ActivityEvent, outs ...chan<- presenter.ActivityEvent) {
	for ev := range in {
		pe := presenter.ActivityEvent{
			Source:  ev.Source,
			Message: ev.Message,
			IsError: ev.IsError,
		}
		for _, out := range outs {
			out <- pe
		}
	}
	for _, out := range outs {
		close(out)
	}
}

// channelActivitySource wraps a channel as a presenter.ActivitySource.
type channelActivitySource struct {
	ch <-chan presenter.ActivityEvent
}

func (s *channelActivitySource) Events() <-chan presenter.ActivityEvent {
	return s.ch
}

// buildWatchersFromDB queries enabled service accounts and registers watchers
// with the orchestrator. Errors are logged but do not prevent startup.
func buildWatchersFromDB(ctx context.Context, repo repository.ServiceConfigRepository, orch *orchestrator.Orchestrator) {
	slackAccounts, err := repo.ListSlackAccounts(ctx)
	if err != nil {
		log.Printf("warning: failed to query slack accounts: %v", err)
	} else {
		for _, acct := range slackAccounts {
			if !acct.Enabled {
				continue
			}
			slackAPI, err := watcher.NewSlackWebClient(acct.Token)
			if err != nil {
				log.Printf("warning: failed to create slack API client for %s: %v", acct.WorkspaceID, err)
				continue
			}
			sw, err := watcher.NewSlackWatcher(slackAPI, watcher.SlackWatcherConfig{WorkspaceID: acct.WorkspaceID})
			if err != nil {
				log.Printf("warning: failed to create slack watcher for %s: %v", acct.WorkspaceID, err)
				continue
			}
			orch.AddWatcher("slack:"+acct.WorkspaceID, sw)
		}
	}

	emailAccounts, err := repo.ListEmailAccounts(ctx)
	if err != nil {
		log.Printf("warning: failed to query email accounts: %v", err)
	} else {
		for _, acct := range emailAccounts {
			if !acct.Enabled {
				continue
			}
			emailAPI, err := watcher.NewIMAPClient(acct.IMAPHost, acct.IMAPPort, acct.Username, acct.Password, acct.Encryption)
			if err != nil {
				log.Printf("warning: failed to create IMAP client for %s: %v", acct.Username, err)
				continue
			}
			ew, err := watcher.NewEmailWatcher(emailAPI, watcher.EmailWatcherConfig{Username: acct.Username})
			if err != nil {
				log.Printf("warning: failed to create email watcher for %s: %v", acct.Username, err)
				continue
			}
			orch.AddWatcher("email:"+acct.Username, ew)
		}
	}
}

// buildVectorAdvisor creates a VectorScoreAdvisor if vector scoring is enabled
// in the router config, or returns nil if disabled.
func buildVectorAdvisor(cfg config.RouterConfig, querier vector.VectorQuerier, msgQuerier decisionengine.MessageQuerier) (decisionengine.VectorScoreAdvisor, error) {
	if !cfg.VectorEnabled {
		return nil, nil
	}
	return decisionengine.NewVectorScoreAdvisor(querier, msgQuerier, decisionengine.VectorAdvisorConfig{
		SimilarityThreshold: cfg.VectorSimilarityThreshold,
		TopN:                cfg.VectorTopN,
		DampingFactor:       cfg.VectorDampingFactor,
	})
}

// osFileSystem implements alert.FileSystem using the real OS.
type osFileSystem struct{}

func (o *osFileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

// wallClock implements planner.Clock using the real system clock.
type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }

// httpClient implements calendar.HTTPClient using the default http.Client.
type httpClient struct{}

func (httpClient) Do(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req) // #nosec G704 -- URL from user's own calendar config
}
