package mfa

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- HMAC-SHA1 is what RFC 6238 authenticator apps implement
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	// totpDigits is the length of a one-time code.
	totpDigits = 6
	// totpPeriod is the validity of a one-time code in seconds.
	totpPeriod = 30
	// totpSkew is the number of periods before and after the current one that are accepted
	// to compensate for clock drift between the server and the authenticator app.
	totpSkew = 1
	// totpSecretSize is the size of a generated secret in bytes (160 bit, see RFC 4226 section 4).
	totpSecretSize = 20
)

var secretEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// generateSecret returns a new random TOTP secret.
func generateSecret() ([]byte, error) {
	secret := make([]byte, totpSecretSize)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	return secret, nil
}

// encodeSecret returns the secret in the base32 form authenticator apps expect.
func encodeSecret(secret []byte) string {
	return secretEncoding.EncodeToString(secret)
}

// hotp implements the HOTP algorithm from RFC 4226.
func hotp(secret []byte, counter uint64) string {
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, counter)

	mac := hmac.New(sha1.New, secret)
	mac.Write(msg)
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff

	return fmt.Sprintf("%0*d", totpDigits, value%1000000)
}

// totpStep returns the TOTP time step (RFC 6238) for the given time.
func totpStep(t time.Time) int64 {
	return t.Unix() / totpPeriod
}

// verifyTOTP checks code against the secret for the time steps around now. Steps up to and
// including lastStep are not accepted, so a code cannot be used twice. It returns the
// matching step.
func verifyTOTP(secret []byte, code string, now time.Time, lastStep int64) (int64, bool) {
	if len(code) != totpDigits {
		return 0, false
	}

	current := totpStep(now)
	matched := int64(0)
	ok := false
	// always check all steps to not leak which one matched
	for step := current - totpSkew; step <= current+totpSkew; step++ {
		if step < 0 {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(hotp(secret, uint64(step))), []byte(code)) == 1 && step > lastStep {
			matched = step
			ok = true
		}
	}
	return matched, ok
}

// provisioningURI returns the otpauth:// URI that authenticator apps read from a QR code.
// See https://github.com/google/google-authenticator/wiki/Key-Uri-Format
func provisioningURI(issuer, account string, secret []byte) string {
	// authenticator apps expect %20 instead of + for spaces
	escape := func(s string) string {
		return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
	}
	// a colon separates issuer and account in the label
	label := url.PathEscape(strings.ReplaceAll(issuer, ":", "")) + ":" + url.PathEscape(account)

	return fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		label, encodeSecret(secret), escape(issuer), totpDigits, totpPeriod)
}
