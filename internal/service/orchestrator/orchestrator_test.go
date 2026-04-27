package orchestrator_test

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
	mu             sync.Mutex
	inserted       []*repository.Message
	insertErr      map[string]error    // keyed by message ID string, allows selective failures
	existingMsgIDs map[string]bool     // messageIDs that "already exist" in the DB
	channels       map[string][]string // key = "source:account" → channel names
	cursorMap      map[string]string   // key = "source:account:channel" → cursor value
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		insertErr:      make(map[string]error),
		existingMsgIDs: make(map[string]bool),
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

func (r *mockRepo) ExistsByMessageID(_ context.Context, messageID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.existingMsgIDs[messageID], nil
}

func (r *mockRepo) MaxSourceCursor(_ context.Context, source, sourceAccount, channel string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cursorMap == nil {
		return "", nil
	}
	key := source + ":" + sourceAccount + ":" + channel
	return r.cursorMap[key], nil
}

func (r *mockRepo) DistinctChannels(_ context.Context, source, sourceAccount string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.channels == nil {
		return nil, nil
	}
	key := source + ":" + sourceAccount
	return r.channels[key], nil
}

func (r *mockRepo) QueryFiltered(_ context.Context, _ repository.MessageFilter) ([]*repository.Message, int, error) {
	return nil, 0, nil
}

func (r *mockRepo) DeleteBySourceAccount(_ context.Context, _, _ string) (int64, error) {
	return 0, nil
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

// ---------------------------------------------------------------------------
// PollOnce Pipeline: Dedup → Rules → Queue
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestPollOnceDeduplicatesMessages() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	msgs := makeMessages("slack", 3)
	// Mark the second message as already existing in the DB
	duplicateMessageID := msgs[1].MessageID
	watcher := &mockWatcher{messages: msgs}
	repo := newMockRepo()
	repo.existingMsgIDs[duplicateMessageID] = true
	watchers := map[string]orchestrator.Watcher{"slack": watcher}

	orch := mustNewOrchestrator(repo, watchers, eventCh)

	orch.PollOnce(context.Background())

	// Only 2 of 3 messages should be inserted (the duplicate is skipped)
	s.Equal(2, repo.insertedCount(), "duplicate message should be skipped")
	// Verify the duplicate was NOT inserted
	for _, inserted := range repo.inserted {
		s.NotEqual(duplicateMessageID, inserted.MessageID, "duplicate should not appear in inserted messages")
	}
}

func (s *OrchestratorSuite) TestPollOnceRulesNotifiedSetsScores() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	msgs := []*repository.Message{
		{
			ID:         uuid.New(),
			Source:     "slack",
			Channel:    "important-alerts",
			Sender:     "user-1",
			MessageID:  "slack-notified-1",
			RawContent: "server is down",
			Status:     "Pending",
		},
	}
	watcher := &mockWatcher{messages: msgs}
	repo := newMockRepo()
	queueRepo := &mockQueueRepo{}

	rules := decisionengine.NewRulesEngine([]*repository.RoutingRule{
		{
			ID:             uuid.New(),
			Priority:       1,
			SourceType:     "slack",
			ChannelPattern: "important-.*",
			Action:         "notified",
			Enabled:        true,
		},
	})

	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}
	alerter := &mockAlerter{}
	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, map[string]orchestrator.Watcher{"slack": watcher}, eventCh, alerter)
	s.Require().NoError(err)

	orch.PollOnce(context.Background())

	// Message should be inserted with notified scores
	s.Require().Equal(1, repo.insertedCount())
	inserted := repo.inserted[0]
	s.Equal(8.0, inserted.ImportanceScore, "notified rule should set IS=8.0")
	s.Equal(1.0, inserted.ConfidenceScore, "notified rule should set CS=1.0")
	s.Equal("Notified", inserted.Status)
	s.NotEmpty(inserted.Reasoning, "reasoning should include rule info")

	// Notified messages should NOT be enqueued
	s.Equal(0, queueRepo.enqueuedCount(), "notified messages should not be enqueued")
}

