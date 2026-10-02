package jws

import (
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/option/v2"
)

type identInsecureNoSignature struct{}
type identCritExtension struct{}

// WithCritExtension declares that the caller understands and will process
// the named "crit" (Critical) header parameter extension(s) per RFC 7515
// Section 4.1.11. The option is variadic and accumulating: a single call
// may register any number of extension names, and the option may be
// passed multiple times to add more.
//
// This option only takes effect when jws.WithCritValidation(true) is
// also passed. With validation enabled, jws.Verify() rejects any JWS
// whose protected header lists a "crit" extension that has not been
// declared via this option, satisfying the RFC's requirement that
// recipients MUST reject any "crit" extension they do not understand.
//
// IMPORTANT: declaring an extension here is a promise to the library
// that the caller knows what the extension means and will perform any
// validation, side effect, or policy enforcement the extension requires
// AFTER jws.Verify() returns successfully. The library cannot inspect
// or enforce the semantics of an extension; it only checks that every
// "crit" entry in the message has been declared. If you register an
// extension and then forget to act on its value, you have effectively
// disabled the protection the producer was trying to obtain by listing
// the extension as critical.
//
// Concretely, the post-verify code path for a declared extension must:
//
//  1. Read the value of the named header from the verified message.
//  2. Apply whatever check or transformation the extension specifies
//     (e.g. for an "x-tenant-binding" extension, refuse to act on the
//     payload unless the binding matches the current tenant).
//  3. Treat any failure of that check as a verification failure for
//     the application's purposes, even though jws.Verify() returned
//     no error.
func WithCritExtension(names ...string) VerifyOption {
	return &verifyOption{option.New(identCritExtension{}, names)}
}

// WithJSON specifies that the result of `jws.Sign()` is serialized in
// JSON format.
//
// If you pass multiple keys to `jws.Sign()`, it will fail unless
// you also pass this option.
func WithJSON(options ...WithJSONSuboption) SignVerifyParseOption {
	var pretty bool
	for _, option := range options {
		switch option.Ident() {
		case identPretty{}:
			if err := option.Value(&pretty); err != nil {
				panic(`jws.WithJSON() option must be of type bool`)
			}
		}
	}

	format := fmtJSON
	if pretty {
		format = fmtJSONPretty
	}
	return &signVerifyParseOption{option.New(identSerialization{}, format)}
}

type withKey struct {
	alg       jwa.KeyAlgorithm
	key       any
	protected Headers
	public    Headers
}

// Protected returns the protected headers. If w.protected is nil and v is
// non-nil, v is stored as the protected headers before returning. This allows
// callers to provide a default Headers that is used only when none was
// explicitly configured via WithProtectedHeaders.
func (w *withKey) Protected(v Headers) Headers {
	if w.protected == nil && v != nil {
		w.protected = v
	}
	return w.protected
}

