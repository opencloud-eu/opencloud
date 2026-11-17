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

package global

import (
	"net/http"

	"github.com/rs/zerolog"
)

// NewMiddlewares contains all the registered new middleware functions.
var NewMiddlewares = map[string]NewMiddleware{}

// NewMiddleware is the function that HTTP middlewares need to register at init time.
type NewMiddleware func(conf map[string]interface{}) (Middleware, int, error)

// RegisterMiddleware registers a new HTTP middleware and its new function.
func RegisterMiddleware(name string, n NewMiddleware) {
	NewMiddlewares[name] = n
}

// Middleware is a middleware http handler.
type Middleware func(h http.Handler) http.Handler

// Services is a map of service name and its new function.
var Services = map[string]NewService{}

// Register registers a new HTTP services with name and new function.
func Register(name string, newFunc NewService) {
	Services[name] = newFunc
}

// NewService is the function that HTTP services need to register at init time.
type NewService func(conf map[string]interface{}, log *zerolog.Logger) (Service, error)

// Service represents a HTTP service.
type Service interface {
	Handler() http.Handler
	Prefix() string
	Close() error
	// List of url relative to the prefix to be unprotected by the authentication
	// middleware. To be seen if we need url-verb fine grained skip checks like
	// GET is public and POST is not.
	Unprotected() []string
}
