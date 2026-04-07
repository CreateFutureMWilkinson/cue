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

// CursorSeedable allows a watcher's poll cursor to be seeded from stored values.
// Watchers that implement this interface will have their cursors set from the
// database before the first poll, so that Poll() only fetches genuinely new messages.
type CursorSeedable interface {
	// SourceInfo returns the source type and account identifier for DB queries.
	SourceInfo() (source string, sourceAccount string)
	// SeedCursor sets the poll cursor for a given channel to the provided value.
	SeedCursor(channel string, cursor string)
}

// Alerter plays audio alerts for notifications.
type Alerter interface {
	PlayNotification(ctx context.Context) error
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
	cfg       OrchestratorConfig
	rules     *decisionengine.RulesEngine
	queueRepo repository.QueueRepository
	repo      repository.MessageRepository
	watchers  map[string]Watcher
	eventCh   chan<- ActivityEvent
	alerter   Alerter

	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.Mutex
	watcherMu sync.RWMutex
	rulesMu   sync.RWMutex
	stopped   bool
}

// NewOrchestrator creates a new Orchestrator, validating all required dependencies.
func NewOrchestrator(cfg OrchestratorConfig, rules *decisionengine.RulesEngine, queueRepo repository.QueueRepository, repo repository.MessageRepository, watchers map[string]Watcher, eventCh chan<- ActivityEvent, alerter Alerter) (*Orchestrator, error) {
	if rules == nil {
		return nil, fmt.Errorf("rules engine is required")
	}
	if queueRepo == nil {
		return nil, fmt.Errorf("queue repository is required")
	}
	if repo == nil {
		return nil, fmt.Errorf("repo is required")
	}
	if watchers == nil {
		watchers = make(map[string]Watcher)
	}

	return &Orchestrator{
		cfg:       cfg,
		rules:     rules,
		queueRepo: queueRepo,
		repo:      repo,
		watchers:  watchers,
		eventCh:   eventCh,
		alerter:   alerter,
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

		// Step 1: Dedup — skip messages already in the DB.
		var newMsgs []*repository.Message
		for _, msg := range msgs {
			exists, err := o.repo.ExistsByMessageID(ctx, msg.MessageID)
			if err != nil {
				o.emitEvent(name, fmt.Sprintf("dedup check error: %v", err), true)
				continue
			}
			if exists {
				continue
			}
			newMsgs = append(newMsgs, msg)
		}

		// Step 2: Rules evaluation.
		var notifiedCount, ignoredCount, queuedCount int
		for _, msg := range newMsgs {
			o.rulesMu.RLock()
			action, matchedRule := o.rules.Evaluate(msg)
			o.rulesMu.RUnlock()

			switch action {
			case "notified":
				msg.ImportanceScore = 8.0
				msg.ConfidenceScore = 1.0
				msg.Status = decisionengine.StatusNotified
				msg.Reasoning = fmt.Sprintf("Rule: %s", matchedRule.Pattern)
				notifiedCount++
			case "ignored":
				msg.ImportanceScore = 0.0
				msg.ConfidenceScore = 1.0
				msg.Status = decisionengine.StatusIgnored
				msg.Reasoning = fmt.Sprintf("Rule: %s", matchedRule.Pattern)
				ignoredCount++
			default: // "queue"
				msg.Status = "Pending"
				msg.ImportanceScore = 0
				msg.ConfidenceScore = 0
				queuedCount++
			}
		}

		// Step 3: Insert all new messages and enqueue pending ones.
		for _, msg := range newMsgs {
			if err := o.repo.Insert(ctx, msg); err != nil {
				o.emitEvent(name, fmt.Sprintf("failed to store %s message: %v", msg.Source, err), true)
				continue
			}
			if msg.Status == "Pending" {
				if err := o.queueRepo.Enqueue(ctx, msg.ID); err != nil {
					o.emitEvent(name, fmt.Sprintf("enqueue error: %v", err), true)
				}
			}
		}

		// Step 4: Emit rules summary.
		o.emitEvent(name, fmt.Sprintf("rules: %d notified, %d ignored, %d queued", notifiedCount, ignoredCount, queuedCount), false)

		// Step 5: Alert if any notified.
		if notifiedCount > 0 && o.alerter != nil {
			if err := o.alerter.PlayNotification(ctx); err != nil {
				o.emitEvent(name, fmt.Sprintf("alert error: %s", err.Error()), false)
			}
		}
	}
}

