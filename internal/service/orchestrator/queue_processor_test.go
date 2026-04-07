package orchestrator_test

import (
	"context"
	"errors"
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
	dequeueEntry  *repository.QueueEntry
	dequeueErr    error
	markDoneID    uuid.UUID
	markDoneErr   error
	markFailedID  uuid.UUID
	markFailedErr error
}

func (m *mockQueueRepo) Enqueue(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockQueueRepo) DequeueOldest(_ context.Context) (*repository.QueueEntry, error) {
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
	return 0, nil
}

func (m *mockQueueRepo) PurgeCompleted(_ context.Context) error {
	return nil
}

func (m *mockQueueRepo) PurgeOlderThan(_ context.Context, _ time.Time) error {
	return nil
}

func (m *mockQueueRepo) PurgeAll(_ context.Context) error {
	return nil
}

func (m *mockQueueRepo) ResetProcessing(_ context.Context) (int64, error) {
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

// mockScorer implements decisionengine.Scorer for testing.
type mockScorer struct {
	result *decisionengine.ScorerResult
	err    error
}

func (m *mockScorer) Score(_ context.Context, _ *repository.Message) (*decisionengine.ScorerResult, error) {
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
