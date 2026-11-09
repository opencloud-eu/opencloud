package jws

import (
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/rsa"
	"fmt"
	"hash"
	"io"

	"github.com/lestrrat-go/blackmagic"
	"github.com/lestrrat-go/dsig"
	"github.com/lestrrat-go/jwx/v3/internal/base64"
	"github.com/lestrrat-go/jwx/v3/internal/json"
	"github.com/lestrrat-go/jwx/v3/internal/keyconv"
	"github.com/lestrrat-go/jwx/v3/internal/tokens"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	jwsbbi "github.com/lestrrat-go/jwx/v3/jws/internal/jwsbb"
	"github.com/lestrrat-go/jwx/v3/jws/jwsbb"
)

// This file implements the streaming detached-payload variant of jws.Sign()
// and jws.Verify(), reached via the jws.WithDetachedPayloadReader() option.
// It deliberately bypasses the jws.Signer / jws.Verifier dispatch path and
// talks to dsig directly, because it needs incremental hashing through a
// hash.Hash — the Signer interface takes a fully materialized []byte payload.
//
// Consequences: algorithms registered via jws.RegisterSigner() /
// jws.RegisterVerifier() are unreachable here, as are algorithm families
// that cannot be driven from a digest (EdDSA, custom).

// streamingSigner carries per-signature state through signStreaming:
// the header prefix already written into hasher, the hasher itself
// (fed with the prefix, ready to consume payload bytes), the resolved
// dsig alg info, and the raw key / unprotected header needed for final
// signing and JSON assembly.
type streamingSigner struct {
	dsigInfo   streamingAlgorithmInfo
	rawKey     any
	hasher     hash.Hash
	hdrEncoded string // base64(protected header JSON)
	public     Headers
}

