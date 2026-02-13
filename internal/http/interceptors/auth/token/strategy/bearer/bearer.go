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

package header

import (
	"mime"
	"net/http"
	"strings"

	"github.com/opencloud-eu/opencloud/internal/http/interceptors/auth/token/registry"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
)

func init() {
	registry.Register("bearer", New)
}

type b struct{}

// New returns a new auth strategy that checks for bearer auth.
func New(m map[string]interface{}) (auth.TokenStrategy, error) {
	return b{}, nil
}

func (b) GetToken(r *http.Request) string {
	// Authorization Request Header Field: https://www.rfc-editor.org/rfc/rfc6750#section-2.1
	if tkn, ok := getFromAuthorizationHeader(r); ok {
		return tkn
	}

	// Form-Encoded Body Parameter: https://www.rfc-editor.org/rfc/rfc6750#section-2.2
	if tkn, ok := getFromBody(r); ok {
		return tkn
	}

	// URI Query Parameter: https://www.rfc-editor.org/rfc/rfc6750#section-2.3
	if tkn, ok := getFromQueryParam(r); ok {
		return tkn
	}

	return ""
}

func getFromAuthorizationHeader(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	tkn := strings.TrimPrefix(auth, "Bearer ")
	return tkn, tkn != ""
}

func getFromBody(r *http.Request) (string, bool) {
	mediatype, _, err := mime.ParseMediaType(r.Header.Get("content-type"))
	if err != nil {
		return "", false
	}
	if mediatype != "application/x-www-form-urlencoded" {
		return "", false
	}
	if err = r.ParseForm(); err != nil {
		return "", false
	}
	tkn := r.Form.Get("access-token")
	return tkn, tkn != ""
}

func getFromQueryParam(r *http.Request) (string, bool) {
	tkn := r.URL.Query().Get("access_token")
	return tkn, tkn != ""
}
