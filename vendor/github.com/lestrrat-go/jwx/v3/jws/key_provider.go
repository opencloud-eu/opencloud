package jws

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"sync"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

// KeyProvider is responsible for providing key(s) to sign or verify a payload.
// Multiple `jws.KeyProvider`s can be passed to `jws.Verify()` or `jws.Sign()`
//
// `jws.Sign()` can only accept static key providers via `jws.WithKey()`,
// while `jws.Verify()` can accept `jws.WithKey()`, `jws.WithKeySet()`,
// `jws.WithVerifyAuto()`, and `jws.WithKeyProvider()`.
//
// Understanding how this works is crucial to learn how this package works.
//
// `jws.Sign()` is straightforward: signatures are created for each
// provided key.
//
// `jws.Verify()` is a bit more involved, because there are cases you
// will want to compute/deduce/guess the keys that you would like to
// use for verification.
//
// The first thing that `jws.Verify()` does is to collect the
// KeyProviders from the option list that the user provided (presented in pseudocode):
//
//	keyProviders := filterKeyProviders(options)
//
// Then, remember that a JWS message may contain multiple signatures in the
// message. For each signature, we call on the KeyProviders to give us
// the key(s) to use on this signature:
//
//	for sig in msg.Signatures {
//	  for kp in keyProviders {
//	    kp.FetchKeys(ctx, sink, sig, msg)
//	    ...
//	  }
//	}
//
// The `sink` argument passed to the KeyProvider is a temporary storage
// for the keys (either a jwk.Key or a "raw" key). The `KeyProvider`
// is responsible for sending keys into the `sink`.
//
// When called, the `KeyProvider` created by `jws.WithKey()` sends the same key,
// `jws.WithKeySet()` sends keys that matches a particular `kid` and `alg`,
// `jws.WithVerifyAuto()` fetches a JWK from the `jku` URL,
// and finally `jws.WithKeyProvider()` allows you to execute arbitrary
// logic to provide keys. If you are providing a custom `KeyProvider`,
// you should execute the necessary checks or retrieval of keys, and
// then send the key(s) to the sink:
//
//	sink.Key(alg, key)
//
// These keys are then retrieved and tried for each signature, until
// a match is found:
//
//	keys := sink.Keys()
//	for key in keys {
//	  if givenSignature == makeSignature(key, payload, ...)) {
//	    return OK
//	  }
//	}
type KeyProvider interface {
	FetchKeys(context.Context, KeySink, *Signature, *Message) error
}

// KeySink is a data storage where `jws.KeyProvider` objects should
// send their keys to.
type KeySink interface {
	Key(jwa.SignatureAlgorithm, any)
}

type algKeyPair struct {
	alg jwa.SignatureAlgorithm
	key any
}

type algKeySink struct {
	mu   sync.Mutex
	list []algKeyPair
}

func (s *algKeySink) Key(alg jwa.SignatureAlgorithm, key any) {
	s.mu.Lock()
	s.list = append(s.list, algKeyPair{alg, key})
	s.mu.Unlock()
}

type staticKeyProvider struct {
	alg jwa.SignatureAlgorithm
	key any
}

func (kp *staticKeyProvider) FetchKeys(_ context.Context, sink KeySink, _ *Signature, _ *Message) error {
	sink.Key(kp.alg, kp.key)
	return nil
}

type keySetProvider struct {
	set                  jwk.Set
	requireKid           bool // true if `kid` must be specified
	useDefault           bool // true if the first key should be used iff there's exactly one key in set
	inferAlgorithm       bool // true if the algorithm should be inferred from key type
	multipleKeysPerKeyID bool // true if we should attempt to match multiple keys per key ID. if false we assume that only one key exists for a given key ID
}

// selectKey examines a single key and, if it is suitable for the given
// signature, adds one or more (algorithm, key) pairs to the sink.
// It returns true if at least one pair was added, false if the key was
// filtered out (e.g. wrong usage, no matching algorithm).
func (kp *keySetProvider) selectKey(sink KeySink, key jwk.Key, sig *Signature, _ *Message) (bool, error) {
	if uk, ok := key.(jwk.UnsupportedKey); ok {
		return false, unsupportedKeyError(uk, `signature verification`)
	}

	if usage, ok := key.KeyUsage(); ok {
		// it's okay if use: "". we'll assume it's "sig"
		if usage != "" && usage != jwk.ForSignature.String() {
			kid, _ := key.KeyID()
			return false, fmt.Errorf(`key with kid %q is marked use=%q, not usable for signature verification (expected %q)`, kid, usage, jwk.ForSignature.String())
		}
	}

	if v, ok := key.Algorithm(); ok {
		salg, ok := jwa.LookupSignatureAlgorithm(v.String())
		if !ok {
			return false, fmt.Errorf(`invalid signature algorithm %q`, v)
		}

		sink.Key(salg, key)
		return true, nil
	}

	if !kp.inferAlgorithm {
		return false, nil
	}

	algs, err := AlgorithmsForKey(key)
	if err != nil {
		return false, fmt.Errorf(`failed to get a list of signature methods for key type %s: %w`, key.KeyType(), err)
	}

	// bail out if the JWT has a `alg` field, and it doesn't match
	if tokAlg, ok := sig.ProtectedHeaders().Algorithm(); ok {
		for _, alg := range algs {
			if tokAlg == alg {
				sink.Key(alg, key)
				return true, nil
			}
		}
		return false, fmt.Errorf(`algorithm in the message does not match any of the inferred algorithms`)
	}

	// Yes, you get to try them all!!!!!!!
	for _, alg := range algs {
		sink.Key(alg, key)
	}
	return len(algs) > 0, nil
}

