package jwk

import (
	"bytes"
	"crypto"
	"errors"
	"fmt"

	"github.com/lestrrat-go/blackmagic"
	"github.com/lestrrat-go/jwx/v3/cert"
	"github.com/lestrrat-go/jwx/v3/internal/json"
	"github.com/lestrrat-go/jwx/v3/jwa"
)

// UnsupportedKey is a placeholder for a JWK Set entry that could not be
// parsed into a usable key. Per RFC 7517 §5, an entry inside a "keys"
// array whose key type is not understood, that is missing required
// members, or whose values are out of the supported range may be retained
// as an UnsupportedKey instead of failing the whole set.
//
// In v3 retention is opt-in: pass `jwk.WithStrictKeySetParsing(false)` to
// `jwk.Parse` (or set it globally via `jwk.Configure`). By default v3
// still fails the whole set on the first unparseable entry, so existing
// callers see no change. (In v4 retention is the default; the option
// carries the same meaning in both, only the default differs.)
//
// A placeholder preserves the entry's original JSON — marshaling it with
// json.Marshal (alone or as part of its set) reproduces the entry, so a
// set containing one round-trips losslessly — and it preserves the error
// that prevented parsing (via [UnsupportedKey.Reason]).
//
// An UnsupportedKey cannot be used for any cryptographic operation:
// [UnsupportedKey.Thumbprint], [UnsupportedKey.PublicKey] and
// [UnsupportedKey.Validate] all return an error wrapping Reason(), and
// the key is rejected by cryptographic consumers such as
// jws.Verify / jwe.Decrypt with a descriptive per-key error.
//
// Use [IsUnsupportedKey] to check whether a key is a placeholder. Use a
// type assertion when you also need the placeholder's details:
//
//	if uk, ok := key.(jwk.UnsupportedKey); ok {
//	    // key type key.KeyType() is not supported by this build;
//	    // uk.Reason() explains why, an extension module may be required.
//	}
type UnsupportedKey interface {
	Key

	// Reason returns the error that prevented the entry from parsing.
	Reason() error

	// isUnsupportedKey seals this interface: only the placeholder type
	// produced by this package implements it. Without the seal, any
	// third-party Key that happens to define a Reason() error method
	// would satisfy UnsupportedKey and be rejected as a placeholder by
	// jwk.Export, jwk.AssignKeyID, and jws/jwe key selection.
	isUnsupportedKey()
}

// IsUnsupportedKey reports whether key is a placeholder retained for a
// JWK Set entry that could not be parsed. Only placeholders produced by
// this package satisfy the check; a user-defined Key type can never be
// mistaken for one. It is the sanctioned way to
// skip placeholders when iterating a set; type-assert to
// [UnsupportedKey] when you also need Reason().
func IsUnsupportedKey(key Key) bool {
	_, ok := key.(UnsupportedKey)
	return ok
}

// unsupportedKey is the concrete implementation of [UnsupportedKey].
//
// It is effectively immutable after construction: the mutators [Set] and
// [Remove] return errors without modifying any field, so no locking is
// required for concurrent reads. The best-effort common members are
// parsed once in [newUnsupportedKey].
type unsupportedKey struct {
	raw    []byte
	reason error

	rawKty     string
	ktyPresent bool
	algorithm  *jwa.KeyAlgorithm
	keyID      *string
}

var _ UnsupportedKey = &unsupportedKey{}
var _ Key = &unsupportedKey{}

// newUnsupportedKey builds a placeholder from the verbatim entry bytes
// and the error that prevented parsing. raw is cloned because the
// decoder buffer it came from may be reused.
func newUnsupportedKey(raw []byte, reason error) *unsupportedKey {
	// reason is always non-nil in practice (a placeholder only exists
	// because ParseKey failed), but guard anyway so a nil can never
	// reach the %w verbs that wrap Reason().
	if reason == nil {
		reason = errors.New(`unspecified parse error`)
	}
	k := &unsupportedKey{
		raw:    bytes.Clone(raw),
		reason: reason,
	}
	k.parseBestEffort()
	return k
}

// parseBestEffort re-parses the minimum set of members needed to make
// the placeholder discoverable and nameable: "kid" (LookupKeyID and the
// duplicate-kid check), "kty" (error messages, KeyType()), and "alg"
// (error messages). A member that fails to parse is simply left absent.
// Everything else stays unparsed — the raw JSON is the entry's
// authoritative representation.
func (k *unsupportedKey) parseBestEffort() {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(k.raw, &fields); err != nil {
		return
	}

	if raw, ok := fields[KeyTypeKey]; ok {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			k.rawKty = s
			k.ktyPresent = true
		}
	}
	if raw, ok := fields[KeyIDKey]; ok {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			k.keyID = &s
		}
	}
	if raw, ok := fields[AlgorithmKey]; ok {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			if alg, err := jwa.KeyAlgorithmFrom(s); err == nil {
				k.algorithm = &alg
			}
		}
	}
}

func (k *unsupportedKey) Reason() error {
	return k.reason
}

// isUnsupportedKey implements the [UnsupportedKey] interface seal.
func (k *unsupportedKey) isUnsupportedKey() {}

