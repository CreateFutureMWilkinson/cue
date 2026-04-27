package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
	"github.com/google/uuid"
)

// EventBroadcaster is the minimal interface handlers need to broadcast events.
type EventBroadcaster interface {
	Broadcast(data []byte) error
}

// AuthTokenCreator is the subset of auth token persistence needed by the approve handler.
type AuthTokenCreator interface {
	Create(ctx context.Context, token *repository.AuthToken) error
}

// PairingStorer is the interface for the in-memory pairing store used by auth handlers.
type PairingStorer interface {
	Create(label string) *PairingRequest
	Get(id uuid.UUID) (*PairingRequest, bool)
	Approve(id uuid.UUID, token string) error
	Deny(id uuid.UUID) error
}

// PairingRequest mirrors server.PairingRequest for handler-level use.
type PairingRequest struct {
	ID        uuid.UUID
	Label     string
	Code      string
	ExpiresAt string
	Status    string
	Token     string
}

// InitiatePairingHandler returns a handler for POST /api/v1/auth/pair.
// It creates a new pairing request and broadcasts a pairing_request event.
func InitiatePairingHandler(store PairingStorer, hub EventBroadcaster) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Label string `json:"label"`
		}
		// Ignore decode errors — label defaults to "".
		_ = json.NewDecoder(r.Body).Decode(&body)

		req := store.Create(body.Label)

		// Broadcast pairing_request event.
		event, _ := json.Marshal(map[string]any{
			"event": "pairing_request",
			"data": map[string]any{
				"request_id": req.ID.String(),
				"label":      req.Label,
				"code":       req.Code,
			},
		})
		_ = hub.Broadcast(event)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"request_id": req.ID.String(),
			"code":       req.Code,
		})
	})
}

// PollPairingHandler returns a handler for GET /api/v1/auth/pair/{id}.
// It returns the current status of a pairing request.
func PollPairingHandler(store PairingStorer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		req, ok := store.Get(id)
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		resp := struct {
			Status string `json:"status"`
			Token  string `json:"token,omitempty"`
		}{
			Status: req.Status,
			Token:  req.Token,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// ApprovePairingHandler returns a handler for POST /api/v1/auth/pair/{id}/approve.
// It generates a token, persists it, approves the pairing request, and broadcasts.
func ApprovePairingHandler(store PairingStorer, tokenRepo AuthTokenCreator, hub EventBroadcaster) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		// Generate 32 random bytes, hex-encode to 64-char plaintext token.
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			http.Error(w, "token generation failed", http.StatusInternalServerError)
			return
		}
		plaintext := hex.EncodeToString(tokenBytes)

		// SHA-256 hash for storage.
		hashBytes := sha256.Sum256([]byte(plaintext))
		tokenHash := hex.EncodeToString(hashBytes[:])

		// Persist the hashed token.
		authToken := &repository.AuthToken{
			ID:        uuid.New(),
			Label:     "paired",
			TokenHash: tokenHash,
			CreatedAt: time.Now(),
			LastSeen:  time.Now(),
		}
		if err := tokenRepo.Create(r.Context(), authToken); err != nil {
			http.Error(w, "token persistence failed", http.StatusInternalServerError)
			return
		}

		// Approve the pairing request with the plaintext token.
		if err := store.Approve(id, plaintext); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		// Broadcast pairing_resolved event.
		event, _ := json.Marshal(map[string]any{
			"event": "pairing_resolved",
			"data": map[string]any{
				"request_id": id.String(),
				"status":     "approved",
			},
		})
		_ = hub.Broadcast(event)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
	})
}

// DenyPairingHandler returns a handler for POST /api/v1/auth/pair/{id}/deny.
// It denies the pairing request and broadcasts a pairing_resolved event.
func DenyPairingHandler(store PairingStorer, hub EventBroadcaster) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		if err := store.Deny(id); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		// Broadcast pairing_resolved event.
		event, _ := json.Marshal(map[string]any{
			"event": "pairing_resolved",
			"data": map[string]any{
				"request_id": id.String(),
				"status":     "denied",
			},
		})
		_ = hub.Broadcast(event)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "denied"})
	})
}
