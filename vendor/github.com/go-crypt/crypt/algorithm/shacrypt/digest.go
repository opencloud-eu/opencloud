package shacrypt

import (
	"crypto/subtle"
	"fmt"
	"strings"

	xcrypt "github.com/go-crypt/x/crypt"

	"github.com/go-crypt/crypt/algorithm"
)

// Digest is a digest which handles SHA-crypt hashes like SHA256 or SHA512.
type Digest struct {
	variant Variant

	iterations int
	salt, key  []byte
}

// Match returns true if the string password matches the current shacrypt.Digest.
func (d *Digest) Match(password string) (match bool) {
	return d.MatchBytes([]byte(password))
}

// MatchBytes returns true if the []byte passwordBytes matches the current shacrypt.Digest.
func (d *Digest) MatchBytes(passwordBytes []byte) (match bool) {
	match, _ = d.MatchBytesAdvanced(passwordBytes)

	return match
}

// MatchAdvanced is the same as Match except if there is an error it returns that as well.
func (d *Digest) MatchAdvanced(password string) (match bool, err error) {
	if match, err = d.MatchBytesAdvanced([]byte(password)); err != nil {
		return match, fmt.Errorf(algorithm.ErrFmtDigestMatch, AlgName, err)
	}

	return match, nil
}

// MatchBytesAdvanced is the same as MatchBytes except if there is an error it returns that as well.
func (d *Digest) MatchBytesAdvanced(passwordBytes []byte) (match bool, err error) {
	if len(d.key) == 0 {
		return false, fmt.Errorf("%w: key has 0 bytes", algorithm.ErrPasswordInvalid)
	}

	return subtle.ConstantTimeCompare(d.key, xcrypt.KeySHACrypt(d.variant.HashFunc(), passwordBytes, d.salt, d.iterations)) == 1, nil
}

// Encode this Digest as a string for storage.
func (d *Digest) Encode() (hash string) {
	switch d.iterations {
	case IterationsDefaultOmitted:
		return strings.ReplaceAll(fmt.Sprintf(EncodingFmtRoundsOmitted,
			d.variant.Prefix(),
			d.salt, d.key,
		), "\n", "")
	default:
		return strings.ReplaceAll(fmt.Sprintf(EncodingFmt,
			d.variant.Prefix(), d.iterations,
			d.salt, d.key,
		), "\n", "")
	}
}

// String returns the storable format of the shacrypt.Digest hash utilizing fmt.Sprintf and shacrypt.EncodingFmt.
func (d *Digest) String() string {
	return d.Encode()
}

func (d *Digest) defaults() {
	switch d.variant {
	case VariantSHA256, VariantSHA512:
		break
	default:
		d.variant = variantDefault
	}

	if d.iterations == 0 {
		d.iterations = d.variant.DefaultIterations()
	}
}
