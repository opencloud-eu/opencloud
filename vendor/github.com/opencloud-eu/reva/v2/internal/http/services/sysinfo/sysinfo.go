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

package sysinfo

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/global"
	"github.com/opencloud-eu/reva/v2/pkg/sysinfo"
)

func init() {
	global.Register(serviceName, New)
}

type config struct {
	Prefix string `mapstructure:"prefix"`
}

type svc struct {
	conf *config
}

const (
	serviceName = "sysinfo"
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
	return []string{"/"}
}

// Handler serves all HTTP requests.
func (s *svc) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := appctx.GetLogger(r.Context())
		if _, err := w.Write([]byte(s.getJSONData())); err != nil {
			log.Err(err).Msg("error writing SysInfo response")
		}
	})
}

func (s *svc) getJSONData() string {
	if data, err := sysinfo.SysInfo.ToJSON(); err == nil {
		return data
	}

	return ""
}

func parseConfig(m map[string]interface{}) (*config, error) {
	cfg := &config{}
	if err := mapstructure.Decode(m, &cfg); err != nil {
		return nil, errors.Wrap(err, "sysinfo: error decoding configuration")
	}
	applyDefaultConfig(cfg)
	return cfg, nil
}

func applyDefaultConfig(conf *config) {
	if conf.Prefix == "" {
		conf.Prefix = serviceName
	}
}

// New returns a new SysInfo service.
func New(m map[string]interface{}, log *zerolog.Logger) (global.Service, error) {
	// Prepare the configuration
	conf, err := parseConfig(m)
	if err != nil {
		return nil, err
	}

	// Create the service
	s := &svc{
		conf: conf,
	}
	return s, nil
}
