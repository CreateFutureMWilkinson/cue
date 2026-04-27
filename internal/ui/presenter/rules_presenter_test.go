package presenter_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Mock RoutingRuleRepository ---

type mockRuleRepo struct {
	rules      []*repository.RoutingRule
	listErr    error
	getRule    *repository.RoutingRule
	getErr     error
	upsertErr  error
	deleteErr  error
	upserted   []*repository.RoutingRule
	deletedIDs []uuid.UUID
}

func (m *mockRuleRepo) ListRules(_ context.Context) ([]*repository.RoutingRule, error) {
	return m.rules, m.listErr
}

func (m *mockRuleRepo) ListRulesBySourceType(_ context.Context, _ string) ([]*repository.RoutingRule, error) {
	return m.rules, m.listErr
}

func (m *mockRuleRepo) ListRulesBySourceAccount(_ context.Context, _ uuid.UUID) ([]*repository.RoutingRule, error) {
	return m.rules, m.listErr
}

func (m *mockRuleRepo) GetRule(_ context.Context, _ uuid.UUID) (*repository.RoutingRule, error) {
	return m.getRule, m.getErr
}

func (m *mockRuleRepo) UpsertRule(_ context.Context, rule *repository.RoutingRule) error {
	if m.upsertErr != nil {
		return m.upsertErr
	}
	m.upserted = append(m.upserted, rule)
	return nil
}

func (m *mockRuleRepo) DeleteRule(_ context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedIDs = append(m.deletedIDs, id)
	return nil
}

// --- Mock QueueRepository ---

type mockQueueRepo struct {
	pending    int
	pendingErr error
}

func (m *mockQueueRepo) Enqueue(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockQueueRepo) DequeueOldest(_ context.Context) (*repository.QueueEntry, error) {
	return nil, nil
}
func (m *mockQueueRepo) MarkDone(_ context.Context, _ uuid.UUID) error   { return nil }
func (m *mockQueueRepo) MarkFailed(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockQueueRepo) PendingCount(_ context.Context) (int, error) {
	return m.pending, m.pendingErr
}
func (m *mockQueueRepo) PurgeCompleted(_ context.Context) error              { return nil }
func (m *mockQueueRepo) PurgeOlderThan(_ context.Context, _ time.Time) error { return nil }
func (m *mockQueueRepo) PurgeAll(_ context.Context) error                    { return nil }
func (m *mockQueueRepo) ResetProcessing(_ context.Context) (int64, error)    { return 0, nil }

// --- Suite ---

type RulesPresenterSuite struct {
	suite.Suite
}

func TestRulesPresenter(t *testing.T) {
	suite.Run(t, new(RulesPresenterSuite))
}

// --- Tests ---

func (s *RulesPresenterSuite) TestListRulesDelegatesToRepo() {
	rule1 := &repository.RoutingRule{
		ID:             uuid.New(),
		Priority:       0,
		SourceType:     "slack",
		ChannelPattern: "general",
		Action:         "notified",
		Enabled:        true,
	}
	rule2 := &repository.RoutingRule{
		ID:             uuid.New(),
		Priority:       1,
		SourceType:     "email",
		ContentPattern: "boss@example.com",
		Action:         "notified",
		Enabled:        true,
	}

	ruleRepo := &mockRuleRepo{rules: []*repository.RoutingRule{rule1, rule2}}
	queueRepo := &mockQueueRepo{}
	p := presenter.NewRulesPresenter(ruleRepo, queueRepo, 50)

	rules, err := p.ListRules(context.Background())
	s.Require().NoError(err)
	s.Require().Len(rules, 2)
	s.Equal(rule1.ID, rules[0].ID)
	s.Equal(rule2.ID, rules[1].ID)
}

