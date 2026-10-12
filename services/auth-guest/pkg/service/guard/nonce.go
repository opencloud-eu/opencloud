// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package guard

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"time"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
)

// Nonce verification errors.
var (
	ErrNonceMissing  = errors.New("nonce missing")
	ErrNonceMismatch = errors.New("nonce mismatch")
	ErrNonceExpired  = errors.New("nonce expired")
)

const nonceLength = 32

// NonceState is the persisted browser-bound nonce.
type NonceState struct {
	Hash   string    `json:"hash,omitempty"`
	Expiry time.Time `json:"expiry,omitzero"`
}

// Nonce issues and verifies browser-bound nonces.
type Nonce struct {
	ttl time.Duration
}

// NewNonce returns a Nonce configured from cfg.
func NewNonce(cfg config.Nonce) Nonce {
	return Nonce{ttl: cfg.TTL}
}

// Issue generates a nonce, stores its hash and expiry, and returns the raw
// nonce to be handed to the client.
func (n Nonce) Issue(s *NonceState, now time.Time) (string, error) {
	b := make([]byte, nonceLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	value := base64.RawURLEncoding.EncodeToString(b)
	s.Hash = hash(value)
	s.Expiry = now.Add(n.ttl)
	return value, nil
}

// Verify checks the nonce against the stored hash and expiry.
func (n Nonce) Verify(s *NonceState, value string, now time.Time) error {
	if s.Hash == "" {
		return ErrNonceMissing
	}
	if subtle.ConstantTimeCompare([]byte(hash(value)), []byte(s.Hash)) != 1 {
		return ErrNonceMismatch
	}
	if !s.Expiry.IsZero() && s.Expiry.Before(now) {
		return ErrNonceExpired
	}
	return nil
}

// Clear removes the stored nonce.
func (n Nonce) Clear(s *NonceState) {
	*s = NonceState{}
}

func hash(nonce string) string {
	sum := sha256.Sum256([]byte(nonce))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
