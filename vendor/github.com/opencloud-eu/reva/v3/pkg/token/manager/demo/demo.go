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

package demo

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"

	auth "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/token"
	"github.com/opencloud-eu/reva/v3/pkg/token/manager/registry"
	"github.com/pkg/errors"
)

func init() {
	registry.Register("demo", New)
}

// New returns a new token manager.
func New(m map[string]interface{}) (token.Manager, error) {
	mngr := manager{}
	return &mngr, nil
}

type manager struct{}

type claims struct {
	User  *user.User             `json:"user"`
	Scope map[string]*auth.Scope `json:"scope"`
}

func (m *manager) MintToken(ctx context.Context, u *user.User, scope map[string]*auth.Scope) (string, error) {
	token, err := encode(&claims{u, scope})
	if err != nil {
		return "", errors.Wrap(err, "error encoding user")
	}
	return token, nil
}

func (m *manager) DismantleToken(ctx context.Context, token string) (*user.User, map[string]*auth.Scope, error) {
	c, err := decode(token)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error decoding claims")
	}
	return c.User, c.Scope, nil
}

// from https://stackoverflow.com/questions/28020070/golang-serialize-and-deserialize-back
// go binary encoder
func encode(c *claims) (string, error) {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(c)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

// from https://stackoverflow.com/questions/28020070/golang-serialize-and-deserialize-back
// go binary decoder
func decode(token string) (*claims, error) {
	c := &claims{}
	by, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
