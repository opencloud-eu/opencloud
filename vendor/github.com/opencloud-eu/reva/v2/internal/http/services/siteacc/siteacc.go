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

package siteacc

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/pkg/siteacc"
	"github.com/opencloud-eu/reva/v2/pkg/siteacc/config"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v2/pkg/rhttp/global"
)

func init() {
	global.Register(serviceName, New)
}

type svc struct {
	conf *config.Configuration
	log  *zerolog.Logger

	siteacc *siteacc.SiteAccounts
}

const (
	serviceName = "siteacc"
)

// Close is called when this service is being stopped.
func (s *svc) Close() error {
	return nil
}

// Prefix returns the main endpoint of this service.
func (s *svc) Prefix() string {
	return s.conf.Prefix
}

// Unprotected returns all endpoints that can be queried without prior authorization.
func (s *svc) Unprotected() []string {
	return s.siteacc.GetPublicEndpoints()
}

// Handler serves all HTTP requests.
func (s *svc) Handler() http.Handler {
	return s.siteacc.RequestHandler()
}

func parseConfig(m map[string]interface{}) (*config.Configuration, error) {
	conf := &config.Configuration{}
	if err := mapstructure.Decode(m, &conf); err != nil {
		return nil, errors.Wrap(err, "error decoding configuration")
	}
	applyDefaultConfig(conf)
	conf.Cleanup()

	if conf.Webserver.URL == "" {
		return nil, errors.Errorf("no webserver URL specified")
	}

	return conf, nil
}

func applyDefaultConfig(conf *config.Configuration) {
	if conf.Prefix == "" {
		conf.Prefix = serviceName
	}

	if conf.Storage.Driver == "" {
		conf.Storage.Driver = "file"
	}

	if conf.Mentix.DataEndpoint == "" {
		conf.Mentix.DataEndpoint = "/sites"
	}

	if conf.Mentix.SiteRegistrationEndpoint == "" {
		conf.Mentix.SiteRegistrationEndpoint = "/sitereg"
	}

	// Enforce a minimum session timeout of 1 minute (and default to 5 minutes)
	if conf.Webserver.SessionTimeout < 60 {
		conf.Webserver.SessionTimeout = 5 * 60
	}
}

// New returns a new Site Accounts service.
func New(m map[string]interface{}, log *zerolog.Logger) (global.Service, error) {
	// Prepare the configuration
	conf, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	// Create the site accounts instance
	siteacc, err := siteacc.New(conf, log)
	if err != nil {
		return nil, errors.Wrap(err, "error creating the site accounts service")
	}

	// Create the service
	s := &svc{
		conf:    conf,
		log:     log,
		siteacc: siteacc,
	}
	return s, nil
}
