package jws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/lestrrat-go/blackmagic"
	"github.com/lestrrat-go/jwx/v3/internal/base64"
	"github.com/lestrrat-go/jwx/v3/internal/json"
	"github.com/lestrrat-go/jwx/v3/internal/pool"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws/jwsbb"
)

// verifyContext holds the state during JWS verification
type verifyContext struct {
	parseOptions       []ParseOption
	dst                *Message
	detachedPayload    []byte
	payloadReader      io.Reader
	keyProviders       []KeyProvider
	keyUsed            any
	validateKey        bool
	critValidation     bool
	criticalExtensions []string
	encoder            Base64Encoder
	//nolint:containedctx
	ctx context.Context
}

var verifyContextPool = pool.New[*verifyContext](allocVerifyContext, freeVerifyContext)

func allocVerifyContext() *verifyContext {
	return &verifyContext{
		encoder: base64.DefaultEncoder(),
		ctx:     context.Background(),
	}
}

func freeVerifyContext(vc *verifyContext) *verifyContext {
	vc.parseOptions = vc.parseOptions[:0]
	vc.dst = nil
	vc.detachedPayload = nil
	vc.payloadReader = nil
	vc.keyProviders = vc.keyProviders[:0]
	vc.keyUsed = nil
	vc.validateKey = false
	vc.critValidation = false
	vc.criticalExtensions = vc.criticalExtensions[:0]
	vc.encoder = base64.DefaultEncoder()
	vc.ctx = context.Background()
	return vc
}

func (vc *verifyContext) ProcessOptions(options []VerifyOption) error {
	//nolint:forcetypeassert
	for _, option := range options {
		switch option.Ident() {
		case identMessage{}:
			if err := option.Value(&vc.dst); err != nil {
				return makeVerifyError(`invalid value for option WithMessage: %w`, err)
			}
		case identDetachedPayload{}:
			if vc.payloadReader != nil {
				return makeVerifyError(`jws.WithDetachedPayload() and jws.WithDetachedPayloadReader() are mutually exclusive`)
			}
			if err := option.Value(&vc.detachedPayload); err != nil {
				return makeVerifyError(`invalid value for option WithDetachedPayload: %w`, err)
			}
			// RFC 7797 "b64" auto-declaration. Detached-payload
			// verification is the canonical use case for b64=false,
			// and the jws package implements b64=false handling
			// natively, so requiring callers to also pass
			// jws.WithCritExtension("b64") is busywork. We declare
			// it implicitly here so application code stays focused
			// on its own crit extensions. This does not relax any
			// other validateCritical check — the b64 header still
			// has to appear in the protected header, the crit list
			// still has to be non-empty / no duplicates / no
			// standard names, etc. Only the "is in the caller's
			// allowlist" check is short-circuited for "b64", and
			// only when WithDetachedPayload was passed.
			vc.criticalExtensions = append(vc.criticalExtensions, "b64")
		case identDetachedPayloadReader{}:
			if vc.detachedPayload != nil {
				return makeVerifyError(`jws.WithDetachedPayload() and jws.WithDetachedPayloadReader() are mutually exclusive`)
			}
			if err := option.Value(&vc.payloadReader); err != nil {
				return makeVerifyError(`invalid value for option WithDetachedPayloadReader: %w`, err)
			}
			// Same RFC 7797 "b64" auto-declaration as for
			// identDetachedPayload; streaming is the other canonical
			// use case for b64=false.
			vc.criticalExtensions = append(vc.criticalExtensions, "b64")
		case identKey{}:
			var pair *withKey
			if err := option.Value(&pair); err != nil {
				return makeVerifyError(`invalid value for option WithKey: %w`, err)
			}

			alg, ok := pair.alg.(jwa.SignatureAlgorithm)
			if !ok {
				return makeVerifyError(`expected algorithm to be of type jwa.SignatureAlgorithm but got (%[1]q, %[1]T)`, pair.alg)
			}

			if err := validateAlgorithmForKey(alg, pair.key); err != nil {
				return makeVerifyError(`%w`, err)
			}

			vc.keyProviders = append(vc.keyProviders, &staticKeyProvider{
				alg: alg,
				key: pair.key,
			})
		case identKeyProvider{}:
			var kp KeyProvider
			if err := option.Value(&kp); err != nil {
				return makeVerifyError(`failed to retrieve key-provider option value: %w`, err)
			}
			vc.keyProviders = append(vc.keyProviders, kp)
		case identKeyUsed{}:
			if err := option.Value(&vc.keyUsed); err != nil {
				return makeVerifyError(`failed to retrieve key-used option value: %w`, err)
			}
		case identContext{}:
			if err := option.Value(&vc.ctx); err != nil {
				return makeVerifyError(`failed to retrieve context option value: %w`, err)
			}
		case identValidateKey{}:
			if err := option.Value(&vc.validateKey); err != nil {
				return makeVerifyError(`failed to retrieve validate-key option value: %w`, err)
			}
		case identCritValidation{}:
			if err := option.Value(&vc.critValidation); err != nil {
				return makeVerifyError(`failed to retrieve crit-validation option value: %w`, err)
			}
		case identCritExtension{}:
			var names []string
			if err := option.Value(&names); err != nil {
				return makeVerifyError(`failed to retrieve crit-extension option value: %w`, err)
			}
			vc.criticalExtensions = append(vc.criticalExtensions, names...)
		case identSerialization{}:
			vc.parseOptions = append(vc.parseOptions, option.(ParseOption))
		case identBase64Encoder{}:
			if err := option.Value(&vc.encoder); err != nil {
				return makeVerifyError(`failed to retrieve base64-encoder option value: %w`, err)
			}
		default:
			return makeVerifyError(`invalid jws.VerifyOption %q passed`, `With`+strings.TrimPrefix(fmt.Sprintf(`%T`, option.Ident()), `jws.ident`))
		}
	}

	if len(vc.keyProviders) < 1 {
		return makeVerifyError(`no key providers have been provided (see jws.WithKey(), jws.WithKeySet(), jws.WithVerifyAuto(), and jws.WithKeyProvider()`)
	}

	return nil
}

