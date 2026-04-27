package sqlite_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
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

	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)
	s.Require().NotNil(repo)

	_, statErr := os.Stat(dbPath)
	s.Require().NoError(statErr, "database file should exist on disk")
}

func (s *MessageRepoSuite) TestInsertAndQueryByID() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
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
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
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
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
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
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
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
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
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
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
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
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
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
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
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

	_, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
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
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
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

func (s *MessageRepoSuite) TestDBAccessorReturnsNonNilDB() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)
	s.Require().NotNil(repo)

	db := repo.DB()
	s.Require().NotNil(db, "DB() should return a non-nil *sql.DB")

	// Verify the returned DB is functional by pinging it.
	err = db.Ping()
	s.Require().NoError(err, "DB() handle should be pingable")
}

// --- Feature 042: QueryByID ---

func (s *MessageRepoSuite) TestQueryByID_ReturnsCorrectMessage() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	msg := makeTestMessage("slack", "Notified", now)
	s.Require().NoError(repo.Insert(ctx, msg))

	// Insert a second message to ensure we get the right one.
	msg2 := makeTestMessage("email", "Buffered", now.Add(time.Second))
	s.Require().NoError(repo.Insert(ctx, msg2))

	got, err := repo.QueryByID(ctx, msg.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal(msg.ID, got.ID)
	s.Equal(msg.Source, got.Source)
	s.Equal(msg.RawContent, got.RawContent)
}
func (s *MessageRepoSuite) TestQueryByID_NullableFieldsHandled() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert message with all nullable fields set.
	rating := 7
	feedback := "good catch"
	vectorID := uuid.New()

	msg := &repository.Message{
		ID:              uuid.New(),
		Source:          "slack",
		SourceAccount:   "workspace-1",
		Channel:         "alerts",
		Sender:          "U999",
		MessageID:       uuid.New().String(),
		RawContent:      "server alert",
		ImportanceScore: 8.5,
		ConfidenceScore: 0.9,
		Status:          "Notified",
		Reasoning:       "outage detected",
		UserRating:      &rating,
		UserFeedback:    &feedback,
		VectorID:        &vectorID,
		CreatedAt:       now,
		UpdatedAt:       now,
		ResolvedAt:      &now,
	}
	s.Require().NoError(repo.Insert(ctx, msg))

	got, err := repo.QueryByID(ctx, msg.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Require().NotNil(got.UserRating)
	s.Equal(7, *got.UserRating)

	s.Require().NotNil(got.UserFeedback)
	s.Equal("good catch", *got.UserFeedback)

	s.Require().NotNil(got.VectorID)
	s.Equal(vectorID, *got.VectorID)

	s.Require().NotNil(got.ResolvedAt)
	s.WithinDuration(now, *got.ResolvedAt, time.Second)

	// Also insert a message with nil nullable fields and verify.
	msg2 := makeTestMessage("email", "Ignored", now.Add(time.Second))
	s.Require().NoError(repo.Insert(ctx, msg2))

	got2, err := repo.QueryByID(ctx, msg2.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got2)
	s.Nil(got2.UserRating)
	s.Nil(got2.UserFeedback)
	s.Nil(got2.VectorID)
	s.Nil(got2.ResolvedAt)
}

func (s *MessageRepoSuite) TestUpsertByMessageID() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
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

// --- Feature 047: MessageType Persistence ---

func (s *MessageRepoSuite) TestMessageTypePersisted() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	msg := makeTestMessage("slack", "Notified", now)
	msg.MessageType = "channel_join"
	s.Require().NoError(repo.Insert(ctx, msg))

	got, err := repo.QueryByID(ctx, msg.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal("channel_join", got.MessageType, "MessageType should round-trip through insert and query")
}

func (s *MessageRepoSuite) TestMessageTypeEmptyStringPersisted() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	msg := makeTestMessage("email", "Buffered", now)
	// MessageType is not set — defaults to empty string.
	s.Require().NoError(repo.Insert(ctx, msg))

	got, err := repo.QueryByID(ctx, msg.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal("", got.MessageType, "MessageType should be empty string when not set")
}

