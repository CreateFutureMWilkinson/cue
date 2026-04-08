package orchestrator_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"
	"github.com/CreateFutureMWilkinson/cue/internal/service/orchestrator"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

// mockQueueRepo implements repository.QueueRepository for testing.
type mockQueueRepo struct {
	mu            sync.Mutex
	enqueued      []uuid.UUID
	dequeueEntry  *repository.QueueEntry
	dequeueErr    error
	dequeueFunc   func() (*repository.QueueEntry, error)
	markDoneID    uuid.UUID
	markDoneErr   error
	markFailedID  uuid.UUID
	markFailedErr error

	pending    int
	pendingErr error

	purgeOlderThanCalled  bool
	purgeOlderThanCutoff  time.Time
	resetProcessingCalled bool
}

func (m *mockQueueRepo) Enqueue(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enqueued = append(m.enqueued, id)
	return nil
}

func (m *mockQueueRepo) enqueuedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.enqueued)
}

func (m *mockQueueRepo) enqueuedIDs() []uuid.UUID {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]uuid.UUID, len(m.enqueued))
	copy(cp, m.enqueued)
	return cp
}

func (m *mockQueueRepo) DequeueOldest(_ context.Context) (*repository.QueueEntry, error) {
	if m.dequeueFunc != nil {
		return m.dequeueFunc()
	}
	return m.dequeueEntry, m.dequeueErr
}

func (m *mockQueueRepo) MarkDone(_ context.Context, id uuid.UUID) error {
	m.markDoneID = id
	return m.markDoneErr
}

func (m *mockQueueRepo) MarkFailed(_ context.Context, id uuid.UUID) error {
	m.markFailedID = id
	return m.markFailedErr
}

func (m *mockQueueRepo) PendingCount(_ context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pending, m.pendingErr
}

func (m *mockQueueRepo) PurgeCompleted(_ context.Context) error {
	return nil
}

func (m *mockQueueRepo) PurgeOlderThan(_ context.Context, cutoff time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.purgeOlderThanCalled = true
	m.purgeOlderThanCutoff = cutoff
	return nil
}

func (m *mockQueueRepo) PurgeAll(_ context.Context) error {
	return nil
}

func (m *mockQueueRepo) ResetProcessing(_ context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resetProcessingCalled = true
	return 0, nil
}

// mockMsgRepo implements repository.MessageRepository for queue processor testing.
type mockMsgRepo struct {
	queryByIDMsg *repository.Message
	queryByIDErr error
	updatedMsg   *repository.Message
	updateErr    error
}

func (m *mockMsgRepo) Insert(_ context.Context, _ *repository.Message) error {
	return nil
}

func (m *mockMsgRepo) Update(_ context.Context, msg *repository.Message) error {
	m.updatedMsg = msg
	return m.updateErr
}

func (m *mockMsgRepo) QueryByID(_ context.Context, _ uuid.UUID) (*repository.Message, error) {
	return m.queryByIDMsg, m.queryByIDErr
}

func (m *mockMsgRepo) QueryByStatus(_ context.Context, _ string) ([]*repository.Message, error) {
	return nil, nil
}

func (m *mockMsgRepo) QueryAll(_ context.Context) ([]*repository.Message, error) {
	return nil, nil
}

func (m *mockMsgRepo) QueryOldestToNewest(_ context.Context, _ int) ([]*repository.Message, error) {
	return nil, nil
}

