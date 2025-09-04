package shacrypt

import (
	"fmt"

	xcrypt "github.com/go-crypt/x/crypt"

	"github.com/go-crypt/crypt/algorithm"
	"github.com/go-crypt/crypt/internal/random"
)

// New returns a *Hasher without any settings configured. This d to a SHA512 hash.Hash
// with 1000000 iterations. These settings can be overridden with the methods with the With prefix.
func New(opts ...Opt) (hasher *Hasher, err error) {
	hasher = &Hasher{}

	if err = hasher.WithOptions(opts...); err != nil {
		return nil, err
	}

	if err = hasher.Validate(); err != nil {
		return nil, err
	}

	return hasher, nil
}

// Hasher is a algorithm.Hash for SHA-crypt which can be initialized via shacrypt.New using a functional options pattern.
type Hasher struct {
	variant Variant

	iterations, bytesSalt int

	d bool
}

// NewSHA256 returns a *Hasher with the SHA256 hash.Hash which d to 1000000 iterations. These
// settings can be overridden with the methods with the With prefix.
func NewSHA256() (hasher *Hasher, err error) {
	return New(
		WithVariant(VariantSHA256),
		WithIterations(VariantSHA256.DefaultIterations()),
	)
}

// NewSHA512 returns a *Hasher with the SHA512 hash.Hash which d to 1000000 iterations. These
// settings can be overridden with the methods with the With prefix.
func NewSHA512() (hasher *Hasher, err error) {
	return New(
		WithVariant(VariantSHA512),
		WithIterations(VariantSHA512.DefaultIterations()),
	)
}

// WithOptions defines the options for this scrypt.Hasher.
func (h *Hasher) WithOptions(opts ...Opt) (err error) {
	for _, opt := range opts {
		if err = opt(h); err != nil {
			return err
		}
	}

	return nil
}

// Hash performs the hashing operation and returns either a shacrypt.Digest as a algorithm.Digest or an error.
func (h *Hasher) Hash(password string) (digest algorithm.Digest, err error) {
	h.defaults()

	if digest, err = h.hash(password); err != nil {
		return nil, fmt.Errorf(algorithm.ErrFmtHasherHash, AlgName, err)
	}

	return digest, nil
}

func (h *Hasher) hash(password string) (digest algorithm.Digest, err error) {
	var salt []byte

	if salt, err = random.CharSetBytes(h.bytesSalt, SaltCharSet); err != nil {
		return nil, fmt.Errorf("%w: %v", algorithm.ErrSaltReadRandomBytes, err)
	}

	return h.hashWithSalt(password, salt)
}

// HashWithSalt overloads the Hash method allowing the user to provide a salt. It's recommended instead to configure the
// salt size and let this be a random value generated using crypto/rand.
func (h *Hasher) HashWithSalt(password string, salt []byte) (digest algorithm.Digest, err error) {
	h.defaults()

	if digest, err = h.hashWithSalt(password, salt); err != nil {
		return nil, fmt.Errorf(algorithm.ErrFmtHasherHash, AlgName, err)
	}

	return digest, nil
}

func (h *Hasher) hashWithSalt(password string, salt []byte) (digest algorithm.Digest, err error) {
	if s := len(salt); s > SaltLengthMax || s < SaltLengthMin {
		return nil, fmt.Errorf("%w: salt bytes must have a length of between %d and %d but has a length of %d", algorithm.ErrSaltInvalid, SaltLengthMin, SaltLengthMax, len(salt))
	}

	d := &Digest{
		variant:    h.variant,
		iterations: h.iterations,
		salt:       salt,
	}

	d.defaults()

	d.key = xcrypt.KeySHACrypt(d.variant.HashFunc(), []byte(password), d.salt, d.iterations)

	return d, nil
}

// MustHash overloads the Hash method and panics if the error is not nil. It's recommended if you use this option to
// utilize the Validate method first or handle the panic appropriately.
func (h *Hasher) MustHash(password string) (digest algorithm.Digest) {
	var err error

	if digest, err = h.Hash(password); err != nil {
		panic(err)
	}

	return digest
}

// Validate checks the settings/parameters for this shacrypt.Hasher and returns an error.
func (h *Hasher) Validate() (err error) {
	h.defaults()

	if err = h.validate(); err != nil {
		return fmt.Errorf(algorithm.ErrFmtHasherValidation, AlgName, err)
	}

	return nil
}

func (h *Hasher) validate() (err error) {
	h.defaults()

	return nil
}

func (h *Hasher) defaults() {
	if h.d {
		return
	}

	h.d = true

	if h.bytesSalt < SaltLengthMin {
		h.bytesSalt = algorithm.SaltLengthDefault
	}
}
