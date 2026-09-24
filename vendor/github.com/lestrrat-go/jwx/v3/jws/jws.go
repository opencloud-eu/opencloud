//go:generate ../tools/cmd/genjws.sh

// Package jws implements the digital signature on JSON based data
// structures as described in https://tools.ietf.org/html/rfc7515
//
// If you do not care about the details, the only things that you
// would need to use are the following functions:
//
//	jws.Sign(payload, jws.WithKey(algorithm, key))
//	jws.Verify(serialized, jws.WithKey(algorithm, key))
//
// To sign, simply use `jws.Sign`. `payload` is a []byte buffer that
// contains whatever data you want to sign. `alg` is one of the
// jwa.SignatureAlgorithm constants from package jwa. For RSA and
// ECDSA family of algorithms, you will need to prepare a private key.
// For HMAC family, you just need a []byte value. The `jws.Sign`
// function will return the encoded JWS message on success.
//
// To verify, use `jws.Verify`. It will parse the `encodedjws` buffer
// and verify the result using `algorithm` and `key`. Upon successful
// verification, the original payload is returned, so you can work on it.
//
// As a sidenote, consider using github.com/lestrrat-go/htmsig if you
// looking for HTTP Message Signatures (RFC9421) -- it uses the same
// underlying signing/verification mechanisms as this module.
package jws

import (
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"
	"sync/atomic"
	"unicode"
	"unicode/utf8"

	"github.com/lestrrat-go/jwx/v3/internal/base64"
	"github.com/lestrrat-go/jwx/v3/internal/json"
	"github.com/lestrrat-go/jwx/v3/internal/pool"
	"github.com/lestrrat-go/jwx/v3/internal/tokens"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws/jwsbb"
)

var registry = json.NewRegistry()

var maxSignatures atomic.Int64

func init() {
	maxSignatures.Store(100)
}

var signers = make(map[jwa.SignatureAlgorithm]Signer)
var muSigner = &sync.Mutex{}

func removeSigner(alg jwa.SignatureAlgorithm) {
	muSigner.Lock()
	defer muSigner.Unlock()
	delete(signers, alg)
}

type defaultSigner struct {
	alg jwa.SignatureAlgorithm
}

func (s defaultSigner) Algorithm() jwa.SignatureAlgorithm {
	return s.alg
}

func (s defaultSigner) Sign(key any, payload []byte) ([]byte, error) {
	return jwsbb.Sign(key, s.alg.String(), payload, nil)
}

type signerAdapter struct {
	signer Signer
}

func (s signerAdapter) Algorithm() jwa.SignatureAlgorithm {
	return s.signer.Algorithm()
}

func (s signerAdapter) Sign(key any, payload []byte) ([]byte, error) {
	return s.signer.Sign(payload, key)
}

const (
	fmtInvalid = 1 << iota
	fmtCompact
	fmtJSON
	fmtJSONPretty
	fmtMax
)

func validateKeyBeforeUse(key any) error {
	jwkKey, ok := key.(jwk.Key)
	if !ok {
		converted, err := jwk.Import(key)
		if err != nil {
			return fmt.Errorf(`could not convert key of type %T to jwk.Key for validation: %w`, key, err)
		}
		jwkKey = converted
	}
	return jwkKey.Validate()
}

