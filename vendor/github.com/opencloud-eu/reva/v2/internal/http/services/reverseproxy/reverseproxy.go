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

package reverseproxy

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-viper/mapstructure/v2"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/global"
	"github.com/rs/zerolog"
)

func init() {
	global.Register("reverseproxy", New)
}

type proxyRule struct {
	Endpoint string `mapstructure:"endpoint" json:"endpoint"`
	Backend  string `mapstructure:"backend" json:"backend"`
}

type config struct {
	ProxyRulesJSON string `mapstructure:"proxy_rules_json"`
}

func (c *config) init() {
	if c.ProxyRulesJSON == "" {
		c.ProxyRulesJSON = "/etc/revad/proxy_rules.json"
	}
}

type svc struct {
	router *chi.Mux
}

// New returns an instance of the reverse proxy service
func New(m map[string]interface{}, log *zerolog.Logger) (global.Service, error) {
	conf := &config{}
	if err := mapstructure.Decode(m, conf); err != nil {
		return nil, err
	}
	conf.init()

	f, err := os.ReadFile(conf.ProxyRulesJSON)
	if err != nil {
		return nil, err
	}

	var rules []proxyRule
	err = json.Unmarshal(f, &rules)
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()

	for _, rule := range rules {
		remote, err := url.Parse(rule.Backend)
		if err != nil {
			// Skip the rule if the backend is not a valid URL
			continue
		}

		proxy := httputil.NewSingleHostReverseProxy(remote)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Host = remote.Host
			if token, ok := ctxpkg.ContextGetToken(r.Context()); ok {
				r.Header.Set(ctxpkg.TokenHeader, token)
			}
			proxy.ServeHTTP(w, r)
		})
		r.Mount(rule.Endpoint, handler)
	}

	_ = chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		log.Debug().Str("service", "reverseproxy").Str("method", method).Str("route", route).Int("middlewares", len(middlewares)).Msg("serving endpoint")
		return nil
	})

	return &svc{router: r}, nil
}

func (s *svc) Close() error {
	return nil
}

func (s *svc) Prefix() string {
	// This service will be served at root
	return ""
}

func (s *svc) Unprotected() []string {
	// TODO: If the services which will be served via the reverse proxy have unprotected endpoints,
	// we won't be able to support those at the moment.
	return []string{}
}

func (s *svc) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.router.ServeHTTP(w, r)
	})
}
