package alert_test

import (
	"context"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/alert"
	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
	"github.com/stretchr/testify/suite"
)

// ---------------------------------------------------------------------------
// Mock: records PlayTimerEnd calls
// ---------------------------------------------------------------------------

type playTimerEndCall struct {
	BlockType  string
	TaskName   string
	Suppressed bool
}

type mockTimerPlayer struct {
	calls []playTimerEndCall
}

func (m *mockTimerPlayer) PlayTimerEnd(_ context.Context, blockType string, taskName string, suppressed bool) (*alert.MissedAlert, error) {
	m.calls = append(m.calls, playTimerEndCall{
		BlockType:  blockType,
		TaskName:   taskName,
		Suppressed: suppressed,
	})
	return nil, nil
}

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

type TimerAlerterAdapterSuite struct {
	suite.Suite
}

func TestTimerAlerterAdapter(t *testing.T) {
	suite.Run(t, new(TimerAlerterAdapterSuite))
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func (s *TimerAlerterAdapterSuite) TestPlayBlockCompleteCallsPlayTimerEndWithFocus() {
	player := &mockTimerPlayer{}
	adapter := alert.NewTimerAlerterAdapter(player)

	adapter.PlayBlockComplete(planner.BlockFocus)

	s.Require().Len(player.calls, 1, "PlayTimerEnd should be called exactly once")
	s.Equal("focus", player.calls[0].BlockType)
	s.Equal("", player.calls[0].TaskName)
	s.Equal(false, player.calls[0].Suppressed)
}
