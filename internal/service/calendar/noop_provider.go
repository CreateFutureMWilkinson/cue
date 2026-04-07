package calendar

import (
	"context"
	"time"
)

// NoopCalendarProvider returns empty events. Used when no calendar is configured.
type NoopCalendarProvider struct{}

// NewNoopCalendarProvider creates a NoopCalendarProvider.
func NewNoopCalendarProvider() *NoopCalendarProvider {
	return &NoopCalendarProvider{}
}

// FetchEvents always returns an empty slice and nil error.
func (n *NoopCalendarProvider) FetchEvents(_ context.Context, _ time.Time) ([]CalendarEvent, error) {
	return []CalendarEvent{}, nil
}
