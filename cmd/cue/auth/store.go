package auth

import (
	"context"
)

// TokenStore is the persistence abstraction for the client's bearer
// token. The default implementation (FileStore) reads and writes
// ~/.cue/client-token; tests inject in-memory implementations.
type TokenStore interface {
	// Load returns the persisted token. Trailing whitespace is
	// stripped. If no token file exists, Load returns ErrNoToken.
	Load(ctx context.Context) (string, error)

	// Save persists token to the backing store. Implementations
	// must guarantee the operation is atomic (crash mid-write must
	// preserve the prior value) and that the resulting file is
	// readable only by the current user (mode 0600 on POSIX).
	Save(ctx context.Context, token string) error

	// Delete removes the persisted token. Subsequent Load calls
	// must return ErrNoToken. Deleting a non-existent token is a
	// no-op; implementations must not return an error in that case.
	Delete(ctx context.Context) error
}

// FileStore persists the token as a plaintext file with mode 0600.
// Save uses an atomic temp-file-and-rename so a crash mid-write
// preserves any prior token.
//
// Construct via NewFileStore. The default path resolves to
// ~/.cue/client-token; pass an explicit path to override (used by
// tests with a t.TempDir()).
type FileStore struct {
	path string
}

// NewFileStore returns a FileStore that persists to the given path.
// An empty path resolves to ~/.cue/client-token. The directory is
// assumed to already exist (created by the existing config bootstrap).
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// Load implements TokenStore.
func (f *FileStore) Load(ctx context.Context) (string, error) {
	return "", ErrNotImplemented
}

// Save implements TokenStore.
func (f *FileStore) Save(ctx context.Context, token string) error {
	return ErrNotImplemented
}

// Delete implements TokenStore.
func (f *FileStore) Delete(ctx context.Context) error {
	return ErrNotImplemented
}
