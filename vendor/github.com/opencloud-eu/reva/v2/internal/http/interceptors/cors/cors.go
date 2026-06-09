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

package cors

import (
	"github.com/go-viper/mapstructure/v2"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/global"
	"github.com/rs/cors"
)

const (
	defaultPriority = 200
)

func init() {
	global.RegisterMiddleware("cors", New)
}

type config struct {
	AllowCredentials   bool     `mapstructure:"allow_credentials"`
	OptionsPassthrough bool     `mapstructure:"options_passthrough"`
	Debug              bool     `mapstructure:"debug"`
	MaxAge             int      `mapstructure:"max_age"`
	Priority           int      `mapstructure:"priority"`
	AllowedMethods     []string `mapstructure:"allowed_methods"`
	AllowedHeaders     []string `mapstructure:"allowed_headers"`
	ExposedHeaders     []string `mapstructure:"exposed_headers"`
	AllowedOrigins     []string `mapstructure:"allowed_origins"`
}

// New creates a new CORS middleware.
func New(m map[string]interface{}) (global.Middleware, int, error) {
	conf := &config{}
	if err := mapstructure.Decode(m, conf); err != nil {
		return nil, 0, err
	}

	if conf.Priority == 0 {
		conf.Priority = defaultPriority
	}

	// apply some defaults to reduce configuration boilerplate
	if len(conf.AllowedOrigins) == 0 {
		conf.AllowedOrigins = []string{"*"}
	}

	if len(conf.AllowedMethods) == 0 {
		conf.AllowedMethods = []string{
			"OPTIONS",
			"HEAD",
			"GET",
			"PUT",
			"POST",
			"DELETE",
			"MKCOL",
			"PROPFIND",
			"PROPPATCH",
			"MOVE",
			"COPY",
			"REPORT",
			"SEARCH",
		}
	}

	if len(conf.AllowedHeaders) == 0 {
		conf.AllowedHeaders = []string{
			"Origin",
			"Accept",
			"Content-Type",
			"Depth",
			"Authorization",
			"Ocs-Apirequest",
			"If-None-Match",
			"If-Match",
			"Destination",
			"Overwrite",
			"X-Request-Id",
			"X-Requested-With",
			"Tus-Resumable",
			"Tus-Checksum-Algorithm",
			"Upload-Concat",
			"Upload-Length",
			"Upload-Metadata",
			"Upload-Defer-Length",
			"Upload-Expires",
			"Upload-Checksum",
			"Upload-Offset",
			"X-HTTP-Method-Override",
		}
	}

	if len(conf.ExposedHeaders) == 0 {
		conf.ExposedHeaders = []string{
			"Location",
		}
	}

	// TODO(jfd): use log from request context, otherwise fmt will be used to log,
	// preventing us from piping the log to eg jq
	c := cors.New(cors.Options{
		AllowCredentials:   conf.AllowCredentials,
		AllowedHeaders:     conf.AllowedHeaders,
		AllowedMethods:     conf.AllowedMethods,
		AllowedOrigins:     conf.AllowedOrigins,
		ExposedHeaders:     conf.ExposedHeaders,
		MaxAge:             conf.MaxAge,
		OptionsPassthrough: conf.OptionsPassthrough,
		Debug:              conf.Debug,
	})

	return c.Handler, conf.Priority, nil
}
