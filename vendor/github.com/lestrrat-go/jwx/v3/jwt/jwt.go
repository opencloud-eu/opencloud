//go:generate ../tools/cmd/genjwt.sh
//go:generate stringer -type=TokenOption -output=token_options_gen.go

package jwt

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lestrrat-go/jwx/v3"
	"github.com/lestrrat-go/jwx/v3/internal/json"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	jwterrs "github.com/lestrrat-go/jwx/v3/jwt/internal/errors"
	"github.com/lestrrat-go/jwx/v3/jwt/internal/types"
)

var muSettings sync.Mutex
var defaultTruncation atomic.Int64

// Settings controls global settings that are specific to JWTs.
func Settings(options ...GlobalOption) {
	muSettings.Lock()
	defer muSettings.Unlock()

	var flattenAudience bool
	var parsePedantic bool
	var parsePrecision = types.MaxPrecision + 1  // illegal value, so we can detect nothing was set
	var formatPrecision = types.MaxPrecision + 1 // illegal value, so we can detect nothing was set
	truncation := time.Duration(-1)
	for _, option := range options {
		switch option.Ident() {
		case identTruncation{}:
			if err := option.Value(&truncation); err != nil {
				panic(fmt.Sprintf("jwt.Settings: value for WithTruncation must be time.Duration: %s", err))
			}
		case identFlattenAudience{}:
			if err := option.Value(&flattenAudience); err != nil {
				panic(fmt.Sprintf("jwt.Settings: value for WithFlattenAudience must be bool: %s", err))
			}
		case identNumericDateParsePedantic{}:
			if err := option.Value(&parsePedantic); err != nil {
				panic(fmt.Sprintf("jwt.Settings: value for WithNumericDateParsePedantic must be bool: %s", err))
			}
		case identNumericDateParsePrecision{}:
			var v int
			if err := option.Value(&v); err != nil {
				panic(fmt.Sprintf("jwt.Settings: value for WithNumericDateParsePrecision must be int: %s", err))
			}
			// only accept this value if it's in our desired range
			if v >= 0 && v <= int(types.MaxPrecision) {
				parsePrecision = uint32(v)
			}
		case identNumericDateFormatPrecision{}:
			var v int
			if err := option.Value(&v); err != nil {
				panic(fmt.Sprintf("jwt.Settings: value for WithNumericDateFormatPrecision must be int: %s", err))
			}
			// only accept this value if it's in our desired range
			if v >= 0 && v <= int(types.MaxPrecision) {
				formatPrecision = uint32(v)
			}
		}
	}

	if parsePrecision <= types.MaxPrecision { // remember we set default to max + 1
		types.ParsePrecision.Store(parsePrecision)
	}

	if formatPrecision <= types.MaxPrecision { // remember we set default to max + 1
		types.FormatPrecision.Store(formatPrecision)
	}

	{
		var newVal uint32
		if parsePedantic {
			newVal = 1
		}
		types.Pedantic.Store(newVal)
	}

	{
		opts := TokenOptionSet(defaultOptions.Load())
		if flattenAudience {
			opts.Enable(FlattenAudience)
		} else {
			opts.Disable(FlattenAudience)
		}
		defaultOptions.Store(opts.Value())
	}

	if truncation >= 0 {
		defaultTruncation.Store(int64(truncation))
	}
}

var registry = json.NewRegistry()

// ParseString calls Parse against a string
func ParseString(s string, options ...ParseOption) (Token, error) {
	tok, err := parseBytes([]byte(s), options...)
	if err != nil {
		return nil, jwterrs.ParseErrorf(`jwt.ParseString`, `failed to parse string: %w`, err)
	}
	return tok, nil
}