// Start launches background polling loops. It performs an immediate first poll,
// then polls at the configured interval. Non-blocking.
func (o *Orchestrator) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	o.cancel = cancel

	// Clean up stale queue entries and recover stuck processing entries.
	interval := time.Duration(o.cfg.PollIntervalSeconds) * time.Second
	if err := o.queueRepo.PurgeOlderThan(ctx, time.Now().Add(-interval)); err != nil {
		o.emitEvent("queue", fmt.Sprintf("purge stale entries: %v", err), true)
	}
	if _, err := o.queueRepo.ResetProcessing(ctx); err != nil {
		o.emitEvent("queue", fmt.Sprintf("reset processing: %v", err), true)
	}

	o.wg.Go(func() {
		// Import baseline messages before first poll.
		if err := o.ImportBaseline(ctx); err != nil {
			o.emitEvent("system", fmt.Sprintf("import baseline error: %v", err), true)
		}

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

// ReloadRules rebuilds the rules engine from the given routing rules.
// Invalid regex patterns are logged and skipped by NewRulesEngine.
func (o *Orchestrator) ReloadRules(rules []*repository.RoutingRule) {
	engine := decisionengine.NewRulesEngine(rules)
	o.rulesMu.Lock()
	o.rules = engine
	o.rulesMu.Unlock()
}

// ImportBaseline fetches all available messages from each watcher and inserts
// them as "Imported" without scoring, routing, or alerting.
func (o *Orchestrator) ImportBaseline(ctx context.Context) error {
	o.watcherMu.RLock()
	snapshot := make(map[string]Watcher, len(o.watchers))
	maps.Copy(snapshot, o.watchers)
	o.watcherMu.RUnlock()

	for name, watcher := range snapshot {
		// Seed cursor from DB if the watcher supports it.
		if seedable, ok := watcher.(CursorSeedable); ok {
			source, sourceAccount := seedable.SourceInfo()
			channels, err := o.repo.DistinctChannels(ctx, source, sourceAccount)
			if err != nil {
				o.emitEvent(name, fmt.Sprintf("import cursor lookup error: %v", err), true)
			}
			for _, ch := range channels {
				cursor, err := o.repo.MaxSourceCursor(ctx, source, sourceAccount, ch)
				if err != nil {
					o.emitEvent(name, fmt.Sprintf("import cursor error for %s: %v", ch, err), true)
					continue
				}
				if cursor != "" {
					seedable.SeedCursor(ch, cursor)
				}
			}
		}

		msgs, err := watcher.Poll(ctx)
		if err != nil {
			o.emitEvent(name, fmt.Sprintf("import error: %v", err), true)
			continue
		}

		o.emitEvent(name, fmt.Sprintf("importing %s: %d messages...", name, len(msgs)), false)

		var newCount int
		for _, msg := range msgs {
			exists, err := o.repo.ExistsByMessageID(ctx, msg.MessageID)
			if err != nil {
				o.emitEvent(name, fmt.Sprintf("import dedup error: %v", err), true)
				continue
			}
			if exists {
				continue
			}

			msg.Status = decisionengine.StatusImported
			msg.ImportanceScore = 0
			msg.ConfidenceScore = 0

			if err := o.repo.Insert(ctx, msg); err != nil {
				o.emitEvent(name, fmt.Sprintf("import insert error: %v", err), true)
				continue
			}
			newCount++
		}

		o.emitEvent(name, fmt.Sprintf("import complete: %d new from %s", newCount, name), false)
	}

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
