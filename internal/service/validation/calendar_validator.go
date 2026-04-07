package validation

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	ics "github.com/arran4/golang-ical"
)

// ICSValidatorOption is a functional option for configuring ICSValidator.
type ICSValidatorOption func(*ICSValidator)

// WithHTTPClient sets the HTTP client for the ICS validator.
func WithHTTPClient(client *http.Client) ICSValidatorOption {
	return func(v *ICSValidator) {
		v.httpClient = client
	}
}

// ICSValidator validates calendar ICS URLs by fetching and parsing the content.
type ICSValidator struct {
	httpClient *http.Client
}

// NewICSValidator creates a new ICSValidator with the given options.
func NewICSValidator(opts ...ICSValidatorOption) *ICSValidator {
	v := &ICSValidator{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// ValidateCalendar fetches the given ICS URL and verifies it returns valid iCalendar data.
func (v *ICSValidator) ValidateCalendar(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating calendar request: %w", err)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching calendar URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("calendar fetch failed: HTTP %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, 1<<20) // 1 MB limit
	if _, err := ics.ParseCalendar(limited); err != nil {
		return fmt.Errorf("invalid iCalendar response: %w", err)
	}

	return nil
}