// Parse parses the JWT token payload and creates a new `jwt.Token` object.
// The token must be encoded in JWS compact format, or a raw JSON form of JWT
// without any signatures.
//
// Signed input is verified by default. Pass `jwt.WithKey()`,
// `jwt.WithKeySet()`, `jwt.WithKeyProvider()`, or
// `jwt.WithVerifyAuto(fetcher, fetchOptions...)` when verification is
// required. A bare `jwt.Parse()` call returns an error; to intentionally
// skip verification, pass `jwt.WithVerify(false)` or use
// `jwt.ParseInsecure()`.
//
// `Parse()` also accepts `ValidateOption` values. Validation runs by default
// after parsing, so `jwt.WithValidate(true)` is only needed to override a
// prior `jwt.WithValidate(false)` in the same option set. Pass
// `jwt.WithValidate(false)` if you need to defer validation and call
// `Validate()` yourself later.
//
// To produce nested JWTs, use
// `jwt.NewSerializer().Sign(...).Encrypt(...).Serialize(...)`. `Parse()` does
// not decrypt JWE envelopes; decrypt the outer JWE before calling it.
//
// During verification, if the JWS headers specify a key ID (`kid`), the
// key used for verification must match the specified ID. If you are somehow
// using a key without a `kid` (which is highly unlikely if you are working
// with a JWT from a well-known provider), you can work around this by
// modifying the `jwk.Key` and setting its `kid` field.
//
// This function takes both ParseOption and ValidateOption types:
// ParseOptions control parsing and verification behavior, and
// ValidateOptions are passed to `Validate()` when automatic validation is
// enabled.
func Parse(s []byte, options ...ParseOption) (Token, error) {
	tok, err := parseBytes(s, options...)
	if err != nil {
		return nil, jwterrs.ParseErrorf(`jwt.Parse`, `failed to parse token: %w`, err)
	}
	return tok, nil
}

// ParseInsecure is exactly the same as Parse(), but it disables
// signature verification and token validation.
//
// `jwt.WithVerify()` and `jwt.WithValidate()` may not be specified
// because they would conflict with the function's purpose. Likewise,
// the key-bearing options `jwt.WithKey()`, `jwt.WithKeySet()`,
// `jwt.WithKeyProvider()`, and `jwt.WithVerifyAuto()` are rejected so
// that typos like `jwt.ParseInsecure(data, jwt.WithKey(...))` cannot
// silently skip verification. Use `jwt.Parse` when a key is available.
func ParseInsecure(s []byte, options ...ParseOption) (Token, error) {
	for _, option := range options {
		switch option.Ident() {
		case identVerify{}, identValidate{}:
			return nil, jwterrs.ParseErrorf(`jwt.ParseInsecure`, `jwt.WithVerify() and jwt.WithValidate() may not be specified`)
		case identKey{}, identKeySet{}, identKeyProvider{}, identVerifyAuto{}:
			return nil, jwterrs.ParseErrorf(`jwt.ParseInsecure`, `key-bearing options (jwt.WithKey, jwt.WithKeySet, jwt.WithKeyProvider, jwt.WithVerifyAuto) may not be specified; use jwt.Parse to verify with a key`)
		}
	}

	options = append(options, WithVerify(false), WithValidate(false))
	tok, err := Parse(s, options...)
	if err != nil {
		return nil, jwterrs.ParseErrorf(`jwt.ParseInsecure`, `failed to parse token: %w`, err)
	}
	return tok, nil
}

// ParseReader calls Parse against an io.Reader.
//
// Bounding the input size is the caller's responsibility: wrap src with
// [io.LimitReader] or [net/http.MaxBytesReader] before passing it in. See
// docs/13-input-size.md for the rationale.
func ParseReader(src io.Reader, options ...ParseOption) (Token, error) {
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, jwterrs.ParseErrorf(`jwt.ParseReader`, `failed to read from token data source: %w`, err)
	}
	tok, err := parseBytes(data, options...)
	if err != nil {
		return nil, jwterrs.ParseErrorf(`jwt.ParseReader`, `failed to parse token: %w`, err)
	}
	return tok, nil
}

type parseCtx struct {
	token              Token
	validateOpts       []ValidateOption
	verifyOpts         []jws.VerifyOption
	localReg           *json.Registry
	strictStringClaims *bool // per-call override; nil = use global
	pedantic           bool
	skipVerification   bool
	validate           bool
	withKeyCount       int
	withKey            *withKey // this is used to detect if we have a WithKey option
}

