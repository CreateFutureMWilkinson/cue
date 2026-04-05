package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotImplemented is returned by stubs awaiting implementation.
var ErrNotImplemented = errors.New("not implemented")

// ReadKeyFromFile reads exactly 32 bytes from rc and closes it,
// returning an error if either the read or the close fails.
func ReadKeyFromFile(rc io.ReadCloser) ([]byte, error) {
	key, readErr := io.ReadAll(rc)
	closeErr := rc.Close()
	if readErr != nil {
		return nil, fmt.Errorf("reading key file: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("closing key file: %w", closeErr)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("key file must be exactly 32 bytes, got %d", len(key))
	}
	return key, nil
}

// Encryptor defines the contract for symmetric encryption/decryption.
type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// KeyFileEncryptor implements Encryptor using AES-256-GCM with a key stored on disk.
type KeyFileEncryptor struct {
	gcm cipher.AEAD
}

// NewKeyFileEncryptor creates a KeyFileEncryptor. If keyPath does not exist, a new
// 32-byte random key is generated and written with mode 0600. If it exists, the file
// must contain exactly 32 bytes. Paths containing ".." traversal components are rejected.
func NewKeyFileEncryptor(keyPath string) (*KeyFileEncryptor, error) {
	// Reject paths containing ".." components to prevent path traversal (G304).
	for part := range strings.SplitSeq(filepath.ToSlash(keyPath), "/") {
		if part == ".." {
			return nil, fmt.Errorf("key path must not contain path traversal (..): %s", keyPath)
		}
	}

	cleaned := filepath.Clean(keyPath)
	dir := filepath.Dir(cleaned)
	base := filepath.Base(cleaned)

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("opening key directory: %w", err)
	}
	defer root.Close()

	var key []byte

	if _, err := root.Stat(base); os.IsNotExist(err) {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("generating key: %w", err)
		}
		f, err := root.OpenFile(base, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return nil, fmt.Errorf("writing key file: %w", err)
		}
		_, writeErr := f.Write(key)
		closeErr := f.Close()
		if writeErr != nil {
			return nil, fmt.Errorf("writing key file: %w", writeErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("closing key file: %w", closeErr)
		}
	} else {
		f, err := root.Open(base)
		if err != nil {
			return nil, fmt.Errorf("reading key file: %w", err)
		}
		key, err = ReadKeyFromFile(f)
		if err != nil {
			return nil, err
		}
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	return &KeyFileEncryptor{gcm: gcm}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with a random nonce.
// Returns the nonce prepended to the ciphertext (nonce || ciphertext).
func (e *KeyFileEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating random nonce: %w", err)
	}
	return e.gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts data produced by Encrypt. The input must contain a nonce prepended
// to the ciphertext and be at least 12 bytes (the GCM nonce size).
func (e *KeyFileEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short: expected at least %d bytes (nonce size), got %d", nonceSize, len(ciphertext))
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return e.gcm.Open(nil, nonce, ct, nil)
}
