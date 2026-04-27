package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/server"
	"github.com/CreateFutureMWilkinson/cue/internal/server/handler"
	"github.com/coder/websocket"
	"github.com/stretchr/testify/suite"
)

// hubAdapter wraps *server.Hub to satisfy handler.Publisher, bridging
// the *server.Subscriber → *handler.Subscription type difference.
type hubAdapter struct {
	hub *server.Hub
}

func (a *hubAdapter) Subscribe(id string) (*handler.Subscription, error) {
	sub, err := a.hub.Subscribe(id)
	if err != nil {
		return nil, err
	}
	return &handler.Subscription{
		ID:     sub.ID,
		Events: sub.Events,
	}, nil
}

func (a *hubAdapter) Unsubscribe(id string) error {
	return a.hub.Unsubscribe(id)
}

// ---------- suite ----------

type WebSocketHandlerSuite struct {
	suite.Suite
}

func TestWebSocketHandler(t *testing.T) {
	suite.Run(t, new(WebSocketHandlerSuite))
}

// ---------- tests ----------

func (s *WebSocketHandlerSuite) TestWebSocketHandler_HappyPath() {
	hub := server.NewHub()
	pub := &hubAdapter{hub: hub}

	srv := httptest.NewServer(handler.WebSocketHandler(pub))
	defer srv.Close()

	// Convert http:// URL to ws://
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	s.Require().NoError(err, "WebSocket dial should succeed")
	defer conn.CloseNow() //nolint:errcheck

	// Publish an activity event through the hub.
	hub.Publish(server.ActivityData{
		Source:  "slack",
		Message: "hello",
		IsError: false,
	})

	// Read one message with a 2s timeout.
	readCtx, readCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer readCancel()

	msgType, payload, err := conn.Read(readCtx)
	s.Require().NoError(err, "should read a message from the WebSocket")
	s.Equal(websocket.MessageText, msgType, "message should be text")

	// Unmarshal and verify envelope fields.
	var env struct {
		Seq  uint64 `json:"seq"`
		Type string `json:"type"`
		Data struct {
			Source string `json:"source"`
		} `json:"data"`
	}
	s.Require().NoError(json.Unmarshal(payload, &env))
	s.Equal("activity", env.Type)
	s.Equal("slack", env.Data.Source)
	s.Greater(env.Seq, uint64(0), "seq should be positive")

	// Close cleanly.
	err = conn.Close(websocket.StatusNormalClosure, "done")
	s.Require().NoError(err, "clean close should succeed")

	// Allow short time for server-side cleanup.
	s.Eventually(func() bool {
		return hub.SubscriberCount() == 0
	}, 500*time.Millisecond, 10*time.Millisecond, "hub should have 0 subscribers after client disconnect")
}

func (s *WebSocketHandlerSuite) TestWebSocketHandler_OriginPolicy() {
	hub := server.NewHub()
	pub := &hubAdapter{hub: hub}

	srv := httptest.NewServer(handler.WebSocketHandler(pub))
	defer srv.Close()

	cases := []struct {
		name       string
		origin     string // empty string means no Origin header
		wantStatus int
	}{
		{
			name:       "missing origin is accepted",
			origin:     "",
			wantStatus: http.StatusSwitchingProtocols,
		},
		{
			name:       "same origin is accepted",
			origin:     srv.URL, // http://<host>:<port> matches the server
			wantStatus: http.StatusSwitchingProtocols,
		},
		{
			name:       "cross origin is rejected",
			origin:     "http://evil.example.com",
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			resp, err := upgradeRequest(srv.URL, tc.origin)
			s.Require().NoError(err, "HTTP request should not fail")
			resp.Body.Close()
			s.Equal(tc.wantStatus, resp.StatusCode)
		})
	}
}

func (s *WebSocketHandlerSuite) TestWebSocketHandler_ConnectionCap() {
	hub := server.NewHub()
	pub := &hubAdapter{hub: hub}

	srv := httptest.NewServer(handler.WebSocketHandler(pub))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	// Open 16 connections — the maximum allowed.
	conns := make([]*websocket.Conn, 0, handler.MaxConnections)
	s.T().Cleanup(func() {
		for _, c := range conns {
			c.CloseNow() //nolint:errcheck
		}
	})

	for i := 0; i < handler.MaxConnections; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c, _, err := websocket.Dial(ctx, wsURL, nil)
		s.Require().NoError(err, "connection %d should succeed", i+1)
		conns = append(conns, c)
	}

	// The 17th connection must be rejected with 503.
	resp, err := upgradeRequest(srv.URL, "")
	s.Require().NoError(err, "HTTP request should not fail")
	defer resp.Body.Close()

	s.Equal(http.StatusServiceUnavailable, resp.StatusCode,
		"17th connection should be rejected with 503")

	s.Equal("5", resp.Header.Get("Retry-After"),
		"response should include Retry-After: 5 header")

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err, "should read response body")

	var errBody struct {
		Error string `json:"error"`
	}
	s.Require().NoError(json.Unmarshal(body, &errBody),
		"response body should be valid JSON")
	s.Equal("too many connections", errBody.Error)
}

func (s *WebSocketHandlerSuite) TestWebSocketHandler_HeartbeatClosesOnTimeout() {
	hub := server.NewHub()
	pub := &hubAdapter{hub: hub}

	// Use very short heartbeat intervals so the test runs fast.
	// The server will ping every 50ms and expect a pong within 100ms.
	h := handler.WebSocketHandlerWithHeartbeat(pub, 50*time.Millisecond, 100*time.Millisecond)
	srv := httptest.NewServer(h)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Dial the WebSocket but do NOT start a read loop. Without reading,
	// the coder/websocket client cannot auto-respond to pings, so server-side
	// conn.Ping calls will time out.
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	s.Require().NoError(err, "WebSocket dial should succeed")
	defer conn.CloseNow() //nolint:errcheck

	// Verify the subscriber was registered.
	s.Eventually(func() bool {
		return hub.SubscriberCount() == 1
	}, 500*time.Millisecond, 10*time.Millisecond, "hub should have 1 subscriber after dial")

	// Wait for the server to detect the heartbeat timeout and unsubscribe.
	// Total budget: interval (50ms) + timeout (100ms) + margin ≈ 200ms,
	// but we allow up to 1s to avoid flakiness.
	s.Eventually(func() bool {
		return hub.SubscriberCount() == 0
	}, 1*time.Second, 10*time.Millisecond,
		"hub should have 0 subscribers after heartbeat timeout")

	// The connection should now be closed on the server side. Any read
	// attempt from the client should return an error.
	readCtx, readCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer readCancel()
	_, _, readErr := conn.Read(readCtx)
	s.Error(readErr, "reading from a server-closed connection should fail")
}

// upgradeRequest sends a raw HTTP request with WebSocket upgrade headers.
// If origin is non-empty it is included as the Origin header.
func upgradeRequest(url, origin string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	return http.DefaultClient.Do(req)
}
