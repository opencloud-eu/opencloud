package handler

import (
	"maps"
	"net/http"
	"strconv"
)

// HTTPRequest contains basic details of an incoming HTTP request.
type HTTPRequest struct {
	// Method is the HTTP method, e.g. POST or PATCH.
	Method string
	// URI is the full HTTP request URI, e.g. /files/fooo.
	URI string
	// RemoteAddr contains the network address that sent the request.
	RemoteAddr string
	// Header contains all HTTP headers as present in the HTTP request.
	Header http.Header
}

type HTTPHeader map[string]string

// HTTPResponse contains basic details of an outgoing HTTP response.
type HTTPResponse struct {
	// StatusCode is status code, e.g. 200 or 400.
	StatusCode int
	// Body is the response body.
	Body string
	// Header contains additional HTTP headers for the response.
	Header HTTPHeader
}

// writeTo writes the HTTP response into w, as specified by the fields in resp.
func (resp HTTPResponse) writeTo(w http.ResponseWriter) {
	headers := w.Header()
	for key, value := range resp.Header {
		headers.Set(key, value)
	}

	if len(resp.Body) > 0 {
		headers.Set("Content-Length", strconv.Itoa(len(resp.Body)))
	}

	w.WriteHeader(resp.StatusCode)

	if len(resp.Body) > 0 {
		w.Write([]byte(resp.Body))
	}
}

// MergeWith returns a copy of resp1, where non-default values from resp2 overwrite
// values from resp1.
func (resp1 HTTPResponse) MergeWith(resp2 HTTPResponse) HTTPResponse {
	// Clone the response 1 and use it as a basis
	newResp := resp1

	// Take the status code and body from response 2 to
	// overwrite values from response 1.
	if resp2.StatusCode != 0 {
		newResp.StatusCode = resp2.StatusCode
	}

	if len(resp2.Body) > 0 {
		newResp.Body = resp2.Body
	}

	// For the headers, me must make a new map to avoid writing
	// into the header map from response 1.
	newResp.Header = make(HTTPHeader, len(resp1.Header)+len(resp2.Header))

	maps.Copy(newResp.Header, resp1.Header)

	maps.Copy(newResp.Header, resp2.Header)

	return newResp
}
