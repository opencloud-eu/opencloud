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

package trace

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var (
	// Propagator is the default Reva propagator.
	Propagator      = propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})
	defaultProvider = revaDefaultTracerProvider{}
)

type revaDefaultTracerProvider struct {
	mutex       sync.RWMutex
	initialized bool
}

// NewTracerProvider returns a new TracerProvider, configured for the specified service
func NewTracerProvider(serviceName, exporter string) trace.TracerProvider {
	var tp *sdktrace.TracerProvider
	if exporter == "otlp" {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		exporter, err := otlptracegrpc.New(ctx)
		if err != nil {
			return nil
		}
		resources, err := resource.New(
			context.Background(),
			resource.WithFromEnv(), // Reads OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME
			resource.WithAttributes(
				attribute.String("service.name", serviceName),
				attribute.String("library.language", "go"),
			),
		)
		if err != nil {
			return nil
		}

		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resources),
		)
	} else {
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.NeverSample()),
		)
	}

	SetDefaultTracerProvider(tp)
	return tp
}

// SetDefaultTracerProvider sets the default trace provider
func SetDefaultTracerProvider(tp trace.TracerProvider) {
	otel.SetTracerProvider(tp)
	defaultProvider.mutex.Lock()
	defer defaultProvider.mutex.Unlock()
	defaultProvider.initialized = true
}

// DefaultProvider returns the "global" default TracerProvider
// Currently used by the pool to get the global tracer
func DefaultProvider() trace.TracerProvider {
	defaultProvider.mutex.RLock()
	defer defaultProvider.mutex.RUnlock()
	return otel.GetTracerProvider()
}