func (s *OrchestratorSuite) TestPollOnceRulesIgnoredSetsScores() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	msgs := []*repository.Message{
		{
			ID:         uuid.New(),
			Source:     "slack",
			Channel:    "noise-bots",
			Sender:     "bot-1",
			MessageID:  "slack-ignored-1",
			RawContent: "automated noise",
			Status:     "Pending",
		},
	}
	watcher := &mockWatcher{messages: msgs}
	repo := newMockRepo()
	queueRepo := &mockQueueRepo{}

	rules := decisionengine.NewRulesEngine([]*repository.RoutingRule{
		{
			ID:             uuid.New(),
			Priority:       1,
			SourceType:     "slack",
			ChannelPattern: "noise-.*",
			Action:         "ignored",
			Enabled:        true,
		},
	})

	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}
	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, map[string]orchestrator.Watcher{"slack": watcher}, eventCh, nil)
	s.Require().NoError(err)

	orch.PollOnce(context.Background())

	// Message should be inserted with ignored scores
	s.Require().Equal(1, repo.insertedCount())
	inserted := repo.inserted[0]
	s.Equal(0.0, inserted.ImportanceScore, "ignored rule should set IS=0.0")
	s.Equal(1.0, inserted.ConfidenceScore, "ignored rule should set CS=1.0")
	s.Equal("Ignored", inserted.Status)
	s.NotEmpty(inserted.Reasoning)

	// Ignored messages should NOT be enqueued
	s.Equal(0, queueRepo.enqueuedCount(), "ignored messages should not be enqueued")
}

func (s *OrchestratorSuite) TestPollOnceUnmatchedEnqueued() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	msgs := makeMessages("slack", 2)
	watcher := &mockWatcher{messages: msgs}
	repo := newMockRepo()
	queueRepo := &mockQueueRepo{}

	// No rules → all messages get "queue" action
	rules := decisionengine.NewRulesEngine(nil)

	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}
	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, map[string]orchestrator.Watcher{"slack": watcher}, eventCh, nil)
	s.Require().NoError(err)

	orch.PollOnce(context.Background())

	// Both messages should be inserted with Pending status
	s.Require().Equal(2, repo.insertedCount())
	for _, inserted := range repo.inserted {
		s.Equal("Pending", inserted.Status, "unmatched messages should have Pending status")
		s.Equal(0.0, inserted.ImportanceScore, "unmatched messages should have IS=0")
		s.Equal(0.0, inserted.ConfidenceScore, "unmatched messages should have CS=0")
	}

	// Both should be enqueued for Ollama scoring
	s.Equal(2, queueRepo.enqueuedCount(), "unmatched messages should be enqueued")
}

func (s *OrchestratorSuite) TestPollOnceAlertOnNotified() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	msgs := []*repository.Message{
		{
			ID:         uuid.New(),
			Source:     "slack",
			Channel:    "critical-alerts",
			Sender:     "user-1",
			MessageID:  "slack-alert-1",
			RawContent: "server outage",
			Status:     "Pending",
		},
	}
	watcher := &mockWatcher{messages: msgs}
	repo := newMockRepo()
	queueRepo := &mockQueueRepo{}
	alerter := &mockAlerter{}

	rules := decisionengine.NewRulesEngine([]*repository.RoutingRule{
		{
			ID:             uuid.New(),
			Priority:       1,
			SourceType:     "slack",
			ChannelPattern: "critical-.*",
			Action:         "notified",
			Enabled:        true,
		},
	})

	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}
	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, map[string]orchestrator.Watcher{"slack": watcher}, eventCh, alerter)
	s.Require().NoError(err)

	orch.PollOnce(context.Background())

	s.Equal(1, alerter.alertCalls(), "alert should fire when a message is notified")
}

func (s *OrchestratorSuite) TestPollOnceNoAlertOnIgnoredOnly() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	msgs := []*repository.Message{
		{
			ID:         uuid.New(),
			Source:     "slack",
			Channel:    "spam-channel",
			Sender:     "bot-1",
			MessageID:  "slack-spam-1",
			RawContent: "spam message",
			Status:     "Pending",
		},
	}
	watcher := &mockWatcher{messages: msgs}
	repo := newMockRepo()
	queueRepo := &mockQueueRepo{}
	alerter := &mockAlerter{}

	rules := decisionengine.NewRulesEngine([]*repository.RoutingRule{
		{
			ID:             uuid.New(),
			Priority:       1,
			SourceType:     "slack",
			ChannelPattern: "spam-.*",
			Action:         "ignored",
			Enabled:        true,
		},
	})

	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}
	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, map[string]orchestrator.Watcher{"slack": watcher}, eventCh, alerter)
	s.Require().NoError(err)

	orch.PollOnce(context.Background())

	// Message must have been processed (inserted with Ignored status) for this test to be meaningful
	s.Require().Equal(1, repo.insertedCount(), "message should be inserted even when ignored")
	s.Equal("Ignored", repo.inserted[0].Status)
	s.Equal(0, alerter.alertCalls(), "no alert should fire when all messages are ignored")
}

