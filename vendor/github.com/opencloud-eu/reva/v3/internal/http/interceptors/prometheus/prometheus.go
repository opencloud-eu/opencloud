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

package prometheus

import (
	"net/http"

	"github.com/opencloud-eu/reva/v3/pkg/rhttp/global"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	defaultPriority = 100
)

func init() {
	global.RegisterMiddleware("prometheus", New)
}

// New returns a new HTTP middleware that counts requests for prometheus metrics
func New(m map[string]interface{}) (global.Middleware, int, error) {
	namespace := m["namespace"].(string)
	if namespace == "" {
		namespace = "reva"
	}
	subsystem := m["subsystem"].(string)
	ph := prometheusHandler{
		counter: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: m["subsystem"].(string),
			Name:      "http_requests_total",
			Help:      "The total number of processed " + subsystem + " HTTP requests for " + namespace,
		}),
	}
	return ph.handler, defaultPriority, nil
}

type prometheusHandler struct {
	h       http.Handler
	counter prometheus.Counter
}

// handler is a logging middleware
func (ph prometheusHandler) handler(h http.Handler) http.Handler {
	ph.h = h
	return ph
}

func (ph prometheusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ph.h.ServeHTTP(w, r)
	ph.counter.Inc()
}
