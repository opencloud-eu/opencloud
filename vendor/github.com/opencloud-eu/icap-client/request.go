package icapclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
)

// Request represents the icap client request data.
type Request struct {
	Method                string
	URL                   *url.URL
	Header                http.Header
	HTTPRequest           *http.Request
	HTTPResponse          *http.Response
	ChunkLength           int
	PreviewBytes          int
	ctx                   context.Context
	previewSet            bool
	bodyFittedInPreview   bool
	remainingPreviewBytes []byte
}

// NewRequest returns a new Request given a context, method, url, http request and http response
// todo: method iota
func NewRequest(ctx context.Context, method, urlStr string, httpReq *http.Request, httpResp *http.Response) (Request, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return Request{}, err
	}

	req := Request{
		Method:       strings.ToUpper(method),
		URL:          u,
		Header:       make(map[string][]string),
		HTTPRequest:  httpReq,
		HTTPResponse: httpResp,
		ctx:          ctx,
	}

	if err := req.validate(); err != nil {
		return Request{}, err
	}

	return req, nil
}

// SetPreview sets the preview bytes in the icap header
// todo: defer close error
func (r *Request) SetPreview(maxBytes int) (err error) {
	var bodyBytes []byte
	var previewBytes int

	// receiving the body bites to determine the preview bytes depending on the request ICAP method
	if r.Method == MethodREQMOD {
		if r.HTTPRequest == nil {
			return nil
		}

		if r.HTTPRequest.Body != nil {
			b, err := io.ReadAll(r.HTTPRequest.Body)
			if err != nil {
				return err
			}
			bodyBytes = b

			defer func() {
				err = errors.Join(err, r.HTTPRequest.Body.Close())
			}()
		}
	}

	if r.Method == MethodRESPMOD {
		if r.HTTPResponse == nil {
			return nil
		}

		if r.HTTPResponse.Body != nil {
			b, err := io.ReadAll(r.HTTPResponse.Body)
			if err != nil {
				return err
			}
			bodyBytes = b

			defer func() {
				err = errors.Join(err, r.HTTPResponse.Body.Close())
			}()
		}
	}

	previewBytes = len(bodyBytes)

	// if the preview byte is 0 or less, there is no question of the body-fitting insides
	if previewBytes > 0 {
		r.bodyFittedInPreview = true
	}

	// if the preview bytes are greater than what was mentioned by the ICAP Server (did not fit in the body)
	if previewBytes > maxBytes {
		previewBytes = maxBytes
		r.bodyFittedInPreview = false
		// storing the rest of the body byte which was not sent as preview for further operations
		r.remainingPreviewBytes = bodyBytes[maxBytes:]
	}

	// set the body to the http message depending on the request method
	if r.Method == MethodREQMOD {
		r.HTTPRequest.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	if r.Method == MethodRESPMOD {
		r.HTTPResponse.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	// assign preview byte information to the header
	r.Header.Set("Preview", strconv.Itoa(previewBytes))
	r.PreviewBytes = previewBytes
	r.previewSet = true

	return err
}

// to the ICAP server to ensure all required headers are set.
func (r *Request) setDefaultRequestHeaders() {
	if _, exists := r.Header["Allow"]; !exists {
		r.Header.Add("Allow", "204") // assigning 204 by default if Allow not provided
	}

	if _, exists := r.Header["Host"]; !exists {
		hostName, _ := os.Hostname()
		r.Header.Add("Host", hostName)
	}
}

// extendHeader extends the current ICAP Request header with a new header.
func (r *Request) extendHeader(hdr http.Header) error {
	for header, values := range hdr {
		if header == previewHeader && r.previewSet {
			continue
		}

		if header == encapsulatedHeader {
			continue
		}

		for _, value := range values {
			if header == previewHeader {
				pb, err := strconv.Atoi(value)
				if err != nil {
					return err
				}

				err = r.SetPreview(pb)
				if err != nil {
					return err
				}

				continue
			}
			r.Header.Add(header, value)
		}
	}

	return nil
}

// validate checks if the ICAP request is valid or not.
func (r *Request) validate() error {
	var err error

	// check if the ICAP request has a context
	if r.ctx == nil {
		err = errors.Join(err, ErrNoContext)
	}

	// check if the ICAP request method is allowed
	if methodAllowed := slices.Contains([]string{
		MethodOPTIONS,
		MethodRESPMOD,
		MethodREQMOD,
	}, r.Method); !methodAllowed {
		err = errors.Join(err, ErrMethodNotAllowed)
	}

	// check if the ICAP url is valid and contains all required fields
	{
		if r.URL.Scheme != schemeICAP {
			err = errors.Join(err, ErrInvalidScheme)
		}

		if r.URL.Host == "" {
			err = errors.Join(err, ErrInvalidHost)
		}
	}

	// check if the ICAP request method is aligned with the http messages
	{
		if r.Method == MethodREQMOD && r.HTTPRequest == nil {
			err = errors.Join(err, ErrREQMODWithoutReq)
		}

		if r.Method == MethodREQMOD && r.HTTPResponse != nil {
			err = errors.Join(err, ErrREQMODWithResp)
		}

		if r.Method == MethodRESPMOD && r.HTTPResponse == nil {
			err = errors.Join(err, ErrRESPMODWithoutResp)
		}
	}

	return err
}
