package sqlite_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	sqlite "github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"

	_ "modernc.org/sqlite"
)

// makeTestMessage creates a test message with the given source, status, and createdAt.
// Nullable fields (UserRating, UserFeedback, VectorID, ResolvedAt) are left nil.
func makeTestMessage(source string, status string, createdAt time.Time) *repository.Message {
	return &repository.Message{
		ID:              uuid.New(),
		Source:          source,
		SourceAccount:   "test-account",
		Channel:         "test-channel",
		Sender:          "test-sender",
		MessageID:       uuid.New().String(),
		RawContent:      "test content",
		ImportanceScore: 7.5,
		ConfidenceScore: 0.85,
		Status:          status,
		Reasoning:       "test reasoning",
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
}

type MessageRepoSuite struct {
	suite.Suite
}

func TestMessage(t *testing.T) {
	suite.Run(t, new(MessageRepoSuite))
}

func (s *MessageRepoSuite) TestCreateDatabase() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := sqlite.NewSQLiteMessageRepository(dbPath)
	s.Require().NoError(err)
	s.Require().NotNil(repo)

	_, statErr := os.Stat(dbPath)
	s.Require().NoError(statErr, "database file should exist on disk")
}

func (s *MessageRepoSuite) TestInsertAndQueryByID() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	rating := 8
	feedback := "very important message"
	vectorID := uuid.New()

	msg := &repository.Message{
		ID:              uuid.New(),
		Source:          "slack",
		SourceAccount:   "workspace-123",
		Channel:         "general",
		Sender:          "U12345",
		MessageID:       "slack-msg-001",
		RawContent:      "Server is on fire!",
		ImportanceScore: 9.5,
		ConfidenceScore: 0.95,
		Status:          "Notified",
		Reasoning:       "Server outage detected",
		UserRating:      &rating,
		UserFeedback:    &feedback,
		VectorID:        &vectorID,
		CreatedAt:       now,
		UpdatedAt:       now,
		ResolvedAt:      &now,
	}

	err = repo.Insert(ctx, msg)
	s.Require().NoError(err)

	results, err := repo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Require().Len(results, 1)

	got := results[0]
	s.Equal(msg.ID, got.ID)
	s.Equal(msg.Source, got.Source)
	s.Equal(msg.SourceAccount, got.SourceAccount)
	s.Equal(msg.Channel, got.Channel)
	s.Equal(msg.Sender, got.Sender)
	s.Equal(msg.MessageID, got.MessageID)
	s.Equal(msg.RawContent, got.RawContent)
	s.InDelta(msg.ImportanceScore, got.ImportanceScore, 0.001)
	s.InDelta(msg.ConfidenceScore, got.ConfidenceScore, 0.001)
	s.Equal(msg.Status, got.Status)
	s.Equal(msg.Reasoning, got.Reasoning)

	s.Require().NotNil(got.UserRating)
	s.Equal(*msg.UserRating, *got.UserRating)

	s.Require().NotNil(got.UserFeedback)
	s.Equal(*msg.UserFeedback, *got.UserFeedback)

	s.Require().NotNil(got.VectorID)
	s.Equal(*msg.VectorID, *got.VectorID)

	s.WithinDuration(msg.CreatedAt, got.CreatedAt, time.Second)
	s.WithinDuration(msg.UpdatedAt, got.UpdatedAt, time.Second)

	s.Require().NotNil(got.ResolvedAt)
	s.WithinDuration(*msg.ResolvedAt, *got.ResolvedAt, time.Second)
}

func (s *MessageRepoSuite) TestQueryByStatus() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	notified := makeTestMessage("slack", "Notified", now)
	buffered := makeTestMessage("slack", "Buffered", now.Add(time.Second))
	ignored := makeTestMessage("email", "Ignored", now.Add(2*time.Second))

	s.Require().NoError(repo.Insert(ctx, notified))
	s.Require().NoError(repo.Insert(ctx, buffered))
	s.Require().NoError(repo.Insert(ctx, ignored))

	results, err := repo.QueryByStatus(ctx, "Notified")
	s.Require().NoError(err)
	s.Len(results, 1)
	s.Equal(notified.ID, results[0].ID)

	results, err = repo.QueryByStatus(ctx, "Buffered")
	s.Require().NoError(err)
	s.Len(results, 1)
	s.Equal(buffered.ID, results[0].ID)

	results, err = repo.QueryByStatus(ctx, "Ignored")
	s.Require().NoError(err)
	s.Len(results, 1)
	s.Equal(ignored.ID, results[0].ID)

	results, err = repo.QueryByStatus(ctx, "Resolved")
	s.Require().NoError(err)
	s.Len(results, 0)
}

