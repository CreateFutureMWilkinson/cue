package ui

import (
	"context"
	"fmt"
	"time"
)

// TickableTimer abstracts the timer presenter for the timer loop.
type TickableTimer interface {
	Tick()
	ElapsedFraction() float64
	IsFlashVisible() bool
	CurrentTaskName() string
}

// TimerWidget abstracts the countdown timer widget for updates.
type TimerWidget interface {
	SetProgress(float64)
	SetFlashVisible(bool)
}

// TaskUpdater abstracts the focus rail task label for updates.
type TaskUpdater interface {
	SetCurrentTask(string)
}

// TimerLoop drives the timer at 1Hz by calling Tick and updating the UI.
// It manages a background goroutine that ticks every second and updates
// the timer widget, flash visibility, and current task display.
type TimerLoop struct {
	timer    TickableTimer
	widget   TimerWidget
	taskView TaskUpdater
	doFunc   UIScheduler
	stopped  bool
	cancel   context.CancelFunc
}

// NewTimerLoop creates a new TimerLoop, validating all dependencies.
func NewTimerLoop(timer TickableTimer, widget TimerWidget, taskView TaskUpdater) (*TimerLoop, error) {
	if timer == nil {
		return nil, fmt.Errorf("timer must not be nil")
	}
	if widget == nil {
		return nil, fmt.Errorf("widget must not be nil")
	}
	if taskView == nil {
		return nil, fmt.Errorf("taskView must not be nil")
	}
	return &TimerLoop{
		timer:    timer,
		widget:   widget,
		taskView: taskView,
	}, nil
}

// SetUIScheduler sets the function used to dispatch widget updates to the UI thread.
func (l *TimerLoop) SetUIScheduler(fn UIScheduler) {
	l.doFunc = fn
}

// scheduleUI dispatches fn via the UIScheduler if set, otherwise calls fn directly.
func (l *TimerLoop) scheduleUI(fn func()) {
	if l.doFunc != nil {
		l.doFunc(fn)
	} else {
		fn()
	}
}

// TickOnce performs a single tick cycle: calls Tick, then updates UI from timer state.
func (l *TimerLoop) TickOnce() {
	if l.stopped {
		return
	}
	l.timer.Tick()
	elapsed := l.timer.ElapsedFraction()
	flash := l.timer.IsFlashVisible()
	task := l.timer.CurrentTaskName()
	l.scheduleUI(func() {
		l.widget.SetProgress(elapsed)
		l.widget.SetFlashVisible(flash)
		l.taskView.SetCurrentTask(task)
	})
}

// Start begins the 1Hz tick loop in a background goroutine.
func (l *TimerLoop) Start(ctx context.Context) {
	ctx, l.cancel = context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				l.TickOnce()
			}
		}
	}()
}

// Stop halts the tick loop.
func (l *TimerLoop) Stop() {
	l.stopped = true
	if l.cancel != nil {
		l.cancel()
	}
}
