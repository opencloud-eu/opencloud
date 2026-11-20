package mfa

import (
	"bytes"
	"encoding/base32"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	microstore "go-micro.dev/v4/store"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

var (
	alan = User{ID: "openclouduuid=4c510ada-c86b-4815-8820-42cdf82c3d51", Username: "alan"}
	mary = User{ID: "openclouduuid=f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c", Username: "mary"}
)

// clock is a time source tests can move forward.
type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestService(t *testing.T) (*Service, *clock, microstore.Store) {
	t.Helper()
	st := microstore.NewMemoryStore()
	svc, err := New(st, []byte("0123456789abcdef0123456789abcdef"), "OpenCloud", log.NopLogger())
	require.NoError(t, err)
	c := &clock{t: time.Unix(1700000000, 0)}
	svc.now = c.now
	return svc, c, st
}

// codeFor returns the current one-time code for a base32 secret, like an authenticator app.
func codeFor(t *testing.T, secret string, now time.Time) string {
	t.Helper()
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	require.NoError(t, err)
	return hotp(raw, uint64(totpStep(now)))
}

// enroll enables two-factor authentication for the user and returns the secret and the
// recovery codes. The clock is advanced so the next code is not a replay.
func enroll(t *testing.T, svc *Service, c *clock, user User) (string, []string) {
	t.Helper()
	secret, _, err := svc.Setup(user)
	require.NoError(t, err)
	codes, err := svc.Enable(user, codeFor(t, secret, c.now()))
	require.NoError(t, err)
	c.advance(totpPeriod * time.Second)
	return secret, codes
}

func TestSetupAndEnable(t *testing.T) {
	svc, c, st := newTestService(t)

	status, err := svc.Status(alan)
	require.NoError(t, err)
	assert.False(t, status.Enabled)

	secret, uri, err := svc.Setup(alan)
	require.NoError(t, err)
	assert.Len(t, secret, 32)
	assert.Contains(t, uri, "otpauth://totp/OpenCloud:alan?secret="+secret)

	t.Run("a started setup does not enable anything", func(t *testing.T) {
		status, err := svc.Status(alan)
		require.NoError(t, err)
		assert.False(t, status.Enabled)

		res, err := svc.VerifyLogin(alan, "")
		require.NoError(t, err)
		assert.Equal(t, LoginNotEnrolled, res)
	})

	t.Run("a wrong code does not enable", func(t *testing.T) {
		_, err := svc.Enable(alan, "000000")
		assert.ErrorIs(t, err, ErrInvalidCode)
	})

	t.Run("another user cannot confirm the setup", func(t *testing.T) {
		_, err := svc.Enable(mary, codeFor(t, secret, c.now()))
		assert.ErrorIs(t, err, ErrNoPendingSetup)
	})

	codes, err := svc.Enable(alan, codeFor(t, secret, c.now()))
	require.NoError(t, err)
	assert.Len(t, codes, recoveryCodeCount)

	status, err = svc.Status(alan)
	require.NoError(t, err)
	assert.True(t, status.Enabled)
	assert.Equal(t, recoveryCodeCount, status.RecoveryCodesLeft)
	assert.True(t, c.now().Equal(status.EnabledAt))

	t.Run("nothing secret is stored in plain text", func(t *testing.T) {
		raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
		require.NoError(t, err)

		recs, err := st.Read(storeKey(alan.ID))
		require.NoError(t, err)
		require.Len(t, recs, 1)
		assert.False(t, bytes.Contains(recs[0].Value, raw))
		assert.NotContains(t, string(recs[0].Value), secret)
		for _, code := range codes {
			assert.NotContains(t, string(recs[0].Value), code)
			assert.NotContains(t, string(recs[0].Value), normalizeCode(code))
		}
	})

	t.Run("a second setup is rejected", func(t *testing.T) {
		_, _, err := svc.Setup(alan)
		assert.ErrorIs(t, err, ErrAlreadyEnabled)
		_, err = svc.Enable(alan, codeFor(t, secret, c.now()))
		assert.ErrorIs(t, err, ErrAlreadyEnabled)
	})

	t.Run("other users are not affected", func(t *testing.T) {
		res, err := svc.VerifyLogin(mary, "")
		require.NoError(t, err)
		assert.Equal(t, LoginNotEnrolled, res)
	})
}

func TestSetupExpires(t *testing.T) {
	svc, c, _ := newTestService(t)

	secret, _, err := svc.Setup(alan)
	require.NoError(t, err)

	c.advance(pendingSetupTTL + time.Second)
	_, err = svc.Enable(alan, codeFor(t, secret, c.now()))
	assert.ErrorIs(t, err, ErrNoPendingSetup)
}

func TestSetupReplacesPendingSecret(t *testing.T) {
	svc, c, _ := newTestService(t)

	first, _, err := svc.Setup(alan)
	require.NoError(t, err)
	second, _, err := svc.Setup(alan)
	require.NoError(t, err)
	require.NotEqual(t, first, second)

	_, err = svc.Enable(alan, codeFor(t, first, c.now()))
	assert.ErrorIs(t, err, ErrInvalidCode)
	_, err = svc.Enable(alan, codeFor(t, second, c.now()))
	assert.NoError(t, err)
}

func TestVerifyLogin(t *testing.T) {
	svc, c, _ := newTestService(t)
	secret, _ := enroll(t, svc, c, alan)

	t.Run("requires a code", func(t *testing.T) {
		for _, code := range []string{"", "  ", " - "} {
			res, err := svc.VerifyLogin(alan, code)
			require.NoError(t, err)
			assert.Equal(t, LoginCodeRequired, res)
			assert.False(t, res.Passed())
		}
	})

	t.Run("rejects a wrong code", func(t *testing.T) {
		res, err := svc.VerifyLogin(alan, "000000")
		require.NoError(t, err)
		assert.Equal(t, LoginCodeInvalid, res)
		assert.False(t, res.Passed())
	})

	t.Run("accepts the current code once", func(t *testing.T) {
		code := codeFor(t, secret, c.now())
		res, err := svc.VerifyLogin(alan, code)
		require.NoError(t, err)
		assert.Equal(t, LoginOK, res)
		assert.True(t, res.Passed())

		res, err = svc.VerifyLogin(alan, code)
		require.NoError(t, err)
		assert.Equal(t, LoginCodeInvalid, res, "a code must not be accepted twice")
	})

	t.Run("accepts a code typed with spaces", func(t *testing.T) {
		c.advance(totpPeriod * time.Second)
		code := codeFor(t, secret, c.now())
		res, err := svc.VerifyLogin(alan, code[:3]+" "+code[3:])
		require.NoError(t, err)
		assert.Equal(t, LoginOK, res)
	})

	t.Run("rejects the code of another user", func(t *testing.T) {
		marySecret, _ := enroll(t, svc, c, mary)
		res, err := svc.VerifyLogin(alan, codeFor(t, marySecret, c.now()))
		require.NoError(t, err)
		assert.Equal(t, LoginCodeInvalid, res)
	})
}

func TestEnableCodeCannotBeReplayedForLogin(t *testing.T) {
	svc, c, _ := newTestService(t)

	secret, _, err := svc.Setup(alan)
	require.NoError(t, err)
	code := codeFor(t, secret, c.now())
	_, err = svc.Enable(alan, code)
	require.NoError(t, err)

	res, err := svc.VerifyLogin(alan, code)
	require.NoError(t, err)
	assert.Equal(t, LoginCodeInvalid, res)
}

func TestRecoveryCodes(t *testing.T) {
	svc, c, _ := newTestService(t)
	secret, codes := enroll(t, svc, c, alan)

	for _, code := range codes {
		assert.Regexp(t, `^[a-z2-7]{5}-[a-z2-7]{5}$`, code)
	}

	t.Run("a recovery code works once", func(t *testing.T) {
		res, err := svc.VerifyLogin(alan, codes[0])
		require.NoError(t, err)
		assert.Equal(t, LoginOK, res)

		status, err := svc.Status(alan)
		require.NoError(t, err)
		assert.Equal(t, recoveryCodeCount-1, status.RecoveryCodesLeft)

		res, err = svc.VerifyLogin(alan, codes[0])
		require.NoError(t, err)
		assert.Equal(t, LoginCodeInvalid, res)
	})

	t.Run("is accepted in upper case and without dash", func(t *testing.T) {
		res, err := svc.VerifyLogin(alan, strings.ToUpper(strings.ReplaceAll(codes[1], "-", "")))
		require.NoError(t, err)
		assert.Equal(t, LoginOK, res)
	})

	t.Run("regenerating invalidates the old codes", func(t *testing.T) {
		_, err := svc.RegenerateRecoveryCodes(alan, "000000")
		assert.ErrorIs(t, err, ErrInvalidCode)

		fresh, err := svc.RegenerateRecoveryCodes(alan, codeFor(t, secret, c.now()))
		require.NoError(t, err)
		require.Len(t, fresh, recoveryCodeCount)

		res, err := svc.VerifyLogin(alan, codes[2])
		require.NoError(t, err)
		assert.Equal(t, LoginCodeInvalid, res)

		res, err = svc.VerifyLogin(alan, fresh[0])
		require.NoError(t, err)
		assert.Equal(t, LoginOK, res)
	})

	t.Run("requires two-factor authentication to be enabled", func(t *testing.T) {
		_, err := svc.RegenerateRecoveryCodes(mary, "000000")
		assert.ErrorIs(t, err, ErrNotEnabled)
	})
}

func TestLockout(t *testing.T) {
	svc, c, _ := newTestService(t)
	secret, codes := enroll(t, svc, c, alan)

	for i := 0; i < maxFailedAttempts-1; i++ {
		res, err := svc.VerifyLogin(alan, "000000")
		require.NoError(t, err)
		assert.Equal(t, LoginCodeInvalid, res)
	}
	res, err := svc.VerifyLogin(alan, "000000")
	require.NoError(t, err)
	assert.Equal(t, LoginLocked, res)

	t.Run("correct codes are rejected while locked", func(t *testing.T) {
		res, err := svc.VerifyLogin(alan, codeFor(t, secret, c.now()))
		require.NoError(t, err)
		assert.Equal(t, LoginLocked, res)

		res, err = svc.VerifyLogin(alan, codes[0])
		require.NoError(t, err)
		assert.Equal(t, LoginLocked, res)

		assert.ErrorIs(t, svc.Disable(alan, codeFor(t, secret, c.now())), ErrLocked)
	})

	t.Run("the recovery code was not consumed while locked", func(t *testing.T) {
		status, err := svc.Status(alan)
		require.NoError(t, err)
		assert.Equal(t, recoveryCodeCount, status.RecoveryCodesLeft)
	})

	t.Run("works again after the lock expired", func(t *testing.T) {
		c.advance(lockDuration + time.Second)
		res, err := svc.VerifyLogin(alan, codeFor(t, secret, c.now()))
		require.NoError(t, err)
		assert.Equal(t, LoginOK, res)
	})

	t.Run("every further round of wrong codes locks longer", func(t *testing.T) {
		c.advance(totpPeriod * time.Second)
		for i := 0; i < maxFailedAttempts-1; i++ {
			_, err := svc.VerifyLogin(alan, "000000")
			require.NoError(t, err)
		}
		// the correct code above reset the count, so this is the first round again
		res, err := svc.VerifyLogin(alan, "000000")
		require.NoError(t, err)
		assert.Equal(t, LoginLocked, res)

		c.advance(lockDuration + time.Second)
		for i := 0; i < maxFailedAttempts-1; i++ {
			res, err := svc.VerifyLogin(alan, "000000")
			require.NoError(t, err)
			assert.Equal(t, LoginCodeInvalid, res, "the count goes on after a lock expired")
		}
		res, err = svc.VerifyLogin(alan, "000000")
		require.NoError(t, err)
		assert.Equal(t, LoginLocked, res)

		c.advance(lockDuration + time.Second)
		res, err = svc.VerifyLogin(alan, codeFor(t, secret, c.now()))
		require.NoError(t, err)
		assert.Equal(t, LoginLocked, res, "the second lock lasts twice as long")

		c.advance(lockDuration)
		res, err = svc.VerifyLogin(alan, codeFor(t, secret, c.now()))
		require.NoError(t, err)
		assert.Equal(t, LoginOK, res)
	})

	t.Run("a correct code resets the counter", func(t *testing.T) {
		for i := 0; i < maxFailedAttempts-1; i++ {
			_, err := svc.VerifyLogin(alan, "000000")
			require.NoError(t, err)
		}
		c.advance(totpPeriod * time.Second)
		res, err := svc.VerifyLogin(alan, codeFor(t, secret, c.now()))
		require.NoError(t, err)
		assert.Equal(t, LoginOK, res)

		res, err = svc.VerifyLogin(alan, "000000")
		require.NoError(t, err)
		assert.Equal(t, LoginCodeInvalid, res)
	})
}

func TestDisable(t *testing.T) {
	svc, c, _ := newTestService(t)
	secret, _ := enroll(t, svc, c, alan)

	assert.ErrorIs(t, svc.Disable(alan, ""), ErrInvalidCode)
	assert.ErrorIs(t, svc.Disable(alan, "000000"), ErrInvalidCode)
	assert.ErrorIs(t, svc.Disable(mary, "000000"), ErrNotEnabled)

	require.NoError(t, svc.Disable(alan, codeFor(t, secret, c.now())))

	status, err := svc.Status(alan)
	require.NoError(t, err)
	assert.False(t, status.Enabled)

	res, err := svc.VerifyLogin(alan, "")
	require.NoError(t, err)
	assert.Equal(t, LoginNotEnrolled, res)

	t.Run("can be set up again with a new secret", func(t *testing.T) {
		again, _ := enroll(t, svc, c, alan)
		assert.NotEqual(t, secret, again)
	})
}

func TestListAndAdminDisable(t *testing.T) {
	svc, c, _ := newTestService(t)
	enroll(t, svc, c, mary)
	enroll(t, svc, c, alan)
	// a setup that was never confirmed is not listed
	_, _, err := svc.Setup(User{ID: "id-of-lynn", Username: "lynn"})
	require.NoError(t, err)

	enrollments, err := svc.List()
	require.NoError(t, err)
	require.Len(t, enrollments, 2)
	assert.Equal(t, alan, enrollments[0].User)
	assert.Equal(t, mary, enrollments[1].User)

	_, err = svc.AdminDisable("lynn")
	assert.ErrorIs(t, err, ErrUserNotFound)
	_, err = svc.AdminDisable("nobody")
	assert.ErrorIs(t, err, ErrUserNotFound)

	user, err := svc.AdminDisable("Alan")
	require.NoError(t, err)
	assert.Equal(t, alan, user)

	user, err = svc.AdminDisable(mary.ID)
	require.NoError(t, err)
	assert.Equal(t, mary, user)

	enrollments, err = svc.List()
	require.NoError(t, err)
	assert.Empty(t, enrollments)

	res, err := svc.VerifyLogin(alan, "")
	require.NoError(t, err)
	assert.Equal(t, LoginNotEnrolled, res)
}

func TestSecretsNeedTheSameEncryptionSecret(t *testing.T) {
	svc, c, st := newTestService(t)
	secret, _ := enroll(t, svc, c, alan)

	other, err := New(st, []byte("a different encryption secret!!!"), "OpenCloud", log.NopLogger())
	require.NoError(t, err)
	other.now = c.now

	// the login must fail instead of falling back to the password only
	res, err := other.VerifyLogin(alan, codeFor(t, secret, c.now()))
	assert.Error(t, err)
	assert.False(t, res.Passed())
}

func TestNew(t *testing.T) {
	_, err := New(nil, []byte("0123456789abcdef0123456789abcdef"), "", log.NopLogger())
	assert.Error(t, err)
	_, err = New(microstore.NewMemoryStore(), []byte("short"), "", log.NopLogger())
	assert.Error(t, err)

	svc, err := New(microstore.NewMemoryStore(), []byte("0123456789abcdef0123456789abcdef"), "", log.NopLogger())
	require.NoError(t, err)
	assert.Equal(t, "OpenCloud", svc.issuer)
}

func TestLockDurationAfter(t *testing.T) {
	assert.Equal(t, lockDuration, lockDurationAfter(maxFailedAttempts))
	assert.Equal(t, 2*lockDuration, lockDurationAfter(2*maxFailedAttempts))
	assert.Equal(t, 8*lockDuration, lockDurationAfter(4*maxFailedAttempts))
	assert.Equal(t, maxLockDuration, lockDurationAfter(10*maxFailedAttempts))
	assert.Equal(t, maxLockDuration, lockDurationAfter(1000*maxFailedAttempts))
}