// Sign generates a JWS message for the given payload and returns
// it in serialized form, which can be in either compact or
// JSON format. Default is compact.
//
// You must pass at least one key to `jws.Sign()` by using `jws.WithKey()`
// option.
//
//	jws.Sign(payload, jws.WithKey(alg, key))
//	jws.Sign(payload, jws.WithJSON(), jws.WithKey(alg1, key1), jws.WithKey(alg2, key2))
//
// Note that in the second example the `jws.WithJSON()` option is
// specified as well. This is because the compact serialization
// format does not support multiple signatures, and users must
// specifically ask for the JSON serialization format.
//
// Read the documentation for `jws.WithKey()` to learn more about the
// possible values that can be used for `alg` and `key`.
//
// You may create JWS messages with the "none" (jwa.NoSignature) algorithm
// if you use the `jws.WithInsecureNoSignature()` option. This option
// can be combined with one or more signature keys, as well as the
// `jws.WithJSON()` option to generate multiple signatures (though
// the usefulness of such constructs is highly debatable)
//
// Note that this library does not allow you to successfully call `jws.Verify()` on
// signatures with the "none" algorithm. To parse these, use `jws.Parse()` instead.
//
// If you want to use a detached payload, use `jws.WithDetachedPayload()` as
// one of the options. When you use this option, you must always set the
// first parameter (`payload`) to `nil`, or the function will return an error
//
// You may also want to look at how to pass protected headers to the
// signing process, as you will likely be required to set the `b64` field
// when using detached payload.
//
// RFC 7797 note: producing an in-band compact JWS with `b64=false`
// (i.e. setting the `b64` protected header to `false` without also
// passing [WithDetachedPayload]) is "NOT RECOMMENDED" per §5.2; strict
// peers commonly reject such messages. The canonical pairing for
// `b64=false` is [WithDetachedPayload] (or [WithDetachedPayloadReader]
// for streaming), which keeps the unencoded payload out of the wire
// format. Sign auto-declares `"b64"` in `crit` whenever `b64=false`
// is set, so the produced JWS is at least RFC 7797 §3 conformant on
// the producer side.
//
// Look for options that return `jws.SignOption` or `jws.SignVerifyOption`
// for a complete list of options that can be passed to this function.
//
// You can use `errors.Is` with `jws.SignError()` to check if an error is from this function.
func Sign(payload []byte, options ...SignOption) ([]byte, error) {
	sc := signContextPool.Get()
	defer signContextPool.Put(sc)

	sc.payload = payload

	if err := sc.ProcessOptions(options); err != nil {
		return nil, makeSignError(prefixJwsSign, `failed to process options: %w`, err)
	}

	lsigner := len(sc.sigbuilders)
	if lsigner == 0 {
		return nil, makeSignError(prefixJwsSign, `no signers available. Specify an algorithm and a key using jws.WithKey()`)
	}

	// Design note: while we could have easily set format = fmtJSON when
	// lsigner > 1, I believe the decision to change serialization formats
	// must be explicitly stated by the caller. Otherwise, I'm pretty sure
	// there would be people filing issues saying "I get JSON when I expected
	// compact serialization".
	//
	// Therefore, instead of making implicit format conversions, we force the
	// user to spell it out as `jws.Sign(..., jws.WithJSON(), jws.WithKey(...), jws.WithKey(...))`
	if sc.format == fmtCompact && lsigner != 1 {
		return nil, makeSignError(prefixJwsSign, `cannot have multiple signers (keys) specified for compact serialization. Use only one jws.WithKey()`)
	}

	if sc.payloadReader != nil {
		return sc.signStreaming()
	}

	// Create a Message object with all the bits and bobs, and we'll
	// serialize it in the end
	var result Message

	if err := sc.PopulateMessage(&result); err != nil {
		return nil, makeSignError(prefixJwsSign, `failed to populate message: %w`, err)
	}
	switch sc.format {
	case fmtJSON:
		return json.Marshal(result)
	case fmtJSONPretty:
		return json.MarshalIndent(result, "", "  ")
	case fmtCompact:
		// Take the only signature object, and convert it into a Compact
		// serialization format
		var compactOpts []CompactOption
		if sc.detached {
			compactOpts = append(compactOpts, WithDetached(true))
		}
		for _, option := range options {
			if copt, ok := option.(CompactOption); ok {
				compactOpts = append(compactOpts, copt)
			}
		}
		return Compact(&result, compactOpts...)
	default:
		return nil, makeSignError(prefixJwsSign, `invalid serialization format`)
	}
}

var allowNoneWhitelist = jwk.WhitelistFunc(func(string) bool {
	return false
})

