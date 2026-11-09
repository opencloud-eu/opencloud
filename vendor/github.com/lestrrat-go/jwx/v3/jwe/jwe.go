//go:generate ../tools/cmd/genjwe.sh

// Package jwe implements JWE as described in https://tools.ietf.org/html/rfc7516.
//
// Legacy note: RSA-PKCS1 v1.5 key encryption (`jwa.RSA1_5()`) is supported
// only for interoperability with existing peers. New applications should
// prefer an RSA-OAEP variant such as `jwa.RSA_OAEP_256()` because PKCS#1 v1.5
// decryption is exposed to Bleichenbacher-style oracle attacks.
package jwe

// #region imports
import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"sync/atomic"

	"github.com/lestrrat-go/blackmagic"
	"github.com/lestrrat-go/jwx/v3/internal/base64"
	"github.com/lestrrat-go/jwx/v3/internal/json"
	"github.com/lestrrat-go/jwx/v3/internal/pool"
	"github.com/lestrrat-go/jwx/v3/internal/tokens"
	"github.com/lestrrat-go/jwx/v3/jwk"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwe/internal/aescbc"
	"github.com/lestrrat-go/jwx/v3/jwe/internal/content_crypt"
	"github.com/lestrrat-go/jwx/v3/jwe/internal/keygen"
)

// #region globals

var maxPBES2Count atomic.Int64
var minPBES2Count atomic.Int64
var pbes2Count atomic.Int64
var maxRecipients atomic.Int64
var maxDecompressBufferSize atomic.Int64
var disabledKeyAlgs atomic.Pointer[map[string]struct{}]

func init() {
	maxPBES2Count.Store(10000)
	minPBES2Count.Store(1000)
	pbes2Count.Store(int64(tokens.PBES2DefaultIterations))
	maxRecipients.Store(100)
	maxDecompressBufferSize.Store(10 * 1024 * 1024) // 10MB
}

func Settings(options ...GlobalOption) {
	for _, option := range options {
		switch option.Ident() {
		case identMaxPBES2Count{}:
			var v int
			if err := option.Value(&v); err != nil {
				panic(fmt.Sprintf("jwe.Settings: value for option WithMaxPBES2Count must be an int: %s", err))
			}
			maxPBES2Count.Store(int64(v))
		case identMinPBES2Count{}:
			var v int
			if err := option.Value(&v); err != nil {
				panic(fmt.Sprintf("jwe.Settings: value for option WithMinPBES2Count must be an int: %s", err))
			}
			minPBES2Count.Store(int64(v))
		case identPBES2Count{}:
			var v int
			if err := option.Value(&v); err != nil {
				panic(fmt.Sprintf("jwe.Settings: value for option WithPBES2Count must be an int: %s", err))
			}
			if v <= 0 {
				v = tokens.PBES2DefaultIterations
			}
			pbes2Count.Store(int64(v))
		case identMaxRecipients{}:
			var v int
			if err := option.Value(&v); err != nil {
				panic(fmt.Sprintf("jwe.Settings: value for option WithMaxRecipients must be an int: %s", err))
			}
			maxRecipients.Store(int64(v))
		case identMaxDecompressBufferSize{}:
			var v int64
			if err := option.Value(&v); err != nil {
				panic(fmt.Sprintf("jwe.Settings: value for option WithMaxDecompressBufferSize must be an int64: %s", err))
			}
			maxDecompressBufferSize.Store(v)
		case identCBCBufferSize{}:
			var v int64
			if err := option.Value(&v); err != nil {
				panic(fmt.Sprintf("jwe.Settings: value for option WithCBCBufferSize must be an int64: %s", err))
			}
			aescbc.SetMaxBufferSize(v)
		case identDisabledKeyAlgorithms{}:
			var algs []jwa.KeyEncryptionAlgorithm
			if err := option.Value(&algs); err != nil {
				panic(fmt.Sprintf("jwe.Settings: value for option WithDisabledKeyAlgorithms must be []jwa.KeyEncryptionAlgorithm: %s", err))
			}
			if len(algs) == 0 {
				disabledKeyAlgs.Store(nil)
				continue
			}
			m := make(map[string]struct{}, len(algs))
			for _, alg := range algs {
				m[alg.String()] = struct{}{}
			}
			disabledKeyAlgs.Store(&m)
		}
	}
}

// isKeyAlgorithmDisabled reports whether alg is in the global
// jwe.WithDisabledKeyAlgorithms set.
func isKeyAlgorithmDisabled(alg jwa.KeyEncryptionAlgorithm) bool {
	m := disabledKeyAlgs.Load()
	if m == nil {
		return false
	}
	_, ok := (*m)[alg.String()]
	return ok
}

const (
	fmtInvalid = iota
	fmtCompact
	fmtJSON
	fmtJSONPretty
	fmtMax
)

var registry = json.NewRegistry()

type recipientBuilder struct {
	alg        jwa.KeyEncryptionAlgorithm
	key        any
	headers    Headers
	pbes2Count int
}

func (b *recipientBuilder) Build(r Recipient, cek []byte, calg jwa.ContentEncryptionAlgorithm, _ *content_crypt.Generic) ([]byte, error) {
	if isKeyAlgorithmDisabled(b.alg) {
		return nil, fmt.Errorf(`jwe.Encrypt: key encryption algorithm %q is disabled by jwe.WithDisabledKeyAlgorithms`, b.alg)
	}
	// we need the raw key for later use
	rawKey := b.key

	var keyID string
	if ke, ok := b.key.(KeyEncrypter); ok {
		if kider, ok := ke.(KeyIDer); ok {
			if v, ok := kider.KeyID(); ok {
				keyID = v
			}
		}
	} else if jwkKey, ok := b.key.(jwk.Key); ok {
		// Meanwhile, grab the kid as well
		if v, ok := jwkKey.KeyID(); ok {
			keyID = v
		}

		var raw any
		if err := jwk.Export(jwkKey, &raw); err != nil {
			return nil, fmt.Errorf(`jwe.Encrypt: recipientBuilder: failed to retrieve raw key out of %T: %w`, b.key, err)
		}

		rawKey = raw
	}

	// Extract ECDH-ES specific parameters if needed.
	var apu, apv []byte

	hdr := b.headers
	if hdr == nil {
		hdr = r.Headers()
	}

	if val, ok := hdr.AgreementPartyUInfo(); ok {
		apu = val
	}

	if val, ok := hdr.AgreementPartyVInfo(); ok {
		apv = val
	}

	// Create the encrypter using the new jwebb pattern
	enc := newEncrypter(b.alg, calg, b.key, rawKey, apu, apv, b.pbes2Count)

	_ = r.SetHeaders(hdr)

	// Populate headers with stuff that we automatically set
	if err := hdr.Set(AlgorithmKey, b.alg); err != nil {
		return nil, fmt.Errorf(`failed to set header: %w`, err)
	}

	if keyID != "" {
		if err := hdr.Set(KeyIDKey, keyID); err != nil {
			return nil, fmt.Errorf(`failed to set header: %w`, err)
		}
	}

	// Handle the encrypted key
	var rawCEK []byte
	enckey, err := enc.EncryptKey(cek)
	if err != nil {
		return nil, fmt.Errorf(`failed to encrypt key: %w`, err)
	}
	if b.alg == jwa.ECDH_ES() || b.alg == jwa.DIRECT() {
		rawCEK = enckey.Bytes()
	} else {
		if err := r.SetEncryptedKey(enckey.Bytes()); err != nil {
			return nil, fmt.Errorf(`failed to set encrypted key: %w`, err)
		}
	}

	// finally, anything specific should go here
	if hp, ok := enckey.(populater); ok {
		if err := hp.Populate(hdr); err != nil {
			return nil, fmt.Errorf(`failed to populate: %w`, err)
		}
	}

	return rawCEK, nil
}

