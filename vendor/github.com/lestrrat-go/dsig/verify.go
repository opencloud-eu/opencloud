package dsig

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"fmt"
)

// Verify verifies a digital signature using the specified key and algorithm.
//
// Deprecated in spirit: in the next major release of dsig (v2), the
// signature of Verify will change to match [VerifyWithOpts], i.e. it
// will accept an additional [crypto.SignerOpts] parameter at the end.
// Callers that need to pass per-call options today should use
// [VerifyWithOpts]; callers that do not can keep using Verify and
// migrate when v2 ships by threading a nil opts argument through at
// the call site.
func Verify(key any, alg string, payload, signature []byte) error {
	return VerifyWithOpts(key, alg, payload, signature, nil)
}

// VerifyWithOpts is like [Verify] but threads an optional
// [crypto.SignerOpts] through to the underlying verifier. For built-in
// families (HMAC, RSA, ECDSA, EdDSA) the opts argument is ignored. For
// Custom-family algorithms whose Meta implements [VerifierWithOpts],
// the opts are forwarded; otherwise the plain [Verifier.Verify] method
// is called and opts are dropped.
//
// This function exists as a transitional API. In the next major release
// of dsig (v2) it will be removed and its signature will become the
// canonical shape of [Verify]. Code that uses VerifyWithOpts today will
// need a mechanical rename to Verify (and nothing else) when v2 ships.
func VerifyWithOpts(key any, alg string, payload, signature []byte, opts crypto.SignerOpts) error {
	info, ok := GetAlgorithmInfo(alg)
	if !ok {
		return fmt.Errorf(`dsig.VerifyWithOpts: unsupported signature algorithm %q`, alg)
	}

	switch info.Family {
	case HMAC:
		return dispatchHMACVerify(key, info, payload, signature)
	case RSA:
		return dispatchRSAVerify(key, info, payload, signature)
	case ECDSA:
		return dispatchECDSAVerify(key, info, payload, signature)
	case EdDSAFamily:
		return dispatchEdDSAVerify(key, info, payload, signature)
	case Custom:
		return dispatchCustomVerify(key, info, payload, signature, opts)
	default:
		return fmt.Errorf(`dsig.VerifyWithOpts: unsupported signature family %q`, info.Family)
	}
}

// VerifyDigest verifies a signature given a pre-computed digest.
//
// For RSA/ECDSA, digest is the hash of the signing input and key is the
// public key used for verification.
//
// For HMAC, digest must be the pre-computed MAC (i.e. the output of
// hmac.New(hashFunc, key) after writing the signing input). The key
// parameter is not used because it is already incorporated into the MAC.
//
// EdDSA and Custom families are not supported and return an error.
//
// Deprecated in spirit: in the next major release of dsig (v2), the
// signature of VerifyDigest will gain a [crypto.SignerOpts] parameter
// to align with [Verify]. No VerifyDigestWithOpts shim exists in v1
// because Custom-family algorithms (the only ones that would benefit
// from per-call opts) are rejected outright today; once a
// DigestVerifier interface for the Custom family is added, the opts
// parameter will appear at the same time.
func VerifyDigest(key any, alg string, digest, signature []byte) error {
	info, ok := GetAlgorithmInfo(alg)
	if !ok {
		return fmt.Errorf(`dsig.VerifyDigest: unsupported signature algorithm %q`, alg)
	}

	switch info.Family {
	case HMAC:
		// key is not used here: the caller has already computed the HMAC
		// (which incorporates the key) and passed it as digest.
		return VerifyHMACDigest(digest, signature)
	case RSA:
		return dispatchRSAVerifyDigest(key, info, digest, signature)
	case ECDSA:
		return dispatchECDSAVerifyDigest(key, info, digest, signature)
	case EdDSAFamily:
		return fmt.Errorf(`dsig.VerifyDigest: EdDSA does not support digest-based verification`)
	case Custom:
		// TODO: a DigestVerifier interface (optional, checked here) would let
		// custom algorithms opt in to digest-based verification.
		return fmt.Errorf(`dsig.VerifyDigest: custom algorithms do not support digest-based verification`)
	default:
		return fmt.Errorf(`dsig.VerifyDigest: unsupported signature family %q`, info.Family)
	}
}

func dispatchRSAVerifyDigest(key any, info AlgorithmInfo, digest, signature []byte) error {
	meta, ok := info.Meta.(RSAFamilyMeta)
	if !ok {
		return fmt.Errorf(`dsig.VerifyDigest: invalid RSA metadata`)
	}

	var pubkey *rsa.PublicKey

	if cs, ok := key.(crypto.Signer); ok {
		cpub := cs.Public()
		switch cpub := cpub.(type) {
		case rsa.PublicKey:
			pubkey = &cpub
		case *rsa.PublicKey:
			pubkey = cpub
		default:
			return fmt.Errorf(`dsig.VerifyDigest: failed to retrieve rsa.PublicKey out of crypto.Signer %T`, key)
		}
	} else {
		var ok bool
		pubkey, ok = key.(*rsa.PublicKey)
		if !ok {
			return fmt.Errorf(`dsig.VerifyDigest: failed to retrieve *rsa.PublicKey out of %T`, key)
		}
	}

	return VerifyRSADigest(pubkey, digest, signature, meta.Hash, meta.PSS)
}

