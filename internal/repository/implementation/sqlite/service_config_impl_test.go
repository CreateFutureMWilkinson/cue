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

type ServiceConfigSuite struct {
	suite.Suite
	repo repository.ServiceConfigRepository
	db   *sql.DB
}

func TestServiceConfig(t *testing.T) {
	suite.Run(t, new(ServiceConfigSuite))
}

func (s *ServiceConfigSuite) SetupTest() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	s.Require().NoError(err)

	_, err = db.Exec("PRAGMA journal_mode=WAL")
	s.Require().NoError(err)

	repo, err := sqlite.NewSQLiteServiceConfigRepository(db)
	s.Require().NoError(err)
	s.Require().NotNil(repo)

	s.db = db
	s.repo = repo
}

func (s *ServiceConfigSuite) TearDownTest() {
	if s.db != nil {
		s.db.Close()
	}
}

// --- Slack Account Tests ---

func (s *ServiceConfigSuite) TestSlackAccountRoundTrip() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	acct := &repository.SlackAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Token:               "xoxb-test-token-123",
		WorkspaceID:         "T0001",
		PollIntervalSeconds: 300,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertSlackAccount(ctx, acct)
	s.Require().NoError(err)

	got, err := s.repo.GetSlackAccount(ctx, acct.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal(acct.ID, got.ID)
	s.Equal(acct.Enabled, got.Enabled)
	s.Equal(acct.Token, got.Token)
	s.Equal(acct.WorkspaceID, got.WorkspaceID)
	s.Equal(acct.PollIntervalSeconds, got.PollIntervalSeconds)
	s.WithinDuration(acct.CreatedAt, got.CreatedAt, time.Second)
	s.WithinDuration(acct.UpdatedAt, got.UpdatedAt, time.Second)
}

func (s *ServiceConfigSuite) TestEmailAccountRoundTrip() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	acct := &repository.EmailAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		IMAPHost:            "imap.gmail.com",
		IMAPPort:            993,
		Username:            "user@gmail.com",
		PasswordEnv:         "CUE_EMAIL_PASSWORD",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertEmailAccount(ctx, acct)
	s.Require().NoError(err)

	got, err := s.repo.GetEmailAccount(ctx, acct.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal(acct.ID, got.ID)
	s.Equal(acct.Enabled, got.Enabled)
	s.Equal(acct.IMAPHost, got.IMAPHost)
	s.Equal(acct.IMAPPort, got.IMAPPort)
	s.Equal(acct.Username, got.Username)
	s.Equal(acct.PasswordEnv, got.PasswordEnv)
	s.Equal(acct.PollIntervalSeconds, got.PollIntervalSeconds)
	s.WithinDuration(acct.CreatedAt, got.CreatedAt, time.Second)
	s.WithinDuration(acct.UpdatedAt, got.UpdatedAt, time.Second)
}

// --- Update Tests ---

func (s *ServiceConfigSuite) TestSlackAccountUpdate() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	acct := &repository.SlackAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Token:               "xoxb-original",
		WorkspaceID:         "T0002",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertSlackAccount(ctx, acct)
	s.Require().NoError(err)

	// Update fields.
	acct.Token = "xoxb-updated"
	acct.Enabled = false
	acct.PollIntervalSeconds = 120
	acct.UpdatedAt = now.Add(time.Minute)

	err = s.repo.UpsertSlackAccount(ctx, acct)
	s.Require().NoError(err)

	got, err := s.repo.GetSlackAccount(ctx, acct.ID)
	s.Require().NoError(err)

	s.Equal("xoxb-updated", got.Token)
	s.Equal(false, got.Enabled)
	s.Equal(120, got.PollIntervalSeconds)
	s.WithinDuration(now.Add(time.Minute), got.UpdatedAt, time.Second)
	// CreatedAt should remain unchanged.
	s.WithinDuration(now, got.CreatedAt, time.Second)
}

func (s *ServiceConfigSuite) TestEmailAccountUpdate() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	acct := &repository.EmailAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		IMAPHost:            "imap.original.com",
		IMAPPort:            993,
		Username:            "original@test.com",
		PasswordEnv:         "ORIG_PASSWORD",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertEmailAccount(ctx, acct)
	s.Require().NoError(err)

	// Update fields.
	acct.IMAPHost = "imap.updated.com"
	acct.IMAPPort = 143
	acct.Enabled = false
	acct.PasswordEnv = "NEW_PASSWORD"
	acct.PollIntervalSeconds = 300
	acct.UpdatedAt = now.Add(time.Minute)

	err = s.repo.UpsertEmailAccount(ctx, acct)
	s.Require().NoError(err)

	got, err := s.repo.GetEmailAccount(ctx, acct.ID)
	s.Require().NoError(err)

	s.Equal("imap.updated.com", got.IMAPHost)
	s.Equal(143, got.IMAPPort)
	s.Equal(false, got.Enabled)
	s.Equal("NEW_PASSWORD", got.PasswordEnv)
	s.Equal(300, got.PollIntervalSeconds)
	s.WithinDuration(now.Add(time.Minute), got.UpdatedAt, time.Second)
	s.WithinDuration(now, got.CreatedAt, time.Second)
}

// --- Delete Tests ---

func (s *ServiceConfigSuite) TestSlackAccountDelete() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	acct := &repository.SlackAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Token:               "xoxb-delete-me",
		WorkspaceID:         "T0003",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertSlackAccount(ctx, acct)
	s.Require().NoError(err)

	err = s.repo.DeleteSlackAccount(ctx, acct.ID)
	s.Require().NoError(err)

	_, err = s.repo.GetSlackAccount(ctx, acct.ID)
	s.ErrorIs(err, repository.ErrNotFound)
}

