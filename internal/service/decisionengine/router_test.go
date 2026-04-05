package decisionengine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/CreateFutureMWilkinson/cue/internal/service/decisionengine"

	"github.com/stretchr/testify/suite"
)

// mockScorer implements decisionengine.Scorer for testing.
type mockScorer struct {
	result *decisionengine.ScorerResult
	err    error
	called bool
}

func (m *mockScorer) Score(_ context.Context, _ *repository.Message) (*decisionengine.ScorerResult, error) {
	m.called = true
	return m.result, m.err
}

// panicScorer panics if called — used to verify deterministic rules short-circuit.
type panicScorer struct{}

func (p *panicScorer) Score(_ context.Context, _ *repository.Message) (*decisionengine.ScorerResult, error) {
	panic("scorer should not be called for deterministic rules")
}

func defaultConfig() decisionengine.RouterConfig {
	return decisionengine.RouterConfig{
		ImportanceThreshold: 7,
		ConfidenceThreshold: 0.8,
	}
}

// --- Suite ---

type RouterSuite struct {
	suite.Suite
}

func TestRouter(t *testing.T) {
	suite.Run(t, new(RouterSuite))
}

// --- Constructor validation ---

func (s *RouterSuite) TestNewRouter_NilScorer() {
	_, err := decisionengine.NewRouter(nil, []string{"alice"}, defaultConfig())
	s.Error(err)
	s.Contains(err.Error(), "scorer")
}

func (s *RouterSuite) TestNewRouter_EmptyUsernames() {
	_, err := decisionengine.NewRouter(&mockScorer{}, []string{}, defaultConfig())
	s.Error(err)
	s.Contains(err.Error(), "usernames")
}

func (s *RouterSuite) TestNewRouter_ValidInputs() {
	r, err := decisionengine.NewRouter(&mockScorer{}, []string{"alice"}, defaultConfig())
	s.NoError(err)
	s.NotNil(r)
}

// --- Deterministic rules ---

