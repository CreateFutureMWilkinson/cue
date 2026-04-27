package server_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/server"
	"github.com/CreateFutureMWilkinson/cue/internal/service/orchestrator"
)

// CompositionIntegrationSuite contains end-to-end tests that verify
// WebSocket clients receive activity envelopes broadcast through
// the composition's event pipeline.
type CompositionIntegrationSuite struct {
	suite.Suite
}

func TestCompositionIntegration(t *testing.T) {
	suite.Run(t, new(CompositionIntegrationSuite))
}

// TestCompositionEndToEndActivityBroadcast verifies that a WebSocket
// client connected via the composition's HTTP server receives activity
// envelopes when events are sent into the composition's EventCh.
//
// This is the capstone acceptance test for Feature 099A: it proves
// that the Composition's Hub (used by the publisher goroutine and
// HubAlerter) is the SAME hub backing the Server's WebSocket handlers.
//
// Expected RED failure: the composition currently constructs the HTTP
// server with its own internal Hub, so the WS subscriber is registered
// on a different Hub than the one the publisher goroutine writes to.
// The conn.Read call times out with context.DeadlineExceeded.
func (s *CompositionIntegrationSuite) TestCompositionEndToEndActivityBroadcast() {
	tmpDir := s.T().TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	cfg := minimalConfig(dbPath)

	ctx := context.Background()

	// 1. Build composition.
	comp, err := server.NewComposition(ctx, cfg)
	s.Require().NoError(err, "NewComposition should succeed")
	s.Require().NotNil(comp, "Composition must be non-nil")
	defer func() {
		_ = comp.Shutdown(ctx)
	}()

	// 2. Expose the composition's HTTP handler via httptest.
	s.Require().NotNil(comp.HTTP, "Composition.HTTP must be non-nil")
	ts := httptest.NewServer(comp.HTTP.Handler())
	defer ts.Close()

	// 3. Dial the WebSocket endpoint.
	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1) + "/api/v1/websocket/events"
	dialCtx, dialCancel := context.WithTimeout(ctx, 2*time.Second)
	defer dialCancel()

	conn, _, err := websocket.Dial(dialCtx, wsURL, nil)
	s.Require().NoError(err, "WebSocket dial should succeed")
	defer conn.CloseNow() //nolint:errcheck

	// 4. Wait for the WS subscriber to register on the server's hub.
	// NOTE: in the RED state the subscriber registers on the *server's*
	// internal hub, not comp.Hub — but SubscriberCount on the server's
	// hub still returns 1 so this wait succeeds. The disconnect only
	// manifests when we try to receive an event published through comp.Hub.
	s.Require().Eventually(func() bool {
		return comp.HTTP.Hub().SubscriberCount() > 0
	}, 500*time.Millisecond, 10*time.Millisecond,
		"WS subscriber should register within 500ms")

	// 5. Inject an activity event into the composition's event channel.
	comp.EventCh <- orchestrator.ActivityEvent{
		Source:  "e2e-test",
		Message: "hello world",
		IsError: false,
	}

	// 6. Read one message from the WebSocket (1s timeout).
	readCtx, readCancel := context.WithTimeout(ctx, 1*time.Second)
	defer readCancel()

	_, data, err := conn.Read(readCtx)
	s.Require().NoError(err, "should read an activity envelope from the WebSocket")

	// 7. Unmarshal and assert envelope fields.
	var env server.ActivityEnvelope
	s.Require().NoError(json.Unmarshal(data, &env), "envelope should be valid JSON")

	s.Equal("activity", env.Type, "envelope type must be 'activity'")
	s.GreaterOrEqual(env.Seq, uint64(1), "seq must be >= 1")

	// Data is deserialized as map[string]any from JSON.
	payload, ok := env.Data.(map[string]any)
	s.Require().True(ok, "envelope Data should be map[string]any, got %T", env.Data)
	s.Equal("e2e-test", payload["source"], "source field must match")
	s.Equal("hello world", payload["message"], "message field must match")
	s.Equal(false, payload["is_error"], "is_error field must be false")
}
