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
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	cal, err := ics.ParseCalendar(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parsing ICS: %w", err)
	}

	if cal == nil {
		return nil, fmt.Errorf("parsing ICS: nil calendar")
	}

	components := cal.Events()
	if len(components) == 0 && strings.Contains(string(body), "VEVENT") {
		return nil, fmt.Errorf("parsing ICS: failed to parse events")
	}

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.AddDate(0, 0, 1)

	var events []CalendarEvent
	for _, event := range components {
		uid := event.Id()
		summary := event.GetProperty(ics.ComponentPropertySummary)
		if summary == nil {
			continue
		}

		dtstart := event.GetProperty(ics.ComponentPropertyDtStart)
		dtend := event.GetProperty(ics.ComponentPropertyDtEnd)
		if dtstart == nil || dtend == nil {
			continue
		}

		allDay := false
		var start, end time.Time

		valueParam := dtstart.ICalParameters["VALUE"]
		if len(valueParam) > 0 && valueParam[0] == "DATE" {
			allDay = true
			start, err = time.Parse("20060102", dtstart.Value)
			if err != nil {
				continue
			}
			end, err = time.Parse("20060102", dtend.Value)
			if err != nil {
				continue
			}
		} else {
			start, err = time.Parse("20060102T150405Z", dtstart.Value)
			if err != nil {
				continue
			}
			end, err = time.Parse("20060102T150405Z", dtend.Value)
			if err != nil {
				continue
			}
		}

		if eventOverlapsDay(start, end, dayStart, dayEnd) {
			events = append(events, CalendarEvent{
				ID:     uid,
				Title:  summary.Value,
				Start:  start,
				End:    end,
				AllDay: allDay,
			})
		}
	}

	return events, nil
}

func eventOverlapsDay(eventStart, eventEnd, dayStart, dayEnd time.Time) bool {
	return eventStart.Before(dayEnd) && eventEnd.After(dayStart)
}
