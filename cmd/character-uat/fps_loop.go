package characteruat

import (
	"fmt"
	"time"
)

// FPSLoopConfig holds the configuration for an FPSLoop.
type FPSLoopConfig struct {
	Counter     *FPSCounter
	Interval    time.Duration
	OnFPSUpdate func(string)
}

// FPSLoop periodically reads FPS from a counter and delivers formatted
// text to a callback. The callback approach allows callers to marshal
// the update onto the correct thread (e.g. fyne.Do for GUI safety).
type FPSLoop struct {
	counter     *FPSCounter
	interval    time.Duration
	onFPSUpdate func(string)
	stop        chan struct{}
}

// NewFPSLoop creates a new FPSLoop from the given config.
func NewFPSLoop(cfg FPSLoopConfig) *FPSLoop {
	return &FPSLoop{
		counter:     cfg.Counter,
		interval:    cfg.Interval,
		onFPSUpdate: cfg.OnFPSUpdate,
		stop:        make(chan struct{}),
	}
}

// Start launches the background goroutine that periodically calls
// Counter.FPS() and passes the formatted string to OnFPSUpdate.
func (l *FPSLoop) Start() {
	go l.run()
}

// Stop halts the loop. No callbacks fire after Stop returns.
func (l *FPSLoop) Stop() {
	close(l.stop)
}

func (l *FPSLoop) run() {
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			fps := l.counter.FPS()
			l.onFPSUpdate(fmt.Sprintf("FPS: %.1f", fps))
		case <-l.stop:
			return
		}
	}
}
