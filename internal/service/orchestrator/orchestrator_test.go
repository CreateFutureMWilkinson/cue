package orchestrator_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/orchestrator"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

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

// mockRouter implements orchestrator.BatchRouter for testing.
type mockRouter struct {
	mu      sync.Mutex
	batches [][]*repository.Message // record each RouteBatch call
	// routeFn lets tests control per-message status assignment
	routeFn func(msg *repository.Message) *repository.Message
}

func (r *mockRouter) RouteBatch(ctx context.Context, msgs []*repository.Message) ([]*repository.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.batches = append(r.batches, msgs)

	results := make([]*repository.Message, 0, len(msgs))
	for _, msg := range msgs {
		if r.routeFn != nil {
			results = append(results, r.routeFn(msg))
		} else {
			// Default: mark as Notified
			msg.Status = "Notified"
			msg.ImportanceScore = 8
			msg.ConfidenceScore = 0.9
			results = append(results, msg)
		}
	}
	return results, nil
}

func (r *mockRouter) batchCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.batches)
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

func (r *mockRepo) insertedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.inserted)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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

func (s *OrchestratorSuite) TestNewOrchestratorRequiresRouter() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	repo := newMockRepo()
	watchers := map[string]orchestrator.Watcher{
		"slack": &mockWatcher{messages: makeMessages("slack", 1)},
	}
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}

	orch, err := orchestrator.NewOrchestrator(cfg, nil, repo, watchers, eventCh)

	s.Error(err)
	s.Nil(orch)
	s.Contains(err.Error(), "router")
}

func (s *OrchestratorSuite) TestNewOrchestratorRequiresRepo() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	router := &mockRouter{}
	watchers := map[string]orchestrator.Watcher{
		"slack": &mockWatcher{messages: makeMessages("slack", 1)},
	}
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}

	orch, err := orchestrator.NewOrchestrator(cfg, router, nil, watchers, eventCh)

	s.Error(err)
	s.Nil(orch)
	s.Contains(err.Error(), "repo")
}

func (s *OrchestratorSuite) TestNewOrchestratorRequiresWatchers() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	router := &mockRouter{}
	repo := newMockRepo()
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}

	// nil watchers
	orch, err := orchestrator.NewOrchestrator(cfg, router, repo, nil, eventCh)
	s.Error(err)
	s.Nil(orch)

	// empty watchers
	orch, err = orchestrator.NewOrchestrator(cfg, router, repo, map[string]orchestrator.Watcher{}, eventCh)
	s.Error(err)
	s.Nil(orch)
}

// ---------------------------------------------------------------------------
// Single Poll Cycle
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestPollCycleRoutesAndStores() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	msgs := makeMessages("slack", 3)
	watcher := &mockWatcher{messages: msgs}
	router := &mockRouter{}
	repo := newMockRepo()
	watchers := map[string]orchestrator.Watcher{"slack": watcher}
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}

	orch, err := orchestrator.NewOrchestrator(cfg, router, repo, watchers, eventCh)
	s.Require().NoError(err)

	// Execute a single poll cycle directly
	orch.PollOnce(context.Background())

	// Router should have received one batch of 3 messages
	s.Equal(1, router.batchCount())
	// All 3 messages should be stored
	s.Equal(3, repo.insertedCount())
}

// ---------------------------------------------------------------------------
// Activity Events
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestPollCycleEmitsActivityEvents() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	msgs := makeMessages("slack", 5)
	watcher := &mockWatcher{messages: msgs}
	callCount := 0
	router := &mockRouter{
		routeFn: func(msg *repository.Message) *repository.Message {
			callCount++
			switch {
			case callCount <= 2:
				msg.Status = "Notified"
			case callCount <= 4:
				msg.Status = "Buffered"
			default:
				msg.Status = "Ignored"
			}
			return msg
		},
	}
	repo := newMockRepo()
	watchers := map[string]orchestrator.Watcher{"slack": watcher}
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}

	orch, err := orchestrator.NewOrchestrator(cfg, router, repo, watchers, eventCh)
	s.Require().NoError(err)

	orch.PollOnce(context.Background())

	// Expect at least two events: "fetched N" and "Routed X NOTIFIED, Y BUFFERED, Z IGNORED"
	events := drainEvents(eventCh, 2, 2*time.Second)
	s.Require().GreaterOrEqual(len(events), 2, "expected at least 2 activity events")

	// First event: fetch summary
	s.Contains(events[0].Message, "5")
	s.Equal("slack", events[0].Source)
	s.False(events[0].IsError)

	// Second event: routing summary with counts
	s.Contains(events[1].Message, "2 NOTIFIED")
	s.Contains(events[1].Message, "2 BUFFERED")
	s.Contains(events[1].Message, "1 IGNORED")
}