func (s *MessageRepoSuite) TestMessageTypeUpdated() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	msg := makeTestMessage("slack", "Buffered", now)
	msg.MessageType = "message"
	s.Require().NoError(repo.Insert(ctx, msg))

	// Update MessageType to channel_join.
	msg.MessageType = "channel_join"
	msg.UpdatedAt = time.Now().Truncate(time.Second)
	err = repo.Update(ctx, msg)
	s.Require().NoError(err)

	got, err := repo.QueryByID(ctx, msg.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal("channel_join", got.MessageType, "MessageType should reflect the updated value")
}

func (s *MessageRepoSuite) TestMessageTypeMigrationIdempotent() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open the repo once to create the schema.
	repo1, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)
	_ = repo1

	// Open the repo a second time on the same database — migration should be idempotent.
	repo2, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	msg := makeTestMessage("slack", "Notified", now)
	msg.MessageType = "channel_join"
	s.Require().NoError(repo2.Insert(ctx, msg))

	got, err := repo2.QueryByID(ctx, msg.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal("channel_join", got.MessageType, "MessageType should round-trip after idempotent migration")
}

// --- Feature 048: Configurable maxMessagesPerSource ---

func (s *MessageRepoSuite) TestConstructorRejectsZeroMaxMessages() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 0)
	s.Error(err, "constructor should reject maxMessagesPerSource=0")
	s.Nil(repo)
}

func (s *MessageRepoSuite) TestConstructorRejectsNegativeMaxMessages() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, -1)
	s.Error(err, "constructor should reject negative maxMessagesPerSource")
	s.Nil(repo)
}

func (s *MessageRepoSuite) TestEvictionAtCustomThreshold() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 5)
	s.Require().NoError(err)

	ctx := context.Background()
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	var oldestID uuid.UUID
	for i := 0; i < 5; i++ {
		msg := makeTestMessage("slack", "Buffered", baseTime.Add(time.Duration(i)*time.Minute))
		if i == 0 {
			oldestID = msg.ID
		}
		s.Require().NoError(repo.Insert(ctx, msg))
	}

	// Insert 6th message — should trigger eviction of the oldest.
	msg6 := makeTestMessage("slack", "Buffered", baseTime.Add(5*time.Minute))
	s.Require().NoError(repo.Insert(ctx, msg6))

	count, err := repo.CountBySource(ctx, "slack")
	s.Require().NoError(err)
	s.Equal(5, count, "should have exactly 5 slack messages after eviction")

	results, err := repo.QueryAll(ctx)
	s.Require().NoError(err)

	foundOldest := false
	foundMsg6 := false
	for _, r := range results {
		if r.ID == oldestID {
			foundOldest = true
		}
		if r.ID == msg6.ID {
			foundMsg6 = true
		}
	}
	s.False(foundOldest, "oldest message should have been evicted")
	s.True(foundMsg6, "6th message should be present")
}

func (s *MessageRepoSuite) TestEvictionAtThresholdOne() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 1)
	s.Require().NoError(err)

	ctx := context.Background()
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	msg1 := makeTestMessage("slack", "Buffered", baseTime)
	s.Require().NoError(repo.Insert(ctx, msg1))

	msg2 := makeTestMessage("slack", "Buffered", baseTime.Add(time.Minute))
	s.Require().NoError(repo.Insert(ctx, msg2))

	count, err := repo.CountBySource(ctx, "slack")
	s.Require().NoError(err)
	s.Equal(1, count, "should have exactly 1 slack message after eviction")

	results, err := repo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Require().Len(results, 1)
	s.Equal(msg2.ID, results[0].ID, "only the latest message should remain")
}

// --- Feature 049: ErrNotFound sentinel ---

