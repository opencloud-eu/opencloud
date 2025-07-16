// Package signedurl provides interfaces and implementations for signing and verifying URLs.
package signedurl

import (
	"time"
)

type Signer interface {
	// Sign signs a URL
	Sign(url, principal string, ttl time.Duration) (string, error)
}

type Verifier interface {
	// Verify verifies a signed URL
	Verify(signedURL string) (string, error)
}

type SignedURLError struct {
	innerErr error
	message  string
}

// NewSignedURLError creates a new SignedURLError with the provided inner error and message.
func NewSignedURLError(innerErr error, message string) SignedURLError {
	return SignedURLError{
		innerErr: innerErr,
		message:  message,
	}
}

var ErrInvalidKey = NewSignedURLError(nil, "invalid key provided")

type SignatureVerificationError struct {
	SignedURLError
}

func NewSignatureVerificationError(innerErr error) SignatureVerificationError {
	return SignatureVerificationError{
		SignedURLError: SignedURLError{
			innerErr: innerErr,
			message:  "signature verification failed",
		},
	}
}

func (e SignatureVerificationError) Is(tgt error) bool {
	// Check if the target error is of type SignatureVerificationError
	if _, ok := tgt.(SignatureVerificationError); ok {
		return true
	}
	return false
}

// Error implements the error interface for errorConst.
func (e SignedURLError) Error() string {
	if e.innerErr != nil {
		return e.message + ": " + e.innerErr.Error()
	}
	return e.message
}

func (e SignedURLError) Unwrap() error {
	return e.innerErr
}
