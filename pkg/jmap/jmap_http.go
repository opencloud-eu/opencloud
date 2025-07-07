package jmap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/version"
)

type HttpJmapUsernameProvider interface {
	// Provide the username for JMAP operations.
	GetUsername(req *http.Request, ctx context.Context, logger *log.Logger) (string, error)
}

type HttpJmapApiClient struct {
	baseurl          string
	jmapurl          string
	client           *http.Client
	usernameProvider HttpJmapUsernameProvider
	masterUser       string
	masterPassword   string
	userAgent        string
}

var (
	_ ApiClient       = &HttpJmapApiClient{}
	_ WellKnownClient = &HttpJmapApiClient{}
)

/*
func bearer(req *http.Request, token string) {
	req.Header.Add("Authorization", "Bearer "+base64.StdEncoding.EncodeToString([]byte(token)))
}
*/

func NewHttpJmapApiClient(baseurl string, jmapurl string, client *http.Client, usernameProvider HttpJmapUsernameProvider, masterUser string, masterPassword string) *HttpJmapApiClient {
	return &HttpJmapApiClient{
		baseurl:          baseurl,
		jmapurl:          jmapurl,
		client:           client,
		usernameProvider: usernameProvider,
		masterUser:       masterUser,
		masterPassword:   masterPassword,
		userAgent:        "OpenCloud/" + version.GetString(),
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

func (h *HttpJmapApiClient) auth(logger *log.Logger, ctx context.Context, req *http.Request) error {
	username, err := h.usernameProvider.GetUsername(req, ctx, logger)
	if err != nil {
		logger.Error().Err(err).Msg("failed to find username")
		return AuthenticationError{Err: err}
	}
	masterUsername := username + "%" + h.masterUser
	req.SetBasicAuth(masterUsername, h.masterPassword)
	return nil
}

func (h *HttpJmapApiClient) authWithUsername(_ *log.Logger, username string, req *http.Request) error {
	masterUsername := username + "%" + h.masterUser
	req.SetBasicAuth(masterUsername, h.masterPassword)
	return nil
}

type HttpError struct {
	Method   string
	Url      string
	Username string
	Op       string
	Err      error
}

func (e HttpError) Error() string {
	return fmt.Sprintf("HTTP error for method=%v url='%v' username='%v' while %v: %v", e.Method, e.Url, e.Username, e.Op, e.Err.Error())
}
func (e HttpError) Unwrap() error {
	return e.Err
}

func (h *HttpJmapApiClient) GetWellKnown(username string, logger *log.Logger) (WellKnownResponse, error) {
	wellKnownUrl := path.Join(h.baseurl, ".well-known", "jmap")

	req, err := http.NewRequest(http.MethodGet, wellKnownUrl, nil)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to create GET request for %v", wellKnownUrl)
		return WellKnownResponse{}, HttpError{Op: "creating request", Method: http.MethodGet, Url: wellKnownUrl, Username: username, Err: err}
	}
	h.authWithUsername(logger, username, req)
	req.Header.Add("Cache-Control", "no-cache, no-store, must-revalidate") // spec recommendation

	res, err := h.client.Do(req)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to perform GET %v", wellKnownUrl)
		return WellKnownResponse{}, HttpError{Op: "performing request", Method: http.MethodGet, Url: wellKnownUrl, Username: username, Err: err}
	}
	if res.StatusCode != 200 {
		logger.Error().Str("status", res.Status).Msg("HTTP response status code is not 200")
		return WellKnownResponse{}, HttpError{Op: "processing response", Method: http.MethodGet, Url: wellKnownUrl, Username: username, Err: fmt.Errorf("status is %v", res.Status)}
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
		return WellKnownResponse{}, HttpError{Op: "reading response body", Method: http.MethodGet, Url: wellKnownUrl, Username: username, Err: err}
	}

	var data WellKnownResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.Error().Str("url", wellKnownUrl).Err(err).Msg("failed to decode JSON payload from .well-known/jmap response")
		return WellKnownResponse{}, HttpError{Op: "reading decoding response JSON payload", Method: http.MethodGet, Url: wellKnownUrl, Username: username, Err: err}
	}

	return data, nil
}

func (h *HttpJmapApiClient) Command(ctx context.Context, logger *log.Logger, session *Session, request Request) ([]byte, error) {
	jmapUrl := h.jmapurl
	if jmapUrl == "" {
		jmapUrl = session.JmapUrl
	}

	bodyBytes, marshalErr := json.Marshal(request)
	if marshalErr != nil {
		logger.Error().Err(marshalErr).Msg("failed to marshall JSON payload")
		return nil, marshalErr
	}

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, jmapUrl, bytes.NewBuffer(bodyBytes))
	if reqErr != nil {
		logger.Error().Err(reqErr).Msgf("failed to create GET request for %v", jmapUrl)
		return nil, reqErr
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", h.userAgent)
	h.auth(logger, ctx, req)

	res, err := h.client.Do(req)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to perform GET %v", jmapUrl)
		return nil, HttpError{Op: "performing request", Method: http.MethodPost, Url: jmapUrl, Username: session.Username, Err: err}
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		logger.Error().Str("status", res.Status).Msg("HTTP response status code is not 2xx")
		return nil, HttpError{Op: "processing response", Method: http.MethodPost, Url: jmapUrl, Username: session.Username, Err: fmt.Errorf("status is %v", res.Status)}
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
		return nil, HttpError{Op: "reading response body", Method: http.MethodPost, Url: jmapUrl, Username: session.Username, Err: err}
	}

	return body, nil
}
