package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"os"
)

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
// must contain exactly 32 bytes.
func NewKeyFileEncryptor(keyPath string) (*KeyFileEncryptor, error) {
	var key []byte

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("generating key: %w", err)
		}
		if err := os.WriteFile(keyPath, key, 0600); err != nil {
			return nil, fmt.Errorf("writing key file: %w", err)
		}
	} else {
		var err error
		key, err = os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("reading key file: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("key file must be exactly 32 bytes, got %d", len(key))
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

// Encrypt encrypts plaintext using AES-256-GCM. Returns nonce || ciphertext.
func (e *KeyFileEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}
	return e.gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts data produced by Encrypt. Input must be at least 12 bytes (nonce size).
func (e *KeyFileEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short: must be at least %d bytes", nonceSize)
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return e.gcm.Open(nil, nonce, ct, nil)
}
