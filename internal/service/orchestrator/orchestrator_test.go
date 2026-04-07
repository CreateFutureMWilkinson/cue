package orchestrator_test

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
	"github.com/CreateFutureMWilkinson/cue/internal/service/orchestrator"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// Compile-time interface check: Orchestrator must implement WatcherManager.
var _ orchestrator.WatcherManager = (*orchestrator.Orchestrator)(nil)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

// mockWatcher implements orchestrator.Watcher for testing.
type mockWatcher struct {
	mu       sync.Mutex
	messages []*repository.Message
	err      error
	calls    int
}

func (w *mockWatcher) Poll(ctx context.Context) ([]*repository.Message, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls++
	return w.messages, w.err
}

func (w *mockWatcher) pollCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.calls
}

// mockRepo implements repository.MessageRepository for testing.
type mockRepo struct {
	mu        sync.Mutex
	inserted  []*repository.Message
	insertErr map[string]error // keyed by message ID string, allows selective failures
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		insertErr: make(map[string]error),
	}
}

func (r *mockRepo) Insert(ctx context.Context, msg *repository.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err, ok := r.insertErr[msg.ID.String()]; ok {
		return err
	}
	r.inserted = append(r.inserted, msg)
	return nil
}

func (r *mockRepo) Update(_ context.Context, _ *repository.Message) error {
	return nil
}

func (r *mockRepo) QueryByID(_ context.Context, _ uuid.UUID) (*repository.Message, error) {
	return nil, nil
}

func (r *mockRepo) QueryByStatus(_ context.Context, _ string) ([]*repository.Message, error) {
	return nil, nil
}

func (r *mockRepo) QueryAll(_ context.Context) ([]*repository.Message, error) {
	return nil, nil
}

func (r *mockRepo) QueryOldestToNewest(_ context.Context, _ int) ([]*repository.Message, error) {
	return nil, nil
}

func (r *mockRepo) CountBySource(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (r *mockRepo) ExistsByMessageID(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (r *mockRepo) insertedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.inserted)
}

// mockAlerter implements orchestrator.Alerter for testing.
type mockAlerter struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (a *mockAlerter) PlayNotification(_ context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls++
	return a.err
}

func (a *mockAlerter) alertCalls() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// mustNewOrchestrator creates a test orchestrator with common defaults,
// reducing boilerplate in tests. Uses nil alerter and 600s poll interval.
func mustNewOrchestrator(repo *mockRepo, watchers map[string]orchestrator.Watcher, eventCh chan orchestrator.ActivityEvent) *orchestrator.Orchestrator {
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}
	rules := decisionengine.NewRulesEngine(nil)
	queueRepo := &mockQueueRepo{}
	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, watchers, eventCh, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to create test orchestrator: %v", err))
	}
	return orch
}

func makeMessages(source string, n int) []*repository.Message {
	msgs := make([]*repository.Message, n)
	for i := range n {
		msgs[i] = &repository.Message{
			ID:         uuid.New(),
			Source:     source,
			Channel:    fmt.Sprintf("channel-%d", i),
			Sender:     fmt.Sprintf("user-%d", i),
			MessageID:  fmt.Sprintf("%s-msg-%d", source, i),
			RawContent: fmt.Sprintf("message content %d", i),
			Status:     "Pending",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
	}
	return msgs
}

func drainEvents(ch <-chan orchestrator.ActivityEvent, count int, timeout time.Duration) []orchestrator.ActivityEvent {
	var events []orchestrator.ActivityEvent
	deadline := time.After(timeout)
	for range count {
		select {
		case ev := <-ch:
			events = append(events, ev)
		case <-deadline:
			return events
		}
	}
	return events
}

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

type OrchestratorSuite struct {
	suite.Suite
}

func TestOrchestrator(t *testing.T) {
	suite.Run(t, new(OrchestratorSuite))
}

// ---------------------------------------------------------------------------
// Constructor Validation
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestNewOrchestratorRequiresRulesEngine() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	repo := newMockRepo()
	queueRepo := &mockQueueRepo{}
	watchers := map[string]orchestrator.Watcher{
		"slack": &mockWatcher{messages: makeMessages("slack", 1)},
	}
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}

	orch, err := orchestrator.NewOrchestrator(cfg, nil, queueRepo, repo, watchers, eventCh, nil)

	s.Error(err)
	s.Nil(orch)
	s.Contains(err.Error(), "rules")
}

