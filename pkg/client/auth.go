package client

import (
	"context"

	"github.com/google/uuid"
)

// PairSession is the response from InitiatePairing.
//
// The server returns this on POST /api/v1/auth/pair with 202 Accepted.
// RequestID is the identifier used for polling and approve/deny; Code is
// the short human-readable pairing code the user reads out-of-band to
// authorize the new client.
type PairSession struct {
	RequestID uuid.UUID `json:"request_id"`
	Code      string    `json:"code"`
}

// PairResult is the response from PollPairing.
//
// Status is one of "pending", "approved", "denied", or "expired". Token
// is populated (non-empty) only when Status == "approved"; the server
// omits it otherwise via `omitempty`.
type PairResult struct {
	Status string `json:"status"`
	Token  string `json:"token,omitempty"`
}

// TokenInfo describes a stored auth token returned by ListTokens.
//
// CreatedAt and LastSeen are RFC3339-formatted strings as emitted by
// the server. Revoked indicates whether the token has been soft-deleted.
type TokenInfo struct {
	ID        uuid.UUID `json:"id"`
	Label     string    `json:"label"`
	CreatedAt string    `json:"created_at"`
	LastSeen  string    `json:"last_seen"`
	Revoked   bool      `json:"revoked"`
}

// AuthClient wraps the /api/v1/auth/* endpoints.
//
// It covers the TOFU pairing flow (Initiate → Poll → Approve/Deny) and
// token management (list, relabel, revoke). There is no /auth/logout
// endpoint — Feature 108 decided clients should simply discard their
// local token or call RevokeToken on their own ID.
type AuthClient interface {
	InitiatePairing(ctx context.Context, label string) (*PairSession, error)
	PollPairing(ctx context.Context, requestID uuid.UUID) (*PairResult, error)
	ApprovePairing(ctx context.Context, requestID uuid.UUID) error
	DenyPairing(ctx context.Context, requestID uuid.UUID) error
	ListTokens(ctx context.Context) ([]TokenInfo, error)
	UpdateTokenLabel(ctx context.Context, id uuid.UUID, label string) error
	RevokeToken(ctx context.Context, id uuid.UUID) error
}

// authAdapter is the concrete AuthClient backed by an *APIClient.
type authAdapter struct {
	client *APIClient
}

// NewAuthClient returns an AuthClient backed by the given APIClient.
func NewAuthClient(c *APIClient) AuthClient {
	return &authAdapter{client: c}
}

// InitiatePairing issues POST /api/v1/auth/pair with the given label and
// returns the server-issued PairSession on 202 Accepted.
func (a *authAdapter) InitiatePairing(ctx context.Context, label string) (*PairSession, error) {
	return nil, ErrNotImplemented
}

// PollPairing issues GET /api/v1/auth/pair/{id} and returns the current
// PairResult. When Status == "approved", Token is populated.
func (a *authAdapter) PollPairing(ctx context.Context, requestID uuid.UUID) (*PairResult, error) {
	return nil, ErrNotImplemented
}

// ApprovePairing issues POST /api/v1/auth/pair/{id}/approve.
func (a *authAdapter) ApprovePairing(ctx context.Context, requestID uuid.UUID) error {
	return ErrNotImplemented
}

// DenyPairing issues POST /api/v1/auth/pair/{id}/deny.
func (a *authAdapter) DenyPairing(ctx context.Context, requestID uuid.UUID) error {
	return ErrNotImplemented
}

// ListTokens issues GET /api/v1/auth/tokens and returns all known tokens.
func (a *authAdapter) ListTokens(ctx context.Context) ([]TokenInfo, error) {
	return nil, ErrNotImplemented
}

// UpdateTokenLabel issues PUT /api/v1/auth/tokens/{id} with the new label.
func (a *authAdapter) UpdateTokenLabel(ctx context.Context, id uuid.UUID, label string) error {
	return ErrNotImplemented
}

// RevokeToken issues DELETE /api/v1/auth/tokens/{id}.
func (a *authAdapter) RevokeToken(ctx context.Context, id uuid.UUID) error {
	return ErrNotImplemented
}
