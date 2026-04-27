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

// Helper functions for common handler patterns

// parseIDFromPath extracts and parses a UUID from the "id" path value.
func parseIDFromPath(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(r.PathValue("id"))
}

// broadcastPairingEvent broadcasts a pairing-related event, ignoring errors.
func broadcastPairingEvent(hub EventBroadcaster, eventType string, requestID uuid.UUID, status string) {
	event, _ := json.Marshal(map[string]any{
		"event": eventType,
		"data": map[string]any{
			"request_id": requestID.String(),
			"status":     status,
		},
	})
	_ = hub.Broadcast(event)
}

// generateBearerToken creates a 64-character hex-encoded random token.
func generateBearerToken() (plaintext, hash string, err error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", err
	}
	plaintext = hex.EncodeToString(tokenBytes)

	hashBytes := sha256.Sum256([]byte(plaintext))
	hash = hex.EncodeToString(hashBytes[:])

	return plaintext, hash, nil
}

// persistAuthToken creates and stores an auth token with the given hash.
func persistAuthToken(ctx context.Context, tokenRepo AuthTokenCreator, tokenHash string) error {
	authToken := &repository.AuthToken{
		ID:        uuid.New(),
		Label:     "paired",
		TokenHash: tokenHash,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
	}
	return tokenRepo.Create(ctx, authToken)
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

		writeJSON(w, http.StatusAccepted, map[string]any{
			"request_id": req.ID.String(),
			"code":       req.Code,
		})
	})
}

// PollPairingHandler returns a handler for GET /api/v1/auth/pair/{id}.
// It returns the current status of a pairing request.
func PollPairingHandler(store PairingStorer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDFromPath(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		req, ok := store.Get(id)
		if !ok {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}

		resp := struct {
			Status string `json:"status"`
			Token  string `json:"token,omitempty"`
		}{
			Status: req.Status,
			Token:  req.Token,
		}

		writeJSON(w, http.StatusOK, resp)
	})
}

// ApprovePairingHandler returns a handler for POST /api/v1/auth/pair/{id}/approve.
// It generates a token, persists it, approves the pairing request, and broadcasts.
func ApprovePairingHandler(store PairingStorer, tokenRepo AuthTokenCreator, hub EventBroadcaster) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDFromPath(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		plaintext, tokenHash, err := generateBearerToken()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "token generation failed")
			return
		}

		if err := persistAuthToken(r.Context(), tokenRepo, tokenHash); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "token persistence failed")
			return
		}

		if err := store.Approve(id, plaintext); err != nil {
			writeJSONError(w, http.StatusConflict, err.Error())
			return
		}

		broadcastPairingEvent(hub, "pairing_resolved", id, "approved")
		writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
	})
}

// DenyPairingHandler returns a handler for POST /api/v1/auth/pair/{id}/deny.
// It denies the pairing request and broadcasts a pairing_resolved event.
func DenyPairingHandler(store PairingStorer, hub EventBroadcaster) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDFromPath(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		if err := store.Deny(id); err != nil {
			writeJSONError(w, http.StatusConflict, err.Error())
			return
		}

		broadcastPairingEvent(hub, "pairing_resolved", id, "denied")
		writeJSON(w, http.StatusOK, map[string]string{"status": "denied"})
	})
}
