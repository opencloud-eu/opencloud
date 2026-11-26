// Copyright 2018-2024 CERN
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

package wellknown

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/rhttp/global"
	"github.com/opencloud-eu/reva/v3/pkg/utils/cfg"
	"github.com/rs/zerolog"
)

func init() {
	global.Register("wellknown", New)
}

type svc struct {
	router chi.Router
	Conf   *config
}

type config struct {
	OCMProvider OcmProviderConfig `mapstructure:"ocmprovider"`
}

// New returns a new wellknown object.
func New(m map[string]interface{}, log *zerolog.Logger) (global.Service, error) {
	var c config
	if err := cfg.Decode(m, &c); err != nil {
		return nil, err
	}

	r := chi.NewRouter()
	s := &svc{
		router: r,
		Conf:   &c,
	}
	if err := s.routerInit(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *svc) routerInit() error {
	wkocmHandler := new(wkocmHandler)
	wkocmHandler.init(&s.Conf.OCMProvider)
	s.router.Get("/.well-known/ocm", wkocmHandler.Ocm)
	s.router.Get("/ocm-provider", wkocmHandler.Ocm)
	return nil
}

func (s *svc) Close() error {
	return nil
}

func (s *svc) Prefix() string {
	return ""
}

func (s *svc) Unprotected() []string {
	return []string{"/", "/.well-known/ocm", "/ocm-provider"}
}

func (s *svc) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := appctx.GetLogger(r.Context())
		log.Debug().Str("path", r.URL.Path).Msg(".well-known routing")

		// unset raw path, otherwise chi uses it to route and then fails to match percent encoded path segments
		r.URL.RawPath = ""
		s.router.ServeHTTP(w, r)
	})
}