func (kp *keySetProvider) FetchKeys(_ context.Context, sink KeySink, sig *Signature, msg *Message) error {
	if kp.requireKid {
		wantedKid, ok := sig.ProtectedHeaders().KeyID()
		if !ok {
			return kp.fetchDefaultKey(sink, sig, msg)
		}
		return kp.fetchKeysByKid(sink, sig, msg, wantedKid)
	}
	return kp.fetchAllKeys(sink, sig, msg)
}

// fetchDefaultKey handles the case where kid is required but the token
// has no kid field. It uses the sole key in the set when useDefault is true.
func (kp *keySetProvider) fetchDefaultKey(sink KeySink, sig *Signature, msg *Message) error {
	if !kp.useDefault {
		return fmt.Errorf(`failed to find matching key: no key ID ("kid") specified in token`)
	}
	if kp.set.Len() > 1 {
		return fmt.Errorf(`failed to find matching key: no key ID ("kid") specified in token but multiple keys available in key set`)
	}

	key, ok := kp.set.Key(0)
	if !ok {
		return fmt.Errorf(`failed to get key at index 0 (empty JWKS?)`)
	}
	_, err := kp.selectKey(sink, key, sig, msg)
	return err
}

// fetchKeysByKid looks up keys by their key ID and adds matching ones to the sink.
func (kp *keySetProvider) fetchKeysByKid(sink KeySink, sig *Signature, msg *Message, wantedKid string) error {
	// <= v2.0.3 backwards compatible case: only match a single key
	// whose key ID matches `wantedKid`
	if !kp.multipleKeysPerKeyID {
		key, ok := kp.set.LookupKeyID(wantedKid)
		if !ok {
			return fmt.Errorf(`failed to find key with key ID %q in key set`, wantedKid)
		}
		_, err := kp.selectKey(sink, key, sig, msg)
		return err
	}

	// multipleKeysPerKeyID: attempt all keys whose key ID matches
	found := false
	var errs []error
	for i := range kp.set.Len() {
		key, _ := kp.set.Key(i)
		if kid, ok := key.KeyID(); !ok || kid != wantedKid {
			continue
		}

		added, err := kp.selectKey(sink, key, sig, msg)
		if err != nil {
			errs = append(errs, fmt.Errorf(`key #%d: %w`, i, err))
			continue
		}
		if added {
			found = true
		}
	}
	if !found {
		if len(errs) > 0 {
			return fmt.Errorf(`failed to select any key with key ID %q: %w`, wantedKid, errors.Join(errs...))
		}
		return fmt.Errorf(`failed to find key with key ID %q in key set`, wantedKid)
	}
	return nil
}

// fetchAllKeys iterates all keys in the set and adds suitable ones to the sink.
//
// When the protected header advertises an `alg`, keys whose type cannot
// produce that algorithm are skipped before reaching selectKey
// (unsupported-key placeholders excepted — see the comment at the check
// below). This bounds verification fan-out to N_keys_of_matching_type
// instead of N_keys when `WithRequireKid(false)` is used against a
// heterogeneous JWKS. The skip is semantics-preserving:
// validateAlgorithmForKey in verify_context would reject the
// incompatible (alg, key) pair before running any verifier anyway.
//
// The allowed-KeyType set is looked up once per FetchKeys call via the
// precomputed algorithmToKeyTypes inverse map, so the per-key check is
// a cheap KeyType equality over a tiny slice (typically 1 element).
// When allowedKtys is nil (no header alg, or alg has no registered
// key type), the filter is skipped.
func (kp *keySetProvider) fetchAllKeys(sink KeySink, sig *Signature, msg *Message) error {
	var allowedKtys []jwa.KeyType
	if hdrAlg, ok := sig.ProtectedHeaders().Algorithm(); ok {
		allowedKtys = keyTypesForAlgorithm(hdrAlg)
	}
	found := false
	var errs []error
	for i := range kp.set.Len() {
		key, ok := kp.set.Key(i)
		if !ok {
			return fmt.Errorf(`failed to get key at index %d`, i)
		}
		// Unsupported-key placeholders are exempt from the prefilter:
		// their raw kty is never a registered KeyType, so the filter
		// would silently skip them and the caller would only see a
		// generic "no keys worked" error. Letting them reach selectKey
		// records the per-key rejection (kid, kty, retained parse
		// reason) in errs instead.
		if allowedKtys != nil && !slices.Contains(allowedKtys, key.KeyType()) && !jwk.IsUnsupportedKey(key) {
			continue
		}
		added, err := kp.selectKey(sink, key, sig, msg)
		if err != nil {
			errs = append(errs, fmt.Errorf(`key #%d: %w`, i, err))
			continue
		}
		if added {
			found = true
		}
	}
	// Only when no candidate reached the sink do the collected per-key
	// errors become the outcome: a key that was skipped without error
	// (e.g. no "alg" member and inference disabled) must not mask the
	// named rejections of the keys that did fail.
	if !found && len(errs) > 0 {
		return fmt.Errorf(`no key in the key set was usable: %w`, errors.Join(errs...))
	}
	return nil
}

