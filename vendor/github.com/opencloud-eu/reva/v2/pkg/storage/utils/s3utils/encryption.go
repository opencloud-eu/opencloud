// SPDX-License-Identifier: Apache-2.0

// Package s3utils provides shared configuration helpers for S3 storage clients.
package s3utils

import (
	"encoding/base64"
	"fmt"

	"github.com/minio/minio-go/v7/pkg/encrypt"
)

// NewSSEC validates a base64-encoded AES-256 customer key and creates the SDK
// encryption options. An empty key leaves server-side encryption unspecified.
// Customer keys must only be sent over HTTPS.
func NewSSEC(key string, secure bool) (encrypt.ServerSide, error) {
	if key == "" {
		return nil, nil
	}
	if !secure {
		return nil, fmt.Errorf("SSE-C requires an HTTPS S3 endpoint")
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("SSE-C key must be base64-encoded")
	}
	if len(decoded) != 32 {
		return nil, fmt.Errorf("SSE-C key must decode to exactly 32 bytes")
	}
	return encrypt.NewSSEC(decoded)
}
