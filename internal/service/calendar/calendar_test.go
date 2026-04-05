package calendar_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/service/calendar"

	"github.com/stretchr/testify/suite"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

// mockHTTPClient implements calendar.HTTPClient for testing.
type mockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *mockHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	return m.response, m.err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const (
	testURL     = "testURL"
	testTimeout = 10 * time.Second
)

func mockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

const validICS = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
UID:event1@example.com
SUMMARY:Team Standup
DTSTART:20260329T090000Z
DTEND:20260329T093000Z
END:VEVENT
BEGIN:VEVENT
UID:event2@example.com
SUMMARY:Lunch Meeting
DTSTART:20260330T120000Z
DTEND:20260330T130000Z
END:VEVENT
BEGIN:VEVENT
UID:event3@example.com
SUMMARY:All Day Event
DTSTART;VALUE=DATE:20260329
DTEND;VALUE=DATE:20260330
END:VEVENT
END:VCALENDAR`

const emptyICS = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
END:VCALENDAR`

const multiDayICS = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VEVENT
UID:multiday@example.com
SUMMARY:Conference
DTSTART:20260329T090000Z
DTEND:20260331T170000Z
END:VEVENT
END:VCALENDAR`

const invalidICS = `this is not valid ics data at all`

// targetDate is 2026-03-29 in UTC.
var targetDate = time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

type ICSProviderSuite struct {
	suite.Suite
}

func TestICSProvider(t *testing.T) {
	suite.Run(t, new(ICSProviderSuite))
}

// ---------------------------------------------------------------------------
// Constructor validation
// ---------------------------------------------------------------------------

func (s *ICSProviderSuite) TestNewICSProvider_EmptyURL() {
	_, err := calendar.NewICSProvider("", &mockHTTPClient{}, testTimeout)
	s.Error(err)
	s.Contains(err.Error(), "url")
}

func (s *ICSProviderSuite) TestNewICSProvider_NilHTTPClient() {
	_, err := calendar.NewICSProvider("testURL", nil, testTimeout)
	s.Error(err)
	s.Contains(err.Error(), "http")
}

func (s *ICSProviderSuite) TestNewICSProvider_ZeroTimeout() {
	_, err := calendar.NewICSProvider("testURL", &mockHTTPClient{}, 0)
	s.Error(err)
	s.Contains(err.Error(), "timeout")
}

func (s *ICSProviderSuite) TestNewICSProvider_ValidArgs() {
	provider, err := calendar.NewICSProvider(
		"testURL",
		&mockHTTPClient{},
		testTimeout,
	)
	s.NoError(err)
	s.NotNil(provider)
}

// ---------------------------------------------------------------------------
// FetchEvents — valid ICS parsing
// ---------------------------------------------------------------------------

func (s *ICSProviderSuite) TestFetchEvents_ParsesValidICS() {
	client := &mockHTTPClient{
		response: mockResponse(200, validICS),
	}
	provider, err := calendar.NewICSProvider(
		"testURL",
		client,
		testTimeout,
	)
	s.Require().NoError(err)

	events, err := provider.FetchEvents(context.Background(), targetDate)
	s.NoError(err)

	// validICS has 2 events on 2026-03-29: "Team Standup" and "All Day Event".
	s.Require().Len(events, 2)

	// Find the timed event and all-day event.
	var standup, allDayEvt calendar.CalendarEvent
	for _, e := range events {
		if e.ID == "event1@example.com" {
			standup = e
		}
		if e.ID == "event3@example.com" {
			allDayEvt = e
		}
	}

	s.Equal("Team Standup", standup.Title)
	s.Equal(time.Date(2026, 3, 29, 9, 0, 0, 0, time.UTC), standup.Start)
	s.Equal(time.Date(2026, 3, 29, 9, 30, 0, 0, time.UTC), standup.End)
	s.False(standup.AllDay)

	s.Equal("All Day Event", allDayEvt.Title)
	s.True(allDayEvt.AllDay)
}

// ---------------------------------------------------------------------------
// FetchEvents — date filtering
// ---------------------------------------------------------------------------

func (s *ICSProviderSuite) TestFetchEvents_FiltersByDate() {
	client := &mockHTTPClient{
		response: mockResponse(200, validICS),
	}
	provider, err := calendar.NewICSProvider(
		"testURL",
		client,
		testTimeout,
	)
	s.Require().NoError(err)

	// Query for 2026-03-30 — only "Lunch Meeting" should appear.
	march30 := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)
	events, err := provider.FetchEvents(context.Background(), march30)
	s.NoError(err)
	s.Require().Len(events, 1)
	s.Equal("event2@example.com", events[0].ID)
	s.Equal("Lunch Meeting", events[0].Title)
}

// ---------------------------------------------------------------------------
// FetchEvents — all-day events
// ---------------------------------------------------------------------------

func (s *ICSProviderSuite) TestFetchEvents_AllDayEvent() {
	client := &mockHTTPClient{
		response: mockResponse(200, validICS),
	}
	provider, err := calendar.NewICSProvider(
		"testURL",
		client,
		testTimeout,
	)
	s.Require().NoError(err)

	events, err := provider.FetchEvents(context.Background(), targetDate)
	s.NoError(err)

	// Find the all-day event.
	var found bool
	for _, e := range events {
		if e.ID == "event3@example.com" {
			found = true
			s.Equal("All Day Event", e.Title)
			s.True(e.AllDay)
			// All-day start should be midnight of the date.
			s.Equal(time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC), e.Start)
		}
	}
	s.True(found, "all-day event should be present in results")
}

// ---------------------------------------------------------------------------
// FetchEvents — empty feed
// ---------------------------------------------------------------------------

func (s *ICSProviderSuite) TestFetchEvents_EmptyFeed() {
	client := &mockHTTPClient{
		response: mockResponse(200, emptyICS),
	}
	provider, err := calendar.NewICSProvider(
		"testURL",
		client,
		testTimeout,
	)
	s.Require().NoError(err)

	events, err := provider.FetchEvents(context.Background(), targetDate)
	s.NoError(err)
	s.Empty(events)
}

// ---------------------------------------------------------------------------
// FetchEvents — HTTP error
// ---------------------------------------------------------------------------

func (s *ICSProviderSuite) TestFetchEvents_HTTPError() {
	client := &mockHTTPClient{
		err: errors.New("connection refused"),
	}
	provider, err := calendar.NewICSProvider(
		"testURL",
		client,
		testTimeout,
	)
	s.Require().NoError(err)

	events, err := provider.FetchEvents(context.Background(), targetDate)
	s.Error(err)
	s.Nil(events)
	s.Contains(err.Error(), "connection refused")
}

// ---------------------------------------------------------------------------
// FetchEvents — non-200 status codes
// ---------------------------------------------------------------------------

func (s *ICSProviderSuite) TestFetchEvents_HTTPNotFound() {
	client := &mockHTTPClient{
		response: mockResponse(404, "not found"),
	}
	provider, err := calendar.NewICSProvider(
		"testURL",
		client,
		testTimeout,
	)
	s.Require().NoError(err)

	events, err := provider.FetchEvents(context.Background(), targetDate)
	s.Error(err)
	s.Nil(events)
	s.Contains(err.Error(), "404")
}

func (s *ICSProviderSuite) TestFetchEvents_HTTPInternalServerError() {
	client := &mockHTTPClient{
		response: mockResponse(500, "internal server error"),
	}
	provider, err := calendar.NewICSProvider(
		"testURL",
		client,
		testTimeout,
	)
	s.Require().NoError(err)

	events, err := provider.FetchEvents(context.Background(), targetDate)
	s.Error(err)
	s.Nil(events)
	s.Contains(err.Error(), "500")
}

// ---------------------------------------------------------------------------
// FetchEvents — invalid ICS data
// ---------------------------------------------------------------------------

func (s *ICSProviderSuite) TestFetchEvents_InvalidICSData() {
	client := &mockHTTPClient{
		response: mockResponse(200, invalidICS),
	}
	provider, err := calendar.NewICSProvider(
		"testURL",
		client,
		testTimeout,
	)
	s.Require().NoError(err)

	events, err := provider.FetchEvents(context.Background(), targetDate)
	s.Error(err)
	s.Nil(events)
}

// ---------------------------------------------------------------------------
// FetchEvents — context cancellation (timeout)
// ---------------------------------------------------------------------------

func (s *ICSProviderSuite) TestFetchEvents_ContextCancelled() {
	client := &mockHTTPClient{
		err: context.Canceled,
	}
	provider, err := calendar.NewICSProvider(
		"testURL",
		client,
		testTimeout,
	)
	s.Require().NoError(err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	events, err := provider.FetchEvents(ctx, targetDate)
	s.Error(err)
	s.Nil(events)
	s.True(errors.Is(err, context.Canceled))
}

func (s *ICSProviderSuite) TestFetchEvents_ContextDeadlineExceeded() {
	client := &mockHTTPClient{
		err: context.DeadlineExceeded,
	}
	provider, err := calendar.NewICSProvider(
		"testURL",
		client,
		testTimeout,
	)
	s.Require().NoError(err)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))
	defer cancel()

	events, err := provider.FetchEvents(ctx, targetDate)
	s.Error(err)
	s.Nil(events)
	s.True(errors.Is(err, context.DeadlineExceeded))
}

// ---------------------------------------------------------------------------
// FetchEvents — multi-day event
// ---------------------------------------------------------------------------

func (s *ICSProviderSuite) TestFetchEvents_MultiDayEventAppearsOnEachDay() {
	client := &mockHTTPClient{
		response: mockResponse(200, multiDayICS),
	}
	provider, err := calendar.NewICSProvider(
		"testURL",
		client,
		testTimeout,
	)
	s.Require().NoError(err)

	// The conference runs 2026-03-29 09:00 to 2026-03-31 17:00.
	// It should appear on March 29, 30, and 31.

	march29 := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
	events29, err := provider.FetchEvents(context.Background(), march29)
	s.NoError(err)
	s.Require().Len(events29, 1)
	s.Equal("Conference", events29[0].Title)

	// Need a fresh response body for each call since the reader is consumed.
	client.response = mockResponse(200, multiDayICS)
	march30 := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)
	events30, err := provider.FetchEvents(context.Background(), march30)
	s.NoError(err)
	s.Require().Len(events30, 1)
	s.Equal("Conference", events30[0].Title)

	client.response = mockResponse(200, multiDayICS)
	march31 := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	events31, err := provider.FetchEvents(context.Background(), march31)
	s.NoError(err)
	s.Require().Len(events31, 1)
	s.Equal("Conference", events31[0].Title)

	// Should NOT appear on April 1 (event ends 2026-03-31 17:00).
	client.response = mockResponse(200, multiDayICS)
	april1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	eventsApril, err := provider.FetchEvents(context.Background(), april1)
	s.NoError(err)
	s.Empty(eventsApril)
}

// ---------------------------------------------------------------------------
// CalendarProvider interface compliance
// ---------------------------------------------------------------------------

func (s *ICSProviderSuite) TestICSProvider_ImplementsCalendarProvider() {
	client := &mockHTTPClient{}
	provider, err := calendar.NewICSProvider(
		"testURL",
		client,
		testTimeout,
	)
	s.Require().NoError(err)

	// Compile-time check: *ICSProvider satisfies CalendarProvider.
	var _ calendar.CalendarProvider = provider
}
