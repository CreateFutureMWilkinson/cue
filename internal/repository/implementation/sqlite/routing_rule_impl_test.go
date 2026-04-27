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

type RoutingRuleSQLiteSuite struct {
	suite.Suite
	repo repository.RoutingRuleRepository
	db   *sql.DB
}

func TestRoutingRuleSQLite(t *testing.T) {
	suite.Run(t, new(RoutingRuleSQLiteSuite))
}

func (s *RoutingRuleSQLiteSuite) SetupTest() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	s.Require().NoError(err)

	_, err = db.Exec("PRAGMA journal_mode=WAL")
	s.Require().NoError(err)

	repo, err := sqlite.NewSQLiteRoutingRuleRepository(db)
	s.Require().NoError(err)
	s.Require().NotNil(repo)

	s.db = db
	s.repo = repo
}

func (s *RoutingRuleSQLiteSuite) TearDownTest() {
	if s.db != nil {
		s.db.Close()
	}
}

// validRule returns a valid routing rule for testing.
func (s *RoutingRuleSQLiteSuite) validRule() *repository.RoutingRule {
	now := time.Now().Truncate(time.Second)
	return &repository.RoutingRule{
		ID:             uuid.New(),
		Priority:       10,
		SourceType:     "slack",
		ChannelPattern: "^general$",
		Action:         "notified",
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (s *RoutingRuleSQLiteSuite) TestNewCreatesTable() {
	// Constructor already called in SetupTest; verify the table exists.
	var tableName string
	err := s.db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='routing_rules'",
	).Scan(&tableName)
	s.Require().NoError(err)
	s.Equal("routing_rules", tableName)
}

func (s *RoutingRuleSQLiteSuite) TestUpsertRuleInsert() {
	ctx := context.Background()
	rule := s.validRule()

	err := s.repo.UpsertRule(ctx, rule)
	s.Require().NoError(err)

	// Read back with raw SQL to verify all fields persisted correctly.
	var (
		idStr            string
		name             string
		priority         int
		sourceType       string
		sourceAccountStr sql.NullString
		channelPattern   string
		contentPattern   string
		messageType      string
		action           string
		enabled          int
		createdAtStr     string
		updatedAtStr     string
	)
	err = s.db.QueryRowContext(ctx,
		"SELECT id, name, priority, source_type, source_account, channel_pattern, content_pattern, message_type, action, enabled, created_at, updated_at FROM routing_rules WHERE id = ?",
		rule.ID.String(),
	).Scan(&idStr, &name, &priority, &sourceType, &sourceAccountStr, &channelPattern, &contentPattern, &messageType, &action, &enabled, &createdAtStr, &updatedAtStr)
	s.Require().NoError(err)

	s.Equal(rule.ID.String(), idStr)
	s.Equal(rule.Name, name)
	s.Equal(rule.Priority, priority)
	s.Equal(rule.SourceType, sourceType)
	s.False(sourceAccountStr.Valid, "source_account should be NULL")
	s.Equal(rule.ChannelPattern, channelPattern)
	s.Equal(rule.ContentPattern, contentPattern)
	s.Equal(rule.MessageType, messageType)
	s.Equal(rule.Action, action)
	s.Equal(1, enabled)

	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	s.Require().NoError(err)
	s.WithinDuration(rule.CreatedAt, createdAt, time.Second)

	updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
	s.Require().NoError(err)
	s.WithinDuration(rule.UpdatedAt, updatedAt, time.Second)
}

func (s *RoutingRuleSQLiteSuite) TestUpsertRuleUpdate() {
	ctx := context.Background()
	rule := s.validRule()

	err := s.repo.UpsertRule(ctx, rule)
	s.Require().NoError(err)

	// Modify fields and upsert again with the same ID.
	rule.Priority = 99
	rule.ChannelPattern = "^random$"
	rule.Action = "ignored"
	rule.Enabled = false
	rule.UpdatedAt = rule.UpdatedAt.Add(time.Minute)

	err = s.repo.UpsertRule(ctx, rule)
	s.Require().NoError(err)

	// Read back with raw SQL to verify update took effect.
	var (
		priority       int
		channelPattern string
		action         string
		enabled        int
	)
	err = s.db.QueryRowContext(ctx,
		"SELECT priority, channel_pattern, action, enabled FROM routing_rules WHERE id = ?",
		rule.ID.String(),
	).Scan(&priority, &channelPattern, &action, &enabled)
	s.Require().NoError(err)

	s.Equal(99, priority)
	s.Equal("^random$", channelPattern)
	s.Equal("ignored", action)
	s.Equal(0, enabled)
}

func (s *RoutingRuleSQLiteSuite) TestUpsertRuleValidationError() {
	ctx := context.Background()
	rule := s.validRule()
	rule.SourceType = "ftp" // invalid source type

	err := s.repo.UpsertRule(ctx, rule)
	s.ErrorIs(err, repository.ErrInvalidRoutingRule)
}

func (s *RoutingRuleSQLiteSuite) TestGetRuleFound() {
	ctx := context.Background()
	rule := s.validRule()

	err := s.repo.UpsertRule(ctx, rule)
	s.Require().NoError(err)

	got, err := s.repo.GetRule(ctx, rule.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got)

	s.Equal(rule.ID, got.ID)
	s.Equal(rule.Priority, got.Priority)
	s.Equal(rule.SourceType, got.SourceType)
	s.Equal(rule.ChannelPattern, got.ChannelPattern)
	s.Equal(rule.ContentPattern, got.ContentPattern)
	s.Equal(rule.MessageType, got.MessageType)
	s.Equal(rule.Action, got.Action)
	s.Equal(rule.Enabled, got.Enabled)
	s.False(got.CreatedAt.IsZero(), "CreatedAt should not be zero")
	s.False(got.UpdatedAt.IsZero(), "UpdatedAt should not be zero")
}

func (s *RoutingRuleSQLiteSuite) TestGetRuleNotFound() {
	ctx := context.Background()

	got, err := s.repo.GetRule(ctx, uuid.New())
	s.ErrorIs(err, repository.ErrNotFound)
	s.Nil(got)
}

// --- Behavior 5: ListRules ---

func (s *RoutingRuleSQLiteSuite) TestListRulesEmpty() {
	ctx := context.Background()

	// Clear seeded defaults so we can verify empty-list behavior.
	_, err := s.db.Exec("DELETE FROM routing_rules")
	s.Require().NoError(err)

	rules, err := s.repo.ListRules(ctx)
	s.Require().NoError(err)
	s.NotNil(rules, "ListRules should return non-nil slice when empty")
	s.Empty(rules)
}

func (s *RoutingRuleSQLiteSuite) TestListRulesSortedByPriority() {
	ctx := context.Background()

	// Clear seeded defaults so we test only the rules we insert.
	_, err := s.db.Exec("DELETE FROM routing_rules")
	s.Require().NoError(err)

	r1 := s.validRule()
	r1.Priority = 5
	r2 := s.validRule()
	r2.Priority = 1
	r3 := s.validRule()
	r3.Priority = 10

	s.Require().NoError(s.repo.UpsertRule(ctx, r1))
	s.Require().NoError(s.repo.UpsertRule(ctx, r2))
	s.Require().NoError(s.repo.UpsertRule(ctx, r3))

	rules, err := s.repo.ListRules(ctx)
	s.Require().NoError(err)
	s.Require().Len(rules, 3)

	s.Equal(1, rules[0].Priority)
	s.Equal(5, rules[1].Priority)
	s.Equal(10, rules[2].Priority)
}

// --- Behavior 6: ListRulesBySourceType ---

func (s *RoutingRuleSQLiteSuite) TestListRulesBySourceTypeFiltered() {
	ctx := context.Background()

	// Clear seeded defaults so we test only the rules we insert.
	_, err := s.db.Exec("DELETE FROM routing_rules")
	s.Require().NoError(err)

	slack1 := s.validRule()
	slack1.SourceType = "slack"
	slack1.Priority = 5

	slack2 := s.validRule()
	slack2.SourceType = "slack"
	slack2.Priority = 1

	email1 := s.validRule()
	email1.SourceType = "email"
	email1.Priority = 3

	s.Require().NoError(s.repo.UpsertRule(ctx, slack1))
	s.Require().NoError(s.repo.UpsertRule(ctx, slack2))
	s.Require().NoError(s.repo.UpsertRule(ctx, email1))

	rules, err := s.repo.ListRulesBySourceType(ctx, "slack")
	s.Require().NoError(err)
	s.Require().Len(rules, 2)

	s.Equal("slack", rules[0].SourceType)
	s.Equal("slack", rules[1].SourceType)
	s.Equal(1, rules[0].Priority, "results should be sorted by priority ascending")
	s.Equal(5, rules[1].Priority)
}

func (s *RoutingRuleSQLiteSuite) TestListRulesBySourceTypeEmpty() {
	ctx := context.Background()

	rules, err := s.repo.ListRulesBySourceType(ctx, "email")
	s.Require().NoError(err)
	s.NotNil(rules, "ListRulesBySourceType should return non-nil slice when empty")
	s.Empty(rules)
}

// --- Behavior: ListRulesBySourceAccount ---

func (s *RoutingRuleSQLiteSuite) TestListRulesBySourceAccountFiltered() {
	ctx := context.Background()

	// Clear seeded defaults
	_, err := s.db.Exec("DELETE FROM routing_rules")
	s.Require().NoError(err)

	accountID := uuid.New()
	otherAccountID := uuid.New()

	r1 := s.validRule()
	r1.SourceAccount = &accountID
	r1.Priority = 1

	r2 := s.validRule()
	r2.SourceAccount = &otherAccountID
	r2.Priority = 2

	r3 := s.validRule()
	r3.SourceAccount = &accountID
	r3.Priority = 3

	s.Require().NoError(s.repo.UpsertRule(ctx, r1))
	s.Require().NoError(s.repo.UpsertRule(ctx, r2))
	s.Require().NoError(s.repo.UpsertRule(ctx, r3))

	rules, err := s.repo.ListRulesBySourceAccount(ctx, accountID)
	s.Require().NoError(err)
	s.Require().Len(rules, 2)
	s.Equal(1, rules[0].Priority)
	s.Equal(3, rules[1].Priority)
}

// --- Behavior 7: DeleteRule ---

func (s *RoutingRuleSQLiteSuite) TestDeleteRule() {
	ctx := context.Background()
	rule := s.validRule()

	s.Require().NoError(s.repo.UpsertRule(ctx, rule))

	err := s.repo.DeleteRule(ctx, rule.ID)
	s.Require().NoError(err)

	got, err := s.repo.GetRule(ctx, rule.ID)
	s.ErrorIs(err, repository.ErrNotFound)
	s.Nil(got)
}

func (s *RoutingRuleSQLiteSuite) TestDeleteRuleIdempotent() {
	ctx := context.Background()

	err := s.repo.DeleteRule(ctx, uuid.New())
	s.NoError(err, "deleting a non-existent rule should not return an error")
}

// --- Behavior: Default rules seeding ---

func (s *RoutingRuleSQLiteSuite) TestNewSeedsDefaultRulesWhenEmpty() {
	// Create an isolated DB so we don't rely on SetupTest's repo.
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "seed-test.db")

	db, err := sql.Open("sqlite", dbPath)
	s.Require().NoError(err)
	defer db.Close()

	_, err = db.Exec("PRAGMA journal_mode=WAL")
	s.Require().NoError(err)

	// Calling the constructor on a fresh DB should seed default rules.
	repo, err := sqlite.NewSQLiteRoutingRuleRepository(db)
	s.Require().NoError(err)
	s.Require().NotNil(repo)

	ctx := context.Background()
	rules, err := repo.ListRules(ctx)
	s.Require().NoError(err)
	s.Require().Len(rules, 2, "expected 2 default rules seeded into empty table")

	// Rule 0: channel_join
	r0 := rules[0]
	s.Equal(0, r0.Priority)
	s.Equal("slack", r0.SourceType)
	s.Equal("channel_join", r0.MessageType)
	s.Equal("notified", r0.Action)
	s.Equal("Channel Join", r0.Name)
	s.True(r0.Enabled)
	s.NotEqual(uuid.Nil, r0.ID, "seeded rule should have a non-zero UUID")
	s.False(r0.CreatedAt.IsZero(), "seeded rule should have a non-zero CreatedAt")
	s.False(r0.UpdatedAt.IsZero(), "seeded rule should have a non-zero UpdatedAt")

	// Rule 1: @username mention
	r1 := rules[1]
	s.Equal(1, r1.Priority)
	s.Equal("slack", r1.SourceType)
	s.Equal("@username", r1.ContentPattern)
	s.Equal("notified", r1.Action)
	s.Equal("@mention", r1.Name)
	s.True(r1.Enabled)
	s.NotEqual(uuid.Nil, r1.ID, "seeded rule should have a non-zero UUID")
	s.False(r1.CreatedAt.IsZero(), "seeded rule should have a non-zero CreatedAt")
	s.False(r1.UpdatedAt.IsZero(), "seeded rule should have a non-zero UpdatedAt")
}