// Encrypt generates a JWE message for the given payload and returns
// it in serialized form, which can be in either compact or
// JSON format. Default is compact. When JSON format is specified and
// there is only one recipient, the resulting serialization is
// automatically converted to flattened JSON serialization format.
//
// You must pass at least one key to `jwe.Encrypt()` by using `jwe.WithKey()`
// option.
//
//	jwe.Encrypt(payload, jwe.WithKey(alg, key))
//	jwe.Encrypt(payload, jwe.WithJSON(), jwe.WithKey(alg1, key1), jwe.WithKey(alg2, key2))
//
// Note that in the second example the `jwe.WithJSON()` option is
// specified as well. This is because the compact serialization
// format does not support multiple recipients, and users must
// specifically ask for the JSON serialization format.
//
// Read the documentation for `jwe.WithKey()` to learn more about the
// possible values that can be used for `alg` and `key`.
//
// `jwa.RSA1_5()` is supported only for interoperability with legacy peers.
// New applications should prefer an RSA-OAEP variant such as
// `jwa.RSA_OAEP_256()` because PKCS#1 v1.5 decryption is exposed to
// Bleichenbacher-style oracle attacks.
// If you enable `jwe.WithCompress()`, this library does not enforce a
// producer-side payload size limit before compression. Callers that accept
// untrusted or arbitrarily large plaintext must bound the input size before
// calling `jwe.Encrypt()`. Recipients may also reject compressed messages
// whose decompressed payload exceeds their `jwe.WithMaxDecompressBufferSize()`
// setting.
//
// Look for options that return `jwe.EncryptOption` or `jwe.EncryptDecryptOption`
// for a complete list of options that can be passed to this function.
//
// As of v3.0.12, users can specify `jwe.WithLegacyHeaderMerging()` to
// disable header merging behavior that was the default prior to v3.0.12.
// Read the documentation for `jwe.WithLegacyHeaderMerging()` for more information.
func Encrypt(payload []byte, options ...EncryptOption) ([]byte, error) {
	ec := encryptContextPool.Get()
	defer encryptContextPool.Put(ec)
	if err := ec.ProcessOptions(options); err != nil {
		return nil, makeEncryptError(`jwe.Encrypt`, `failed to process options: %w`, err)
	}
	ret, err := ec.EncryptMessage(payload, nil)
	if err != nil {
		return nil, makeEncryptError(`jwe.Encrypt`, `%w`, err)
	}
	return ret, nil
}

// EncryptStatic is exactly like Encrypt, except it accepts a static
// content encryption key (CEK). It is separated out from the main
// Encrypt function such that the latter does not accidentally use a static
// CEK.
//
// Unless `jwe.WithContentEncryption()` is provided, `EncryptStatic` uses
// `jwa.A256GCM()`, which requires a 32-byte CEK.
//
// The CEK used to encrypt the payload must match the selected content
// encryption algorithm:
//
//   - `jwa.A128GCM()`: 16 bytes
//   - `jwa.A192GCM()`: 24 bytes
//   - `jwa.A256GCM()`: 32 bytes
//   - `jwa.A128CBC_HS256()`: 32 bytes
//   - `jwa.A192CBC_HS384()`: 48 bytes
//   - `jwa.A256CBC_HS512()`: 64 bytes
//
// `EncryptStatic` validates the final CEK length before payload encryption
// and returns an error if it does not match the selected `enc` algorithm.
//
// NOTE: when the chosen key-encryption algorithm derives the CEK rather than
// wrapping it — specifically `jwa.DIRECT()` and bare `jwa.ECDH_ES()` (without
// a key-wrap suffix) — the `cek` argument supplied here is ignored for
// content encryption. In those modes the effective CEK is the shared/derived
// key produced by the `jwe.WithKey()` input, and the byte-length check
// described above is enforced against that derived CEK, not against the
// value passed as `cek`. To pin the CEK deterministically, pair
// `EncryptStatic` only with key-wrapping algorithms such as
// `jwa.RSA_OAEP()`, `jwa.A256KW()`, or `jwa.ECDH_ES_A256KW()`.
//
// DO NOT attempt to use this function unless you completely understand the
// security implications to using static CEKs. You have been warned.
//
// This function is currently considered EXPERIMENTAL, and is subject to
// future changes across minor/micro versions.
func EncryptStatic(payload, cek []byte, options ...EncryptOption) ([]byte, error) {
	if len(cek) <= 0 {
		return nil, makeEncryptError(`jwe.EncryptStatic`, `empty CEK`)
	}
	ec := encryptContextPool.Get()
	defer encryptContextPool.Put(ec)
	if err := ec.ProcessOptions(options); err != nil {
		return nil, makeEncryptError(`jwe.EncryptStatic`, `failed to process options: %w`, err)
	}
	ret, err := ec.EncryptMessage(payload, cek)
	if err != nil {
		return nil, makeEncryptError(`jwe.EncryptStatic`, `%w`, err)
	}
	return ret, nil
}

// decryptContext holds the state during JWE decryption, similar to JWS verifyContext
type decryptContext struct {
	keyProviders            []KeyProvider
	keyUsed                 any
	cek                     *[]byte
	dst                     *Message
	maxRecipients           int
	maxDecompressBufferSize int64
	maxPBES2Count           int
	minPBES2Count           int
	critValidation          bool
	criticalExtensions      []string
	//nolint:containedctx
	ctx context.Context
}

var decryptContextPool = pool.New(allocDecryptContext, freeDecryptContext)

func allocDecryptContext() *decryptContext {
	return &decryptContext{
		ctx: context.Background(),
	}
}

func freeDecryptContext(dc *decryptContext) *decryptContext {
	dc.keyProviders = dc.keyProviders[:0]
	dc.keyUsed = nil
	dc.cek = nil
	dc.dst = nil
	dc.maxRecipients = 0
	dc.maxDecompressBufferSize = 0
	dc.maxPBES2Count = 0
	dc.minPBES2Count = 0
	dc.critValidation = false
	dc.criticalExtensions = dc.criticalExtensions[:0]
	dc.ctx = context.Background()
	return dc
}