// keyTypesForAlgorithm returns the registered key types that can
// produce the given signature algorithm. The inverse map is maintained
// at registration time so this is an O(1) lookup. Returns nil if no
// key type is registered for alg, which signals callers to skip the
// prefilter.
func keyTypesForAlgorithm(alg jwa.SignatureAlgorithm) []jwa.KeyType {
	muAlgorithmMaps.RLock()
	defer muAlgorithmMaps.RUnlock()
	// Copy so the caller can safely iterate without holding the
	// lock; RegisterAlgorithmForKeyType may append concurrently
	// after we return. Typical length is 1.
	return slices.Clone(algorithmToKeyTypes[alg])
}

type jkuProvider struct {
	fetcher jwk.Fetcher
	options []jwk.FetchOption
}

func (kp jkuProvider) FetchKeys(ctx context.Context, sink KeySink, sig *Signature, _ *Message) error {
	if kp.fetcher == nil {
		kp.fetcher = jwk.FetchFunc(jwk.Fetch)
	}

	kid, ok := sig.ProtectedHeaders().KeyID()
	if !ok {
		return fmt.Errorf(`use of "jku" requires that the payload contain a "kid" field in the protected header`)
	}

	// errors here can't be reliably passed to the consumers.
	// it's unfortunate, but if you need this control, you are
	// going to have to write your own fetcher
	u, ok := sig.ProtectedHeaders().JWKSetURL()
	if !ok || u == "" {
		return fmt.Errorf(`use of "jku" field specified, but the field is empty`)
	}
	uo, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf(`failed to parse "jku": %w`, err)
	}
	if uo.Scheme != "https" {
		return fmt.Errorf(`url in "jku" must be HTTPS`)
	}

	set, err := kp.fetcher.Fetch(ctx, u, kp.options...)
	if err != nil {
		return fmt.Errorf(`failed to fetch %q: %w`, u, err)
	}

	key, ok := set.LookupKeyID(kid)
	if !ok {
		return fmt.Errorf(`jku: key with "kid" %q not found in JWKS fetched from %q`, kid, u)
	}

	if uk, ok := key.(jwk.UnsupportedKey); ok {
		return fmt.Errorf(`jku: key with "kid" %q from %q has unsupported key type %q and cannot be used for signature verification; an extension module may be required to parse it: %w`, kid, u, uk.KeyType().String(), uk.Reason())
	}

	if usage, ok := key.KeyUsage(); ok {
		if usage != "" && usage != jwk.ForSignature.String() {
			return fmt.Errorf(`key with kid %q is marked use=%q, not usable for signature verification (expected %q)`, kid, usage, jwk.ForSignature.String())
		}
	}

	algs, err := AlgorithmsForKey(key)
	if err != nil {
		return fmt.Errorf(`failed to get a list of signature methods for key type %s: %w`, key.KeyType(), err)
	}

	hdrAlg, ok := sig.ProtectedHeaders().Algorithm()
	if !ok {
		// The jku provider routes a key by matching both "kid" and
		// "alg" against the JWS protected header. With no alg in the
		// header there's nothing to pin the signature algorithm to,
		// so reject explicitly rather than returning no keys and
		// letting the outer verify loop surface a generic "could not
		// be verified with any of the keys" message.
		return fmt.Errorf(`use of "jku" requires that the protected header contain an "alg" field`)
	}

	for _, alg := range algs {
		if hdrAlg != alg {
			continue
		}

		sink.Key(alg, key)
		return nil
	}

	return fmt.Errorf(`algorithm %q in JWS header does not match any algorithm for key type %s from jku`, hdrAlg, key.KeyType())
}

// KeyProviderFunc is a type of KeyProvider that is implemented by
// a single function. You can use this to create ad-hoc `KeyProvider`
// instances.
type KeyProviderFunc func(context.Context, KeySink, *Signature, *Message) error

func (kp KeyProviderFunc) FetchKeys(ctx context.Context, sink KeySink, sig *Signature, msg *Message) error {
	return kp(ctx, sink, sig, msg)
}
