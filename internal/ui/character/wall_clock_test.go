package character_test

import (
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/character"
	"github.com/stretchr/testify/suite"
)

// WallClockSuite tests the WallClock implementation.
type WallClockSuite struct {
	suite.Suite
}

func TestWallClock(t *testing.T) {
	suite.Run(t, new(WallClockSuite))
}

func (s *WallClockSuite) TestWallClockNowReturnsReasonableTime() {
	clock := character.WallClock{}
	before := time.Now()
	got := clock.Now()
	after := time.Now()

	s.True(got.After(before) || got.Equal(before),
		"WallClock.Now() should be >= time before call")
	s.True(got.Before(after) || got.Equal(after),
		"WallClock.Now() should be <= time after call")
	s.InDelta(0, got.Sub(before).Seconds(), 1.0,
		"WallClock.Now() should be within 1 second of time.Now()")
}

func (s *WallClockSuite) TestWallClockNewTickerDelivers() {
	clock := character.WallClock{}
	ticker := clock.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	select {
	case t := <-ticker.Chan():
		s.False(t.IsZero(), "ticker should deliver a non-zero time")
	case <-time.After(200 * time.Millisecond):
		s.Fail("WallClock.NewTicker(50ms) did not deliver within 200ms")
	}
}
