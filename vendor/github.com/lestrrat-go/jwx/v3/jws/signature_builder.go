package jws

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"slices"

	"github.com/lestrrat-go/dsig"

	"github.com/lestrrat-go/jwx/v3/internal/json"
	"github.com/lestrrat-go/jwx/v3/internal/keyconv"
	"github.com/lestrrat-go/jwx/v3/internal/pool"
	"github.com/lestrrat-go/jwx/v3/internal/tokens"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	jwsbbi "github.com/lestrrat-go/jwx/v3/jws/internal/jwsbb"
	"github.com/lestrrat-go/jwx/v3/jws/jwsbb"
)

var signatureBuilderPool = pool.New[*signatureBuilder](allocSignatureBuilder, freeSignatureBuilder)

// signatureBuilder is a transient object that is used to build
// a single JWS signature.
//
// In a multi-signature JWS message, each message is paired with
// the following:
// - a signer (the object that takes a buffer and key and generates a signature)
// - a key (the key that is used to sign the payload)
// - protected headers (the headers that are protected by the signature)
// - public headers (the headers that are not protected by the signature)
//
// This object stores all of this information in one place.
//
// This object does NOT take care of any synchronization, because it is
// meant to be used in a single-threaded context.
type signatureBuilder struct {
	alg       jwa.SignatureAlgorithm
	signer    Signer
	signer2   Signer2
	key       any
	protected Headers
	public    Headers
}

func allocSignatureBuilder() *signatureBuilder {
	return &signatureBuilder{}
}

func freeSignatureBuilder(sb *signatureBuilder) *signatureBuilder {
	sb.alg = jwa.EmptySignatureAlgorithm()
	sb.signer = nil
	sb.signer2 = nil
	sb.key = nil
	sb.protected = nil
	sb.public = nil
	return sb
}

// requireECDSACurve enforces the RFC 7518 Section 3.4 binding between an ES*
// algorithm and the curve its key must sit on. It is only reached when the
// caller asked for it with jws.WithStrictECDSA(true).
//
// Anything that is not an ECDSA signature passes straight through, as does an
// ECDSA-family algorithm outside the three JOSE built-ins (an extension on its
// own curve, such as ES256K) and a key whose curve cannot be read. Deciding
// those cases is not this check's job; only positive evidence of a mismatch is
// an error.
func requireECDSACurve(alg jwa.SignatureAlgorithm, key any) error {
	dsigAlg, ok := jwsbb.GetDsigAlgorithm(alg.String())
	if !ok {
		return nil
	}

	info, ok := dsig.GetAlgorithmInfo(dsigAlg)
	if !ok || info.Family != dsig.ECDSA {
		return nil
	}

	rawKey, ok := unwrapECDSASignKey(key)
	if !ok {
		return nil
	}

	return jwsbbi.RequireECDSACurve(alg.String(), dsigAlg, rawKey)
}

// unwrapECDSASignKey returns the key jwsbbi.RequireECDSACurve should inspect.
// That function reads the curve off a raw key or a crypto.Signer, so a
// jwk.Key has to be unwrapped first.
//
// The bool is false when key is a jwk.Key holding something other than an
// ECDSA private key, which leaves the curve unreadable. The caller skips the
// check in that case and lets the signer reject the key on its own terms.
func unwrapECDSASignKey(key any) (any, bool) {
	if _, ok := key.(jwk.Key); !ok {
		return key, true
	}

	var privkey *ecdsa.PrivateKey
	if err := keyconv.ECDSAPrivateKey(&privkey, key); err != nil {
		return nil, false
	}
	return privkey, true
}

func (sb *signatureBuilder) Build(sc *signContext, payload []byte) (*Signature, error) {
	if sc.strictECDSA {
		if err := requireECDSACurve(sb.alg, sb.key); err != nil {
			return nil, makeSignError(prefixJwsSign, `%w`, err)
		}
	}

	// Clone caller-provided headers before mutating so that re-using the
	// same Headers instance across multiple Sign calls does not cause
	// cross-contamination of alg/kid.
	var protected Headers
	if sb.protected != nil {
		cloned, err := sb.protected.Clone()
		if err != nil {
			return nil, makeSignError(prefixJwsSign, `failed to clone protected headers: %w`, err)
		}
		protected = cloned
	} else {
		protected = NewHeaders()
	}

	if err := protected.Set(AlgorithmKey, sb.alg); err != nil {
		return nil, makeSignError(prefixJwsSign, `failed to set "alg" header: %w`, err)
	}

	if key, ok := sb.key.(jwk.Key); ok {
		if kid, ok := key.KeyID(); ok && kid != "" {
			if err := protected.Set(KeyIDKey, kid); err != nil {
				return nil, makeSignError(prefixJwsSign, `failed to set "kid" header: %w`, err)
			}
		}
	}

	// RFC 7797 §3 requires producers that set "b64":false to also list
	// "b64" in "crit". Auto-declare it in the protected header so a
	// caller who set b64=false but forgot the crit declaration does not
	// emit a non-conformant stream that strict verifiers refuse.
	// Idempotent: if "b64" is already in crit, the list is unchanged.
	// If crit is unset, it is created with just "b64".
	if !getB64Value(protected) {
		crit, _ := protected.Critical()
		if !slices.Contains(crit, "b64") {
			crit = append(crit, "b64")
			if err := protected.Set(CriticalKey, crit); err != nil {
				return nil, makeSignError(prefixJwsSign, `failed to set "crit" header: %w`, err)
			}
		}
	}

	hdrs, err := mergeHeaders(sb.public, protected)
	if err != nil {
		return nil, makeSignError(prefixJwsSign, `failed to merge headers: %w`, err)
	}

	// raw, json format headers
	hdrbuf, err := json.Marshal(hdrs)
	if err != nil {
		return nil, fmt.Errorf(`failed to marshal headers: %w`, err)
	}

	// check if we need to base64 encode the payload
	b64 := getB64Value(hdrs)
	if !b64 && !sc.detached {
		if bytes.IndexByte(payload, tokens.Period) != -1 {
			return nil, fmt.Errorf(`compact serialization with b64=false requires payload to contain no "." characters per RFC 7797 §5.2; use jws.WithDetachedPayload to keep the payload out of the wire format`)
		}
	}

	combined := jwsbb.SignBuffer(nil, hdrbuf, payload, sc.encoder, b64)

	var sig Signature
	sig.protected = protected
	sig.headers = sb.public

	if sb.signer2 != nil {
		signature, err := sb.signer2.Sign(sb.key, combined)
		if err != nil {
			return nil, fmt.Errorf(`failed to sign payload: %w`, err)
		}
		sig.signature = signature
		return &sig, nil
	}

	if sb.signer == nil {
		panic("can't get here")
	}

	signature, err := sb.signer.Sign(combined, sb.key)
	if err != nil {
		return nil, fmt.Errorf(`failed to sign payload: %w`, err)
	}

	sig.signature = signature

	return &sig, nil
}