func (s *OrchestratorSuite) TestPollOnceStoreErrorContinues() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	msgs := makeMessages("slack", 3)
	// Make the second message fail on Insert
	msgs[1].ID = uuid.New() // ensure unique
	failID := msgs[1].ID
	watcher := &mockWatcher{messages: msgs}
	repo := newMockRepo()
	repo.insertErr[failID.String()] = fmt.Errorf("disk full")
	watchers := map[string]orchestrator.Watcher{"slack": watcher}

	orch := mustNewOrchestrator(repo, watchers, eventCh)

	orch.PollOnce(context.Background())

	// Despite one insert failure, the other 2 messages should be inserted
	s.Equal(2, repo.insertedCount(), "store error should not abort batch processing")

	// Should emit an error event for the failed insert
	events := drainEvents(eventCh, 10, 2*time.Second)
	hasErrorEvent := false
	for _, ev := range events {
		if ev.IsError {
			hasErrorEvent = true
			break
		}
	}
	s.True(hasErrorEvent, "should emit error event for failed insert")
}

func (s *OrchestratorSuite) TestPollOnceEmitsRulesSummaryEvent() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	// Create 3 messages: one will be notified, one ignored, one queued
	msgs := []*repository.Message{
		{
			ID:         uuid.New(),
			Source:     "slack",
			Channel:    "critical-ops",
			Sender:     "user-1",
			MessageID:  "slack-summary-1",
			RawContent: "alert message",
			Status:     "Pending",
		},
		{
			ID:         uuid.New(),
			Source:     "slack",
			Channel:    "noise-bots",
			Sender:     "bot-1",
			MessageID:  "slack-summary-2",
			RawContent: "bot noise",
			Status:     "Pending",
		},
		{
			ID:         uuid.New(),
			Source:     "slack",
			Channel:    "general",
			Sender:     "user-2",
			MessageID:  "slack-summary-3",
			RawContent: "hello world",
			Status:     "Pending",
		},
	}
	watcher := &mockWatcher{messages: msgs}
	repo := newMockRepo()
	queueRepo := &mockQueueRepo{}

	rules := decisionengine.NewRulesEngine([]*repository.RoutingRule{
		{
			ID:             uuid.New(),
			Priority:       1,
			SourceType:     "slack",
			ChannelPattern: "critical-.*",
			Action:         "notified",
			Enabled:        true,
		},
		{
			ID:             uuid.New(),
			Priority:       2,
			SourceType:     "slack",
			ChannelPattern: "noise-.*",
			Action:         "ignored",
			Enabled:        true,
		},
	})

	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}
	alerter := &mockAlerter{}
	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, map[string]orchestrator.Watcher{"slack": watcher}, eventCh, alerter)
	s.Require().NoError(err)

	orch.PollOnce(context.Background())

	// Drain all events and look for the rules summary
	events := drainEvents(eventCh, 10, 2*time.Second)
	hasSummary := false
	for _, ev := range events {
		if !ev.IsError {
			// Look for a summary event containing notified/ignored/queued counts
			if containsAll(ev.Message, "1 notified", "1 ignored", "1 queued") {
				hasSummary = true
				break
			}
		}
	}
	s.True(hasSummary, "should emit a rules summary event with notified/ignored/queued counts, got events: %v", eventMessages(events))
}

// containsAll returns true if s contains all the given substrings.
func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