func (vc *verifyContext) VerifyMessage(buf []byte) ([]byte, error) {
	if vc.payloadReader != nil {
		return vc.verifyStreaming(buf)
	}

	msg, err := Parse(buf, vc.parseOptions...)
	if err != nil {
		return nil, makeVerifyError(`failed to parse jws: %w`, err)
	}
	defer msg.clearRaw()

	if vc.detachedPayload != nil {
		if len(msg.payload) != 0 {
			return nil, makeVerifyError(`can't specify detached payload for JWS with payload`)
		}

		msg.payload = vc.detachedPayload
	}

	verifyBuf := pool.ByteSlice().Get()

	// Because deferred functions bind to the current value of the variable,
	// we can't just use `defer pool.ByteSlice().Put(verifyBuf)` here.
	// Instead, we use a closure to reference the _variable_.
	// it would be better if we could call it directly, but there are
	// too many place we may return from this function
	defer func() {
		pool.ByteSlice().Put(verifyBuf)
	}()

	errs := pool.ErrorSlice().Get()
	defer func() {
		pool.ErrorSlice().Put(errs)
	}()
	for idx, sig := range msg.signatures {
		// Honor caller's deadline between signatures. Without this
		// check, a hostile JWS with many signatures keeps the loop
		// running long after the deadline; only kp.FetchKeys had
		// visibility into vc.ctx, and not every key provider observes
		// it. Cheap (~1ns) on the success path.
		if err := vc.ctx.Err(); err != nil {
			return nil, makeVerifyError(`%w`, err)
		}

		var rawHeaders []byte
		if rbp, ok := sig.protected.(interface{ rawBuffer() []byte }); ok {
			if raw := rbp.rawBuffer(); raw != nil {
				rawHeaders = raw
			}
		}

		if rawHeaders == nil {
			protected, err := json.Marshal(sig.protected)
			if err != nil {
				return nil, makeVerifyError(`failed to marshal "protected" for signature #%d: %w`, idx+1, err)
			}
			rawHeaders = protected
		}

		if vc.critValidation {
			if err := validateB64InCritIfFalse(sig.protected); err != nil {
				errs = append(errs, makeVerifyError(`signature #%d: %w`, idx+1, err))
				continue
			}
			if err := validateCritical(sig.protected, vc.criticalExtensions); err != nil {
				errs = append(errs, makeVerifyError(`signature #%d has invalid "crit" header: %w`, idx+1, err))
				continue
			}
		}

		verifyBuf = verifyBuf[:0]
		verifyBuf = jwsbb.SignBuffer(verifyBuf, rawHeaders, msg.payload, vc.encoder, msg.b64)
		keysAttempted := 0
		for i, kp := range vc.keyProviders {
			// Honor caller's deadline between key providers.
			if err := vc.ctx.Err(); err != nil {
				return nil, makeVerifyError(`%w`, err)
			}

			var sink algKeySink
			if err := kp.FetchKeys(vc.ctx, &sink, sig, msg); err != nil {
				errs = append(errs, makeVerifyError(`signature #%d: key provider %d failed: %w`, idx+1, i, err))
				continue
			}

			for _, pair := range sink.list {
				// Honor caller's deadline between (alg,key) pairs.
				// Under WithRequireKid(false) + WithInferAlgorithmFromKey(true)
				// + a large JWKS, this inner loop is the dominant
				// cost — checking ctx between attempts caps the
				// post-deadline crypto work at one operation.
				if err := vc.ctx.Err(); err != nil {
					return nil, makeVerifyError(`%w`, err)
				}

				alg := pair.alg
				key := pair.key
				keysAttempted++

				if err := vc.tryKey(verifyBuf, alg, key, msg, sig); err != nil {
					errs = append(errs, makeVerifyError(`failed to verify signature #%d with key %T: %w`, idx+1, key, err))
					continue
				}

				return msg.payload, nil
			}
		}
		if keysAttempted == 0 {
			errs = append(errs, makeVerifyError(`signature #%d: no matching keys were provided by any key provider`, idx+1))
		} else if looseOpts := vc.namedLooseKeySetOptions(); len(looseOpts) > 0 && keysAttempted > 1 {
			// When a loose keySet config widened the candidate set, name
			// the option(s) so the operator can see why a single Verify
			// call paid N× the cost — the un-attributed message gets
			// mis-diagnosed by adding more keys instead of fixing the
			// JWS or tightening the config.
			errs = append(errs, makeVerifyError(
				`signature #%d: tried %d (alg,key) pair(s) but none verified successfully; %s widened the candidate set`,
				idx+1, keysAttempted, strings.Join(looseOpts, " and ")))
		} else {
			errs = append(errs, makeVerifyError(`signature #%d: tried %d key(s) but none verified successfully`, idx+1, keysAttempted))
		}
	}
	return nil, makeVerifyError(`could not verify message using any of the signatures or keys: %w`, errors.Join(errs...))
}

