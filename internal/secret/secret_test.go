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
