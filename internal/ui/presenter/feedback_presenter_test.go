package presenter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Mock ---

type mockBufferReviewer struct {
	messages     []*repository.Message
	getErr       error
	getCalled    bool
	count        int
	countErr     error
	countCalled  bool
	saveRatingID uuid.UUID
	saveRating   int
	saveFeedback *string
	saveErr      error
	saveCalled   bool
	deleteID     uuid.UUID
	deleteErr    error
	deleteCalled bool
}

func (m *mockBufferReviewer) GetBufferedMessages(_ context.Context) ([]*repository.Message, error) {
	m.getCalled = true
	return m.messages, m.getErr
}

func (m *mockBufferReviewer) CountBuffered(_ context.Context) (int, error) {
	m.countCalled = true
	return m.count, m.countErr
}

func (m *mockBufferReviewer) SaveRating(_ context.Context, messageID uuid.UUID, rating int, feedback *string) error {
	m.saveCalled = true
	m.saveRatingID = messageID
	m.saveRating = rating
	m.saveFeedback = feedback
	return m.saveErr
}

func (m *mockBufferReviewer) DeleteMessage(_ context.Context, messageID uuid.UUID) error {
	m.deleteCalled = true
	m.deleteID = messageID
	return m.deleteErr
}

// --- Suite ---

type FeedbackPresenterSuite struct {
	suite.Suite
	reviewer  *mockBufferReviewer
	presenter *presenter.FeedbackPresenter
}

func TestFeedbackPresenter(t *testing.T) {
	suite.Run(t, new(FeedbackPresenterSuite))
}

func (s *FeedbackPresenterSuite) SetupTest() {
	s.reviewer = &mockBufferReviewer{}
	p, err := presenter.NewFeedbackPresenter(s.reviewer)
	s.Require().NoError(err)
	s.presenter = p
}

// --- Constructor ---

func (s *FeedbackPresenterSuite) TestNewPresenterNilReviewerReturnsError() {
	_, err := presenter.NewFeedbackPresenter(nil)
	s.Error(err)
}

// --- Load ---

func (s *FeedbackPresenterSuite) TestLoadFetchesBufferedMessages() {
	now := time.Now()
	msg1 := &repository.Message{
		ID:              uuid.New(),
		Source:          "slack",
		Sender:          "alice",
		Channel:         "general",
		RawContent:      "oldest message",
		ImportanceScore: 7.0,
		ConfidenceScore: 0.5,
		CreatedAt:       now.Add(-20 * time.Minute),
		Status:          "Buffered",
	}
	msg2 := &repository.Message{
		ID:              uuid.New(),
		Source:          "email",
		Sender:          "bob",
		Channel:         "inbox",
		RawContent:      "newer message",
		ImportanceScore: 8.0,
		ConfidenceScore: 0.6,
		CreatedAt:       now.Add(-10 * time.Minute),
		Status:          "Buffered",
	}

	s.reviewer.messages = []*repository.Message{msg1, msg2}

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)
	s.True(s.reviewer.getCalled)
	s.True(s.presenter.HasCurrent())

	current := s.presenter.Current()
	s.Require().NotNil(current)
	// Oldest first (matching buffer service behavior)
	s.Equal(msg1.ID, current.ID)
	s.Equal("slack", current.Source)
	s.Equal("alice", current.Sender)
	s.Equal("general", current.Channel)
	s.Equal("oldest message", current.Content)
	s.InDelta(7.0, current.ImportanceScore, 0.001)
	s.InDelta(0.5, current.ConfidenceScore, 0.001)
}

func (s *FeedbackPresenterSuite) TestLoadEmptyResultHasCurrentFalse() {
	s.reviewer.messages = []*repository.Message{}

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)

	s.False(s.presenter.HasCurrent())
	s.Nil(s.presenter.Current())
}

func (s *FeedbackPresenterSuite) TestLoadPropagatesError() {
	s.reviewer.getErr = errors.New("database down")

	err := s.presenter.Load(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "database down")
}

// --- Current / Counter ---

func (s *FeedbackPresenterSuite) TestCurrentReturnsFirstAfterLoad() {
	msgID := uuid.New()
	s.reviewer.messages = []*repository.Message{
		{
			ID:              msgID,
			Source:          "slack",
			Sender:          "alice",
			Channel:         "general",
			RawContent:      "test message",
			ImportanceScore: 7.5,
			ConfidenceScore: 0.7,
			CreatedAt:       time.Now(),
			Status:          "Buffered",
		},
	}

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)

	current := s.presenter.Current()
	s.Require().NotNil(current)
	s.Equal(msgID, current.ID)
}

