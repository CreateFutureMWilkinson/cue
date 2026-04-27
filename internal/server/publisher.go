package server

import "github.com/CreateFutureMWilkinson/cue/internal/server/handler"

// hubPublisher is an adapter that implements handler.Publisher by wrapping
// *Hub, allowing WebSocket handlers to subscribe/unsubscribe without
// importing the server package (which would create an import cycle).
// It translates between the server package's Subscriber type and the
// handler package's Subscription type.
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
