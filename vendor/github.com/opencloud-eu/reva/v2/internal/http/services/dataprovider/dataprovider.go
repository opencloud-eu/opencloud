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

package dataprovider

import (
	"fmt"
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	datatxregistry "github.com/opencloud-eu/reva/v2/pkg/rhttp/datatx/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/global"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp/router"
	"github.com/opencloud-eu/reva/v2/pkg/storage"
	"github.com/opencloud-eu/reva/v2/pkg/storage/fs/registry"
)

func init() {
	global.Register("dataprovider", New)
}

type config struct {
	Prefix             string                            `mapstructure:"prefix" docs:"data;The prefix to be used for this HTTP service"`
	Driver             string                            `mapstructure:"driver" docs:"localhome;The storage driver to be used."`
	Drivers            map[string]map[string]interface{} `mapstructure:"drivers" docs:"url:pkg/storage/fs/localhome/localhome.go;The configuration for the storage driver"`
	DataTXs            map[string]map[string]interface{} `mapstructure:"data_txs" docs:"url:pkg/rhttp/datatx/manager/simple/simple.go;The configuration for the data tx protocols"`
	NatsAddress        string                            `mapstructure:"nats_address"`
	NatsClusterID      string                            `mapstructure:"nats_clusterID"`
	NatsTLSInsecure    bool                              `mapstructure:"nats_tls_insecure"`
	NatsRootCACertPath string                            `mapstructure:"nats_root_ca_cert_path"`
	NatsEnableTLS      bool                              `mapstructure:"nats_enable_tls"`
	NatsUsername       string                            `mapstructure:"nats_username"`
	NatsPassword       string                            `mapstructure:"nats_password"`
}

func (c *config) init() {
	if c.Prefix == "" {
		c.Prefix = "data"
	}
	if c.Driver == "" {
		c.Driver = "localhome"
	}
}

type svc struct {
	conf    *config
	handler http.Handler
	storage storage.FS
	dataTXs map[string]http.Handler
}

// New returns a new datasvc
func New(m map[string]interface{}, log *zerolog.Logger) (global.Service, error) {
	conf := &config{}
	if err := mapstructure.Decode(m, conf); err != nil {
		return nil, err
	}

	conf.init()

	var evstream events.Stream

	if conf.NatsAddress == "" || conf.NatsClusterID == "" {
		log.Warn().Msg("missing or incomplete nats configuration. Events will not be published.")
	} else {
		s, err := stream.NatsFromConfig("dataprovider", false, stream.NatsConfig{
			Endpoint:             conf.NatsAddress,
			Cluster:              conf.NatsClusterID,
			EnableTLS:            conf.NatsEnableTLS,
			TLSInsecure:          conf.NatsTLSInsecure,
			TLSRootCACertificate: conf.NatsRootCACertPath,
			AuthUsername:         conf.NatsUsername,
			AuthPassword:         conf.NatsPassword,
		})
		if err != nil {
			return nil, err
		}

		evstream = s
	}

	fs, err := getFS(conf, evstream, log)
	if err != nil {
		return nil, err
	}

	dataTXs, err := getDataTXs(conf, fs, evstream, log)
	if err != nil {
		return nil, err
	}

	s := &svc{
		storage: fs,
		conf:    conf,
		dataTXs: dataTXs,
	}

	err = s.setHandler()
	return s, err
}

func getFS(c *config, stream events.Stream, log *zerolog.Logger) (storage.FS, error) {
	if f, ok := registry.NewFuncs[c.Driver]; ok {
		return f(c.Drivers[c.Driver], stream, log)
	}
	return nil, fmt.Errorf("driver not found: %s", c.Driver)
}

func getDataTXs(c *config, fs storage.FS, publisher events.Publisher, log *zerolog.Logger) (map[string]http.Handler, error) {
	if c.DataTXs == nil {
		c.DataTXs = make(map[string]map[string]interface{})
	}
	if len(c.DataTXs) == 0 {
		c.DataTXs["simple"] = make(map[string]interface{})
		c.DataTXs["simple"]["cache_store"] = "noop"
		c.DataTXs["simple"]["cache_database"] = "reva"
		c.DataTXs["spaces"] = make(map[string]interface{})
		c.DataTXs["spaces"]["cache_store"] = "noop"
		c.DataTXs["spaces"]["cache_database"] = "reva"
		c.DataTXs["tus"] = make(map[string]interface{})
		c.DataTXs["tus"]["cache_store"] = "noop"
		c.DataTXs["tus"]["cache_database"] = "reva"
	}

	txs := make(map[string]http.Handler)
	for t := range c.DataTXs {
		if f, ok := datatxregistry.NewFuncs[t]; ok {
			if tx, err := f(c.DataTXs[t], publisher, log); err == nil {
				if handler, err := tx.Handler(fs); err == nil {
					txs[t] = handler
				}
			}
		}
	}
	return txs, nil
}

func (s *svc) Close() error {
	return nil
}

func (s *svc) Unprotected() []string {
	return []string{
		"/tus",
	}
}

func (s *svc) Prefix() string {
	return s.conf.Prefix
}

func (s *svc) Handler() http.Handler {
	return s.handler
}

func (s *svc) setHandler() error {

	s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := appctx.GetLogger(r.Context())
		log.Debug().Msgf("dataprovider routing: path=%s", r.URL.Path)

		head, tail := router.ShiftPath(r.URL.Path)

		if handler, ok := s.dataTXs[head]; ok {
			r.URL.Path = tail
			handler.ServeHTTP(w, r)
			return
		}

		// If we don't find a prefix match for any of the protocols, upload the resource
		// through the direct HTTP protocol
		if handler, ok := s.dataTXs["simple"]; ok {
			handler.ServeHTTP(w, r)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	})

	return nil
}
