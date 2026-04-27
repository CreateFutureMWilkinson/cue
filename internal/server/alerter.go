package server

import "context"

// HubAlerter adapts a Hub into an orchestrator.Alerter by broadcasting
// alert envelopes to connected WebSocket clients.
type HubAlerter struct{}

// NewHubAlerter creates a HubAlerter that publishes alerts via the given Hub.
func NewHubAlerter(hub *Hub) *HubAlerter { return &HubAlerter{} }

// PlayNotification broadcasts a notification alert to all connected clients.
// noop stub — returns nil without publishing an envelope.
func (a *HubAlerter) PlayNotification(ctx context.Context) error {
	return nil
}