// WithKey is used to pass a static algorithm/key pair to either `jws.Sign()` or `jws.Verify()`.
//
// IMPORTANT: Although `alg` is typed as `jwa.KeyAlgorithm` for compatibility
// with `(jwk.Key).Algorithm()`, JWS only accepts `jwa.SignatureAlgorithm`
// values here. Passing a key-encryption algorithm such as `jwa.A128KW()` to
// `jws.WithKey()` compiles, but `jws.Sign()` / `jws.Verify()` reject it at runtime.
//
// The `alg` parameter is the identifier for the signature algorithm that should be used.
// It is of type `jwa.KeyAlgorithm` so that the value in `(jwk.Key).Algorithm()` can be
// directly passed to the option, but that is only valid when the JWK is already known to
// be intended for JWS and its `alg` value is a `jwa.SignatureAlgorithm`.
//
// The `alg` parameter cannot be "none" (jwa.NoSignature) for security reasons.
// You will have to use a separate, more explicit option to allow the use of "none"
// algorithm (WithInsecureNoSignature).
//
// The algorithm specified in the `alg` parameter MUST be able to support
// the type of key you provided, otherwise an error is returned.
//
// RFC 7518 Section 3.4 binds each of ES256/ES384/ES512 to one elliptic curve
// (ES256/P-256, ES384/P-384, ES512/P-521), but `jws.Sign()` does not enforce
// that by default: a key on any other curve still signs, and the JWS it
// produces is one strict JOSE implementations reject. Pass
// `jws.WithStrictECDSA(true)` to `jws.Sign()` to reject the mismatch instead.
// `jws.Verify()` is unaffected either way and keeps inferring algorithms from
// a key's curve exactly as before.
//
// Any of the following is accepted for the `key` parameter:
// * A "raw" key (e.g. rsa.PrivateKey, ecdsa.PrivateKey, etc)
// * A crypto.Signer
// * A jwk.Key
//
// Note that due to technical reasons, this library is NOT able to differentiate
// between a valid/invalid key for given algorithm if the key implements crypto.Signer
// and the key is from an external library. For example, while we can tell that it is
// invalid to use `jwk.WithKey(jwa.RSA256, ecdsaPrivateKey)` because the key is
// presumably from `crypto/ecdsa` or this library, if you use a KMS wrapper
// that implements crypto.Signer that is outside of the go standard library or this
// library, we will not be able to properly catch the misuse of such keys --
// the output will happily generate an ECDSA signature even in the presence of
// `jwa.RSA256`
//
// A `crypto.Signer` is used when the private part of a key is
// kept in an inaccessible location, such as hardware.
// `crypto.Signer` is currently supported for RSA, ECDSA, and EdDSA
// family of algorithms. You may consider using `github.com/jwx-go/crypto-signer`
// if you would like to use keys stored in GCP/AWS KMS services.
//
// If the key is a jwk.Key and the key contains a key ID (`kid` field),
// then it is added to the protected header generated by the signature.
//
// `jws.WithKey()` can further accept suboptions to change signing behavior
// when used with `jws.Sign()`. `jws.WithProtected()` and `jws.WithPublic()`
// can be passed to specify JWS headers that should be used whe signing.
//
// If the protected headers contain "b64" field, then the boolean value for the field
// is respected when serializing. That is, if you specify a header with
// `{"b64": false}`, then the payload is not base64 encoded.
//
// These suboptions are ignored when the `jws.WithKey()` option is used with `jws.Verify()`.
func WithKey(alg jwa.KeyAlgorithm, key any, options ...WithKeySuboption) SignVerifyOption {
	// Implementation note: this option is shared between Sign() and
	// Verify(). As such we don't create a KeyProvider here because
	// if used in Sign() we would be doing something else.
	var protected, public Headers
	for _, option := range options {
		switch option.Ident() {
		case identProtectedHeaders{}:
			if err := option.Value(&protected); err != nil {
				panic(`jws.WithKey() option must be of type Headers`)
			}
		case identPublicHeaders{}:
			if err := option.Value(&public); err != nil {
				panic(`jws.WithKey() option must be of type Headers`)
			}
		}
	}

	return &signVerifyOption{
		option.New(identKey{}, &withKey{
			alg:       alg,
			key:       key,
			protected: protected,
			public:    public,
		}),
	}
}

// WithKeySet specifies a JWKS (jwk.Set) to use for verification.
//
// Because a JWKS can contain multiple keys and this library cannot tell
// which one of the keys should be used for verification, we by default
// require that both `alg` and `kid` fields in the JWS _and_ the
// key match before a key is considered to be used.
//
// There are ways to override this behavior, but they must be explicitly
// specified by the caller.
//
// To work with keys/JWS messages not having a `kid` field, you may specify
// the suboption `WithKeySetRequired` via `jws.WithKey(key, jws.WithRequireKid(false))`.
// This will allow the library to proceed without having to match the `kid` field.
//
// However, it will still check if the `alg` fields in the JWS message and the key(s)
// match. If you must work with JWS messages that do not have an `alg` field,
// you will need to use `jws.WithKeySet(key, jws.WithInferAlgorithm(true))`.
//
// See the documentation for `WithInferAlgorithm()` for more details.
func WithKeySet(set jwk.Set, options ...WithKeySetSuboption) VerifyOption {
	requireKid := true
	var useDefault, inferAlgorithm, multipleKeysPerKeyID bool
	for _, option := range options {
		switch option.Ident() {
		case identRequireKid{}:
			if err := option.Value(&requireKid); err != nil {
				panic(`jws.WithKeySet() option must be of type bool`)
			}
		case identUseDefault{}:
			if err := option.Value(&useDefault); err != nil {
				panic(`jws.WithKeySet() option must be of type bool`)
			}
		case identMultipleKeysPerKeyID{}:
			if err := option.Value(&multipleKeysPerKeyID); err != nil {
				panic(`jws.WithKeySet() option must be of type bool`)
			}
		case identInferAlgorithmFromKey{}:
			if err := option.Value(&inferAlgorithm); err != nil {
				panic(`jws.WithKeySet() option must be of type bool`)
			}
		}
	}

	return WithKeyProvider(&keySetProvider{
		set:                  set,
		requireKid:           requireKid,
		useDefault:           useDefault,
		multipleKeysPerKeyID: multipleKeysPerKeyID,
		inferAlgorithm:       inferAlgorithm,
	})
}