func (s *FeedbackPresenterSuite) TestCounterReturnsOneOfNAfterLoad() {
	s.reviewer.messages = []*repository.Message{
		{ID: uuid.New(), Source: "slack", Sender: "a", Channel: "c", RawContent: "m1", CreatedAt: time.Now(), Status: "Buffered"},
		{ID: uuid.New(), Source: "slack", Sender: "b", Channel: "c", RawContent: "m2", CreatedAt: time.Now(), Status: "Buffered"},
		{ID: uuid.New(), Source: "email", Sender: "c", Channel: "c", RawContent: "m3", CreatedAt: time.Now(), Status: "Buffered"},
	}

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)

	s.Equal("1 of 3", s.presenter.Counter())
}

// --- SaveRating ---

func (s *FeedbackPresenterSuite) TestSaveRatingDelegatesToReviewerAndAdvances() {
	msg1ID := uuid.New()
	msg2ID := uuid.New()
	s.reviewer.messages = []*repository.Message{
		{ID: msg1ID, Source: "slack", Sender: "a", Channel: "c", RawContent: "m1", CreatedAt: time.Now(), Status: "Buffered"},
		{ID: msg2ID, Source: "email", Sender: "b", Channel: "c", RawContent: "m2", CreatedAt: time.Now(), Status: "Buffered"},
	}

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)

	feedback := "good catch"
	err = s.presenter.SaveRating(context.Background(), 8, &feedback)
	s.Require().NoError(err)

	// Verify delegation
	s.True(s.reviewer.saveCalled)
	s.Equal(msg1ID, s.reviewer.saveRatingID)
	s.Equal(8, s.reviewer.saveRating)
	s.Require().NotNil(s.reviewer.saveFeedback)
	s.Equal("good catch", *s.reviewer.saveFeedback)

	// Should have advanced to next message
	s.True(s.presenter.HasCurrent())
	current := s.presenter.Current()
	s.Require().NotNil(current)
	s.Equal(msg2ID, current.ID)
}

func (s *FeedbackPresenterSuite) TestSaveRatingNilFeedbackAccepted() {
	s.reviewer.messages = []*repository.Message{
		{ID: uuid.New(), Source: "slack", Sender: "a", Channel: "c", RawContent: "m1", CreatedAt: time.Now(), Status: "Buffered"},
	}

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)

	err = s.presenter.SaveRating(context.Background(), 5, nil)
	s.Require().NoError(err)
	s.True(s.reviewer.saveCalled)
	s.Nil(s.reviewer.saveFeedback)
}

func (s *FeedbackPresenterSuite) TestSaveRatingWhenNoCurrentReturnsError() {
	s.reviewer.messages = []*repository.Message{}

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)

	err = s.presenter.SaveRating(context.Background(), 5, nil)
	s.Error(err)
}

func (s *FeedbackPresenterSuite) TestSaveRatingPropagatesReviewerError() {
	s.reviewer.messages = []*repository.Message{
		{ID: uuid.New(), Source: "slack", Sender: "a", Channel: "c", RawContent: "m1", CreatedAt: time.Now(), Status: "Buffered"},
	}
	s.reviewer.saveErr = errors.New("save failed")

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)

	err = s.presenter.SaveRating(context.Background(), 5, nil)
	s.Error(err)
	s.Contains(err.Error(), "save failed")
}

// --- Skip ---

func (s *FeedbackPresenterSuite) TestSkipAdvancesToNextMessage() {
	msg1ID := uuid.New()
	msg2ID := uuid.New()
	s.reviewer.messages = []*repository.Message{
		{ID: msg1ID, Source: "slack", Sender: "a", Channel: "c", RawContent: "m1", CreatedAt: time.Now(), Status: "Buffered"},
		{ID: msg2ID, Source: "email", Sender: "b", Channel: "c", RawContent: "m2", CreatedAt: time.Now(), Status: "Buffered"},
	}

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)

	s.presenter.Skip()

	s.True(s.presenter.HasCurrent())
	current := s.presenter.Current()
	s.Require().NotNil(current)
	s.Equal(msg2ID, current.ID)
}

func (s *FeedbackPresenterSuite) TestSkipOnLastMessageHasCurrentBecomesFalse() {
	s.reviewer.messages = []*repository.Message{
		{ID: uuid.New(), Source: "slack", Sender: "a", Channel: "c", RawContent: "m1", CreatedAt: time.Now(), Status: "Buffered"},
	}

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)
	s.True(s.presenter.HasCurrent())

	s.presenter.Skip()

	s.False(s.presenter.HasCurrent())
	s.Nil(s.presenter.Current())
}

