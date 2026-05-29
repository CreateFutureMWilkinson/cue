// Package auth implements the client-side TOFU bootstrap library that
// loads or auto-issues the bearer token used by the SDK to talk to the
// cue-server.
//
// The package owns three concerns:
//
//  1. On-disk persistence of the plaintext bearer token at
//     ~/.cue/client-token (mode 0600, atomic save).
//  2. The Bootstrap flow that loads the token if present, or triggers
//     the SDK's transparent first-client TOKEN_ISSUED auto-issue and
//     persists the issued token.
//  3. A small error taxonomy so callers can render meaningful UX.
//
// Storage trade-off: ErrTokenStoreUnwritable deliberately includes the
// orphaned plaintext token in its message so a user can recover the
// token manually if disk persistence fails after the server already
// issued it. Losing a freshly-issued token would be worse than the
// small leak risk in a local error dialog/log.
package auth

import (
	"errors"
)

// ErrNoToken is returned by TokenStore.Load when no token file is
// present on disk. This is the only Load error that triggers
// Bootstrap's auto-issue probe path; any other error short-circuits
// to ErrTokenStoreUnreadable.
var ErrNoToken = errors.New("no client token on disk")

// Bootstrap error sentinels. Each wraps the underlying cause via
// fmt.Errorf("...: %w", err) so callers can use errors.Is to branch
// and errors.Unwrap (or %+v formatting) to surface the root cause.
var (
	// ErrPairingRequired indicates the server has tokens but this
	// client is not paired. The server returned a plain 401 with no
	// TOKEN_ISSUED body.
	ErrPairingRequired = errors.New("pairing required")

	// ErrServerUnreachable indicates a transport/network failure
	// during the auto-issue probe. The wrapped error is the
	// underlying transport error.
	ErrServerUnreachable = errors.New("server unreachable")

	// ErrTokenStoreUnreadable indicates a non-ErrNoToken failure
	// reading the on-disk token file (permissions, IO error). The
	// caller should NOT fall back to probing — a transient permission
	// error must not silently re-pair against a server that has
	// already issued us a token.
	ErrTokenStoreUnreadable = errors.New("token store unreadable")

	// ErrTokenStoreUnwritable indicates a failure persisting an
	// auto-issued token. The error message includes the orphaned
	// token so the user can recover it manually; this is a
	// deliberate trade-off documented on the package.
	ErrTokenStoreUnwritable = errors.New("token store unwritable")
)
