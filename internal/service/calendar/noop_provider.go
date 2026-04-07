package calendar

import (
	"context"
	"errors"
	"time"
)

var errNotImplemented = errors.New("not implemented")

// NoopCalendarProvider returns empty events. Used when no calendar is configured.
type NoopCalendarProvider struct{}

// NewNoopCalendarProvider creates a NoopCalendarProvider.
func NewNoopCalendarProvider() *NoopCalendarProvider {
	return &NoopCalendarProvider{}
}

// FetchEvents returns an error (stub).
func (n *NoopCalendarProvider) FetchEvents(ctx context.Context, date time.Time) ([]CalendarEvent, error) {
	return nil, errNotImplemented
}