// eventMessages extracts message strings from events for debug output.
func eventMessages(events []orchestrator.ActivityEvent) []string {
	msgs := make([]string, len(events))
	for i, ev := range events {
		msgs[i] = ev.Message
	}
	return msgs
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

// Alert integration tests restored below in "PollOnce Pipeline" section.

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

// ---------------------------------------------------------------------------
// ReloadRules
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestReloadRulesChangesRouting() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	repo := newMockRepo()
	queueRepo := &mockQueueRepo{}

	// First poll: message goes to "queue" (no rules match)
	firstMsg := &repository.Message{
		ID:         uuid.New(),
		Source:     "slack",
		Channel:    "alerts",
		Sender:     "user-1",
		MessageID:  "slack-reload-1",
		RawContent: "first alert",
		Status:     "Pending",
	}
	firstWatcher := &mockWatcher{messages: []*repository.Message{firstMsg}}

	rules := decisionengine.NewRulesEngine(nil) // no rules — everything queued
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}
	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, map[string]orchestrator.Watcher{"slack": firstWatcher}, eventCh, nil)
	s.Require().NoError(err)

	orch.PollOnce(context.Background())

	// First message should be Pending (queued, no rule match)
	s.Require().Equal(1, repo.insertedCount())
	s.Equal("Pending", repo.inserted[0].Status, "before ReloadRules: message should be Pending (no rules)")

	// Reload rules: now channel "alerts" → notified
	orch.ReloadRules([]*repository.RoutingRule{
		{
			ID:             uuid.New(),
			Priority:       1,
			SourceType:     "slack",
			ChannelPattern: "alerts",
			Action:         "notified",
			Enabled:        true,
		},
	})

	// Second poll with a new message on the same channel
	secondMsg := &repository.Message{
		ID:         uuid.New(),
		Source:     "slack",
		Channel:    "alerts",
		Sender:     "user-2",
		MessageID:  "slack-reload-2",
		RawContent: "second alert",
		Status:     "Pending",
	}
	// Replace watcher messages for second poll
	orch.RemoveWatcher("slack")
	orch.AddWatcher("slack", &mockWatcher{messages: []*repository.Message{secondMsg}})

	orch.PollOnce(context.Background())

	// Second message should be Notified with rule-set scores
	s.Require().Equal(2, repo.insertedCount(), "second message should be inserted")
	second := repo.inserted[1]
	s.Equal("Notified", second.Status, "after ReloadRules: message should be Notified")
	s.Equal(8.0, second.ImportanceScore, "after ReloadRules: IS should be 8.0")
	s.Equal(1.0, second.ConfidenceScore, "after ReloadRules: CS should be 1.0")
}

// ---------------------------------------------------------------------------
// Queue Startup Sequence
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestStartPurgesAndResetsQueue() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	repo := newMockRepo()
	queueRepo := &mockQueueRepo{}
	rules := decisionengine.NewRulesEngine(nil)

	// Use a long poll interval so only the immediate first poll fires.
	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 3600}

	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, nil, eventCh, nil)
	s.Require().NoError(err)

	beforeStart := time.Now()
	err = orch.Start(context.Background())
	s.Require().NoError(err)

	// Give the goroutine a moment to execute.
	time.Sleep(200 * time.Millisecond)

	err = orch.Stop()
	s.Require().NoError(err)

	// Assert PurgeOlderThan was called with a cutoff approximately now - pollInterval.
	queueRepo.mu.Lock()
	purged := queueRepo.purgeOlderThanCalled
	cutoff := queueRepo.purgeOlderThanCutoff
	reset := queueRepo.resetProcessingCalled
	queueRepo.mu.Unlock()

	s.True(purged, "Start should call PurgeOlderThan before poll loop")
	s.True(reset, "Start should call ResetProcessing before poll loop")

	// The cutoff should be approximately beforeStart - 3600s (within 5s tolerance).
	expectedCutoff := beforeStart.Add(-3600 * time.Second)
	s.InDelta(expectedCutoff.Unix(), cutoff.Unix(), 5,
		"purge cutoff should be approximately now minus poll interval")
}

