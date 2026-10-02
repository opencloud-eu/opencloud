# github.com/lestrrat-go/dsig [![CI](https://github.com/lestrrat-go/dsig/actions/workflows/ci.yml/badge.svg)](https://github.com/lestrrat-go/dsig/actions/workflows/ci.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/lestrrat-go/dsig.svg)](https://pkg.go.dev/github.com/lestrrat-go/dsig)

Go module providing low-level digital signature operations.

While there are many standards for generating and verifying digital signatures, the core operations are virtually the same. This module implements the core functionality of digital signature generation / verifications in a framework agnostic way.

# Features

* RSA signatures (PKCS1v15 and PSS)
* ECDSA signatures (P-256, P-384, P-521)
* EdDSA signatures (Ed25519, Ed448)
* ML-DSA post-quantum signatures (ML-DSA-44, ML-DSA-65, ML-DSA-87), on Go 1.27 and later
* HMAC signatures (SHA-256, SHA-384, SHA-512)
* Support for crypto.Signer interface
* Custom algorithm registration via `Signer`/`Verifier` interfaces

# SYNOPSIS

<!-- INCLUDE(examples/dsig_readme_example_test.go) -->
```go
package examples_test

import (
  "crypto/ecdsa"
  "crypto/ed25519"
  "crypto/elliptic"
  "crypto/rand"
  "crypto/rsa"
  "fmt"

  "github.com/lestrrat-go/dsig"
)

func Example() {
  payload := []byte("hello world")

  // RSA signing and verification
  {
    privKey, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
      fmt.Printf("failed to generate RSA key: %s\n", err)
      return
    }

    // Sign with RSA-PSS SHA256
    signature, err := dsig.Sign(privKey, dsig.RSAPSSWithSHA256, payload, nil)
    if err != nil {
      fmt.Printf("failed to sign with RSA: %s\n", err)
      return
    }

    // Verify with RSA-PSS SHA256
    err = dsig.Verify(&privKey.PublicKey, dsig.RSAPSSWithSHA256, payload, signature)
    if err != nil {
      fmt.Printf("failed to verify RSA signature: %s\n", err)
      return
    }
  }

  // ECDSA signing and verification
  {
    privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
      fmt.Printf("failed to generate ECDSA key: %s\n", err)
      return
    }

    // Sign with ECDSA P-256 SHA256
    signature, err := dsig.Sign(privKey, dsig.ECDSAWithP256AndSHA256, payload, nil)
    if err != nil {
      fmt.Printf("failed to sign with ECDSA: %s\n", err)
      return
    }

    // Verify with ECDSA P-256 SHA256
    err = dsig.Verify(&privKey.PublicKey, dsig.ECDSAWithP256AndSHA256, payload, signature)
    if err != nil {
      fmt.Printf("failed to verify ECDSA signature: %s\n", err)
      return
    }
  }

  // EdDSA signing and verification
  {
    pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
      fmt.Printf("failed to generate Ed25519 key: %s\n", err)
      return
    }

    // Sign with EdDSA
    signature, err := dsig.Sign(privKey, dsig.EdDSA, payload, nil)
    if err != nil {
      fmt.Printf("failed to sign with EdDSA: %s\n", err)
      return
    }

    // Verify with EdDSA
    err = dsig.Verify(pubKey, dsig.EdDSA, payload, signature)
    if err != nil {
      fmt.Printf("failed to verify EdDSA signature: %s\n", err)
      return
    }
  }

  // HMAC signing and verification
  {
    key := []byte("secret-key")

    // Sign with HMAC SHA256
    signature, err := dsig.Sign(key, dsig.HMACWithSHA256, payload, nil)
    if err != nil {
      fmt.Printf("failed to sign with HMAC: %s\n", err)
      return
    }

    // Verify with HMAC SHA256
    err = dsig.Verify(key, dsig.HMACWithSHA256, payload, signature)
    if err != nil {
      fmt.Printf("failed to verify HMAC signature: %s\n", err)
      return
    }
  }
  // OUTPUT:
}
```
source: [examples/dsig_readme_example_test.go](https://github.com/lestrrat-go/dsig/blob/v1/examples/dsig_readme_example_test.go)
<!-- END INCLUDE -->

# Supported Algorithms

| Constant | Algorithm | Key Type |
|----------|-----------|----------|
| `HMACWithSHA256` | HMAC using SHA-256 | []byte |
| `HMACWithSHA384` | HMAC using SHA-384 | []byte |
| `HMACWithSHA512` | HMAC using SHA-512 | []byte |
| `RSAPKCS1v15WithSHA256` | RSA PKCS#1 v1.5 using SHA-256 | *rsa.PrivateKey / *rsa.PublicKey |
| `RSAPKCS1v15WithSHA384` | RSA PKCS#1 v1.5 using SHA-384 | *rsa.PrivateKey / *rsa.PublicKey |
| `RSAPKCS1v15WithSHA512` | RSA PKCS#1 v1.5 using SHA-512 | *rsa.PrivateKey / *rsa.PublicKey |
| `RSAPSSWithSHA256` | RSA PSS using SHA-256 | *rsa.PrivateKey / *rsa.PublicKey |
| `RSAPSSWithSHA384` | RSA PSS using SHA-384 | *rsa.PrivateKey / *rsa.PublicKey |
| `RSAPSSWithSHA512` | RSA PSS using SHA-512 | *rsa.PrivateKey / *rsa.PublicKey |
| `ECDSAWithP256AndSHA256` | ECDSA using P-256 and SHA-256 | *ecdsa.PrivateKey / *ecdsa.PublicKey |
| `ECDSAWithP384AndSHA384` | ECDSA using P-384 and SHA-384 | *ecdsa.PrivateKey / *ecdsa.PublicKey |
| `ECDSAWithP521AndSHA512` | ECDSA using P-521 and SHA-512 | *ecdsa.PrivateKey / *ecdsa.PublicKey |
| `EdDSA` | EdDSA using Ed25519 or Ed448 | ed25519.PrivateKey / ed25519.PublicKey |
| `MLDSA44` | ML-DSA-44 (FIPS 204), NIST level 2 | *mldsa.PrivateKey / *mldsa.PublicKey |
| `MLDSA65` | ML-DSA-65 (FIPS 204), NIST level 3 | *mldsa.PrivateKey / *mldsa.PublicKey |
| `MLDSA87` | ML-DSA-87 (FIPS 204), NIST level 5 | *mldsa.PrivateKey / *mldsa.PublicKey |

The three ML-DSA algorithms need Go 1.27 or later, which is when `crypto/mldsa`
joins the standard library. On earlier toolchains the constants are not declared
and the algorithms are not registered.

Name the constant that matches the key you generated. A key knows its own
parameter set, and naming a different one is an error, so a key cannot be used
under a weaker set by accident:

```go
sk, _ := mldsa.GenerateKey(mldsa.MLDSA65())

sig, _ := dsig.Sign(sk, dsig.MLDSA65, payload, nil)
err := dsig.Verify(sk.PublicKey(), dsig.MLDSA65, payload, sig)

_, err = dsig.Sign(sk, dsig.MLDSA44, payload, nil)
// dsig.SignWithOpts: ML-DSA parameter set mismatch: key is ML-DSA-65, algorithm is ML-DSA-44
```

## ML-DSA context strings

ML-DSA can mix a caller-chosen string into the signature. Give each job a
different context and one key can sign for several of them without a signature
made for one job verifying as another, so a login token cannot be presented as
a file receipt.

Signing and verifying must use the same context. A verifier that supplies the
wrong one, or none at all, sees an ordinary invalid signature and cannot tell
which mistake was made:

```go
login := &mldsa.Options{Context: "my-app/login-token"}
receipt := &mldsa.Options{Context: "my-app/file-receipt"}

sig, _ := dsig.SignWithOpts(sk, dsig.MLDSA65, payload, login, nil)

err := dsig.VerifyWithOpts(sk.PublicKey(), dsig.MLDSA65, payload, sig, login)
// nil

err = dsig.VerifyWithOpts(sk.PublicKey(), dsig.MLDSA65, payload, sig, receipt)
// mldsa: invalid signature
```

A context is at most 255 bytes, and it is empty when opts is nil.

# Description

This library provides low-level digital signature operations. It does minimal parameter validation for performance, uses strongly typed APIs, and has minimal dependencies.

# Contributions

## Issues

For bug reports and feature requests, please include failing tests when possible.

## Pull Requests

Please include tests that exercise your changes.

# Related Libraries

* [github.com/lestrrat-go/jwx](https://github.com/lestrrat-go/jwx) - JOSE (JWA/JWE/JWK/JWS/JWT) implementation