func (vc *verifyContext) tryKey(verifyBuf []byte, alg jwa.SignatureAlgorithm, key any, msg *Message, sig *Signature) error {
	// Reject placeholders before any verifier — including a custom
	// Verifier2 — can see them. A custom KeyProvider can sink an
	// (alg, key) pair directly, bypassing keySetProvider.selectKey and
	// validateAlgorithmForKey, so this is the last chokepoint before
	// key material is used.
	if uk, ok := key.(jwk.UnsupportedKey); ok {
		return unsupportedKeyError(uk, `signature verification`)
	}

	if vc.validateKey {
		if err := validateKeyBeforeUse(key); err != nil {
			return fmt.Errorf(`failed to validate key before verification: %w`, err)
		}
	}

	verifier, err := VerifierFor(alg)
	if err != nil {
		return fmt.Errorf(`failed to get verifier for algorithm %q: %w`, alg, err)
	}

	if err := verifier.Verify(key, verifyBuf, sig.signature); err != nil {
		return verificationError{err}
	}

	// Verification succeeded
	if vc.keyUsed != nil {
		if err := blackmagic.AssignIfCompatible(vc.keyUsed, key); err != nil {
			return fmt.Errorf(`failed to assign used key (%T) to %T: %w`, key, vc.keyUsed, err)
		}
	}

	if vc.dst != nil {
		*(vc.dst) = *msg
	}

	return nil
}

// validateB64InCritIfFalse enforces RFC 7797 §3: producers that set
// b64=false in the protected header MUST also list "b64" in the protected
// header's "crit" array. The check runs alongside (and before)
// validateCritical so a non-conformant b64=false JWS is rejected up front
// regardless of whether the caller has supplied a crit allowlist via
// jws.WithCritExtension. Without this check, jws.Verify silently honors
// b64=false on the wire and computes its signing input differently from a
// strictly conformant verifier — exactly the cross-implementation
// disagreement RFC 7797 §6 was designed to prevent. VerifyCompactFast
// rejects any b64-bearing message outright via jws.ErrB64Present(); this
// helper is the slow-path mirror that targets only the non-conformant
// shape rather than blanket-refusing b64=false.
func validateB64InCritIfFalse(protected Headers) error {
	if getB64Value(protected) {
		return nil
	}
	if !protected.Has(CriticalKey) {
		return makeVerifyError(`protected header has "b64":false but no "crit"; RFC 7797 §3 requires producers that set "b64":false to list "b64" in "crit"`)
	}
	crit, _ := protected.Critical()
	if !slices.Contains(crit, "b64") {
		return makeVerifyError(`protected header has "b64":false but "crit" does not list "b64"; RFC 7797 §3 requires producers that set "b64":false to list "b64" in "crit"`)
	}
	return nil
}