func (dc *decryptContext) ProcessOptions(options []DecryptOption) error {
	dc.maxRecipients = int(maxRecipients.Load())
	dc.maxDecompressBufferSize = maxDecompressBufferSize.Load()
	dc.maxPBES2Count = int(maxPBES2Count.Load())
	dc.minPBES2Count = int(minPBES2Count.Load())

	for _, option := range options {
		switch option.Ident() {
		case identMessage{}:
			if err := option.Value(&dc.dst); err != nil {
				return fmt.Errorf("jwe.decrypt: WithMessage must be a *jwe.Message: %w", err)
			}
		case identKeyProvider{}:
			var kp KeyProvider
			if err := option.Value(&kp); err != nil {
				return fmt.Errorf("jwe.decrypt: WithKeyProvider must be a KeyProvider: %w", err)
			}
			dc.keyProviders = append(dc.keyProviders, kp)
		case identKeyUsed{}:
			if err := option.Value(&dc.keyUsed); err != nil {
				return fmt.Errorf("jwe.decrypt: WithKeyUsed must be an any: %w", err)
			}
		case identKey{}:
			var pair *withKey
			if err := option.Value(&pair); err != nil {
				return fmt.Errorf("jwe.decrypt: WithKey must be a *withKey: %w", err)
			}
			alg, ok := pair.alg.(jwa.KeyEncryptionAlgorithm)
			if !ok {
				return fmt.Errorf("jwe.decrypt: WithKey() option must be specified using jwa.KeyEncryptionAlgorithm (got %T)", pair.alg)
			}
			if err := validateAlgorithmForKey(alg, pair.key); err != nil {
				return fmt.Errorf("jwe.WithKey: %w", err)
			}
			dc.keyProviders = append(dc.keyProviders, &staticKeyProvider{alg: alg, key: pair.key})
		case identCEK{}:
			if err := option.Value(&dc.cek); err != nil {
				return fmt.Errorf("jwe.decrypt: WithCEK must be a *[]byte: %w", err)
			}
		case identMaxRecipients{}:
			if err := option.Value(&dc.maxRecipients); err != nil {
				return fmt.Errorf("jwe.decrypt: WithMaxRecipients must be int: %w", err)
			}
		case identMaxDecompressBufferSize{}:
			if err := option.Value(&dc.maxDecompressBufferSize); err != nil {
				return fmt.Errorf("jwe.decrypt: WithMaxDecompressBufferSize must be int64: %w", err)
			}
		case identMaxPBES2Count{}:
			if err := option.Value(&dc.maxPBES2Count); err != nil {
				return fmt.Errorf("jwe.decrypt: WithMaxPBES2Count must be int: %w", err)
			}
		case identMinPBES2Count{}:
			if err := option.Value(&dc.minPBES2Count); err != nil {
				return fmt.Errorf("jwe.decrypt: WithMinPBES2Count must be int: %w", err)
			}
		case identContext{}:
			if err := option.Value(&dc.ctx); err != nil {
				return fmt.Errorf("jwe.decrypt: WithContext must be a context.Context: %w", err)
			}
		case identCritValidation{}:
			if err := option.Value(&dc.critValidation); err != nil {
				return fmt.Errorf("jwe.decrypt: WithCritValidation must be a bool: %w", err)
			}
		case identCritExtension{}:
			var names []string
			if err := option.Value(&names); err != nil {
				return fmt.Errorf("jwe.decrypt: WithCritExtension must be a string: %w", err)
			}
			dc.criticalExtensions = append(dc.criticalExtensions, names...)
		}
	}

	if len(dc.keyProviders) < 1 {
		return fmt.Errorf(`jwe.Decrypt: no key providers have been provided (see jwe.WithKey(), jwe.WithKeySet(), and jwe.WithKeyProvider()`)
	}

	return nil
}

// validateCritical checks the "crit" header per RFC 7516 Section 4.1.13
// (which references RFC 7515 Section 4.1.11). It enforces:
//   - the list is non-empty
//   - no entry is the empty string
//   - no entry duplicates another
//   - no entry names a standard JOSE/JWE header parameter
//   - every entry appears as a header parameter in the protected header
//   - every entry is in the caller-supplied allowedExtensions allowlist
//
// The last check is the central RFC requirement: recipients MUST reject
// any "crit" extension they do not understand, and the only way the
// library knows which extensions the caller understands is via the
// allowlist (populated from jwe.WithCritExtension()).
func validateCritical(protected Headers, allowedExtensions []string) error {
	if !protected.Has(CriticalKey) {
		return nil
	}

	crit, _ := protected.Critical()
	if len(crit) == 0 {
		return makeDecryptError(`"crit" header must not be empty`)
	}

	seen := make(map[string]struct{}, len(crit))
	for _, name := range crit {
		if name == "" {
			return makeDecryptError(`"crit" header must not contain an empty extension name`)
		}
		if _, dup := seen[name]; dup {
			return makeDecryptError(`"crit" header must not contain duplicate extension %q`, name)
		}
		seen[name] = struct{}{}

		// RFC 7515 Section 4.1.11: "crit" MUST NOT include names defined
		// by the JOSE Header specification itself.
		if slices.Contains(stdHeaderNames, name) {
			return makeDecryptError(`"crit" header must not contain standard header parameter %q`, name)
		}

		// The extension must be present in the protected header.
		if !protected.Has(name) {
			return makeDecryptError(`"crit" header references extension %q, but it is not present in the protected header`, name)
		}

		// The recipient must have declared support for the extension.
		if !slices.Contains(allowedExtensions, name) {
			return makeDecryptError(`"crit" header references extension %q, but the recipient has not declared support for it (use jwe.WithCritExtension(%q))`, name, name)
		}
	}

	return nil
}

// concatAAD returns the AAD value used to seal or open a JWE payload:
// the protected-header segment, optionally followed by ASCII '.' and
// the caller-supplied external aad (RFC 7516 §5.1 step 14 / §5.2
// step 14). A fresh slice is always allocated so the caller's computed
// and aad slices are never appended into, which matters because
// computedAad often aliases a Message field whose backing array is
// still referenced elsewhere.
func concatAAD(computed, aad []byte) []byte {
	if len(aad) == 0 {
		return computed
	}
	out := make([]byte, len(computed)+1+len(aad))
	n := copy(out, computed)
	out[n] = tokens.Period
	copy(out[n+1:], aad)
	return out
}

