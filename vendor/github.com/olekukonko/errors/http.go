package errors

import (
	"fmt"
	"net/http"
)

// httpConfig holds resolved options for an HTTPError call.
type httpConfig struct {
	fallbackCode int
	includeBody  bool
	bodyFn       func(error) string
}

// HTTPOption configures an HTTPError call.
type HTTPOption func(*httpConfig)

// WithFallbackCode sets the HTTP status used when err carries no valid code.
// Default is 500 (Internal Server Error).
func WithFallbackCode(code int) HTTPOption {
	return func(c *httpConfig) { c.fallbackCode = code }
}

// WithBody controls whether the error message is written as the response body.
// Default is true.
func WithBody(include bool) HTTPOption {
	return func(c *httpConfig) { c.includeBody = include }
}

// WithBodyFunc sets a custom function that produces the response body string
// from the error. Overrides WithBody when set.
//
// Example — return JSON instead of plain text:
//
//	errors.HTTPError(w, err,
//	    errors.WithBodyFunc(func(e error) string {
//	        return fmt.Sprintf(`{"error":%q}`, e.Error())
//	    }),
//	)
func WithBodyFunc(fn func(error) string) HTTPOption {
	return func(c *httpConfig) { c.bodyFn = fn }
}

// HTTPError writes err to w as an HTTP error response.
//
// Status code resolution (first match wins):
// err is *Error with Code() in the valid HTTP range (100–599)
// WithFallbackCode option (default 500)
//
// Content-Type defaults to text/plain unless WithBodyFunc provides content
// that implies a different type (caller must set the header themselves in
// that case — use WithBodyFunc + manual header setting).
//
// Example — simplest usage, plain text body, 500 fallback:
//
//	errors.HTTPError(w, err)
//
// Example — custom fallback status:
//
//	errors.HTTPError(w, err, errors.WithFallbackCode(http.StatusBadGateway))
//
// Example — suppress body (header only):
//
//	errors.HTTPError(w, err, errors.WithBody(false))
//
// Example — JSON body:
//
//	errors.HTTPError(w, err,
//	    errors.WithBodyFunc(func(e error) string {
//	        return fmt.Sprintf(`{"error":%q,"code":%d}`,
//	            e.Error(), errors.HTTPStatusCode(e, 500))
//	    }),
//	)
func HTTPError(w http.ResponseWriter, err error, opts ...HTTPOption) {
	cfg := &httpConfig{
		fallbackCode: http.StatusInternalServerError,
		includeBody:  true,
	}
	for _, o := range opts {
		o(cfg)
	}

	code := HTTPStatusCode(err, cfg.fallbackCode)

	if cfg.bodyFn != nil {
		w.WriteHeader(code)
		if err != nil {
			_, _ = fmt.Fprint(w, cfg.bodyFn(err))
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	if cfg.includeBody && err != nil {
		_, _ = fmt.Fprintln(w, err.Error())
	}
}

// HTTPStatusCode returns the HTTP status code embedded in err.
// If err is nil, has no code, or the code is outside the valid HTTP range
// (100–599), defaultCode is returned.
//
// Example:
//
//	status := errors.HTTPStatusCode(err, http.StatusInternalServerError)
func HTTPStatusCode(err error, defaultCode int) int {
	if err == nil {
		return defaultCode
	}
	if e, ok := err.(*Error); ok {
		if c := e.Code(); c >= http.StatusContinue && c <= 599 {
			return c
		}
	}
	return defaultCode
}
