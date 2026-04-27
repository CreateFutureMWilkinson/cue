package server

import "github.com/CreateFutureMWilkinson/cue/internal/server/handler"

// hubPublisher adapts *Hub to the handler.Publisher interface, allowing
// the WebSocket handler to subscribe/unsubscribe without importing the
// server package (which would create a cycle).
type hubPublisher struct{ hub *Hub }

func newHubPublisher(h *Hub) *hubPublisher { return &hubPublisher{hub: h} }

func (p *hubPublisher) Subscribe(id string) (*handler.Subscription, error) {
	sub, err := p.hub.Subscribe(id)
	if err != nil {
		return nil, err
	}
	return &handler.Subscription{ID: sub.ID, Events: sub.Events}, nil
}

func (p *hubPublisher) Unsubscribe(id string) error { return p.hub.Unsubscribe(id) }
