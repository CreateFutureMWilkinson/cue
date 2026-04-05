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

// GetUserChannels returns the list of channels the authenticated user belongs to.
func (c *SlackWebClient) GetUserChannels(ctx context.Context) ([]SlackChannel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/conversations.list", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkHTTPStatus(resp); err != nil {
		return nil, err
	}

	var result slackChannelResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	channels := make([]SlackChannel, len(result.Channels))
	for i, ch := range result.Channels {
		channels[i] = SlackChannel{ID: ch.ID, Name: ch.Name}
	}
	return channels, nil
}

// GetChannelMessages returns messages from a channel, optionally filtering by oldest timestamp.
func (c *SlackWebClient) GetChannelMessages(ctx context.Context, channelID string, oldest string) ([]SlackMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/conversations.history", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	q := req.URL.Query()
	q.Set("channel", channelID)
	if oldest != "" {
		q.Set("oldest", oldest)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkHTTPStatus(resp); err != nil {
		return nil, err
	}

	var result slackMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return convertMessages(result.Messages), nil
}

// GetThreadReplies returns messages in a thread.
func (c *SlackWebClient) GetThreadReplies(ctx context.Context, channelID string, threadTS string) ([]SlackMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/conversations.replies", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	q := req.URL.Query()
	q.Set("channel", channelID)
	q.Set("ts", threadTS)
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkHTTPStatus(resp); err != nil {
		return nil, err
	}

	var result slackMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return convertMessages(result.Messages), nil
}

// checkHTTPStatus returns an error for non-200 HTTP status codes.
func checkHTTPStatus(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limited by Slack API")
	case http.StatusUnauthorized:
		return fmt.Errorf("slack auth error: invalid or expired token")
	default:
		return fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
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
