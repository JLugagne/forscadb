package filecache

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "forscadb"
	keyringUser    = "master-key"
)

// masterKey returns the AES-256 key from the OS keychain, generating one if absent.
func masterKey() ([]byte, error) {
	encoded, err := keyring.Get(keyringService, keyringUser)
	if err == keyring.ErrNotFound {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("crypto: generate key: %w", err)
		}
		encoded = hex.EncodeToString(key)
		if err := keyring.Set(keyringService, keyringUser, encoded); err != nil {
			return nil, fmt.Errorf("crypto: store key in keyring: %w", err)
		}
		return key, nil
	}
	if err != nil {
		return nil, fmt.Errorf("crypto: read keyring: %w", err)
	}
	key, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("crypto: decode key: %w", err)
	}
	return key, nil
}

// encryptPassword encrypts plaintext with AES-256-GCM and returns a base64-encoded ciphertext.
// Returns an empty string for empty input.
func encryptPassword(plaintext string, key []byte) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptPassword decrypts a base64-encoded AES-256-GCM ciphertext.
// Returns an empty string for empty input.
func decryptPassword(encoded string, key []byte) (string, error) {
	if encoded == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("crypto: base64 decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("crypto: ciphertext too short")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: decrypt: %w", err)
	}
	return string(plaintext), nil
}