func (dc *decryptContext) DecryptMessage(buf []byte) ([]byte, error) {
	msg, err := parseJSONOrCompact(buf, true, dc.maxRecipients)
	if err != nil {
		return nil, fmt.Errorf(`jwe.Decrypt: failed to parse buffer: %w`, err)
	}

	// Validate the "crit" header per RFC 7516 Section 4.1.13. The check
	// runs against the protected header only — RFC says "crit" MUST live
	// there — and short-circuits before any key-decrypt or content-decrypt
	// work happens.
	if dc.critValidation {
		if err := validateCritical(msg.protectedHeaders, dc.criticalExtensions); err != nil {
			return nil, err
		}
	}

	// Clone the shared (top-level) protected header as our working copy.
	// We deliberately do NOT merge msg.unprotectedHeaders (the shared,
	// top-level *unprotected* header) here: it is never covered by the
	// AEAD tag, so it must not contribute algorithm parameters.
	//
	// Per-recipient unprotected headers are a separate case — RFC 7516
	// §5.3 explicitly permits them to carry recipient-specific algorithm
	// parameters (alg, epk, p2s, p2c, iv, tag, apu, apv, …), and
	// decryptContent merges recipient.Headers() onto this base below.
	// That merge is bounded by WithMaxRecipients and, for PBES2, by
	// WithMaxPBES2Count (applied per recipient).
	h, err := msg.protectedHeaders.Clone()
	if err != nil {
		return nil, fmt.Errorf(`jwe.Decrypt: failed to copy protected headers: %w`, err)
	}

	var aad []byte
	if aadContainer := msg.authenticatedData; aadContainer != nil {
		aad = base64.Encode(aadContainer)
	}

	var computedAad []byte
	if len(msg.rawProtectedHeaders) > 0 {
		computedAad = msg.rawProtectedHeaders
	} else {
		// this is probably not required once msg.Decrypt is deprecated
		var err error
		computedAad, err = msg.protectedHeaders.Encode()
		if err != nil {
			return nil, fmt.Errorf(`jwe.Decrypt: failed to encode protected headers: %w`, err)
		}
	}

	// for each recipient, attempt to match the key providers
	// if we have no recipients, pretend like we only have one
	recipients := msg.recipients
	if len(recipients) == 0 {
		r := NewRecipient()
		if err := r.SetHeaders(msg.protectedHeaders); err != nil {
			return nil, fmt.Errorf(`jwe.Decrypt: failed to set headers to recipient: %w`, err)
		}
		recipients = append(recipients, r)
	}

	errs := make([]error, 0, len(recipients))
	for _, recipient := range recipients {
		// Honor caller's deadline between recipients. Symmetric with
		// the per-keyProvider and per-(alg,key) checks in tryRecipient.
		if err := dc.ctx.Err(); err != nil {
			return nil, makeDecryptError(`%w`, err)
		}

		decrypted, err := dc.tryRecipient(msg, recipient, h, aad, computedAad)
		if err != nil {
			errs = append(errs, makeRecipientError(err))
			continue
		}
		if dc.dst != nil {
			*dc.dst = *msg
			dc.dst.rawProtectedHeaders = nil
			dc.dst.storeProtectedHeaders = false
		}
		return decrypted, nil
	}
	// Bound the joined-error count so a hostile JWE with many recipients
	// can't produce an unbounded error string. Keep the first
	// decryptErrorJoinCap entries verbatim and replace the rest with a
	// single "... and N more" sentinel.
	return nil, fmt.Errorf(`jwe.Decrypt: failed to decrypt any of the recipients: %w`, joinDecryptErrors(errs))
}

// decryptErrorJoinCap caps how many per-recipient constituent errors
// get joined into the final Decrypt error so the resulting err.Error()
// can't grow unboundedly under a hostile multi-recipient JWE.
const decryptErrorJoinCap = 10

func joinDecryptErrors(errs []error) error {
	if len(errs) <= decryptErrorJoinCap {
		return errors.Join(errs...)
	}
	kept := make([]error, decryptErrorJoinCap, decryptErrorJoinCap+1)
	copy(kept, errs[:decryptErrorJoinCap])
	kept = append(kept, fmt.Errorf("... and %d more error(s) suppressed", len(errs)-decryptErrorJoinCap))
	return errors.Join(kept...)
}

func (dc *decryptContext) tryRecipient(msg *Message, recipient Recipient, protectedHeaders Headers, aad, computedAad []byte) ([]byte, error) {
	var tried int
	var lastError error
	for i, kp := range dc.keyProviders {
		// Honor caller's deadline between key providers.
		if err := dc.ctx.Err(); err != nil {
			return nil, err
		}

		var sink algKeySink
		if err := kp.FetchKeys(dc.ctx, &sink, recipient, msg); err != nil {
			return nil, fmt.Errorf(`key provider %d failed: %w`, i, err)
		}

		for _, pair := range sink.list {
			// Honor caller's deadline between (alg,key) pairs.
			if err := dc.ctx.Err(); err != nil {
				return nil, err
			}

			tried++
			// alg is converted here because pair.alg is of type jwa.KeyAlgorithm.
			// this may seem ugly, but we're trying to avoid declaring separate
			// structs for `alg jwa.KeyEncryptionAlgorithm` and `alg jwa.SignatureAlgorithm`
			//nolint:forcetypeassert
			alg := pair.alg.(jwa.KeyEncryptionAlgorithm)
			key := pair.key

			decrypted, err := dc.decryptContent(msg, alg, key, recipient, protectedHeaders, aad, computedAad)
			if err != nil {
				lastError = err
				continue
			}

			if dc.keyUsed != nil {
				if err := blackmagic.AssignIfCompatible(dc.keyUsed, key); err != nil {
					return nil, fmt.Errorf(`failed to assign used key (%T) to %T: %w`, key, dc.keyUsed, err)
				}
			}
			return decrypted, nil
		}
	}
	return nil, fmt.Errorf(`jwe.Decrypt: tried %d keys, but failed to match any of the keys with recipient (last error = %s)`, tried, lastError)
}

