package auth_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/cmd/cue/auth"
	"github.com/CreateFutureMWilkinson/cue/pkg/client"
)

// BootstrapSuite covers the Bootstrap entry point.
type BootstrapSuite struct {
	suite.Suite
}

func TestBootstrap(t *testing.T) {
	suite.Run(t, new(BootstrapSuite))
}

// fakeStore is an in-memory TokenStore for tests. Each method's behavior
// is overridable per-test via the *Override fields.
type fakeStore struct {
	loadToken string
	loadErr   error
	saveErr   error
	deleteErr error

	saveCalls []string
	loadCalls int
}

func (f *fakeStore) Load(ctx context.Context) (string, error) {
	f.loadCalls++
	return f.loadToken, f.loadErr
}

func (f *fakeStore) Save(ctx context.Context, token string) error {
	f.saveCalls = append(f.saveCalls, token)
	return f.saveErr
}

func (f *fakeStore) Delete(ctx context.Context) error {
	return f.deleteErr
}

// B1: Bootstrap returns nil and calls sdk.SetToken when the store has a
// token. The probe must NOT be called.
func (s *BootstrapSuite) TestBootstrapShortCircuitsOnExistingToken() {
	store := &fakeStore{loadToken: "T1", loadErr: nil}
	sdk := client.New("http://unused")
	probe := func(ctx context.Context, _ *client.APIClient) error {
		s.FailNow("probe must not be called when store has a token")
		return nil
	}

	err := auth.Bootstrap(context.Background(), store, sdk, probe)
	s.Require().NoError(err)
	s.Equal("T1", sdk.Token())
	s.Empty(store.saveCalls, "Save must not be called for existing-token path")
}

// B2: Bootstrap probes on ErrNoToken and persists sdk.Token() after
// the probe succeeds (simulating SDK's TOKEN_ISSUED auto-retry).
func (s *BootstrapSuite) TestBootstrapProbesAndPersists() {
	store := &fakeStore{loadErr: auth.ErrNoToken}
	sdk := client.New("http://unused")
	probe := func(ctx context.Context, c *client.APIClient) error {
		c.SetToken("T2")
		return nil
	}

	err := auth.Bootstrap(context.Background(), store, sdk, probe)
	s.Require().NoError(err)
	s.Equal("T2", sdk.Token())
	s.Equal([]string{"T2"}, store.saveCalls)
}

// B3: Bootstrap returns ErrPairingRequired when probe returns a plain
// 401 *client.APIError (no TOKEN_ISSUED).
func (s *BootstrapSuite) TestBootstrapReturnsPairingRequiredOn401() {
	store := &fakeStore{loadErr: auth.ErrNoToken}
	sdk := client.New("http://unused")
	probe := func(ctx context.Context, _ *client.APIClient) error {
		return &client.APIError{StatusCode: 401, Code: client.ErrCodeUnauthorized, Message: "missing or invalid bearer token"}
	}

	err := auth.Bootstrap(context.Background(), store, sdk, probe)
	s.Require().Error(err)
	s.True(errors.Is(err, auth.ErrPairingRequired), "expected ErrPairingRequired, got %v", err)
}

// B4: Bootstrap returns ErrServerUnreachable on a transport error
// (anything that is NOT a *client.APIError).
func (s *BootstrapSuite) TestBootstrapReturnsServerUnreachableOnTransportError() {
	store := &fakeStore{loadErr: auth.ErrNoToken}
	sdk := client.New("http://unused")
	transportErr := &net.OpError{Op: "dial", Err: errors.New("connection refused")}
	probe := func(ctx context.Context, _ *client.APIClient) error {
		return transportErr
	}

	err := auth.Bootstrap(context.Background(), store, sdk, probe)
	s.Require().Error(err)
	s.True(errors.Is(err, auth.ErrServerUnreachable), "expected ErrServerUnreachable, got %v", err)
	s.True(errors.Is(err, transportErr), "wrapped transport error must remain reachable")
}

// B5: Bootstrap returns ErrTokenStoreUnreadable on a non-ErrNoToken
// Load error AND must not call probe (safety: don't re-pair against a
// server that already issued us a token).
func (s *BootstrapSuite) TestBootstrapReturnsUnreadableAndSkipsProbe() {
	ioErr := errors.New("disk on fire")
	store := &fakeStore{loadErr: ioErr}
	sdk := client.New("http://unused")
	probe := func(ctx context.Context, _ *client.APIClient) error {
		s.FailNow("probe must not be called on Load error other than ErrNoToken")
		return nil
	}

	err := auth.Bootstrap(context.Background(), store, sdk, probe)
	s.Require().Error(err)
	s.True(errors.Is(err, auth.ErrTokenStoreUnreadable), "expected ErrTokenStoreUnreadable, got %v", err)
	s.True(errors.Is(err, ioErr), "wrapped IO error must remain reachable")
}

// B6: Bootstrap returns ErrTokenStoreUnwritable when probe succeeds
// but Save fails. The error message must include the orphaned token
// for emergency recovery.
func (s *BootstrapSuite) TestBootstrapReturnsUnwritableWithOrphanedToken() {
	saveErr := errors.New("read-only filesystem")
	store := &fakeStore{loadErr: auth.ErrNoToken, saveErr: saveErr}
	sdk := client.New("http://unused")
	probe := func(ctx context.Context, c *client.APIClient) error {
		c.SetToken("ORPHANED-XYZ")
		return nil
	}

	err := auth.Bootstrap(context.Background(), store, sdk, probe)
	s.Require().Error(err)
	s.True(errors.Is(err, auth.ErrTokenStoreUnwritable), "expected ErrTokenStoreUnwritable, got %v", err)
	s.True(errors.Is(err, saveErr), "wrapped save error must remain reachable")
	s.Contains(err.Error(), "ORPHANED-XYZ", "error must include the orphaned token for recovery")
}

// B7: Bootstrap returns a clear server-bug error when the probe
// succeeds but sdk.Token() remains empty.
func (s *BootstrapSuite) TestBootstrapReturnsServerBugWhenTokenStillEmpty() {
	store := &fakeStore{loadErr: auth.ErrNoToken}
	sdk := client.New("http://unused")
	probe := func(ctx context.Context, _ *client.APIClient) error {
		return nil // success without setting a token
	}

	err := auth.Bootstrap(context.Background(), store, sdk, probe)
	s.Require().Error(err)
	s.Contains(strings.ToLower(err.Error()), "auto-issue", "expected error to mention auto-issue")
	s.Empty(store.saveCalls, "Save must not be called when no token was issued")
}
