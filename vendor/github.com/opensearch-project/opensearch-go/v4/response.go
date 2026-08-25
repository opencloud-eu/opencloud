// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.
//
// Modifications Copyright OpenSearch Contributors. See
// GitHub history for details.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opensearch

import (
	"fmt"
	"io"
	"net/http"
)

const httpStatusCodeThreshold = 299

// Response represents the API response.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       io.ReadCloser
	rawBody    []byte
}

// RawBody returns the buffered response bytes for inspection or
// comparison testing.
//
// The returned slice aliases memory owned by Response and may be backed
// by pooled storage in future implementations. Callers must NOT mutate
// the slice and must NOT retain it past the lifetime of the request
// handling: copy with bytes.Clone if either is needed, or use
// HijackBody to transfer ownership to the caller.
//
// Populated only for non-error responses where Client.Do successfully
// decoded the body into a typed dataPointer. Returns nil when:
//
//   - The response was streamed without buffering (no dataPointer).
//   - The response was an error (4xx/5xx); read Body directly for
//     error-response bodies, or use ParseError to extract a typed error.
func (r *Response) RawBody() []byte {
	return r.rawBody
}

// HijackBody returns the buffered response bytes and transfers
// ownership to the caller, clearing r.rawBody so the Response no
// longer references the buffer. After Hijack the caller may mutate
// or retain the slice indefinitely; subsequent RawBody calls return
// nil. Use this when the buffer outlives the Response (logging,
// background processing, test fixtures).
func (r *Response) HijackBody() []byte {
	body := r.rawBody
	r.rawBody = nil
	return body
}

// NewResponse creates a Response with the given status, body, and headers.
func NewResponse(statusCode int, body io.ReadCloser, header http.Header) *Response {
	return &Response{
		StatusCode: statusCode,
		Body:       body,
		Header:     header,
	}
}

// String returns the response as a string.
func (r Response) String() string {
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			body = []byte(fmt.Sprintf("<error reading response body: %v>", err))
		}
		return fmt.Sprintf("%s %s", r.Status(), body)
	}
	return r.Status()
}

// Status retuens the response status as string.
func (r Response) Status() string {
	status := http.StatusText(r.StatusCode)
	if status == "" {
		status = "<nil>"
	}
	return fmt.Sprintf("[%d %s]", r.StatusCode, status)
}

// IsError returns true when the response status indicates failure.
func (r *Response) IsError() bool {
	return r.StatusCode > httpStatusCodeThreshold
}
