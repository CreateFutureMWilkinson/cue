package ui

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// SegmentState describes whether a timer segment is future, current, or elapsed.
type SegmentState int

const (
	// SegmentFuture indicates the segment has not yet been reached.
	SegmentFuture SegmentState = iota
	// SegmentCurrent indicates the segment is currently active.
	SegmentCurrent
	// SegmentElapsed indicates the segment has already passed.
	SegmentElapsed
)

const (
	segmentCount    = 45
	segmentInterval = 8 // degrees

	shortLength  = 12.0
	mediumLength = 24.0
	longLength   = 36.0

	timerMinSize = 120.0

	// Alpha transparency values for segment colors
	elapsedAlpha = 64
)

var (
	futureColor  = color.NRGBA{R: 0xFF, G: 0xCE, B: 0x1B, A: 0xFF}
	elapsedColor = color.NRGBA{R: 0xFF, G: 0xCE, B: 0x1B, A: elapsedAlpha}
)

// SegmentInfo describes a single segment of the countdown timer ring.
type SegmentInfo struct {
	AngleDeg float64
	Length   float64
	State    SegmentState
	Color    color.NRGBA
}

// CountdownTimer is a custom Fyne widget that displays a circular burndown
// timer with 45 line segments at 8-degree intervals.
type CountdownTimer struct {
	widget.BaseWidget
	progress float64
}

// NewCountdownTimer creates a new countdown timer widget with zero progress.
func NewCountdownTimer() *CountdownTimer {
	t := &CountdownTimer{}
	t.ExtendBaseWidget(t)
	t.Hide()
	return t
}

// Segments returns the 45 segment descriptors based on current progress.
func (t *CountdownTimer) Segments() []SegmentInfo {
	segments := make([]SegmentInfo, segmentCount)
	elapsedCount := int(math.Round(t.progress * float64(segmentCount)))

	for i := range segmentCount {
		angle := float64((i + 1) * segmentInterval)
		length := segmentLength(angle)

		state := SegmentFuture
		segmentColor := futureColor
		if i < elapsedCount {
			state = SegmentElapsed
			segmentColor = elapsedColor
		}

		segments[i] = SegmentInfo{
			AngleDeg: angle,
			Length:   length,
			State:    state,
			Color:    segmentColor,
		}
	}
	return segments
}

// SetProgress sets the timer progress from 0.0 (no elapsed) to 1.0 (all elapsed).
func (t *CountdownTimer) SetProgress(p float64) {
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	t.progress = p
	t.Refresh()
}

// Reset clears the timer progress back to zero.
func (t *CountdownTimer) Reset() {
	t.SetProgress(0)
}

// SetFlashVisible controls visibility of the current (first non-elapsed) segment
// for the 1Hz flash animation. When false, the current segment is hidden.
func (t *CountdownTimer) SetFlashVisible(_ bool) {
	// TODO: implement flash visibility
}

// MinSize returns the minimum size for the countdown timer widget.
func (t *CountdownTimer) MinSize() fyne.Size {
	return fyne.NewSize(timerMinSize, timerMinSize)
}

// CreateRenderer returns a minimal renderer for the countdown timer.
func (t *CountdownTimer) CreateRenderer() fyne.WidgetRenderer {
	return &countdownTimerRenderer{timer: t}
}

// segmentLength returns the line length for a segment at the given angle.
func segmentLength(angle float64) float64 {
	mod := math.Mod(angle, 360)
	if mod == 0 || mod == 90 || mod == 180 || mod == 270 {
		return longLength
	}
	if mod == 45 || mod == 135 || mod == 225 || mod == 315 {
		return mediumLength
	}
	return shortLength
}

// countdownTimerRenderer is a minimal renderer satisfying fyne.WidgetRenderer.
type countdownTimerRenderer struct {
	timer *CountdownTimer
}

func (r *countdownTimerRenderer) Layout(_ fyne.Size) {}
func (r *countdownTimerRenderer) MinSize() fyne.Size { return r.timer.MinSize() }
func (r *countdownTimerRenderer) Refresh()           {}
func (r *countdownTimerRenderer) Destroy()           {}
func (r *countdownTimerRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{canvas.NewRectangle(color.Transparent)}
}
