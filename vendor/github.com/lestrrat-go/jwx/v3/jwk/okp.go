package jwk

import (
	"bytes"
	"crypto"
	"crypto/ecdh"
	"crypto/ed25519"
	"fmt"
	"reflect"
	"sync"

	"github.com/lestrrat-go/jwx/v3/internal/base64"
	"github.com/lestrrat-go/jwx/v3/jwa"
)

func init() {
	RegisterKeyExporter(KeyKind(jwa.OKP().String()), KeyExportFunc(okpJWKToRaw))
}

func okpKeyKind(crv func() (jwa.EllipticCurveAlgorithm, bool)) KeyKind {
	if c, ok := crv(); ok {
		return KeyKind(jwa.OKP().String() + ":" + c.String())
	}
	return KeyKind(jwa.OKP().String())
}

func (k *okpPublicKey) KeyKind() KeyKind  { return okpKeyKind(k.Crv) }
func (k *okpPrivateKey) KeyKind() KeyKind { return okpKeyKind(k.Crv) }

// Mental note:
//
// Curve25519 refers to a particular curve, and is represented in its Montgomery form.
//
// Ed25519 refers to the biratinally equivalent curve of Curve25519, except it's in Edwards form.
// Ed25519 is the name of the curve and the also the signature scheme using that curve.
// The full name of the scheme is Edwards Curve Digital Signature Algorithm, and thus it is
// also referred to as EdDSA.
//
// X25519 refers to the Diffie-Hellman key exchange protocol that uses Cruve25519.
// Because this is an elliptic curve based Diffie Hellman protocol, it is also referred to
// as ECDH.
//
// OKP keys are used to represent private/public pairs of these elliptic curve
// keys. But note that the name just means Octet Key Pair.

func (k *okpPublicKey) Import(rawKeyIf any) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	var crv jwa.EllipticCurveAlgorithm
	switch rawKey := rawKeyIf.(type) {
	case ed25519.PublicKey:
		crv = jwa.Ed25519()
		if err := validateOKPPublicKeySize(crv, rawKey); err != nil {
			return err
		}
		k.x = rawKey
		k.crv = &crv
	case *ecdh.PublicKey:
		crv = jwa.X25519()
		xbuf := rawKey.Bytes()
		if err := validateOKPPublicKeySize(crv, xbuf); err != nil {
			return err
		}
		k.x = xbuf
		k.crv = &crv
	default:
		muOKPRawKeyImporters.RLock()
		defer muOKPRawKeyImporters.RUnlock()
		for _, fn := range okpRawKeyImporters {
			c, x, _, ok := fn(rawKeyIf)
			if ok {
				k.x = x
				crv = c
				k.crv = &crv
				return nil
			}
		}
		return fmt.Errorf(`unknown key type %T`, rawKeyIf)
	}

	return nil
}

func (k *okpPrivateKey) Import(rawKeyIf any) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	var crv jwa.EllipticCurveAlgorithm
	switch rawKey := rawKeyIf.(type) {
	case ed25519.PrivateKey:
		crv = jwa.Ed25519()
		if len(rawKey) != ed25519.PrivateKeySize {
			return fmt.Errorf(`ed25519: wrong private key size`)
		}
		k.d = rawKey.Seed()
		k.x = rawKey.Public().(ed25519.PublicKey) //nolint:forcetypeassert
		k.crv = &crv
	case *ecdh.PrivateKey:
		crv = jwa.X25519()
		dbuf := rawKey.Bytes()
		if err := validateOKPPrivateKeySize(crv, dbuf); err != nil {
			return err
		}
		xbuf := rawKey.PublicKey().Bytes()
		if err := validateOKPPublicKeySize(crv, xbuf); err != nil {
			return err
		}
		k.d = dbuf
		k.x = xbuf
		k.crv = &crv
	default:
		muOKPRawKeyImporters.RLock()
		defer muOKPRawKeyImporters.RUnlock()
		for _, fn := range okpRawKeyImporters {
			c, x, d, ok := fn(rawKeyIf)
			if ok {
				k.x = x
				k.d = d
				crv = c
				k.crv = &crv
				return nil
			}
		}
		return fmt.Errorf(`unknown key type %T`, rawKeyIf)
	}

	return nil
}

// OKPRawKeyImporter tries to import a raw key as an OKP key.
// Returns the curve, x, d (nil for public), and true if handled.
type OKPRawKeyImporter func(key any) (crv jwa.EllipticCurveAlgorithm, x, d []byte, ok bool)

var muOKPRawKeyImporters sync.RWMutex
var okpRawKeyImporters []OKPRawKeyImporter

// RegisterOKPRawKeyImporter registers a function that can import raw keys as OKP keys.
func RegisterOKPRawKeyImporter(fn OKPRawKeyImporter) {
	muOKPRawKeyImporters.Lock()
	defer muOKPRawKeyImporters.Unlock()
	okpRawKeyImporters = append(okpRawKeyImporters, fn)
}