func (s *ServiceConfigSuite) TestEmailAccountDelete() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	acct := &repository.EmailAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		IMAPHost:            "imap.delete.com",
		IMAPPort:            993,
		Username:            "delete@test.com",
		PasswordEnv:         "DEL_PASSWORD",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertEmailAccount(ctx, acct)
	s.Require().NoError(err)

	err = s.repo.DeleteEmailAccount(ctx, acct.ID)
	s.Require().NoError(err)

	_, err = s.repo.GetEmailAccount(ctx, acct.ID)
	s.ErrorIs(err, repository.ErrNotFound)
}

// --- Delete Unknown ID Tests ---

func (s *ServiceConfigSuite) TestDeleteUnknownSlackAccount() {
	ctx := context.Background()
	err := s.repo.DeleteSlackAccount(ctx, uuid.New())
	s.NoError(err)
}

func (s *ServiceConfigSuite) TestDeleteUnknownEmailAccount() {
	ctx := context.Background()
	err := s.repo.DeleteEmailAccount(ctx, uuid.New())
	s.NoError(err)
}

// --- Get Unknown ID Tests ---

func (s *ServiceConfigSuite) TestGetUnknownSlackAccount() {
	ctx := context.Background()
	_, err := s.repo.GetSlackAccount(ctx, uuid.New())
	s.ErrorIs(err, repository.ErrNotFound)
}

func (s *ServiceConfigSuite) TestGetUnknownEmailAccount() {
	ctx := context.Background()
	_, err := s.repo.GetEmailAccount(ctx, uuid.New())
	s.ErrorIs(err, repository.ErrNotFound)
}

// --- List Empty Tests ---

func (s *ServiceConfigSuite) TestListSlackAccountsEmpty() {
	ctx := context.Background()
	results, err := s.repo.ListSlackAccounts(ctx)
	s.Require().NoError(err)
	s.NotNil(results, "empty list should return non-nil slice")
	s.Len(results, 0)
}

func (s *ServiceConfigSuite) TestListEmailAccountsEmpty() {
	ctx := context.Background()
	results, err := s.repo.ListEmailAccounts(ctx)
	s.Require().NoError(err)
	s.NotNil(results, "empty list should return non-nil slice")
	s.Len(results, 0)
}

// --- List Multiple Tests ---

func (s *ServiceConfigSuite) TestListSlackAccountsMultiple() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	ids := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		id := uuid.New()
		ids[i] = id
		acct := &repository.SlackAccount{
			ID:                  id,
			Enabled:             true,
			Token:               "xoxb-list-" + id.String()[:8],
			WorkspaceID:         "T100" + string(rune('0'+i)),
			PollIntervalSeconds: 600,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		err := s.repo.UpsertSlackAccount(ctx, acct)
		s.Require().NoError(err)
	}

	results, err := s.repo.ListSlackAccounts(ctx)
	s.Require().NoError(err)
	s.Len(results, 3)

	// Verify all inserted IDs are present in results.
	gotIDs := make(map[uuid.UUID]bool)
	for _, r := range results {
		gotIDs[r.ID] = true
	}
	for _, id := range ids {
		s.True(gotIDs[id], "expected account %s in list results", id)
	}
}

func (s *ServiceConfigSuite) TestListEmailAccountsMultiple() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	ids := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		id := uuid.New()
		ids[i] = id
		acct := &repository.EmailAccount{
			ID:                  id,
			Enabled:             true,
			IMAPHost:            "imap.list.com",
			IMAPPort:            993,
			Username:            "user" + string(rune('0'+i)) + "@list.com",
			PasswordEnv:         "LIST_PASSWORD",
			PollIntervalSeconds: 600,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		err := s.repo.UpsertEmailAccount(ctx, acct)
		s.Require().NoError(err)
	}

	results, err := s.repo.ListEmailAccounts(ctx)
	s.Require().NoError(err)
	s.Len(results, 3)

	gotIDs := make(map[uuid.UUID]bool)
	for _, r := range results {
		gotIDs[r.ID] = true
	}
	for _, id := range ids {
		s.True(gotIDs[id], "expected account %s in list results", id)
	}
}

// --- Unique Constraint Tests ---

func (s *ServiceConfigSuite) TestSlackAccountUniqueWorkspaceID() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	acct1 := &repository.SlackAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Token:               "xoxb-first",
		WorkspaceID:         "T-DUPLICATE",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertSlackAccount(ctx, acct1)
	s.Require().NoError(err)

	// Second account with different ID but same workspace_id should fail.
	acct2 := &repository.SlackAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Token:               "xoxb-second",
		WorkspaceID:         "T-DUPLICATE",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err = s.repo.UpsertSlackAccount(ctx, acct2)
	s.Error(err, "upserting a second account with the same workspace_id should fail")
}

func (s *ServiceConfigSuite) TestEmailAccountUniqueUsername() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	acct1 := &repository.EmailAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		IMAPHost:            "imap.unique.com",
		IMAPPort:            993,
		Username:            "duplicate@unique.com",
		PasswordEnv:         "UNIQUE_PASSWORD",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertEmailAccount(ctx, acct1)
	s.Require().NoError(err)

	// Second account with different ID but same username should fail.
	acct2 := &repository.EmailAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		IMAPHost:            "imap.unique2.com",
		IMAPPort:            993,
		Username:            "duplicate@unique.com",
		PasswordEnv:         "UNIQUE_PASSWORD_2",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err = s.repo.UpsertEmailAccount(ctx, acct2)
	s.Error(err, "upserting a second account with the same username should fail")
}
