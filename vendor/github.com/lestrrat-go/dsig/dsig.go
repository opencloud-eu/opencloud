// Package dsig provides digital signature operations for Go.
// It contains low-level signature generation and verification tools that
// can be used by other signing libraries
//
// The package follows these design principles:
// 1. Does minimal checking of input parameters (for performance); callers need to ensure that the parameters are valid.
// 2. All exported functions are strongly typed (i.e. they do not take `any` types unless they absolutely have to).
// 3. Does not rely on other high-level packages (standalone, except for internal packages).
package dsig

import (
	"crypto"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"sync"
)

// Family represents the cryptographic algorithm family
type Family int

const (
	InvalidFamily Family = iota
	HMAC
	RSA
	ECDSA
	EdDSAFamily
	Custom
	maxFamily
)

// String returns the string representation of the Family
func (f Family) String() string {
	switch f {
	case HMAC:
		return "HMAC"
	case RSA:
		return "RSA"
	case ECDSA:
		return "ECDSA"
	case EdDSAFamily:
		return "EdDSA"
	case Custom:
		return "Custom"
	default:
		return "InvalidFamily"
	}
}

// AlgorithmInfo contains metadata about a digital signature algorithm
type AlgorithmInfo struct {
	Family Family // The cryptographic family (HMAC, RSA, ECDSA, EdDSA)
	Meta   any    // Family-specific metadata
}

// HMACFamilyMeta contains metadata specific to HMAC algorithms
type HMACFamilyMeta struct {
	HashFunc func() hash.Hash // Hash function constructor
}

// RSAFamilyMeta contains metadata specific to RSA algorithms
type RSAFamilyMeta struct {
	Hash crypto.Hash // Hash algorithm
	PSS  bool        // Whether to use PSS padding (false = PKCS#1 v1.5)
}

// ECDSAFamilyMeta contains metadata specific to ECDSA algorithms
type ECDSAFamilyMeta struct {
	Hash crypto.Hash // Hash algorithm
}

// EdDSAFamilyMeta contains metadata specific to EdDSA algorithms
// Currently EdDSA doesn't need specific metadata, but this provides extensibility
type EdDSAFamilyMeta struct {
	// Reserved for future use
}

// Signer is an interface for custom signing implementations.
// For the Custom algorithm family, info.Meta must implement this interface
// to support signing. The implementation struct can carry any additional
// metadata it needs (hash functions, curves, etc.).
type Signer interface {
	Sign(key any, payload []byte, rand io.Reader) ([]byte, error)
}

// SignerWithOpts is an optional interface that Custom-family signers
// can implement to receive a per-call [crypto.SignerOpts]. The
// canonical use case is ML-DSA, whose Sign method accepts an
// *mldsa.Options carrying a domain-separation context that the plain
// [Signer] interface cannot convey. Custom Meta values that do not
// implement this interface still work with [SignWithOpts]: the
// dispatcher falls back to the plain [Signer.Sign] method and the opts
// argument is dropped.
//
// Implementing both [Signer] and SignerWithOpts is supported, but
// implementing only SignerWithOpts is sufficient because the dispatcher
// checks for it first.
type SignerWithOpts interface {
	SignWithOpts(key any, payload []byte, opts crypto.SignerOpts, rand io.Reader) ([]byte, error)
}

// Verifier is an interface for custom verification implementations.
// For the Custom algorithm family, info.Meta must implement this interface
// to support verification. The implementation struct can carry any additional
// metadata it needs (hash functions, curves, etc.).
type Verifier interface {
	Verify(key any, payload, signature []byte) error
}

// VerifierWithOpts is the verification counterpart of [SignerWithOpts].
// See [SignerWithOpts] for usage notes.
type VerifierWithOpts interface {
	VerifyWithOpts(key any, payload, signature []byte, opts crypto.SignerOpts) error
}

var algorithms = make(map[string]AlgorithmInfo)
var builtinAlgorithms = make(map[string]struct{})
var muAlgorithms sync.RWMutex

