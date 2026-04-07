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
		ID:        uuid.New(),
		Priority:  10,
		Source:    "slack",
		Field:     "channel",
		Negate:    false,
		Pattern:   "^general$",
		Action:    "notified",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
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
		idStr        string
		priority     int
		source       string
		field        string
		negate       int
		pattern      string
		action       string
		enabled      int
		createdAtStr string
		updatedAtStr string
	)
	err = s.db.QueryRowContext(ctx,
		"SELECT id, priority, source, field, negate, pattern, action, enabled, created_at, updated_at FROM routing_rules WHERE id = ?",
		rule.ID.String(),
	).Scan(&idStr, &priority, &source, &field, &negate, &pattern, &action, &enabled, &createdAtStr, &updatedAtStr)
	s.Require().NoError(err)

	s.Equal(rule.ID.String(), idStr)
	s.Equal(rule.Priority, priority)
	s.Equal(rule.Source, source)
	s.Equal(rule.Field, field)
	s.Equal(0, negate)
	s.Equal(rule.Pattern, pattern)
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
	rule.Pattern = "^random$"
	rule.Action = "ignored"
	rule.Negate = true
	rule.Enabled = false
	rule.UpdatedAt = rule.UpdatedAt.Add(time.Minute)

	err = s.repo.UpsertRule(ctx, rule)
	s.Require().NoError(err)

	// Read back with raw SQL to verify update took effect.
	var (
		priority int
		pattern  string
		action   string
		negate   int
		enabled  int
	)
	err = s.db.QueryRowContext(ctx,
		"SELECT priority, pattern, action, negate, enabled FROM routing_rules WHERE id = ?",
		rule.ID.String(),
	).Scan(&priority, &pattern, &action, &negate, &enabled)
	s.Require().NoError(err)

	s.Equal(99, priority)
	s.Equal("^random$", pattern)
	s.Equal("ignored", action)
	s.Equal(1, negate)
	s.Equal(0, enabled)
}

func (s *RoutingRuleSQLiteSuite) TestUpsertRuleValidationError() {
	ctx := context.Background()
	rule := s.validRule()
	rule.Source = "ftp" // invalid source

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
	s.Equal(rule.Source, got.Source)
	s.Equal(rule.Field, got.Field)
	s.Equal(rule.Negate, got.Negate)
	s.Equal(rule.Pattern, got.Pattern)
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

	rules, err := s.repo.ListRules(ctx)
	s.Require().NoError(err)
	s.NotNil(rules, "ListRules should return non-nil slice when empty")
	s.Empty(rules)
}

func (s *RoutingRuleSQLiteSuite) TestListRulesSortedByPriority() {
	ctx := context.Background()

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

// --- Behavior 6: ListRulesBySource ---

func (s *RoutingRuleSQLiteSuite) TestListRulesBySourceFiltered() {
	ctx := context.Background()

	slack1 := s.validRule()
	slack1.Source = "slack"
	slack1.Priority = 5

	slack2 := s.validRule()
	slack2.Source = "slack"
	slack2.Priority = 1

	email1 := s.validRule()
	email1.Source = "email"
	email1.Field = "sender"
	email1.Priority = 3

	s.Require().NoError(s.repo.UpsertRule(ctx, slack1))
	s.Require().NoError(s.repo.UpsertRule(ctx, slack2))
	s.Require().NoError(s.repo.UpsertRule(ctx, email1))

	rules, err := s.repo.ListRulesBySource(ctx, "slack")
	s.Require().NoError(err)
	s.Require().Len(rules, 2)

	s.Equal("slack", rules[0].Source)
	s.Equal("slack", rules[1].Source)
	s.Equal(1, rules[0].Priority, "results should be sorted by priority ascending")
	s.Equal(5, rules[1].Priority)
}

func (s *RoutingRuleSQLiteSuite) TestListRulesBySourceEmpty() {
	ctx := context.Background()

	rules, err := s.repo.ListRulesBySource(ctx, "email")
	s.Require().NoError(err)
	s.NotNil(rules, "ListRulesBySource should return non-nil slice when empty")
	s.Empty(rules)
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