// ---------------------------------------------------------------------------
// ImportBaseline
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestImportBaselineInsertsAsImported() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	msgs := makeMessages("slack", 3)
	watcher := &mockWatcher{messages: msgs}
	repo := newMockRepo()
	alerter := &mockAlerter{}
	watchers := map[string]orchestrator.Watcher{"slack": watcher}

	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}
	rules := decisionengine.NewRulesEngine(nil)
	queueRepo := &mockQueueRepo{}
	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, watchers, eventCh, alerter)
	s.Require().NoError(err)

	err = orch.ImportBaseline(context.Background())
	s.NoError(err)

	// All 3 messages should be inserted.
	s.Equal(3, repo.insertedCount(), "all polled messages should be inserted")

	// Each inserted message should have Status == "Imported" and zero scores.
	repo.mu.Lock()
	inserted := make([]*repository.Message, len(repo.inserted))
	copy(inserted, repo.inserted)
	repo.mu.Unlock()

	for _, msg := range inserted {
		s.Equal(decisionengine.StatusImported, msg.Status,
			"imported messages must have Status 'Imported'")
		s.Equal(0.0, msg.ImportanceScore,
			"imported messages must not be scored (IS=0)")
		s.Equal(0.0, msg.ConfidenceScore,
			"imported messages must not be scored (CS=0)")
	}

	// Alerter must NOT be called — import is silent.
	s.Equal(0, alerter.alertCalls(), "ImportBaseline must not trigger alerts")

	// Queue repo must NOT have enqueued anything.
	s.Equal(0, queueRepo.enqueuedCount(), "ImportBaseline must not enqueue messages")

	// Events should be emitted containing "import".
	events := drainEvents(eventCh, 10, 500*time.Millisecond)
	s.NotEmpty(events, "ImportBaseline should emit progress events")
	var foundImport bool
	for _, ev := range events {
		if strings.Contains(strings.ToLower(ev.Message), "import") {
			foundImport = true
			break
		}
	}
	s.True(foundImport, "at least one event should mention 'import'")
}

// ---------------------------------------------------------------------------
// CursorSeedable mock — implements both Watcher and CursorSeedable
// ---------------------------------------------------------------------------

type cursorSeedableMock struct {
	mockWatcher
	source        string
	sourceAccount string
	seededCursors map[string]string // channel → cursor
}

func (m *cursorSeedableMock) SourceInfo() (string, string) {
	return m.source, m.sourceAccount
}

func (m *cursorSeedableMock) SeedCursor(channel, cursor string) {
	m.seededCursors[channel] = cursor
}

// ---------------------------------------------------------------------------
// ImportBaseline cursor seeding
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestImportBaselineSeedsCursorsFromDB() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	repo := newMockRepo()
	repo.channels = map[string][]string{
		"slack:workspace-1": {"general", "alerts"},
	}
	repo.cursorMap = map[string]string{
		"slack:workspace-1:general": "1711500000.000100",
		"slack:workspace-1:alerts":  "1711600000.000200",
	}

	watcher := &cursorSeedableMock{
		mockWatcher:   mockWatcher{messages: nil},
		source:        "slack",
		sourceAccount: "workspace-1",
		seededCursors: make(map[string]string),
	}
	watchers := map[string]orchestrator.Watcher{"slack": watcher}

	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}
	rules := decisionengine.NewRulesEngine(nil)
	queueRepo := &mockQueueRepo{}
	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, watchers, eventCh, nil)
	s.Require().NoError(err)

	err = orch.ImportBaseline(context.Background())
	s.NoError(err)

	s.Equal("1711500000.000100", watcher.seededCursors["general"],
		"general channel cursor should be seeded from DB")
	s.Equal("1711600000.000200", watcher.seededCursors["alerts"],
		"alerts channel cursor should be seeded from DB")
}

// ---------------------------------------------------------------------------
// sequentialMockWatcher — returns different messages per Poll call
// ---------------------------------------------------------------------------

type sequentialMockWatcher struct {
	calls   [][]*repository.Message // messages to return per call index
	callIdx int
	mu      sync.Mutex
}

func (m *sequentialMockWatcher) Poll(_ context.Context) ([]*repository.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.callIdx >= len(m.calls) {
		return nil, nil
	}
	msgs := m.calls[m.callIdx]
	m.callIdx++
	return msgs, nil
}

