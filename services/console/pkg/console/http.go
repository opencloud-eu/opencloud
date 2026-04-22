package console

import (
	"fmt"
	"io"
	"net/http"
)

type HTTPRequest struct {
	*http.Request
}

func NewHTTPRequest(method, url string, body io.Reader) (*HTTPRequest, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	return &HTTPRequest{req}, nil
}

func (r *HTTPRequest) SetBearerAuth(bearer string) {
	r.Header.Add("Authorization", "Bearer "+bearer)
}

func (r *HTTPRequest) AsDefault() *http.Request {
	return r.Request
}