func (s *RouterSuite) TestRoute_ChannelJoin_SetsNotified() {
	r, err := decisionengine.NewRouter(&panicScorer{}, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msg := &repository.Message{
		MessageType: "channel_join",
		RawContent:  "You were added to #general",
	}

	result, err := r.Route(context.Background(), msg)
	s.NoError(err)
	s.Equal(9.0, result.ImportanceScore)
	s.Equal(1.0, result.ConfidenceScore)
	s.Equal("Notified", result.Status)
	s.Contains(result.Reasoning, "channel")
}

func (s *RouterSuite) TestRoute_AtMention_SetsNotified() {
	r, err := decisionengine.NewRouter(&panicScorer{}, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msg := &repository.Message{
		MessageType: "message",
		RawContent:  "hey @alice check this out",
	}

	result, err := r.Route(context.Background(), msg)
	s.NoError(err)
	s.Equal(8.0, result.ImportanceScore)
	s.Equal(1.0, result.ConfidenceScore)
	s.Equal("Notified", result.Status)
	s.Contains(result.Reasoning, "mention")
}

func (s *RouterSuite) TestRoute_AtMention_CaseInsensitive() {
	r, err := decisionengine.NewRouter(&panicScorer{}, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msg := &repository.Message{
		MessageType: "message",
		RawContent:  "hey @Alice check this out",
	}

	result, err := r.Route(context.Background(), msg)
	s.NoError(err)
	s.Equal(8.0, result.ImportanceScore)
	s.Equal(1.0, result.ConfidenceScore)
	s.Equal("Notified", result.Status)
}

func (s *RouterSuite) TestRoute_AtMention_MultipleUsernames() {
	r, err := decisionengine.NewRouter(&panicScorer{}, []string{"alice", "bob"}, defaultConfig())
	s.Require().NoError(err)

	msg := &repository.Message{
		MessageType: "message",
		RawContent:  "hey @bob can you review?",
	}

	result, err := r.Route(context.Background(), msg)
	s.NoError(err)
	s.Equal(8.0, result.ImportanceScore)
	s.Equal("Notified", result.Status)
}

func (s *RouterSuite) TestRoute_ChannelJoinTakesPrecedence() {
	r, err := decisionengine.NewRouter(&panicScorer{}, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msg := &repository.Message{
		MessageType: "channel_join",
		RawContent:  "hey @alice you were added",
	}

	result, err := r.Route(context.Background(), msg)
	s.NoError(err)
	s.Equal(9.0, result.ImportanceScore, "channel_join should take precedence over @mention")
}

// --- Scorer-based routing with thresholds ---

func (s *RouterSuite) TestRoute_ScorerHighImportanceHighConfidence_Notified() {
	scorer := &mockScorer{result: &decisionengine.ScorerResult{
		ImportanceScore: 9.0,
		ConfidenceScore: 0.9,
		Reasoning:       "server outage detected",
	}}
	r, err := decisionengine.NewRouter(scorer, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msg := &repository.Message{MessageType: "message", RawContent: "production is down"}

	result, err := r.Route(context.Background(), msg)
	s.NoError(err)
	s.Equal("Notified", result.Status)
}

func (s *RouterSuite) TestRoute_ScorerHighImportanceLowConfidence_Buffered() {
	scorer := &mockScorer{result: &decisionengine.ScorerResult{
		ImportanceScore: 8.0,
		ConfidenceScore: 0.5,
		Reasoning:       "might be important",
	}}
	r, err := decisionengine.NewRouter(scorer, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msg := &repository.Message{MessageType: "message", RawContent: "something happened"}

	result, err := r.Route(context.Background(), msg)
	s.NoError(err)
	s.Equal("Buffered", result.Status)
}

func (s *RouterSuite) TestRoute_ScorerLowImportance_Ignored() {
	scorer := &mockScorer{result: &decisionengine.ScorerResult{
		ImportanceScore: 3.0,
		ConfidenceScore: 0.9,
		Reasoning:       "casual chat",
	}}
	r, err := decisionengine.NewRouter(scorer, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msg := &repository.Message{MessageType: "message", RawContent: "nice weather today"}

	result, err := r.Route(context.Background(), msg)
	s.NoError(err)
	s.Equal("Ignored", result.Status)
}

func (s *RouterSuite) TestRoute_ScorerExactThreshold_Notified() {
	scorer := &mockScorer{result: &decisionengine.ScorerResult{
		ImportanceScore: 7.0,
		ConfidenceScore: 0.8,
		Reasoning:       "at threshold",
	}}
	r, err := decisionengine.NewRouter(scorer, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msg := &repository.Message{MessageType: "message", RawContent: "borderline message"}

	result, err := r.Route(context.Background(), msg)
	s.NoError(err)
	s.Equal("Notified", result.Status, "exact threshold should be NOTIFIED (>= not >)")
}

func (s *RouterSuite) TestRoute_ScorerBelowImportanceThreshold_Ignored() {
	scorer := &mockScorer{result: &decisionengine.ScorerResult{
		ImportanceScore: 6.99,
		ConfidenceScore: 0.95,
		Reasoning:       "just below threshold",
	}}
	r, err := decisionengine.NewRouter(scorer, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msg := &repository.Message{MessageType: "message", RawContent: "almost important"}

	result, err := r.Route(context.Background(), msg)
	s.NoError(err)
	s.Equal("Ignored", result.Status)
}

func (s *RouterSuite) TestRoute_ScorerReasoningPreserved() {
	scorer := &mockScorer{result: &decisionengine.ScorerResult{
		ImportanceScore: 9.0,
		ConfidenceScore: 0.9,
		Reasoning:       "server outage detected",
	}}
	r, err := decisionengine.NewRouter(scorer, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msg := &repository.Message{MessageType: "message", RawContent: "production is down"}

	result, err := r.Route(context.Background(), msg)
	s.NoError(err)
	s.Equal("server outage detected", result.Reasoning)
}

// --- Error fallback ---

func (s *RouterSuite) TestRoute_ScorerError_FallbackBuffered() {
	scorer := &mockScorer{err: errors.New("connection timeout")}
	r, err := decisionengine.NewRouter(scorer, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msg := &repository.Message{MessageType: "message", RawContent: "some message"}

	result, err := r.Route(context.Background(), msg)
	s.NoError(err)
	s.Equal(7.0, result.ImportanceScore)
	s.Equal(0.0, result.ConfidenceScore)
	s.Equal("Buffered", result.Status)
}

func (s *RouterSuite) TestRoute_ScorerError_ReasoningContainsError() {
	scorer := &mockScorer{err: errors.New("timeout")}
	r, err := decisionengine.NewRouter(scorer, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msg := &repository.Message{MessageType: "message", RawContent: "some message"}

	result, err := r.Route(context.Background(), msg)
	s.NoError(err)
	s.Contains(result.Reasoning, "timeout")
}

// --- Custom thresholds ---

func (s *RouterSuite) TestRoute_CustomThresholds() {
	scorer := &mockScorer{result: &decisionengine.ScorerResult{
		ImportanceScore: 6.0,
		ConfidenceScore: 0.7,
		Reasoning:       "moderate importance",
	}}
	cfg := decisionengine.RouterConfig{
		ImportanceThreshold: 5,
		ConfidenceThreshold: 0.6,
	}
	r, err := decisionengine.NewRouter(scorer, []string{"alice"}, cfg)
	s.Require().NoError(err)

	msg := &repository.Message{MessageType: "message", RawContent: "custom threshold message"}

	result, err := r.Route(context.Background(), msg)
	s.NoError(err)
	s.Equal("Notified", result.Status, "should pass custom thresholds")
}

// --- Batch routing ---

func (s *RouterSuite) TestRouteBatch_MixedMessages() {
	scorer := &mockScorer{result: &decisionengine.ScorerResult{
		ImportanceScore: 9.0,
		ConfidenceScore: 0.9,
		Reasoning:       "important",
	}}
	r, err := decisionengine.NewRouter(scorer, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msgs := []*repository.Message{
		{MessageType: "channel_join", RawContent: "joined"},
		{MessageType: "message", RawContent: "hey @alice"},
		{MessageType: "message", RawContent: "normal message"},
	}

	results, err := r.RouteBatch(context.Background(), msgs)
	s.NoError(err)
	s.Len(results, 3)
	s.Equal(9.0, results[0].ImportanceScore) // channel_join
	s.Equal(8.0, results[1].ImportanceScore) // @mention
	s.Equal("Notified", results[2].Status)   // scorer
}

func (s *RouterSuite) TestRouteBatch_EmptySlice() {
	r, err := decisionengine.NewRouter(&mockScorer{}, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	results, err := r.RouteBatch(context.Background(), []*repository.Message{})
	s.NoError(err)
	s.Empty(results)
}

func (s *RouterSuite) TestRouteBatch_ScorerFailureDoesNotAbortBatch() {
	scorer := &mockScorer{err: errors.New("scorer failed")}
	r, err := decisionengine.NewRouter(scorer, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msgs := []*repository.Message{
		{MessageType: "message", RawContent: "first"},
		{MessageType: "channel_join", RawContent: "second"},
	}

	results, err := r.RouteBatch(context.Background(), msgs)
	s.NoError(err)
	s.Len(results, 2)
	s.Equal("Buffered", results[0].Status) // fallback
	s.Equal("Notified", results[1].Status) // deterministic
}

// --- No sender-based rules ---

func (s *RouterSuite) TestRoute_NormalMessage_NoSenderBasedImportance() {
	scorer := &mockScorer{result: &decisionengine.ScorerResult{
		ImportanceScore: 5.0,
		ConfidenceScore: 0.9,
		Reasoning:       "normal",
	}}
	r, err := decisionengine.NewRouter(scorer, []string{"alice"}, defaultConfig())
	s.Require().NoError(err)

	msg1 := &repository.Message{MessageType: "message", Sender: "ceo@company.com", RawContent: "hello"}
	msg2 := &repository.Message{MessageType: "message", Sender: "intern@company.com", RawContent: "hello"}

	result1, err := r.Route(context.Background(), msg1)
	s.NoError(err)
	result2, err := r.Route(context.Background(), msg2)
	s.NoError(err)

	s.Equal(result1.Status, result2.Status, "sender identity must not affect routing")
	s.Equal(result1.ImportanceScore, result2.ImportanceScore)
}
