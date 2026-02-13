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

package header

import (
	"net/http"

	"github.com/opencloud-eu/opencloud/internal/http/interceptors/auth/token/registry"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
)

func init() {
	registry.Register("header", New)
}

type strategy struct {
	header string
}

// New returns a new auth strategy that checks for basic auth.
func New(m map[string]interface{}) (auth.TokenStrategy, error) {
	return &strategy{header: ctxpkg.TokenHeader}, nil
}

func (s *strategy) GetToken(r *http.Request) string {
	return r.Header.Get(s.header)
}
