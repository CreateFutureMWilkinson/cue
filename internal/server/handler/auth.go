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

// AuthTokenManager is the interface for token CRUD operations.
type AuthTokenManager interface {
	AuthTokenCreator
	List(ctx context.Context) ([]repository.AuthToken, error)
	UpdateLabel(ctx context.Context, id uuid.UUID, label string) error
	Revoke(ctx context.Context, id uuid.UUID) error
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
//
// @Summary      Begin a pairing handshake
// @Description  Creates a pairing request, returning a request_id and a short
// @Description  numeric code the user types into the desktop UI to approve.
// @Description  Also broadcasts a "pairing_request" event over the WebSocket
// @Description  channel for already-paired clients to display.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      object  false  "Optional {label: string} for the new device"
// @Success      202      {object}  map[string]any
// @Router       /api/v1/auth/pair [post]
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
//
// @Summary      Poll pairing request status
// @Description  Returns the current status of a pairing request. Once the
// @Description  request is approved, the response includes the bearer token
// @Description  the client must use for subsequent authenticated requests.
// @Tags         auth
// @Produce      json
// @Param        id   path      string  true  "Pairing request UUID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /api/v1/auth/pair/{id} [get]
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
//
// @Summary      Approve a pending pairing request
// @Description  Generates a fresh bearer token, persists its hash, attaches
// @Description  the plaintext to the pairing request (so the polling client
// @Description  can pick it up), and broadcasts a "pairing_resolved" event.
// @Tags         auth
// @Produce      json
// @Param        id   path      string  true  "Pairing request UUID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      409  {object}  map[string]string  "request already resolved"
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/auth/pair/{id}/approve [post]
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
//
// @Summary      Deny a pending pairing request
// @Tags         auth
// @Produce      json
// @Param        id   path      string  true  "Pairing request UUID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      409  {object}  map[string]string  "request already resolved"
// @Router       /api/v1/auth/pair/{id}/deny [post]
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

// ListTokensHandler returns a handler for GET /api/v1/auth/tokens.
//
// @Summary      List bearer tokens
// @Description  Returns metadata for all paired tokens (id, label, timestamps,
// @Description  revoked flag). Plaintext tokens are never returned by this API.
// @Tags         auth
// @Produce      json
// @Success      200  {array}   object
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/auth/tokens [get]
func ListTokensHandler(repo AuthTokenManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokens, err := repo.List(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list tokens")
			return
		}

		type tokenResp struct {
			ID        string `json:"id"`
			Label     string `json:"label"`
			CreatedAt string `json:"created_at"`
			LastSeen  string `json:"last_seen"`
			Revoked   bool   `json:"revoked"`
		}
		resp := make([]tokenResp, len(tokens))
		for i, t := range tokens {
			resp[i] = tokenResp{
				ID:        t.ID.String(),
				Label:     t.Label,
				CreatedAt: t.CreatedAt.Format(time.RFC3339),
				LastSeen:  t.LastSeen.Format(time.RFC3339),
				Revoked:   t.Revoked,
			}
		}
		writeJSON(w, http.StatusOK, resp)
	})
}

// UpdateTokenLabelHandler returns a handler for PUT /api/v1/auth/tokens/{id}.
//
// @Summary      Update bearer token label
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "Token UUID"
// @Param        request  body      object  true  "{label: string}"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/auth/tokens/{id} [put]
func UpdateTokenLabelHandler(repo AuthTokenManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDFromPath(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var body struct {
			Label string `json:"label"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if err := repo.UpdateLabel(r.Context(), id, body.Label); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to update label")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	})
}

// RevokeTokenHandler returns a handler for DELETE /api/v1/auth/tokens/{id}.
//
// @Summary      Revoke a bearer token
// @Description  Marks the token as revoked. Future requests using the token
// @Description  will fail authentication.
// @Tags         auth
// @Produce      json
// @Param        id   path      string  true  "Token UUID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/auth/tokens/{id} [delete]
func RevokeTokenHandler(repo AuthTokenManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDFromPath(r)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		if err := repo.Revoke(r.Context(), id); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to revoke token")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
	})
}