// --- Delete ---

func (s *FeedbackPresenterSuite) TestDeleteDelegatesToReviewerAndRemovesFromList() {
	msg1ID := uuid.New()
	msg2ID := uuid.New()
	s.reviewer.messages = []*repository.Message{
		{ID: msg1ID, Source: "slack", Sender: "a", Channel: "c", RawContent: "m1", CreatedAt: time.Now(), Status: "Buffered"},
		{ID: msg2ID, Source: "email", Sender: "b", Channel: "c", RawContent: "m2", CreatedAt: time.Now(), Status: "Buffered"},
	}

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)

	err = s.presenter.Delete(context.Background())
	s.Require().NoError(err)

	// Verify delegation
	s.True(s.reviewer.deleteCalled)
	s.Equal(msg1ID, s.reviewer.deleteID)

	// Should advance to next, and total count should decrease
	s.True(s.presenter.HasCurrent())
	current := s.presenter.Current()
	s.Require().NotNil(current)
	s.Equal(msg2ID, current.ID)
}

func (s *FeedbackPresenterSuite) TestDeletePropagatesReviewerError() {
	s.reviewer.messages = []*repository.Message{
		{ID: uuid.New(), Source: "slack", Sender: "a", Channel: "c", RawContent: "m1", CreatedAt: time.Now(), Status: "Buffered"},
	}
	s.reviewer.deleteErr = errors.New("delete failed")

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)

	err = s.presenter.Delete(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "delete failed")
}

func (s *FeedbackPresenterSuite) TestDeleteLastRemainingMessageHasCurrentBecomesFalse() {
	s.reviewer.messages = []*repository.Message{
		{ID: uuid.New(), Source: "slack", Sender: "a", Channel: "c", RawContent: "m1", CreatedAt: time.Now(), Status: "Buffered"},
	}

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)
	s.True(s.presenter.HasCurrent())

	err = s.presenter.Delete(context.Background())
	s.Require().NoError(err)

	s.False(s.presenter.HasCurrent())
	s.Nil(s.presenter.Current())
}

// --- Counter updates ---

func (s *FeedbackPresenterSuite) TestCounterUpdatesAfterSaveRating() {
	s.reviewer.messages = []*repository.Message{
		{ID: uuid.New(), Source: "slack", Sender: "a", Channel: "c", RawContent: "m1", CreatedAt: time.Now(), Status: "Buffered"},
		{ID: uuid.New(), Source: "slack", Sender: "b", Channel: "c", RawContent: "m2", CreatedAt: time.Now(), Status: "Buffered"},
		{ID: uuid.New(), Source: "slack", Sender: "c", Channel: "c", RawContent: "m3", CreatedAt: time.Now(), Status: "Buffered"},
	}

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)
	s.Equal("1 of 3", s.presenter.Counter())

	err = s.presenter.SaveRating(context.Background(), 7, nil)
	s.Require().NoError(err)
	s.Equal("2 of 3", s.presenter.Counter())
}

func (s *FeedbackPresenterSuite) TestCounterUpdatesAfterSkip() {
	s.reviewer.messages = []*repository.Message{
		{ID: uuid.New(), Source: "slack", Sender: "a", Channel: "c", RawContent: "m1", CreatedAt: time.Now(), Status: "Buffered"},
		{ID: uuid.New(), Source: "slack", Sender: "b", Channel: "c", RawContent: "m2", CreatedAt: time.Now(), Status: "Buffered"},
	}

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)
	s.Equal("1 of 2", s.presenter.Counter())

	s.presenter.Skip()
	s.Equal("2 of 2", s.presenter.Counter())
}

func (s *FeedbackPresenterSuite) TestCounterUpdatesAfterDelete() {
	s.reviewer.messages = []*repository.Message{
		{ID: uuid.New(), Source: "slack", Sender: "a", Channel: "c", RawContent: "m1", CreatedAt: time.Now(), Status: "Buffered"},
		{ID: uuid.New(), Source: "slack", Sender: "b", Channel: "c", RawContent: "m2", CreatedAt: time.Now(), Status: "Buffered"},
		{ID: uuid.New(), Source: "slack", Sender: "c", Channel: "c", RawContent: "m3", CreatedAt: time.Now(), Status: "Buffered"},
	}

	err := s.presenter.Load(context.Background())
	s.Require().NoError(err)
	s.Equal("1 of 3", s.presenter.Counter())

	err = s.presenter.Delete(context.Background())
	s.Require().NoError(err)
	// After delete, total decreases and position stays at 1
	s.Equal("1 of 2", s.presenter.Counter())
}