// Note: the crypto.Signer → *ecdsa.PublicKey extraction below duplicates
// logic in VerifyECDSACryptoSigner. We can't call that function because it
// hashes the payload internally. If the extraction logic changes, update both.
func dispatchECDSAVerifyDigest(key any, info AlgorithmInfo, digest, signature []byte) error {
	pubkey, cs, isCryptoSigner, err := ecdsaGetVerifierKey(key)
	if err != nil {
		return fmt.Errorf(`dsig.VerifyDigest: %w`, err)
	}
	if isCryptoSigner {
		cpub := cs.Public()
		switch cpub := cpub.(type) {
		case ecdsa.PublicKey:
			pubkey = &cpub
		case *ecdsa.PublicKey:
			pubkey = cpub
		default:
			return fmt.Errorf(`dsig.VerifyDigest: expected *ecdsa.PublicKey from crypto.Signer, got %T`, cpub)
		}
	}
	return VerifyECDSADigest(pubkey, digest, signature)
}

func dispatchHMACVerify(key any, info AlgorithmInfo, payload, signature []byte) error {
	meta, ok := info.Meta.(HMACFamilyMeta)
	if !ok {
		return fmt.Errorf(`dsig.Verify: invalid HMAC metadata`)
	}

	var hmackey []byte
	if err := toHMACKey(&hmackey, key); err != nil {
		return fmt.Errorf(`dsig.Verify: %w`, err)
	}
	return VerifyHMAC(hmackey, payload, signature, meta.HashFunc)
}

func dispatchRSAVerify(key any, info AlgorithmInfo, payload, signature []byte) error {
	meta, ok := info.Meta.(RSAFamilyMeta)
	if !ok {
		return fmt.Errorf(`dsig.Verify: invalid RSA metadata`)
	}

	var pubkey *rsa.PublicKey

	if cs, ok := key.(crypto.Signer); ok {
		cpub := cs.Public()
		switch cpub := cpub.(type) {
		case rsa.PublicKey:
			pubkey = &cpub
		case *rsa.PublicKey:
			pubkey = cpub
		default:
			return fmt.Errorf(`dsig.Verify: failed to retrieve rsa.PublicKey out of crypto.Signer %T`, key)
		}
	} else {
		var ok bool
		pubkey, ok = key.(*rsa.PublicKey)
		if !ok {
			return fmt.Errorf(`dsig.Verify: failed to retrieve *rsa.PublicKey out of %T`, key)
		}
	}

	return VerifyRSA(pubkey, payload, signature, meta.Hash, meta.PSS)
}

func dispatchECDSAVerify(key any, info AlgorithmInfo, payload, signature []byte) error {
	meta, ok := info.Meta.(ECDSAFamilyMeta)
	if !ok {
		return fmt.Errorf(`dsig.Verify: invalid ECDSA metadata`)
	}

	pubkey, cs, isCryptoSigner, err := ecdsaGetVerifierKey(key)
	if err != nil {
		return fmt.Errorf(`dsig.Verify: %w`, err)
	}
	if isCryptoSigner {
		return VerifyECDSACryptoSigner(cs, payload, signature, meta.Hash)
	}
	return VerifyECDSA(pubkey, payload, signature, meta.Hash)
}

func dispatchEdDSAVerify(key any, _ AlgorithmInfo, payload, signature []byte) error {
	var pubkey ed25519.PublicKey
	signer, ok := key.(crypto.Signer)
	if ok {
		v := signer.Public()
		pubkey, ok = v.(ed25519.PublicKey)
		if !ok {
			return fmt.Errorf(`dsig.Verify: expected crypto.Signer.Public() to return ed25519.PublicKey, but got %T`, v)
		}
	} else {
		var ok bool
		pubkey, ok = key.(ed25519.PublicKey)
		if !ok {
			return fmt.Errorf(`dsig.Verify: failed to retrieve ed25519.PublicKey out of %T`, key)
		}
	}

	return VerifyEdDSA(pubkey, payload, signature)
}

func dispatchCustomVerify(key any, info AlgorithmInfo, payload, signature []byte, opts crypto.SignerOpts) error {
	if verifier, ok := info.Meta.(VerifierWithOpts); ok {
		return verifier.VerifyWithOpts(key, payload, signature, opts)
	}
	verifier, ok := info.Meta.(Verifier)
	if !ok {
		return fmt.Errorf(`dsig.Verify: algorithm has no verifier registered`)
	}
	return verifier.Verify(key, payload, signature)
}

func ecdsaGetVerifierKey(key any) (*ecdsa.PublicKey, crypto.Signer, bool, error) {
	cs, isCryptoSigner := key.(crypto.Signer)
	if isCryptoSigner {
		switch key.(type) {
		case ecdsa.PublicKey, *ecdsa.PublicKey:
			// if it's ecdsa.PublicKey, it's more efficient to
			// go through the non-crypto.Signer route. Set isCryptoSigner to false
			isCryptoSigner = false
		}
	}

	if isCryptoSigner {
		return nil, cs, true, nil
	}

	pubkey, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return nil, nil, false, fmt.Errorf(`invalid key type %T. *ecdsa.PublicKey is required`, key)
	}

	return pubkey, nil, false, nil
}
