package presenter

import (
	"fmt"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/service/planner"
)

const (
	// Timer ring has 45 segments total
	timerRingSegments = 45

	// Flash interval for 1Hz toggle (500ms on, 500ms off)
	flashIntervalMs = 500
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

	elapsed, duration := tp.getTimings()

	if elapsed >= duration && !tp.alerted {
		tp.alerted = true
		tp.handleBlockCompletion()
	}
}

// getTimings returns the elapsed time and total duration for the current block.
func (tp *TimerPresenter) getTimings() (elapsed, duration time.Duration) {
	elapsed = tp.clock.Now().Sub(tp.block.Start)
	duration = tp.block.End.Sub(tp.block.Start)
	return elapsed, duration
}

// handleBlockCompletion fires alerts and callbacks when a block completes.
func (tp *TimerPresenter) handleBlockCompletion() {
	if tp.block.Type != planner.BlockMeeting {
		tp.alerter.PlayBlockComplete(tp.block.Type)
	}
	if tp.onBlockComplete != nil {
		tp.onBlockComplete()
	}
}

// ActiveSegment returns the current segment index (0-44) based on elapsed fraction.
func (tp *TimerPresenter) ActiveSegment() int {
	fraction := tp.ElapsedFraction()
	// Add small epsilon before truncating to handle floating point precision
	// loss from integer nanosecond division in time.Duration arithmetic.
	segment := int(fraction*timerRingSegments + 1e-9)
	return min(segment, timerRingSegments-1)
}

// ElapsedFraction returns the fraction of the block that has elapsed (0.0-1.0).
func (tp *TimerPresenter) ElapsedFraction() float64 {
	if !tp.running {
		return 0.0
	}
	elapsed, duration := tp.getTimings()
	if duration <= 0 {
		return 0.0
	}
	fraction := float64(elapsed) / float64(duration)
	return clampFraction(fraction)
}

// clampFraction ensures the fraction is within the valid range of 0.0-1.0.
func clampFraction(fraction float64) float64 {
	return max(0.0, min(1.0, fraction))
}

// IsFlashVisible returns whether the flash indicator should be visible (1Hz toggle).
func (tp *TimerPresenter) IsFlashVisible() bool {
	if !tp.running {
		return false
	}
	elapsed := tp.clock.Now().Sub(tp.block.Start)
	ms := elapsed.Milliseconds()
	return (ms/flashIntervalMs)%2 == 0
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
	_, duration := tp.getTimings()
	return duration
}