// Verify checks if the given JWS message is verifiable using `alg` and `key`.
// `key` may be a "raw" key (e.g. rsa.PublicKey) or a jwk.Key
//
// If the verification is successful, `err` is nil, and the content of the
// payload that was signed is returned. If you need more fine-grained
// control of the verification process, manually generate a
// `Verifier` in `verify` subpackage, and call `Verify` method on it.
// If you need to access signatures and JOSE headers in a JWS message,
// use `Parse` function to get `Message` object.
//
// Because the use of "none" (jwa.NoSignature) algorithm is strongly discouraged,
// this function DOES NOT consider it a success when `{"alg":"none"}` is
// encountered in the message (it would also be counterintuitive when the code says
// it _verified_ something when in fact it did no such thing). If you want to
// accept messages with "none" signature algorithm, use `jws.Parse` to get the
// raw JWS message.
//
// The error returned by this function is of type can be checked against
// `jws.VerifyError()` and `jws.VerificationError()`. The latter is returned
// when the verification process itself fails (e.g. invalid signature, wrong key),
// while the former is returned when any other part of the `jws.Verify()`
// function fails.
//
// When `jws.WithDetachedPayloadReader()` is used, the payload is streamed
// from the caller's `io.Reader` and is not extracted from the JWS envelope.
// In that case, the returned `[]byte` is a non-nil zero-length slice on
// success; the verified bytes are whatever the caller read from the Reader.
// Do not treat the returned slice as "the payload is empty" — callers that
// need the payload bytes must retain their own copy.
//
// Context cancellation is governed by [WithContext]. The slow-path verify
// loop checks ctx.Err() between each signature, each key provider, and
// each (alg, key) attempt; jkuProvider passes ctx to its underlying
// jwk.Fetcher; the streaming path checks ctx between payload Reads.
// staticKeyProvider and keySetProvider do not consult ctx inside
// FetchKeys themselves (their backing data is already in memory) — see
// the [WithContext] godoc for the full per-layer breakdown.
func Verify(buf []byte, options ...VerifyOption) ([]byte, error) {
	vc := verifyContextPool.Get()
	defer verifyContextPool.Put(vc)

	if err := vc.ProcessOptions(options); err != nil {
		return nil, makeVerifyError(`failed to process options: %w`, err)
	}

	return vc.VerifyMessage(buf)
}

// getB64Value reads the typed "b64" header field and returns its value,
// or RFC 7797's default of true when the field is unset.
func getB64Value(hdr Headers) bool {
	v, ok := hdr.B64()
	if !ok {
		return true // RFC 7797 default
	}
	return v
}

func detectParseFormat(src []byte) int {
	for i := 0; i < len(src); {
		r := rune(src[i])
		width := 1
		if r >= utf8.RuneSelf {
			r, width = utf8.DecodeRune(src[i:])
		}
		if !unicode.IsSpace(r) {
			if r == tokens.OpenCurlyBracket {
				return fmtJSON
			}
			return fmtCompact
		}
		i += width
	}
	return 0
}

// Parse parses contents from the given source and creates a jws.Message
// struct. By default the input can be in either compact or full JSON serialization.
//
// You may pass `jws.WithJSON()` and/or `jws.WithCompact()` to specify
// explicitly which format to use. If neither or both is specified, the function
// will attempt to autodetect the format. If one or the other is specified,
// only the specified format will be attempted.
//
// Bounding the input size is the caller's responsibility; this function
// trusts the caller-provided src. See docs/13-input-size.md.
//
// On error, returns a jws.ParseError.
func Parse(src []byte, options ...ParseOption) (*Message, error) {
	maxSigs := int(maxSignatures.Load())

	var formats int
	for _, option := range options {
		switch option.Ident() {
		case identSerialization{}:
			var v int
			if err := option.Value(&v); err != nil {
				return nil, makeParseError(`jws.Parse`, `failed to retrieve serialization option value: %w`, err)
			}
			switch v {
			case fmtJSON:
				formats |= fmtJSON
			case fmtCompact:
				formats |= fmtCompact
			}
		case identMaxSignatures{}:
			if err := option.Value(&maxSigs); err != nil {
				return nil, makeParseError(`jws.Parse`, `failed to retrieve max signatures option value: %w`, err)
			}
			if maxSigs <= 0 {
				return nil, makeParseError(`jws.Parse`, `WithMaxSignatures must be greater than zero`)
			}
		}
	}

	// if format is 0 or both JSON/Compact, auto detect
	if v := formats & (fmtJSON | fmtCompact); v == 0 || v == fmtJSON|fmtCompact {
		formats = detectParseFormat(src)
	}

	if formats&fmtCompact == fmtCompact {
		msg, err := parseCompact(src)
		if err != nil {
			return nil, makeParseError(`jws.Parse`, `failed to parse compact format: %w`, err)
		}
		return msg, nil
	} else if formats&fmtJSON == fmtJSON {
		msg, err := parseJSON(src, maxSigs)
		if err != nil {
			return nil, makeParseError(`jws.Parse`, `failed to parse JSON format: %w`, err)
		}
		return msg, nil
	}

	return nil, makeParseError(`jws.Parse`, `invalid byte sequence`)
}

