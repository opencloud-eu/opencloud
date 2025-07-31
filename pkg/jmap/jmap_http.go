package jmap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/version"
)

type HttpJmapApiClient struct {
	baseurl        url.URL
	client         *http.Client
	masterUser     string
	masterPassword string
	userAgent      string
}

var (
	_ ApiClient     = &HttpJmapApiClient{}
	_ SessionClient = &HttpJmapApiClient{}
)

/*
func bearer(req *http.Request, token string) {
	req.Header.Add("Authorization", "Bearer "+base64.StdEncoding.EncodeToString([]byte(token)))
}
*/

func NewHttpJmapApiClient(baseurl url.URL, client *http.Client, masterUser string, masterPassword string) *HttpJmapApiClient {
	return &HttpJmapApiClient{
		baseurl:        baseurl,
		client:         client,
		masterUser:     masterUser,
		masterPassword: masterPassword,
		userAgent:      "OpenCloud/" + version.GetString(),
	}
}

func (h *HttpJmapApiClient) Close() error {
	h.client.CloseIdleConnections()
	return nil
}

type AuthenticationError struct {
	Err error
}

func (e AuthenticationError) Error() string {
	return fmt.Sprintf("failed to find user for authentication: %v", e.Err.Error())
}
func (e AuthenticationError) Unwrap() error {
	return e.Err
}

func (h *HttpJmapApiClient) auth(username string, _ *log.Logger, req *http.Request) error {
	masterUsername := username + "%" + h.masterUser
	req.SetBasicAuth(masterUsername, h.masterPassword)
	return nil
}

func (h *HttpJmapApiClient) GetSession(username string, logger *log.Logger) (SessionResponse, Error) {
	wellKnownUrl := h.baseurl.JoinPath(".well-known", "jmap").String()

	req, err := http.NewRequest(http.MethodGet, wellKnownUrl, nil)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to create GET request for %v", wellKnownUrl)
		return SessionResponse{}, SimpleError{code: JmapErrorInvalidHttpRequest, err: err}
	}
	h.auth(username, logger, req)
	req.Header.Add("Cache-Control", "no-cache, no-store, must-revalidate") // spec recommendation

	res, err := h.client.Do(req)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to perform GET %v", wellKnownUrl)
		return SessionResponse{}, SimpleError{code: JmapErrorInvalidHttpRequest, err: err}
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		logger.Error().Str("status", res.Status).Msg("HTTP response status code is not 200")
		return SessionResponse{}, SimpleError{code: JmapErrorServerResponse, err: fmt.Errorf("JMAP API response status is %v", res.Status)}
	}
	if res.Body != nil {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				logger.Error().Err(err).Msg("failed to close response body")
			}
		}(res.Body)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error().Err(err).Msg("failed to read response body")
		return SessionResponse{}, SimpleError{code: JmapErrorReadingResponseBody, err: err}
	}

	var data SessionResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.Error().Str("url", wellKnownUrl).Err(err).Msg("failed to decode JSON payload from .well-known/jmap response")
		return SessionResponse{}, SimpleError{code: JmapErrorDecodingResponseBody, err: err}
	}

	return data, nil
}

func (h *HttpJmapApiClient) Command(ctx context.Context, logger *log.Logger, session *Session, request Request) ([]byte, Error) {
	jmapUrl := session.JmapUrl.String()

	bodyBytes, err := json.Marshal(request)
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshall JSON payload")
		return nil, SimpleError{code: JmapErrorEncodingRequestBody, err: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, jmapUrl, bytes.NewBuffer(bodyBytes))
	if err != nil {
		logger.Error().Err(err).Msgf("failed to create POST request for %v", jmapUrl)
		return nil, SimpleError{code: JmapErrorCreatingRequest, err: err}
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", h.userAgent)
	h.auth(session.Username, logger, req)

	res, err := h.client.Do(req)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to perform POST %v", jmapUrl)
		return nil, SimpleError{code: JmapErrorSendingRequest, err: err}
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		logger.Error().Str("status", res.Status).Msg("HTTP response status code is not 2xx")
		return nil, SimpleError{code: JmapErrorServerResponse, err: err}
	}
	if res.Body != nil {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				logger.Error().Err(err).Msg("failed to close response body")
			}
		}(res.Body)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error().Err(err).Msg("failed to read response body")
		return nil, SimpleError{code: JmapErrorServerResponse, err: err}
	}

	return body, nil
}
