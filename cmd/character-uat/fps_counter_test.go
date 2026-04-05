package characteruat_test

import (
	"sync"
	"testing"
	"time"

	characteruat "github.com/CreateFutureMWilkinson/cue/cmd/character-uat"

	"github.com/stretchr/testify/suite"
)

type FPSCounterSuite struct {
	suite.Suite
}

func TestFPSCounter(t *testing.T) {
	suite.Run(t, new(FPSCounterSuite))
}

// --- Test Case 1: Tick counting and FPS calculation ---

func (s *FPSCounterSuite) TestFPSAfterTicks() {
	counter := characteruat.NewFPSCounter()

	// Simulate 60 ticks over approximately 1 second.
	ticks := 60
	interval := time.Second / time.Duration(ticks)

	for i := 0; i < ticks; i++ {
		counter.Tick()
		time.Sleep(interval)
	}

	fps := counter.FPS()

	// Allow generous tolerance: expect roughly 60 FPS (+/- 30).
	// The key assertion is that FPS is positive and in a reasonable range.
	s.Greater(fps, 20.0, "FPS should be greater than 20 after 60 ticks in ~1s")
	s.Less(fps, 120.0, "FPS should be less than 120 after 60 ticks in ~1s")
}

func (s *FPSCounterSuite) TestFPSProportionalToTickRate() {
	// Slower tick rate should yield lower FPS.
	counter := characteruat.NewFPSCounter()

	ticks := 10
	interval := 100 * time.Millisecond // 10 ticks over ~1s = ~10 FPS

	for i := 0; i < ticks; i++ {
		counter.Tick()
		time.Sleep(interval)
	}

	fps := counter.FPS()
	s.Greater(fps, 3.0, "FPS should be greater than 3 at ~10 ticks/sec")
	s.Less(fps, 30.0, "FPS should be less than 30 at ~10 ticks/sec")
}

// --- Test Case 2: Zero-time edge cases ---

func (s *FPSCounterSuite) TestFPSReturnsZeroBeforeAnyTicks() {
	counter := characteruat.NewFPSCounter()

	fps := counter.FPS()
	s.Equal(0.0, fps, "FPS should be 0 when no ticks have occurred")
}

func (s *FPSCounterSuite) TestFPSReturnsZeroImmediatelyAfterCreation() {
	counter := characteruat.NewFPSCounter()

	// Call FPS immediately, no ticks at all.
	s.Equal(0.0, counter.FPS(), "FPS should be 0 immediately after NewFPSCounter()")
}

// --- Test Case 3: Concurrent access thread safety ---

func (s *FPSCounterSuite) TestConcurrentTickAndFPS() {
	counter := characteruat.NewFPSCounter()

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Spawn multiple goroutines calling Tick().
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					counter.Tick()
				}
			}
		}()
	}

	// Spawn multiple goroutines calling FPS().
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					_ = counter.FPS()
				}
			}
		}()
	}

	// Let them race for a bit.
	time.Sleep(500 * time.Millisecond)
	close(done)
	wg.Wait()

	// If we get here without a race detector panic, the test passes.
	// Also verify FPS returns something reasonable (positive after many ticks).
	fps := counter.FPS()
	s.GreaterOrEqual(fps, 0.0, "FPS should be non-negative after concurrent access")
}