func (dc *decryptContext) decryptContent(msg *Message, alg jwa.KeyEncryptionAlgorithm, key any, recipient Recipient, protectedHeaders Headers, aad, computedAad []byte) ([]byte, error) {
	if isKeyAlgorithmDisabled(alg) {
		return nil, makeDecryptError(`key encryption algorithm %q is disabled by jwe.WithDisabledKeyAlgorithms`, alg)
	}
	if jwkKey, ok := key.(jwk.Key); ok {
		var raw any
		if err := jwk.Export(jwkKey, &raw); err != nil {
			return nil, fmt.Errorf(`failed to retrieve raw key from %T: %w`, key, err)
		}
		key = raw
	}

	ce, ok := msg.protectedHeaders.ContentEncryption()
	if !ok {
		return nil, fmt.Errorf(`jwe.Decrypt: failed to retrieve content encryption algorithm from protected headers`)
	}
	dec := newDecrypter(alg, ce, key).
		AuthenticatedData(aad).
		ComputedAuthenticatedData(computedAad).
		InitializationVector(msg.initializationVector).
		Tag(msg.tag).
		CEK(dc.cek)

	// RFC 7516 §7.2.1 requires header parameter names to be disjoint
	// across the protected, shared-unprotected, and per-recipient
	// header locations. For "alg" specifically, allowing protected
	// and per-recipient headers to declare conflicting values is an
	// algorithm-confusion vector: an attacker who can rewrite the
	// per-recipient (unprotected) location can claim a different alg
	// than the integrity-protected one, and the alg-match loop below
	// would silently break on whichever it sees first.
	//
	// Compact-form JWE legitimately has the same alg value in both
	// places — parseCompact synthesizes a per-recipient header by
	// cloning the protected header (minus enc), so a strict-disjoint
	// check would reject every compact JWE. We therefore allow the
	// duplication when the values agree, and reject only when they
	// disagree.
	if rh := recipient.Headers(); rh != nil {
		if recipAlg, recipHas := rh.Algorithm(); recipHas {
			if protectedAlg, protectedHas := protectedHeaders.Algorithm(); protectedHas && protectedAlg != recipAlg {
				return nil, makeDecryptError(`malformed JWE — "alg" header value differs between protected (%q) and per-recipient (%q) headers (RFC 7516 §7.2.1)`, protectedAlg, recipAlg)
			}
		}
	}

	// The "alg" header can be in either protected or per-recipient
	// headers. With disjointness enforced above, only one location can
	// have it, so iteration order does not affect security; we keep
	// per-recipient first to match the historical preference for
	// recipient-specific algs in multi-recipient JWE.
	var algMatched bool
	for _, hdr := range []Headers{recipient.Headers(), protectedHeaders} {
		v, ok := hdr.Algorithm()
		if !ok {
			continue
		}

		if v == alg {
			algMatched = true
			break
		}
		// if we found something but didn't match, it's a failure
		return nil, fmt.Errorf(`jwe.Decrypt: key (%q) and recipient (%q) algorithms do not match`, alg, v)
	}
	if !algMatched {
		return nil, fmt.Errorf(`jwe.Decrypt: failed to find "alg" header in either protected or per-recipient headers`)
	}

	h2, err := protectedHeaders.Clone()
	if err != nil {
		return nil, fmt.Errorf(`jwe.Decrypt: failed to copy headers (1): %w`, err)
	}

	h2, err = h2.Merge(recipient.Headers())
	if err != nil {
		return nil, fmt.Errorf(`failed to copy headers (2): %w`, err)
	}

	switch alg {
	case jwa.ECDH_ES(), jwa.ECDH_ES_A128KW(), jwa.ECDH_ES_A192KW(), jwa.ECDH_ES_A256KW():
		var epk any
		if err := h2.Get(EphemeralPublicKeyKey, &epk); err != nil {
			return nil, fmt.Errorf(`failed to get 'epk' field: %w`, err)
		}
		switch epk := epk.(type) {
		case jwk.ECDSAPublicKey:
			var pubkey ecdsa.PublicKey
			if err := jwk.Export(epk, &pubkey); err != nil {
				return nil, fmt.Errorf(`failed to get public key: %w`, err)
			}
			dec.PublicKey(&pubkey)
		case jwk.OKPPublicKey:
			var pubkey any
			if err := jwk.Export(epk, &pubkey); err != nil {
				return nil, fmt.Errorf(`failed to get public key: %w`, err)
			}
			dec.PublicKey(pubkey)
		default:
			return nil, fmt.Errorf("unexpected 'epk' type %T for alg %s", epk, alg)
		}

		if apu, ok := h2.AgreementPartyUInfo(); ok && len(apu) > 0 {
			dec.AgreementPartyUInfo(apu)
		}
		if apv, ok := h2.AgreementPartyVInfo(); ok && len(apv) > 0 {
			dec.AgreementPartyVInfo(apv)
		}
	case jwa.A128GCMKW(), jwa.A192GCMKW(), jwa.A256GCMKW():
		var ivB64 string
		if h2.Has(InitializationVectorKey) {
			if err := h2.Get(InitializationVectorKey, &ivB64); err != nil {
				return nil, fmt.Errorf(`field %q is not a string: %w`, InitializationVectorKey, err)
			}
			iv, err := base64.DecodeString(ivB64)
			if err != nil {
				return nil, fmt.Errorf(`failed to b64-decode 'iv': %w`, err)
			}
			dec.KeyInitializationVector(iv)
		}
		var tagB64 string
		if h2.Has(TagKey) {
			if err := h2.Get(TagKey, &tagB64); err != nil {
				return nil, fmt.Errorf(`field %q is not a string: %w`, TagKey, err)
			}
			tag, err := base64.DecodeString(tagB64)
			if err != nil {
				return nil, fmt.Errorf(`failed to b64-decode 'tag': %w`, err)
			}
			dec.KeyTag(tag)
		}
	case jwa.PBES2_HS256_A128KW(), jwa.PBES2_HS384_A192KW(), jwa.PBES2_HS512_A256KW():
		var saltB64 string
		if err := h2.Get(SaltKey, &saltB64); err != nil {
			return nil, fmt.Errorf(`failed to get %q field`, SaltKey)
		}

		// Parse p2c into int64 directly. Float64 cannot represent
		// integers above 2^53 exactly; comparing a parsed value
		// against a high MaxPBES2Count cap in float-space and then
		// casting via int(...) lets out-of-range values silently
		// round into the accepted range when callers raise the cap
		// past 2^53. int64 keeps the bound check exact.
		var count int64
		if json.UseNumber() {
			var n json.Number
			if err := h2.Get(CountKey, &n); err != nil {
				return nil, fmt.Errorf(`failed to get %q field`, CountKey)
			}
			c, err := n.Int64()
			if err != nil {
				return nil, fmt.Errorf(`invalid 'p2c' value: %q is not a valid integer: %w`, n.String(), err)
			}
			count = c
		} else {
			var v float64
			if err := h2.Get(CountKey, &v); err != nil {
				return nil, fmt.Errorf(`failed to get %q field`, CountKey)
			}
			if math.IsNaN(v) || math.IsInf(v, 0) || math.Trunc(v) != v {
				return nil, fmt.Errorf(`invalid 'p2c' value: not a positive integer (got %v)`, v)
			}
			// Use explicit float-domain bounds (2^63 / -2^63) so
			// the comparison is platform-independent and does not
			// go through math.MaxInt64's implicit conversion.
			const (
				int64MaxAsFloat = float64(1 << 63) // 2^63, smallest float > MaxInt64
				int64MinAsFloat = -int64MaxAsFloat // -2^63, exact float = MinInt64
			)
			if v >= int64MaxAsFloat || v < int64MinAsFloat {
				return nil, fmt.Errorf(`invalid 'p2c' value: not representable as int64 (got %v)`, v)
			}
			count = int64(v)
		}

		maxCount := dc.maxPBES2Count
		minCount := dc.minPBES2Count
		if count < int64(minCount) {
			return nil, fmt.Errorf(`invalid 'p2c' value: %d is below WithMinPBES2Count=%d (RFC 7518 §4.8.1.2 floor; loosen via jwe.WithMinPBES2Count)`, count, minCount)
		}
		if count > int64(maxCount) {
			return nil, fmt.Errorf(`invalid 'p2c' value: %d exceeds WithMaxPBES2Count=%d (DoS amplification cap; raise via jwe.WithMaxPBES2Count)`, count, maxCount)
		}
		salt, err := base64.DecodeString(saltB64)
		if err != nil {
			return nil, fmt.Errorf(`failed to b64-decode 'salt': %w`, err)
		}
		dec.KeySalt(salt)
		dec.KeyCount(int(count))
	}

	plaintext, err := dec.Decrypt(recipient, msg.cipherText, msg)
	if err != nil {
		return nil, fmt.Errorf(`jwe.Decrypt: decryption failed: %w`, err)
	}

	if v, ok := h2.Compression(); ok && v == jwa.Deflate() {
		buf, err := uncompress(plaintext, dc.maxDecompressBufferSize)
		if err != nil {
			return nil, fmt.Errorf(`jwe.Decrypt: failed to uncompress payload: %w`, err)
		}
		plaintext = buf
	}

	if plaintext == nil {
		return nil, fmt.Errorf(`failed to find matching recipient`)
	}

	return plaintext, nil
}