// ParseString parses contents from the given source and creates a jws.Message
// struct. The input can be in either compact or full JSON serialization.
//
// On error, returns a jws.ParseError.
func ParseString(src string, options ...ParseOption) (*Message, error) {
	msg, err := Parse([]byte(src), options...)
	if err != nil {
		return nil, makeParseError(`jws.ParseString`, `failed to parse string: %w`, err)
	}
	return msg, nil
}

// ParseReader parses contents from the given source and creates a jws.Message
// struct. The input can be in either compact or full JSON serialization.
//
// Bounding the input size is the caller's responsibility: wrap src with
// [io.LimitReader] or [net/http.MaxBytesReader] before passing it in. See
// docs/13-input-size.md for the rationale.
//
// On error, returns a jws.ParseError.
func ParseReader(src io.Reader, options ...ParseOption) (*Message, error) {
	buf, err := io.ReadAll(src)
	if err != nil {
		return nil, makeParseError(`jws.ParseReader`, `failed to read from io.Reader: %w`, err)
	}
	return Parse(buf, options...)
}

func parseJSON(data []byte, maxSigs int) (result *Message, err error) {
	var m Message
	m.maxSignatures = maxSigs
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf(`failed to unmarshal jws message: %w`, err)
	}
	return &m, nil
}

// SplitCompact splits a JWS in compact format and returns its three parts
// separately: protected headers, payload and signature.
// On error, returns a jws.ParseError.
//
// Deprecated: This is a low-level API that will be removed in v4.
// Use the jwsbb package directly instead.
func SplitCompact(src []byte) ([]byte, []byte, []byte, error) {
	hdr, payload, signature, err := jwsbb.SplitCompact(src)
	if err != nil {
		return nil, nil, nil, makeParseError(`jws.Parse`, `%w`, err)
	}
	return hdr, payload, signature, nil
}

// SplitCompactString splits a JWT and returns its three parts
// separately: protected headers, payload and signature.
// On error, returns a jws.ParseError.
//
// Deprecated: This is a low-level API that will be removed in v4.
// Use the jwsbb package directly instead.
func SplitCompactString(src string) ([]byte, []byte, []byte, error) {
	hdr, payload, signature, err := jwsbb.SplitCompactString(src)
	if err != nil {
		return nil, nil, nil, makeParseError(`jws.Parse`, `%w`, err)
	}
	return hdr, payload, signature, nil
}

// SplitCompactReader splits a JWT and returns its three parts
// separately: protected headers, payload and signature.
// On error, returns a jws.ParseError.
//
// Deprecated: This is a low-level API that will be removed in v4.
// Use the jwsbb package directly instead.
func SplitCompactReader(rdr io.Reader) ([]byte, []byte, []byte, error) {
	hdr, payload, signature, err := jwsbb.SplitCompactReader(rdr)
	if err != nil {
		return nil, nil, nil, makeParseError(`jws.Parse`, `%w`, err)
	}
	return hdr, payload, signature, nil
}

func parseCompact(data []byte) (m *Message, err error) {
	protected, payload, signature, err := jwsbb.SplitCompact(data)
	if err != nil {
		return nil, makeParseError(`jws.Parse`, `invalid compact serialization format: %w`, err)
	}
	return parse(protected, payload, signature)
}

func parse(protected, payload, signature []byte) (*Message, error) {
	decodedHeader, err := base64.Decode(protected)
	if err != nil {
		return nil, fmt.Errorf(`failed to decode protected headers: %w`, err)
	}

	hdr := NewHeaders()
	if err := json.Unmarshal(decodedHeader, hdr); err != nil {
		return nil, fmt.Errorf(`failed to parse JOSE headers: %w`, err)
	}

	var decodedPayload []byte
	b64 := getB64Value(hdr)
	if !b64 {
		decodedPayload = payload
	} else {
		v, err := base64.Decode(payload)
		if err != nil {
			return nil, fmt.Errorf(`failed to decode payload: %w`, err)
		}
		decodedPayload = v
	}

	decodedSignature, err := base64.Decode(signature)
	if err != nil {
		return nil, fmt.Errorf(`failed to decode signature: %w`, err)
	}
	if len(decodedSignature) == 0 {
		alg, ok := hdr.Algorithm()
		if !ok || alg != jwa.NoSignature() {
			return nil, fmt.Errorf(`empty compact signature requires protected header "alg" to be "none"`)
		}
	}

	var msg Message
	msg.payload = decodedPayload
	msg.signatures = append(msg.signatures, &Signature{
		protected: hdr,
		signature: decodedSignature,
	})
	msg.b64 = b64
	return &msg, nil
}