// signStreaming is invoked from Sign() when sc.payloadReader is set. It
// assembles the signing input for each configured signer by feeding
// base64(header) "." base64(payload) (or raw payload when b64=false)
// into a hash.Hash, then calls dsig.SignDigest once per signer. When
// more than one signer is registered the payload is streamed once and
// fanned out to each hasher via [io.MultiWriter]; the general JSON
// serialization is used for the output.
func (sc *signContext) signStreaming() ([]byte, error) {
	if sc.none != nil {
		return nil, makeSignError(prefixJwsSign, `jws.WithInsecureNoSignature() cannot be combined with jws.WithDetachedPayloadReader(); use jws.Sign with jws.WithInsecureNoSignature() if you really need an unsecured in-memory JWS`)
	}

	streamEncoder, ok := base64.AsStreamEncoder(sc.encoder)
	if !ok {
		return nil, makeSignError(prefixJwsSign, `jws.WithDetachedPayloadReader() requires a base64 encoder with a NewEncoder(io.Writer) io.WriteCloser method (interface jws.Base64StreamEncoder). The configured encoder %T does not provide one. Install a stream-capable encoder via jwx.Settings(jwx.WithBase64Encoder(...)) or jws.WithBase64Encoder(...); the default encoding/base64.RawURLEncoding satisfies this automatically.`, sc.encoder)
	}

	signers := make([]streamingSigner, 0, len(sc.sigbuilders))
	var b64 bool
	var b64Set bool

	for idx, sb := range sc.sigbuilders {
		alg := sb.alg
		if alg == jwa.NoSignature() {
			return nil, makeSignError(prefixJwsSign, `"none" (jwa.NoSignature) cannot be used with jws.WithDetachedPayloadReader()`)
		}

		dsigInfo, err := resolveStreamingAlgorithm(alg)
		if err != nil {
			return nil, makeSignError(prefixJwsSign, `signature %d: %w`, idx, err)
		}

		if sc.validateKey {
			if err := validateKeyBeforeUse(sb.key); err != nil {
				return nil, makeSignError(prefixJwsSign, `failed to validate key for signature %d: %w`, idx, err)
			}
		}

		rawKey, err := convertStreamingSignKey(sb.key, dsigInfo.Family)
		if err != nil {
			return nil, makeSignError(prefixJwsSign, `failed to convert key for signature %d: %w`, idx, err)
		}

		// The non-streaming path runs the same check from
		// signatureBuilder.Build, which this path does not go through.
		if sc.strictECDSA && dsigInfo.Family == dsig.ECDSA {
			if err := jwsbbi.RequireECDSACurve(alg.String(), dsigInfo.Name, rawKey); err != nil {
				return nil, makeSignError(prefixJwsSign, `signature %d: %w`, idx, err)
			}
		}

		protected, err := cloneOrNewHeaders(sb.protected)
		if err != nil {
			return nil, makeSignError(prefixJwsSign, `failed to clone protected headers for signature %d: %w`, idx, err)
		}
		if err := protected.Set(AlgorithmKey, alg); err != nil {
			return nil, makeSignError(prefixJwsSign, `failed to set "alg" header for signature %d: %w`, idx, err)
		}
		if jwkKey, ok := sb.key.(jwk.Key); ok {
			var kid string
			if err := jwkKey.Get(jwk.KeyIDKey, &kid); err == nil && kid != "" {
				if err := protected.Set(KeyIDKey, kid); err != nil {
					return nil, makeSignError(prefixJwsSign, `failed to set "kid" header for signature %d: %w`, idx, err)
				}
			}
		}

		// For compact serialization RFC 7515 requires the unprotected
		// header to be merged into the protected header because there
		// is no separate slot for it on the wire. JSON serializations
		// keep them separate.
		signingHeaders := protected
		if sc.format == fmtCompact {
			signingHeaders, err = mergeHeaders(sb.public, protected)
			if err != nil {
				return nil, makeSignError(prefixJwsSign, `failed to merge headers for signature %d: %w`, idx, err)
			}
		}

		// A multi-signature JWS has a single payload segment on the
		// wire, so every signer must agree on the RFC 7797 "b64" flag
		// or the produced JWS is internally inconsistent.
		thisB64 := getB64Value(signingHeaders)
		if !b64Set {
			b64 = thisB64
			b64Set = true
		} else if thisB64 != b64 {
			return nil, makeSignError(prefixJwsSign, `signature %d disagrees with earlier signers on the RFC 7797 "b64" flag; all signers must use the same b64 value`, idx)
		}

		hdrbuf, err := json.Marshal(signingHeaders)
		if err != nil {
			return nil, makeSignError(prefixJwsSign, `failed to marshal headers for signature %d: %w`, idx, err)
		}

		hasher, err := newStreamingHasher(dsigInfo, rawKey)
		if err != nil {
			return nil, makeSignError(prefixJwsSign, `failed to create hasher for signature %d: %w`, idx, err)
		}

		hdrEncoded := streamEncoder.EncodeToString(hdrbuf)
		if _, err := hasher.Write([]byte(hdrEncoded)); err != nil {
			return nil, makeSignError(prefixJwsSign, `failed to write signing prefix for signature %d: %w`, idx, err)
		}
		if _, err := hasher.Write([]byte{tokens.Period}); err != nil {
			return nil, makeSignError(prefixJwsSign, `failed to write signing prefix for signature %d: %w`, idx, err)
		}

		signers = append(signers, streamingSigner{
			dsigInfo:   dsigInfo,
			rawKey:     rawKey,
			hasher:     hasher,
			hdrEncoded: hdrEncoded,
			public:     sb.public,
		})
	}

	hashers := make([]hash.Hash, len(signers))
	for i := range signers {
		hashers[i] = signers[i].hasher
	}
	if err := streamPayloadIntoHashers(hashers, sc.payloadReader, b64, streamEncoder); err != nil {
		return nil, makeSignError(prefixJwsSign, `failed to stream payload: %w`, err)
	}

	rawSignatures := make([][]byte, len(signers))
	for i, st := range signers {
		sig, err := dsig.SignDigest(st.rawKey, st.dsigInfo.Name, st.hasher.Sum(nil), nil)
		if err != nil {
			return nil, makeSignError(prefixJwsSign, `failed to sign digest for signature %d: %w`, i, err)
		}
		rawSignatures[i] = sig
	}

	switch sc.format {
	case fmtCompact:
		// Upstream validation in jws.Sign guarantees compact implies
		// exactly one signer, but guard against future drift.
		if len(signers) != 1 {
			return nil, makeSignError(prefixJwsSign, `compact serialization requires exactly one signature, got %d`, len(signers))
		}
		sigEncoded := streamEncoder.EncodeToString(rawSignatures[0])
		buf := make([]byte, 0, len(signers[0].hdrEncoded)+2+len(sigEncoded))
		buf = append(buf, signers[0].hdrEncoded...)
		buf = append(buf, tokens.Period, tokens.Period)
		buf = append(buf, sigEncoded...)
		return buf, nil
	case fmtJSON, fmtJSONPretty:
		return assembleStreamingDetachedJSON(signers, rawSignatures, streamEncoder, sc.format == fmtJSONPretty)
	default:
		return nil, makeSignError(prefixJwsSign, `unexpected serialization format %d`, sc.format)
	}
}

