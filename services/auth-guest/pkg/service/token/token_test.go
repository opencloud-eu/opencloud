// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testShareID = "e0123456-7890-abcd-ef01-234567890abc"

func TestGenerateAndString(t *testing.T) {
	svc := NewTokenService()

	tok, err := svc.Generate(testShareID)
	require.NoError(t, err)
	assert.Equal(t, Hash(testShareID), tok.ShareIDHash)
	assert.NotEmpty(t, tok.SecretHash())
	assert.NotEmpty(t, tok.String())
}

func TestGenerateRandomizesSecret(t *testing.T) {
	svc := NewTokenService()

	tok1, err := svc.Generate(testShareID)
	require.NoError(t, err)
	tok2, err := svc.Generate(testShareID)
	require.NoError(t, err)

	assert.Equal(t, tok1.ShareIDHash, tok2.ShareIDHash)
	assert.NotEqual(t, tok1.SecretHash(), tok2.SecretHash())
	assert.NotEqual(t, tok1.String(), tok2.String())

	other, err := svc.Generate("9f9f9f9-9f9f-9f9f-9f9f-9f9f9f9f9f9f")
	require.NoError(t, err)
	assert.NotEqual(t, tok1.ShareIDHash, other.ShareIDHash)
}

func TestParse(t *testing.T) {
	svc := NewTokenService()
	original, err := svc.Generate(testShareID)
	require.NoError(t, err)

	tests := []struct {
		name    string
		encoded string
		wantErr bool
	}{
		{name: "valid", encoded: original.String()},
		{name: "wrong version", encoded: "v2." + original.ShareIDHash + "." + original.secret, wantErr: true},
		{name: "missing version", encoded: original.ShareIDHash + "." + original.secret, wantErr: true},
		{name: "too many parts", encoded: original.String() + ".extra", wantErr: true},
		{name: "empty share hash", encoded: "v1.." + original.secret, wantErr: true},
		{name: "empty secret", encoded: "v1." + original.ShareIDHash + ".", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := svc.Parse(tt.encoded)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidToken)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, original.ShareIDHash, parsed.ShareIDHash)
			assert.Equal(t, original.SecretHash(), parsed.SecretHash())
			assert.Equal(t, original.String(), parsed.String())
		})
	}
}

func TestVerify(t *testing.T) {
	svc := NewTokenService()
	tok, err := svc.Generate(testShareID)
	require.NoError(t, err)

	tests := []struct {
		name             string
		token            Token
		storedSecretHash string
		wantErr          bool
	}{
		{name: "valid", token: *tok, storedSecretHash: tok.SecretHash()},
		{name: "wrong stored secret", token: *tok, storedSecretHash: Hash("other-secret"), wantErr: true},
		{name: "missing fields", token: Token{ShareIDHash: tok.ShareIDHash}, storedSecretHash: tok.SecretHash(), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Verify(tt.token, tt.storedSecretHash)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidToken)
				return
			}
			assert.NoError(t, err)
		})
	}
}
