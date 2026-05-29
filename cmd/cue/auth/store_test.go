package auth_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CreateFutureMWilkinson/cue/cmd/cue/auth"
)

// FileStoreSuite covers the on-disk TokenStore implementation.
type FileStoreSuite struct {
	suite.Suite
}

func TestFileStore(t *testing.T) {
	suite.Run(t, new(FileStoreSuite))
}

// pathInTempDir returns a path under t.TempDir() that the test will
// use as the FileStore backing file. The parent dir already exists.
func (s *FileStoreSuite) pathInTempDir() string {
	return filepath.Join(s.T().TempDir(), "client-token")
}

// A1: FileStore.Load returns the token with trailing whitespace stripped.
func (s *FileStoreSuite) TestLoadStripsTrailingWhitespace() {
	path := s.pathInTempDir()
	s.Require().NoError(os.WriteFile(path, []byte("abc\n"), 0o600))

	store := auth.NewFileStore(path)
	tok, err := store.Load(context.Background())
	s.Require().NoError(err)
	s.Equal("abc", tok)
}

// A2: FileStore.Load returns ErrNoToken when the file is absent.
func (s *FileStoreSuite) TestLoadReturnsErrNoTokenWhenAbsent() {
	store := auth.NewFileStore(s.pathInTempDir())
	_, err := store.Load(context.Background())
	s.Require().Error(err)
	s.True(errors.Is(err, auth.ErrNoToken), "expected ErrNoToken, got %v", err)
}

// A3: FileStore.Load returns a wrapped error on permission denied.
// The error must NOT be ErrNoToken so Bootstrap routes it to
// ErrTokenStoreUnreadable instead of probing.
func (s *FileStoreSuite) TestLoadWrapsPermissionError() {
	if os.Geteuid() == 0 {
		s.T().Skip("permission tests are meaningless when running as root")
	}
	path := s.pathInTempDir()
	s.Require().NoError(os.WriteFile(path, []byte("abc"), 0o000))
	s.T().Cleanup(func() { _ = os.Chmod(path, 0o600) })

	store := auth.NewFileStore(path)
	_, err := store.Load(context.Background())
	s.Require().Error(err)
	s.False(errors.Is(err, auth.ErrNoToken), "permission error must not be ErrNoToken")
}

// A4: FileStore.Save writes the file with mode 0600.
func (s *FileStoreSuite) TestSaveWritesMode0600() {
	path := s.pathInTempDir()
	store := auth.NewFileStore(path)
	s.Require().NoError(store.Save(context.Background(), "secret-token"))

	info, err := os.Stat(path)
	s.Require().NoError(err)
	s.Equal(os.FileMode(0o600), info.Mode().Perm())

	got, err := os.ReadFile(path)
	s.Require().NoError(err)
	s.Equal("secret-token", string(got))
}

// A5: FileStore.Save is atomic — the previous content survives if the
// rename never lands. We approximate "crash mid-write" by writing a
// pre-existing token then asserting that no .tmp file is visible after
// a successful save and that the token file content reflects the new
// value (rename, not in-place truncate).
func (s *FileStoreSuite) TestSaveIsAtomic() {
	path := s.pathInTempDir()
	s.Require().NoError(os.WriteFile(path, []byte("OLD"), 0o600))

	store := auth.NewFileStore(path)
	s.Require().NoError(store.Save(context.Background(), "NEW"))

	got, err := os.ReadFile(path)
	s.Require().NoError(err)
	s.Equal("NEW", string(got))

	// No leftover .tmp file should be visible to readers.
	_, err = os.Stat(path + ".tmp")
	s.True(os.IsNotExist(err), "temp file %s.tmp should not remain after save", path)
}

// A5b: When Save fails (e.g., destination dir is read-only), the
// previous token must remain intact. This exercises the atomic
// guarantee on the failure path.
func (s *FileStoreSuite) TestSaveFailurePreservesPreviousToken() {
	if os.Geteuid() == 0 {
		s.T().Skip("permission tests are meaningless when running as root")
	}
	dir := s.T().TempDir()
	path := filepath.Join(dir, "client-token")
	s.Require().NoError(os.WriteFile(path, []byte("OLD"), 0o600))

	// Make the directory read-only so the temp-file create or rename fails.
	s.Require().NoError(os.Chmod(dir, 0o500))
	s.T().Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	store := auth.NewFileStore(path)
	err := store.Save(context.Background(), "NEW")
	s.Require().Error(err)

	// Restore dir perms so we can read the file back.
	s.Require().NoError(os.Chmod(dir, 0o700))
	got, err := os.ReadFile(path)
	s.Require().NoError(err)
	s.Equal("OLD", string(got), "failed save must not corrupt the previous token")
}

// A6: FileStore.Delete removes the file; subsequent Load returns ErrNoToken.
func (s *FileStoreSuite) TestDeleteRemovesFile() {
	path := s.pathInTempDir()
	store := auth.NewFileStore(path)
	s.Require().NoError(store.Save(context.Background(), "T"))

	s.Require().NoError(store.Delete(context.Background()))

	_, err := store.Load(context.Background())
	s.True(errors.Is(err, auth.ErrNoToken), "expected ErrNoToken after Delete, got %v", err)
}

// A6b: Delete on a non-existent file is a no-op.
func (s *FileStoreSuite) TestDeleteAbsentIsNoop() {
	store := auth.NewFileStore(s.pathInTempDir())
	s.Require().NoError(store.Delete(context.Background()))
}
