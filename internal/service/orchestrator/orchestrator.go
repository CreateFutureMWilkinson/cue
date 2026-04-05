package orchestrator

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"sync"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
)

// WatcherManager allows dynamic addition and removal of watchers at runtime.
type WatcherManager interface {
	AddWatcher(name string, w Watcher)
	RemoveWatcher(name string)
	ListWatcherNames() []string
}

// Watcher polls an external source for new messages.
type Watcher interface {
	Poll(ctx context.Context) ([]*repository.Message, error)
}

// Alerter plays audio alerts for notifications.
type Alerter interface {
	PlayNotification(ctx context.Context) error
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
	alerter  Alerter

	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.Mutex
	watcherMu sync.RWMutex
	stopped   bool
}

// NewOrchestrator creates a new Orchestrator, validating all required dependencies.
func NewOrchestrator(cfg OrchestratorConfig, router BatchRouter, repo repository.MessageRepository, watchers map[string]Watcher, eventCh chan<- ActivityEvent, alerter Alerter) (*Orchestrator, error) {
	if router == nil {
		return nil, fmt.Errorf("router is required")
	}
	if repo == nil {
		return nil, fmt.Errorf("repo is required")
	}
	if watchers == nil {
		watchers = make(map[string]Watcher)
	}

	return &Orchestrator{
		cfg:      cfg,
		router:   router,
		repo:     repo,
		watchers: watchers,
		eventCh:  eventCh,
		alerter:  alerter,
	}, nil
}

// emitEvent sends an activity event to the event channel.
// If the channel is full, the event is dropped to prevent blocking.
func (o *Orchestrator) emitEvent(source, message string, isError bool) {
	select {
	case o.eventCh <- ActivityEvent{
		Source:  source,
		Message: message,
		IsError: isError,
	}:
	default:
	}
}

// AddWatcher registers a named watcher. Replaces if duplicate name.
func (o *Orchestrator) AddWatcher(name string, w Watcher) {
	o.watcherMu.Lock()
	defer o.watcherMu.Unlock()
	o.watchers[name] = w
}

// RemoveWatcher removes a watcher by name. No-op if unknown.
func (o *Orchestrator) RemoveWatcher(name string) {
	o.watcherMu.Lock()
	defer o.watcherMu.Unlock()
	delete(o.watchers, name)
}

// ListWatcherNames returns a sorted slice of registered watcher names.
func (o *Orchestrator) ListWatcherNames() []string {
	o.watcherMu.RLock()
	defer o.watcherMu.RUnlock()
	names := make([]string, 0, len(o.watchers))
	for name := range o.watchers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// PollOnce executes a single poll cycle across all watchers.
func (o *Orchestrator) PollOnce(ctx context.Context) {
	// Snapshot watchers under read lock, then release before doing work.
	o.watcherMu.RLock()
	if len(o.watchers) == 0 {
		o.watcherMu.RUnlock()
		o.emitEvent("system", "No watchers configured", false)
		return
	}
	snapshot := make(map[string]Watcher, len(o.watchers))
	maps.Copy(snapshot, o.watchers)
	o.watcherMu.RUnlock()

	for name, watcher := range snapshot {
		msgs, err := watcher.Poll(ctx)
		if err != nil {
			o.emitEvent(name, fmt.Sprintf("poll error: %s", err.Error()), true)
			continue
		}

		o.emitEvent(name, fmt.Sprintf("fetched %d messages", len(msgs)), false)

		routed, err := o.router.RouteBatch(ctx, msgs)
		if err != nil {
			o.emitEvent(name, fmt.Sprintf("routing error: %s", err.Error()), true)
			continue
		}

		notified, buffered, ignored := countByStatus(routed)
		for _, msg := range routed {
			if err := o.repo.Insert(ctx, msg); err != nil {
				continue
			}
		}

		o.emitEvent(name, fmt.Sprintf("Routed %d NOTIFIED, %d BUFFERED, %d IGNORED", notified, buffered, ignored), false)

		if notified > 0 && o.alerter != nil {
			if err := o.alerter.PlayNotification(ctx); err != nil {
				o.emitEvent(name, fmt.Sprintf("alert error: %s", err.Error()), false)
			}
		}
	}
}

// countByStatus tallies routed messages by their status.
func countByStatus(msgs []*repository.Message) (notified, buffered, ignored int) {
	for _, msg := range msgs {
		switch msg.Status {
		case decisionengine.StatusNotified:
			notified++
		case decisionengine.StatusBuffered:
			buffered++
		case decisionengine.StatusIgnored:
			ignored++
		}
	}
	return
}

// Start launches background polling loops. It performs an immediate first poll,
// then polls at the configured interval. Non-blocking.
func (o *Orchestrator) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	o.cancel = cancel

	o.wg.Go(func() {
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
	})

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