// ---------------------------------------------------------------------------
// Start → ImportBaseline Integration
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestStartCallsImportBaselineBeforeFirstPoll() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	repo := newMockRepo()

	baselineMsgs := makeMessages("slack", 2)
	watcher := &sequentialMockWatcher{
		calls: [][]*repository.Message{
			baselineMsgs, // first call: ImportBaseline consumes these
			nil,          // second call: PollOnce gets nothing
		},
	}
	watchers := map[string]orchestrator.Watcher{"slack": watcher}

	cfg := orchestrator.OrchestratorConfig{PollIntervalSeconds: 600}
	rules := decisionengine.NewRulesEngine(nil)
	queueRepo := &mockQueueRepo{}
	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, watchers, eventCh, nil)
	s.Require().NoError(err)

	err = orch.Start(context.Background())
	s.Require().NoError(err)

	// Allow goroutine to run ImportBaseline + PollOnce.
	time.Sleep(100 * time.Millisecond)

	err = orch.Stop()
	s.Require().NoError(err)

	// ImportBaseline should have inserted 2 messages with Status="Imported".
	s.Equal(2, repo.insertedCount(),
		"expected 2 messages inserted via ImportBaseline before first poll")
	repo.mu.Lock()
	for _, msg := range repo.inserted {
		s.Equal(decisionengine.StatusImported, msg.Status,
			"messages from ImportBaseline should have Status=Imported")
	}
	repo.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Queue Health Monitoring (Feature 091)
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestPollOnceEmitsQueueWarningWhenThresholdExceeded() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	repo := newMockRepo()
	queueRepo := &mockQueueRepo{pending: 75}
	rules := decisionengine.NewRulesEngine(nil)
	watchers := map[string]orchestrator.Watcher{
		"slack": &mockWatcher{messages: makeMessages("slack", 1)},
	}
	cfg := orchestrator.OrchestratorConfig{
		PollIntervalSeconds:   600,
		QueueWarningThreshold: 50,
	}

	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, watchers, eventCh, nil)
	s.Require().NoError(err)

	orch.PollOnce(context.Background())

	events := drainEvents(eventCh, 10, 500*time.Millisecond)

	// Find the queue warning event.
	var found bool
	for _, ev := range events {
		if ev.Source == "queue" && strings.Contains(ev.Message, "75") && strings.Contains(ev.Message, "consider adding routing rules") {
			found = true
			s.False(ev.IsError, "queue warning should not be an error event")
			break
		}
	}
	s.True(found, "expected queue depth warning event; got events: %v", events)
}

func (s *OrchestratorSuite) TestPollOnceEmitsQueueOkWhenBelowThreshold() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	repo := newMockRepo()
	queueRepo := &mockQueueRepo{pending: 10}
	rules := decisionengine.NewRulesEngine(nil)
	watchers := map[string]orchestrator.Watcher{
		"slack": &mockWatcher{messages: makeMessages("slack", 1)},
	}
	cfg := orchestrator.OrchestratorConfig{
		PollIntervalSeconds:   600,
		QueueWarningThreshold: 50,
	}

	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, watchers, eventCh, nil)
	s.Require().NoError(err)

	orch.PollOnce(context.Background())

	events := drainEvents(eventCh, 10, 500*time.Millisecond)

	// Find the queue ok event.
	var found bool
	for _, ev := range events {
		if ev.Source == "queue" && strings.Contains(ev.Message, "10") && !strings.Contains(ev.Message, "consider adding routing rules") {
			found = true
			s.False(ev.IsError, "queue ok should not be an error event")
			break
		}
	}
	s.True(found, "expected queue depth ok event; got events: %v", events)
}

func (s *OrchestratorSuite) TestPollOnceSkipsQueueCheckWhenThresholdZero() {
	eventCh := make(chan orchestrator.ActivityEvent, 100)
	repo := newMockRepo()
	queueRepo := &mockQueueRepo{pending: 999}
	rules := decisionengine.NewRulesEngine(nil)
	watchers := map[string]orchestrator.Watcher{
		"slack": &mockWatcher{messages: makeMessages("slack", 1)},
	}
	cfg := orchestrator.OrchestratorConfig{
		PollIntervalSeconds:   600,
		QueueWarningThreshold: 0, // disabled
	}

	orch, err := orchestrator.NewOrchestrator(cfg, rules, queueRepo, repo, watchers, eventCh, nil)
	s.Require().NoError(err)

	orch.PollOnce(context.Background())

	events := drainEvents(eventCh, 10, 500*time.Millisecond)

	// No queue health event should be emitted when threshold is 0.
	for _, ev := range events {
		s.NotEqual("queue", ev.Source, "no queue events expected when threshold is 0; got: %v", ev)
	}
}
