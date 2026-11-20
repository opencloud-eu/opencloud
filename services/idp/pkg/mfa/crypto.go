package mfa

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"errors"
)

const (
	// minMasterSecretSize is the minimal size of the key the encryption key is derived from.
	minMasterSecretSize = 16
	// keyInfo binds the derived key to its purpose. Changing it invalidates all stored secrets.
	keyInfo = "opencloud idp mfa secret encryption v1"
)

// sealer encrypts the TOTP secrets before they are written to the store.
type sealer struct {
	aead cipher.AEAD
}

// newSealer derives an AES-256-GCM key from the master secret.
func newSealer(masterSecret []byte) (*sealer, error) {
	if len(masterSecret) < minMasterSecretSize {
		return nil, errors.New("mfa: encryption secret is too short")
	}

	key, err := hkdf.Key(sha256.New, masterSecret, nil, keyInfo, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &sealer{aead: aead}, nil
}

// seal encrypts plaintext. The binding is authenticated but not encrypted, it ties the
// ciphertext to its owner so it cannot be copied to another user's record.
func (s *sealer) seal(plaintext []byte, binding string) ([]byte, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return s.aead.Seal(nonce, nonce, plaintext, []byte(binding)), nil
}

// open decrypts a value created by seal.
func (s *sealer) open(sealed []byte, binding string) ([]byte, error) {
	if len(sealed) < s.aead.NonceSize() {
		return nil, errors.New("mfa: sealed value is too short")
	}
	nonce, ciphertext := sealed[:s.aead.NonceSize()], sealed[s.aead.NonceSize():]
	return s.aead.Open(nil, nonce, ciphertext, []byte(binding))
}
