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

package bearer

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/opencloud-eu/opencloud/internal/http/interceptors/auth/credential/registry"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
)

func init() {
	registry.Register("bearer", New)
}

type strategy struct{}

// New returns a new auth strategy that checks "Bearer" OAuth Access Tokens
// See https://tools.ietf.org/html/rfc6750#section-6.1
func New(m map[string]interface{}) (auth.CredentialStrategy, error) {
	return &strategy{}, nil
}

func (s *strategy) GetCredentials(w http.ResponseWriter, r *http.Request) (*auth.Credentials, error) {
	// 1. check Authorization header
	hdr := r.Header.Get("Authorization")
	token := strings.TrimPrefix(hdr, "Bearer ")
	if token != "" {
		return &auth.Credentials{Type: "bearer", ClientSecret: token}, nil
	}
	// TODO 2. check form encoded body parameter for POST requests, see https://tools.ietf.org/html/rfc6750#section-2.2

	// 3. check uri query parameter, see https://tools.ietf.org/html/rfc6750#section-2.3
	tokens, ok := r.URL.Query()["access_token"]
	if !ok || len(tokens[0]) < 1 {
		return nil, fmt.Errorf("no bearer auth provided")
	}
	return &auth.Credentials{Type: "bearer", ClientSecret: tokens[0]}, nil

}

func (s *strategy) AddWWWAuthenticate(w http.ResponseWriter, r *http.Request, realm string) {
	// TODO read realm from forwarded header?
	if realm == "" {
		// fall back to hostname if not configured
		realm = r.Host
	}
	w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Bearer realm="%s"`, realm))
}