func validateOKPPublicKeySize(alg jwa.EllipticCurveAlgorithm, xbuf []byte) error {
	switch alg {
	case jwa.Ed25519():
		if len(xbuf) != ed25519.PublicKeySize {
			return fmt.Errorf(`ed25519: wrong public key size`)
		}
	case jwa.X25519():
		if len(xbuf) != 32 {
			return fmt.Errorf(`x25519: wrong public key size`)
		}
	}
	return nil
}

func validateOKPPrivateKeySize(alg jwa.EllipticCurveAlgorithm, dbuf []byte) error {
	switch alg {
	case jwa.Ed25519():
		if len(dbuf) != ed25519.SeedSize {
			return fmt.Errorf(`ed25519: wrong private key size`)
		}
	case jwa.X25519():
		if len(dbuf) != 32 {
			return fmt.Errorf(`x25519: wrong private key size`)
		}
	}
	return nil
}

func buildOKPPublicKey(alg jwa.EllipticCurveAlgorithm, xbuf []byte) (any, error) {
	if err := validateOKPPublicKeySize(alg, xbuf); err != nil {
		return nil, err
	}

	switch alg {
	case jwa.Ed25519():
		return ed25519.PublicKey(xbuf), nil
	case jwa.X25519():
		ret, err := ecdh.X25519().NewPublicKey(xbuf)
		if err != nil {
			return nil, fmt.Errorf(`failed to parse x25519 public key %x (size %d): %w`, xbuf, len(xbuf), err)
		}
		return ret, nil
	default:
		return nil, fmt.Errorf(`invalid curve algorithm %s`, alg)
	}
}

func buildOKPPrivateKey(alg jwa.EllipticCurveAlgorithm, xbuf []byte, dbuf []byte) (any, error) {
	if len(dbuf) == 0 {
		return nil, fmt.Errorf(`cannot use empty seed`)
	}
	if err := validateOKPPublicKeySize(alg, xbuf); err != nil {
		return nil, err
	}
	if err := validateOKPPrivateKeySize(alg, dbuf); err != nil {
		return nil, err
	}

	switch alg {
	case jwa.Ed25519():
		ret := ed25519.NewKeyFromSeed(dbuf)
		//nolint:forcetypeassert
		if !bytes.Equal(xbuf, ret.Public().(ed25519.PublicKey)) {
			return nil, fmt.Errorf(`ed25519: invalid x value given d value`)
		}
		return ret, nil
	case jwa.X25519():
		ret, err := ecdh.X25519().NewPrivateKey(dbuf)
		if err != nil {
			return nil, fmt.Errorf(`x25519: unable to construct x25519 private key from seed: %w`, err)
		}
		return ret, nil
	default:
		return nil, fmt.Errorf(`invalid curve algorithm %s`, alg)
	}
}

var okpConvertibleKeys = []reflect.Type{
	reflect.TypeFor[OKPPrivateKey](),
	reflect.TypeFor[OKPPublicKey](),
}

// This is half baked. I think it will blow up if we used ecdh.* keys and/or x25519 keys
func okpJWKToRaw(key Key, _ any /* this is unused because this is half baked */) (any, error) {
	extracted, err := extractEmbeddedKey(key, okpConvertibleKeys)
	if err != nil {
		return nil, fmt.Errorf(`jwk.OKP: failed to extract embedded key: %w`, err)
	}

	switch key := extracted.(type) {
	case OKPPrivateKey:
		// rlocker is unexported with unexported methods, so only our
		// concrete types implement it. A successful assertion lets us
		// type-assert to the concrete struct and read fields directly
		// under a single batch lock. This avoids nested RLock (which
		// deadlocks when a writer is pending) while preserving an
		// atomic snapshot of all fields.
		var crv jwa.EllipticCurveAlgorithm
		var hasCrv bool
		var x, d []byte
		if locker, ok := key.(rlocker); ok {
			locker.rlock()
			concrete := key.(*okpPrivateKey) //nolint:forcetypeassert // rlocker is unexported; only our concrete types implement it
			if concrete.crv != nil {
				crv = *(concrete.crv)
				hasCrv = true
			}
			x, d = concrete.x, concrete.d
			locker.runlock()
		} else {
			// External implementation — use self-locking interface getters.
			var ok bool
			if crv, ok = key.Crv(); !ok {
				return nil, fmt.Errorf(`missing "crv" field`)
			}
			hasCrv = true
			if x, ok = key.X(); !ok {
				return nil, fmt.Errorf(`missing "x" field`)
			}
			if d, ok = key.D(); !ok {
				return nil, fmt.Errorf(`missing "d" field`)
			}
		}

		if !hasCrv {
			return nil, fmt.Errorf(`missing "crv" field`)
		}
		if x == nil {
			return nil, fmt.Errorf(`missing "x" field`)
		}
		if d == nil {
			return nil, fmt.Errorf(`missing "d" field`)
		}

		privk, err := buildOKPPrivateKey(crv, x, d)
		if err != nil {
			return nil, fmt.Errorf(`jwk.OKPPrivateKey: failed to build private key: %w`, err)
		}
		return privk, nil
	case OKPPublicKey:
		// See OKPPrivateKey case above for explanation of the rlocker pattern.
		var crv jwa.EllipticCurveAlgorithm
		var hasCrv bool
		var x []byte
		if locker, ok := key.(rlocker); ok {
			locker.rlock()
			concrete := key.(*okpPublicKey) //nolint:forcetypeassert // rlocker is unexported; only our concrete types implement it
			if concrete.crv != nil {
				crv = *(concrete.crv)
				hasCrv = true
			}
			x = concrete.x
			locker.runlock()
		} else {
			var ok bool
			if crv, ok = key.Crv(); !ok {
				return nil, fmt.Errorf(`missing "crv" field`)
			}
			hasCrv = true
			if x, ok = key.X(); !ok {
				return nil, fmt.Errorf(`missing "x" field`)
			}
		}

		if !hasCrv {
			return nil, fmt.Errorf(`missing "crv" field`)
		}
		if x == nil {
			return nil, fmt.Errorf(`missing "x" field`)
		}

		pubk, err := buildOKPPublicKey(crv, x)
		if err != nil {
			return nil, fmt.Errorf(`jwk.OKPPublicKey: failed to build public key: %w`, err)
		}
		return pubk, nil
	default:
		return nil, ContinueError()
	}
}