// encryptContext holds the state during JWE encryption, similar to JWS signContext
type encryptContext struct {
	calg                jwa.ContentEncryptionAlgorithm
	compression         jwa.CompressionAlgorithm
	format              int
	pbes2Count          int
	authenticatedData   []byte
	builders            []*recipientBuilder
	protected           Headers
	legacyHeaderMerging bool
}

var encryptContextPool = pool.New(allocEncryptContext, freeEncryptContext)

func allocEncryptContext() *encryptContext {
	return &encryptContext{
		calg:        jwa.A256GCM(),
		compression: jwa.NoCompress(),
		format:      fmtCompact,
	}
}

func freeEncryptContext(ec *encryptContext) *encryptContext {
	ec.calg = jwa.A256GCM()
	ec.compression = jwa.NoCompress()
	ec.format = fmtCompact
	ec.pbes2Count = 0
	ec.authenticatedData = nil
	ec.builders = ec.builders[:0]
	ec.protected = nil
	return ec
}

func (ec *encryptContext) ProcessOptions(options []EncryptOption) error {
	ec.legacyHeaderMerging = true
	ec.pbes2Count = int(pbes2Count.Load())
	var mergeProtected bool
	var useRawCEK bool
	for _, option := range options {
		switch option.Ident() {
		case identKey{}:
			var wk *withKey
			if err := option.Value(&wk); err != nil {
				return fmt.Errorf("jwe.encrypt: WithKey must be a *withKey: %w", err)
			}
			v, ok := wk.alg.(jwa.KeyEncryptionAlgorithm)
			if !ok {
				return fmt.Errorf("jwe.encrypt: WithKey() option must be specified using jwa.KeyEncryptionAlgorithm (got %T)", wk.alg)
			}
			if err := validateAlgorithmForKey(v, wk.key); err != nil {
				return fmt.Errorf("jwe.WithKey: %w", err)
			}
			if v == jwa.DIRECT() || v == jwa.ECDH_ES() {
				useRawCEK = true
			}
			ec.builders = append(ec.builders, &recipientBuilder{
				alg:     v,
				key:     wk.key,
				headers: wk.headers,
			})
		case identPBES2Count{}:
			var v int
			if err := option.Value(&v); err != nil {
				return fmt.Errorf("jwe.encrypt: WithPBES2Count must be int: %w", err)
			}
			if v > 0 {
				ec.pbes2Count = v
			}
		case identContentEncryptionAlgorithm{}:
			var c jwa.ContentEncryptionAlgorithm
			if err := option.Value(&c); err != nil {
				return err
			}
			ec.calg = c
		case identCompress{}:
			var comp jwa.CompressionAlgorithm
			if err := option.Value(&comp); err != nil {
				return err
			}
			ec.compression = comp
		case identAuthenticateData{}:
			var aad []byte
			if err := option.Value(&aad); err != nil {
				return err
			}
			ec.authenticatedData = aad
		case identMergeProtectedHeaders{}:
			var mp bool
			if err := option.Value(&mp); err != nil {
				return err
			}
			mergeProtected = mp
		case identProtectedHeaders{}:
			var hdrs Headers
			if err := option.Value(&hdrs); err != nil {
				return err
			}
			if !mergeProtected || ec.protected == nil {
				ec.protected = hdrs
			} else {
				merged, err := ec.protected.Merge(hdrs)
				if err != nil {
					return fmt.Errorf(`failed to merge headers: %w`, err)
				}
				ec.protected = merged
			}
		case identSerialization{}:
			var fmtOpt int
			if err := option.Value(&fmtOpt); err != nil {
				return err
			}
			ec.format = fmtOpt
		case identLegacyHeaderMerging{}:
			var v bool
			if err := option.Value(&v); err != nil {
				return err
			}
			ec.legacyHeaderMerging = v
		}
	}

	// We need to have at least one builder
	switch l := len(ec.builders); {
	case l == 0:
		return fmt.Errorf(`missing key encryption builders: use jwe.WithKey() to specify one`)
	case l > 1:
		if ec.format == fmtCompact {
			return fmt.Errorf(`cannot use compact serialization when multiple recipients exist (check the number of WithKey() argument, or use WithJSON())`)
		}
	}

	if len(ec.authenticatedData) > 0 && ec.format == fmtCompact {
		return fmt.Errorf(`cannot use compact serialization with external authenticated data (use WithJSON())`)
	}

	if useRawCEK {
		if len(ec.builders) != 1 {
			return fmt.Errorf(`multiple recipients for ECDH-ES/DIRECT mode are not supported`)
		}
	}

	return nil
}

var msgPool = pool.New(allocMessage, freeMessage)

func allocMessage() *Message {
	return &Message{
		recipients: make([]Recipient, 0, 1),
	}
}

func freeMessage(msg *Message) *Message {
	msg.cipherText = nil
	msg.initializationVector = nil
	if hdr := msg.protectedHeaders; hdr != nil {
		headerPool.Put(hdr)
	}
	msg.protectedHeaders = nil
	msg.unprotectedHeaders = nil
	msg.recipients = nil // reuse should be done elsewhere
	msg.authenticatedData = nil
	msg.tag = nil
	msg.rawProtectedHeaders = nil
	msg.storeProtectedHeaders = false
	return msg
}

var headerPool = pool.New(NewHeaders, freeHeaders)

func freeHeaders(h Headers) Headers {
	if c, ok := h.(interface{ clear() }); ok {
		c.clear()
	}
	return h
}

var recipientPool = pool.New(NewRecipient, freeRecipient)

func freeRecipient(r Recipient) Recipient {
	// Return the recipient's headers to headerPool and install a fresh
	// instance so the next recipientPool.Get() never hands out a
	// pointer the caller may still hold a reference to. This is safe
	// because WithPerRecipientHeaders clones the caller-supplied
	// Headers, so anything we receive here is already library-owned.
	if h := r.Headers(); h != nil {
		headerPool.Put(h)
		_ = r.SetHeaders(headerPool.Get())
	}

	if sr, ok := r.(*stdRecipient); ok {
		sr.encryptedKey = nil
	}
	return r
}

var recipientSlicePool = pool.NewSlicePool(allocRecipientSlice, freeRecipientSlice)

func allocRecipientSlice() []Recipient {
	return make([]Recipient, 0, 1)
}

func freeRecipientSlice(rs []Recipient) []Recipient {
	for _, r := range rs {
		recipientPool.Put(r)
	}
	return rs[:0]
}

