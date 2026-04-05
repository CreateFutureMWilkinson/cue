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
	progress     float64
	flashVisible bool
}

// NewCountdownTimer creates a new countdown timer widget with zero progress.
func NewCountdownTimer() *CountdownTimer {
	t := &CountdownTimer{flashVisible: true}
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
func (t *CountdownTimer) SetFlashVisible(visible bool) {
	t.flashVisible = visible
}

// MinSize returns the minimum size for the countdown timer widget.
func (t *CountdownTimer) MinSize() fyne.Size {
	return fyne.NewSize(timerMinSize, timerMinSize)
}

// CreateRenderer returns a renderer that draws 45 line segments in a ring.
func (t *CountdownTimer) CreateRenderer() fyne.WidgetRenderer {
	r := &countdownTimerRenderer{timer: t}
	segments := t.Segments()
	for i := range segmentCount {
		line := canvas.NewLine(segments[i].Color)
		seg := segments[i]
		if seg.Length == longLength || seg.Length == mediumLength {
			line.StrokeWidth = 3.0
		} else {
			line.StrokeWidth = 2.0
		}
		r.lines[i] = line
	}
	return r
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

const canonicalRadius = 120.0

// countdownTimerRenderer draws the 45 line segments of the countdown timer ring.
type countdownTimerRenderer struct {
	timer *CountdownTimer
	lines [segmentCount]*canvas.Line
}

func (r *countdownTimerRenderer) Layout(size fyne.Size) {
	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	centerX := float64(size.Width / 2)
	centerY := float64(size.Height / 2)
	radius := float64(min(size.Width, size.Height)) / 2 * 0.9
	scale := radius / canonicalRadius

	segments := r.timer.Segments()
	for i, seg := range segments {
		angleRad := seg.AngleDeg * math.Pi / 180.0

		outerX := centerX + radius*math.Sin(angleRad)
		outerY := centerY - radius*math.Cos(angleRad)
		innerX := centerX + (radius-seg.Length*scale)*math.Sin(angleRad)
		innerY := centerY - (radius-seg.Length*scale)*math.Cos(angleRad)

		r.lines[i].Position1 = fyne.NewPos(float32(outerX), float32(outerY))
		r.lines[i].Position2 = fyne.NewPos(float32(innerX), float32(innerY))
	}
}

func (r *countdownTimerRenderer) MinSize() fyne.Size {
	return r.timer.MinSize()
}

func (r *countdownTimerRenderer) Refresh() {
	segments := r.timer.Segments()

	// Find the first non-elapsed segment index for flash logic.
	currentIdx := -1
	for i, seg := range segments {
		if seg.State != SegmentElapsed {
			currentIdx = i
			break
		}
	}

	for i, seg := range segments {
		r.lines[i].StrokeColor = seg.Color
		r.lines[i].Hidden = false

		// Flash logic: hide current segment when flash is not visible.
		if i == currentIdx && !r.timer.flashVisible {
			r.lines[i].Hidden = true
		}

		r.lines[i].Refresh()
	}
}

func (r *countdownTimerRenderer) Destroy() {}

func (r *countdownTimerRenderer) Objects() []fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, segmentCount)
	for i, line := range r.lines {
		objects[i] = line
	}
	return objects
}