type CustomDecoder = json.CustomDecoder
type CustomDecodeFunc = json.CustomDecodeFunc

// RegisterCustomField allows users to specify that a private field
// be decoded as an instance of the specified type. This option has
// a global effect.
//
// For example, suppose you have a custom field `x-birthday`, which
// you want to represent as a string formatted in RFC3339 in JSON,
// but want it back as `time.Time`.
//
// In such case you would register a custom field as follows
//
//	jws.RegisterCustomField(`x-birthday`, time.Time{})
//
// Then you can use a `time.Time` variable to extract the value
// of `x-birthday` field, instead of having to use `any`
// and later convert it to `time.Time`
//
//	var bday time.Time
//	_ = hdr.Get(`x-birthday`, &bday)
//
// If you need a more fine-tuned control over the decoding process,
// you can register a `CustomDecoder`. For example, below shows
// how to register a decoder that can parse RFC1123 format string:
//
//	jws.RegisterCustomField(`x-birthday`, jws.CustomDecodeFunc(func(data []byte) (any, error) {
//	  return time.Parse(time.RFC1123, string(data))
//	}))
//
// Please note that use of custom fields can be problematic if you
// are using a library that does not implement MarshalJSON/UnmarshalJSON
// and you try to roundtrip from an object to JSON, and then back to an object.
// For example, in the above example, you can _parse_ time values formatted
// in the format specified in RFC822, but when you convert an object into
// JSON, it will be formatted in RFC3339, because that's what `time.Time`
// likes to do. To avoid this, it's always better to use a custom type
// that wraps your desired type (in this case `time.Time`) and implement
// MarshalJSON and UnmashalJSON.
func RegisterCustomField(name string, object any) {
	registry.Register(name, object)
}

// curver is implemented by jwk.Key types that carry curve information.
type curver interface {
	Crv() (jwa.EllipticCurveAlgorithm, bool)
}

// Helpers for signature verification
var muAlgorithmMaps sync.RWMutex
var keyTypeToAlgorithms = make(map[jwa.KeyType][]jwa.SignatureAlgorithm)
var algorithmToKeyTypes = make(map[jwa.SignatureAlgorithm][]jwa.KeyType)
var curveToAlgorithms = make(map[jwa.EllipticCurveAlgorithm][]jwa.SignatureAlgorithm)

func init() {
	RegisterAlgorithmForKeyType(jwa.OKP(), jwa.EdDSA())
	RegisterAlgorithmForCurve(jwa.Ed25519(), jwa.EdDSAEd25519())
	for _, alg := range []jwa.SignatureAlgorithm{jwa.HS256(), jwa.HS384(), jwa.HS512()} {
		RegisterAlgorithmForKeyType(jwa.OctetSeq(), alg)
	}
	for _, alg := range []jwa.SignatureAlgorithm{jwa.RS256(), jwa.RS384(), jwa.RS512(), jwa.PS256(), jwa.PS384(), jwa.PS512()} {
		RegisterAlgorithmForKeyType(jwa.RSA(), alg)
	}
	for _, alg := range []jwa.SignatureAlgorithm{jwa.ES256(), jwa.ES384(), jwa.ES512()} {
		RegisterAlgorithmForKeyType(jwa.EC(), alg)
	}
}

// RegisterAlgorithmForKeyType registers an additional algorithm as valid for
// the given key type. This is used internally by init() and can also be called
// from external modules that provide support for additional algorithms (e.g. Ed448).
func RegisterAlgorithmForKeyType(kty jwa.KeyType, alg jwa.SignatureAlgorithm) {
	muAlgorithmMaps.Lock()
	defer muAlgorithmMaps.Unlock()
	keyTypeToAlgorithms[kty] = append(keyTypeToAlgorithms[kty], alg)
	if !slices.Contains(algorithmToKeyTypes[alg], kty) {
		algorithmToKeyTypes[alg] = append(algorithmToKeyTypes[alg], kty)
	}
}

