//go:build go1.27

package dsig

import (
	"crypto"
	"crypto/mldsa"
	"fmt"
	"io"
)

// ML-DSA signature algorithms, the post-quantum scheme specified in FIPS 204.
// The three names identify the three parameter sets, which differ in security
// level and in key and signature sizes.
//
// These names match what crypto/mldsa's Parameters.String reports, so the
// parameter set a key carries can be compared against the algorithm name
// directly.
//
// ML-DSA is available only when dsig is built with Go 1.27 or later, which is
// when crypto/mldsa becomes part of the standard library. On earlier
// toolchains these algorithms are not registered and not declared.
const (
	MLDSA44 = "ML-DSA-44"
	MLDSA65 = "ML-DSA-65"
	MLDSA87 = "ML-DSA-87"
)

func init() {
	for _, params := range []mldsa.Parameters{mldsa.MLDSA44(), mldsa.MLDSA65(), mldsa.MLDSA87()} {
		name := params.String()
		if err := RegisterAlgorithm(name, AlgorithmInfo{
			Family: MLDSAFamily,
			Meta:   &mldsaAlgorithm{params: params},
		}); err != nil {
			panic(fmt.Sprintf("failed to register algorithm %s: %v", name, err))
		}
		builtinAlgorithms[name] = struct{}{}
	}
}

// SignMLDSA generates an ML-DSA signature for the given payload.
//
// opts may be nil, which signs payload directly with no context. Pass an
// *[mldsa.Options] to supply a domain-separation context, which [VerifyMLDSA]
// then requires to match.
//
// opts is a [crypto.SignerOpts] so that both of ML-DSA's signing modes stay
// expressible. Passing [crypto.MLDSAMu] means payload holds a pre-hashed μ
// message representative. That mode is a shortcut for callers who already have
// μ, and it produces an ordinary signature; [VerifyMLDSA] checks it against the
// original message, and the verify side needs no counterpart.
//
// crypto/mldsa rejects any other opts value, so a mistaken type cannot be
// silently downgraded to a context-free signature.
func SignMLDSA(key *mldsa.PrivateKey, payload []byte, opts crypto.SignerOpts) ([]byte, error) {
	if key == nil {
		return nil, fmt.Errorf(`dsig.SignMLDSA: key cannot be nil`)
	}
	// The io.Reader argument is ignored by crypto/mldsa; signing draws its own
	// randomness. SignDeterministic is the variant that draws none.
	return key.Sign(nil, payload, opts)
}

// VerifyMLDSA verifies an ML-DSA signature for the given payload.
//
// opts may be nil. It must carry the same Context that was used to produce the
// signature, otherwise verification fails.
//
// Verification has a single mode, so opts is a concrete *[mldsa.Options]. μ is
// derived from the message, so a signature made from a pre-hashed μ verifies
// here against the original message.
func VerifyMLDSA(key *mldsa.PublicKey, payload, signature []byte, opts *mldsa.Options) error {
	if key == nil {
		return fmt.Errorf(`dsig.VerifyMLDSA: key cannot be nil`)
	}
	return mldsa.Verify(key, payload, signature, opts)
}

// mldsaAlgorithm is the Custom-family adapter that binds one ML-DSA parameter
// set to the registry. It carries the parameter set so that every operation can
// check the caller's key against the algorithm that was asked for.
type mldsaAlgorithm struct {
	params mldsa.Parameters
}

// requireMLDSAParams reports whether a caller-supplied key belongs to the
// parameter set this algorithm was registered for. crypto/mldsa's Parameters is
// a comparable value naming one of the three FIPS 204 sets, so a plain
// comparison suffices.
//
// The check matters because the key owns the parameter set, and the call only
// names one. Without it, an ML-DSA-65 key would happily produce and verify
// ML-DSA-65 signatures while the caller believed it had selected ML-DSA-44.
// Anything that reads the algorithm name to decide a post-quantum security
// level would then be misled, so the mismatch is an error.
func (a *mldsaAlgorithm) requireMLDSAParams(got mldsa.Parameters) error {
	if got != a.params {
		return fmt.Errorf(`ML-DSA parameter set mismatch: key is %s, algorithm is %s`, got, a.params)
	}
	return nil
}

