package presenter_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
	"github.com/stretchr/testify/suite"
)

// mockCharacter records state transitions for testing.
type mockCharacter struct {
	name   string
	state  character.CharacterState
	mu     sync.Mutex
	states []character.CharacterState
}

func newMockCharacter() *mockCharacter {
	return &mockCharacter{name: "mock", state: character.StateIdle}
}

func (m *mockCharacter) Name() string { return m.name }

func (m *mockCharacter) TransitionTo(s character.CharacterState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = s
	m.states = append(m.states, s)
}

func (m *mockCharacter) CurrentState() character.CharacterState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *mockCharacter) Widget() fyne.CanvasObject { return nil }

func (m *mockCharacter) recordedStates() []character.CharacterState {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]character.CharacterState, len(m.states))
	copy(result, m.states)
	return result
}

type CharacterPresenterSuite struct {
	suite.Suite
}

func TestCharacterPresenter(t *testing.T) {
	suite.Run(t, new(CharacterPresenterSuite))
}

func (s *CharacterPresenterSuite) TestWorkingEvent() {
	char := newMockCharacter()
	source := newMockActivitySource()

	cp, err := presenter.NewCharacterPresenter(char, source, 500*time.Millisecond)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cp.Start(ctx)
	defer cp.Stop()

	source.ch <- presenter.ActivityEvent{Source: "slack", Message: "fetching messages"}

	// Allow time for the event to be processed.
	time.Sleep(50 * time.Millisecond)

	s.Equal(character.StateWorking, char.CurrentState())
}

func (s *CharacterPresenterSuite) TestNotifyingEvent() {
	char := newMockCharacter()
	source := newMockActivitySource()

	cp, err := presenter.NewCharacterPresenter(char, source, 500*time.Millisecond)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cp.Start(ctx)
	defer cp.Stop()

	source.ch <- presenter.ActivityEvent{Source: "router", Message: "NOTIFIED: important message"}

	time.Sleep(50 * time.Millisecond)

	s.Equal(character.StateNotifying, char.CurrentState())
}

func (s *CharacterPresenterSuite) TestErrorEvent() {
	char := newMockCharacter()
	source := newMockActivitySource()

	cp, err := presenter.NewCharacterPresenter(char, source, 500*time.Millisecond)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cp.Start(ctx)
	defer cp.Stop()

	source.ch <- presenter.ActivityEvent{Source: "slack", Message: "connection failed", IsError: true}

	time.Sleep(50 * time.Millisecond)

	s.Equal(character.StateError, char.CurrentState())
}

func (s *CharacterPresenterSuite) TestStateDecay() {
	char := newMockCharacter()
	source := newMockActivitySource()

	decayDuration := 50 * time.Millisecond
	cp, err := presenter.NewCharacterPresenter(char, source, decayDuration)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cp.Start(ctx)
	defer cp.Stop()

	source.ch <- presenter.ActivityEvent{Source: "slack", Message: "fetching messages"}

	// Wait for event processing.
	time.Sleep(20 * time.Millisecond)
	s.Equal(character.StateWorking, char.CurrentState())

	// Wait for decay to fire (longer than decay duration).
	time.Sleep(80 * time.Millisecond)
	s.Equal(character.StateIdle, char.CurrentState())
}

func (s *CharacterPresenterSuite) TestStartStop() {
	char := newMockCharacter()
	source := newMockActivitySource()

	cp, err := presenter.NewCharacterPresenter(char, source, 500*time.Millisecond)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cp.Start(ctx)

	// Send an event and verify it processes.
	source.ch <- presenter.ActivityEvent{Source: "slack", Message: "fetching messages"}
	time.Sleep(50 * time.Millisecond)
	s.Equal(character.StateWorking, char.CurrentState())

	// Stop and verify clean shutdown (no panic).
	s.NotPanics(func() {
		cp.Stop()
	})
}
