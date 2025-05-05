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

func (h *HttpJmapApiClient) authWithUsername(logger *log.Logger, username string, req *http.Request) error {
	masterUsername := username + "%" + h.masterUser
	req.SetBasicAuth(masterUsername, h.masterPassword)
	return nil
}

func (h *HttpJmapApiClient) GetWellKnown(username string, logger *log.Logger) (WellKnownJmap, error) {
	wellKnownUrl := h.baseurl + "/.well-known/jmap"

	req, err := http.NewRequest(http.MethodGet, wellKnownUrl, nil)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to create GET request for %v", wellKnownUrl)
		return WellKnownJmap{}, err
	}
	h.authWithUsername(logger, username, req)

	res, err := h.client.Do(req)
	if err != nil {
		logger.Error().Err(err).Msgf("failed to perform GET %v", wellKnownUrl)
		return WellKnownJmap{}, err
	}
	if res.StatusCode != 200 {
		logger.Error().Str("status", res.Status).Msg("HTTP response status code is not 200")
		return WellKnownJmap{}, fmt.Errorf("HTTP response status is %v", res.Status)
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
		return WellKnownJmap{}, err
	}

	var data WellKnownJmap
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.Error().Str("url", wellKnownUrl).Err(err).Msg("failed to decode JSON payload from .well-known/jmap response")
		return WellKnownJmap{}, err
	}

	return data, nil
}

func (h *HttpJmapApiClient) Command(ctx context.Context, logger *log.Logger, request map[string]any) ([]byte, error) {
	jmapUrl := h.jmapurl

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
	req.Header.Add("User-Agent", "OpenCloud/"+version.GetString())
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