func (s *MessageRepoSuite) TestQueryByID_UnknownID_ReturnsErrNotFound() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()

	got, err := repo.QueryByID(ctx, uuid.New())
	s.ErrorIs(err, repository.ErrNotFound)
	s.Nil(got, "message should be nil for unknown ID")
}

// --- Feature: ExistsByMessageID ---

func (s *MessageRepoSuite) TestExistsByMessageIDReturnsTrueForExistingMessage() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	msg := makeTestMessage("slack", "Notified", now)
	msg.MessageID = "slack-exists-001"
	s.Require().NoError(repo.Insert(ctx, msg))

	exists, err := repo.ExistsByMessageID(ctx, "slack-exists-001")
	s.NoError(err)
	s.True(exists, "ExistsByMessageID should return true for an inserted message's MessageID")
}

// --- Feature: SourceCursor Persistence ---

func (s *MessageRepoSuite) TestSourceCursorPersisted() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	msg := makeTestMessage("slack", "Notified", now)
	msg.SourceCursor = "1711500000.000100"
	s.Require().NoError(repo.Insert(ctx, msg))

	got, err := repo.QueryByID(ctx, msg.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal("1711500000.000100", got.SourceCursor, "SourceCursor should round-trip through insert and query")
}

// --- Feature 088: MaxSourceCursor ---

func (s *MessageRepoSuite) TestMaxSourceCursorReturnsHighestCursor() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert 3 slack messages in channel "general" with different SourceCursor values.
	msg1 := makeTestMessage("slack", "Notified", now)
	msg1.SourceAccount = "workspace-1"
	msg1.Channel = "general"
	msg1.SourceCursor = "100"
	s.Require().NoError(repo.Insert(ctx, msg1))

	msg2 := makeTestMessage("slack", "Notified", now.Add(time.Second))
	msg2.SourceAccount = "workspace-1"
	msg2.Channel = "general"
	msg2.SourceCursor = "300"
	s.Require().NoError(repo.Insert(ctx, msg2))

	msg3 := makeTestMessage("slack", "Notified", now.Add(2*time.Second))
	msg3.SourceAccount = "workspace-1"
	msg3.Channel = "general"
	msg3.SourceCursor = "200"
	s.Require().NoError(repo.Insert(ctx, msg3))

	cursor, err := repo.MaxSourceCursor(ctx, "slack", "workspace-1", "general")
	s.Require().NoError(err)
	s.Equal("300", cursor, "MaxSourceCursor should return the highest cursor value")
}

func (s *MessageRepoSuite) TestMaxSourceCursorReturnsEmptyForNoMatches() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()

	cursor, err := repo.MaxSourceCursor(ctx, "slack", "workspace-1", "general")
	s.Require().NoError(err)
	s.Equal("", cursor, "MaxSourceCursor should return empty string when no matching records exist")
}

func (s *MessageRepoSuite) TestQueryByID_CancelledContext_ReturnsContextError() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := repo.QueryByID(ctx, uuid.New())
	s.Error(err, "expected error when context is cancelled")
	s.Nil(got, "message should be nil when context is cancelled")
}

// --- Feature 088: DistinctChannels ---

func (s *MessageRepoSuite) TestDistinctChannelsReturnsUniqueChannels() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert 3 slack messages: 2 in "general", 1 in "alerts", all workspace-1.
	msg1 := makeTestMessage("slack", "Notified", now)
	msg1.SourceAccount = "workspace-1"
	msg1.Channel = "general"
	s.Require().NoError(repo.Insert(ctx, msg1))

	msg2 := makeTestMessage("slack", "Buffered", now.Add(time.Second))
	msg2.SourceAccount = "workspace-1"
	msg2.Channel = "general"
	s.Require().NoError(repo.Insert(ctx, msg2))

	msg3 := makeTestMessage("slack", "Notified", now.Add(2*time.Second))
	msg3.SourceAccount = "workspace-1"
	msg3.Channel = "alerts"
	s.Require().NoError(repo.Insert(ctx, msg3))

	channels, err := repo.DistinctChannels(ctx, "slack", "workspace-1")
	s.Require().NoError(err)

	sort.Strings(channels)
	s.Equal([]string{"alerts", "general"}, channels, "should return exactly the two distinct channels")
}