// RegisterAlgorithmForCurve registers an algorithm as valid for the given
// elliptic curve. When [AlgorithmsForKey] can determine the curve of a key,
// it returns the union of key-type-level algorithms and curve-specific
// algorithms instead of all algorithms for the key type.
//
// This function is append-only and deduplicates entries, so builtin
// registrations cannot be overwritten by external modules.
func RegisterAlgorithmForCurve(crv jwa.EllipticCurveAlgorithm, alg jwa.SignatureAlgorithm) {
	muAlgorithmMaps.Lock()
	defer muAlgorithmMaps.Unlock()
	if slices.Contains(curveToAlgorithms[crv], alg) {
		return
	}
	curveToAlgorithms[crv] = append(curveToAlgorithms[crv], alg)
}

// AlgorithmsForKey returns the possible signature algorithms that can
// be used for a given key. It only takes in consideration keys/algorithms
// for verification purposes, as this is the only usage where one may need
// dynamically figure out which method to use.
//
// When the key's curve can be determined (via [jwk.Key] Crv() method or
// inferred from the raw Go type), curve-specific algorithms registered via
// [RegisterAlgorithmForCurve] are combined with key-type-level algorithms
// to produce a more precise result.
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
func AlgorithmsForKey(key any) ([]jwa.SignatureAlgorithm, error) {
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
		return nil, fmt.Errorf(`%w: key type %T cannot be used for signing (ecdh keys are key-agreement only)`, errUnclassifiableKey, key)
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
				algs, err := AlgorithmsForKey(pub)
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
			outer := fmt.Errorf(`%w: unknown key type %T`, errUnclassifiableKey, key)
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

	muAlgorithmMaps.RLock()
	defer muAlgorithmMaps.RUnlock()

	ktyAlgs, ok := keyTypeToAlgorithms[kty]
	if !ok {
		return nil, fmt.Errorf(`%w: unregistered key type %q`, errUnclassifiableKey, kty)
	}

	// If we know the curve and there are curve-specific registrations,
	// return only key-type-level algorithms (those not registered under
	// any curve) plus curve-specific algorithms for this curve.
	if hasCrv {
		crvAlgs := curveToAlgorithms[crv]
		return filterAlgorithmsForCurve(ktyAlgs, crvAlgs), nil
	}

	return ktyAlgs, nil
}