func (s *RoutingRuleSQLiteSuite) TestNewDoesNotSeedWhenRulesExist() {
	// Create an isolated DB and manually insert a rule before calling the constructor.
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "no-seed-test.db")

	db, err := sql.Open("sqlite", dbPath)
	s.Require().NoError(err)
	defer db.Close()

	_, err = db.Exec("PRAGMA journal_mode=WAL")
	s.Require().NoError(err)

	// Manually create the table so we can insert a rule before the constructor runs.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS routing_rules (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			priority INTEGER NOT NULL,
			source_type TEXT NOT NULL,
			source_account TEXT,
			channel_pattern TEXT NOT NULL DEFAULT '',
			content_pattern TEXT NOT NULL DEFAULT '',
			message_type TEXT NOT NULL DEFAULT '',
			action TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`)
	s.Require().NoError(err)

	// Insert a pre-existing rule.
	now := time.Now().Truncate(time.Second).Format(time.RFC3339)
	preExistingID := uuid.New().String()
	_, err = db.Exec(
		"INSERT INTO routing_rules (id, name, priority, source_type, channel_pattern, content_pattern, message_type, action, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		preExistingID, "test", 5, "slack", "^general$", "", "", "notified", 1, now, now,
	)
	s.Require().NoError(err)

	// Now call the constructor — it should NOT seed defaults because rules already exist.
	repo, err := sqlite.NewSQLiteRoutingRuleRepository(db)
	s.Require().NoError(err)
	s.Require().NotNil(repo)

	ctx := context.Background()
	rules, err := repo.ListRules(ctx)
	s.Require().NoError(err)
	s.Require().Len(rules, 1, "constructor should not seed defaults when rules already exist")
	s.Equal(preExistingID, rules[0].ID.String())
}