func (s *MessageRepoSuite) TestDistinctChannelsReturnsEmptyForNoMatches() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()

	channels, err := repo.DistinctChannels(ctx, "slack", "workspace-1")
	s.Require().NoError(err)
	s.Empty(channels, "should return empty result when no messages match")
}

// --- Feature: QueryFiltered ---

func (s *MessageRepoSuite) TestQueryFilteredByStatus() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert 2 Notified messages and 1 Ignored message.
	msg1 := makeTestMessage("slack", "Notified", now)
	s.Require().NoError(repo.Insert(ctx, msg1))

	msg2 := makeTestMessage("slack", "Notified", now.Add(time.Second))
	s.Require().NoError(repo.Insert(ctx, msg2))

	msg3 := makeTestMessage("email", "Ignored", now.Add(2*time.Second))
	s.Require().NoError(repo.Insert(ctx, msg3))

	results, total, err := repo.QueryFiltered(ctx, repository.MessageFilter{
		Status: "Notified",
		Limit:  50,
	})
	s.Require().NoError(err)
	s.Equal(2, total, "total count should reflect matching messages before pagination")
	s.Require().Len(results, 2, "should return exactly 2 Notified messages")

	for _, r := range results {
		s.Equal("Notified", r.Status, "all returned messages should have status Notified")
	}
}

// --- DeleteBySourceAccount ---

func (s *MessageRepoSuite) TestDeleteBySourceAccount_DeletesMatching() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	// Insert 2 slack/T123 messages and 1 email/user@test.com message.
	slack1 := makeTestMessage("slack", "Notified", now)
	slack1.SourceAccount = "T123"
	s.Require().NoError(repo.Insert(ctx, slack1))

	slack2 := makeTestMessage("slack", "Buffered", now.Add(time.Second))
	slack2.SourceAccount = "T123"
	s.Require().NoError(repo.Insert(ctx, slack2))

	email1 := makeTestMessage("email", "Notified", now.Add(2*time.Second))
	email1.SourceAccount = "user@test.com"
	s.Require().NoError(repo.Insert(ctx, email1))

	// Delete slack/T123 messages.
	count, err := repo.DeleteBySourceAccount(ctx, "slack", "T123")
	s.Require().NoError(err)
	s.Equal(int64(2), count, "should report 2 rows deleted")

	// Only the email message should remain.
	remaining, err := repo.QueryAll(ctx)
	s.Require().NoError(err)
	s.Require().Len(remaining, 1)
	s.Equal(email1.ID, remaining[0].ID)
	s.Equal("email", remaining[0].Source)
	s.Equal("user@test.com", remaining[0].SourceAccount)
}

func (s *MessageRepoSuite) TestDeleteBySourceAccount_NoMatches() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()

	count, err := repo.DeleteBySourceAccount(ctx, "slack", "nonexistent")
	s.Require().NoError(err)
	s.Equal(int64(0), count, "should return 0 when no messages match")
}

func (s *MessageRepoSuite) TestScoringModelAndExamplesUsedRoundTrip() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	repo, err := sqlite.NewSQLiteMessageRepository(dbPath, 100)
	s.Require().NoError(err)

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	msg := makeTestMessage("slack", "Notified", now)
	msg.ScoringModel = "neural-chat"
	msg.ExamplesUsed = 3

	err = repo.Insert(ctx, msg)
	s.Require().NoError(err)

	got, err := repo.QueryByID(ctx, msg.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal("neural-chat", got.ScoringModel, "ScoringModel should round-trip through insert and query")
	s.Equal(3, got.ExamplesUsed, "ExamplesUsed should round-trip through insert and query")
}