func parseBytes(data []byte, options ...ParseOption) (Token, error) {
	var ctx parseCtx

	// Validation is turned on by default. You need to specify
	// jwt.WithValidate(false) if you want to disable it
	ctx.validate = true

	// Verification is required (i.e., it is assumed that the incoming
	// data is in JWS format) unless the user explicitly asks for
	// it to be skipped.
	verification := true

	var verifyOpts []Option
	for _, o := range options {
		if v, ok := o.(ValidateOption); ok {
			ctx.validateOpts = append(ctx.validateOpts, v)
			// context is used for both verification and validation, so we can't just continue
			switch o.Ident() {
			case identContext{}:
			default:
				continue
			}
		}

		switch o.Ident() {
		case identKey{}:
			// it would be nice to be able to detect if ctx.verifyOpts[0]
			// is a WithKey option, but unfortunately at that point we have
			// already converted the options to a jws option, which means
			// we can no longer compare its Ident() to jwt.identKey{}.
			// So let's just count this here
			ctx.withKeyCount++
			if ctx.withKeyCount == 1 {
				if err := o.Value(&ctx.withKey); err != nil {
					return nil, fmt.Errorf("jws.parseBytes: value for WithKey option must be a *jwt.withKey: %w", err)
				}
			}
			verifyOpts = append(verifyOpts, o)
		case identKeySet{}, identVerifyAuto{}, identKeyProvider{}, identBase64Encoder{}, identContext{}:
			verifyOpts = append(verifyOpts, o)
		case identToken{}:
			var token Token
			if err := o.Value(&token); err != nil {
				return nil, fmt.Errorf("jws.parseBytes: value for WithToken option must be a jwt.Token: %w", err)
			}
			ctx.token = token
		case identPedantic{}:
			if err := o.Value(&ctx.pedantic); err != nil {
				return nil, fmt.Errorf("jws.parseBytes: value for WithPedantic option must be a bool: %w", err)
			}
		case identValidate{}:
			if err := o.Value(&ctx.validate); err != nil {
				return nil, fmt.Errorf("jws.parseBytes: value for WithValidate option must be a bool: %w", err)
			}
		case identVerify{}:
			if err := o.Value(&verification); err != nil {
				return nil, fmt.Errorf("jws.parseBytes: value for WithVerify option must be a bool: %w", err)
			}
		case identTypedClaim{}:
			var pair claimPair
			if err := o.Value(&pair); err != nil {
				return nil, fmt.Errorf("jws.parseBytes: value for WithTypedClaim option must be claimPair: %w", err)
			}
			if ctx.localReg == nil {
				ctx.localReg = json.NewRegistry()
			}
			ctx.localReg.Register(pair.Name, pair.Value)
		case identStrictStringClaims{}:
			var v bool
			if err := o.Value(&v); err != nil {
				return nil, fmt.Errorf("jwt.parseBytes: value for WithStrictStringClaims must be bool: %w", err)
			}
			ctx.strictStringClaims = &v
		}
	}

	if !verification {
		ctx.skipVerification = true
	}

	lvo := len(verifyOpts)
	if lvo == 0 && verification {
		return nil, fmt.Errorf(`jwt.Parse: no keys for verification are provided (use jwt.WithVerify(false) to explicitly skip)`)
	}

	if lvo > 0 {
		converted, err := toVerifyOptions(verifyOpts...)
		if err != nil {
			return nil, fmt.Errorf(`jwt.Parse: failed to convert options into jws.VerifyOption: %w`, err)
		}
		ctx.verifyOpts = converted
	}

	data = bytes.TrimSpace(data)
	return parse(&ctx, data)
}

const (
	_JwsVerifyInvalid = iota
	_JwsVerifyDone
	_JwsVerifyExpectNested
	_JwsVerifySkipped
)

var _ = _JwsVerifyInvalid