func (s *RulesPresenterSuite) TestSaveRuleValidatesAndUpserts() {
	ruleRepo := &mockRuleRepo{}
	queueRepo := &mockQueueRepo{}
	p := presenter.NewRulesPresenter(ruleRepo, queueRepo, 50)

	validRule := &repository.RoutingRule{
		ID:             uuid.New(),
		Priority:       0,
		SourceType:     "slack",
		ChannelPattern: "general",
		Action:         "notified",
		Enabled:        true,
	}

	// Valid rule should be saved via UpsertRule.
	err := p.SaveRule(context.Background(), validRule)
	s.Require().NoError(err)
	s.Require().Len(ruleRepo.upserted, 1)
	s.Equal(validRule.ID, ruleRepo.upserted[0].ID)

	// Invalid rule (bad source type) should return validation error, not reach repo.
	invalidRule := &repository.RoutingRule{
		ID:         uuid.New(),
		Priority:   0,
		SourceType: "fax",
		Action:     "notified",
		Enabled:    true,
	}

	err = p.SaveRule(context.Background(), invalidRule)
	s.Error(err)
	s.ErrorIs(err, repository.ErrInvalidRoutingRule)
	// Repo should not have received the invalid rule.
	s.Len(ruleRepo.upserted, 1) // still 1 from above
}

func (s *RulesPresenterSuite) TestDeleteRuleDelegatesToRepo() {
	ruleRepo := &mockRuleRepo{}
	queueRepo := &mockQueueRepo{}
	p := presenter.NewRulesPresenter(ruleRepo, queueRepo, 50)

	id := uuid.New()
	err := p.DeleteRule(context.Background(), id)
	s.Require().NoError(err)
	s.Require().Len(ruleRepo.deletedIDs, 1)
	s.Equal(id, ruleRepo.deletedIDs[0])
}

func (s *RulesPresenterSuite) TestReorderRuleShiftsPriorities() {
	// Set up three rules at priorities 0, 1, 2.
	id0 := uuid.New()
	id1 := uuid.New()
	id2 := uuid.New()

	rules := []*repository.RoutingRule{
		{ID: id0, Priority: 0, SourceType: "slack", ChannelPattern: "a", Action: "notified", Enabled: true},
		{ID: id1, Priority: 1, SourceType: "slack", ChannelPattern: "b", Action: "notified", Enabled: true},
		{ID: id2, Priority: 2, SourceType: "slack", ChannelPattern: "c", Action: "notified", Enabled: true},
	}

	ruleRepo := &mockRuleRepo{rules: rules}
	queueRepo := &mockQueueRepo{}
	p := presenter.NewRulesPresenter(ruleRepo, queueRepo, 50)

	// Move rule id2 (priority 2) to priority 0.
	// Expected: id2 -> 0, id0 -> 1, id1 -> 2
	err := p.ReorderRule(context.Background(), id2, 0)
	s.Require().NoError(err)

	// At least the moved rule and the shifted rules should have been upserted.
	s.GreaterOrEqual(len(ruleRepo.upserted), 2, "should upsert at least the moved rule and shifted rules")

	// Find the upserted priorities by ID.
	priorities := map[uuid.UUID]int{}
	for _, r := range ruleRepo.upserted {
		priorities[r.ID] = r.Priority
	}

	s.Equal(0, priorities[id2], "moved rule should have priority 0")
}

func (s *RulesPresenterSuite) TestToggleRuleUpdatesEnabled() {
	id := uuid.New()
	rule := &repository.RoutingRule{
		ID:             id,
		Priority:       0,
		SourceType:     "slack",
		ChannelPattern: "general",
		Action:         "notified",
		Enabled:        true,
	}

	ruleRepo := &mockRuleRepo{getRule: rule}
	queueRepo := &mockQueueRepo{}
	p := presenter.NewRulesPresenter(ruleRepo, queueRepo, 50)

	// Toggle to disabled.
	err := p.ToggleRule(context.Background(), id, false)
	s.Require().NoError(err)
	s.Require().Len(ruleRepo.upserted, 1)
	s.False(ruleRepo.upserted[0].Enabled, "rule should be disabled after toggle")
}

func (s *RulesPresenterSuite) TestQueueDepthDelegatesToRepo() {
	ruleRepo := &mockRuleRepo{}
	queueRepo := &mockQueueRepo{pending: 42}
	p := presenter.NewRulesPresenter(ruleRepo, queueRepo, 50)

	depth, err := p.QueueDepth(context.Background())
	s.Require().NoError(err)
	s.Equal(42, depth)
}

