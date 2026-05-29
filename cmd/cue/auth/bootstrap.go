package auth

import (
	"context"
	"errors"
	"fmt"

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
	token, err := store.Load(ctx)
	if err == nil {
		sdk.SetToken(token)
		return nil
	}
	if !errors.Is(err, ErrNoToken) {
		// Refuse to probe on any other Load error: a transient
		// permission failure must not silently re-pair against a
		// server that has already issued us a token.
		return fmt.Errorf("%w: %w", ErrTokenStoreUnreadable, err)
	}

	// No token on disk: drive the SDK's transparent auto-issue path.
	probeErr := probe(ctx, sdk)
	if probeErr != nil {
		var apiErr *client.APIError
		if errors.As(probeErr, &apiErr) && apiErr.Code == client.ErrCodeUnauthorized {
			return fmt.Errorf("%w: %w", ErrPairingRequired, probeErr)
		}
		if errors.As(probeErr, &apiErr) {
			// Some other API-level error — surface as-is.
			return probeErr
		}
		return fmt.Errorf("%w: %w", ErrServerUnreachable, probeErr)
	}

	newToken := sdk.Token()
	if newToken == "" {
		// Server-bug guard: probe returned 2xx without the SDK's
		// TOKEN_ISSUED auto-retry having installed a token. This
		// would mean the server accepted an unauthenticated request
		// without issuing a token — a contract violation.
		return errors.New("probe succeeded without auto-issue: server accepted request but did not issue a token")
	}

	if saveErr := store.Save(ctx, newToken); saveErr != nil {
		// Include the orphaned token in the error so the user can
		// recover it manually if disk persistence has failed. This
		// is a deliberate trade-off documented on the package.
		return fmt.Errorf("%w: failed to persist auto-issued token %q: %w", ErrTokenStoreUnwritable, newToken, saveErr)
	}
	return nil
}