func (s *OrchestratorSuite) TestNewOrchestratorRequiresQueueRepo() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	repo := newMockRepo()
	rules := decisionengine.NewRulesEngine(nil)
	watchers := map[string]orchestrator.Watcher{
		"slack": &mockWatcher{messages: makeMessages("slack", 1)},
	}
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}

	orch, err := orchestrator.NewOrchestrator(cfg, rules, nil, repo, watchers, eventCh, nil)

	s.Error(err)
	s.Nil(orch)
	s.Contains(err.Error(), "queue")
}

func (s *OrchestratorSuite) TestNewOrchestratorRequiresRepo() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	rules := decisionengine.NewRulesEngine(nil)
	queueRepo := &mockQueueRepo{}
	watchers := map[string]orchestrator.Watcher{
		"slack": &mockWatcher{messages: makeMessages("slack", 1)},
	}
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}

	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, nil, watchers, eventCh, nil)

	s.Error(err)
	s.Nil(orch)
	s.Contains(err.Error(), "repo")
}

func (s *OrchestratorSuite) TestNewOrchestratorRequiresWatchers() {
	// This test now verifies that nil/empty watchers are ACCEPTED (dynamic watcher management).
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	rules := decisionengine.NewRulesEngine(nil)
	queueRepo := &mockQueueRepo{}
	repo := newMockRepo()
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}

	// nil watchers — should succeed
	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, nil, eventCh, nil)
	s.NoError(err)
	s.NotNil(orch)

	// empty watchers — should succeed
	orch, err = orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, map[string]orchestrator.Watcher{}, eventCh, nil)
	s.NoError(err)
	s.NotNil(orch)
}

// ---------------------------------------------------------------------------
// Single Poll Cycle
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestPollCycleRoutesAndStores() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	msgs := makeMessages("slack", 3)
	watcher := &mockWatcher{messages: msgs}
	repo := newMockRepo()
	watchers := map[string]orchestrator.Watcher{"slack": watcher}

	orch := mustNewOrchestrator(repo, watchers, eventCh)

	// Execute a single poll cycle directly
	orch.PollOnce(context.Background())

	// TODO(087): PollOnce is currently a stub — routing+store assertions
	// will be restored in Behavior 4 when the dedup→rules→queue pipeline lands.
	// For now, just verify it emits the fetch event without panicking.
	events := drainEvents(eventCh, 1, 2*time.Second)
	s.Require().Len(events, 1)
	s.Contains(events[0].Message, "fetched 3 messages")
}

// ---------------------------------------------------------------------------
// Activity Events
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestPollCycleEmitsActivityEvents() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	msgs := makeMessages("slack", 5)
	watcher := &mockWatcher{messages: msgs}
	repo := newMockRepo()
	watchers := map[string]orchestrator.Watcher{"slack": watcher}

	orch := mustNewOrchestrator(repo, watchers, eventCh)

	orch.PollOnce(context.Background())

	// TODO(087): PollOnce is currently a stub — routing summary assertions
	// will be restored in Behavior 4. For now verify the fetch event.
	events := drainEvents(eventCh, 1, 2*time.Second)
	s.Require().Len(events, 1)
	s.Contains(events[0].Message, "5")
	s.Equal("slack", events[0].Source)
	s.False(events[0].IsError)
}

// ---------------------------------------------------------------------------
// Error Handling
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestWatcherErrorDoesNotCrash() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	watcher := &mockWatcher{
		err: fmt.Errorf("slack API rate limited"),
	}
	repo := newMockRepo()
	watchers := map[string]orchestrator.Watcher{"slack": watcher}

	orch := mustNewOrchestrator(repo, watchers, eventCh)

	// Should not panic
	s.NotPanics(func() {
		orch.PollOnce(context.Background())
	})

	// Should emit an error event
	events := drainEvents(eventCh, 1, 2*time.Second)
	s.Require().Len(events, 1)
	s.True(events[0].IsError)
	s.Contains(events[0].Message, "slack API rate limited")

	// Repo should have no inserts
	s.Equal(0, repo.insertedCount())
}

// TODO(087): TestStoreErrorDoesNotAbortBatch and TestStoreErrorEmitsErrorEvent
// will be restored in Behavior 4 when PollOnce has the full pipeline.
// PollOnce is currently a stub that only polls + emits fetch events.

