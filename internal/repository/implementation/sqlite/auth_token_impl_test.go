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

type AuthTokenSuite struct {
	suite.Suite
	repo repository.AuthTokenRepository
	db   *sql.DB
}

func TestAuthToken(t *testing.T) {
	suite.Run(t, new(AuthTokenSuite))
}

func (s *AuthTokenSuite) SetupTest() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	s.Require().NoError(err)

	_, err = db.Exec("PRAGMA journal_mode=WAL")
	s.Require().NoError(err)

	repo, err := sqlite.NewSQLiteAuthTokenRepository(db)
	s.Require().NoError(err)
	s.Require().NotNil(repo)

	s.db = db
	s.repo = repo
}

func (s *AuthTokenSuite) TearDownTest() {
	if s.db != nil {
		s.db.Close()
	}
}

// validToken returns a valid AuthToken for testing.
func (s *AuthTokenSuite) validToken(label, hash string) *repository.AuthToken {
	now := time.Now().Truncate(time.Second).UTC()
	return &repository.AuthToken{
		ID:        uuid.New(),
		Label:     label,
		TokenHash: hash,
		CreatedAt: now,
		LastSeen:  now,
		Revoked:   false,
	}
}

// --- Behavior 1: Create + LookupByHash ---

func (s *AuthTokenSuite) TestCreateAndLookupByHash() {
	ctx := context.Background()
	token := s.validToken("my-laptop", "abc123def456")

	err := s.repo.Create(ctx, token)
	s.Require().NoError(err)

	got, err := s.repo.LookupByHash(ctx, "abc123def456")
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal(token.ID, got.ID)
	s.Equal(token.Label, got.Label)
	s.Equal(token.TokenHash, got.TokenHash)
	s.Equal(token.Revoked, got.Revoked)
	s.WithinDuration(token.CreatedAt, got.CreatedAt, time.Second)
	s.WithinDuration(token.LastSeen, got.LastSeen, time.Second)
}

// --- Behavior 2: List ---

func (s *AuthTokenSuite) TestListMultipleTokens() {
	ctx := context.Background()

	t1 := s.validToken("laptop", "hash-aaa")
	t2 := s.validToken("phone", "hash-bbb")
	t3 := s.validToken("tablet", "hash-ccc")

	s.Require().NoError(s.repo.Create(ctx, t1))
	s.Require().NoError(s.repo.Create(ctx, t2))
	s.Require().NoError(s.repo.Create(ctx, t3))

	tokens, err := s.repo.List(ctx)
	s.Require().NoError(err)
	s.Require().Len(tokens, 3)

	// Verify all created tokens are present (order not guaranteed).
	ids := make(map[uuid.UUID]bool)
	for _, tok := range tokens {
		ids[tok.ID] = true
	}
	s.True(ids[t1.ID], "token 1 should be in list")
	s.True(ids[t2.ID], "token 2 should be in list")
	s.True(ids[t3.ID], "token 3 should be in list")
}

// --- Behavior 3: Revoke ---

func (s *AuthTokenSuite) TestRevoke() {
	ctx := context.Background()
	token := s.validToken("revoke-me", "hash-revoke")

	s.Require().NoError(s.repo.Create(ctx, token))

	err := s.repo.Revoke(ctx, token.ID)
	s.Require().NoError(err)

	got, err := s.repo.LookupByHash(ctx, "hash-revoke")
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.True(got.Revoked, "token should be revoked after Revoke()")
}

// --- Behavior 4: UpdateLabel ---

func (s *AuthTokenSuite) TestUpdateLabel() {
	ctx := context.Background()
	token := s.validToken("old-label", "hash-label")

	s.Require().NoError(s.repo.Create(ctx, token))

	err := s.repo.UpdateLabel(ctx, token.ID, "new-label")
	s.Require().NoError(err)

	got, err := s.repo.LookupByHash(ctx, "hash-label")
	s.Require().NoError(err)
	s.Require().NotNil(got)
	s.Equal("new-label", got.Label)
}

// --- Behavior 5: DeleteAll ---

func (s *AuthTokenSuite) TestDeleteAll() {
	ctx := context.Background()

	t1 := s.validToken("one", "hash-del1")
	t2 := s.validToken("two", "hash-del2")

	s.Require().NoError(s.repo.Create(ctx, t1))
	s.Require().NoError(s.repo.Create(ctx, t2))

	err := s.repo.DeleteAll(ctx)
	s.Require().NoError(err)

	tokens, err := s.repo.List(ctx)
	s.Require().NoError(err)
	s.Empty(tokens, "list should be empty after DeleteAll")
}

// --- Behavior 6: LookupByHash not found ---

func (s *AuthTokenSuite) TestLookupByHashNotFound() {
	ctx := context.Background()

	got, err := s.repo.LookupByHash(ctx, "nonexistent-hash")
	s.ErrorIs(err, repository.ErrNotFound)
	s.Nil(got)
}

// --- Constructor: table creation ---

func (s *AuthTokenSuite) TestNewCreatesTable() {
	var tableName string
	err := s.db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='auth_tokens'",
	).Scan(&tableName)
	s.Require().NoError(err)
	s.Equal("auth_tokens", tableName)
}
