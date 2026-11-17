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

package pool

import (
	"crypto/tls"
	"os"
	"strconv"

	rtrace "github.com/opencloud-eu/reva/v3/pkg/trace"
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	_defaultMaxCallRecvMsgSize = 10240000
)

// NewConn creates a new connection to a grpc server
// with open census tracing support.
// TODO(labkode): make grpc tls configurable.
func NewConn(target string, opts ...Option) (*grpc.ClientConn, error) {

	options := ClientOptions{}
	if err := options.init(); err != nil {
		return nil, err
	}

	// then overwrite with supplied options
	for _, opt := range opts {
		opt(&options)
	}

	var cred credentials.TransportCredentials
	switch options.tlsMode {
	case TLSOff:
		cred = insecure.NewCredentials()
	case TLSInsecure:
		tlsConfig := tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		}
		cred = credentials.NewTLS(&tlsConfig)
	case TLSOn:
		if options.caCert != "" {
			var err error
			if cred, err = credentials.NewClientTLSFromFile(options.caCert, ""); err != nil {
				return nil, err
			}
		} else {
			// Use system's cert pool
			cred = credentials.NewTLS(&tls.Config{})
		}
	}

	// NOTE: We need to configure some grpc options in a central place.
	// If many services configure the (e.g.) gateway client differently, one will be pick randomly. This leads to inconsistencies when using the single binary.
	// To avoid inconsistencies and race conditions we get the configuration here.
	// Please do NOT follow the pattern of calling `os.Getenv` in the wild without consulting docu team first.
	maxRcvMsgSize := _defaultMaxCallRecvMsgSize
	if e := os.Getenv("OC_GRPC_MAX_RECEIVED_MESSAGE_SIZE"); e != "" {
		s, err := strconv.Atoi(e)
		if err != nil || s <= 0 {
			return nil, errors.Wrap(err, "grpc max message size is not a valid int")
		}
		maxRcvMsgSize = s
	}

	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(cred),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxRcvMsgSize),
		),
		grpc.WithDefaultServiceConfig(`{
			"loadBalancingPolicy":"round_robin"
		}`),
		/* we may want to retry more often than the default transparent retry, see https://grpc.io/docs/guides/retry/#retry-configuration
		grpc.WithDefaultServiceConfig(`{
			"loadBalancingPolicy":"round_robin"
			"methodConfig": [
				{
					"name": [
						{ "service": "grpc.examples.echo.Echo" }
					],
					"retryPolicy": {
						"maxAttempts": 3,
						"initialBackoff": "0.1s",
						"maxBackoff": "1s",
						"backoffMultiplier": 2,
						"retryableStatusCodes": ["UNAVAILABLE", "CANCELLED", "RESOURCE_EXHAUSTED", "DEADLINE_EXCEEDED"]
					}
				}
			]
		}`),
		*/
		grpc.WithStatsHandler(otelgrpc.NewClientHandler(
			otelgrpc.WithTracerProvider(
				options.tracerProvider,
			),
			otelgrpc.WithPropagators(
				rtrace.Propagator,
			),
		)),
	)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