func (a *mldsaAlgorithm) privateKey(key any) (*mldsa.PrivateKey, error) {
	sk, ok := key.(*mldsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf(`expected *mldsa.PrivateKey, got %T`, key)
	}
	if err := a.requireMLDSAParams(sk.PublicKey().Parameters()); err != nil {
		return nil, err
	}
	return sk, nil
}

// publicKey narrows the key types the verify surface accepts. A private key is
// allowed so callers holding only one half do not have to unwrap it themselves.
func (a *mldsaAlgorithm) publicKey(key any) (*mldsa.PublicKey, error) {
	var pk *mldsa.PublicKey
	switch k := key.(type) {
	case *mldsa.PublicKey:
		pk = k
	case *mldsa.PrivateKey:
		pk = k.PublicKey()
	default:
		return nil, fmt.Errorf(`expected *mldsa.PublicKey or *mldsa.PrivateKey, got %T`, key)
	}
	if err := a.requireMLDSAParams(pk.Parameters()); err != nil {
		return nil, err
	}
	return pk, nil
}

// mldsaOptions narrows a crypto.SignerOpts to the concrete type crypto/mldsa
// accepts. A non-nil value of any other type is an error. Dropping it would let
// a caller believe their Context was in force while the operation actually ran
// with an empty context, which is a signature substitution vector for schemes
// that rely on domain separation.
func mldsaOptions(opts crypto.SignerOpts) (*mldsa.Options, error) {
	if opts == nil {
		return nil, nil
	}
	mldsaOpts, ok := opts.(*mldsa.Options)
	if !ok {
		return nil, fmt.Errorf(`expected *mldsa.Options, got %T`, opts)
	}
	return mldsaOpts, nil
}

func (a *mldsaAlgorithm) Sign(key any, payload []byte, _ io.Reader) ([]byte, error) {
	sk, err := a.privateKey(key)
	if err != nil {
		return nil, fmt.Errorf(`dsig.Sign: %w`, err)
	}
	return SignMLDSA(sk, payload, nil)
}

// SignWithOpts implements [SignerWithOpts], forwarding an *mldsa.Options
// Context to crypto/mldsa.
func (a *mldsaAlgorithm) SignWithOpts(key any, payload []byte, opts crypto.SignerOpts, _ io.Reader) ([]byte, error) {
	sk, err := a.privateKey(key)
	if err != nil {
		return nil, fmt.Errorf(`dsig.SignWithOpts: %w`, err)
	}
	// Validated but deliberately not narrowed. SignMLDSA takes a
	// crypto.SignerOpts, so converting to a typed nil here would hand
	// crypto/mldsa a non-nil interface holding a nil pointer.
	if _, err := mldsaOptions(opts); err != nil {
		return nil, fmt.Errorf(`dsig.SignWithOpts: %w`, err)
	}
	return SignMLDSA(sk, payload, opts)
}

func (a *mldsaAlgorithm) Verify(key any, payload, signature []byte) error {
	pk, err := a.publicKey(key)
	if err != nil {
		return fmt.Errorf(`dsig.Verify: %w`, err)
	}
	return VerifyMLDSA(pk, payload, signature, nil)
}

// VerifyWithOpts implements [VerifierWithOpts]. See [SignerWithOpts] for the
// rationale on rejecting a foreign opts type.
func (a *mldsaAlgorithm) VerifyWithOpts(key any, payload, signature []byte, opts crypto.SignerOpts) error {
	pk, err := a.publicKey(key)
	if err != nil {
		return fmt.Errorf(`dsig.VerifyWithOpts: %w`, err)
	}
	mldsaOpts, err := mldsaOptions(opts)
	if err != nil {
		return fmt.Errorf(`dsig.VerifyWithOpts: %w`, err)
	}
	return VerifyMLDSA(pk, payload, signature, mldsaOpts)
}