func (ec *encryptContext) EncryptMessage(payload []byte, cek []byte) ([]byte, error) {
	// Get protected headers from pool and copy contents from context
	protected := headerPool.Get()
	if userSupplied := ec.protected; userSupplied != nil {
		ec.protected = nil // Clear from context
		if err := userSupplied.Copy(protected); err != nil {
			return nil, fmt.Errorf(`failed to copy protected headers: %w`, err)
		}
	}

	// There is exactly one content encrypter.
	contentcrypt, err := content_crypt.NewGeneric(ec.calg)
	if err != nil {
		return nil, fmt.Errorf(`failed to create AES encrypter: %w`, err)
	}

	// Generate CEK if not provided
	if len(cek) <= 0 {
		bk, err := keygen.Random(contentcrypt.KeySize())
		if err != nil {
			return nil, fmt.Errorf(`failed to generate key: %w`, err)
		}
		cek = bk.Bytes()
	}

	var useRawCEK bool
	for _, builder := range ec.builders {
		if builder.alg == jwa.DIRECT() || builder.alg == jwa.ECDH_ES() {
			useRawCEK = true
			break
		}
	}

	lbuilders := len(ec.builders)
	recipients := recipientSlicePool.GetCapacity(lbuilders)
	defer recipientSlicePool.Put(recipients)

	for i, builder := range ec.builders {
		builder.pbes2Count = ec.pbes2Count
		r := recipientPool.Get()
		defer recipientPool.Put(r)

		// some builders require hint from the contentcrypt object
		rawCEK, err := builder.Build(r, cek, ec.calg, contentcrypt)
		if err != nil {
			return nil, fmt.Errorf(`failed to create recipient #%d: %w`, i, err)
		}
		recipients = append(recipients, r)

		// Kinda feels weird, but if useRawCEK == true, we asserted earlier
		// that len(builders) == 1, so this is OK
		if useRawCEK {
			cek = rawCEK
		}
	}

	if len(cek) != contentcrypt.KeySize() {
		return nil, fmt.Errorf(`content encryption key length %d does not match enc %q (expected %d bytes)`, len(cek), ec.calg.String(), contentcrypt.KeySize())
	}

	if err := protected.Set(ContentEncryptionKey, ec.calg); err != nil {
		return nil, fmt.Errorf(`failed to set "enc" in protected header: %w`, err)
	}

	if ec.compression != jwa.NoCompress() {
		payload, err = compress(payload)
		if err != nil {
			return nil, fmt.Errorf(`failed to compress payload before encryption: %w`, err)
		}
		if err := protected.Set(CompressionKey, ec.compression); err != nil {
			return nil, fmt.Errorf(`failed to set "zip" in protected header: %w`, err)
		}
	}

	// fmtCompact does not have per-recipient headers, nor a "header" field.
	// In this mode, we're going to have to merge everything to the protected
	// header.
	if ec.format == fmtCompact {
		// We have already established that the number of builders is 1 in
		// ec.ProcessOptions(). But we're going to be pedantic
		if lbuilders != 1 {
			return nil, fmt.Errorf(`internal error: expected exactly one recipient builder (got %d)`, lbuilders)
		}

		// when we're using compact format, we can safely merge per-recipient
		// headers into the protected header, if any
		h, err := protected.Merge(recipients[0].Headers())
		if err != nil {
			return nil, fmt.Errorf(`failed to merge protected headers for compact serialization: %w`, err)
		}
		protected = h
		// per-recipient headers, if any, will be ignored in compact format
	} else {
		// If it got here, it's JSON (could be pretty mode, too).
		if lbuilders == 1 {
			// If it got here, then we're doing flattened JSON serialization.
			// In this mode, we should merge per-recipient headers into the protected header,
			// but we also need to make sure that the "header" field is reset so that
			// it does not contain the same fields as the protected header.
			//
			// However, old behavior was to merge per-recipient headers into the
			// protected header when there was only one recipient, AND leave the
			// original "header" field as is, so we need to support that for backwards compatibility.
			//
			// The legacy merging only takes effect when there is exactly one recipient.
			//
			// This behavior can be disabled by passing jwe.WithLegacyHeaderMerging(false)
			// If the user has explicitly asked for merging, do it
			h, err := protected.Merge(recipients[0].Headers())
			if err != nil {
				return nil, fmt.Errorf(`failed to merge protected headers for flattenend JSON format: %w`, err)
			}
			protected = h

			if !ec.legacyHeaderMerging {
				// Clear per-recipient headers, since they have been merged.
				// But we only do it when legacy merging is disabled.
				// Note: we should probably introduce a Reset() method in v4
				if err := recipients[0].SetHeaders(NewHeaders()); err != nil {
					return nil, fmt.Errorf(`failed to clear per-recipient headers after merging: %w`, err)
				}
			}
		}
	}

	protectedAAD, err := protected.Encode()
	if err != nil {
		return nil, fmt.Errorf(`failed to base64 encode protected headers: %w`, err)
	}

	contentAAD := concatAAD(protectedAAD, base64.Encode(ec.authenticatedData))
	iv, ciphertext, tag, err := contentcrypt.Encrypt(cek, payload, contentAAD)
	if err != nil {
		return nil, fmt.Errorf(`failed to encrypt payload: %w`, err)
	}

	// Fast path for compact serialization: assemble directly from
	// pre-encoded headers and raw fields, avoiding the full Message
	// construction and redundant header re-encoding that Compact() does.
	if ec.format == fmtCompact {
		return compactSerialize(protectedAAD, recipients[0].EncryptedKey(), iv, ciphertext, tag), nil
	}

	msg := msgPool.Get()
	defer msgPool.Put(msg)

	if err := msg.Set(CipherTextKey, ciphertext); err != nil {
		return nil, fmt.Errorf(`failed to set %s: %w`, CipherTextKey, err)
	}
	if err := msg.Set(InitializationVectorKey, iv); err != nil {
		return nil, fmt.Errorf(`failed to set %s: %w`, InitializationVectorKey, err)
	}
	if err := msg.Set(ProtectedHeadersKey, protected); err != nil {
		return nil, fmt.Errorf(`failed to set %s: %w`, ProtectedHeadersKey, err)
	}
	if err := msg.Set(RecipientsKey, recipients); err != nil {
		return nil, fmt.Errorf(`failed to set %s: %w`, RecipientsKey, err)
	}
	if err := msg.Set(TagKey, tag); err != nil {
		return nil, fmt.Errorf(`failed to set %s: %w`, TagKey, err)
	}
	if len(ec.authenticatedData) > 0 {
		if err := msg.Set(AuthenticatedDataKey, ec.authenticatedData); err != nil {
			return nil, fmt.Errorf(`failed to set %s: %w`, AuthenticatedDataKey, err)
		}
	}

	switch ec.format {
	case fmtJSON:
		return json.Marshal(msg)
	case fmtJSONPretty:
		return json.MarshalIndent(msg, "", "  ")
	default:
		return nil, fmt.Errorf(`invalid serialization`)
	}
}

