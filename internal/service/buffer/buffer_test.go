package buffer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/buffer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

// mockMessageRepo implements buffer.MessageRepository for testing.
type mockMessageRepo struct {
	queryByStatusResult []*repository.Message
	queryByStatusErr    error
	queryByStatusArg    string // captures the status arg passed to QueryByStatus

	updateCalls []*repository.Message // tracks all Update calls
	updateErr   error
}

func (m *mockMessageRepo) QueryByStatus(_ context.Context, status string) ([]*repository.Message, error) {
	m.queryByStatusArg = status
	return m.queryByStatusResult, m.queryByStatusErr
}

func (m *mockMessageRepo) Update(_ context.Context, msg *repository.Message) error {
	m.updateCalls = append(m.updateCalls, msg)
	return m.updateErr
}

// mockVectorEmbedder implements buffer.VectorEmbedder for testing.
type mockVectorEmbedder struct {
	resultID *uuid.UUID
	err      error
	called   bool
	lastMsg  uuid.UUID
	lastText string
}

func (m *mockVectorEmbedder) StoreEmbedding(_ context.Context, messageID uuid.UUID, content string) (*uuid.UUID, error) {
	m.called = true
	m.lastMsg = messageID
	m.lastText = content
	return m.resultID, m.err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestMessage(id uuid.UUID, createdAt time.Time) *repository.Message {
	return &repository.Message{
		ID:              id,
		Source:          "slack",
		SourceAccount:   "T123",
		Channel:         "general",
		Sender:          "U456",
		MessageID:       "msg-" + id.String()[:8],
		MessageType:     "message",
		RawContent:      "test content for " + id.String()[:8],
		ImportanceScore: 7.0,
		ConfidenceScore: 0.5,
		Status:          "Buffered",
		Reasoning:       "might be important",
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
}

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

type BufferServiceSuite struct {
	suite.Suite
}

func TestBufferService(t *testing.T) {
	suite.Run(t, new(BufferServiceSuite))
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func (s *BufferServiceSuite) TestNewBufferService_NilRepo_ReturnsError() {
	embedder := &mockVectorEmbedder{}

	bs, err := buffer.NewBufferService(nil, embedder)

	s.Error(err)
	s.Nil(bs)
	s.Contains(err.Error(), "repo")
}

func (s *BufferServiceSuite) TestNewBufferService_NilEmbedder_OK() {
	repo := &mockMessageRepo{}

	bs, err := buffer.NewBufferService(repo, nil)

	s.NoError(err)
	s.NotNil(bs)
}

func (s *BufferServiceSuite) TestNewBufferService_ValidInputs() {
	repo := &mockMessageRepo{}
	embedder := &mockVectorEmbedder{}

	bs, err := buffer.NewBufferService(repo, embedder)

	s.NoError(err)
	s.NotNil(bs)
}

// ---------------------------------------------------------------------------
// GetBufferedMessages
// ---------------------------------------------------------------------------

func (s *BufferServiceSuite) TestGetBufferedMessages_SortsOldestFirst() {
	now := time.Now()
	msg1 := newTestMessage(uuid.New(), now.Add(-1*time.Hour)) // oldest
	msg2 := newTestMessage(uuid.New(), now)                   // newest
	msg3 := newTestMessage(uuid.New(), now.Add(-30*time.Minute))

	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{msg2, msg1, msg3}, // unsorted
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	result, err := bs.GetBufferedMessages(context.Background())

	s.NoError(err)
	s.Require().Len(result, 3)
	s.Equal(msg1.ID, result[0].ID, "oldest message should be first")
	s.Equal(msg3.ID, result[1].ID, "middle message should be second")
	s.Equal(msg2.ID, result[2].ID, "newest message should be last")
}

func (s *BufferServiceSuite) TestGetBufferedMessages_QueriesBufferedStatus() {
	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{},
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	_, err = bs.GetBufferedMessages(context.Background())

	s.NoError(err)
	s.Equal("Buffered", repo.queryByStatusArg)
}

func (s *BufferServiceSuite) TestGetBufferedMessages_Empty_ReturnsEmptySlice() {
	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{},
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	result, err := bs.GetBufferedMessages(context.Background())

	s.NoError(err)
	s.NotNil(result, "should return empty slice, not nil")
	s.Empty(result)
}

func (s *BufferServiceSuite) TestGetBufferedMessages_RepoError_Propagates() {
	repoErr := errors.New("database connection lost")
	repo := &mockMessageRepo{
		queryByStatusErr: repoErr,
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	result, err := bs.GetBufferedMessages(context.Background())

	s.Error(err)
	s.Nil(result)
	s.ErrorIs(err, repoErr)
}

// ---------------------------------------------------------------------------
// CountBuffered
// ---------------------------------------------------------------------------

func (s *BufferServiceSuite) TestCountBuffered_ReturnsCount() {
	now := time.Now()
	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{
			newTestMessage(uuid.New(), now),
			newTestMessage(uuid.New(), now),
			newTestMessage(uuid.New(), now),
		},
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	count, err := bs.CountBuffered(context.Background())

	s.NoError(err)
	s.Equal(3, count)
}

func (s *BufferServiceSuite) TestCountBuffered_RepoError_Propagates() {
	repoErr := errors.New("query failed")
	repo := &mockMessageRepo{
		queryByStatusErr: repoErr,
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	count, err := bs.CountBuffered(context.Background())

	s.Error(err)
	s.Equal(0, count)
	s.ErrorIs(err, repoErr)
}

// ---------------------------------------------------------------------------
// SaveRating
// ---------------------------------------------------------------------------

func (s *BufferServiceSuite) TestSaveRating_SetsFieldsCorrectly() {
	msgID := uuid.New()
	now := time.Now()
	msg := newTestMessage(msgID, now.Add(-1*time.Hour))

	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{msg},
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	beforeSave := time.Now()
	feedback := "very relevant message"
	err = bs.SaveRating(context.Background(), msgID, 8, &feedback)

	s.NoError(err)
	s.Require().NotEmpty(repo.updateCalls, "Update should have been called")

	updated := repo.updateCalls[0]
	s.Equal("Resolved", updated.Status)
	s.NotNil(updated.UserRating)
	s.Equal(8, *updated.UserRating)
	s.NotNil(updated.UserFeedback)
	s.Equal("very relevant message", *updated.UserFeedback)
	s.NotNil(updated.ResolvedAt, "ResolvedAt must be set")
	s.False(updated.ResolvedAt.Before(beforeSave), "ResolvedAt should be recent")
	s.False(updated.UpdatedAt.Before(beforeSave), "UpdatedAt should be recent")
}

func (s *BufferServiceSuite) TestSaveRating_RatingTooLow_Error() {
	msgID := uuid.New()
	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{newTestMessage(msgID, time.Now())},
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	err = bs.SaveRating(context.Background(), msgID, -1, nil)

	s.Error(err)
	s.Contains(err.Error(), "rating")
}

func (s *BufferServiceSuite) TestSaveRating_RatingTooHigh_Error() {
	msgID := uuid.New()
	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{newTestMessage(msgID, time.Now())},
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	err = bs.SaveRating(context.Background(), msgID, 11, nil)

	s.Error(err)
	s.Contains(err.Error(), "rating")
}

func (s *BufferServiceSuite) TestSaveRating_NilFeedback_OK() {
	msgID := uuid.New()
	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{newTestMessage(msgID, time.Now())},
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	err = bs.SaveRating(context.Background(), msgID, 5, nil)

	s.NoError(err)
	s.Require().NotEmpty(repo.updateCalls)
	s.Nil(repo.updateCalls[0].UserFeedback)
}

func (s *BufferServiceSuite) TestSaveRating_WithFeedback() {
	msgID := uuid.New()
	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{newTestMessage(msgID, time.Now())},
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	feedback := "this was actually critical"
	err = bs.SaveRating(context.Background(), msgID, 9, &feedback)

	s.NoError(err)
	s.Require().NotEmpty(repo.updateCalls)
	s.NotNil(repo.updateCalls[0].UserFeedback)
	s.Equal("this was actually critical", *repo.updateCalls[0].UserFeedback)
}

func (s *BufferServiceSuite) TestSaveRating_TriggersEmbedding() {
	msgID := uuid.New()
	msg := newTestMessage(msgID, time.Now())
	vectorID := uuid.New()

	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{msg},
	}
	embedder := &mockVectorEmbedder{
		resultID: &vectorID,
	}
	bs, err := buffer.NewBufferService(repo, embedder)
	s.Require().NoError(err)

	err = bs.SaveRating(context.Background(), msgID, 7, nil)

	s.NoError(err)
	s.True(embedder.called, "embedder.StoreEmbedding should have been called")
	s.Equal(msgID, embedder.lastMsg)
	s.Equal(msg.RawContent, embedder.lastText)

	// VectorID should be set on the message via a second Update call
	s.Require().GreaterOrEqual(len(repo.updateCalls), 2,
		"Update should be called twice: once for rating, once for VectorID")
	lastUpdate := repo.updateCalls[len(repo.updateCalls)-1]
	s.NotNil(lastUpdate.VectorID)
	s.Equal(vectorID, *lastUpdate.VectorID)
}

func (s *BufferServiceSuite) TestSaveRating_NilEmbedder_SkipsEmbedding() {
	msgID := uuid.New()
	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{newTestMessage(msgID, time.Now())},
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	// Should not panic when embedder is nil
	err = bs.SaveRating(context.Background(), msgID, 5, nil)

	s.NoError(err)
	// Only one Update call (no embedding update)
	s.Len(repo.updateCalls, 1)
}

func (s *BufferServiceSuite) TestSaveRating_EmbeddingFailure_StillSucceeds() {
	msgID := uuid.New()
	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{newTestMessage(msgID, time.Now())},
	}
	embedder := &mockVectorEmbedder{
		err: errors.New("embedding service unavailable"),
	}
	bs, err := buffer.NewBufferService(repo, embedder)
	s.Require().NoError(err)

	err = bs.SaveRating(context.Background(), msgID, 5, nil)

	s.NoError(err, "embedding failure must not fail the save")
	s.True(embedder.called)
	// Rating Update should still have happened
	s.Require().NotEmpty(repo.updateCalls)
	s.Equal("Resolved", repo.updateCalls[0].Status)
}

func (s *BufferServiceSuite) TestSaveRating_MessageNotFound_Error() {
	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{
			newTestMessage(uuid.New(), time.Now()), // different ID
		},
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	unknownID := uuid.New()
	err = bs.SaveRating(context.Background(), unknownID, 5, nil)

	s.Error(err)
	s.Contains(err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// DeleteMessage
// ---------------------------------------------------------------------------

func (s *BufferServiceSuite) TestDeleteMessage_TransitionsToResolved() {
	msgID := uuid.New()
	msg := newTestMessage(msgID, time.Now().Add(-1*time.Hour))

	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{msg},
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	beforeDelete := time.Now()
	err = bs.DeleteMessage(context.Background(), msgID)

	s.NoError(err)
	s.Require().NotEmpty(repo.updateCalls)

	updated := repo.updateCalls[0]
	s.Equal("Resolved", updated.Status)
	s.NotNil(updated.ResolvedAt)
	s.False(updated.ResolvedAt.Before(beforeDelete))
	s.False(updated.UpdatedAt.Before(beforeDelete))
}

func (s *BufferServiceSuite) TestDeleteMessage_DoesNotEmbed() {
	msgID := uuid.New()
	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{newTestMessage(msgID, time.Now())},
	}
	embedder := &mockVectorEmbedder{}
	bs, err := buffer.NewBufferService(repo, embedder)
	s.Require().NoError(err)

	err = bs.DeleteMessage(context.Background(), msgID)

	s.NoError(err)
	s.False(embedder.called, "embedder should NOT be called during delete")
	// Verify the message was actually processed (Update called)
	s.Require().NotEmpty(repo.updateCalls, "Update must be called during delete")
	s.Equal("Resolved", repo.updateCalls[0].Status)
}

func (s *BufferServiceSuite) TestDeleteMessage_NotFound_Error() {
	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{
			newTestMessage(uuid.New(), time.Now()),
		},
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	unknownID := uuid.New()
	err = bs.DeleteMessage(context.Background(), unknownID)

	s.Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *BufferServiceSuite) TestDeleteMessage_RepoError_Propagates() {
	msgID := uuid.New()
	repo := &mockMessageRepo{
		queryByStatusResult: []*repository.Message{newTestMessage(msgID, time.Now())},
		updateErr:           errors.New("disk full"),
	}
	bs, err := buffer.NewBufferService(repo, nil)
	s.Require().NoError(err)

	err = bs.DeleteMessage(context.Background(), msgID)

	s.Error(err)
	s.ErrorIs(err, repo.updateErr)
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func (s *BufferServiceSuite) TestSaveRating_BoundaryValues() {
	s.Run("rating=0 is valid", func() {
		msgID := uuid.New()
		repo := &mockMessageRepo{
			queryByStatusResult: []*repository.Message{newTestMessage(msgID, time.Now())},
		}
		bs, err := buffer.NewBufferService(repo, nil)
		s.Require().NoError(err)

		err = bs.SaveRating(context.Background(), msgID, 0, nil)
		s.NoError(err)
		s.Require().NotEmpty(repo.updateCalls, "Update must be called for rating=0")
		s.Equal(0, *repo.updateCalls[0].UserRating)
	})

	s.Run("rating=10 is valid", func() {
		msgID := uuid.New()
		repo := &mockMessageRepo{
			queryByStatusResult: []*repository.Message{newTestMessage(msgID, time.Now())},
		}
		bs, err := buffer.NewBufferService(repo, nil)
		s.Require().NoError(err)

		err = bs.SaveRating(context.Background(), msgID, 10, nil)
		s.NoError(err)
		s.Require().NotEmpty(repo.updateCalls, "Update must be called for rating=10")
		s.Equal(10, *repo.updateCalls[0].UserRating)
	})
}
