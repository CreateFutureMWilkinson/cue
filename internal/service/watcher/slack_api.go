package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackClientOption is a functional option for configuring SlackWebClient.
type SlackClientOption func(*SlackWebClient)

// WithBaseURL sets the base URL for the Slack API client.
func WithBaseURL(url string) SlackClientOption {
	return func(c *SlackWebClient) {
		c.baseURL = url
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) SlackClientOption {
	return func(c *SlackWebClient) {
		c.httpClient.Timeout = d
	}
}

// SlackWebClient is a real HTTP client for the Slack Web API.
type SlackWebClient struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// NewSlackWebClient creates a new SlackWebClient with the given token and options.
func NewSlackWebClient(token string, opts ...SlackClientOption) (*SlackWebClient, error) {
	if token == "" {
		return nil, fmt.Errorf("slack API token must not be empty")
	}

	c := &SlackWebClient{
		token:      token,
		baseURL:    "https://slack.com/api",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// slackChannelResponse is the JSON response from conversations.list.
type slackChannelResponse struct {
	OK       bool           `json:"ok"`
	Error    string         `json:"error,omitempty"`
	Channels []slackChannel `json:"channels"`
}

type slackChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// slackMessageResponse is the JSON response from conversations.history and conversations.replies.
type slackMessageResponse struct {
	OK       bool               `json:"ok"`
	Error    string             `json:"error,omitempty"`
	Messages []slackMessageJSON `json:"messages"`
}

type slackMessageJSON struct {
	TS       string `json:"ts"`
	User     string `json:"user"`
	Text     string `json:"text"`
	ThreadTS string `json:"thread_ts,omitempty"`
}

// doRequest is a helper method that handles common HTTP request patterns for Slack API calls.
func (c *SlackWebClient) doRequest(ctx context.Context, endpoint string, params map[string]string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating request for %s: %w", endpoint, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	if len(params) > 0 {
		q := req.URL.Query()
		for key, value := range params {
			if value != "" {
				q.Set(key, value)
			}
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if err := checkHTTPStatus(resp); err != nil {
		return fmt.Errorf("slack API error for %s: %w", endpoint, err)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding response from %s: %w", endpoint, err)
	}

	return nil
}

// GetUserChannels returns the list of channels the authenticated user belongs to.
func (c *SlackWebClient) GetUserChannels(ctx context.Context) ([]SlackChannel, error) {
	var result slackChannelResponse
	if err := c.doRequest(ctx, "/conversations.list", nil, &result); err != nil {
		return nil, err
	}

	channels := make([]SlackChannel, len(result.Channels))
	for i, ch := range result.Channels {
		channels[i] = SlackChannel{ID: ch.ID, Name: ch.Name}
	}
	return channels, nil
}

// GetChannelMessages returns messages from a channel, optionally filtering by oldest timestamp.
func (c *SlackWebClient) GetChannelMessages(ctx context.Context, channelID string, oldest string) ([]SlackMessage, error) {
	params := map[string]string{
		"channel": channelID,
		"oldest":  oldest,
	}

	var result slackMessageResponse
	if err := c.doRequest(ctx, "/conversations.history", params, &result); err != nil {
		return nil, err
	}

	return convertMessages(result.Messages), nil
}

// GetThreadReplies returns messages in a thread.
func (c *SlackWebClient) GetThreadReplies(ctx context.Context, channelID string, threadTS string) ([]SlackMessage, error) {
	params := map[string]string{
		"channel": channelID,
		"ts":      threadTS,
	}

	var result slackMessageResponse
	if err := c.doRequest(ctx, "/conversations.replies", params, &result); err != nil {
		return nil, err
	}

	return convertMessages(result.Messages), nil
}

// checkHTTPStatus returns an error for non-200 HTTP status codes with consistent messaging.
func checkHTTPStatus(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limited by Slack API")
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication failed: invalid or expired token")
	case http.StatusForbidden:
		return fmt.Errorf("forbidden: insufficient permissions")
	case http.StatusNotFound:
		return fmt.Errorf("resource not found")
	default:
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
}

// convertMessages maps raw JSON messages to SlackMessage structs.
func convertMessages(msgs []slackMessageJSON) []SlackMessage {
	result := make([]SlackMessage, len(msgs))
	for i, m := range msgs {
		result[i] = SlackMessage{
			ID:        m.TS,
			Sender:    m.User,
			Text:      m.Text,
			Timestamp: m.TS,
			ThreadTS:  m.ThreadTS,
		}
	}
	return result
}
