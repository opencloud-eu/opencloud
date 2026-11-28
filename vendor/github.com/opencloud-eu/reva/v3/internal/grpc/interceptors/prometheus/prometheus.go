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
	"context"

	"github.com/opencloud-eu/reva/v3/pkg/rgrpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
)

const (
	defaultPriority = 100
)

func init() {
	rgrpc.RegisterUnaryInterceptor("prometheus", NewUnary)
}

// NewUnary returns a new unary interceptor
// that counts grpc calls.
func NewUnary(m map[string]interface{}) (grpc.UnaryServerInterceptor, int, error) {
	interceptor, err := interceptorFromConfig(m)
	if err != nil {
		return nil, 0, err
	}
	return interceptor, defaultPriority, nil
}

// NewStream returns a new server stream interceptor
// that counts grpc calls.
func NewStream() grpc.StreamServerInterceptor {
	interceptor := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, ss)
		// TODO count res codes & errors
		return err
	}
	return interceptor
}

func interceptorFromConfig(m map[string]interface{}) (grpc.UnaryServerInterceptor, error) {
	namespace := m["namespace"].(string)
	if namespace == "" {
		namespace = "reva"
	}
	subsystem := m["subsystem"].(string)
	reqProcessed := promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "grpc_requests_total",
		Help:      "The total number of processed " + subsystem + " GRPC requests for " + namespace,
	})
	interceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		res, err := handler(ctx, req)
		reqProcessed.Inc()
		return res, err
	}
	return interceptor, nil
}