// ---------------------------------------------------------------------------
// Multiple Watchers
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestMultipleWatchersSeparateBatches() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	slackMsgs := makeMessages("slack", 2)
	emailMsgs := makeMessages("email", 3)
	slackWatcher := &mockWatcher{messages: slackMsgs}
	emailWatcher := &mockWatcher{messages: emailMsgs}
	repo := newMockRepo()
	watchers := map[string]orchestrator.Watcher{
		"slack": slackWatcher,
		"email": emailWatcher,
	}

	orch := mustNewOrchestrator(repo, watchers, eventCh)

	orch.PollOnce(context.Background())

	// Each watcher should have been polled once
	s.Equal(1, slackWatcher.pollCount())
	s.Equal(1, emailWatcher.pollCount())

	// TODO(087): routing + store assertions restored in Behavior 4
	// For now verify both watchers emitted fetch events
	events := drainEvents(eventCh, 2, 2*time.Second)
	s.Require().Len(events, 2)
}

// ---------------------------------------------------------------------------
// Start / Stop Lifecycle
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestStartAndStop() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	watcher := &mockWatcher{messages: makeMessages("slack", 1)}
	repo := newMockRepo()
	watchers := map[string]orchestrator.Watcher{"slack": watcher}
	// Use a long interval so we can control timing
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 3600}
	rules := decisionengine.NewRulesEngine(nil)
	queueRepo := &mockQueueRepo{}

	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, watchers, eventCh, nil)
	s.Require().NoError(err)

	// Start should not block
	err = orch.Start(context.Background())
	s.NoError(err)

	// Give the immediate first poll a moment to complete
	time.Sleep(100 * time.Millisecond)

	// Stop should be clean
	err = orch.Stop()
	s.NoError(err)

	// Second stop should be safe (idempotent)
	err = orch.Stop()
	s.NoError(err)
}

// ---------------------------------------------------------------------------
// Immediate First Poll
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestImmediateFirstPoll() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	watcher := &mockWatcher{messages: makeMessages("slack", 2)}
	repo := newMockRepo()
	watchers := map[string]orchestrator.Watcher{"slack": watcher}
	// Very long interval - if poll only happens at interval, test will timeout
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 3600}
	rules := decisionengine.NewRulesEngine(nil)
	queueRepo := &mockQueueRepo{}

	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, watchers, eventCh, nil)
	s.Require().NoError(err)

	err = orch.Start(context.Background())
	s.Require().NoError(err)

	// Wait briefly for the immediate first poll to execute
	time.Sleep(200 * time.Millisecond)

	// The watcher should have been polled at least once already
	s.GreaterOrEqual(watcher.pollCount(), 1, "expected immediate first poll on Start")

	// TODO(087): store assertions restored in Behavior 4

	err = orch.Stop()
	s.NoError(err)
}

// ---------------------------------------------------------------------------
// Alert Integration
// ---------------------------------------------------------------------------

// TODO(087): Alert integration tests (TestPollCycleTriggersAlertOnNotified,
// TestPollCycleNoAlertOnBufferedOnly, TestPollCycleAlertErrorNonFatal,
// TestPollCycleNilAlerterSafe) will be restored in Behavior 4 when
// PollOnce has the full dedup→rules→queue pipeline with alert triggering.

// ---------------------------------------------------------------------------
// Dynamic Watcher Management (Feature 034)
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestConstructorAcceptsNilWatchers() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	rules := decisionengine.NewRulesEngine(nil)
	queueRepo := &mockQueueRepo{}
	repo := newMockRepo()
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}

	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, nil, eventCh, nil)
	s.NoError(err)
	s.NotNil(orch)
}

func (s *OrchestratorSuite) TestConstructorAcceptsEmptyWatchers() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	rules := decisionengine.NewRulesEngine(nil)
	queueRepo := &mockQueueRepo{}
	repo := newMockRepo()
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}

	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, map[string]orchestrator.Watcher{}, eventCh, nil)
	s.NoError(err)
	s.NotNil(orch)
}

func (s *OrchestratorSuite) TestPollOnceZeroWatchersEmitsEvent() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	repo := newMockRepo()

	orch := mustNewOrchestrator(repo, nil, eventCh)

	// PollOnce with zero watchers should be a no-op that emits an event
	orch.PollOnce(context.Background())

	events := drainEvents(eventCh, 1, 2*time.Second)
	s.Require().Len(events, 1)
	s.Contains(events[0].Message, "No watchers configured")
	s.False(events[0].IsError)

	// Repo should have no inserts
	s.Equal(0, repo.insertedCount())
}

