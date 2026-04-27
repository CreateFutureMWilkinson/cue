package server

import (
	"context"

	"github.com/CreateFutureMWilkinson/cue/internal/service/orchestrator"
)

// Compile-time verification that HubAlerter implements orchestrator.Alerter.
var _ orchestrator.Alerter = (*HubAlerter)(nil)

// HubAlerter is an orchestrator.Alerter implementation that broadcasts alert
// envelopes to connected WebSocket clients via a Hub, instead of playing local
// audio. This allows the web-based GUI to render its own notification sounds or
// visual cues in response to alert events.
type HubAlerter struct {
	hub *Hub
}

// NewHubAlerter creates a HubAlerter that publishes alerts via the given Hub.
func NewHubAlerter(hub *Hub) *HubAlerter { return &HubAlerter{hub: hub} }

// PlayNotification broadcasts a notification alert to all connected clients.
func (a *HubAlerter) PlayNotification(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.hub.PublishAlert(AlertData{Kind: "notification"})
	return nil
}
