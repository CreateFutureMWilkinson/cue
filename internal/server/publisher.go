package server

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/server/handler"
)

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

// Broadcast delegates to the wrapped Hub's Broadcast method, implementing
// handler.EventBroadcaster for pairing event delivery.
func (p *hubPublisher) Broadcast(data []byte) error { return p.hub.Broadcast(data) }

// pairingStoreAdapter adapts *PairingStore to handler.PairingStorer,
// converting between server.PairingRequest and handler.PairingRequest.
type pairingStoreAdapter struct{ store *PairingStore }

func newPairingStoreAdapter(s *PairingStore) *pairingStoreAdapter {
	return &pairingStoreAdapter{store: s}
}

func (a *pairingStoreAdapter) Create(label string) *handler.PairingRequest {
	r := a.store.Create(label)
	return convertPairingRequest(r)
}

func (a *pairingStoreAdapter) Get(id uuid.UUID) (*handler.PairingRequest, bool) {
	r, ok := a.store.Get(id)
	if !ok {
		return nil, false
	}
	return convertPairingRequest(r), true
}

func (a *pairingStoreAdapter) Approve(id uuid.UUID, token string) error {
	return a.store.Approve(id, token)
}

func (a *pairingStoreAdapter) Deny(id uuid.UUID) error {
	return a.store.Deny(id)
}

func convertPairingRequest(r *PairingRequest) *handler.PairingRequest {
	return &handler.PairingRequest{
		ID:        r.ID,
		Label:     r.Label,
		Code:      r.Code,
		ExpiresAt: r.ExpiresAt.Format(time.RFC3339),
		Status:    r.Status,
		Token:     r.Token,
	}
}

// HistoryJSON implements handler.HistoryProvider by delegating to the
// wrapped Hub's History method and marshaling the result to JSON.
func (p *hubPublisher) HistoryJSON(sinceSeq uint64) ([]byte, error) {
	resp := p.hub.History(sinceSeq)
	return json.Marshal(resp)
}