// RegisterAlgorithm registers a new digital signature algorithm with the specified family and metadata.
//
// info.Meta should contain extra metadata for some algorithms. HMAC, RSA, and ECDSA
// families need their respective metadata (HMACFamilyMeta, RSAFamilyMeta, and
// ECDSAFamilyMeta). Metadata for EdDSA is optional. For the Custom family, Meta
// must implement at least one of the Signer, SignerWithOpts, Verifier, or
// VerifierWithOpts interfaces.
//
// Re-registration of an already-registered algorithm name is rejected. Use
// UnregisterAlgorithm to remove it first if you need to replace it.
func RegisterAlgorithm(name string, info AlgorithmInfo) error {
	muAlgorithms.Lock()
	defer muAlgorithms.Unlock()

	if _, exists := algorithms[name]; exists {
		return fmt.Errorf("algorithm %s is already registered", name)
	}

	// Validate the metadata matches the family
	switch info.Family {
	case HMAC:
		if _, ok := info.Meta.(HMACFamilyMeta); !ok {
			return fmt.Errorf("invalid HMAC metadata for algorithm %s", name)
		}
	case RSA:
		if _, ok := info.Meta.(RSAFamilyMeta); !ok {
			return fmt.Errorf("invalid RSA metadata for algorithm %s", name)
		}
	case ECDSA:
		if _, ok := info.Meta.(ECDSAFamilyMeta); !ok {
			return fmt.Errorf("invalid ECDSA metadata for algorithm %s", name)
		}
	case EdDSAFamily:
		// EdDSA metadata is optional for now
	case Custom:
		_, isSigner := info.Meta.(Signer)
		_, isSignerWithOpts := info.Meta.(SignerWithOpts)
		_, isVerifier := info.Meta.(Verifier)
		_, isVerifierWithOpts := info.Meta.(VerifierWithOpts)
		if !isSigner && !isSignerWithOpts && !isVerifier && !isVerifierWithOpts {
			return fmt.Errorf("custom algorithm %s: Meta must implement Signer, SignerWithOpts, Verifier, or VerifierWithOpts", name)
		}
	default:
		return fmt.Errorf("unsupported algorithm family %s for algorithm %s", info.Family, name)
	}

	algorithms[name] = info
	return nil
}

// UnregisterAlgorithm removes a previously registered algorithm by name.
// Built-in algorithms cannot be unregistered.
// It is a no-op if the algorithm is not registered.
func UnregisterAlgorithm(name string) error {
	muAlgorithms.Lock()
	defer muAlgorithms.Unlock()

	if _, ok := builtinAlgorithms[name]; ok {
		return fmt.Errorf("algorithm %s is a built-in algorithm and cannot be unregistered", name)
	}

	delete(algorithms, name)
	return nil
}

// GetAlgorithmInfo retrieves the algorithm information for a given algorithm name.
// Returns the info and true if found, zero value and false if not found.
func GetAlgorithmInfo(name string) (AlgorithmInfo, bool) {
	muAlgorithms.RLock()
	defer muAlgorithms.RUnlock()

	info, ok := algorithms[name]
	return info, ok
}

func init() {
	// Register all standard algorithms with their metadata
	toRegister := map[string]AlgorithmInfo{
		// HMAC algorithms
		HMACWithSHA256: {
			Family: HMAC,
			Meta: HMACFamilyMeta{
				HashFunc: sha256.New,
			},
		},
		HMACWithSHA384: {
			Family: HMAC,
			Meta: HMACFamilyMeta{
				HashFunc: sha512.New384,
			},
		},
		HMACWithSHA512: {
			Family: HMAC,
			Meta: HMACFamilyMeta{
				HashFunc: sha512.New,
			},
		},

		// RSA PKCS#1 v1.5 algorithms
		RSAPKCS1v15WithSHA256: {
			Family: RSA,
			Meta: RSAFamilyMeta{
				Hash: crypto.SHA256,
				PSS:  false,
			},
		},
		RSAPKCS1v15WithSHA384: {
			Family: RSA,
			Meta: RSAFamilyMeta{
				Hash: crypto.SHA384,
				PSS:  false,
			},
		},
		RSAPKCS1v15WithSHA512: {
			Family: RSA,
			Meta: RSAFamilyMeta{
				Hash: crypto.SHA512,
				PSS:  false,
			},
		},

		// RSA PSS algorithms
		RSAPSSWithSHA256: {
			Family: RSA,
			Meta: RSAFamilyMeta{
				Hash: crypto.SHA256,
				PSS:  true,
			},
		},
		RSAPSSWithSHA384: {
			Family: RSA,
			Meta: RSAFamilyMeta{
				Hash: crypto.SHA384,
				PSS:  true,
			},
		},
		RSAPSSWithSHA512: {
			Family: RSA,
			Meta: RSAFamilyMeta{
				Hash: crypto.SHA512,
				PSS:  true,
			},
		},

		// ECDSA algorithms
		ECDSAWithP256AndSHA256: {
			Family: ECDSA,
			Meta: ECDSAFamilyMeta{
				Hash: crypto.SHA256,
			},
		},
		ECDSAWithP384AndSHA384: {
			Family: ECDSA,
			Meta: ECDSAFamilyMeta{
				Hash: crypto.SHA384,
			},
		},
		ECDSAWithP521AndSHA512: {
			Family: ECDSA,
			Meta: ECDSAFamilyMeta{
				Hash: crypto.SHA512,
			},
		},

		// EdDSA algorithm
		EdDSA: {
			Family: EdDSAFamily,
			Meta:   EdDSAFamilyMeta{},
		},
	}

	for name, info := range toRegister {
		if err := RegisterAlgorithm(name, info); err != nil {
			panic(fmt.Sprintf("failed to register algorithm %s: %v", name, err))
		}
		builtinAlgorithms[name] = struct{}{}
	}
}

