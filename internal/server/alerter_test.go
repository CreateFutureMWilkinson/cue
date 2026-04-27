package server_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/internal/server"
	"github.com/CreateFutureMWilkinson/cue/internal/service/orchestrator"
)

// Compile-time assertion: HubAlerter satisfies orchestrator.Alerter.
var _ orchestrator.Alerter = (*server.HubAlerter)(nil)

type AlerterSuite struct {
	suite.Suite
}

func TestAlerter(t *testing.T) {
	suite.Run(t, new(AlerterSuite))
}

// TestPlayNotificationPublishesAlertEnvelopes verifies that calling
// PlayNotification publishes alert envelopes to the hub and that
// successive calls each append a new envelope.
func (s *AlerterSuite) TestPlayNotificationPublishesAlertEnvelopes() {
	hub := server.NewHub()
	alerter := server.NewHubAlerter(hub)
	ctx := context.Background()

	// First call.
	err := alerter.PlayNotification(ctx)
	s.Require().NoError(err, "first PlayNotification must not error")

	// Second call.
	err = alerter.PlayNotification(ctx)
	s.Require().NoError(err, "second PlayNotification must not error")

	// After two calls, history must contain exactly 2 alert envelopes.
	hist := hub.History(0)
	s.Require().Len(hist.Events, 2, "two PlayNotification calls must produce two envelopes")

	for i, env := range hist.Events {
		s.Equal("alert", env.Type, "envelope %d Type must be 'alert'", i)
		s.Greater(env.Seq, uint64(0), "envelope %d Seq must be > 0", i)

		alertData, ok := env.Data.(server.AlertData)
		s.Require().Truef(ok, "envelope %d Data must be server.AlertData, got %T", i, env.Data)
		s.NotEmpty(alertData.Kind, "envelope %d AlertData.Kind must be non-empty", i)
	}

	// Sequences must be monotonically increasing.
	s.Less(hist.Events[0].Seq, hist.Events[1].Seq, "envelope sequences must be monotonically increasing")
}
