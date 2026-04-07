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
	"github.com/CreateFutureMWilkinson/cue/internal/secret"

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

	keyPath := filepath.Join(tmpDir, "test.key")
	enc, err := secret.NewKeyFileEncryptor(keyPath)
	s.Require().NoError(err)

	repo, err := sqlite.NewSQLiteServiceConfigRepository(db, enc)
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
		Password:            "my-secret-password",
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
	s.Equal(acct.Password, got.Password)
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
		Password:            "orig-password",
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
	acct.Password = "new-password"
	acct.PollIntervalSeconds = 300
	acct.UpdatedAt = now.Add(time.Minute)

	err = s.repo.UpsertEmailAccount(ctx, acct)
	s.Require().NoError(err)

	got, err := s.repo.GetEmailAccount(ctx, acct.ID)
	s.Require().NoError(err)

	s.Equal("imap.updated.com", got.IMAPHost)
	s.Equal(143, got.IMAPPort)
	s.Equal(false, got.Enabled)
	s.Equal("new-password", got.Password)
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
		Password:            "del-password",
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
			Password:            "list-password",
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

func (s *ServiceConfigSuite) TestSlackAccountUsernameRoundTrip() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	acct := &repository.SlackAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Token:               "xoxb-username-test",
		WorkspaceID:         "T-USERNAME",
		Username:            "testuser",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertSlackAccount(ctx, acct)
	s.Require().NoError(err)

	got, err := s.repo.GetSlackAccount(ctx, acct.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal("testuser", got.Username, "Username must survive upsert/get round-trip")
}

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

func (s *ServiceConfigSuite) TestEmailAccountEncryptionFieldRoundTrip() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	acct := &repository.EmailAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		IMAPHost:            "imap.encryption.com",
		IMAPPort:            993,
		Username:            "encrypt@test.com",
		Password:            "enc-password",
		Encryption:          "starttls",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertEmailAccount(ctx, acct)
	s.Require().NoError(err)

	got, err := s.repo.GetEmailAccount(ctx, acct.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal("starttls", got.Encryption)
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
		Password:            "unique-password",
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
		Password:            "unique-password-2",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err = s.repo.UpsertEmailAccount(ctx, acct2)
	s.Error(err, "upserting a second account with the same username should fail")
}

// --- Slack Token Encryption Verification ---

func (s *ServiceConfigSuite) TestSlackTokenEncryptedAtRest() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	token := "xoxb-plaintext-token-visible"

	acct := &repository.SlackAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Token:               token,
		WorkspaceID:         "T-ENC-SLACK",
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertSlackAccount(ctx, acct)
	s.Require().NoError(err)

	// Read the raw value from SQLite and verify it is NOT the plaintext token.
	var rawToken []byte
	err = s.db.QueryRowContext(ctx,
		"SELECT token_encrypted FROM slack_accounts WHERE id = ?",
		acct.ID.String(),
	).Scan(&rawToken)
	s.Require().NoError(err)
	s.NotEqual([]byte(token), rawToken, "token must be encrypted at rest")
}

// --- Email Password Encryption Verification ---

func (s *ServiceConfigSuite) TestEmailPasswordEncryptedAtRest() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	password := "super-secret-email-password"

	acct := &repository.EmailAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		IMAPHost:            "imap.enc.com",
		IMAPPort:            993,
		Username:            "enc@test.com",
		Password:            password,
		PollIntervalSeconds: 600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertEmailAccount(ctx, acct)
	s.Require().NoError(err)

	var rawPassword []byte
	err = s.db.QueryRowContext(ctx,
		"SELECT password_encrypted FROM email_accounts WHERE id = ?",
		acct.ID.String(),
	).Scan(&rawPassword)
	s.Require().NoError(err)
	s.NotEqual([]byte(password), rawPassword, "password must be encrypted at rest")
}

// --- Calendar Account CRUD Tests ---

func (s *ServiceConfigSuite) TestCalendarAccountRoundTrip() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	acct := &repository.CalendarAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Name:                "Work Calendar",
		ICSURL:              "https://calendar.example.com/feed.ics",
		PollIntervalSeconds: 3600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertCalendarAccount(ctx, acct)
	s.Require().NoError(err)

	got, err := s.repo.GetCalendarAccount(ctx, acct.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal(acct.ID, got.ID)
	s.Equal(acct.Enabled, got.Enabled)
	s.Equal(acct.Name, got.Name)
	s.Equal(acct.ICSURL, got.ICSURL)
	s.Equal(acct.PollIntervalSeconds, got.PollIntervalSeconds)
	s.WithinDuration(acct.CreatedAt, got.CreatedAt, time.Second)
	s.WithinDuration(acct.UpdatedAt, got.UpdatedAt, time.Second)
}

