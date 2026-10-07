// Package keyalg works out which signature algorithms a key can be used
// with, and owns the registration tables it reads to decide.
//
// The answer is a guess, on purpose. jws.Verify uses it to pick
// algorithms to try when a JWKS key has no "alg" field, and option
// handling uses it to catch a key that clearly does not go with the
// algorithm asked for. It is not a check for whether a key and an
// algorithm are a valid pair, and the list can be wider than any one RFC
// allows for a given key.
//
// This package is internal to jwx. The jws package still has
// AlgorithmsForKey, a one-line wrapper over [Candidates], but that is
// deprecated and was never meant for callers outside jwx. Everything in
// the tree calls this package instead.
package keyalg

import (
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

// ErrUnclassifiableKey is the common sentinel for [Candidates] failures:
// the key shape cannot be matched to any registered key type for signing.
// Three different code paths land here — Import-failed, kty-not-registered,
// and shape-rejected (e.g. ecdh) — but they're all the same logical "we
// can't classify this key" outcome from the caller's perspective.
// Wrap-with-this lets callers branch on errors.Is instead of
// pattern-matching the three error-message shapes.
//
// The jws package re-exports this through jws.ErrUnclassifiableKey().
var ErrUnclassifiableKey = errors.New("jws: key cannot be classified for signing")

// curver is implemented by jwk.Key types that carry curve information.
type curver interface {
	Crv() (jwa.EllipticCurveAlgorithm, bool)
}

var mu sync.RWMutex
var keyTypeToAlgorithms = make(map[jwa.KeyType][]jwa.SignatureAlgorithm)
var algorithmToKeyTypes = make(map[jwa.SignatureAlgorithm][]jwa.KeyType)
var curveToAlgorithms = make(map[jwa.EllipticCurveAlgorithm][]jwa.SignatureAlgorithm)

func init() {
	RegisterForKeyType(jwa.OKP(), jwa.EdDSA())
	RegisterForCurve(jwa.Ed25519(), jwa.EdDSAEd25519())
	for _, alg := range []jwa.SignatureAlgorithm{jwa.HS256(), jwa.HS384(), jwa.HS512()} {
		RegisterForKeyType(jwa.OctetSeq(), alg)
	}
	for _, alg := range []jwa.SignatureAlgorithm{jwa.RS256(), jwa.RS384(), jwa.RS512(), jwa.PS256(), jwa.PS384(), jwa.PS512()} {
		RegisterForKeyType(jwa.RSA(), alg)
	}
	for _, alg := range []jwa.SignatureAlgorithm{jwa.ES256(), jwa.ES384(), jwa.ES512()} {
		RegisterForKeyType(jwa.EC(), alg)
	}
}

// RegisterForKeyType records alg as usable with keys of type kty.
//
// This backs jws.RegisterAlgorithmForKeyType, which extension modules
// call from init() to add their own algorithms.
func RegisterForKeyType(kty jwa.KeyType, alg jwa.SignatureAlgorithm) {
	mu.Lock()
	defer mu.Unlock()
	keyTypeToAlgorithms[kty] = append(keyTypeToAlgorithms[kty], alg)
	if !slices.Contains(algorithmToKeyTypes[alg], kty) {
		algorithmToKeyTypes[alg] = append(algorithmToKeyTypes[alg], kty)
	}
}

// RegisterForCurve scopes alg to the given elliptic curve. When
// [Candidates] can determine a key's curve, an algorithm registered under
// some curve is offered only for keys on that curve, instead of for every
// key of its key type.
//
// This backs jws.RegisterAlgorithmForCurve. It is append-only and
// deduplicates entries, so builtin registrations cannot be overwritten by
// external modules.
func RegisterForCurve(crv jwa.EllipticCurveAlgorithm, alg jwa.SignatureAlgorithm) {
	mu.Lock()
	defer mu.Unlock()
	if slices.Contains(curveToAlgorithms[crv], alg) {
		return
	}
	curveToAlgorithms[crv] = append(curveToAlgorithms[crv], alg)
}

// KeyTypesFor returns the key types registered for alg. The reverse index
// is maintained at registration time so this is an O(1) lookup. It returns
// nil if no key type is registered for alg, which signals callers to skip
// any prefilter.
func KeyTypesFor(alg jwa.SignatureAlgorithm) []jwa.KeyType {
	mu.RLock()
	defer mu.RUnlock()
	// Copy so the caller can safely iterate without holding the lock;
	// RegisterForKeyType may append concurrently after we return.
	// Typical length is 1.
	return slices.Clone(algorithmToKeyTypes[alg])
}

// Candidates returns the signature algorithms that key could be used
// with. It only takes into consideration keys/algorithms for verification
// purposes, as this is the only usage where one may need to dynamically
// figure out which method to use.
//
// When the key's curve is known, algorithms registered for that curve via
// [RegisterForCurve] are combined with key-type-level algorithms to
// produce a more precise result. The curve is known for a [jwk.Key] that
// has a Crv() method, for raw ed25519 keys, and for any raw key that
// reaches the [jwk.Import] fallback below.
//
// ECDSA is the exception. A raw [ecdsa.PublicKey] or [ecdsa.PrivateKey] is
// classified by key type alone and its Curve field is never read. No
// builtin registration binds P-256, P-384, or P-521 to an algorithm
// either, so every EC key reports the full ES* list no matter which curve
// it sits on. RFC 7518 Section 3.4 is stricter than that; jws.Sign
// enforces it only when the caller passes jws.WithStrictECDSA(true).
//
// Accepted key shapes (resolved in order):
//
//  1. [jwk.Key] — kty is read directly; if the implementation also exposes
//     Crv(), the curve refines the result.
//  2. Stdlib crypto types: [rsa.PublicKey] / [rsa.PrivateKey] (and pointer
//     forms), [ecdsa.PublicKey] / [ecdsa.PrivateKey] (and pointer forms),
//     [ed25519.PublicKey], [ed25519.PrivateKey], and [byte] slices for
//     symmetric keys.
//  3. [crypto/ecdh.PublicKey] / [crypto/ecdh.PrivateKey] (and pointer
//     forms) — explicitly rejected; ECDH keys are key-agreement only.
//     Returns an error wrapping [ErrUnclassifiableKey].
//  4. [crypto.Signer] (e.g. KMS-backed adapters) — resolved once via
//     .Public(); the public key is then re-classified through tiers 1–2
//     or the [jwk.Import] fallback below. To prevent infinite recursion,
//     a Signer whose .Public() is itself a Signer is left for the
//     downstream dispatcher to handle.
//  5. [jwk.Import] fallback — anything else is offered to the import
//     registry, allowing extension modules to register their own raw key
//     types.
//
// All "we cannot classify this key" failures wrap [ErrUnclassifiableKey],
// so callers can branch with errors.Is rather than pattern-matching error
// strings. The wrapping error keeps the concrete %T or %q diagnostic in
// its message for human readers.
func Candidates(key any) ([]jwa.SignatureAlgorithm, error) {
	var kty jwa.KeyType
	var crv jwa.EllipticCurveAlgorithm
	var hasCrv bool

	switch key := key.(type) {
	case jwk.Key:
		kty = key.KeyType()
		if ck, ok := key.(curver); ok {
			crv, hasCrv = ck.Crv()
		}
	case rsa.PublicKey, *rsa.PublicKey, rsa.PrivateKey, *rsa.PrivateKey:
		kty = jwa.RSA()
	case ecdsa.PublicKey, *ecdsa.PublicKey, ecdsa.PrivateKey, *ecdsa.PrivateKey:
		kty = jwa.EC()
	case ed25519.PublicKey, ed25519.PrivateKey:
		kty = jwa.OKP()
		crv = jwa.Ed25519()
		hasCrv = true
	case *ecdh.PublicKey, ecdh.PublicKey, *ecdh.PrivateKey, ecdh.PrivateKey:
		// ecdh keys are for key agreement (X25519/X448), not signing.
		// Reject at the API boundary instead of returning a misleading
		// algorithm list that would fail deeper in the signing stack.
		return nil, fmt.Errorf(`%w: key type %T cannot be used for signing (ecdh keys are key-agreement only)`, ErrUnclassifiableKey, key)
	case []byte:
		kty = jwa.OctetSeq()
	default:
		// For crypto.Signer from external packages (e.g. KMS-backed signers),
		// extract the underlying public key type via .Public().
		// Standard library types (*rsa.PrivateKey, etc.) are already handled
		// by the concrete cases above.
		var signerPubErr error
		if signer, ok := key.(crypto.Signer); ok {
			pub := signer.Public()
			// Guard: only recurse if the public key is not itself a crypto.Signer,
			// to prevent infinite recursion from pathological implementations.
			if _, isSigner := pub.(crypto.Signer); !isSigner {
				algs, err := Candidates(pub)
				if err == nil {
					return algs, nil
				}
				// Save the inner classification error so a
				// downstream Import-fallback failure can surface
				// both diagnostics. A successful Import discards
				// signerPubErr — only the eventual failure path
				// joins them.
				signerPubErr = err
			}
		}
		imported, err := jwk.Import(key)
		if err != nil {
			outer := fmt.Errorf(`%w: unknown key type %T`, ErrUnclassifiableKey, key)
			if signerPubErr != nil {
				return nil, errors.Join(outer, signerPubErr)
			}
			return nil, outer
		}
		kty = imported.KeyType()
		if ck, ok := imported.(curver); ok {
			crv, hasCrv = ck.Crv()
		}
	}

	mu.RLock()
	defer mu.RUnlock()

	ktyAlgs, ok := keyTypeToAlgorithms[kty]
	if !ok {
		return nil, fmt.Errorf(`%w: unregistered key type %q`, ErrUnclassifiableKey, kty)
	}

	// If we know the curve and there are curve-specific registrations,
	// return only key-type-level algorithms (those not registered under
	// any curve) plus curve-specific algorithms for this curve.
	if hasCrv {
		crvAlgs := curveToAlgorithms[crv]
		return filterForCurve(ktyAlgs, crvAlgs), nil
	}

	return ktyAlgs, nil
}

// filterForCurve returns the subset of ktyAlgs that are not registered
// under any curve (i.e., generic for the key type) plus the curve-specific
// algorithms from crvAlgs.
func filterForCurve(ktyAlgs, crvAlgs []jwa.SignatureAlgorithm) []jwa.SignatureAlgorithm {
	var result []jwa.SignatureAlgorithm

	// Add key-type-level algorithms that are not claimed by any curve
	for _, alg := range ktyAlgs {
		if !isRegisteredUnderAnyCurve(alg) {
			result = append(result, alg)
		}
	}

	// Add curve-specific algorithms
	result = append(result, crvAlgs...)
	return result
}

func isRegisteredUnderAnyCurve(alg jwa.SignatureAlgorithm) bool {
	for _, algs := range curveToAlgorithms {
		if slices.Contains(algs, alg) {
			return true
		}
	}
	return false
}
