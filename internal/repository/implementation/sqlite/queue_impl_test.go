package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

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

// Compile-time interface satisfaction check.
var _ repository.QueueRepository = (*sqlite.SQLiteQueueRepository)(nil)