// Decrypt takes encrypted payload, and information required to decrypt the
// payload (e.g. the key encryption algorithm and the corresponding
// key to decrypt the JWE message) in its optional arguments. See
// the examples and list of options that return a DecryptOption for possible
// values. Upon successful decryption returns the decrypted payload.
//
// The JWE message can be either compact or full JSON format.
//
// When using `jwe.WithKey()`, you can pass a `jwa.KeyAlgorithm`
// for convenience: this is mainly to allow you to directly pass the result of `(jwk.Key).Algorithm()`.
// However, do note that while `(jwk.Key).Algorithm()` could very well contain key encryption
// algorithms, it could also contain other types of values, such as _signature algorithms_.
// In order for `jwe.Decrypt` to work properly, the `alg` parameter must be of type
// `jwa.KeyEncryptionAlgorithm` or otherwise it will cause an error.
//
// When using `jwe.WithKey()`, the value must be a private key.
// It can be either in its raw format (e.g. *rsa.PrivateKey) or a jwk.Key
//
// When the encrypted message is also compressed, the decompressed payload must be
// smaller than the size specified by the `jwe.WithMaxDecompressBufferSize` setting,
// which defaults to 10MB. If the decompressed payload is larger than this size,
// an error is returned.
//
// You can opt to change the MaxDecompressBufferSize setting globally, or on a
// per-call basis by passing the `jwe.WithMaxDecompressBufferSize` option to
// either `jwe.Settings()` or `jwe.Decrypt()`:
//
//	jwe.Settings(jwe.WithMaxDecompressBufferSize(10*1024*1024)) // changes value globally
//	jwe.Decrypt(..., jwe.WithMaxDecompressBufferSize(250*1024)) // changes just for this call
//
// PBES2 amplification: PBES2 algorithms (PBES2-HS256+A128KW, etc.)
// derive the CEK via PBKDF2 with the iteration count taken from the
// JWE's `p2c` header. An attacker-controlled iteration count multiplied
// by `WithMaxRecipients` is the major CPU-amplification vector on the
// decrypt side. Bound it via `WithMaxPBES2Count` (default 1,000,000)
// and reject too-low counts via `WithMinPBES2Count` (default 1000;
// RFC 7518 §4.8.1.2 floor — note OWASP 2023 recommends ≥600,000 for
// production password-derived key material). Both options accept a
// `Settings()` global or a per-call value.
func Decrypt(buf []byte, options ...DecryptOption) ([]byte, error) {
	dc := decryptContextPool.Get()
	defer decryptContextPool.Put(dc)

	if err := dc.ProcessOptions(options); err != nil {
		return nil, makeDecryptError(`failed to process options: %w`, err)
	}

	ret, err := dc.DecryptMessage(buf)
	if err != nil {
		// DecryptMessage already returns errors prefixed with
		// "jwe.Decrypt:" — wrap as decryptError without adding a
		// second prefix, otherwise multi-recipient errors carry
		// the "jwe.Decrypt:" string multiple times.
		return nil, decryptError{err}
	}
	return ret, nil
}

// Parse parses the JWE message into a Message object. The JWE message
// can be either compact or full JSON format.
//
// Bounding the input size is the caller's responsibility; this function
// trusts the caller-provided buf. See docs/13-input-size.md.
func Parse(buf []byte, _ ...ParseOption) (*Message, error) {
	return parseJSONOrCompact(buf, false, int(maxRecipients.Load()))
}

// errors are wrapped within this function, because we call it directly
// from Decrypt as well.
func parseJSONOrCompact(buf []byte, storeProtectedHeaders bool, maxR int) (*Message, error) {
	buf = bytes.TrimSpace(buf)
	if len(buf) == 0 {
		return nil, makeParseError(`jwe.Parse`, `empty buffer`)
	}

	var msg *Message
	var err error
	if buf[0] == tokens.OpenCurlyBracket {
		msg, err = parseJSON(buf, storeProtectedHeaders)
	} else {
		msg, err = parseCompact(buf, storeProtectedHeaders)
	}

	if err != nil {
		return nil, makeParseError(`jwe.Parse`, `%w`, err)
	}

	if maxR > 0 && len(msg.recipients) > maxR {
		return nil, makeParseError(`jwe.Parse`, `too many recipients in JWE message (%d > %d)`, len(msg.recipients), maxR)
	}

	return msg, nil
}

// ParseString is the same as Parse, but takes a string.
func ParseString(s string, _ ...ParseOption) (*Message, error) {
	msg, err := Parse([]byte(s))
	if err != nil {
		return nil, makeParseError(`jwe.ParseString`, `%w`, err)
	}
	return msg, nil
}

// ParseReader is the same as Parse, but takes an io.Reader.
//
// Bounding the input size is the caller's responsibility: wrap src with
// [io.LimitReader] or [net/http.MaxBytesReader] before passing it in. See
// docs/13-input-size.md for the rationale.
func ParseReader(src io.Reader, _ ...ParseOption) (*Message, error) {
	buf, err := io.ReadAll(src)
	if err != nil {
		return nil, makeParseError(`jwe.ParseReader`, `failed to read from io.Reader: %w`, err)
	}
	msg, err := Parse(buf)
	if err != nil {
		return nil, makeParseError(`jwe.ParseReader`, `%w`, err)
	}
	return msg, nil
}

func parseJSON(buf []byte, storeProtectedHeaders bool) (*Message, error) {
	m := NewMessage()
	m.storeProtectedHeaders = storeProtectedHeaders
	if err := json.Unmarshal(buf, &m); err != nil {
		return nil, fmt.Errorf(`failed to parse JSON: %w`, err)
	}
	return m, nil
}

func parseCompact(buf []byte, storeProtectedHeaders bool) (*Message, error) {
	var parts [5][]byte
	var ok bool

	for i := range 4 {
		parts[i], buf, ok = bytes.Cut(buf, []byte{tokens.Period})
		if !ok {
			return nil, fmt.Errorf(`compact JWE format must have five parts (%d)`, i+1)
		}
	}
	// Validate that the last part does not contain more dots
	if bytes.ContainsRune(buf, tokens.Period) {
		return nil, errors.New(`compact JWE format must have five parts, not more`)
	}
	parts[4] = buf

	hdrbuf, err := base64.Decode(parts[0])
	if err != nil {
		return nil, fmt.Errorf(`failed to parse first part of compact form: %w`, err)
	}

	protected := NewHeaders()
	if err := json.Unmarshal(hdrbuf, protected); err != nil {
		return nil, fmt.Errorf(`failed to parse header JSON: %w`, err)
	}

	ivbuf, err := base64.Decode(parts[2])
	if err != nil {
		return nil, fmt.Errorf(`failed to base64 decode iv: %w`, err)
	}

	ctbuf, err := base64.Decode(parts[3])
	if err != nil {
		return nil, fmt.Errorf(`failed to base64 decode content: %w`, err)
	}

	tagbuf, err := base64.Decode(parts[4])
	if err != nil {
		return nil, fmt.Errorf(`failed to base64 decode tag: %w`, err)
	}

	m := NewMessage()
	if err := m.Set(CipherTextKey, ctbuf); err != nil {
		return nil, fmt.Errorf(`failed to set %s: %w`, CipherTextKey, err)
	}
	if err := m.Set(InitializationVectorKey, ivbuf); err != nil {
		return nil, fmt.Errorf(`failed to set %s: %w`, InitializationVectorKey, err)
	}
	if err := m.Set(ProtectedHeadersKey, protected); err != nil {
		return nil, fmt.Errorf(`failed to set %s: %w`, ProtectedHeadersKey, err)
	}

	if err := m.makeDummyRecipient(string(parts[1]), protected); err != nil {
		return nil, fmt.Errorf(`failed to setup recipient: %w`, err)
	}

	if err := m.Set(TagKey, tagbuf); err != nil {
		return nil, fmt.Errorf(`failed to set %s: %w`, TagKey, err)
	}

	if storeProtectedHeaders {
		// This is later used for decryption.
		m.rawProtectedHeaders = parts[0]
	}

	return m, nil
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
//	jwe.RegisterCustomField(`x-birthday`, jwe.CustomDecodeFunc(func(data []byte) (any, error) {
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