// unsupportederr wraps the placeholder's Reason() in an error explaining
// that the operation cannot be performed on an unsupported key.
func (k *unsupportedKey) unsupportederr(op string) error {
	kid := ""
	if k.keyID != nil {
		kid = *k.keyID
	}
	return fmt.Errorf(`jwk: cannot %s an unsupported key (kty=%q, kid=%q): the entry could not be parsed: %w`, op, k.rawKty, kid, k.reason)
}

func (k *unsupportedKey) KeyType() jwa.KeyType {
	if !k.ktyPresent {
		return jwa.EmptyKeyType()
	}
	return jwa.NewKeyType(k.rawKty)
}

func (k *unsupportedKey) Algorithm() (jwa.KeyAlgorithm, bool) {
	if k.algorithm != nil {
		return *k.algorithm, true
	}
	return nil, false
}

func (k *unsupportedKey) KeyID() (string, bool) {
	if k.keyID != nil {
		return *k.keyID, true
	}
	return "", false
}

// The remaining standard members are not mirrored: nothing consumes them
// on a placeholder (key selection rejects it before ever checking usage),
// and the raw JSON already carries them for round-tripping.

func (k *unsupportedKey) KeyOps() (KeyOperationList, bool) {
	return nil, false
}

func (k *unsupportedKey) KeyUsage() (string, bool) {
	return "", false
}

func (k *unsupportedKey) X509CertChain() (*cert.Chain, bool) {
	return nil, false
}

func (k *unsupportedKey) X509CertThumbprint() (string, bool) {
	return "", false
}

func (k *unsupportedKey) X509CertThumbprintS256() (string, bool) {
	return "", false
}

func (k *unsupportedKey) X509URL() (string, bool) {
	return "", false
}

func (k *unsupportedKey) Has(name string) bool {
	switch name {
	case KeyTypeKey:
		return k.ktyPresent
	case AlgorithmKey:
		return k.algorithm != nil
	case KeyIDKey:
		return k.keyID != nil
	default:
		return false
	}
}

// Get retrieves the best-effort common members (kty, alg, kid) into dst.
// Any other field is reported as absent — the raw JSON remains its only
// representation.
func (k *unsupportedKey) Get(name string, dst any) error {
	switch name {
	case KeyTypeKey:
		if !k.ktyPresent {
			return fmt.Errorf(`field %q not found`, name)
		}
		return blackmagic.AssignIfCompatible(dst, k.KeyType())
	case AlgorithmKey:
		if k.algorithm == nil {
			return fmt.Errorf(`field %q not found`, name)
		}
		return blackmagic.AssignIfCompatible(dst, *k.algorithm)
	case KeyIDKey:
		if k.keyID == nil {
			return fmt.Errorf(`field %q not found`, name)
		}
		return blackmagic.AssignIfCompatible(dst, *k.keyID)
	default:
		return fmt.Errorf(`field %q not found`, name)
	}
}

func (k *unsupportedKey) Keys() []string {
	keys := make([]string, 0, 3)
	if k.ktyPresent {
		keys = append(keys, KeyTypeKey)
	}
	if k.algorithm != nil {
		keys = append(keys, AlgorithmKey)
	}
	if k.keyID != nil {
		keys = append(keys, KeyIDKey)
	}
	return keys
}

// Set always returns an error: the verbatim raw JSON is the single
// source of truth for serialization, so mutation is not allowed (it
// would make the marshaled form diverge from the accessor view).
func (k *unsupportedKey) Set(string, any) error {
	return k.unsupportederr("modify")
}

// Remove always returns an error, for the same reason as [Set].
func (k *unsupportedKey) Remove(string) error {
	return k.unsupportederr("modify")
}

// Validate reports the retained parse error: a placeholder is by
// definition not a valid key. The error is wrapped in a key validation
// error so it classifies like every other built-in Key.Validate failure
// (jwk.IsKeyValidationError is true), while Reason() stays reachable
// through the wrapping chain.
func (k *unsupportedKey) Validate() error {
	return NewKeyValidationError(k.unsupportederr("validate"))
}

// Thumbprint always returns an error: RFC 7638 thumbprints require the
// per-kty required members, which are not understood for a placeholder.
func (k *unsupportedKey) Thumbprint(crypto.Hash) ([]byte, error) {
	return nil, k.unsupportederr("compute the thumbprint of")
}

// PublicKey always returns an error: whether the entry contains private
// material is unknowable, so no public projection can be derived safely.
func (k *unsupportedKey) PublicKey() (Key, error) {
	return nil, k.unsupportederr("derive the public key of")
}

// Clone returns an independent copy of the placeholder. Placeholders are
// first-class set members, so they clone like any other key.
func (k *unsupportedKey) Clone() (Key, error) {
	dst := &unsupportedKey{
		raw:        bytes.Clone(k.raw),
		reason:     k.reason,
		rawKty:     k.rawKty,
		ktyPresent: k.ktyPresent,
	}
	if k.algorithm != nil {
		tmp := *k.algorithm
		dst.algorithm = &tmp
	}
	if k.keyID != nil {
		tmp := *k.keyID
		dst.keyID = &tmp
	}
	return dst, nil
}

// MarshalJSON emits the verbatim raw JSON of the original entry. This is
// the round-trip guarantee: a set containing a placeholder re-serializes
// the unknown entry unchanged.
func (k *unsupportedKey) MarshalJSON() ([]byte, error) {
	return bytes.Clone(k.raw), nil
}
