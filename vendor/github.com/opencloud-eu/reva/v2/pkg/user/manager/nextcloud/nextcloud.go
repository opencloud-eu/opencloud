// Copyright 2018-2021 CERN
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

package nextcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"

	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/user"
	"github.com/opencloud-eu/reva/v2/pkg/user/manager/registry"
	"github.com/pkg/errors"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	// "github.com/opencloud-eu/reva/v2/pkg/errtypes"
)

func init() {
	registry.Register("nextcloud", New)
}

// Manager is the Nextcloud-based implementation of the share.Manager interface
// see https://github.com/cs3org/reva/blob/v1.13.0/pkg/user/user.go#L29-L35
type Manager struct {
	client       *http.Client
	sharedSecret string
	endPoint     string
}

// UserManagerConfig contains config for a Nextcloud-based UserManager
type UserManagerConfig struct {
	EndPoint     string `mapstructure:"endpoint" docs:";The Nextcloud backend endpoint for user management"`
	SharedSecret string `mapstructure:"shared_secret"`
	MockHTTP     bool   `mapstructure:"mock_http"`
}

func (c *UserManagerConfig) init() {
	if c.EndPoint == "" {
		c.EndPoint = "http://localhost/end/point?"
	}
}

func parseConfig(m map[string]interface{}) (*UserManagerConfig, error) {
	c := &UserManagerConfig{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	c.init()
	return c, nil
}

// Action describes a REST request to forward to the Nextcloud backend
type Action struct {
	verb string
	argS string
}

// New returns a user manager implementation that reads a json file to provide user metadata.
func New(m map[string]interface{}) (user.Manager, error) {
	c, err := parseConfig(m)
	if err != nil {
		return nil, err
	}
	c.init()

	return NewUserManager(c)
}

// NewUserManager returns a new Nextcloud-based UserManager
func NewUserManager(c *UserManagerConfig) (*Manager, error) {
	var client *http.Client
	if c.MockHTTP {
		// Wait for SetHTTPClient to be called later
		client = nil
	} else {
		if len(c.EndPoint) == 0 {
			return nil, errors.New("Please specify 'endpoint' in '[grpc.services.userprovider.drivers.nextcloud]'")
		}
		client = &http.Client{}
	}

	return &Manager{
		endPoint:     c.EndPoint, // e.g. "http://nc/apps/sciencemesh/"
		sharedSecret: c.SharedSecret,
		client:       client,
	}, nil
}

// SetHTTPClient sets the HTTP client
func (um *Manager) SetHTTPClient(c *http.Client) {
	um.client = c
}

func getUser(ctx context.Context) (*userpb.User, error) {
	u, ok := ctxpkg.ContextGetUser(ctx)
	if !ok {
		err := errors.Wrap(errtypes.UserRequired(""), "nextcloud storage driver: error getting user from ctx")
		return nil, err
	}
	return u, nil
}

func (um *Manager) do(ctx context.Context, a Action, username string) (int, []byte, error) {
	url := um.endPoint + "~" + username + "/api/user/" + a.verb
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(a.argS))
	if err != nil {
		panic(err)
	}
	req.Header.Set("X-Reva-Secret", um.sharedSecret)

	req.Header.Set("Content-Type", "application/json")
	fmt.Println(url)
	resp, err := um.client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp.StatusCode, body, err
}

// Configure method as defined in https://github.com/cs3org/reva/blob/v1.13.0/pkg/user/user.go#L29-L35
func (um *Manager) Configure(ml map[string]interface{}) error {
	return nil
}

// GetUser method as defined in https://github.com/cs3org/reva/blob/v1.13.0/pkg/user/user.go#L29-L35
func (um *Manager) GetUser(ctx context.Context, uid *userpb.UserId, skipFetchingGroups bool) (*userpb.User, error) {
	bodyStr, _ := json.Marshal(uid)
	_, respBody, err := um.do(ctx, Action{"GetUser", string(bodyStr)}, "unauthenticated")
	if err != nil {
		return nil, err
	}
	result := &userpb.User{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}
	return result, err
}

// GetUserByClaim method as defined in https://github.com/cs3org/reva/blob/v1.13.0/pkg/user/user.go#L29-L35
func (um *Manager) GetUserByClaim(ctx context.Context, claim, value string, skipFetchingGroups bool) (*userpb.User, error) {
	type paramsObj struct {
		Claim string `json:"claim"`
		Value string `json:"value"`
	}
	bodyObj := &paramsObj{
		Claim: claim,
		Value: value,
	}
	user, err := getUser(ctx)
	if err != nil {
		return nil, err
	}

	bodyStr, _ := json.Marshal(bodyObj)
	_, respBody, err := um.do(ctx, Action{"GetUserByClaim", string(bodyStr)}, user.Username)
	if err != nil {
		return nil, err
	}
	result := &userpb.User{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return nil, err
	}
	return result, err
}

// GetUserGroups method as defined in https://github.com/cs3org/reva/blob/v1.13.0/pkg/user/user.go#L29-L35
func (um *Manager) GetUserGroups(ctx context.Context, uid *userpb.UserId) ([]string, error) {
	bodyStr, err := json.Marshal(uid)
	if err != nil {
		return nil, err
	}
	user, err := getUser(ctx)
	if err != nil {
		return nil, err
	}

	_, respBody, err := um.do(ctx, Action{"GetUserGroups", string(bodyStr)}, user.Username)
	if err != nil {
		return nil, err
	}
	var gs []string
	err = json.Unmarshal(respBody, &gs)
	if err != nil {
		return nil, err
	}
	return gs, err
}

// FindUsers method as defined in https://github.com/cs3org/reva/blob/v1.13.0/pkg/user/user.go#L29-L35
func (um *Manager) FindUsers(ctx context.Context, query, tenantID string, skipFetchingGroups bool) ([]*userpb.User, error) {
	if tenantID != "" {
		return nil, errtypes.NotSupported("tenant filter not supported in nextcloud user manager")
	}

	user, err := getUser(ctx)
	if err != nil {
		return nil, err
	}

	_, respBody, err := um.do(ctx, Action{"FindUsers", query}, user.Username)
	if err != nil {
		return nil, err
	}
	var respArr []userpb.User
	err = json.Unmarshal(respBody, &respArr)
	if err != nil {
		return nil, err
	}
	var pointers = make([]*userpb.User, len(respArr))
	for i := 0; i < len(respArr); i++ {
		pointers[i] = &respArr[i]
	}
	return pointers, err
}
