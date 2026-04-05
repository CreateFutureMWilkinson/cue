package characteruat_test

import (
	"sync/atomic"
	"testing"
	"time"

	characteruat "github.com/CreateFutureMWilkinson/cue/cmd/character-uat"

	"github.com/stretchr/testify/suite"
)

type FPSUpdateThreadSafetySuite struct {
	suite.Suite
}

func TestFPSUpdateThreadSafety(t *testing.T) {
	suite.Run(t, new(FPSUpdateThreadSafetySuite))
}

// TestFPSLoopUsesCallback verifies that the FPS update loop calls an
// injectable OnFPSUpdate callback rather than directly mutating a Fyne
// widget from a background goroutine. In production the callback wraps
// the label mutation in fyne.Do(), ensuring thread safety on Wayland.
func (s *FPSUpdateThreadSafetySuite) TestFPSLoopUsesCallback() {
	var callCount atomic.Int64

	// Create an FPSLoop with a custom callback that records invocations.
	loop := characteruat.NewFPSLoop(characteruat.FPSLoopConfig{
		Counter:  characteruat.NewFPSCounter(),
		Interval: 50 * time.Millisecond,
		OnFPSUpdate: func(fpsText string) {
			callCount.Add(1)
		},
	})

	// Start the loop and let it run a few ticks.
	loop.Start()
	time.Sleep(200 * time.Millisecond)
	loop.Stop()

	// The callback must have been invoked at least once.
	s.Greater(callCount.Load(), int64(0),
		"OnFPSUpdate callback should be called by the FPS loop")
}

// TestFPSLoopCallbackReceivesFormattedText verifies that the callback
// receives a formatted FPS string (e.g. "FPS: 0.0") rather than raw data.
func (s *FPSUpdateThreadSafetySuite) TestFPSLoopCallbackReceivesFormattedText() {
	var lastText atomic.Value

	loop := characteruat.NewFPSLoop(characteruat.FPSLoopConfig{
		Counter:  characteruat.NewFPSCounter(),
		Interval: 50 * time.Millisecond,
		OnFPSUpdate: func(fpsText string) {
			lastText.Store(fpsText)
		},
	})

	loop.Start()
	time.Sleep(150 * time.Millisecond)
	loop.Stop()

	got, ok := lastText.Load().(string)
	s.Require().True(ok, "OnFPSUpdate should have been called with a string")
	s.Contains(got, "FPS:", "callback text should contain 'FPS:' prefix")
}

// TestFPSLoopStopsCleanly verifies that Stop() halts the loop and no
// further callbacks are made after Stop returns.
func (s *FPSUpdateThreadSafetySuite) TestFPSLoopStopsCleanly() {
	var callCount atomic.Int64

	loop := characteruat.NewFPSLoop(characteruat.FPSLoopConfig{
		Counter:  characteruat.NewFPSCounter(),
		Interval: 50 * time.Millisecond,
		OnFPSUpdate: func(fpsText string) {
			callCount.Add(1)
		},
	})

	loop.Start()
	time.Sleep(200 * time.Millisecond)
	loop.Stop()

	countAtStop := callCount.Load()
	time.Sleep(200 * time.Millisecond)
	countAfterWait := callCount.Load()

	s.Equal(countAtStop, countAfterWait,
		"no further callbacks should fire after Stop()")
}

// TestFPSLoopMultipleStopCallsSafe verifies that calling Stop() multiple
// times doesn't panic (closing an already-closed channel would panic).
func (s *FPSUpdateThreadSafetySuite) TestFPSLoopMultipleStopCallsSafe() {
	loop := characteruat.NewFPSLoop(characteruat.FPSLoopConfig{
		Counter:     characteruat.NewFPSCounter(),
		Interval:    50 * time.Millisecond,
		OnFPSUpdate: func(fpsText string) {},
	})

	loop.Start()
	time.Sleep(100 * time.Millisecond)

	// Multiple Stop() calls should not panic.
	s.NotPanics(func() {
		loop.Stop()
		loop.Stop()
		loop.Stop()
	}, "calling Stop() multiple times should be safe")
}

// TestNewFPSLoopValidatesConfig verifies that NewFPSLoop panics on invalid config.
func (s *FPSUpdateThreadSafetySuite) TestNewFPSLoopValidatesConfig() {
	validConfig := characteruat.FPSLoopConfig{
		Counter:     characteruat.NewFPSCounter(),
		Interval:    50 * time.Millisecond,
		OnFPSUpdate: func(string) {},
	}

	// Valid config should not panic.
	s.NotPanics(func() {
		characteruat.NewFPSLoop(validConfig)
	}, "valid config should not panic")

	// Nil counter should panic.
	s.Panics(func() {
		cfg := validConfig
		cfg.Counter = nil
		characteruat.NewFPSLoop(cfg)
	}, "nil Counter should panic")

	// Zero interval should panic.
	s.Panics(func() {
		cfg := validConfig
		cfg.Interval = 0
		characteruat.NewFPSLoop(cfg)
	}, "zero Interval should panic")

	// Negative interval should panic.
	s.Panics(func() {
		cfg := validConfig
		cfg.Interval = -1 * time.Millisecond
		characteruat.NewFPSLoop(cfg)
	}, "negative Interval should panic")

	// Nil callback should panic.
	s.Panics(func() {
		cfg := validConfig
		cfg.OnFPSUpdate = nil
		characteruat.NewFPSLoop(cfg)
	}, "nil OnFPSUpdate should panic")
}
