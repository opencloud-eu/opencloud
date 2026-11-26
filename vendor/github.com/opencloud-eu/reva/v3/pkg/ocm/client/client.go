// Copyright 2018-2023 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/rhttp"
	"github.com/pkg/errors"
)

// ErrTokenInvalid is the error returned by the invite-accepted
// endpoint when the token is not valid.
var ErrTokenInvalid = errors.New("the invitation token is invalid")

// ErrServiceNotTrusted is the error returned by the invite-accepted
// endpoint when the service is not trusted to accept invitations.
var ErrServiceNotTrusted = errors.New("service is not trusted to accept invitations")

// ErrUserAlreadyAccepted is the error returned by the invite-accepted
// endpoint when a user is already know by the remote cloud.
var ErrUserAlreadyAccepted = errors.New("user already accepted an invitation token")

// ErrTokenNotFound is the error returned by the invite-accepted
// endpoint when the request is done using a not existing token.
var ErrTokenNotFound = errors.New("token not found")

// ErrInvalidParameters is the error returned by the shares endpoint
// when the request does not contain required properties.
var ErrInvalidParameters = errors.New("invalid parameters")

// OCMClient is the client for an OCM provider.
type OCMClient struct {
	client *http.Client
}

// Config is the configuration to be used for the OCMClient.
type Config struct {
	Timeout  time.Duration
	Insecure bool
}

// New returns a new OCMClient.
func New(c *Config) *OCMClient {
	return &OCMClient{
		client: rhttp.GetHTTPClient(
			rhttp.Timeout(c.Timeout),
			rhttp.Insecure(c.Insecure),
		),
	}
}

// InviteAccepted informs the sender that the invitation was accepted to start sharing
// https://cs3org.github.io/OCM-API/docs.html?branch=develop&repo=OCM-API&user=cs3org#/paths/~1invite-accepted/post
func (c *OCMClient) InviteAccepted(ctx context.Context, endpoint string, r *InviteAcceptedRequest) (*User, error) {
	url, err := url.JoinPath(endpoint, "invite-accepted")
	if err != nil {
		return nil, err
	}

	body, err := r.toJSON()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "error doing request")
	}
	defer resp.Body.Close()

	return c.parseInviteAcceptedResponse(resp)
}

func (c *OCMClient) parseInviteAcceptedResponse(r *http.Response) (*User, error) {
	switch r.StatusCode {
	case http.StatusOK:
		var u User
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			return nil, errors.Wrap(err, "error decoding response body")
		}
		return &u, nil
	case http.StatusBadRequest:
		return nil, ErrTokenInvalid
	case http.StatusNotFound:
		return nil, ErrTokenNotFound
	case http.StatusConflict:
		return nil, ErrUserAlreadyAccepted
	case http.StatusForbidden:
		return nil, ErrServiceNotTrusted
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding response body")
	}
	return nil, errtypes.InternalError(string(body))
}

// NewShare creates a new share.
// https://github.com/cs3org/OCM-API/blob/develop/spec.yaml
func (c *OCMClient) NewShare(ctx context.Context, endpoint string, r *NewShareRequest) (*NewShareResponse, error) {
	url, err := url.JoinPath(endpoint, "shares")
	if err != nil {
		return nil, err
	}

	body, err := r.toJSON()
	if err != nil {
		return nil, err
	}

	log := appctx.GetLogger(ctx)
	log.Debug().Msgf("Sending OCM /shares POST to %s: %s", url, body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "error doing request")
	}
	defer resp.Body.Close()

	return c.parseNewShareResponse(resp)
}

func (c *OCMClient) parseNewShareResponse(r *http.Response) (*NewShareResponse, error) {
	switch r.StatusCode {
	case http.StatusOK, http.StatusCreated:
		var res NewShareResponse
		err := json.NewDecoder(r.Body).Decode(&res)
		return &res, err
	case http.StatusBadRequest:
		return nil, ErrInvalidParameters
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrServiceNotTrusted
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding response body")
	}
	return nil, errtypes.InternalError(string(body))
}

// Discovery returns a number of properties used to discover the capabilities offered by a remote cloud storage.
// https://cs3org.github.io/OCM-API/docs.html?branch=develop&repo=OCM-API&user=cs3org#/paths/~1ocm-provider/get
func (c *OCMClient) Discovery(ctx context.Context, endpoint string) (*Capabilities, error) {
	url, err := url.JoinPath(endpoint, "shares")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "error doing request")
	}
	defer resp.Body.Close()

	var cap Capabilities
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return nil, err
	}

	return &cap, nil
}

// NotifyRemote sends a notification to a remote OCM instance.
// Send a notification to a remote party about a previously known entity
// Notifications are optional messages. They are expected to be used to inform the other party about a change about a previously known entity,
// such as a share or a trusted user. For example, a notification MAY be sent by a recipient to let the provider know that
// the recipient declined a share. In this case, the provider site MAY mark the share as declined for its user(s). Similarly,
// it MAY be sent by a provider to let the recipient know that the provider removed a given share, such that the recipient MAY clean it up from its database.
// A notification MAY also be sent to let a recipient know that the provider removed that recipient from the list of trusted users, along with any related share.
// The recipient MAY reciprocally remove that provider from the list of trusted users, along with any related share.
// https://cs3org.github.io/OCM-API/docs.html?branch=develop&repo=OCM-API&user=cs3org#/paths/~1notifications/post
func (c *OCMClient) NotifyRemote(ctx context.Context, endpoint string, r *NotificationRequest) error {
	url, err := url.JoinPath(endpoint, "notifications")
	if err != nil {
		return err
	}
	body, err := r.ToJSON()
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return errors.Wrap(err, "error creating request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "error doing request")
	}
	defer resp.Body.Close()

	err = c.parseNotifyRemoteResponse(resp, nil)
	if err != nil {
		appctx.GetLogger(ctx).Err(err).Msg("error notifying remote OCM instance")
		return err
	}
	return nil
}

func (c *OCMClient) parseNotifyRemoteResponse(r *http.Response, resp any) error {
	var err error
	switch r.StatusCode {
	case http.StatusOK, http.StatusCreated:
		if resp == nil {
			return nil
		}
		err := json.NewDecoder(r.Body).Decode(resp)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("http status code: %v, error decoding response body", r.StatusCode))
		}
		return nil
	case http.StatusBadRequest:
		err = ErrInvalidParameters
	case http.StatusUnauthorized, http.StatusForbidden:
		err = ErrServiceNotTrusted
	default:
		err = errtypes.InternalError("request finished whit code " + strconv.Itoa(r.StatusCode))
	}

	body, err2 := io.ReadAll(r.Body)
	if err2 != nil {
		return errors.Wrap(err, "error reading response body "+err2.Error())
	}
	return errors.Wrap(err, string(body))
}
