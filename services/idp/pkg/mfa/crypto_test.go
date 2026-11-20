package mfa

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSealer(t *testing.T) {
	s, err := newSealer([]byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)

	sealed, err := s.seal([]byte("secret"), "user-1")
	require.NoError(t, err)
	assert.NotContains(t, string(sealed), "secret")

	t.Run("round trip", func(t *testing.T) {
		plain, err := s.open(sealed, "user-1")
		require.NoError(t, err)
		assert.Equal(t, []byte("secret"), plain)
	})

	t.Run("uses a fresh nonce", func(t *testing.T) {
		again, err := s.seal([]byte("secret"), "user-1")
		require.NoError(t, err)
		assert.NotEqual(t, sealed, again)
	})

	t.Run("is bound to the user", func(t *testing.T) {
		_, err := s.open(sealed, "user-2")
		assert.Error(t, err)
	})

	t.Run("detects tampering", func(t *testing.T) {
		tampered := append([]byte{}, sealed...)
		tampered[len(tampered)-1] ^= 1
		_, err := s.open(tampered, "user-1")
		assert.Error(t, err)
		_, err = s.open([]byte("short"), "user-1")
		assert.Error(t, err)
	})

	t.Run("needs the same master secret", func(t *testing.T) {
		other, err := newSealer([]byte("another master secret, long enough"))
		require.NoError(t, err)
		_, err = other.open(sealed, "user-1")
		assert.Error(t, err)
	})

	t.Run("rejects short master secrets", func(t *testing.T) {
		_, err := newSealer([]byte("short"))
		assert.Error(t, err)
	})
}
