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

package requestid

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/opencloud-eu/reva/v3/pkg/rhttp/global"
)

const (
	defaultPriority = 100
)

func init() {
	global.RegisterMiddleware("requestid", New)
}

// New returns a new HTTP middleware that adds the X-Request-ID to the context
func New(m map[string]interface{}) (global.Middleware, int, error) {
	rh := requestIDHandler{}
	return rh.handler, defaultPriority, nil
}

type requestIDHandler struct {
	h http.Handler
}

// handler is a request id middleware
func (rh requestIDHandler) handler(h http.Handler) http.Handler {
	rh.h = middleware.RequestID(h)
	return rh
}

func (rh requestIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rh.h.ServeHTTP(w, r)
}
