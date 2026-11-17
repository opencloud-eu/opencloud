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

package ocmd

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/rhttp/global"
	"github.com/opencloud-eu/reva/v3/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v3/pkg/utils/cfg"
	"github.com/rs/zerolog"
)

func init() {
	global.Register("ocmd", New)
}

type config struct {
	Prefix                     string `mapstructure:"prefix"`
	GatewaySvc                 string `mapstructure:"gatewaysvc"                    validate:"required"`
	ExposeRecipientDisplayName bool   `mapstructure:"expose_recipient_display_name"`
}

func (c *config) ApplyDefaults() {
	c.GatewaySvc = sharedconf.GetGatewaySVC(c.GatewaySvc)
	if c.Prefix == "" {
		c.Prefix = "ocm"
	}
}

type svc struct {
	Conf   *config
	router chi.Router
}

// New returns a new ocmd object, that implements
// the OCM APIs specified in https://cs3org.github.io/OCM-API/docs.html
func New(m map[string]interface{}, log *zerolog.Logger) (global.Service, error) {
	var c config
	if err := cfg.Decode(m, &c); err != nil {
		return nil, err
	}

	r := chi.NewRouter()
	s := &svc{
		Conf:   &c,
		router: r,
	}

	if err := s.routerInit(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *svc) routerInit() error {
	sharesHandler := new(sharesHandler)
	invitesHandler := new(invitesHandler)
	notifHandler := new(notifHandler)

	if err := sharesHandler.init(s.Conf); err != nil {
		return err
	}
	if err := invitesHandler.init(s.Conf); err != nil {
		return err
	}
	if err := notifHandler.init(s.Conf); err != nil {
		return err
	}

	s.router.Post("/shares", sharesHandler.CreateShare)
	s.router.Post("/invite-accepted", invitesHandler.AcceptInvite)
	s.router.Post("/notifications", notifHandler.Notifications)
	return nil
}

// Close performs cleanup.
func (s *svc) Close() error {
	return nil
}

func (s *svc) Prefix() string {
	return s.Conf.Prefix
}

func (s *svc) Unprotected() []string {
	return []string{"/invite-accepted", "/shares", "/notifications"}
}

func (s *svc) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := appctx.GetLogger(r.Context())
		log.Debug().Str("path", r.URL.Path).Msg("ocm routing")

		// unset raw path, otherwise chi uses it to route and then fails to match percent encoded path segments
		r.URL.RawPath = ""
		s.router.ServeHTTP(w, r)
	})
}