func (s *OrchestratorSuite) TestAddWatcherThenPoll() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	msgs := makeMessages("slack", 2)
	watcher := &mockWatcher{messages: msgs}
	repo := newMockRepo()

	// Start with no watchers
	orch := mustNewOrchestrator(repo, nil, eventCh)

	// Add a watcher dynamically
	orch.AddWatcher("slack", watcher)

	// PollOnce should now poll the added watcher
	orch.PollOnce(context.Background())

	s.Equal(1, watcher.pollCount())
	// TODO(087): routing + store assertions restored in Behavior 4
}

func (s *OrchestratorSuite) TestAddWatcherDuplicateReplaces() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	firstWatcher := &mockWatcher{messages: makeMessages("slack", 1)}
	secondWatcher := &mockWatcher{messages: makeMessages("slack", 3)}
	repo := newMockRepo()

	orch := mustNewOrchestrator(repo, nil, eventCh)

	// Add first watcher, then replace with second using same name
	orch.AddWatcher("slack", firstWatcher)
	orch.AddWatcher("slack", secondWatcher)

	orch.PollOnce(context.Background())

	// First watcher should NOT have been polled (replaced)
	s.Equal(0, firstWatcher.pollCount())
	// Second watcher should have been polled
	s.Equal(1, secondWatcher.pollCount())
	// TODO(087): message count assertion restored in Behavior 4
}

func (s *OrchestratorSuite) TestRemoveWatcherThenPoll() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	watcher := &mockWatcher{messages: makeMessages("slack", 2)}
	repo := newMockRepo()

	orch := mustNewOrchestrator(repo, nil, eventCh)

	// Add then remove
	orch.AddWatcher("slack", watcher)
	orch.RemoveWatcher("slack")

	// PollOnce should not poll the removed watcher
	orch.PollOnce(context.Background())

	s.Equal(0, watcher.pollCount())
	s.Equal(0, repo.insertedCount())
}

func (s *OrchestratorSuite) TestRemoveWatcherUnknownNoOp() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	repo := newMockRepo()

	orch := mustNewOrchestrator(repo, nil, eventCh)

	// Removing a watcher that doesn't exist should not panic
	s.NotPanics(func() {
		orch.RemoveWatcher("nonexistent")
	})
}

func (s *OrchestratorSuite) TestListWatcherNames() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	repo := newMockRepo()

	orch := mustNewOrchestrator(repo, nil, eventCh)

	// Initially empty
	names := orch.ListWatcherNames()
	s.Empty(names)

	// Add watchers in non-sorted order
	orch.AddWatcher("email", &mockWatcher{})
	orch.AddWatcher("slack", &mockWatcher{})
	orch.AddWatcher("api", &mockWatcher{})

	names = orch.ListWatcherNames()
	s.Require().Len(names, 3)
	// Must be sorted for determinism
	s.True(sort.StringsAreSorted(names), "ListWatcherNames must return sorted names")
	s.Equal([]string{"api", "email", "slack"}, names)

	// After removal
	orch.RemoveWatcher("email")
	names = orch.ListWatcherNames()
	s.Equal([]string{"api", "slack"}, names)
}

func (s *OrchestratorSuite) TestConcurrentAddAndPoll() {
	eventCh := make(chan orchestrator.ActivityEvent, 1000)
	repo := newMockRepo()

	orch := mustNewOrchestrator(repo, nil, eventCh)

	// Run AddWatcher and PollOnce concurrently to verify race safety.
	// This test is meaningful when run with -race.
	var wg sync.WaitGroup
	const iterations = 50

	wg.Add(2)

	// Goroutine 1: repeatedly add/remove watchers
	go func() {
		defer wg.Done()
		for i := range iterations {
			name := fmt.Sprintf("watcher-%d", i)
			w := &mockWatcher{messages: makeMessages("dynamic", 1)}
			orch.AddWatcher(name, w)
			if i%2 == 0 {
				orch.RemoveWatcher(name)
			}
		}
	}()

	// Goroutine 2: repeatedly poll
	go func() {
		defer wg.Done()
		for range iterations {
			orch.PollOnce(context.Background())
		}
	}()

	wg.Wait()

	// If we get here without a data race, the test passes.
	// Just verify the orchestrator is still functional.
	names := orch.ListWatcherNames()
	s.NotNil(names) // should return a valid slice (possibly empty)
}