func verifyJWS(ctx *parseCtx, payload []byte) ([]byte, int, error) {
	lvo := len(ctx.verifyOpts)
	if lvo == 0 {
		return nil, _JwsVerifySkipped, nil
	}

	if lvo == 1 && ctx.withKeyCount == 1 {
		wk := ctx.withKey
		alg, ok := wk.alg.(jwa.SignatureAlgorithm)
		if ok && len(wk.options) == 0 {
			verified, err := jws.VerifyCompactFast(wk.key, payload, alg)
			if err == nil {
				return verified, peekJWSNestedState(ctx, payload), nil
			}
			// VerifyCompactFast refuses crit-bearing messages. In v3
			// jws.Verify defaults critValidation=false, so the generic
			// fall-through path would still silently accept "crit".
			// Force the strict path here: jwt.Parse must not be laxer
			// than jws.Verify + WithCritValidation.
			if errors.Is(err, jws.ErrCritPresent()) {
				verifyOpts := append(ctx.verifyOpts, jws.WithCompact(), jws.WithCritValidation(true))
				verified, err := jws.Verify(payload, verifyOpts...)
				if err != nil {
					return nil, _JwsVerifyDone, err
				}
				return verified, peekJWSNestedState(ctx, payload), nil
			}
			return nil, _JwsVerifyDone, err
		}
	}

	verifyOpts := append(ctx.verifyOpts, jws.WithCompact())
	verified, err := jws.Verify(payload, verifyOpts...)
	if err != nil {
		return nil, _JwsVerifyDone, err
	}
	return verified, peekJWSNestedState(ctx, payload), nil
}

// peekJWSNestedState returns _JwsVerifyExpectNested when pedantic mode is on
// and the verified JWS protected header carries cty=JWT (RFC 7519 §5.2 — the
// payload is itself a Nested JWT; the outer envelope expects another signed/
// encrypted layer wrapping the JWT, not a raw JWT). Otherwise returns
// _JwsVerifyDone. The signature has already been verified at this point, so
// re-parsing the protected header is safe — it operates on bytes the producer
// signed.
func peekJWSNestedState(ctx *parseCtx, payload []byte) int {
	if !ctx.pedantic {
		return _JwsVerifyDone
	}
	msg, err := jws.Parse(payload, jws.WithCompact())
	if err != nil || len(msg.Signatures()) == 0 {
		return _JwsVerifyDone
	}
	hdr := msg.Signatures()[0].ProtectedHeaders()
	if hdr == nil {
		return _JwsVerifyDone
	}
	cty, ok := hdr.ContentType()
	if !ok {
		return _JwsVerifyDone
	}
	if cty == "JWT" {
		return _JwsVerifyExpectNested
	}
	return _JwsVerifyDone
}