// verifyStreaming is invoked from VerifyMessage() when vc.payloadReader is
// set. It parses the JWS envelope through jws.Parse, then re-builds the
// signing input by feeding base64(header) "." base64(payload) into a
// hash.Hash fed from the supplied io.Reader and calls dsig.VerifyDigest.
func (vc *verifyContext) verifyStreaming(buf []byte) ([]byte, error) {
	if len(vc.keyProviders) != 1 {
		return nil, makeVerifyError(`jws.WithDetachedPayloadReader() requires exactly one jws.WithKey(); jws.WithKeySet(), jws.WithKeyProvider() and jws.WithVerifyAuto() are not supported on the streaming path`)
	}
	staticKP, ok := vc.keyProviders[0].(*staticKeyProvider)
	if !ok {
		return nil, makeVerifyError(`jws.WithDetachedPayloadReader() requires exactly one jws.WithKey(); jws.WithKeySet(), jws.WithKeyProvider() and jws.WithVerifyAuto() are not supported on the streaming path`)
	}
	alg := staticKP.alg
	key := staticKP.key

	if alg == jwa.NoSignature() {
		return nil, makeVerifyError(`"none" (jwa.NoSignature) cannot be used with jws.WithDetachedPayloadReader(); use jws.Parse if you need to inspect an unsecured JWS`)
	}

	dsigInfo, err := resolveStreamingAlgorithm(alg)
	if err != nil {
		return nil, makeVerifyError(`%w`, err)
	}

	streamEncoder, ok := base64.AsStreamEncoder(vc.encoder)
	if !ok {
		return nil, makeVerifyError(`jws.WithDetachedPayloadReader() requires a base64 encoder with a NewEncoder(io.Writer) io.WriteCloser method (interface jws.Base64StreamEncoder). The configured encoder %T does not provide one. Install a stream-capable encoder via jwx.Settings(jwx.WithBase64Encoder(...)) or jws.WithBase64Encoder(...); the default encoding/base64.RawURLEncoding satisfies this automatically.`, vc.encoder)
	}

	msg, err := Parse(buf, vc.parseOptions...)
	if err != nil {
		return nil, makeVerifyError(`failed to parse jws: %w`, err)
	}
	defer msg.clearRaw()

	if len(msg.signatures) != 1 {
		return nil, makeVerifyError(`jws.WithDetachedPayloadReader() supports only single-signature JWS, got %d`, len(msg.signatures))
	}
	if len(msg.payload) != 0 {
		return nil, makeVerifyError(`JWS must not have an embedded payload when jws.WithDetachedPayloadReader() is used`)
	}

	sig := msg.signatures[0]

	var rawHeaders []byte
	if rbp, ok := sig.protected.(interface{ rawBuffer() []byte }); ok {
		rawHeaders = rbp.rawBuffer()
	}
	if rawHeaders == nil {
		rawHeaders, err = json.Marshal(sig.protected)
		if err != nil {
			return nil, makeVerifyError(`failed to marshal "protected": %w`, err)
		}
	}

	if vc.critValidation {
		if err := validateB64InCritIfFalse(sig.protected); err != nil {
			return nil, makeVerifyError(`%w`, err)
		}
		if err := validateCritical(sig.protected, vc.criticalExtensions); err != nil {
			return nil, makeVerifyError(`invalid "crit" header: %w`, err)
		}
	}

	if vc.validateKey {
		if err := validateKeyBeforeUse(key); err != nil {
			return nil, makeVerifyError(`failed to validate key before verification: %w`, err)
		}
	}

	rawKey, err := convertStreamingVerifyKey(key, dsigInfo.Family)
	if err != nil {
		return nil, makeVerifyError(`failed to convert key: %w`, err)
	}

	hasher, err := newStreamingHasher(dsigInfo, rawKey)
	if err != nil {
		return nil, makeVerifyError(`failed to create hasher: %w`, err)
	}
	hdrEncoded := streamEncoder.EncodeToString(rawHeaders)
	if _, err := hasher.Write([]byte(hdrEncoded)); err != nil {
		return nil, makeVerifyError(`failed to write signing prefix: %w`, err)
	}
	if _, err := hasher.Write([]byte{tokens.Period}); err != nil {
		return nil, makeVerifyError(`failed to write signing prefix: %w`, err)
	}
	if err := streamPayloadIntoHashers([]hash.Hash{hasher}, vc.payloadReader, msg.b64, streamEncoder); err != nil {
		return nil, makeVerifyError(`failed to stream payload: %w`, err)
	}

	if err := dsig.VerifyDigest(rawKey, dsigInfo.Name, hasher.Sum(nil), sig.signature); err != nil {
		return nil, makeVerifyError(`failed to verify signature: %w`, verificationError{err})
	}

	if vc.keyUsed != nil {
		if err := blackmagic.AssignIfCompatible(vc.keyUsed, key); err != nil {
			return nil, makeVerifyError(`failed to assign key to keyUsed: %w`, err)
		}
	}
	if vc.dst != nil {
		*vc.dst = *msg
	}
	// Non-nil zero-length slice is the sentinel: the payload was streamed
	// from the caller so there are no bytes to hand back, but returning
	// nil would be indistinguishable from "ignored return" and invite
	// `len(payload) == 0` silent-logic bugs in callers.
	return []byte{}, nil
}

