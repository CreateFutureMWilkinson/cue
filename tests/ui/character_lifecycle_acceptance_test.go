//go:build ui_acceptance

package ui_acceptance_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// CharacterLifecycleAcceptanceSuite asserts the cross-package wiring
// of activity envelopes, alert envelopes, and shutdown into character
// state transitions, as defined in Feature-114.
type CharacterLifecycleAcceptanceSuite struct {
	suite.Suite
}

func TestCharacterLifecycleAcceptance(t *testing.T) {
	suite.Run(t, new(CharacterLifecycleAcceptanceSuite))
}

// recordingCharacter is a minimal Character implementation that records
// every state transition. Used to drive end-to-end wiring assertions.
type recordingCharacter struct {
	mu       sync.Mutex
	current  character.CharacterState
	history  []character.CharacterState
	shutdown chan struct{}
}

func newRecordingCharacter() *recordingCharacter {
	return &recordingCharacter{
		current:  character.StateStarting,
		shutdown: make(chan struct{}),
	}
}

func (c *recordingCharacter) Name() string { return "recording" }

func (c *recordingCharacter) TransitionTo(s character.CharacterState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = s
	c.history = append(c.history, s)
}

func (c *recordingCharacter) CurrentState() character.CharacterState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

func (c *recordingCharacter) Widget() fyne.CanvasObject { return nil }

func (c *recordingCharacter) Close() {}

func (c *recordingCharacter) Shutdown() <-chan struct{} {
	c.TransitionTo(character.StateShuttingDown)
	close(c.shutdown)
	return c.shutdown
}

func (c *recordingCharacter) Recorded() []character.CharacterState {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]character.CharacterState, len(c.history))
	copy(out, c.history)
	return out
}

// alertChan adapts a chan presenter.AlertEvent to presenter.AlertSource.
type alertChan struct{ ch chan presenter.AlertEvent }

func (a *alertChan) Events() <-chan presenter.AlertEvent { return a.ch }

// AC: An alert envelope causes the character to transition to StateNotifying.
func (s *CharacterLifecycleAcceptanceSuite) TestAlertEnvelopeTransitionsToNotifying() {
	char := newRecordingCharacter()
	activitySrc := newMockActivitySource()
	alertSrc := &alertChan{ch: make(chan presenter.AlertEvent, 1)}

	cp, err := presenter.NewCharacterPresenter(char, activitySrc, alertSrc, 500*time.Millisecond)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cp.Start(ctx)
	defer cp.Stop()

	alertSrc.ch <- presenter.AlertEvent{Kind: "notification"}
	time.Sleep(50 * time.Millisecond)

	s.Equal(character.StateNotifying, char.CurrentState(),
		"alert envelope must drive character to StateNotifying")
}

// AC: A queue-depth heartbeat activity envelope must NOT change state.
func (s *CharacterLifecycleAcceptanceSuite) TestQueueDepthHeartbeatIgnored() {
	char := newRecordingCharacter()
	activitySrc := newMockActivitySource()
	alertSrc := &alertChan{ch: make(chan presenter.AlertEvent, 1)}

	cp, err := presenter.NewCharacterPresenter(char, activitySrc, alertSrc, 500*time.Millisecond)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cp.Start(ctx)
	defer cp.Stop()

	activitySrc.ch <- presenter.ActivityEvent{
		Source:  "queue",
		Message: "Ollama queue depth: 3",
	}
	time.Sleep(50 * time.Millisecond)

	s.Equal(character.StateStarting, char.CurrentState(),
		"queue-depth heartbeat must not transition state away from initial")
	s.Empty(char.Recorded(), "heartbeat must record no transitions")
}

// AC: Character interface exposes Shutdown() <-chan struct{} and the
// shutdown channel closes after the character transitions through
// StateShuttingDown.
func (s *CharacterLifecycleAcceptanceSuite) TestCharacterShutdownTransitionsAndCloses() {
	char := newRecordingCharacter()

	// Compile-time: the interface must include Shutdown.
	var iface character.Character = char
	done := iface.Shutdown()

	select {
	case <-done:
	case <-time.After(time.Second):
		s.FailNow("Shutdown channel never closed")
	}

	recorded := char.Recorded()
	s.Require().NotEmpty(recorded)
	s.Equal(character.StateShuttingDown, recorded[len(recorded)-1],
		"Shutdown must transition through StateShuttingDown")
}

// AC: A non-heartbeat activity envelope drives StateWorking.
func (s *CharacterLifecycleAcceptanceSuite) TestActivityEventTransitionsToWorking() {
	char := newRecordingCharacter()
	activitySrc := newMockActivitySource()
	alertSrc := &alertChan{ch: make(chan presenter.AlertEvent, 1)}

	cp, err := presenter.NewCharacterPresenter(char, activitySrc, alertSrc, 500*time.Millisecond)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cp.Start(ctx)
	defer cp.Stop()

	activitySrc.ch <- presenter.ActivityEvent{
		Source:  "slack",
		Message: "fetched 3 messages",
	}
	time.Sleep(50 * time.Millisecond)

	s.Equal(character.StateWorking, char.CurrentState())
}
