package alert

import (
	"context"

	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
)

// TimerPlayer abstracts the PlayTimerEnd method for the adapter.
type TimerPlayer interface {
	PlayTimerEnd(ctx context.Context, blockType string, taskName string, suppressed bool) (*MissedAlert, error)
}

// TimerAlerterAdapter adapts a TimerPlayer to the presenter.TimerAlerter interface.
type TimerAlerterAdapter struct {
	player TimerPlayer
}

// NewTimerAlerterAdapter creates a new adapter wrapping the given TimerPlayer.
func NewTimerAlerterAdapter(player TimerPlayer) *TimerAlerterAdapter {
	return &TimerAlerterAdapter{player: player}
}

// PlayBlockComplete converts BlockType to string and fires the alert (fire-and-forget).
func (a *TimerAlerterAdapter) PlayBlockComplete(blockType planner.BlockType) {
	var bt string
	switch blockType {
	case planner.BlockFocus:
		bt = "focus"
	case planner.BlockShortBreak:
		bt = "short_break"
	case planner.BlockLongBreak:
		bt = "long_break"
	case planner.BlockMeeting:
		bt = "meeting"
	default:
		bt = "unknown"
	}
	_, _ = a.player.PlayTimerEnd(context.Background(), bt, "", false)
}