// streamingAlgorithmInfo carries the resolved dsig metadata plus the dsig
// algorithm name, since dsig.AlgorithmInfo itself does not include it.
type streamingAlgorithmInfo struct {
	dsig.AlgorithmInfo

	Name string
}

// resolveStreamingAlgorithm maps a JWS algorithm to its dsig metadata and
// enforces the family restrictions for the streaming path. It routes through
// jwsbb.GetDsigAlgorithm so algorithms registered by extension modules work
// just like algorithms built in to jws.
func resolveStreamingAlgorithm(alg jwa.SignatureAlgorithm) (streamingAlgorithmInfo, error) {
	dsigAlg, ok := jwsbb.GetDsigAlgorithm(alg.String())
	if !ok {
		// For custom dsig algorithms registered directly with dsig the JWS
		// name may equal the dsig name.
		dsigAlg = alg.String()
	}
	info, ok := dsig.GetAlgorithmInfo(dsigAlg)
	if !ok {
		return streamingAlgorithmInfo{}, fmt.Errorf(`unsupported algorithm %q; use jws.WithDetachedPayload() if you need the general detached path`, alg)
	}
	switch info.Family {
	case dsig.EdDSAFamily:
		return streamingAlgorithmInfo{}, fmt.Errorf(`algorithm %q is incompatible with streaming detached payloads: RFC 8032 EdDSA signs the full message, not a pre-computed digest, so the payload cannot be streamed; use jws.WithDetachedPayload() if the payload fits in memory, or a digest-based algorithm such as HS256, RS256, or ES256`, alg)
	case dsig.Custom:
		return streamingAlgorithmInfo{}, fmt.Errorf(`algorithm %q is a custom-family algorithm and does not support streaming because the library cannot know whether the algorithm pre-hashes the payload; use jws.WithDetachedPayload() if the payload fits in memory`, alg)
	}
	return streamingAlgorithmInfo{AlgorithmInfo: info, Name: dsigAlg}, nil
}

