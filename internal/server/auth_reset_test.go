package server_test

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
	"github.com/CreateFutureMWilkinson/cue/internal/server"

	_ "modernc.org/sqlite"
)

type AuthResetSuite struct {
	suite.Suite
}

func TestAuthReset(t *testing.T) {
	suite.Run(t, new(AuthResetSuite))
}

func (s *AuthResetSuite) TestResetAuthDeletesAllTokens() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open DB and seed two tokens.
	db, err := sql.Open("sqlite", dbPath)
	s.Require().NoError(err)

	_, err = db.Exec("PRAGMA journal_mode=WAL")
	s.Require().NoError(err)

	repo, err := sqlite.NewSQLiteAuthTokenRepository(db)
	s.Require().NoError(err)

	now := time.Now().Truncate(time.Second)
	for i := 0; i < 2; i++ {
		err = repo.Create(context.Background(), &repository.AuthToken{
			ID:        uuid.New(),
			Label:     "test-token",
			TokenHash: uuid.New().String(),
			CreatedAt: now,
			LastSeen:  now,
		})
		s.Require().NoError(err)
	}

	// Verify tokens exist.
	tokens, err := repo.List(context.Background())
	s.Require().NoError(err)
	s.Require().Len(tokens, 2)

	db.Close()

	// Act: call ResetAuth.
	err = server.ResetAuth(dbPath)
	s.Require().NoError(err)

	// Assert: reopen and verify empty.
	db2, err := sql.Open("sqlite", dbPath)
	s.Require().NoError(err)
	defer db2.Close()

	repo2, err := sqlite.NewSQLiteAuthTokenRepository(db2)
	s.Require().NoError(err)

	tokens, err = repo2.List(context.Background())
	s.Require().NoError(err)
	s.Empty(tokens)
}

func (s *AuthResetSuite) TestResetAuthOnEmptyDB() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create the DB with schema but no tokens.
	db, err := sql.Open("sqlite", dbPath)
	s.Require().NoError(err)

	_, err = db.Exec("PRAGMA journal_mode=WAL")
	s.Require().NoError(err)

	_, err = sqlite.NewSQLiteAuthTokenRepository(db)
	s.Require().NoError(err)

	db.Close()

	// Act: ResetAuth on empty table should succeed.
	err = server.ResetAuth(dbPath)
	s.NoError(err)
}
