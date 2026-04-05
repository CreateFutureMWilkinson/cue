package characteruat

import (
	"sync"
	"time"
)

// FPSCounter tracks render frame rate by counting Tick calls and
// computing frames-per-second on demand.
type FPSCounter struct {
	mu        sync.Mutex
	frames    int
	lastCheck time.Time
	current   float64
}

// NewFPSCounter creates a new FPSCounter with zero initial FPS.
func NewFPSCounter() *FPSCounter {
	return &FPSCounter{}
}

// Tick records a single rendered frame.
func (f *FPSCounter) Tick() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.lastCheck.IsZero() {
		f.lastCheck = time.Now()
	}
	f.frames++
}

// FPS returns the current frames-per-second. It calculates the rate
// from ticks accumulated since the last FPS call and resets the counter.
// Returns 0.0 if no ticks have occurred.
func (f *FPSCounter) FPS() float64 {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.frames == 0 {
		return 0.0
	}

	now := time.Now()
	elapsed := now.Sub(f.lastCheck).Seconds()
	if elapsed <= 0 {
		return f.current
	}

	f.current = float64(f.frames) / elapsed
	f.frames = 0
	f.lastCheck = now

	return f.current
}
