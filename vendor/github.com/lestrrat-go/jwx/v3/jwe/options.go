package jwe

import (
	"bytes"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/option/v2"
)

type identCritExtension struct{}
type identDisabledKeyAlgorithms struct{}

// WithDisabledKeyAlgorithms returns a process-global option for jwe.Settings()
// that refuses the named key encryption algorithms in both directions. After
// the call returns, jwe.Encrypt() will not produce a recipient using any
// listed algorithm, and jwe.Decrypt() will reject any recipient whose "alg"
// is in the list, before any cryptographic work runs. The check fires per
// recipient: a multi-recipient JWE is rejected as soon as a disabled "alg"
// is seen on any recipient.
//
// The list is replaced (not unioned) on each Settings() call. To clear the
// disabled set, call jwe.Settings(jwe.WithDisabledKeyAlgorithms()) with no
// arguments.
//
// This is a deployment-time policy hook for the canonical "disable RSA1_5"
// case (RFC 8725 §3.1) and similar legacy-algorithm bans. The jwa package
// does not unregister these algorithms — keeping them registered preserves
// header parsing for diagnostic logs, while this option blocks any actual
// crypto use.
func WithDisabledKeyAlgorithms(algorithms ...jwa.KeyEncryptionAlgorithm) GlobalOption {
	return &globalOption{option.New(identDisabledKeyAlgorithms{}, algorithms)}
}

// WithCritExtension declares that the caller understands and will process
// the named "crit" (Critical) header parameter extension(s) per RFC 7516
// Section 4.1.13 (which references RFC 7515 Section 4.1.11). The option
// is variadic and accumulating: a single call may register any number
// of extension names, and the option may be passed multiple times to add
// more.
//
// This option only takes effect when jwe.WithCritValidation(true) is
// also passed. With validation enabled, jwe.Decrypt() rejects any JWE
// whose protected header lists a "crit" extension that has not been
// declared via this option, satisfying the RFC's requirement that
// recipients MUST reject any "crit" extension they do not understand.
//
// IMPORTANT: declaring an extension here is a promise to the library
// that the caller knows what the extension means and will perform any
// validation, side effect, or policy enforcement the extension requires
// AFTER jwe.Decrypt() returns successfully. The library cannot inspect
// or enforce the semantics of an extension; it only checks that every
// "crit" entry in the message has been declared. If you register an
// extension and then forget to act on its value, you have effectively
// disabled the protection the producer was trying to obtain by listing
// the extension as critical.
//
// Concretely, the post-decrypt code path for a declared extension must:
//
//  1. Read the value of the named header from the decrypted message.
//  2. Apply whatever check or transformation the extension specifies
//     (e.g. for an "x-tenant-binding" extension, refuse to act on the
//     payload unless the binding matches the current tenant).
//  3. Treat any failure of that check as a decryption failure for
//     the application's purposes, even though jwe.Decrypt() returned
//     no error.
func WithCritExtension(names ...string) DecryptOption {
	return &decryptOption{option.New(identCritExtension{}, names)}
}

// WithProtectedHeaders is used to specify contents of the protected header.
// Some fields such as "enc" and "zip" will be overwritten when encryption is
// performed.
//
// There is no equivalent for unprotected headers in this implementation
func WithProtectedHeaders(h Headers) EncryptOption {
	cloned, _ := h.Clone()
	return &encryptOption{option.New(identProtectedHeaders{}, cloned)}
}

// WithAuthenticateData specifies the external Additional Authenticated Data
// to use when encrypting a JSON JWE.
//
// The data is copied before it is stored in the option. External Additional
// Authenticated Data is not supported by compact serialization; pass
// WithJSON() to select JSON serialization.
func WithAuthenticateData(aad []byte) EncryptOption {
	return &encryptOption{option.New(identAuthenticateData{}, bytes.Clone(aad))}
}

type withKey struct {
	alg     jwa.KeyAlgorithm
	key     any
	headers Headers
}

type WithKeySuboption interface {
	Option
	withKeySuboption()
}

type withKeySuboption struct {
	Option
}

func (*withKeySuboption) withKeySuboption() {}

// WithPerRecipientHeaders is used to pass header values for each recipient.
// Note that these headers are by definition _unprotected_.
//
// The supplied Headers is cloned before being stored in the option, so the
// caller retains exclusive ownership of the original instance and the
// library never mutates or pools it.
func WithPerRecipientHeaders(hdr Headers) WithKeySuboption {
	if hdr != nil {
		if cloned, err := hdr.Clone(); err == nil {
			hdr = cloned
		}
	}
	return &withKeySuboption{option.New(identPerRecipientHeaders{}, hdr)}
}

