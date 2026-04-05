package secret_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CreateFutureMWilkinson/cue/internal/secret"
	"github.com/stretchr/testify/suite"
)

type KeyFileEncryptorSuite struct {
	suite.Suite
}

func TestKeyFileEncryptor(t *testing.T) {
	suite.Run(t, new(KeyFileEncryptorSuite))
}

func (s *KeyFileEncryptorSuite) TestRoundTrip() {
	keyPath := filepath.Join(s.T().TempDir(), "test.key")

	enc, err := secret.NewKeyFileEncryptor(keyPath)
	s.Require().NoError(err)

	plaintext := []byte("hello, world!")
	ciphertext, err := enc.Encrypt(plaintext)
	s.Require().NoError(err)

	decrypted, err := enc.Decrypt(ciphertext)
	s.Require().NoError(err)
	s.Equal(plaintext, decrypted)
}

func (s *KeyFileEncryptorSuite) TestTamperedCiphertext() {
	keyPath := filepath.Join(s.T().TempDir(), "test.key")

	enc, err := secret.NewKeyFileEncryptor(keyPath)
	s.Require().NoError(err)

	plaintext := []byte("sensitive data")
	ciphertext, err := enc.Encrypt(plaintext)
	s.Require().NoError(err)

	// Tamper with a byte in the ciphertext (after the nonce).
	ciphertext[len(ciphertext)-1] ^= 0xFF

	_, err = enc.Decrypt(ciphertext)
	s.Error(err)
}

func (s *KeyFileEncryptorSuite) TestDecryptTooShortInput() {
	keyPath := filepath.Join(s.T().TempDir(), "test.key")

	enc, err := secret.NewKeyFileEncryptor(keyPath)
	s.Require().NoError(err)

	// Fewer than 12 bytes (nonce size).
	shortInput := []byte("tooshort")
	s.Less(len(shortInput), 12)

	_, err = enc.Decrypt(shortInput)
	s.Error(err)
}

func (s *KeyFileEncryptorSuite) TestKeyFileCreation() {
	keyPath := filepath.Join(s.T().TempDir(), "test.key")

	_, err := secret.NewKeyFileEncryptor(keyPath)
	s.Require().NoError(err)

	// Verify file exists with correct permissions and size.
	info, err := os.Stat(keyPath)
	s.Require().NoError(err)
	s.Equal(os.FileMode(0600), info.Mode().Perm())
	s.Equal(int64(32), info.Size())
}

func (s *KeyFileEncryptorSuite) TestKeyFileReuse() {
	keyPath := filepath.Join(s.T().TempDir(), "test.key")

	enc1, err := secret.NewKeyFileEncryptor(keyPath)
	s.Require().NoError(err)

	plaintext := []byte("cross-instance test")
	ciphertext, err := enc1.Encrypt(plaintext)
	s.Require().NoError(err)

	// Create a second encryptor with the same key path.
	enc2, err := secret.NewKeyFileEncryptor(keyPath)
	s.Require().NoError(err)

	decrypted, err := enc2.Decrypt(ciphertext)
	s.Require().NoError(err)
	s.Equal(plaintext, decrypted)
}

func (s *KeyFileEncryptorSuite) TestInvalidKeyLength() {
	keyPath := filepath.Join(s.T().TempDir(), "bad.key")

	// Write a key file that is not 32 bytes.
	err := os.WriteFile(keyPath, []byte("too-short-key"), 0600)
	s.Require().NoError(err)

	_, err = secret.NewKeyFileEncryptor(keyPath)
	s.Error(err)
	s.Contains(err.Error(), "32")
}

func (s *KeyFileEncryptorSuite) TestPathTraversalPrevention() {
	// Create two separate temp directories.
	allowedDir := s.T().TempDir()
	secretDir := s.T().TempDir()

	// Write a valid 32-byte key into secretDir.
	keyData := make([]byte, 32)
	for i := range keyData {
		keyData[i] = byte(i)
	}
	keyFileName := "traversal.key"
	err := os.WriteFile(filepath.Join(secretDir, keyFileName), keyData, 0600)
	s.Require().NoError(err)

	// Build a traversal path that starts in allowedDir but escapes via "../"
	// to reach the key in secretDir. Use string concatenation (not filepath.Join)
	// to preserve the ".." component — filepath.Join would normalize it away.
	// e.g. /tmp/allowedXXX/../secretXXX/traversal.key
	traversalPath := allowedDir + string(filepath.Separator) + ".." +
		string(filepath.Separator) + filepath.Base(secretDir) +
		string(filepath.Separator) + keyFileName

	// Verify the traversal path actually resolves to the real key file
	// (proving the path is valid on disk, just not scoped).
	resolved, err := filepath.EvalSymlinks(traversalPath)
	s.Require().NoError(err)
	expected, err := filepath.EvalSymlinks(filepath.Join(secretDir, keyFileName))
	s.Require().NoError(err)
	s.Equal(expected, resolved, "traversal path should resolve to the secret key on disk")

	// After the fix, NewKeyFileEncryptor should reject this path because
	// os.OpenRoot scopes access to the key file's parent directory and
	// "../" escapes that root.
	_, err = secret.NewKeyFileEncryptor(traversalPath)
	s.Error(err, "path traversal via '../' should be rejected")
}

func (s *KeyFileEncryptorSuite) TestNonceUniqueness() {
	keyPath := filepath.Join(s.T().TempDir(), "test.key")

	enc, err := secret.NewKeyFileEncryptor(keyPath)
	s.Require().NoError(err)

	plaintext := []byte("identical input")
	ct1, err := enc.Encrypt(plaintext)
	s.Require().NoError(err)

	ct2, err := enc.Encrypt(plaintext)
	s.Require().NoError(err)

	// Two encryptions of the same plaintext must produce different ciphertexts.
	s.NotEqual(ct1, ct2)
}
