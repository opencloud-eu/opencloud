// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package guard

import (
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNonceIssueAndVerify(t *testing.T) {
	now := time.Now()
	n := NewNonce(config.Nonce{TTL: time.Minute})
	var s NonceState

	nonce, err := n.Issue(&s, now)
	require.NoError(t, err)
	assert.NotEmpty(t, nonce)
	assert.NotEmpty(t, s.Hash)

	require.NoError(t, n.Verify(&s, nonce, now))
	assert.ErrorIs(t, n.Verify(&s, "other", now), ErrNonceMismatch)
}

func TestNonceVerifyMissing(t *testing.T) {
	n := NewNonce(config.Nonce{TTL: time.Minute})
	var s NonceState
	assert.ErrorIs(t, n.Verify(&s, "whatever", time.Now()), ErrNonceMissing)
}

func TestNonceVerifyExpired(t *testing.T) {
	now := time.Now()
	n := NewNonce(config.Nonce{TTL: time.Minute})
	var s NonceState

	nonce, err := n.Issue(&s, now)
	require.NoError(t, err)
	assert.ErrorIs(t, n.Verify(&s, nonce, now.Add(2*time.Minute)), ErrNonceExpired)
}

func TestNonceClear(t *testing.T) {
	now := time.Now()
	n := NewNonce(config.Nonce{TTL: time.Minute})
	var s NonceState

	nonce, err := n.Issue(&s, now)
	require.NoError(t, err)

	n.Clear(&s)
	assert.Empty(t, s.Hash)
	assert.ErrorIs(t, n.Verify(&s, nonce, now), ErrNonceMissing)
}