// newStreamingHasher returns a hash.Hash preloaded with the key material
// the family needs (HMAC is keyed; RSA/ECDSA just hash).
func newStreamingHasher(info streamingAlgorithmInfo, key any) (hash.Hash, error) {
	switch info.Family {
	case dsig.HMAC:
		meta, ok := info.Meta.(dsig.HMACFamilyMeta)
		if !ok {
			return nil, fmt.Errorf(`invalid HMAC metadata`)
		}
		keyBytes, ok := key.([]byte)
		if !ok {
			// Route through keyconv so the error matches the non-streaming
			// HMAC path (e.g., passing a string secret surfaces
			// `keyconv: expected []byte, got string`) instead of the
			// terser type-assertion failure.
			if err := keyconv.ByteSliceKey(&keyBytes, key); err != nil {
				return nil, fmt.Errorf(`failed to convert HMAC key to []byte (streaming path): %w`, err)
			}
		}
		return hmac.New(meta.HashFunc, keyBytes), nil
	case dsig.RSA:
		meta, ok := info.Meta.(dsig.RSAFamilyMeta)
		if !ok {
			return nil, fmt.Errorf(`invalid RSA metadata`)
		}
		return meta.Hash.New(), nil
	case dsig.ECDSA:
		meta, ok := info.Meta.(dsig.ECDSAFamilyMeta)
		if !ok {
			return nil, fmt.Errorf(`invalid ECDSA metadata`)
		}
		return meta.Hash.New(), nil
	default:
		return nil, fmt.Errorf(`unsupported algorithm family %q for streaming`, info.Family)
	}
}

// streamPayloadIntoHashers copies the payload once and fans it out to
// every hasher via [io.MultiWriter]. When b64=true the bytes are
// routed through per-hasher [io.WriteCloser]s returned by the
// [base64.StreamEncoder] (each encoder keeps an unflushed 3-byte tail,
// so the wrappers cannot be shared). When b64=false the hashers receive
// the payload bytes directly.
func streamPayloadIntoHashers(hashers []hash.Hash, payload io.Reader, encodePayload bool, enc base64.StreamEncoder) error {
	if len(hashers) == 0 {
		return fmt.Errorf(`no hashers to stream into`)
	}
	if !encodePayload {
		writers := make([]io.Writer, len(hashers))
		for i, h := range hashers {
			writers[i] = h
		}
		if _, err := io.Copy(io.MultiWriter(writers...), payload); err != nil {
			return fmt.Errorf(`failed to stream payload: %w`, err)
		}
		return nil
	}

	encoders := make([]io.WriteCloser, len(hashers))
	writers := make([]io.Writer, len(hashers))
	for i, h := range hashers {
		encoders[i] = enc.NewEncoder(h)
		writers[i] = encoders[i]
	}
	if _, err := io.Copy(io.MultiWriter(writers...), payload); err != nil {
		for _, w := range encoders {
			_ = w.Close()
		}
		return fmt.Errorf(`failed to stream payload through base64 encoder: %w`, err)
	}
	for i, w := range encoders {
		if err := w.Close(); err != nil {
			return fmt.Errorf(`failed to close base64 encoder for signature %d: %w`, i, err)
		}
	}
	return nil
}

type streamingDetachedJSONEntry struct {
	Header    json.RawMessage `json:"header,omitempty"`
	Protected string          `json:"protected"`
	Signature string          `json:"signature"`
}

