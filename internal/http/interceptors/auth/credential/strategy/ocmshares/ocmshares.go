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

package ocmshares

import (
	"fmt"
	"net/http"

	"github.com/opencloud-eu/opencloud/internal/http/interceptors/auth/credential/registry"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
)

func init() {
	registry.Register("ocmshares", New)
}

const (
	headerShareToken = "ocm-token"
)

type strategy struct{}

// New returns a new auth strategy that handles public share verification.
func New(m map[string]interface{}) (auth.CredentialStrategy, error) {
	return &strategy{}, nil
}

func (s *strategy) GetCredentials(w http.ResponseWriter, r *http.Request) (*auth.Credentials, error) {
	token := r.Header.Get(headerShareToken)
	if token == "" {
		token = r.URL.Query().Get(headerShareToken)
	}
	if token == "" {
		return nil, fmt.Errorf("no ocm token provided")
	}

	return &auth.Credentials{Type: "ocmshares", ClientID: token}, nil
}

func (s *strategy) AddWWWAuthenticate(w http.ResponseWriter, r *http.Request, realm string) {
	// TODO read realm from forwarded header?
}
