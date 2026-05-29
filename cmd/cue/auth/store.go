package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
// Construct via NewFileStore. The directory is assumed to already
// exist (created by the existing config bootstrap).
type FileStore struct {
	path string
}

// NewFileStore returns a FileStore that persists to the given path.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// Load reads the token from disk, stripping trailing whitespace.
//
// Returns ErrNoToken when the file does not exist (the only error
// that Bootstrap routes to the auto-issue probe path). Any other
// IO/permission error is returned wrapped so callers can surface
// it as ErrTokenStoreUnreadable.
func (f *FileStore) Load(ctx context.Context) (string, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrNoToken
		}
		return "", fmt.Errorf("read token file %s: %w", f.path, err)
	}
	return strings.TrimRight(string(data), " \t\r\n"), nil
}

// Save atomically writes token to disk with mode 0600.
//
// Implementation: write to <path>.tmp in the same directory, fsync,
// close, then rename onto the destination. A crash before rename
// leaves the previous token intact.
func (f *FileStore) Save(ctx context.Context, token string) error {
	dir := filepath.Dir(f.path)
	tmp := f.path + ".tmp"

	// Open with O_TRUNC so a stale .tmp from a crashed prior save
	// does not corrupt the new write. The path is supplied by the
	// constructor caller (cmd/cue resolves to ~/.cue/client-token);
	// no untrusted input flows in.
	file, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) // #nosec G304 -- caller-controlled path under ~/.cue
	if err != nil {
		return fmt.Errorf("create temp token file in %s: %w", dir, err)
	}

	if _, err := file.Write([]byte(token)); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("write temp token file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("fsync temp token file: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close temp token file: %w", err)
	}

	if err := os.Rename(tmp, f.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename temp token file to %s: %w", f.path, err)
	}
	return nil
}

// Delete removes the token file. A non-existent file is treated as
// success (idempotent reset).
func (f *FileStore) Delete(ctx context.Context) error {
	if err := os.Remove(f.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove token file %s: %w", f.path, err)
	}
	return nil
}