func (s *MessageRepoSuite) TestUpdateMessage() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	msg := makeTestMessage("slack", "Buffered", now)
	s.Require().NoError(repo.Insert(ctx, msg))

	// Verify initial nullable fields are nil.
	results, err := repo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Require().Len(results, 1)
	s.Nil(results[0].UserRating)
	s.Nil(results[0].UserFeedback)
	s.Nil(results[0].ResolvedAt)

	// Update the message.
	rating := 8
	feedback := "important"
	resolvedAt := time.Now().Truncate(time.Second)
	updatedAt := time.Now().Truncate(time.Second)

	msg.UserRating = &rating
	msg.UserFeedback = &feedback
	msg.Status = "Resolved"
	msg.ResolvedAt = &resolvedAt
	msg.UpdatedAt = updatedAt

	err = repo.Update(ctx, msg)
	s.Require().NoError(err)

	// Query back and verify updated fields.
	results, err = repo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Require().Len(results, 1)

	got := results[0]
	s.Equal("Resolved", got.Status)

	s.Require().NotNil(got.UserRating)
	s.Equal(8, *got.UserRating)

	s.Require().NotNil(got.UserFeedback)
	s.Equal("important", *got.UserFeedback)

	s.Require().NotNil(got.ResolvedAt)
	s.WithinDuration(resolvedAt, *got.ResolvedAt, time.Second)

	// UpdatedAt should be at or after the original CreatedAt.
	s.True(got.UpdatedAt.Compare(now) >= 0, "UpdatedAt should be at or after original CreatedAt")
}

func (s *MessageRepoSuite) TestFIFOEviction() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	var oldestID uuid.UUID
	for i := 0; i < 100; i++ {
		msg := makeTestMessage("slack", "Buffered", baseTime.Add(time.Duration(i)*time.Minute))
		if i == 0 {
			oldestID = msg.ID
		}
		s.Require().NoError(repo.Insert(ctx, msg))
	}

	// Insert message 101 — should trigger eviction of the oldest.
	msg101 := makeTestMessage("slack", "Buffered", baseTime.Add(100*time.Minute))
	s.Require().NoError(repo.Insert(ctx, msg101))

	count, err := repo.CountBySource(ctx, "slack")
	s.Require().NoError(err)
	s.Equal(100, count, "should have exactly 100 slack messages after eviction")

	results, err := repo.QueryAll(ctx)
	s.Require().NoError(err)

	// Verify oldest message was evicted.
	foundOldest := false
	foundMsg101 := false
	for _, r := range results {
		if r.ID == oldestID {
			foundOldest = true
		}
		if r.ID == msg101.ID {
			foundMsg101 = true
		}
	}
	s.False(foundOldest, "oldest message should have been evicted")
	s.True(foundMsg101, "message 101 should be present")
}

func (s *MessageRepoSuite) TestFIFOEvictionSourceIsolation() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Insert 100 slack messages.
	for i := 0; i < 100; i++ {
		msg := makeTestMessage("slack", "Buffered", baseTime.Add(time.Duration(i)*time.Minute))
		s.Require().NoError(repo.Insert(ctx, msg))
	}

	// Insert 100 email messages.
	for i := 0; i < 100; i++ {
		msg := makeTestMessage("email", "Buffered", baseTime.Add(time.Duration(i)*time.Minute))
		s.Require().NoError(repo.Insert(ctx, msg))
	}

	// Insert 101st slack message — triggers eviction only for slack.
	msg101 := makeTestMessage("slack", "Buffered", baseTime.Add(100*time.Minute))
	s.Require().NoError(repo.Insert(ctx, msg101))

	slackCount, err := repo.CountBySource(ctx, "slack")
	s.Require().NoError(err)
	s.Equal(100, slackCount, "slack should have exactly 100 messages")

	emailCount, err := repo.CountBySource(ctx, "email")
	s.Require().NoError(err)
	s.Equal(100, emailCount, "email should still have exactly 100 messages (untouched)")

	all, err := repo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Equal(200, len(all), "total messages should be 200")
}

func (s *MessageRepoSuite) TestQueryOldestToNewest() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()
	baseTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	// Insert 5 messages with distinct timestamps, in shuffled order.
	timestamps := []time.Time{
		baseTime,
		baseTime.Add(1 * time.Hour),
		baseTime.Add(2 * time.Hour),
		baseTime.Add(3 * time.Hour),
		baseTime.Add(4 * time.Hour),
	}

	// Insert in reverse order to ensure ordering comes from the query, not insertion order.
	for i := len(timestamps) - 1; i >= 0; i-- {
		msg := makeTestMessage("slack", "Buffered", timestamps[i])
		s.Require().NoError(repo.Insert(ctx, msg))
	}

	results, err := repo.QueryOldestToNewest(ctx, 5)
	s.Require().NoError(err)
	s.Require().Len(results, 5)

	for i := 0; i < len(results)-1; i++ {
		s.True(
			results[i].CreatedAt.Before(results[i+1].CreatedAt),
			"message %d (CreatedAt=%v) should be before message %d (CreatedAt=%v)",
			i, results[i].CreatedAt, i+1, results[i+1].CreatedAt,
		)
	}
}

