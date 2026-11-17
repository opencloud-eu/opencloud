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

package auth

import (
	"context"
	"net/http"

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	registry "github.com/cs3org/go-cs3apis/cs3/auth/registry/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/plugin"
)

// Manager is the interface to implement to authenticate users
type Manager interface {
	plugin.Plugin
	Authenticate(ctx context.Context, clientID, clientSecret string) (*user.User, map[string]*authpb.Scope, error)
}

// Credentials contains the auth type, client id and secret.
type Credentials struct {
	Type         string
	ClientID     string
	ClientSecret string
}

// CredentialStrategy obtains Credentials from the request.
type CredentialStrategy interface {
	GetCredentials(w http.ResponseWriter, r *http.Request) (*Credentials, error)
	AddWWWAuthenticate(w http.ResponseWriter, r *http.Request, realm string)
}

// TokenStrategy obtains a token from the request.
// If token does not exist returns an empty string.
type TokenStrategy interface {
	GetToken(r *http.Request) string
}

// TokenWriter stores the token in a http response.
type TokenWriter interface {
	WriteToken(token string, w http.ResponseWriter)
}

// Registry is the interface that auth registries implement
// for discovering auth providers
type Registry interface {
	ListProviders(ctx context.Context) ([]*registry.ProviderInfo, error)
	GetProvider(ctx context.Context, authType string) (*registry.ProviderInfo, error)
}