// validateCritical checks the "crit" header per RFC 7515 Section 4.1.11.
// It enforces:
//   - the list is non-empty
//   - no entry is the empty string
//   - no entry duplicates another
//   - no entry names a standard JOSE header parameter
//   - every entry appears as a header parameter in the protected header
//   - every entry is in the caller-supplied allowedExtensions allowlist
//
// The last check is the central RFC requirement: recipients MUST reject
// any "crit" extension they do not understand, and the only way the
// library knows which extensions the caller understands is via the
// allowlist (populated from jws.WithCritExtension()).
//
// As a convenience, the RFC 7797 "b64" extension is auto-declared into
// allowedExtensions whenever the caller passes jws.WithDetachedPayload
// — see the identDetachedPayload case in ProcessOptions. The auto-
// declaration only short-circuits the allowlist check; every other
// rule above still applies to the "b64" entry.
func validateCritical(protected Headers, allowedExtensions []string) error {
	if !protected.Has(CriticalKey) {
		return nil
	}

	crit, _ := protected.Critical()
	if len(crit) == 0 {
		return makeVerifyError(`"crit" header must not be empty`)
	}

	seen := make(map[string]struct{}, len(crit))
	for _, name := range crit {
		if name == "" {
			return makeVerifyError(`"crit" header must not contain an empty extension name`)
		}
		if _, dup := seen[name]; dup {
			return makeVerifyError(`"crit" header must not contain duplicate extension %q`, name)
		}
		seen[name] = struct{}{}

		// RFC 7515 Section 4.1.11: "crit" MUST NOT include names defined
		// by the JOSE Header specification itself. The "b64" parameter
		// is RFC 7797, not RFC 7515 — listing it in "crit" is the
		// canonical use of the field per RFC 7797 §3 — so exclude it
		// from this check even though it is a typed field on stdHeaders.
		if name != B64Key && slices.Contains(stdHeaderNames, name) {
			return makeVerifyError(`"crit" header must not contain standard header parameter %q`, name)
		}

		// The extension must be present in the protected header.
		if !protected.Has(name) {
			return makeVerifyError(`"crit" header references extension %q, but it is not present in the protected header`, name)
		}

		// The recipient must have declared support for the extension.
		if !slices.Contains(allowedExtensions, name) {
			if name == B64Key {
				// b64=false is the canonical RFC 7797 case. The
				// auto-declare only fires for WithDetachedPayload /
				// WithDetachedPayloadReader; in-band b64=false still
				// requires the caller to opt in explicitly.
				return makeVerifyError(`"crit" header references extension "b64", but the recipient has not declared support for it; pass jws.WithCritExtension("b64") to accept in-band b64=false (auto-declare only fires for jws.WithDetachedPayload / jws.WithDetachedPayloadReader)`)
			}
			return makeVerifyError(`"crit" header references extension %q, but the recipient has not declared support for it (use jws.WithCritExtension(%q))`, name, name)
		}
	}

	return nil
}

// namedLooseKeySetOptions inspects the registered key providers and
// returns the human-readable names of the loose-config keySet options
// in effect for this verify call: jws.WithRequireKid(false) and/or
// jws.WithInferAlgorithmFromKey(true). These are the options whose
// presence widens the per-signature (alg,key) candidate set beyond
// the default "kid + alg pin" of one. The names are used in the final
// "could not verify" error so an operator sees which options produced
// the fan-out without grep'ing the source.
func (vc *verifyContext) namedLooseKeySetOptions() []string {
	var requireKidFalse, inferAlgorithm bool
	for _, kp := range vc.keyProviders {
		ksp, ok := kp.(*keySetProvider)
		if !ok {
			continue
		}
		if !ksp.requireKid {
			requireKidFalse = true
		}
		if ksp.inferAlgorithm {
			inferAlgorithm = true
		}
	}
	var names []string
	if requireKidFalse {
		names = append(names, "jws.WithRequireKid(false)")
	}
	if inferAlgorithm {
		names = append(names, "jws.WithInferAlgorithmFromKey(true)")
	}
	return names
}
