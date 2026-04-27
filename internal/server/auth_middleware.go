package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"
)

// AuthTokenLookup is the subset of repository.AuthTokenRepository needed by the auth middleware.
type AuthTokenLookup interface {
	LookupByHash(ctx context.Context, hash string) (*repository.AuthToken, error)
	CountActive(ctx context.Context) (int, error)
	Create(ctx context.Context, token *repository.AuthToken) error
	UpdateLastSeen(ctx context.Context, id uuid.UUID, t time.Time) error
}

// authErrorResponse is the JSON structure for authentication errors.
type authErrorResponse struct {
	Error authErrorDetail `json:"error"`
}

type authErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(authErrorResponse{
		Error: authErrorDetail{
			Code:    "UNAUTHORIZED",
			Message: message,
		},
	})
}

// isExemptPath returns true for routes that bypass authentication.
func isExemptPath(path string) bool {
	switch path {
	case "/health", "/health/ready", "/api/v1/health", "/api/v1/health/ready":
		return true
	}
	if strings.HasPrefix(path, "/api/v1/auth/pair") {
		return true
	}
	return false
}

// generateToken creates a new random token and returns plaintext and hash.
func generateToken() (plaintext, hash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	plaintext = hex.EncodeToString(raw)
	hash = hashToken(plaintext)
	return plaintext, hash, nil
}

// tryAutoIssueFirstClient checks whether zero active tokens exist and, if so,
// generates and persists a new token, writing the TOKEN_ISSUED response.
// Returns true if the auto-issue flow was triggered (response already written).
func tryAutoIssueFirstClient(w http.ResponseWriter, ctx context.Context, repo AuthTokenLookup) bool {
	count, err := repo.CountActive(ctx)
	if err != nil || count > 0 {
		return false
	}

	plaintext, hexHash, err := generateToken()
	if err != nil {
		writeAuthError(w, "internal error generating token")
		return true
	}

	now := time.Now()
	token := &repository.AuthToken{
		ID:        uuid.New(),
		TokenHash: hexHash,
		Label:     "",
		CreatedAt: now,
		LastSeen:  now,
	}
	if err := repo.Create(ctx, token); err != nil {
		writeAuthError(w, "internal error storing token")
		return true
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    "TOKEN_ISSUED",
			"message": "First client — token auto-issued",
		},
		"token": plaintext,
	})
	return true
}

// hashToken computes the SHA-256 hash of a token string.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// extractBearerToken parses the Authorization header and returns the Bearer token.
// Returns empty string if the header is malformed or not a Bearer token.
func extractBearerToken(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return ""
	}
	return parts[1]
}

// handleAuthenticationFailure attempts auto-issue for first client, otherwise writes error.
func handleAuthenticationFailure(w http.ResponseWriter, ctx context.Context, repo AuthTokenLookup, message string) {
	if tryAutoIssueFirstClient(w, ctx, repo) {
		return
	}
	writeAuthError(w, message)
}

// AuthMiddleware returns HTTP middleware that validates Bearer tokens against
// the given AuthTokenLookup repository. When authEnabled is false the middleware
// passes all requests through without checking credentials.
func AuthMiddleware(repo AuthTokenLookup, authEnabled bool) func(http.Handler) http.Handler {
	// Last-seen throttle state: tokenHash → last time we fired UpdateLastSeen.
	var lastSeenMu sync.Mutex
	lastSeenMap := make(map[string]time.Time)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authEnabled {
				next.ServeHTTP(w, r)
				return
			}

			if isExemptPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				handleAuthenticationFailure(w, r.Context(), repo, "missing Authorization header")
				return
			}

			rawToken := extractBearerToken(authHeader)
			if rawToken == "" {
				writeAuthError(w, "malformed Authorization header")
				return
			}

			hexHash := hashToken(rawToken)
			token, err := repo.LookupByHash(r.Context(), hexHash)
			if err != nil {
				handleAuthenticationFailure(w, r.Context(), repo, "invalid token")
				return
			}

			if token.Revoked {
				writeAuthError(w, "token revoked")
				return
			}

			// Throttled last-seen update: at most once per minute per token.
			lastSeenMu.Lock()
			lastUpdate, exists := lastSeenMap[hexHash]
			shouldUpdate := !exists || time.Since(lastUpdate) >= time.Minute
			if shouldUpdate {
				lastSeenMap[hexHash] = time.Now()
			}
			lastSeenMu.Unlock()

			if shouldUpdate {
				go repo.UpdateLastSeen(context.Background(), token.ID, time.Now())
			}

			next.ServeHTTP(w, r)
		})
	}
}
