package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
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

// AuthMiddleware returns HTTP middleware that validates Bearer tokens against
// the given AuthTokenLookup repository. When authEnabled is false the middleware
// passes all requests through without checking credentials.
func AuthMiddleware(repo AuthTokenLookup, authEnabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authEnabled {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, "missing Authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
				writeAuthError(w, "malformed Authorization header")
				return
			}

			rawToken := parts[1]
			hash := sha256.Sum256([]byte(rawToken))
			hexHash := hex.EncodeToString(hash[:])

			token, err := repo.LookupByHash(r.Context(), hexHash)
			if err != nil {
				writeAuthError(w, "invalid token")
				return
			}

			if token.Revoked {
				writeAuthError(w, "token revoked")
				return
			}

			// Fire and forget — update last seen timestamp.
			go repo.UpdateLastSeen(r.Context(), token.ID, time.Now())

			next.ServeHTTP(w, r)
		})
	}
}