func (s *MessageRepoSuite) TestQueryAll() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert 3 Notified, 2 Buffered, 1 Ignored.
	for i := 0; i < 3; i++ {
		msg := makeTestMessage("slack", "Notified", now.Add(time.Duration(i)*time.Second))
		s.Require().NoError(repo.Insert(ctx, msg))
	}
	for i := 0; i < 2; i++ {
		msg := makeTestMessage("email", "Buffered", now.Add(time.Duration(i+3)*time.Second))
		s.Require().NoError(repo.Insert(ctx, msg))
	}
	msg := makeTestMessage("slack", "Ignored", now.Add(5*time.Second))
	s.Require().NoError(repo.Insert(ctx, msg))

	results, err := repo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Len(results, 6)
}

func (s *MessageRepoSuite) TestCountBySource() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()
	baseTime := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 50; i++ {
		msg := makeTestMessage("slack", "Buffered", baseTime.Add(time.Duration(i)*time.Second))
		s.Require().NoError(repo.Insert(ctx, msg))
	}
	for i := 0; i < 30; i++ {
		msg := makeTestMessage("email", "Buffered", baseTime.Add(time.Duration(i)*time.Second))
		s.Require().NoError(repo.Insert(ctx, msg))
	}

	slackCount, err := repo.CountBySource(ctx, "slack")
	s.Require().NoError(err)
	s.Equal(50, slackCount)

	emailCount, err := repo.CountBySource(ctx, "email")
	s.Require().NoError(err)
	s.Equal(30, emailCount)

	unknownCount, err := repo.CountBySource(ctx, "unknown")
	s.Require().NoError(err)
	s.Equal(0, unknownCount)
}

func (s *MessageRepoSuite) TestWALMode() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	_, err := sqlite.NewSQLiteMessageRepository(dbPath)
	s.Require().NoError(err)

	db, err := sql.Open("sqlite", dbPath)
	s.Require().NoError(err)
	defer db.Close()

	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	s.Require().NoError(err)
	s.Equal("wal", journalMode)
}

func (s *MessageRepoSuite) TestNullableFields() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert message with all nullable fields as nil.
	msg := makeTestMessage("slack", "Buffered", now)
	s.Require().NoError(repo.Insert(ctx, msg))

	// Verify all nullable fields are nil.
	results, err := repo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Require().Len(results, 1)

	got := results[0]
	s.Nil(got.UserRating, "UserRating should be nil")
	s.Nil(got.UserFeedback, "UserFeedback should be nil")
	s.Nil(got.VectorID, "VectorID should be nil")
	s.Nil(got.ResolvedAt, "ResolvedAt should be nil")

	// Update: set only UserRating.
	rating := 5
	msg.UserRating = &rating
	msg.UpdatedAt = time.Now().Truncate(time.Second)
	err = repo.Update(ctx, msg)
	s.Require().NoError(err)

	// Query back and verify partial update.
	results, err = repo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Require().Len(results, 1)

	got = results[0]
	s.Require().NotNil(got.UserRating, "UserRating should now be set")
	s.Equal(5, *got.UserRating)
	s.Nil(got.UserFeedback, "UserFeedback should still be nil")
	s.Nil(got.VectorID, "VectorID should still be nil")
	s.Nil(got.ResolvedAt, "ResolvedAt should still be nil")
}

func (s *MessageRepoSuite) TestUpsertByMessageID() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert first message with a specific MessageID.
	msg1 := makeTestMessage("slack", "Buffered", now)
	msg1.MessageID = "slack-123"
	msg1.RawContent = "original"
	s.Require().NoError(repo.Insert(ctx, msg1))

	// Insert second message with the same MessageID but different content.
	msg2 := makeTestMessage("slack", "Buffered", now.Add(time.Second))
	msg2.MessageID = "slack-123"
	msg2.RawContent = "updated"
	s.Require().NoError(repo.Insert(ctx, msg2))

	// Should have exactly 1 message (upsert, not duplicate).
	results, err := repo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Require().Len(results, 1, "upsert should prevent duplicate MessageIDs")
	s.Equal("updated", results[0].RawContent, "content should be updated to the latest insert")
}
