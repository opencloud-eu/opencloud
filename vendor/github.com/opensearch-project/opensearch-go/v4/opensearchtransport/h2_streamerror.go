// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import "fmt"

// h2StreamError is a local struct with the same field layout as
// golang.org/x/net/http2.StreamError. Go 1.21 added an As method on the
// vendored internal http2.StreamError (see net/http h2_error.go) that uses
// reflection to match target structs by field name and type convertibility.
// Because h2StreamError has identical field names and convertible types,
// errors.As(err, &target) succeeds for HTTP/2 stream errors
// returned by net/http without importing golang.org/x/net/http2.
//
// Implements error so that errors.As accepts h2StreamError as a target
// (errors.As panics if the target does not implement error or an interface).
//
// This catches RST_STREAM frames (e.g., REFUSED_STREAM, CANCEL) sent by the
// server. Note that HTTP/2 GOAWAY is handled transparently by Go's transport
// (which retries affected requests on a new connection) and surfaces as a
// separate unexported error type, not as a StreamError.
//
// Reference: https://go.dev/src/net/http/h2_error.go
// Reference: https://go.dev/src/net/http/h2_error_test.go
type h2StreamError struct {
	StreamID uint32
	Code     uint32 // http2.ErrCode is uint32; convertible via reflection
	Cause    error
}

func (e h2StreamError) Error() string {
	return fmt.Sprintf("stream error: stream ID %d; HTTP/2 error code = %d", e.StreamID, e.Code)
}