func (m *mockMsgRepo) CountBySource(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (m *mockMsgRepo) ExistsByMessageID(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (m *mockMsgRepo) MaxSourceCursor(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}

func (m *mockMsgRepo) DistinctChannels(_ context.Context, _, _ string) ([]string, error) {
	return nil, nil
}

// mockScorer implements decisionengine.Scorer for testing.
type mockScorer struct {
	result *decisionengine.ScorerResult
	err    error
}

func (m *mockScorer) ScoreWithContext(_ context.Context, _ *repository.Message, _ []decisionengine.FewShotExample) (*decisionengine.ScorerResult, error) {
	return m.result, m.err
}

// mockFewShotProvider implements decisionengine.FewShotProvider for testing.
type mockFewShotProvider struct {
	examples []decisionengine.FewShotExample
	err      error
	called   bool
	content  string // last content passed to GetExamples
}

func (m *mockFewShotProvider) GetExamples(_ context.Context, content string) ([]decisionengine.FewShotExample, error) {
	m.called = true
	m.content = content
	return m.examples, m.err
}

// mockCapturingScorer implements decisionengine.Scorer and captures the examples argument.
type mockCapturingScorer struct {
	result         *decisionengine.ScorerResult
	err            error
	calledExamples []decisionengine.FewShotExample
	calledWithNil  bool
}

func (m *mockCapturingScorer) ScoreWithContext(_ context.Context, _ *repository.Message, examples []decisionengine.FewShotExample) (*decisionengine.ScorerResult, error) {
	if examples == nil {
		m.calledWithNil = true
	}
	m.calledExamples = examples
	return m.result, m.err
}

// mockQueueAlerter implements orchestrator.Alerter for queue processor testing.
type mockQueueAlerter struct {
	playCount int
	playErr   error
}

func (m *mockQueueAlerter) PlayNotification(_ context.Context) error {
	m.playCount++
	return m.playErr
}

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

// QueueProcessorSuite tests the QueueProcessor behavior.
type QueueProcessorSuite struct {
	suite.Suite
}

func TestQueueProcessor(t *testing.T) {
	suite.Run(t, new(QueueProcessorSuite))
}

func (s *QueueProcessorSuite) TestProcessOneSuccessNotified() {
	// Arrange
	entryID := uuid.New()
	msgID := uuid.New()

	queueRepo := &mockQueueRepo{
		dequeueEntry: &repository.QueueEntry{
			ID:        entryID,
			MessageID: msgID,
			Status:    "pending",
		},
	}

	msg := &repository.Message{
		ID:     msgID,
		Status: "Pending",
		Source: "slack",
	}
	msgRepo := &mockMsgRepo{
		queryByIDMsg: msg,
	}

	scorer := &mockScorer{
		result: &decisionengine.ScorerResult{
			ImportanceScore: 9.0,
			ConfidenceScore: 0.9,
			Reasoning:       "important",
		},
	}

	alerter := &mockQueueAlerter{}
	eventCh := make(chan orchestrator.ActivityEvent, 10)

	processor, err := orchestrator.NewQueueProcessor(
		queueRepo,
		msgRepo,
		scorer,
		alerter,
		eventCh,
		7,   // importanceThreshold
		0.8, // confidenceThreshold
		time.Second,
	)
	s.Require().NoError(err)

	// Act
	processed, err := processor.ProcessOne(context.Background())

	// Assert
	s.Require().NoError(err)
	s.True(processed)

	s.Require().NotNil(msgRepo.updatedMsg)
	s.Equal(9.0, msgRepo.updatedMsg.ImportanceScore)
	s.Equal(0.9, msgRepo.updatedMsg.ConfidenceScore)
	s.Equal("Notified", msgRepo.updatedMsg.Status)
	s.Equal("important", msgRepo.updatedMsg.Reasoning)

	s.Equal(entryID, queueRepo.markDoneID)
	s.Equal(1, alerter.playCount)
}

func (s *QueueProcessorSuite) TestProcessOneEmptyQueueReturnsFalse() {
	// Arrange
	queueRepo := &mockQueueRepo{
		dequeueEntry: nil,
		dequeueErr:   nil,
	}

	msgRepo := &mockMsgRepo{}
	scorer := &mockScorer{}
	alerter := &mockQueueAlerter{}
	eventCh := make(chan orchestrator.ActivityEvent, 10)

	processor, err := orchestrator.NewQueueProcessor(
		queueRepo,
		msgRepo,
		scorer,
		alerter,
		eventCh,
		7,   // importanceThreshold
		0.8, // confidenceThreshold
		time.Second,
	)
	s.Require().NoError(err)

	// Act
	processed, err := processor.ProcessOne(context.Background())

	// Assert
	s.NoError(err)
	s.False(processed)
	s.Nil(msgRepo.updatedMsg, "no message should be updated when queue is empty")
	s.Equal(0, alerter.playCount, "no alert should play when queue is empty")
}

func (s *QueueProcessorSuite) TestProcessOneScorerErrorMarksBUFFERED() {
	// Arrange
	entryID := uuid.New()
	msgID := uuid.New()

	queueRepo := &mockQueueRepo{
		dequeueEntry: &repository.QueueEntry{
			ID:        entryID,
			MessageID: msgID,
			Status:    "pending",
		},
	}

	msg := &repository.Message{
		ID:     msgID,
		Status: "Pending",
		Source: "slack",
	}
	msgRepo := &mockMsgRepo{
		queryByIDMsg: msg,
	}

	scorer := &mockScorer{
		err: errors.New("ollama timeout"),
	}

	alerter := &mockQueueAlerter{}
	eventCh := make(chan orchestrator.ActivityEvent, 10)

	processor, err := orchestrator.NewQueueProcessor(
		queueRepo,
		msgRepo,
		scorer,
		alerter,
		eventCh,
		7,   // importanceThreshold
		0.8, // confidenceThreshold
		time.Second,
	)
	s.Require().NoError(err)

	// Act
	processed, err := processor.ProcessOne(context.Background())

	// Assert
	s.Require().NoError(err)
	s.True(processed)

	s.Require().NotNil(msgRepo.updatedMsg)
	s.Equal(7.0, msgRepo.updatedMsg.ImportanceScore)
	s.Equal(0.0, msgRepo.updatedMsg.ConfidenceScore)
	s.Equal("Buffered", msgRepo.updatedMsg.Status)
	s.Contains(msgRepo.updatedMsg.Reasoning, "Ollama scoring failed")

	s.Equal(entryID, queueRepo.markFailedID)
	s.Equal(uuid.Nil, queueRepo.markDoneID)
	s.Equal(0, alerter.playCount)
}

func (s *QueueProcessorSuite) TestStartStopProcessesEntriesAndStops() {
	// Arrange
	entryID := uuid.New()
	msgID := uuid.New()

	var callCount atomic.Int32
	entry := &repository.QueueEntry{
		ID:        entryID,
		MessageID: msgID,
		Status:    "pending",
	}

	queueRepo := &mockQueueRepo{
		dequeueFunc: func() (*repository.QueueEntry, error) {
			// First call returns an entry; subsequent calls return nil (empty queue).
			if callCount.Add(1) == 1 {
				return entry, nil
			}
			return nil, nil
		},
	}

	msg := &repository.Message{
		ID:     msgID,
		Status: "Pending",
		Source: "slack",
	}
	msgRepo := &mockMsgRepo{
		queryByIDMsg: msg,
	}

	scorer := &mockScorer{
		result: &decisionengine.ScorerResult{
			ImportanceScore: 9.0,
			ConfidenceScore: 0.9,
			Reasoning:       "important",
		},
	}

	alerter := &mockQueueAlerter{}
	eventCh := make(chan orchestrator.ActivityEvent, 10)

	processor, err := orchestrator.NewQueueProcessor(
		queueRepo,
		msgRepo,
		scorer,
		alerter,
		eventCh,
		7,                // importanceThreshold
		0.8,              // confidenceThreshold
		time.Millisecond, // very short cooldown
	)
	s.Require().NoError(err)

	// Act
	processor.Start(context.Background())
	time.Sleep(50 * time.Millisecond)
	processor.Stop()

	// Assert — the background loop should have processed the entry.
	s.Require().NotNil(msgRepo.updatedMsg, "Start should launch a goroutine that calls ProcessOne")
	s.Equal("Notified", msgRepo.updatedMsg.Status)
	s.Equal(entryID, queueRepo.markDoneID)
}

func (s *QueueProcessorSuite) TestProcessOne_FewShotProviderExamplesPassedToScorer() {
	// Arrange
	entryID := uuid.New()
	msgID := uuid.New()

	queueRepo := &mockQueueRepo{
		dequeueEntry: &repository.QueueEntry{
			ID:        entryID,
			MessageID: msgID,
			Status:    "pending",
		},
	}

	msg := &repository.Message{
		ID:         msgID,
		Status:     "Pending",
		Source:     "slack",
		RawContent: "server is on fire",
	}
	msgRepo := &mockMsgRepo{
		queryByIDMsg: msg,
	}

	fewShotExamples := []decisionengine.FewShotExample{
		{Content: "prod outage", UserRating: 9, Similarity: 0.95},
		{Content: "standup reminder", UserRating: 2, Similarity: 0.85},
	}

	fewShotMock := &mockFewShotProvider{
		examples: fewShotExamples,
	}

	scorer := &mockCapturingScorer{
		result: &decisionengine.ScorerResult{
			ImportanceScore: 9.0,
			ConfidenceScore: 0.9,
			Reasoning:       "server fire is critical",
			ScoringModel:    "neural-chat",
		},
	}

	alerter := &mockQueueAlerter{}
	eventCh := make(chan orchestrator.ActivityEvent, 10)

	processor, err := orchestrator.NewQueueProcessor(
		queueRepo,
		msgRepo,
		scorer,
		alerter,
		eventCh,
		7,   // importanceThreshold
		0.8, // confidenceThreshold
		time.Second,
	)
	s.Require().NoError(err)

	processor.SetFewShotProvider(fewShotMock)

	// Act
	processed, err := processor.ProcessOne(context.Background())

	// Assert
	s.Require().NoError(err)
	s.True(processed)

	// FewShotProvider should have been called with the message content
	s.True(fewShotMock.called, "FewShotProvider.GetExamples should be called")
	s.Equal("server is on fire", fewShotMock.content, "GetExamples should receive message content")

	// Scorer should have received the examples from the provider
	s.Require().NotNil(scorer.calledExamples, "ScoreWithContext should receive examples")
	s.Len(scorer.calledExamples, 2, "ScoreWithContext should receive 2 examples")

	// Message should record examples used and scoring model
	s.Require().NotNil(msgRepo.updatedMsg)
	s.Equal(2, msgRepo.updatedMsg.ExamplesUsed, "ExamplesUsed should be set to number of examples")
	s.Equal("neural-chat", msgRepo.updatedMsg.ScoringModel, "ScoringModel should be set from scorer result")
}

func (s *QueueProcessorSuite) TestProcessOne_NilFewShotProvider_ScoresWithoutExamples() {
	// Arrange
	entryID := uuid.New()
	msgID := uuid.New()

	queueRepo := &mockQueueRepo{
		dequeueEntry: &repository.QueueEntry{
			ID:        entryID,
			MessageID: msgID,
			Status:    "pending",
		},
	}

	msg := &repository.Message{
		ID:         msgID,
		Status:     "Pending",
		Source:     "slack",
		RawContent: "server is on fire",
	}
	msgRepo := &mockMsgRepo{
		queryByIDMsg: msg,
	}

	scorer := &mockCapturingScorer{
		result: &decisionengine.ScorerResult{
			ImportanceScore: 8.5,
			ConfidenceScore: 0.85,
			Reasoning:       "server issue detected",
			ScoringModel:    "neural-chat",
		},
	}

	alerter := &mockQueueAlerter{}
	eventCh := make(chan orchestrator.ActivityEvent, 10)

	processor, err := orchestrator.NewQueueProcessor(
		queueRepo,
		msgRepo,
		scorer,
		alerter,
		eventCh,
		7,   // importanceThreshold
		0.8, // confidenceThreshold
		time.Second,
	)
	s.Require().NoError(err)

	// No FewShotProvider set (nil by default)

	// Act
	processed, err := processor.ProcessOne(context.Background())

	// Assert
	s.Require().NoError(err)
	s.True(processed)

	// Scorer should have been called with nil examples
	s.True(scorer.calledWithNil, "ScoreWithContext should be called with nil examples when no FewShotProvider")
	s.Nil(scorer.calledExamples, "calledExamples should be nil when no FewShotProvider")

	// Message should record zero examples used
	s.Require().NotNil(msgRepo.updatedMsg)
	s.Equal(0, msgRepo.updatedMsg.ExamplesUsed, "ExamplesUsed should be 0 when no FewShotProvider")
	s.Equal("neural-chat", msgRepo.updatedMsg.ScoringModel, "ScoringModel should still be set")
	s.Equal("Notified", msgRepo.updatedMsg.Status)
}

func (s *QueueProcessorSuite) TestProcessOne_FewShotProviderError_ScoresWithoutExamples() {
	// Arrange
	entryID := uuid.New()
	msgID := uuid.New()

	queueRepo := &mockQueueRepo{
		dequeueEntry: &repository.QueueEntry{
			ID:        entryID,
			MessageID: msgID,
			Status:    "pending",
		},
	}

	msg := &repository.Message{
		ID:         msgID,
		Status:     "Pending",
		Source:     "slack",
		RawContent: "server is on fire",
	}
	msgRepo := &mockMsgRepo{
		queryByIDMsg: msg,
	}

	fewShotMock := &mockFewShotProvider{
		err: errors.New("vector store connection failed"),
	}

	scorer := &mockCapturingScorer{
		result: &decisionengine.ScorerResult{
			ImportanceScore: 8.0,
			ConfidenceScore: 0.75,
			Reasoning:       "server issue without context",
			ScoringModel:    "neural-chat",
		},
	}

	alerter := &mockQueueAlerter{}
	eventCh := make(chan orchestrator.ActivityEvent, 10)

	processor, err := orchestrator.NewQueueProcessor(
		queueRepo,
		msgRepo,
		scorer,
		alerter,
		eventCh,
		7,   // importanceThreshold
		0.8, // confidenceThreshold
		time.Second,
	)
	s.Require().NoError(err)

	processor.SetFewShotProvider(fewShotMock)

	// Act
	processed, err := processor.ProcessOne(context.Background())

	// Assert
	s.Require().NoError(err)
	s.True(processed)

	// FewShotProvider should have been called but failed
	s.True(fewShotMock.called, "FewShotProvider.GetExamples should be called")
	s.Equal("server is on fire", fewShotMock.content, "GetExamples should receive message content")

	// Scorer should have been called with nil examples due to graceful degradation
	s.True(scorer.calledWithNil, "ScoreWithContext should be called with nil examples when GetExamples fails")
	s.Nil(scorer.calledExamples, "calledExamples should be nil when GetExamples fails")

	// Message should record zero examples used despite having a provider
	s.Require().NotNil(msgRepo.updatedMsg)
	s.Equal(0, msgRepo.updatedMsg.ExamplesUsed, "ExamplesUsed should be 0 when GetExamples fails")
	s.Equal("neural-chat", msgRepo.updatedMsg.ScoringModel, "ScoringModel should still be set")
	s.Equal("Buffered", msgRepo.updatedMsg.Status) // IS=8.0, CS=0.75 → Buffered
}

func (s *QueueProcessorSuite) TestProcessOne_ScoringModelRecordedOnMessage() {
	// Arrange
	entryID := uuid.New()
	msgID := uuid.New()

	queueRepo := &mockQueueRepo{
		dequeueEntry: &repository.QueueEntry{
			ID:        entryID,
			MessageID: msgID,
			Status:    "pending",
		},
	}

	msg := &repository.Message{
		ID:     msgID,
		Status: "Pending",
		Source: "slack",
	}
	msgRepo := &mockMsgRepo{
		queryByIDMsg: msg,
	}

	scorer := &mockScorer{
		result: &decisionengine.ScorerResult{
			ImportanceScore: 6.5,
			ConfidenceScore: 0.9,
			Reasoning:       "routine update",
			ScoringModel:    "llama3.2:1b",
		},
	}

	alerter := &mockQueueAlerter{}
	eventCh := make(chan orchestrator.ActivityEvent, 10)

	processor, err := orchestrator.NewQueueProcessor(
		queueRepo,
		msgRepo,
		scorer,
		alerter,
		eventCh,
		7,   // importanceThreshold
		0.8, // confidenceThreshold
		time.Second,
	)
	s.Require().NoError(err)

	// Act
	processed, err := processor.ProcessOne(context.Background())

	// Assert
	s.Require().NoError(err)
	s.True(processed)

	s.Require().NotNil(msgRepo.updatedMsg)
	s.Equal("llama3.2:1b", msgRepo.updatedMsg.ScoringModel, "ScoringModel should match the result from scorer")
	s.Equal("Ignored", msgRepo.updatedMsg.Status) // IS=6.5 < 7 → Ignored
}
