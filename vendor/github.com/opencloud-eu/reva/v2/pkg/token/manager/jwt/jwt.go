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

package jwt

import (
	"context"
	"time"

	auth "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/go-viper/mapstructure/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/token"
	"github.com/opencloud-eu/reva/v2/pkg/token/manager/registry"
	"github.com/pkg/errors"
)

const defaultExpiration int64 = 86400 // 1 day
const defaultLeeway int64 = 5         // 5 seconds

func init() {
	registry.Register("jwt", New)
}

type config struct {
	Secret          string `mapstructure:"secret"`
	Expires         int64  `mapstructure:"expires"`
	tokenTimeLeeway int64  `mapstructure:"token_leeway"`
}

type manager struct {
	conf *config
}

// claims are custom claims for the JWT token.
type claims struct {
	jwt.RegisteredClaims
	User  *user.User             `json:"user"`
	Scope map[string]*auth.Scope `json:"scope"`
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

// New returns an implementation of the token manager that uses JWT as tokens.
func New(value map[string]interface{}) (token.Manager, error) {
	c, err := parseConfig(value)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing config")
	}

	if c.Expires == 0 {
		c.Expires = defaultExpiration
	}

	if c.tokenTimeLeeway == 0 {
		c.tokenTimeLeeway = defaultLeeway
	}

	c.Secret = sharedconf.GetJWTSecret(c.Secret)

	if c.Secret == "" {
		return nil, errors.New("jwt: secret for signing payloads is not defined in config")
	}

	m := &manager{conf: c}
	return m, nil
}

func (m *manager) MintToken(ctx context.Context, u *user.User, scope map[string]*auth.Scope) (string, error) {
	ttl := time.Duration(m.conf.Expires) * time.Second
	newClaims := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			Issuer:    u.Id.Idp,
			Audience:  jwt.ClaimStrings{"reva"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		User:  u,
		Scope: scope,
	}

	t := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), newClaims)

	tkn, err := t.SignedString([]byte(m.conf.Secret))
	if err != nil {
		return "", errors.Wrapf(err, "error signing token with claims %+v", newClaims)
	}

	return tkn, nil
}

func (m *manager) DismantleToken(ctx context.Context, tkn string) (*user.User, map[string]*auth.Scope, error) {
	keyfunc := func(token *jwt.Token) (interface{}, error) {
		return []byte(m.conf.Secret), nil
	}
	token, err := jwt.ParseWithClaims(tkn, &claims{}, keyfunc, jwt.WithLeeway(time.Duration(m.conf.tokenTimeLeeway)*time.Second))

	if err != nil {
		return nil, nil, errors.Wrap(err, "error parsing token")
	}

	if claims, ok := token.Claims.(*claims); ok && token.Valid {
		return claims.User, claims.Scope, nil
	}

	return nil, nil, errtypes.InvalidCredentials("invalid token")
}
