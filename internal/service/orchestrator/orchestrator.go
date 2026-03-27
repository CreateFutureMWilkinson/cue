package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// Watcher polls an external source for new messages.
type Watcher interface {
	Poll(ctx context.Context) ([]*repository.Message, error)
}

// BatchRouter routes a batch of messages, assigning importance/confidence/status.
type BatchRouter interface {
	RouteBatch(ctx context.Context, msgs []*repository.Message) ([]*repository.Message, error)
}

// ActivityEvent represents a system event for the activity log.
type ActivityEvent struct {
	Source  string
	Message string
	IsError bool
}

// OrchestratorConfig holds configuration for the orchestrator.
type OrchestratorConfig struct {
	PollIntervalSeconds int
}

// Orchestrator coordinates polling, routing, and storing of messages.
type Orchestrator struct {
	cfg      OrchestratorConfig
	router   BatchRouter
	repo     repository.MessageRepository
	watchers map[string]Watcher
	eventCh  chan<- ActivityEvent

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex
	stopped bool
}

// NewOrchestrator creates a new Orchestrator, validating all required dependencies.
func NewOrchestrator(cfg OrchestratorConfig, router BatchRouter, repo repository.MessageRepository, watchers map[string]Watcher, eventCh chan<- ActivityEvent) (*Orchestrator, error) {
	if router == nil {
		return nil, fmt.Errorf("router is required")
	}
	if repo == nil {
		return nil, fmt.Errorf("repo is required")
	}
	if len(watchers) == 0 {
		return nil, fmt.Errorf("watchers must not be empty")
	}

	return &Orchestrator{
		cfg:      cfg,
		router:   router,
		repo:     repo,
		watchers: watchers,
		eventCh:  eventCh,
	}, nil
}

// PollOnce executes a single poll cycle across all watchers.
func (o *Orchestrator) PollOnce(ctx context.Context) {
	for name, watcher := range o.watchers {
		msgs, err := watcher.Poll(ctx)
		if err != nil {
			o.eventCh <- ActivityEvent{
				Source:  name,
				Message: fmt.Sprintf("poll error: %s", err.Error()),
				IsError: true,
			}
			continue
		}

		o.eventCh <- ActivityEvent{
			Source:  name,
			Message: fmt.Sprintf("fetched %d messages", len(msgs)),
			IsError: false,
		}

		routed, err := o.router.RouteBatch(ctx, msgs)
		if err != nil {
			o.eventCh <- ActivityEvent{
				Source:  name,
				Message: fmt.Sprintf("routing error: %s", err.Error()),
				IsError: true,
			}
			continue
		}

		var notified, buffered, ignored int
		for _, msg := range routed {
			switch msg.Status {
			case "Notified":
				notified++
			case "Buffered":
				buffered++
			case "Ignored":
				ignored++
			}

			if err := o.repo.Insert(ctx, msg); err != nil {
				continue
			}
		}

		o.eventCh <- ActivityEvent{
			Source:  name,
			Message: fmt.Sprintf("Routed %d NOTIFIED, %d BUFFERED, %d IGNORED", notified, buffered, ignored),
			IsError: false,
		}
	}
}

// Start launches background polling loops. It performs an immediate first poll,
// then polls at the configured interval. Non-blocking.
func (o *Orchestrator) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	o.cancel = cancel

	o.wg.Add(1)
	go func() {
		defer o.wg.Done()

		// Immediate first poll
		o.PollOnce(ctx)

		interval := time.Duration(o.cfg.PollIntervalSeconds) * time.Second
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				o.PollOnce(ctx)
			}
		}
	}()

	return nil
}

// Stop gracefully shuts down the orchestrator. It is idempotent.
func (o *Orchestrator) Stop() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.stopped {
		return nil
	}
	o.stopped = true

	if o.cancel != nil {
		o.cancel()
	}
	o.wg.Wait()
	return nil
}
