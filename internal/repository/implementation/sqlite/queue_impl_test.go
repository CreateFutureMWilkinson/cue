package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	sqlite "github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"

	_ "modernc.org/sqlite"
)

type QueueRepositorySuite struct {
	suite.Suite
	db   *sql.DB
	repo repository.QueueRepository
}

func TestQueueRepository(t *testing.T) {
	suite.Run(t, new(QueueRepositorySuite))
}

func (s *QueueRepositorySuite) SetupTest() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	s.Require().NoError(err)

	_, err = db.Exec("PRAGMA journal_mode=WAL")
	s.Require().NoError(err)

	_, err = db.Exec("PRAGMA foreign_keys=ON")
	s.Require().NoError(err)

	// Create messages table since ollama_queue has a FK reference to it.
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY
	)`)
	s.Require().NoError(err)

	repo, err := sqlite.NewSQLiteQueueRepository(db)
	s.Require().NoError(err)
	s.Require().NotNil(repo)

	s.db = db
	s.repo = repo
}

func (s *QueueRepositorySuite) TearDownTest() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *QueueRepositorySuite) TestConstructorReturnsNonNilRepository() {
	s.NotNil(s.repo)
}

func (s *QueueRepositorySuite) TestEnqueueInsertsEntryWithPendingStatus() {
	ctx := context.Background()
	messageID := uuid.New()

	// Insert a message row to satisfy the foreign key constraint.
	_, err := s.db.Exec("INSERT INTO messages (id) VALUES (?)", messageID.String())
	s.Require().NoError(err)

	// Enqueue the message.
	err = s.repo.Enqueue(ctx, messageID)
	s.Require().NoError(err)

	// Query the ollama_queue table directly to verify the entry.
	var id, msgID, status, enqueuedAt string
	err = s.db.QueryRow(
		"SELECT id, message_id, status, enqueued_at FROM ollama_queue WHERE message_id = ?",
		messageID.String(),
	).Scan(&id, &msgID, &status, &enqueuedAt)
	s.Require().NoError(err, "expected one row in ollama_queue")

	s.Equal(messageID.String(), msgID)
	s.Equal("pending", status)
	s.NotEmpty(enqueuedAt)

	// Verify id is a valid UUID.
	_, err = uuid.Parse(id)
	s.NoError(err, "id should be a valid UUID")
}

func (s *QueueRepositorySuite) TestDequeueOldestReturnsNilWhenEmpty() {
	ctx := context.Background()

	entry, err := s.repo.DequeueOldest(ctx)
	s.NoError(err)
	s.Nil(entry)
}

func (s *QueueRepositorySuite) TestDequeueOldestReturnsOldestPendingEntry() {
	ctx := context.Background()

	firstID := uuid.New()
	secondID := uuid.New()

	// Insert message rows to satisfy the foreign key constraint.
	_, err := s.db.Exec("INSERT INTO messages (id) VALUES (?)", firstID.String())
	s.Require().NoError(err)
	_, err = s.db.Exec("INSERT INTO messages (id) VALUES (?)", secondID.String())
	s.Require().NoError(err)

	// Enqueue first, then second.
	err = s.repo.Enqueue(ctx, firstID)
	s.Require().NoError(err)
	err = s.repo.Enqueue(ctx, secondID)
	s.Require().NoError(err)

	// Dequeue should return the first (oldest) entry.
	entry, err := s.repo.DequeueOldest(ctx)
	s.Require().NoError(err)
	s.Require().NotNil(entry)

	s.Equal(firstID, entry.MessageID)
	s.Equal("processing", entry.Status)
	s.NotEqual(uuid.Nil, entry.ID, "entry ID should be a valid non-nil UUID")
	s.False(entry.EnqueuedAt.IsZero(), "EnqueuedAt should not be zero")
}

func (s *QueueRepositorySuite) TestDequeueOldestSkipsNonPendingEntries() {
	ctx := context.Background()

	firstID := uuid.New()
	secondID := uuid.New()

	// Insert message rows to satisfy the foreign key constraint.
	_, err := s.db.Exec("INSERT INTO messages (id) VALUES (?)", firstID.String())
	s.Require().NoError(err)
	_, err = s.db.Exec("INSERT INTO messages (id) VALUES (?)", secondID.String())
	s.Require().NoError(err)

	// Enqueue first, then second.
	err = s.repo.Enqueue(ctx, firstID)
	s.Require().NoError(err)
	err = s.repo.Enqueue(ctx, secondID)
	s.Require().NoError(err)

	// Dequeue once — first message becomes "processing".
	_, err = s.repo.DequeueOldest(ctx)
	s.Require().NoError(err)

	// Dequeue again — should skip the first (now "processing") and return the second.
	entry, err := s.repo.DequeueOldest(ctx)
	s.Require().NoError(err)
	s.Require().NotNil(entry)

	s.Equal(secondID, entry.MessageID)
}

func (s *QueueRepositorySuite) TestMarkDoneSetsStatusToDone() {
	ctx := context.Background()
	messageID := uuid.New()

	// Insert a message row to satisfy the foreign key constraint.
	_, err := s.db.Exec("INSERT INTO messages (id) VALUES (?)", messageID.String())
	s.Require().NoError(err)

	// Enqueue, then dequeue to move to "processing".
	err = s.repo.Enqueue(ctx, messageID)
	s.Require().NoError(err)

	entry, err := s.repo.DequeueOldest(ctx)
	s.Require().NoError(err)
	s.Require().NotNil(entry)

	// Mark done.
	err = s.repo.MarkDone(ctx, entry.ID)
	s.Require().NoError(err)

	// Verify status is "done" directly in the database.
	var status string
	err = s.db.QueryRow(
		"SELECT status FROM ollama_queue WHERE id = ?", entry.ID.String(),
	).Scan(&status)
	s.Require().NoError(err)
	s.Equal("done", status)
}

func (s *QueueRepositorySuite) TestMarkFailedSetsStatusToFailed() {
	ctx := context.Background()
	messageID := uuid.New()

	// Insert a message row to satisfy the foreign key constraint.
	_, err := s.db.Exec("INSERT INTO messages (id) VALUES (?)", messageID.String())
	s.Require().NoError(err)

	// Enqueue, then dequeue to move to "processing".
	err = s.repo.Enqueue(ctx, messageID)
	s.Require().NoError(err)

	entry, err := s.repo.DequeueOldest(ctx)
	s.Require().NoError(err)
	s.Require().NotNil(entry)

	// Mark failed.
	err = s.repo.MarkFailed(ctx, entry.ID)
	s.Require().NoError(err)

	// Verify status is "failed" directly in the database.
	var status string
	err = s.db.QueryRow(
		"SELECT status FROM ollama_queue WHERE id = ?", entry.ID.String(),
	).Scan(&status)
	s.Require().NoError(err)
	s.Equal("failed", status)
}

func (s *QueueRepositorySuite) TestMarkDoneNonExistentIDReturnsError() {
	ctx := context.Background()

	err := s.repo.MarkDone(ctx, uuid.New())
	s.Error(err)
}

func (s *QueueRepositorySuite) TestPendingCountReturnsCorrectCount() {
	ctx := context.Background()

	// Insert 3 message rows and enqueue them.
	for i := 0; i < 3; i++ {
		msgID := uuid.New()
		_, err := s.db.Exec("INSERT INTO messages (id) VALUES (?)", msgID.String())
		s.Require().NoError(err)
		err = s.repo.Enqueue(ctx, msgID)
		s.Require().NoError(err)
	}

	// All 3 should be pending.
	count, err := s.repo.PendingCount(ctx)
	s.Require().NoError(err)
	s.Equal(3, count)

	// Dequeue one (becomes "processing").
	_, err = s.repo.DequeueOldest(ctx)
	s.Require().NoError(err)

	// Now only 2 should be pending.
	count, err = s.repo.PendingCount(ctx)
	s.Require().NoError(err)
	s.Equal(2, count)
}

func (s *QueueRepositorySuite) TestPurgeCompletedRemovesDoneAndFailed() {
	ctx := context.Background()

	// Insert 3 message rows and enqueue them.
	ids := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		ids[i] = uuid.New()
		_, err := s.db.Exec("INSERT INTO messages (id) VALUES (?)", ids[i].String())
		s.Require().NoError(err)
		err = s.repo.Enqueue(ctx, ids[i])
		s.Require().NoError(err)
	}

	// Dequeue all 3 (all become "processing").
	entries := make([]*repository.QueueEntry, 3)
	for i := 0; i < 3; i++ {
		entry, err := s.repo.DequeueOldest(ctx)
		s.Require().NoError(err)
		s.Require().NotNil(entry)
		entries[i] = entry
	}

	// MarkDone on first, MarkFailed on second, third stays "processing".
	err := s.repo.MarkDone(ctx, entries[0].ID)
	s.Require().NoError(err)
	err = s.repo.MarkFailed(ctx, entries[1].ID)
	s.Require().NoError(err)

	// Purge completed (done + failed).
	err = s.repo.PurgeCompleted(ctx)
	s.Require().NoError(err)

	// Only the "processing" entry should remain.
	var rowCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM ollama_queue").Scan(&rowCount)
	s.Require().NoError(err)
	s.Equal(1, rowCount)
}

func (s *QueueRepositorySuite) TestPurgeOlderThanRemovesOldEntries() {
	ctx := context.Background()

	// Insert first message and enqueue it.
	oldMsgID := uuid.New()
	_, err := s.db.Exec("INSERT INTO messages (id) VALUES (?)", oldMsgID.String())
	s.Require().NoError(err)
	err = s.repo.Enqueue(ctx, oldMsgID)
	s.Require().NoError(err)

	// Backdate the enqueued_at to well in the past.
	_, err = s.db.Exec(
		"UPDATE ollama_queue SET enqueued_at = ? WHERE message_id = ?",
		"2020-01-01T00:00:00Z", oldMsgID.String(),
	)
	s.Require().NoError(err)

	// Insert second message and enqueue it (current timestamp).
	recentMsgID := uuid.New()
	_, err = s.db.Exec("INSERT INTO messages (id) VALUES (?)", recentMsgID.String())
	s.Require().NoError(err)
	err = s.repo.Enqueue(ctx, recentMsgID)
	s.Require().NoError(err)

	// Purge entries older than 1 hour ago.
	cutoff := time.Now().Add(-1 * time.Hour)
	err = s.repo.PurgeOlderThan(ctx, cutoff)
	s.Require().NoError(err)

	// Only the recent entry should remain.
	var rowCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM ollama_queue").Scan(&rowCount)
	s.Require().NoError(err)
	s.Equal(1, rowCount)
}

func (s *QueueRepositorySuite) TestPurgeAllRemovesAllEntries() {
	ctx := context.Background()

	// Insert 3 message rows and enqueue them.
	for i := 0; i < 3; i++ {
		msgID := uuid.New()
		_, err := s.db.Exec("INSERT INTO messages (id) VALUES (?)", msgID.String())
		s.Require().NoError(err)
		err = s.repo.Enqueue(ctx, msgID)
		s.Require().NoError(err)
	}

	// Purge all.
	err := s.repo.PurgeAll(ctx)
	s.Require().NoError(err)

	// No rows should remain.
	var rowCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM ollama_queue").Scan(&rowCount)
	s.Require().NoError(err)
	s.Equal(0, rowCount)
}

func (s *QueueRepositorySuite) TestResetProcessingResetsTopending() {
	ctx := context.Background()

	// Insert 2 message rows, enqueue, and dequeue both (both become "processing").
	for i := 0; i < 2; i++ {
		msgID := uuid.New()
		_, err := s.db.Exec("INSERT INTO messages (id) VALUES (?)", msgID.String())
		s.Require().NoError(err)
		err = s.repo.Enqueue(ctx, msgID)
		s.Require().NoError(err)
	}
	_, err := s.repo.DequeueOldest(ctx)
	s.Require().NoError(err)
	_, err = s.repo.DequeueOldest(ctx)
	s.Require().NoError(err)

	// Reset processing.
	count, err := s.repo.ResetProcessing(ctx)
	s.Require().NoError(err)
	s.Equal(int64(2), count)

	// Both should now be "pending".
	var pendingCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM ollama_queue WHERE status = 'pending'").Scan(&pendingCount)
	s.Require().NoError(err)
	s.Equal(2, pendingCount)
}

func (s *QueueRepositorySuite) TestResetProcessingReturnsZeroWhenNoneProcessing() {
	ctx := context.Background()

	// Insert a message and enqueue it (stays "pending").
	msgID := uuid.New()
	_, err := s.db.Exec("INSERT INTO messages (id) VALUES (?)", msgID.String())
	s.Require().NoError(err)
	err = s.repo.Enqueue(ctx, msgID)
	s.Require().NoError(err)

	// Reset processing — nothing is "processing".
	count, err := s.repo.ResetProcessing(ctx)
	s.Require().NoError(err)
	s.Equal(int64(0), count)
}

// Compile-time interface satisfaction check.
var _ repository.QueueRepository = (*sqlite.SQLiteQueueRepository)(nil)
