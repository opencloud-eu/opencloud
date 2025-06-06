package jmap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/version"
)

type HttpJmapUsernameProvider interface {
	GetUsername(ctx context.Context, logger *log.Logger) (string, error)
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

func (h *HttpJmapApiClient) auth(logger *log.Logger, ctx context.Context, req *http.Request) error {
	username, err := h.usernameProvider.GetUsername(ctx, logger)
	if err != nil {
		logger.Error().Err(err).Msg("failed to find username")
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

func (h *HttpJmapApiClient) GetWellKnown(username string, logger *log.Logger) (WellKnownResponse, error) {
	wellKnownUrl := h.baseurl + "/.well-known/jmap"

	req, err := http.NewRequest(http.MethodGet, wellKnownUrl, nil)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to create GET request for %v", wellKnownUrl)
		return WellKnownResponse{}, err
	}
	h.authWithUsername(logger, username, req)
	req.Header.Add("Cache-Control", "no-cache, no-store, must-revalidate") // spec recommendation

	res, err := h.client.Do(req)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to perform GET %v", wellKnownUrl)
		return WellKnownResponse{}, err
	}
	if res.StatusCode != 200 {
		logger.Error().Str("status", res.Status).Msg("HTTP response status code is not 200")
		return WellKnownResponse{}, fmt.Errorf("HTTP response status is %v", res.Status)
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
		return WellKnownResponse{}, err
	}

	var data WellKnownResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.Error().Str("url", wellKnownUrl).Err(err).Msg("failed to decode JSON payload from .well-known/jmap response")
		return WellKnownResponse{}, err
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
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		logger.Error().Str("status", res.Status).Msg("HTTP response status code is not 2xx")
		return nil, fmt.Errorf("HTTP response status is %v", res.Status)
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
		return nil, err
	}

	return body, nil
}