// filterAlgorithmsForCurve returns the subset of ktyAlgs that are not
// registered under any curve (i.e., generic for the key type) plus the
// curve-specific algorithms from crvAlgs.
func filterAlgorithmsForCurve(ktyAlgs, crvAlgs []jwa.SignatureAlgorithm) []jwa.SignatureAlgorithm {
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

// unsupportedKeyError builds the rejection error for a jwk.UnsupportedKey
// placeholder that reached a cryptographic entry point. op names the
// operation the placeholder cannot perform (e.g. "signature verification").
// The error names the placeholder's kid and kty, and wraps the retained
// parse error from Reason().
func unsupportedKeyError(uk jwk.UnsupportedKey, op string) error {
	kid, _ := uk.KeyID()
	return fmt.Errorf(`key with kid %q has unsupported key type %q and cannot be used for %s; an extension module may be required to parse it: %w`, kid, uk.KeyType().String(), op, uk.Reason())
}

// validateAlgorithmForKey checks that alg is compatible with key.
// A jwk.UnsupportedKey placeholder is rejected up front — before any of
// the carve-outs below — because it carries no usable key material for
// any algorithm, custom or built-in.
// Three classification failures are intentionally allowed through:
// (a) a nil key, used by keyless algorithms (see GH910);
// (b) any key handed to an algorithm with a user-registered custom
// Signer2/Verifier2 — custom implementations may accept arbitrary key
// types that AlgorithmsForKey cannot classify; and
// (c) an opaque crypto.Signer whose .Public() is itself a crypto.Signer,
// the one case AlgorithmsForKey refuses to recurse into.
// Every other classification failure is surfaced so callers get a crisp
// option-boundary rejection instead of a deep-stack error.
func validateAlgorithmForKey(alg jwa.SignatureAlgorithm, key any) error {
	if uk, ok := key.(jwk.UnsupportedKey); ok {
		return fmt.Errorf(`jws.WithKey: %w`, unsupportedKeyError(uk, `signing or signature verification`))
	}
	if key == nil {
		return nil
	}
	algs, err := AlgorithmsForKey(key)
	if err != nil {
		if hasCustomSigVerifier(alg) {
			return nil
		}
		if signer, ok := key.(crypto.Signer); ok {
			if _, isSigner := signer.Public().(crypto.Signer); isSigner {
				return nil
			}
		}
		return fmt.Errorf(`jws.WithKey: %w`, err)
	}
	if !slices.Contains(algs, alg) {
		if hasCustomSigVerifier(alg) {
			return nil
		}
		return fmt.Errorf(`jws.WithKey: algorithm %q is not compatible with key type %T`, alg, key)
	}
	return nil
}

// hasCustomSigVerifier reports whether a non-default Signer2 or
// Verifier2 has been registered for alg. When this is true, key-type
// validation must be skipped: the custom implementation decides what
// key types it accepts.
func hasCustomSigVerifier(alg jwa.SignatureAlgorithm) bool {
	muSigner2DB.RLock()
	s, sok := signer2DB[alg]
	muSigner2DB.RUnlock()
	if sok {
		if _, isDefault := s.(defaultSigner); !isDefault {
			return true
		}
	}
	muVerifier2DB.RLock()
	v, vok := verifier2DB[alg]
	muVerifier2DB.RUnlock()
	if vok {
		if _, isDefault := v.(defaultVerifier); !isDefault {
			return true
		}
	}
	return false
}

// Settings allows you to set global settings for this JWS operations.
//
// Currently, the only setting available is `jws.WithLegacySigners()`,
// which for various reason is now a no-op.
func Settings(options ...GlobalOption) {
	for _, option := range options {
		switch option.Ident() {
		case identLegacySigners{}:
		case identMaxSignatures{}:
			var v int
			if err := option.Value(&v); err != nil {
				panic(fmt.Sprintf("jws.Settings: value for WithMaxSignatures must be an int: %s", err))
			}
			if v <= 0 {
				panic("jws.Settings: WithMaxSignatures must be greater than zero")
			}
			maxSignatures.Store(int64(v))
		}
	}
}

// VerifyCompactFast is a fast path verification function for JWS messages
// in compact serialization format.
//
// This function is considered experimental, and may change or be removed
// in the future.
//
// VerifyCompactFast performs signature verification on a JWS compact
// serialization without fully parsing the message into a jws.Message object.
// This makes it more efficient for cases where you only need to verify
// the signature and extract the payload, without needing access to headers
// or other JWS metadata.
//
// Returns the original payload that was signed if verification succeeds.
//
// Unlike jws.Verify(), this function requires you to specify the
// algorithm explicitly rather than extracting it from the JWS headers.
// This can be useful for performance-critical applications where the
// algorithm is known in advance.
//
// Since this function avoids doing many checks that jws.Verify would perform,
// you must ensure to perform the necessary checks including ensuring that algorithm is safe to use for your payload yourself.
//
// VerifyCompactFast cross-checks the protected header's "alg" against
// the caller-supplied alg: if the header omits "alg" (required by
// RFC 7515 §4.1.1) or advertises a different value, it returns a
// verification error. This prevents silently verifying a message
// under a different discipline than the one its header advertises.
//
// VerifyCompactFast refuses messages whose protected header carries a
// "crit" list. RFC 7515 §4.1.11 requires every critical extension to be
// understood by the recipient, and the fast path has no WithCritExtension
// allowlist to consult. On crit-present input it returns a sentinel error
// that callers can detect with errors.Is(err, jws.ErrCritPresent()) and
// retry through jws.Verify, which enforces the full validateCritical rule
// set. Applications that may legitimately receive "crit" headers should
// call jws.Verify directly.
//
// VerifyCompactFast assumes the JWS uses the default "b64":true
// (base64url-encoded) payload encoding. Any protected header carrying
// a "b64" entry is refused with jws.ErrB64Present(), regardless of
// whether "crit" also lists it: the fast path's signing-input
// reconstruction and post-verify base64 decode both depend on the
// default encoding, and a non-conformant b64=false producer (one that
// omits "b64" from "crit") would otherwise verify cryptographically
// while returning bytes that differ from the producer's intent.
// Detached-payload callers must use jws.Verify with jws.WithDetachedPayload
// regardless, since VerifyCompactFast has no way to accept a detached
// payload.
func VerifyCompactFast(key any, compact []byte, alg jwa.SignatureAlgorithm) ([]byte, error) {
	if err := validateAlgorithmForKey(alg, key); err != nil {
		return nil, makeVerifyError(`%w`, err)
	}

	algstr := alg.String()

	// Split the serialized JWS into its components
	hdr, payload, encodedSig, err := jwsbb.SplitCompact(compact)
	if err != nil {
		return nil, makeVerifyError("failed to split compact: %w", err)
	}

	parsedHdr := jwsbb.HeaderParseCompact(hdr)

	// Refuse crit-bearing messages: the fast path has no WithCritExtension
	// allowlist, so accepting them would silently violate RFC 7515 §4.1.11.
	// Callers that wrap VerifyCompactFast can detect this via
	// errors.Is(err, jws.ErrCritPresent()) and fall through to jws.Verify.
	// The sentinel is wrapped in verifyError so the same error also matches
	// errors.Is(err, jws.VerifyError()) — fast-path refusals are a verify
	// error, just one with a more specific classification available.
	if jwsbb.HeaderHas(parsedHdr, CriticalKey) {
		return nil, verifyError{errCritPresent}
	}

	// Refuse "b64"-bearing messages, regardless of whether "crit" also
	// lists it. The signing-input reconstruction and the post-verify
	// base64 decode both assume the default b64=true encoding; a
	// b64=false JWS that the fast path "verified" would either fail the
	// post-verify base64 decode with a misleading error, or — worse —
	// return base64-decoded garbage as the payload while the producer's
	// raw bytes silently disagree. jws.Verify has the WithDetachedPayload
	// / WithCritExtension machinery to handle b64=false correctly. As with
	// the crit refusal above, the sentinel is wrapped in verifyError so the
	// same error matches both jws.ErrB64Present() and jws.VerifyError().
	if jwsbb.HeaderHas(parsedHdr, "b64") {
		return nil, verifyError{errB64Present}
	}

	// Cross-check the protected header "alg" against the caller-supplied
	// alg. RFC 7515 §4.1.1 makes "alg" mandatory in the protected header
	// for compact serialization, and a mismatch between what the message
	// advertises and the discipline under which we verify is the sort of
	// silent divergence that downstream code (e.g. JWT consumers) should
	// not be asked to re-discover on its own.
	hdrAlg, err := jwsbb.HeaderGetString(parsedHdr, AlgorithmKey)
	if err != nil {
		return nil, verifyError{verificationError{fmt.Errorf(`jws.Verify: failed to extract %q from protected header: %w`, AlgorithmKey, err)}}
	}
	if hdrAlg != algstr {
		return nil, verifyError{verificationError{fmt.Errorf(`jws.Verify: protected header %q %q does not match caller-supplied algorithm %q`, AlgorithmKey, hdrAlg, algstr)}}
	}

	signature, err := base64.Decode(encodedSig)
	if err != nil {
		return nil, makeVerifyError("failed to decode signature: %w", err)
	}

	// Instead of appending, copy the data from hdr/payload
	lvb := len(hdr) + 1 + len(payload)
	verifyBuf := pool.ByteSlice().GetCapacity(lvb)
	verifyBuf = verifyBuf[:lvb]
	copy(verifyBuf, hdr)
	verifyBuf[len(hdr)] = tokens.Period
	copy(verifyBuf[len(hdr)+1:], payload)
	defer pool.ByteSlice().Put(verifyBuf)

	// Verify the signature
	if verifier2, err := VerifierFor(alg); err == nil {
		if err := verifier2.Verify(key, verifyBuf, signature); err != nil {
			return nil, verifyError{verificationError{fmt.Errorf("signature verification failed for %s: %w", algstr, err)}}
		}
	} else {
		legacyVerifier, err := NewVerifier(alg)
		if err != nil {
			return nil, makeVerifyError("failed to create verifier for %s: %w", algstr, err)
		}
		if err := legacyVerifier.Verify(verifyBuf, signature, key); err != nil {
			return nil, verifyError{verificationError{fmt.Errorf("signature verification failed for %s: %w", algstr, err)}}
		}
	}

	decoded, err := base64.Decode(payload)
	if err != nil {
		return nil, makeVerifyError("failed to decode payload: %w", err)
	}
	return decoded, nil
}
