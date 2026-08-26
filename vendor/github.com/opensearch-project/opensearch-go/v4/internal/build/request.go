// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

// Package build constructs [http.Request] values for the OpenSearch client
// without the overhead of [http.NewRequest] (which calls [url.Parse] on every
// invocation). All paths produced by the typed path builders are simple
// relative paths with no scheme, host, query, or fragment, so parsing is
// unnecessary. The transport layer prepends the base URL before sending.
package build

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	// HeaderContentType is the canonical HTTP Content-Type header name.
	HeaderContentType = "Content-Type"

	// ContentTypeJSON is the default body Content-Type for OpenSearch
	// requests.
	ContentTypeJSON = "application/json"

	// ContentTypeNDJSON is the body Content-Type for newline-delimited
	// JSON request bodies (e.g. _bulk, _msearch, _msearch/template).
	ContentTypeNDJSON = "application/x-ndjson"
)

// headerContentType retains the lowercase name for internal lookups
// where the existing call sites (and the http.Header map's canonical
// form) use the unexported identifier.
const headerContentType = HeaderContentType

var headerContentTypeJSON = []string{ContentTypeJSON}

// NullJSON is the JSON `null` literal as a byte slice. Generated MarshalJSON
// implementations return it for the empty/zero case.
var NullJSON = []byte("null")

// errMissingMethod is returned when an empty HTTP method is passed to Request.
var errMissingMethod = errors.New("HTTP method must be specified")

var (
	errInvalidMethod = errors.New("net/http: invalid method")
	errInvalidPath   = errors.New("invalid percent-encoding in path")
)

// requestBuf coalesces http.Request and url.URL into a single heap allocation.
// Taking the address of buf.req causes the entire struct to escape, but only
// as one object rather than two separate allocations for Request and URL.
type requestBuf struct {
	req http.Request
	url url.URL
}

// Request constructs an [http.Request] from a method, path, body, query
// parameters, and headers. The path must be a relative URL path (no scheme or
// host); the transport layer prepends the base URL before sending.
//
// When body != nil, Content-Type defaults to "application/json"; callers that
// need a different body type (e.g. "application/x-ndjson" for _bulk) must
// supply Content-Type in headers and the caller's value wins.
//
// Header is intentionally left nil when body is nil and no custom headers are
// passed. The transport layer ([opensearch.Client.Perform]) lazily allocates
// the Header map before setting User-Agent and other transport headers. This
// keeps the fast path (GET with no body) down to 2 heap allocations: the
// coalesced requestBuf and the path string from the path builder.
func Request(method string, path string, body io.Reader, params map[string]string, headers http.Header) (*http.Request, error) {
	if method == "" {
		return nil, errMissingMethod
	}

	if !validMethod(method) {
		return nil, errInvalidMethod
	}

	var buf requestBuf
	// Setting both RawPath and Path ensures url.URL.EscapedPath() honors
	// the percent-encoded form produced by the path builders (e.g. %2F for
	// a literal slash inside a document ID). Without RawPath, EscapedPath
	// re-encodes Path from scratch and leaves '/' and '..' untouched,
	// enabling path traversal and segment injection.
	//
	// A PathUnescape failure means the path contains an invalid percent
	// sequence (e.g. "%ZZ"). Falling through and assigning the raw bytes
	// to Path would cause EscapedPath() to silently re-encode the literal
	// '%' as "%25", corrupting the wire form. Reject the request.
	buf.url.RawPath = path
	p, err := url.PathUnescape(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errInvalidPath, err.Error())
	}
	buf.url.Path = p
	buf.req.Method = method
	buf.req.URL = &buf.url
	buf.req.Proto = "HTTP/1.1"
	buf.req.ProtoMajor = 1
	buf.req.ProtoMinor = 1

	if body != nil {
		rc, ok := body.(io.ReadCloser)
		if !ok {
			rc = io.NopCloser(body)
		}
		buf.req.Body = rc

		// Snapshot known reader types so GetBody can replay the body for
		// retries. Mirrors the behavior of http.NewRequest.
		switch v := body.(type) {
		case *bytes.Buffer:
			buf.req.ContentLength = int64(v.Len())
			b := v.Bytes()
			buf.req.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(b)), nil
			}
		case *bytes.Reader:
			buf.req.ContentLength = int64(v.Len())
			snapshot := *v
			buf.req.GetBody = func() (io.ReadCloser, error) {
				r := snapshot
				return io.NopCloser(&r), nil
			}
		case *strings.Reader:
			buf.req.ContentLength = int64(v.Len())
			snapshot := *v
			buf.req.GetBody = func() (io.ReadCloser, error) {
				r := snapshot
				return io.NopCloser(&r), nil
			}
		}

		buf.req.Header = make(http.Header, len(headers)+1)
		// Seed default Content-Type only when the caller did not supply
		// one; otherwise the caller's value wins (e.g. application/x-ndjson
		// for _bulk and _msearch).
		if _, ok := headers[headerContentType]; !ok {
			buf.req.Header[headerContentType] = headerContentTypeJSON
		}
	} else if len(headers) > 0 {
		buf.req.Header = make(http.Header, len(headers))
	}

	for k, vv := range headers {
		for _, v := range vv {
			buf.req.Header.Add(k, v)
		}
	}

	if len(params) > 0 {
		q := buf.url.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		buf.url.RawQuery = q.Encode()
	}

	return &buf.req, nil
}

// isTokenChar reports whether c is a valid HTTP token character per
// RFC 7230 section 3.2.6. The set is the same as net/http's internal
// isTokenTable.
func isTokenChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z':
		return true
	case c >= 'A' && c <= 'Z':
		return true
	case c >= '0' && c <= '9':
		return true
	}
	switch c {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	}
	return false
}

// validMethod reports whether method is a syntactically valid HTTP
// method per RFC 7230. Empty methods are rejected by the caller; this
// function rejects any non-token byte (control chars, spaces, lowercase
// letters within ASCII range are still tokens, but methods like "get"
// or "GET,POST" or "PROP[]FIND" use non-token punctuation and are
// rejected here).
func validMethod(method string) bool {
	if method == "" {
		return false
	}
	for i := range method {
		if !isTokenChar(method[i]) {
			return false
		}
	}
	return true
}