// verify parameter exists to make sure that we don't accidentally skip
// over verification just because alg == ""  or key == nil or something.
func parse(ctx *parseCtx, data []byte) (Token, error) {
	payload := data
	const maxDecodeLevels = 2

	// If cty = `JWT`, we expect this to be a nested structure
	var expectNested bool

OUTER:
	for i := range maxDecodeLevels {
		switch kind := jwx.GuessFormat(payload); kind {
		case jwx.JWT:
			if ctx.pedantic {
				if expectNested {
					return nil, fmt.Errorf(`expected nested encrypted/signed payload, got raw JWT`)
				}
			}

			if i == 0 {
				// We were NOT enveloped in other formats
				if !ctx.skipVerification {
					if _, _, err := verifyJWS(ctx, payload); err != nil {
						return nil, err
					}
				}
			}

			break OUTER
		case jwx.InvalidFormat:
			return nil, UnknownPayloadTypeError()
		case jwx.UnknownFormat:
			// "Unknown" may include invalid JWTs, for example, those who lack "aud"
			// claim. We could be pedantic and reject these
			if ctx.pedantic {
				return nil, fmt.Errorf(`unknown JWT format (pedantic)`)
			}

			if i == 0 {
				// We were NOT enveloped in other formats
				if !ctx.skipVerification {
					if _, _, err := verifyJWS(ctx, payload); err != nil {
						return nil, err
					}
				}
			}
			break OUTER
		case jwx.JWS:
			// Food for thought: This is going to break if you have multiple layers of
			// JWS enveloping using different keys. It is highly unlikely use case,
			// but it might happen.

			// skipVerification should only be set to true by us. It's used
			// when we just want to parse the JWT out of a payload
			if !ctx.skipVerification {
				// nested return value means:
				// false (next envelope _may_ need to be processed)
				// true (next envelope MUST be processed)
				v, state, err := verifyJWS(ctx, payload)
				if err != nil {
					return nil, err
				}

				if state != _JwsVerifySkipped {
					payload = v

					// We only check for cty and typ if the pedantic flag is enabled
					if !ctx.pedantic {
						continue
					}

					if state == _JwsVerifyExpectNested {
						expectNested = true
						continue OUTER
					}

					// if we're not nested, we found our target. bail out of this loop
					break OUTER
				}
			}

			// No verification. Parse the LOOP-LOCAL `payload` (not the
			// original `data`); for a 2-layer nested JWS, iter 2 must
			// see the inner JWS bytes that iter 1 produced, not re-
			// parse the outer envelope.
			m, err := jws.Parse(payload, jws.WithCompact())
			if err != nil {
				return nil, fmt.Errorf(`invalid jws message: %w`, err)
			}
			payload = m.Payload()
		default:
			return nil, fmt.Errorf(`unsupported format (layer: #%d)`, i+1)
		}
		expectNested = false
	}

	if ctx.token == nil {
		ctx.token = New()
	}

	if ctx.localReg != nil || ctx.strictStringClaims != nil {
		dcToken, ok := ctx.token.(TokenWithDecodeCtx)
		if !ok {
			return nil, fmt.Errorf(`typed claim or strict string claims was requested, but the token (%T) does not support DecodeCtx`, ctx.token)
		}

		var strict bool
		if ctx.strictStringClaims != nil {
			strict = *ctx.strictStringClaims
		}

		dc := json.NewDecodeCtxStrictStrings(ctx.localReg, strict)
		dcToken.SetDecodeCtx(dc)
		defer func() { dcToken.SetDecodeCtx(nil) }()
	}

	if err := json.Unmarshal(payload, ctx.token); err != nil {
		return nil, fmt.Errorf(`failed to parse token: %w`, err)
	}

	if ctx.validate {
		if err := Validate(ctx.token, ctx.validateOpts...); err != nil {
			return nil, err
		}
	}
	return ctx.token, nil
}

