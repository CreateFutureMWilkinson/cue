package presenter_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/ui/presenter"
)

// --- Mock ActivitySource ---

type mockActivitySource struct {
	ch chan presenter.ActivityEvent
}

func newMockActivitySource() *mockActivitySource {
	return &mockActivitySource{
		ch: make(chan presenter.ActivityEvent, 100),
	}
}

func (m *mockActivitySource) Events() <-chan presenter.ActivityEvent {
	return m.ch
}

// --- Suite ---

type ActivityPresenterSuite struct {
	suite.Suite
	source *mockActivitySource
}

func TestActivityPresenter(t *testing.T) {
	suite.Run(t, new(ActivityPresenterSuite))
}

func (s *ActivityPresenterSuite) SetupTest() {
	s.source = newMockActivitySource()
}

// Test 1: Constructor rejects nil source.
func (s *ActivityPresenterSuite) TestNewActivityPresenter_NilSource_ReturnsError() {
	p, err := presenter.NewActivityPresenter(nil, 500)
	s.Nil(p)
	s.Error(err)
	s.Contains(err.Error(), "source")
}

// Test 2: Constructor rejects maxEntries <= 0.
func (s *ActivityPresenterSuite) TestNewActivityPresenter_ZeroMaxEntries_ReturnsError() {
	p, err := presenter.NewActivityPresenter(s.source, 0)
	s.Nil(p)
	s.Error(err)
	s.Contains(err.Error(), "maxEntries")
}

func (s *ActivityPresenterSuite) TestNewActivityPresenter_NegativeMaxEntries_ReturnsError() {
	p, err := presenter.NewActivityPresenter(s.source, -1)
	s.Nil(p)
	s.Error(err)
	s.Contains(err.Error(), "maxEntries")
}

// Test 3: Start reads events from source and appends to entries.
func (s *ActivityPresenterSuite) TestStart_ConsumesEventsFromSource() {
	p, err := presenter.NewActivityPresenter(s.source, 500)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p.Start(ctx)

	s.source.ch <- presenter.ActivityEvent{
		Source:  "Slack",
		Message: "fetched 12 messages",
		IsError: false,
	}

	s.Eventually(func() bool {
		return len(p.Entries()) == 1
	}, time.Second, 10*time.Millisecond)

	entries := p.Entries()
	s.Equal("Slack", entries[0].Source)
	s.Equal("fetched 12 messages", entries[0].Message)
	s.False(entries[0].IsError)

	p.Stop()
}

// Test 4: Entries returns events newest-first.
func (s *ActivityPresenterSuite) TestEntries_NewestFirst() {
	p, err := presenter.NewActivityPresenter(s.source, 500)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p.Start(ctx)

	s.source.ch <- presenter.ActivityEvent{Source: "Slack", Message: "first"}
	s.source.ch <- presenter.ActivityEvent{Source: "Email", Message: "second"}
	s.source.ch <- presenter.ActivityEvent{Source: "Slack", Message: "third"}

	s.Eventually(func() bool {
		return len(p.Entries()) == 3
	}, time.Second, 10*time.Millisecond)

	entries := p.Entries()
	s.Equal("third", entries[0].Message)
	s.Equal("second", entries[1].Message)
	s.Equal("first", entries[2].Message)

	p.Stop()
}

// Test 5: OnUpdate callback invoked on each new event.
func (s *ActivityPresenterSuite) TestSetOnUpdate_CalledOnEachEvent() {
	p, err := presenter.NewActivityPresenter(s.source, 500)
	s.Require().NoError(err)

	var mu sync.Mutex
	callCount := 0
	p.SetOnUpdate(func() {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p.Start(ctx)

	s.source.ch <- presenter.ActivityEvent{Source: "Slack", Message: "one"}
	s.source.ch <- presenter.ActivityEvent{Source: "Email", Message: "two"}

	s.Eventually(func() bool {
		mu.Lock()
		defer mu.Unlock()
		return callCount == 2
	}, time.Second, 10*time.Millisecond)

	p.Stop()
}

// Test 6: Stop halts the goroutine — no more events consumed after stop.
func (s *ActivityPresenterSuite) TestStop_HaltsEventConsumption() {
	p, err := presenter.NewActivityPresenter(s.source, 500)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p.Start(ctx)

	s.source.ch <- presenter.ActivityEvent{Source: "Slack", Message: "before stop"}

	s.Eventually(func() bool {
		return len(p.Entries()) == 1
	}, time.Second, 10*time.Millisecond)

	p.Stop()

	// Send event after stop — should not be consumed.
	s.source.ch <- presenter.ActivityEvent{Source: "Email", Message: "after stop"}

	// Give a brief window for any erroneous consumption.
	time.Sleep(50 * time.Millisecond)

	s.Equal(1, len(p.Entries()))
	s.Equal("before stop", p.Entries()[0].Message)
}

// Test 7: Ring buffer caps at maxEntries — oldest entries dropped when full.
func (s *ActivityPresenterSuite) TestRingBuffer_CapsAtMaxEntries() {
	maxEntries := 3
	p, err := presenter.NewActivityPresenter(s.source, maxEntries)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p.Start(ctx)

	for i := 0; i < 5; i++ {
		s.source.ch <- presenter.ActivityEvent{
			Source:  "Slack",
			Message: time.Now().Format(time.RFC3339Nano),
		}
	}

	s.Eventually(func() bool {
		return len(p.Entries()) == maxEntries
	}, time.Second, 10*time.Millisecond)

	// Should have exactly maxEntries, not 5.
	s.Equal(maxEntries, len(p.Entries()))

	p.Stop()
}

// Test 8: Error flag preserved correctly on ActivityEntry.
func (s *ActivityPresenterSuite) TestErrorFlag_PreservedOnEntry() {
	p, err := presenter.NewActivityPresenter(s.source, 500)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p.Start(ctx)

	s.source.ch <- presenter.ActivityEvent{Source: "Email", Message: "connection error", IsError: true}
	s.source.ch <- presenter.ActivityEvent{Source: "Slack", Message: "ok", IsError: false}

	s.Eventually(func() bool {
		return len(p.Entries()) == 2
	}, time.Second, 10*time.Millisecond)

	entries := p.Entries()
	// Newest first: "ok" at [0], "connection error" at [1].
	s.False(entries[0].IsError)
	s.True(entries[1].IsError)

	p.Stop()
}

// Test 9: Timestamp assigned to each entry (non-zero).
func (s *ActivityPresenterSuite) TestTimestamp_AssignedNonZero() {
	p, err := presenter.NewActivityPresenter(s.source, 500)
	s.Require().NoError(err)

	before := time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p.Start(ctx)

	s.source.ch <- presenter.ActivityEvent{Source: "Slack", Message: "event"}

	s.Eventually(func() bool {
		return len(p.Entries()) == 1
	}, time.Second, 10*time.Millisecond)

	entry := p.Entries()[0]
	s.False(entry.Timestamp.IsZero())
	s.True(entry.Timestamp.After(before) || entry.Timestamp.Equal(before))
	s.True(entry.Timestamp.Before(time.Now().Add(time.Second)))

	p.Stop()
}

// Test 10: Entries returns empty slice before any events.
func (s *ActivityPresenterSuite) TestEntries_EmptyBeforeAnyEvents() {
	p, err := presenter.NewActivityPresenter(s.source, 500)
	s.Require().NoError(err)

	entries := p.Entries()
	s.NotNil(entries)
	s.Empty(entries)
}