func makeOKPPublicKey(src Key) (Key, error) {
	newKey := newOKPPublicKey()

	// Iterate and copy everything except for the bits that should not be in the public key
	for _, k := range src.Keys() {
		switch k {
		case OKPDKey:
			continue
		default:
			var v any
			if err := src.Get(k, &v); err != nil {
				return nil, fmt.Errorf(`failed to get field %q: %w`, k, err)
			}

			if err := newKey.Set(k, v); err != nil {
				return nil, fmt.Errorf(`failed to set field %q: %w`, k, err)
			}
		}
	}

	return newKey, nil
}

func (k *okpPrivateKey) PublicKey() (Key, error) {
	return makeOKPPublicKey(k)
}

func (k *okpPublicKey) PublicKey() (Key, error) {
	return makeOKPPublicKey(k)
}

func okpThumbprint(hash crypto.Hash, crv, x string) []byte {
	h := hash.New()
	fmt.Fprint(h, `{"crv":"`)
	fmt.Fprint(h, crv)
	fmt.Fprint(h, `","kty":"OKP","x":"`)
	fmt.Fprint(h, x)
	fmt.Fprint(h, `"}`)
	return h.Sum(nil)
}

// Thumbprint returns the JWK thumbprint using the indicated
// hashing algorithm, according to RFC 7638 / 8037
func (k *okpPublicKey) Thumbprint(hash crypto.Hash) ([]byte, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	crv, ok := k.Crv()
	if !ok {
		return nil, fmt.Errorf(`missing "crv" field`)
	}
	return okpThumbprint(
		hash,
		crv.String(),
		base64.EncodeToString(k.x),
	), nil
}

// Thumbprint returns the JWK thumbprint using the indicated
// hashing algorithm, according to RFC 7638 / 8037
func (k *okpPrivateKey) Thumbprint(hash crypto.Hash) ([]byte, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	crv, ok := k.Crv()
	if !ok {
		return nil, fmt.Errorf(`missing "crv" field`)
	}

	return okpThumbprint(
		hash,
		crv.String(),
		base64.EncodeToString(k.x),
	), nil
}

func validateOKPKey(key interface {
	Crv() (jwa.EllipticCurveAlgorithm, bool)
	X() ([]byte, bool)
}) error {
	crv, ok := key.Crv()
	if !ok || crv == jwa.InvalidEllipticCurve() {
		return fmt.Errorf(`invalid curve algorithm`)
	}

	x, ok := key.X()
	if !ok || len(x) == 0 {
		return fmt.Errorf(`missing "x" field`)
	}
	if err := validateOKPPublicKeySize(crv, x); err != nil {
		return err
	}

	if priv, ok := key.(keyWithD); ok {
		d, ok := priv.D()
		if !ok || len(d) == 0 {
			return fmt.Errorf(`missing "d" field`)
		}
		if err := validateOKPPrivateKeySize(crv, d); err != nil {
			return err
		}
	}
	return nil
}

func (k *okpPublicKey) Validate() error {
	k.mu.RLock()
	defer k.mu.RUnlock()
	if err := validateOKPKey(k); err != nil {
		return NewKeyValidationError(fmt.Errorf(`jwk.OKPPublicKey: %w`, err))
	}
	return nil
}

func (k *okpPrivateKey) Validate() error {
	k.mu.RLock()
	defer k.mu.RUnlock()
	if err := validateOKPKey(k); err != nil {
		return NewKeyValidationError(fmt.Errorf(`jwk.OKPPrivateKey: %w`, err))
	}
	return nil
}