// Sign is a convenience function to create a signed JWT token serialized in
// compact form.
//
// It accepts either a raw key (e.g. rsa.PrivateKey, ecdsa.PrivateKey, etc)
// or a jwk.Key, and the name of the algorithm that should be used to sign
// the token.
//
// For well-known algorithms with no special considerations (e.g. detached
// payloads, extra protected headers, etc), this function will automatically
// take the fast path and bypass the jws.Sign() machinery, which improves
// performance significantly.
//
// If the key is a jwk.Key and the key contains a key ID (`kid` field),
// then it is added to the protected header generated by the signature
//
// The algorithm specified in the `alg` parameter must be able to support
// the type of key you provided, otherwise an error is returned.
// For convenience `alg` is of type jwa.KeyAlgorithm so you can pass
// the return value of `(jwk.Key).Algorithm()` directly, but in practice
// it must be an instance of jwa.SignatureAlgorithm, otherwise an error
// is returned.
//
// The protected header will also automatically have the `typ` field set
// to the literal value `JWT`, unless you provide a custom value for it
// by jws.WithProtectedHeaders option, that can be passed to `jwt.WithKey“.
func Sign(t Token, options ...SignOption) ([]byte, error) {
	// fast path; can only happen if there is exactly one option
	if len(options) == 1 && (options[0].Ident() == identKey{}) {
		// The option must be a withKey option.
		var wk *withKey
		if err := options[0].Value(&wk); err == nil {
			alg, ok := wk.alg.(jwa.SignatureAlgorithm)
			if !ok {
				return nil, fmt.Errorf(`jwt.Sign: invalid algorithm type %T. jwa.SignatureAlgorithm is required`, wk.alg)
			}

			// Reject algorithm names that would require JSON escaping
			// in the protected header. Unlike kid (which may be attacker-
			// influenced and silently falls through to jws.Sign), an
			// unsafe alg is almost certainly a caller bug or an injection
			// attempt, so we fail fast rather than emit any signature.
			if !fastPathAlgSafe(alg.String()) {
				return nil, fmt.Errorf(`jwt.Sign: algorithm %q contains bytes that require JSON escaping`, alg.String())
			}

			// Check if option contains anything other than alg/key
			if len(wk.options) == 0 {
				// If the key carries a kid that would require JSON escaping,
				// skip the fast path (which concatenates kid raw into the
				// protected header) and fall through to jws.Sign.
				fastSafe := true
				if jwkKey, ok := wk.key.(jwk.Key); ok {
					if v, ok := jwkKey.KeyID(); ok && !fastPathKidSafe(v) {
						fastSafe = false
					}
				}
				if fastSafe {
					// yay, we have something we can put in the FAST PATH!
					return signFast(t, alg, wk.key)
				}
			}
			// fallthrough
		}
		// fallthrough
	}

	var soptions []jws.SignOption
	if l := len(options); l > 0 {
		// we need to from SignOption to Option because ... reasons
		// (todo: when go1.18 prevails, use type parameters
		rawoptions := make([]Option, l)
		for i, option := range options {
			rawoptions[i] = option
		}

		converted, err := toSignOptions(rawoptions...)
		if err != nil {
			return nil, fmt.Errorf(`jwt.Sign: failed to convert options into jws.SignOption: %w`, err)
		}
		soptions = converted
	}
	return NewSerializer().sign(soptions...).Serialize(t)
}

// Equal compares two JWT tokens. Do not use `reflect.Equal` or the like
// to compare tokens as they will also compare extra detail such as
// sync.Mutex objects used to control concurrent access.
//
// The comparison for values is currently done using a simple equality ("=="),
// except for time.Time, which uses time.Equal after dropping the monotonic
// clock and truncating the values to 1 second accuracy.
//
// if both t1 and t2 are nil, returns true
func Equal(t1, t2 Token) bool {
	if t1 == nil && t2 == nil {
		return true
	}

	// we already checked for t1 == t2 == nil, so safe to do this
	if t1 == nil || t2 == nil {
		return false
	}

	j1, err := json.Marshal(t1)
	if err != nil {
		return false
	}

	j2, err := json.Marshal(t2)
	if err != nil {
		return false
	}

	return bytes.Equal(j1, j2)
}

func (t *stdToken) Clone() (Token, error) {
	dst := New()

	dst.Options().Set(*(t.Options()))
	for _, k := range t.Keys() {
		var v any
		if err := t.Get(k, &v); err != nil {
			return nil, fmt.Errorf(`jwt.Clone: failed to get %s: %w`, k, err)
		}
		if err := dst.Set(k, v); err != nil {
			return nil, fmt.Errorf(`jwt.Clone failed to set %s: %w`, k, err)
		}
	}
	return dst, nil
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
//	jwt.RegisterCustomField(`x-birthday`, time.Time{})
//
// Then you can use a `time.Time` variable to extract the value
// of `x-birthday` field, instead of having to use `any`
// and later convert it to `time.Time`
//
//	var bday time.Time
//	_ = token.Get(`x-birthday`, &bday)
//
// If you need a more fine-tuned control over the decoding process,
// you can register a `CustomDecoder`. For example, below shows
// how to register a decoder that can parse RFC822 format string:
//
//	jwt.RegisterCustomField(`x-birthday`, jwt.CustomDecodeFunc(func(data []byte) (any, error) {
//	  return time.Parse(time.RFC822, string(data))
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

func getDefaultTruncation() time.Duration {
	return time.Duration(defaultTruncation.Load())
}
