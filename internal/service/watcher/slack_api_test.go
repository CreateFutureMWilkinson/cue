package watcher_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/service/watcher"
	"github.com/stretchr/testify/suite"
)

type SlackWebClientSuite struct {
	suite.Suite
}

func TestSlackWebClient(t *testing.T) {
	suite.Run(t, new(SlackWebClientSuite))
}

func (s *SlackWebClientSuite) TestConstructorRejectsEmptyToken() {
	client, err := watcher.NewSlackWebClient("")
	s.Error(err)
	s.Nil(client)
	s.Contains(err.Error(), "token")
}

func (s *SlackWebClientSuite) TestGetUserChannels() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/conversations.list", r.URL.Path)
		s.Equal("Bearer xoxp-test-token", r.Header.Get("Authorization"))

		resp := map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{"id": "C001", "name": "general"},
				{"id": "C002", "name": "random"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := watcher.NewSlackWebClient("xoxp-test-token", watcher.WithBaseURL(server.URL))
	s.Require().NoError(err)

	channels, err := client.GetUserChannels(context.Background())
	s.NoError(err)
	s.Require().Len(channels, 2)
	s.Equal("C001", channels[0].ID)
	s.Equal("general", channels[0].Name)
	s.Equal("C002", channels[1].ID)
	s.Equal("random", channels[1].Name)
}

func (s *SlackWebClientSuite) TestGetChannelMessages() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/conversations.history", r.URL.Path)
		s.Equal("C001", r.URL.Query().Get("channel"))

		resp := map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{
					"ts":   "1234.5678",
					"user": "U123",
					"text": "hello world",
				},
				{
					"ts":        "1234.5679",
					"user":      "U456",
					"text":      "threaded reply",
					"thread_ts": "1234.5678",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := watcher.NewSlackWebClient("xoxp-test-token", watcher.WithBaseURL(server.URL))
	s.Require().NoError(err)

	messages, err := client.GetChannelMessages(context.Background(), "C001", "")
	s.NoError(err)
	s.Require().Len(messages, 2)

	s.Equal("1234.5678", messages[0].Timestamp)
	s.Equal("U123", messages[0].Sender)
	s.Equal("hello world", messages[0].Text)

	s.Equal("1234.5679", messages[1].Timestamp)
	s.Equal("U456", messages[1].Sender)
	s.Equal("threaded reply", messages[1].Text)
	s.Equal("1234.5678", messages[1].ThreadTS)
}

func (s *SlackWebClientSuite) TestGetChannelMessagesWithOldest() {
	var capturedOldest string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedOldest = r.URL.Query().Get("oldest")

		resp := map[string]interface{}{
			"ok":       true,
			"messages": []map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := watcher.NewSlackWebClient("xoxp-test-token", watcher.WithBaseURL(server.URL))
	s.Require().NoError(err)

	_, err = client.GetChannelMessages(context.Background(), "C001", "1111.0000")
	s.NoError(err)
	s.Equal("1111.0000", capturedOldest)
}

func (s *SlackWebClientSuite) TestGetThreadReplies() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("/conversations.replies", r.URL.Path)
		s.Equal("C001", r.URL.Query().Get("channel"))
		s.Equal("1234.5678", r.URL.Query().Get("ts"))

		resp := map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{
					"ts":        "1234.5678",
					"user":      "U123",
					"text":      "parent message",
					"thread_ts": "1234.5678",
				},
				{
					"ts":        "1234.5679",
					"user":      "U456",
					"text":      "reply message",
					"thread_ts": "1234.5678",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := watcher.NewSlackWebClient("xoxp-test-token", watcher.WithBaseURL(server.URL))
	s.Require().NoError(err)

	messages, err := client.GetThreadReplies(context.Background(), "C001", "1234.5678")
	s.NoError(err)
	s.Require().Len(messages, 2)
	s.Equal("parent message", messages[0].Text)
	s.Equal("U123", messages[0].Sender)
	s.Equal("reply message", messages[1].Text)
	s.Equal("U456", messages[1].Sender)
}

func (s *SlackWebClientSuite) TestRateLimitReturnsError() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"ok": false, "error": "ratelimited"}`)
	}))
	defer server.Close()

	client, err := watcher.NewSlackWebClient("xoxp-test-token", watcher.WithBaseURL(server.URL))
	s.Require().NoError(err)

	_, err = client.GetUserChannels(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "rate limit")
}

func (s *SlackWebClientSuite) TestInvalidTokenReturnsError() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"ok": false, "error": "invalid_auth"}`)
	}))
	defer server.Close()

	client, err := watcher.NewSlackWebClient("xoxp-bad-token", watcher.WithBaseURL(server.URL))
	s.Require().NoError(err)

	_, err = client.GetUserChannels(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "auth")
}

func (s *SlackWebClientSuite) TestNetworkTimeout() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block longer than the client timeout
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	client, err := watcher.NewSlackWebClient("xoxp-test-token",
		watcher.WithBaseURL(server.URL),
		watcher.WithTimeout(100*time.Millisecond),
	)
	s.Require().NoError(err)

	_, err = client.GetUserChannels(context.Background())
	s.Error(err)
}

func (s *SlackWebClientSuite) TestMalformedJSON() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{this is not valid json`)
	}))
	defer server.Close()

	client, err := watcher.NewSlackWebClient("xoxp-test-token", watcher.WithBaseURL(server.URL))
	s.Require().NoError(err)

	_, err = client.GetUserChannels(context.Background())
	s.Error(err)
}

func (s *SlackWebClientSuite) TestEmptyResponseReturnsEmptySlice() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"ok":       true,
			"channels": []map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := watcher.NewSlackWebClient("xoxp-test-token", watcher.WithBaseURL(server.URL))
	s.Require().NoError(err)

	channels, err := client.GetUserChannels(context.Background())
	s.NoError(err)
	s.NotNil(channels)
	s.Empty(channels)
}
