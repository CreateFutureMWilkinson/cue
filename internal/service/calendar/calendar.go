package calendar

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
)

const (
	// Time format constants for ICS parsing
	dateOnlyFormat = "20060102"
	dateTimeFormat = "20060102T150405Z"

	// ICS property values
	valueTypeDate = "DATE"
)

type CalendarEvent struct {
	ID     string
	Title  string
	Start  time.Time
	End    time.Time
	AllDay bool
}

type CalendarProvider interface {
	FetchEvents(ctx context.Context, date time.Time) ([]CalendarEvent, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type ICSProvider struct {
	url        string
	httpClient HTTPClient
	timeout    time.Duration
}

func NewICSProvider(url string, httpClient HTTPClient, timeout time.Duration) (*ICSProvider, error) {
	if url == "" {
		return nil, fmt.Errorf("url must not be empty")
	}
	if httpClient == nil {
		return nil, fmt.Errorf("httpClient must not be nil")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive")
	}
	return &ICSProvider{
		url:        url,
		httpClient: httpClient,
		timeout:    timeout,
	}, nil
}

func (p *ICSProvider) FetchEvents(ctx context.Context, date time.Time) ([]CalendarEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ICS fetch failed with HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	cal, err := ics.ParseCalendar(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parsing ICS calendar: %w", err)
	}

	if cal == nil {
		return nil, fmt.Errorf("parsing ICS calendar: received nil calendar")
	}

	icsEvents := cal.Events()
	if len(icsEvents) == 0 && strings.Contains(string(body), "VEVENT") {
		return nil, fmt.Errorf("parsing ICS calendar: failed to parse events from VEVENT data")
	}

	dayStart, dayEnd := calculateDayBounds(date)

	var events []CalendarEvent
	for _, icsEvent := range icsEvents {
		eventUID := icsEvent.Id()
		summaryProp := icsEvent.GetProperty(ics.ComponentPropertySummary)
		if summaryProp == nil {
			continue
		}

		startTimeProp := icsEvent.GetProperty(ics.ComponentPropertyDtStart)
		endTimeProp := icsEvent.GetProperty(ics.ComponentPropertyDtEnd)
		if startTimeProp == nil || endTimeProp == nil {
			continue
		}

		eventStart, eventEnd, isAllDay, err := parseEventTimes(startTimeProp, endTimeProp)
		if err != nil {
			continue
		}

		if eventOverlapsDay(eventStart, eventEnd, dayStart, dayEnd) {
			events = append(events, CalendarEvent{
				ID:     eventUID,
				Title:  summaryProp.Value,
				Start:  eventStart,
				End:    eventEnd,
				AllDay: isAllDay,
			})
		}
	}

	return events, nil
}

// calculateDayBounds returns the start and end time boundaries for the given date.
// Start is midnight UTC of the date, end is midnight UTC of the next day.
func calculateDayBounds(date time.Time) (dayStart, dayEnd time.Time) {
	dayStart = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd = dayStart.AddDate(0, 0, 1)
	return dayStart, dayEnd
}

// parseEventTimes extracts start and end times from ICS event properties,
// determining whether the event is all-day based on VALUE=DATE parameter.
func parseEventTimes(startProp, endProp *ics.IANAProperty) (start, end time.Time, isAllDay bool, err error) {
	if startProp == nil || endProp == nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("missing start or end time property")
	}

	valueParam := startProp.ICalParameters["VALUE"]
	if len(valueParam) > 0 && valueParam[0] == valueTypeDate {
		// All-day event
		isAllDay = true
		start, err = time.Parse(dateOnlyFormat, startProp.Value)
		if err != nil {
			return time.Time{}, time.Time{}, false, fmt.Errorf("parsing all-day start date: %w", err)
		}
		end, err = time.Parse(dateOnlyFormat, endProp.Value)
		if err != nil {
			return time.Time{}, time.Time{}, false, fmt.Errorf("parsing all-day end date: %w", err)
		}
	} else {
		// Timed event
		start, err = time.Parse(dateTimeFormat, startProp.Value)
		if err != nil {
			return time.Time{}, time.Time{}, false, fmt.Errorf("parsing start datetime: %w", err)
		}
		end, err = time.Parse(dateTimeFormat, endProp.Value)
		if err != nil {
			return time.Time{}, time.Time{}, false, fmt.Errorf("parsing end datetime: %w", err)
		}
	}

	return start, end, isAllDay, nil
}

func eventOverlapsDay(eventStart, eventEnd, dayStart, dayEnd time.Time) bool {
	return eventStart.Before(dayEnd) && eventEnd.After(dayStart)
}