// WithKey is used to pass a static algorithm/key pair to either `jwe.Encrypt()` or `jwe.Decrypt()`.
// Either a raw key or `jwk.Key` may be passed as `key`. If `key` is a `jwk.Key`,
// it must export to one of the raw key types described below.
//
// The `alg` parameter is the identifier for the key encryption algorithm that should be used.
// It is of type `jwa.KeyAlgorithm` but in reality you can only pass `jwa.KeyEncryptionAlgorithm`
// types. It is this way so that the value in `(jwk.Key).Algorithm()` can be directly
// passed to the option. If you specify other algorithm types such as `jwa.SignatureAlgorithm`,
// then you will get an error when `jwe.Encrypt()` or `jwe.Decrypt()` is executed.
//
// Built-in algorithm/key pairs are:
//
//   - `jwa.RSA1_5()` and `jwa.RSA_OAEP*()`: `*rsa.PublicKey` for `jwe.Encrypt()`
//     and the matching `*rsa.PrivateKey` for `jwe.Decrypt()`
//   - `jwa.A128KW()`, `jwa.A192KW()`, `jwa.A256KW()`, `jwa.A128GCMKW()`,
//     `jwa.A192GCMKW()`, and `jwa.A256GCMKW()`: shared symmetric key bytes of
//     the size required by the selected algorithm
//   - `jwa.DIRECT()`: shared symmetric key bytes used as the CEK. The key length
//     must match the selected `enc`, and DIRECT supports only a single recipient
//   - `jwa.ECDH_ES()` and `jwa.ECDH_ES_A*KW()`: recipient public key for
//     `jwe.Encrypt()` and the matching private key for `jwe.Decrypt()`. Built-in
//     support accepts `*ecdsa.PublicKey`, `*ecdsa.PrivateKey`,
//     `*ecdh.PublicKey`, and `*ecdh.PrivateKey`; `jwa.ECDH_ES()` also supports
//     only a single recipient
//   - `jwa.PBES2_*()`: password bytes
//
// `jwa.RSA1_5()` is supported only for interoperability with legacy peers.
// New applications should prefer an RSA-OAEP variant such as
// `jwa.RSA_OAEP_256()` because PKCS#1 v1.5 decryption is exposed to
// Bleichenbacher-style oracle attacks.
//
// Additional algorithms may be added by extension packages, but the key must
// still match the contract for the selected `alg`.
//
// Unlike `jwe.WithKeySet()`, the `kid` field does not need to match for the key
// to be tried.
func WithKey(alg jwa.KeyAlgorithm, key any, options ...WithKeySuboption) EncryptDecryptOption {
	var hdr Headers
	for _, option := range options {
		switch option.Ident() {
		case identPerRecipientHeaders{}:
			if err := option.Value(&hdr); err != nil {
				panic(`jwe.WithKey() requires Headers value for WithPerRecipientHeaders option`)
			}
		}
	}

	return &encryptDecryptOption{option.New(identKey{}, &withKey{
		alg:     alg,
		key:     key,
		headers: hdr,
	})}
}

// WithKeySet specifies a JWKS (jwk.Set) to use for decryption. The
// recipient's `kid` header selects a key from the set, and the key's
// `alg` (or, when the JWK lacks `alg`, the recipient's declared `alg`)
// drives the decrypt-time dispatch.
//
// By default WithKeySet requires the JWE to carry a `kid` header that
// matches a key in the set. Pass `WithRequireKid(false)` to fall back
// to trying every key in the set (slower, looser; intended for legacy
// peers that don't emit `kid`). Per-key errors from the set are
// surfaced via `errors.Join` when nothing matched, so a caller
// debugging "why didn't my keyset match" sees the per-key reasons.
//
// Security note: the recipient's per-recipient header is unprotected.
// When the selected JWK has no `alg`, the keyset provider falls back
// to the per-recipient `alg`, then the protected header's `alg`.
// `jwe.Decrypt` re-checks `alg` against the integrity-protected
// protected header before any cryptographic call (RFC 7516 §7.2.1
// disjointness).
func WithKeySet(set jwk.Set, options ...WithKeySetSuboption) DecryptOption {
	requireKid := true
	for _, option := range options {
		switch option.Ident() {
		case identRequireKid{}:
			if err := option.Value(&requireKid); err != nil {
				panic(`jwe.WithKeySet() requires bool value for WithRequireKid option`)
			}
		}
	}

	return WithKeyProvider(&keySetProvider{
		set:        set,
		requireKid: requireKid,
	})
}

// WithJSON specifies that the result of `jwe.Encrypt()` is serialized in
// JSON format.
//
// If you pass multiple keys to `jwe.Encrypt()`, it will fail unless
// you also pass this option.
func WithJSON(options ...WithJSONSuboption) EncryptOption {
	var pretty bool
	for _, option := range options {
		switch option.Ident() {
		case identPretty{}:
			if err := option.Value(&pretty); err != nil {
				panic(`jwe.WithJSON() requires bool value for WithPretty option`)
			}
		}
	}

	format := fmtJSON
	if pretty {
		format = fmtJSONPretty
	}
	return &encryptOption{option.New(identSerialization{}, format)}
}