type streamingDetachedJSONGeneral struct {
	Signatures []streamingDetachedJSONEntry `json:"signatures"`
}

// assembleStreamingDetachedJSON emits the flattened form for a single
// signature and the general form for multiple. The "payload" member is
// omitted in both cases per RFC 7515 Appendix F.
func assembleStreamingDetachedJSON(signers []streamingSigner, rawSignatures [][]byte, enc base64.StreamEncoder, pretty bool) ([]byte, error) {
	entries := make([]streamingDetachedJSONEntry, len(signers))
	for i, st := range signers {
		e := streamingDetachedJSONEntry{
			Protected: st.hdrEncoded,
			Signature: enc.EncodeToString(rawSignatures[i]),
		}
		if st.public != nil {
			hdrjs, err := json.Marshal(st.public)
			if err != nil {
				return nil, makeSignError(prefixJwsSign, `failed to marshal unprotected header for signature %d: %w`, i, err)
			}
			e.Header = hdrjs
		}
		entries[i] = e
	}

	var payload any
	if len(entries) == 1 {
		payload = entries[0]
	} else {
		payload = streamingDetachedJSONGeneral{Signatures: entries}
	}

	if pretty {
		out, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, makeSignError(prefixJwsSign, `failed to marshal JSON output: %w`, err)
		}
		return out, nil
	}
	out, err := json.Marshal(payload)
	if err != nil {
		return nil, makeSignError(prefixJwsSign, `failed to marshal JSON output: %w`, err)
	}
	return out, nil
}

// cloneOrNewHeaders returns a defensive copy of hdr, or a fresh empty
// Headers if hdr is nil. The streaming path mutates the protected headers
// to set "alg" / "kid", so we never mutate a caller-supplied value.
func cloneOrNewHeaders(hdr Headers) (Headers, error) {
	if hdr == nil {
		return NewHeaders(), nil
	}
	return hdr.Clone()
}

func convertStreamingSignKey(key any, family dsig.Family) (any, error) {
	if _, ok := key.(jwk.Key); !ok {
		return key, nil
	}
	switch family {
	case dsig.HMAC:
		var rawKey []byte
		if err := keyconv.ByteSliceKey(&rawKey, key); err != nil {
			return nil, fmt.Errorf(`failed to convert HMAC key: %w`, err)
		}
		return rawKey, nil
	case dsig.RSA:
		var rawKey rsa.PrivateKey
		if err := keyconv.RSAPrivateKey(&rawKey, key); err != nil {
			return nil, fmt.Errorf(`failed to convert RSA key: %w`, err)
		}
		return &rawKey, nil
	case dsig.ECDSA:
		var rawKey ecdsa.PrivateKey
		if err := keyconv.ECDSAPrivateKey(&rawKey, key); err != nil {
			return nil, fmt.Errorf(`failed to convert ECDSA key: %w`, err)
		}
		return &rawKey, nil
	default:
		return key, nil
	}
}

func convertStreamingVerifyKey(key any, family dsig.Family) (any, error) {
	if _, ok := key.(jwk.Key); !ok {
		return key, nil
	}
	switch family {
	case dsig.HMAC:
		var rawKey []byte
		if err := keyconv.ByteSliceKey(&rawKey, key); err != nil {
			return nil, fmt.Errorf(`failed to convert HMAC key: %w`, err)
		}
		return rawKey, nil
	case dsig.RSA:
		var rawKey rsa.PublicKey
		if err := keyconv.RSAPublicKey(&rawKey, key); err != nil {
			return nil, fmt.Errorf(`failed to convert RSA key: %w`, err)
		}
		return &rawKey, nil
	case dsig.ECDSA:
		var rawKey ecdsa.PublicKey
		if err := keyconv.ECDSAPublicKey(&rawKey, key); err != nil {
			return nil, fmt.Errorf(`failed to convert ECDSA key: %w`, err)
		}
		return &rawKey, nil
	default:
		return key, nil
	}
}