// ---------------------------------------------------------------------------
// Error Handling
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestWatcherErrorDoesNotCrash() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	watcher := &mockWatcher{
		err: fmt.Errorf("slack API rate limited"),
	}
	router := &mockRouter{}
	repo := newMockRepo()
	watchers := map[string]orchestrator.Watcher{"slack": watcher}
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}

	orch, err := orchestrator.NewOrchestrator(cfg, router, repo, watchers, eventCh)
	s.Require().NoError(err)

	// Should not panic
	s.NotPanics(func() {
		orch.PollOnce(context.Background())
	})

	// Should emit an error event
	events := drainEvents(eventCh, 1, 2*time.Second)
	s.Require().Len(events, 1)
	s.True(events[0].IsError)
	s.Contains(events[0].Message, "slack API rate limited")

	// Router should not have been called (no messages to route)
	s.Equal(0, router.batchCount())
	// Repo should have no inserts
	s.Equal(0, repo.insertedCount())
}

func (s *OrchestratorSuite) TestStoreErrorDoesNotAbortBatch() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	msgs := makeMessages("slack", 3)
	watcher := &mockWatcher{messages: msgs}
	router := &mockRouter{}
	repo := newMockRepo()

	// Make the second message fail to store
	repo.insertErr[msgs[1].ID.String()] = fmt.Errorf("database locked")

	watchers := map[string]orchestrator.Watcher{"slack": watcher}
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}

	orch, err := orchestrator.NewOrchestrator(cfg, router, repo, watchers, eventCh)
	s.Require().NoError(err)

	orch.PollOnce(context.Background())

	// 2 out of 3 should be stored successfully (msgs[0] and msgs[2])
	s.Equal(2, repo.insertedCount())
}

// ---------------------------------------------------------------------------
// Multiple Watchers
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestMultipleWatchersSeparateBatches() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	slackMsgs := makeMessages("slack", 2)
	emailMsgs := makeMessages("email", 3)
	slackWatcher := &mockWatcher{messages: slackMsgs}
	emailWatcher := &mockWatcher{messages: emailMsgs}
	router := &mockRouter{}
	repo := newMockRepo()
	watchers := map[string]orchestrator.Watcher{
		"slack": slackWatcher,
		"email": emailWatcher,
	}
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}

	orch, err := orchestrator.NewOrchestrator(cfg, router, repo, watchers, eventCh)
	s.Require().NoError(err)

	orch.PollOnce(context.Background())

	// Each watcher should have been polled once
	s.Equal(1, slackWatcher.pollCount())
	s.Equal(1, emailWatcher.pollCount())

	// Router should have received 2 separate batches (one per watcher)
	s.Equal(2, router.batchCount())

	// All 5 messages should be stored
	s.Equal(5, repo.insertedCount())
}

// ---------------------------------------------------------------------------
// Start / Stop Lifecycle
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestStartAndStop() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	watcher := &mockWatcher{messages: makeMessages("slack", 1)}
	router := &mockRouter{}
	repo := newMockRepo()
	watchers := map[string]orchestrator.Watcher{"slack": watcher}
	// Use a long interval so we can control timing
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 3600}

	orch, err := orchestrator.NewOrchestrator(cfg, router, repo, watchers, eventCh)
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
	router := &mockRouter{}
	repo := newMockRepo()
	watchers := map[string]orchestrator.Watcher{"slack": watcher}
	// Very long interval - if poll only happens at interval, test will timeout
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 3600}

	orch, err := orchestrator.NewOrchestrator(cfg, router, repo, watchers, eventCh)
	s.Require().NoError(err)

	err = orch.Start(context.Background())
	s.Require().NoError(err)

	// Wait briefly for the immediate first poll to execute
	time.Sleep(200 * time.Millisecond)

	// The watcher should have been polled at least once already
	s.GreaterOrEqual(watcher.pollCount(), 1, "expected immediate first poll on Start")

	// Messages should have been routed and stored
	s.GreaterOrEqual(repo.insertedCount(), 2, "expected messages stored from immediate first poll")

	err = orch.Stop()
	s.NoError(err)
}
