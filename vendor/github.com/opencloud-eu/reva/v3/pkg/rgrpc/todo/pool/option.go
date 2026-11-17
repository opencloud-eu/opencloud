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
	"github.com/opencloud-eu/reva/v3/pkg/sharedconf"
	rtrace "github.com/opencloud-eu/reva/v3/pkg/trace"
	"go-micro.dev/v4/registry"
	"go.opentelemetry.io/otel/trace"
)

// Option is used to pass client options
type Option func(opts *ClientOptions)

// ClientOptions represent additional options (e.g. tls settings) for the grpc clients
type ClientOptions struct {
	tlsMode        TLSMode
	caCert         string
	tracerProvider trace.TracerProvider
	registry       registry.Registry
}

func (o *ClientOptions) init() error {
	// default to shared settings
	sharedOpt := sharedconf.GRPCClientOptions()
	var err error

	if o.tlsMode, err = StringToTLSMode(sharedOpt.TLSMode); err != nil {
		return err
	}
	o.caCert = sharedOpt.CACertFile
	o.tracerProvider = rtrace.DefaultProvider()
	return nil
}

// WithTLSMode allows to set the TLSMode option for grpc clients
func WithTLSMode(v TLSMode) Option {
	return func(o *ClientOptions) {
		o.tlsMode = v
	}
}

// WithTLSCACert allows to set the CA Certificate for grpc clients
func WithTLSCACert(v string) Option {
	return func(o *ClientOptions) {
		o.caCert = v
	}
}

// WithTracerProvider allows to set the opentelemetry tracer provider for grpc clients
func WithTracerProvider(v trace.TracerProvider) Option {
	return func(o *ClientOptions) {
		o.tracerProvider = v
	}
}

// WithRegistry allows to set the registry for service lookup
func WithRegistry(v registry.Registry) Option {
	return func(o *ClientOptions) {
		o.registry = v
	}
}