func (s *ServiceConfigSuite) TestCalendarAccountUpdate() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	acct := &repository.CalendarAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Name:                "Original Calendar",
		ICSURL:              "https://calendar.example.com/orig.ics",
		PollIntervalSeconds: 3600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertCalendarAccount(ctx, acct)
	s.Require().NoError(err)

	acct.Name = "Updated Calendar"
	acct.ICSURL = "https://calendar.example.com/updated.ics"
	acct.Enabled = false
	acct.PollIntervalSeconds = 1800
	acct.UpdatedAt = now.Add(time.Minute)

	err = s.repo.UpsertCalendarAccount(ctx, acct)
	s.Require().NoError(err)

	got, err := s.repo.GetCalendarAccount(ctx, acct.ID)
	s.Require().NoError(err)

	s.Equal("Updated Calendar", got.Name)
	s.Equal("https://calendar.example.com/updated.ics", got.ICSURL)
	s.False(got.Enabled)
	s.Equal(1800, got.PollIntervalSeconds)
	s.WithinDuration(now.Add(time.Minute), got.UpdatedAt, time.Second)
	s.WithinDuration(now, got.CreatedAt, time.Second)
}

func (s *ServiceConfigSuite) TestCalendarAccountDelete() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	acct := &repository.CalendarAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Name:                "Delete Me Calendar",
		ICSURL:              "https://calendar.example.com/delete.ics",
		PollIntervalSeconds: 3600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertCalendarAccount(ctx, acct)
	s.Require().NoError(err)

	err = s.repo.DeleteCalendarAccount(ctx, acct.ID)
	s.Require().NoError(err)

	_, err = s.repo.GetCalendarAccount(ctx, acct.ID)
	s.ErrorIs(err, repository.ErrNotFound)
}

func (s *ServiceConfigSuite) TestGetUnknownCalendarAccount() {
	ctx := context.Background()
	_, err := s.repo.GetCalendarAccount(ctx, uuid.New())
	s.ErrorIs(err, repository.ErrNotFound)
}

func (s *ServiceConfigSuite) TestDeleteUnknownCalendarAccount() {
	ctx := context.Background()
	err := s.repo.DeleteCalendarAccount(ctx, uuid.New())
	s.NoError(err)
}

func (s *ServiceConfigSuite) TestListCalendarAccountsEmpty() {
	ctx := context.Background()
	results, err := s.repo.ListCalendarAccounts(ctx)
	s.Require().NoError(err)
	s.NotNil(results, "empty list should return non-nil slice")
	s.Len(results, 0)
}

func (s *ServiceConfigSuite) TestListCalendarAccountsMultiple() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	ids := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		id := uuid.New()
		ids[i] = id
		acct := &repository.CalendarAccount{
			ID:                  id,
			Enabled:             true,
			Name:                "Calendar " + string(rune('A'+i)),
			ICSURL:              "https://calendar.example.com/" + id.String()[:8] + ".ics",
			PollIntervalSeconds: 3600,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		err := s.repo.UpsertCalendarAccount(ctx, acct)
		s.Require().NoError(err)
	}

	results, err := s.repo.ListCalendarAccounts(ctx)
	s.Require().NoError(err)
	s.Len(results, 3)

	gotIDs := make(map[uuid.UUID]bool)
	for _, r := range results {
		gotIDs[r.ID] = true
	}
	for _, id := range ids {
		s.True(gotIDs[id], "expected calendar account %s in list results", id)
	}
}

func (s *ServiceConfigSuite) TestCalendarAccountUniqueName() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	acct1 := &repository.CalendarAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Name:                "Duplicate Name",
		ICSURL:              "https://calendar.example.com/first.ics",
		PollIntervalSeconds: 3600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertCalendarAccount(ctx, acct1)
	s.Require().NoError(err)

	acct2 := &repository.CalendarAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Name:                "Duplicate Name",
		ICSURL:              "https://calendar.example.com/second.ics",
		PollIntervalSeconds: 3600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err = s.repo.UpsertCalendarAccount(ctx, acct2)
	s.Error(err, "upserting a second calendar account with the same name should fail")
}

// --- Calendar ICS URL Encryption Verification ---

func (s *ServiceConfigSuite) TestCalendarICSURLEncryptedAtRest() {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	icsURL := "https://calendar.example.com/secret-feed.ics"

	acct := &repository.CalendarAccount{
		ID:                  uuid.New(),
		Enabled:             true,
		Name:                "Encrypted Calendar",
		ICSURL:              icsURL,
		PollIntervalSeconds: 3600,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	err := s.repo.UpsertCalendarAccount(ctx, acct)
	s.Require().NoError(err)

	var rawURL []byte
	err = s.db.QueryRowContext(ctx,
		"SELECT ics_url_encrypted FROM calendar_accounts WHERE id = ?",
		acct.ID.String(),
	).Scan(&rawURL)
	s.Require().NoError(err)
	s.NotEqual([]byte(icsURL), rawURL, "ICS URL must be encrypted at rest")
}
