package auth

import (
	"context"

	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// Probe is a function that performs any authenticated request through
// the SDK. Its purpose is to trigger the SDK's transparent first-client
// TOKEN_ISSUED auto-retry on a fresh server. It returns nil on success
// and *client.APIError on HTTP-level errors. Transport/network errors
// surface as plain errors (not *APIError) and are routed by Bootstrap
// to ErrServerUnreachable.
type Probe func(ctx context.Context, sdk *client.APIClient) error

// DefaultProbe issues GET /api/v1/auth/tokens via the SDK's AuthClient.
// It is cheap, idempotent, and only succeeds when the client is
// authenticated — making it a clean trigger for the SDK's TOKEN_ISSUED
// auto-retry on a fresh server.
var DefaultProbe Probe = func(ctx context.Context, sdk *client.APIClient) error {
	_, err := client.NewAuthClient(sdk).ListTokens(ctx)
	return err
}

// Bootstrap loads the on-disk token into the SDK if present. If no
// token is on disk, it calls probe to trigger the SDK's transparent
// first-client TOKEN_ISSUED auto-issue, then persists whatever token
// the SDK ends up holding.
//
// On return:
//   - nil error: sdk.Token() is non-empty and the token is on disk.
//   - ErrPairingRequired: server has tokens but this client isn't paired.
//   - ErrServerUnreachable: network error during probe.
//   - ErrTokenStoreUnreadable: disk error reading the token file.
//   - ErrTokenStoreUnwritable: disk error writing the auto-issued token
//     (the wrapped message includes the orphaned token for recovery).
func Bootstrap(ctx context.Context, store TokenStore, sdk *client.APIClient, probe Probe) error {
	return ErrNotImplemented
}
