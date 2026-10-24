package jwsbb

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"

	"github.com/lestrrat-go/dsig"
)

// This file enforces the RFC 7518 Section 3.4 binding between an ECDSA JWS
// algorithm and the curve its key must sit on (ES256/P-256, ES384/P-384,
// ES512/P-521). It is sign-side only, and jws reaches it only when the caller
// passes jws.WithStrictECDSA(true).
//
// The check is opt-in because the old permissive behavior is an interop
// defect, not a security hole: the signer controls both the key and the
// algorithm at the call site, and the JWS it produces is a genuine signature
// under its own key. Turning the check on by default would break working
// callers to fix a conformance problem they may not have.
//
// jws.Verify never reaches this file at all. It infers algorithms from a key
// when a JWKS entry carries no "alg" (see jws/internal/keyalg.Candidates and
// the deprecated jws.AlgorithmsForKey, whose godoc freezes that inference),
// and it must stay exactly as permissive as it is today.

// RequireECDSACurve reports whether key sits on the curve RFC 7518 Section
// 3.4 binds joseAlg to. It returns nil -- never an error -- when the binding
// cannot be established: dsigAlg is an ECDSA-family algorithm outside the
// three JOSE built-ins (e.g. ES256K, whether from the jwx_es256k build tag
// or an extension module), or key carries no readable curve. Only positive
// evidence of a mismatch is an error.
func RequireECDSACurve(joseAlg, dsigAlg string, key any) error {
	want, ok := curveForDsigAlgorithm(dsigAlg)
	if !ok {
		return nil
	}

	pub := ecdsaPublicKeyOf(key)
	if pub == nil || pub.Curve == nil {
		return nil
	}

	if pub.Curve == want {
		return nil
	}
	gotParams := pub.Curve.Params()
	if gotParams == nil {
		return nil
	}
	wantParams := want.Params()
	if wantParams != nil && gotParams.Name == wantParams.Name {
		return nil
	}

	return fmt.Errorf(`ECDSA curve mismatch: key is on %s, algorithm %q requires %s`,
		curveName(pub.Curve), joseAlg, curveName(want))
}

// curveForDsigAlgorithm maps a dsig ECDSA algorithm name to the curve RFC
// 7518 Section 3.4 requires for it. Only the three JOSE built-ins are
// known; anything else (custom-curve extensions such as ES256K) misses
// deliberately, so the caller passes the key through unchecked.
func curveForDsigAlgorithm(dsigAlg string) (elliptic.Curve, bool) {
	switch dsigAlg {
	case dsig.ECDSAWithP256AndSHA256:
		return elliptic.P256(), true
	case dsig.ECDSAWithP384AndSHA384:
		return elliptic.P384(), true
	case dsig.ECDSAWithP521AndSHA512:
		return elliptic.P521(), true
	default:
		return nil, false
	}
}

// ecdsaPublicKeyOf extracts an *ecdsa.PublicKey from key, or nil when key is
// not (or does not expose) an ECDSA key. Callers pass an already-converted
// key (jwk.Key unwrapping happens before this is called), so only the raw Go
// crypto forms and an opaque crypto.Signer are handled here.
func ecdsaPublicKeyOf(key any) *ecdsa.PublicKey {
	switch k := key.(type) {
	case *ecdsa.PrivateKey:
		if k == nil {
			return nil
		}
		return &k.PublicKey
	case ecdsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PublicKey:
		return k
	case ecdsa.PublicKey:
		return &k
	case crypto.Signer:
		pub, ok := k.Public().(*ecdsa.PublicKey)
		if !ok {
			return nil
		}
		return pub
	default:
		return nil
	}
}

// curveName returns crv.Params().Name, guarding a nil Params() the same way
// the comparison in RequireECDSACurve does.
func curveName(crv elliptic.Curve) string {
	if crv == nil {
		return "<nil>"
	}
	params := crv.Params()
	if params == nil {
		return "<unknown>"
	}
	return params.Name
}
