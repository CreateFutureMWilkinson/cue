package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackValidatorOption is a functional option for configuring SlackAPIValidator.
type SlackValidatorOption func(*SlackAPIValidator)

// WithSlackBaseURL sets the base URL for the Slack API validator.
func WithSlackBaseURL(url string) SlackValidatorOption {
	return func(v *SlackAPIValidator) {
		v.baseURL = url
	}
}

// SlackAPIValidator validates Slack credentials by calling the Slack auth.test API.
type SlackAPIValidator struct {
	baseURL    string
	httpClient *http.Client
}

// NewSlackAPIValidator creates a new SlackAPIValidator with the given options.
func NewSlackAPIValidator(opts ...SlackValidatorOption) *SlackAPIValidator {
	v := &SlackAPIValidator{
		baseURL:    "https://slack.com/api",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// slackAuthResponse is the JSON response from Slack's auth.test endpoint.
type slackAuthResponse struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	UserID string `json:"user_id,omitempty"`
	Team   string `json:"team,omitempty"`
}

// ValidateSlack validates a Slack token by calling the auth.test API endpoint.
func (v *SlackAPIValidator) ValidateSlack(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+"/auth.test", nil)
	if err != nil {
		return fmt.Errorf("creating Slack auth request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing Slack auth request: %w", err)
	}
	defer resp.Body.Close()

	var result slackAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decoding Slack auth response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("Slack authentication failed: %s", result.Error)
	}

	return nil
}
