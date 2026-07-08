// Provides common utility functions for HTTP.
package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type readCloser struct {
	io.Reader
	closer func() error
}

func (c readCloser) Close() error {
	if c.closer != nil {
		return c.closer()
	}
	return nil
}

func combinedReader(peekBuffer *bytes.Buffer, body io.Reader) io.Reader {
	if body != nil && body != http.NoBody {
		if peekBuffer != nil {
			return io.MultiReader(peekBuffer, body)
		} else {
			return body
		}
	} else {
		if peekBuffer != nil {
			return peekBuffer
		} else {
			return http.NoBody
		}
	}
}

// Reads up to <size> amount of bytes from the response, and returns a ReadCloser that concatenates over those
// bytes as well as the rest of the bytes streamed in <body>, to ensure that the whole response can still be
// read.
//
// Note that if <size> is 0, then this function does nothing.
//
// Furthermore, this function never closes the <body> Reader, that is left to the caller.
//
// # Parameters
//   - body: the HTTP response body, which may be nil or http.NoBody
//   - size: the number of bytes that will be read from the beginning of the response body,
//     which may be 0 (in which case this function won't do anything)
//
// # Return values
//   - a Reader that provides the full response, including the first bytes that were peeked
//     up to <size> bytes of the beginning of the response body
//   - the first <size> bytes of the response
//   - a boolean: whether the peeked bytes contain the full response or whether it's truncated
//   - an error if an error occured while reading the body response
func PeekResponse(body io.ReadCloser, size int64) (io.ReadCloser, []byte, bool, error) {
	var peekBuffer *bytes.Buffer = nil
	truncated := false
	peek := []byte{}
	if body != nil && body != http.NoBody && size > 0 {
		peekBuffer = new(bytes.Buffer)
		limitedReader := io.LimitReader(body, size)
		tee := io.TeeReader(limitedReader, peekBuffer)
		if _, err := io.Copy(io.Discard, tee); err != nil {
			return nil, peek, false, err
		}

		if int64(peekBuffer.Len()) >= size {
			truncated = true
		}

		peek = peekBuffer.Bytes()

		return readCloser{
			closer: body.Close,
			Reader: combinedReader(peekBuffer, body),
		}, peek, truncated, nil
	} else {
		return body, peek, truncated, nil
	}
}

// Extracts information from an HTTP request and passes it to a function, typically for logging.
//
// It also extracts the <maxBodySize> first bytes of the request body, if maxBodySize is > 0.
func DumpHttpRequest(req *http.Request, maxBodySize int64, closure func(method string, uri string, content string, truncated bool)) error {
	var logBuilder bytes.Buffer
	uri := ""
	if req.URL != nil {
		uri = req.URL.String()
	}
	fmt.Fprintf(&logBuilder, "%s %s\n", req.Method, uri)
	for key, values := range req.Header {
		if key == "Authorization" {
			continue
		}
		for _, value := range values {
			fmt.Fprintf(&logBuilder, "%s: %s\n", key, value)
		}
	}
	truncated := false
	if maxBodySize >= 0 && req.Body != nil && req.Body != http.NoBody {
		peekBuffer := new(bytes.Buffer)
		limitedReader := io.LimitReader(req.Body, maxBodySize)
		tee := io.TeeReader(limitedReader, peekBuffer)
		if _, err := io.Copy(io.Discard, tee); err != nil {
			return fmt.Errorf("failed to peek at the request body for tracing: %v", err)
		} else {
			logBuilder.Write(peekBuffer.Bytes())
			if int64(peekBuffer.Len()) >= maxBodySize {
				truncated = true
			}
			fullBodyReader := io.MultiReader(peekBuffer, req.Body)
			req.Body = io.NopCloser(fullBodyReader)
		}
	}
	closure(req.Method, uri, logBuilder.String(), truncated)
	return nil
}

// Extracts information from an HTTP response and passes it to a function, typically for logging.
//
// It also extracts the <maxBodySize> first bytes of the response body, if maxBodySize is > 0.
func DumpHttpResponse(resp *http.Response, body io.ReadCloser, maxBodySize int64,
	closure func(method string, uri string, content string, truncated bool),
) (io.ReadCloser, error) {
	var logBuilder bytes.Buffer
	fmt.Fprintf(&logBuilder, "%s\n", resp.Status)
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Fprintf(&logBuilder, "%s: %s\n", key, value)
		}
	}

	var peekBuffer *bytes.Buffer = nil
	truncated := false
	if body != nil && body != http.NoBody && maxBodySize > 0 {
		peekBuffer = new(bytes.Buffer)
		limitedReader := io.LimitReader(body, maxBodySize)
		tee := io.TeeReader(limitedReader, peekBuffer)
		if _, err := io.Copy(io.Discard, tee); err != nil {
			return nil, err
		}

		if int64(peekBuffer.Len()) >= maxBodySize {
			truncated = true
		}

		logBuilder.Write(peekBuffer.Bytes())
	}

	req := resp.Request
	if req != nil {
		closure(req.Method, req.URL.String(), logBuilder.String(), truncated)
	} else {
		closure("", "", logBuilder.String(), truncated)
	}

	if peekBuffer != nil {
		return readCloser{
			closer: body.Close,
			Reader: combinedReader(peekBuffer, body),
		}, nil
	} else {
		return body, nil
	}
}
