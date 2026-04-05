package presenter

import (
	"fmt"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
)

// TimerAlerter plays audio alerts when time blocks complete.
type TimerAlerter interface {
	PlayBlockComplete(blockType planner.BlockType)
}

// TimerPresenter manages countdown timer state for the UI.
type TimerPresenter struct {
	clock   planner.Clock
	alerter TimerAlerter

	block   planner.TimeBlock
	running bool
	alerted bool

	onTick          func()
	onBlockComplete func()
}

// NewTimerPresenter creates a new TimerPresenter with the given dependencies.
func NewTimerPresenter(clock planner.Clock, alerter TimerAlerter) (*TimerPresenter, error) {
	if clock == nil {
		return nil, fmt.Errorf("clock must not be nil")
	}
	if alerter == nil {
		return nil, fmt.Errorf("alerter must not be nil")
	}
	return &TimerPresenter{
		clock:   clock,
		alerter: alerter,
	}, nil
}

// Start begins a countdown for the given time block.
func (tp *TimerPresenter) Start(block planner.TimeBlock) {
	tp.block = block
	tp.running = true
	tp.alerted = false
}

// Stop stops the timer.
func (tp *TimerPresenter) Stop() {
	tp.running = false
}

// Tick is called by the UI to update timer state and fire callbacks.
func (tp *TimerPresenter) Tick() {
	if !tp.running {
		return
	}

	if tp.onTick != nil {
		tp.onTick()
	}

	elapsed := tp.clock.Now().Sub(tp.block.Start)
	duration := tp.block.End.Sub(tp.block.Start)

	if elapsed >= duration && !tp.alerted {
		tp.alerted = true
		if tp.block.Type != planner.BlockMeeting {
			tp.alerter.PlayBlockComplete(tp.block.Type)
		}
		if tp.onBlockComplete != nil {
			tp.onBlockComplete()
		}
	}
}

// ActiveSegment returns the current segment index (0-44) based on elapsed fraction.
func (tp *TimerPresenter) ActiveSegment() int {
	fraction := tp.ElapsedFraction()
	// Add small epsilon before truncating to handle floating point precision
	// loss from integer nanosecond division in time.Duration arithmetic.
	segment := int(fraction*45 + 1e-9)
	if segment > 44 {
		segment = 44
	}
	return segment
}

// ElapsedFraction returns the fraction of the block that has elapsed (0.0-1.0).
func (tp *TimerPresenter) ElapsedFraction() float64 {
	if !tp.running {
		return 0.0
	}
	elapsed := tp.clock.Now().Sub(tp.block.Start)
	duration := tp.block.End.Sub(tp.block.Start)
	if duration <= 0 {
		return 0.0
	}
	fraction := float64(elapsed) / float64(duration)
	if fraction > 1.0 {
		fraction = 1.0
	}
	if fraction < 0.0 {
		fraction = 0.0
	}
	return fraction
}

// IsFlashVisible returns whether the flash indicator should be visible (1Hz toggle).
func (tp *TimerPresenter) IsFlashVisible() bool {
	if !tp.running {
		return false
	}
	elapsed := tp.clock.Now().Sub(tp.block.Start)
	ms := elapsed.Milliseconds()
	return (ms/500)%2 == 0
}

// CurrentTaskName returns the current block's task name, or empty if not running.
func (tp *TimerPresenter) CurrentTaskName() string {
	if !tp.running {
		return ""
	}
	return tp.block.TaskName
}

// BlockType returns the current block's type.
func (tp *TimerPresenter) BlockType() planner.BlockType {
	return tp.block.Type
}

// IsRunning returns whether the timer is currently running.
func (tp *TimerPresenter) IsRunning() bool {
	return tp.running
}

// SetOnTick registers a callback to be called on each tick.
func (tp *TimerPresenter) SetOnTick(fn func()) {
	tp.onTick = fn
}

// SetOnBlockComplete registers a callback to be called when the block completes.
func (tp *TimerPresenter) SetOnBlockComplete(fn func()) {
	tp.onBlockComplete = fn
}

// Duration returns the total duration of the current block.
func (tp *TimerPresenter) Duration() time.Duration {
	return tp.block.End.Sub(tp.block.Start)
}
