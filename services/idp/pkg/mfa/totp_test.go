package mfa

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// test vectors from RFC 4226 appendix D
func TestHOTP(t *testing.T) {
	secret := []byte("12345678901234567890")
	expected := []string{"755224", "287082", "359152", "969429", "338314", "254676", "287922", "162583", "399871", "520489"}
	for counter, code := range expected {
		assert.Equal(t, code, hotp(secret, uint64(counter)))
	}
}

// test vectors from RFC 6238 appendix B (SHA1), truncated to 6 digits
func TestVerifyTOTPVectors(t *testing.T) {
	secret := []byte("12345678901234567890")
	for unix, code := range map[int64]string{
		59:          "287082",
		1111111109:  "081804",
		1111111111:  "050471",
		1234567890:  "005924",
		2000000000:  "279037",
		20000000000: "353130",
	} {
		step, ok := verifyTOTP(secret, code, time.Unix(unix, 0), 0)
		assert.True(t, ok, "code for %d", unix)
		assert.Equal(t, unix/30, step)
	}
}

func TestVerifyTOTP(t *testing.T) {
	secret := []byte("12345678901234567890")
	now := time.Unix(1111111111, 0)
	current := totpStep(now)

	t.Run("accepts the previous and the next step", func(t *testing.T) {
		step, ok := verifyTOTP(secret, hotp(secret, uint64(current-1)), now, 0)
		assert.True(t, ok)
		assert.Equal(t, current-1, step)

		step, ok = verifyTOTP(secret, hotp(secret, uint64(current+1)), now, 0)
		assert.True(t, ok)
		assert.Equal(t, current+1, step)
	})

	t.Run("rejects steps outside of the window", func(t *testing.T) {
		_, ok := verifyTOTP(secret, hotp(secret, uint64(current-2)), now, 0)
		assert.False(t, ok)
		_, ok = verifyTOTP(secret, hotp(secret, uint64(current+2)), now, 0)
		assert.False(t, ok)
	})

	t.Run("rejects a code that was used already", func(t *testing.T) {
		code := hotp(secret, uint64(current))
		_, ok := verifyTOTP(secret, code, now, current)
		assert.False(t, ok)
		_, ok = verifyTOTP(secret, code, now, current+1)
		assert.False(t, ok)
		_, ok = verifyTOTP(secret, code, now, current-1)
		assert.True(t, ok)
	})

	t.Run("rejects malformed codes", func(t *testing.T) {
		for _, code := range []string{"", "12345", "1234567", "abcdef"} {
			_, ok := verifyTOTP(secret, code, now, 0)
			assert.False(t, ok, code)
		}
	})

	t.Run("rejects the code of another secret", func(t *testing.T) {
		_, ok := verifyTOTP([]byte("another secret 12345"), hotp(secret, uint64(current)), now, 0)
		assert.False(t, ok)
	})
}

func TestProvisioningURI(t *testing.T) {
	secret := []byte("12345678901234567890")
	uri := provisioningURI("My Cloud", "alan@example.org", secret)

	u, err := url.Parse(uri)
	require.NoError(t, err)
	assert.Equal(t, "otpauth", u.Scheme)
	assert.Equal(t, "totp", u.Host)
	assert.Equal(t, "/My Cloud:alan@example.org", u.Path)
	assert.Equal(t, "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", u.Query().Get("secret"))
	assert.Equal(t, "My Cloud", u.Query().Get("issuer"))
	assert.Equal(t, "SHA1", u.Query().Get("algorithm"))
	assert.Equal(t, "6", u.Query().Get("digits"))
	assert.Equal(t, "30", u.Query().Get("period"))
	assert.NotContains(t, uri, "+")
}

func TestGenerateSecret(t *testing.T) {
	a, err := generateSecret()
	require.NoError(t, err)
	b, err := generateSecret()
	require.NoError(t, err)
	assert.Len(t, a, totpSecretSize)
	assert.NotEqual(t, a, b)
}
