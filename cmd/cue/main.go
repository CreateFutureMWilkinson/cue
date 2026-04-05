package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/CreateFutureMWilkinson/cue/internal/alert"
	"github.com/CreateFutureMWilkinson/cue/internal/config"
	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"
	"github.com/CreateFutureMWilkinson/cue/internal/service/buffer"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
	"github.com/CreateFutureMWilkinson/cue/internal/service/orchestrator"
	"github.com/CreateFutureMWilkinson/cue/internal/service/watcher"
	"github.com/CreateFutureMWilkinson/cue/internal/ui"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

const (
	// configRelPath is the path to the config file relative to the user's home directory.
	configRelPath = ".cue/config.toml"

	// eventChannelBuffer is the capacity of the activity event channels.
	eventChannelBuffer = 100
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("cue: %v", err)
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

	// Open SQLite database.
	repo, err := sqlite.NewSQLiteMessageRepository(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}

	// Create router with placeholder scorer.
	router, err := decisionengine.NewRouter(
		&placeholderScorer{},
		[]string{"user"},
		decisionengine.RouterConfig{
			ImportanceThreshold: cfg.Orchestrator.Router.ImportanceThreshold,
			ConfidenceThreshold: cfg.Orchestrator.Router.ConfidenceThreshold,
		},
	)
	if err != nil {
		return fmt.Errorf("creating router: %w", err)
	}

	// Create watchers with placeholder API clients.
	slackWatcher, err := watcher.NewSlackWatcher(&placeholderSlackAPI{}, cfg.Slack)
	if err != nil {
		return fmt.Errorf("creating slack watcher: %w", err)
	}
	emailWatcher, err := watcher.NewEmailWatcher(&placeholderEmailAPI{}, cfg.Email)
	if err != nil {
		return fmt.Errorf("creating email watcher: %w", err)
	}
	watchers := map[string]orchestrator.Watcher{
		"slack": slackWatcher,
		"email": emailWatcher,
	}

	// Create buffer service.
	bufferSvc, err := buffer.NewBufferService(repo, nil)
	if err != nil {
		return fmt.Errorf("creating buffer service: %w", err)
	}

	// Create alert service.
	alertSvc, err := alert.NewAlertService(
		alert.AlertConfig{AudioEnabled: cfg.Notification.AudioEnabled},
		alert.NewBeeepBeeper(),
	)
	if err != nil {
		return fmt.Errorf("creating alert service: %w", err)
	}

	// Activity event channel bridges orchestrator -> presenter.
	orchEventCh := make(chan orchestrator.ActivityEvent, eventChannelBuffer)

	// Create orchestrator.
	orch, err := orchestrator.NewOrchestrator(
		orchestrator.OrchestratorConfig{
			PollIntervalSeconds: cfg.Slack.PollIntervalSeconds,
		},
		router,
		repo,
		watchers,
		orchEventCh,
		alertSvc,
	)
	if err != nil {
		return fmt.Errorf("creating orchestrator: %w", err)
	}

	// Bridge channel: convert orchestrator events to presenter events.
	presenterEventCh := make(chan presenter.ActivityEvent, eventChannelBuffer)
	go bridgeEvents(orchEventCh, presenterEventCh)

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
		notifPresenter, activityPresenter, feedbackPresenter, alertSvc,
	)
	if err != nil {
		return fmt.Errorf("creating app presenter: %w", err)
	}

	// Start orchestrator.
	ctx := context.Background()
	if err := orch.Start(ctx); err != nil {
		return fmt.Errorf("starting orchestrator: %w", err)
	}

	// Start app presenter.
	if err := appPresenter.Start(ctx); err != nil {
		return fmt.Errorf("starting app presenter: %w", err)
	}

	// Create and run the Fyne window (blocks until quit).
	mainWindow := ui.NewMainWindow(cfg.GUI, notifPresenter, activityPresenter, feedbackPresenter, appPresenter)
	mainWindow.Run()

	// Graceful shutdown.
	_ = appPresenter.Shutdown(ctx)
	_ = orch.Stop()

	return nil
}

// bridgeEvents converts orchestrator.ActivityEvent to presenter.ActivityEvent.
func bridgeEvents(in <-chan orchestrator.ActivityEvent, out chan<- presenter.ActivityEvent) {
	for ev := range in {
		out <- presenter.ActivityEvent{
			Source:  ev.Source,
			Message: ev.Message,
			IsError: ev.IsError,
		}
	}
	close(out)
}

// channelActivitySource wraps a channel as a presenter.ActivitySource.
type channelActivitySource struct {
	ch <-chan presenter.ActivityEvent
}

func (s *channelActivitySource) Events() <-chan presenter.ActivityEvent {
	return s.ch
}

// --- Placeholder implementations for APIs not yet built ---

type placeholderScorer struct{}

func (p *placeholderScorer) Score(_ context.Context, msg *repository.Message) (*decisionengine.ScorerResult, error) {
	return &decisionengine.ScorerResult{
		ImportanceScore: 5.0,
		ConfidenceScore: 0.5,
		Reasoning:       "placeholder scorer",
	}, nil
}

type placeholderSlackAPI struct{}

func (p *placeholderSlackAPI) GetUserChannels(_ context.Context) ([]watcher.SlackChannel, error) {
	return nil, nil
}

func (p *placeholderSlackAPI) GetChannelMessages(_ context.Context, _ string, _ string) ([]watcher.SlackMessage, error) {
	return nil, nil
}

func (p *placeholderSlackAPI) GetThreadReplies(_ context.Context, _ string, _ string) ([]watcher.SlackMessage, error) {
	return nil, nil
}

type placeholderEmailAPI struct{}

func (p *placeholderEmailAPI) FetchNewMessages(_ context.Context, _ uint32) ([]watcher.EmailMessage, error) {
	return nil, nil
}