func (s *RulesPresenterSuite) TestQueueWarningThresholdReturnsWarnAt() {
	ruleRepo := &mockRuleRepo{}
	queueRepo := &mockQueueRepo{}
	p := presenter.NewRulesPresenter(ruleRepo, queueRepo, 75)

	s.Equal(75, p.QueueWarningThreshold())
}

// --- Reloader callback tests ---

func (s *RulesPresenterSuite) TestSaveRuleCallsReloader() {
	ruleRepo := &mockRuleRepo{}
	queueRepo := &mockQueueRepo{}

	callCount := 0
	reloader := func() { callCount++ }

	p := presenter.NewRulesPresenter(ruleRepo, queueRepo, 50, presenter.WithReloader(reloader))

	rule := &repository.RoutingRule{
		ID:             uuid.New(),
		Priority:       0,
		SourceType:     "slack",
		ChannelPattern: "general",
		Action:         "notified",
		Enabled:        true,
	}

	err := p.SaveRule(context.Background(), rule)
	s.Require().NoError(err)
	s.Equal(1, callCount, "reloader should be called once after SaveRule")
}

func (s *RulesPresenterSuite) TestDeleteRuleCallsReloader() {
	ruleRepo := &mockRuleRepo{}
	queueRepo := &mockQueueRepo{}

	callCount := 0
	reloader := func() { callCount++ }

	p := presenter.NewRulesPresenter(ruleRepo, queueRepo, 50, presenter.WithReloader(reloader))

	err := p.DeleteRule(context.Background(), uuid.New())
	s.Require().NoError(err)
	s.Equal(1, callCount, "reloader should be called once after DeleteRule")
}

func (s *RulesPresenterSuite) TestReorderRuleCallsReloader() {
	id0 := uuid.New()
	id1 := uuid.New()

	rules := []*repository.RoutingRule{
		{ID: id0, Priority: 0, SourceType: "slack", ChannelPattern: "a", Action: "notified", Enabled: true},
		{ID: id1, Priority: 1, SourceType: "slack", ChannelPattern: "b", Action: "notified", Enabled: true},
	}

	ruleRepo := &mockRuleRepo{rules: rules}
	queueRepo := &mockQueueRepo{}

	callCount := 0
	reloader := func() { callCount++ }

	p := presenter.NewRulesPresenter(ruleRepo, queueRepo, 50, presenter.WithReloader(reloader))

	err := p.ReorderRule(context.Background(), id1, 0)
	s.Require().NoError(err)
	s.Equal(1, callCount, "reloader should be called once after ReorderRule")
}

func (s *RulesPresenterSuite) TestToggleRuleCallsReloader() {
	id := uuid.New()
	rule := &repository.RoutingRule{
		ID:             id,
		Priority:       0,
		SourceType:     "slack",
		ChannelPattern: "general",
		Action:         "notified",
		Enabled:        true,
	}

	ruleRepo := &mockRuleRepo{getRule: rule}
	queueRepo := &mockQueueRepo{}

	callCount := 0
	reloader := func() { callCount++ }

	p := presenter.NewRulesPresenter(ruleRepo, queueRepo, 50, presenter.WithReloader(reloader))

	err := p.ToggleRule(context.Background(), id, false)
	s.Require().NoError(err)
	s.Equal(1, callCount, "reloader should be called once after ToggleRule")
}

func (s *RulesPresenterSuite) TestMutationWithoutReloaderDoesNotPanic() {
	ruleRepo := &mockRuleRepo{}
	queueRepo := &mockQueueRepo{}

	// No WithReloader option — reloader is nil.
	p := presenter.NewRulesPresenter(ruleRepo, queueRepo, 50)

	rule := &repository.RoutingRule{
		ID:             uuid.New(),
		Priority:       0,
		SourceType:     "slack",
		ChannelPattern: "general",
		Action:         "notified",
		Enabled:        true,
	}

	// These should not panic even though no reloader is set.
	s.NotPanics(func() {
		_ = p.SaveRule(context.Background(), rule)
	}, "SaveRule without reloader should not panic")

	s.NotPanics(func() {
		_ = p.DeleteRule(context.Background(), uuid.New())
	}, "DeleteRule without reloader should not panic")
}
