// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package pin

import (
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	for range 50 {
		got, err := Generate()
		require.NoError(t, err)
		assert.Len(t, got, Length)
		for _, r := range got {
			assert.True(t, unicode.IsDigit(r))
		}
	}
}

func TestGenerateUnique(t *testing.T) {
	seen := make(map[string]struct{})
	for range 100 {
		got, err := Generate()
		require.NoError(t, err)
		seen[got] = struct{}{}
	}

	assert.GreaterOrEqual(t, len(seen), 2)
}

func TestHashProducesArgon2id(t *testing.T) {
	h, err := Hash("123456")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(h, "$argon2id$"))
	assert.NotEqual(t, "123456", h)
}

func TestHashEmptyPin(t *testing.T) {
	_, err := Hash("")
	assert.ErrorIs(t, err, ErrInvalidPin)
}

func TestVerify(t *testing.T) {
	h, err := Hash("123456")
	require.NoError(t, err)

	match, err := Verify("123456", h)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = Verify("654321", h)
	require.NoError(t, err)
	assert.False(t, match)
}

func TestVerifyEmptyHash(t *testing.T) {
	_, err := Verify("123456", "")
	assert.ErrorIs(t, err, ErrInvalidHash)
}

func TestVerifyMalformedHash(t *testing.T) {
	_, err := Verify("123456", "not-a-hash")
	assert.ErrorIs(t, err, ErrInvalidHash)
}