// WithVerifyAuto enables automatic verification of the signature using the JWKS specified in
// the `jku` header. Note that by default this option will _reject_ any jku
// provided by the JWS message. Read on for details.
//
// The JWKS is retrieved by the `jwk.Fetcher` specified in the first argument.
// If the fetcher object is nil, the default fetcher, which is the `jwk.Fetch()`
// function (wrapped in the `jwk.FetchFunc` type) is used.
//
// The remaining arguments are passed to the `(jwk.Fetcher).Fetch` method
// when the JWKS is retrieved.
//
//	jws.WithVerifyAuto(nil) // uses jwk.Fetch
//	jws.WithVerifyAuto(jwk.NewCachedFetcher(...)) // uses cached fetcher
//	jws.WithVerifyAuto(myFetcher) // use your custom fetcher
//
// By default a whitelist that disallows all URLs is added to the options
// passed to the fetcher. You must explicitly specify a whitelist that allows
// the URLs you trust. This default behavior is provided because by design
// of the JWS specification it is the/ caller's responsibility to verify if
// the URL specified in the `jku` header can be trusted -- thus by default
// we trust nothing.
//
// Users are free to specify an open whitelist if they so choose, but this must
// be explicitly done:
//
//	jws.WithVerifyAuto(nil, jwk.WithFetchWhitelist(jwk.InsecureWhitelist()))
//
// You can also use `jwk.CachedFetcher` to use cached JWKS objects, but do note
// that this object is not really designed to accommodate a large set of
// arbitrary URLs. Use `jwk.CachedFetcher` as the first argument if you only
// have a small set of URLs that you trust. For anything more complex, you should
// implement your own `jwk.Fetcher` object.
func WithVerifyAuto(f jwk.Fetcher, options ...jwk.FetchOption) VerifyOption {
	// the option MUST start with a "disallow no whitelist" to force
	// users provide a whitelist
	options = append(append([]jwk.FetchOption(nil), jwk.WithFetchWhitelist(allowNoneWhitelist)), options...)

	return WithKeyProvider(jkuProvider{
		fetcher: f,
		options: options,
	})
}

type withInsecureNoSignature struct {
	protected Headers
}

// Protected returns the protected headers. If w.protected is nil and v is
// non-nil, v is stored as the protected headers before returning. This allows
// callers to provide a default Headers that is used only when none was
// explicitly configured via WithProtectedHeaders.
func (w *withInsecureNoSignature) Protected(v Headers) Headers {
	if w.protected == nil && v != nil {
		w.protected = v
	}
	return w.protected
}

// WithInsecureNoSignature creates an option that allows the user to use the
// "none" signature algorithm.
//
// Please note that this is insecure, and should never be used in production
// (this is exactly why specifying "none"/jwa.NoSignature to `jws.WithKey()`
// results in an error when `jws.Sign()` is called -- we do not allow using
// "none" by accident)
//
// TODO: create specific suboption set for this option
func WithInsecureNoSignature(options ...WithKeySuboption) SignOption {
	var protected Headers
	for _, option := range options {
		switch option.Ident() {
		case identProtectedHeaders{}:
			if err := option.Value(&protected); err != nil {
				panic(`jws.WithInsecureNoSignature() option must be of type Headers`)
			}
		}
	}

	return &signOption{
		option.New(identInsecureNoSignature{},
			&withInsecureNoSignature{
				protected: protected,
			},
		),
	}
}
