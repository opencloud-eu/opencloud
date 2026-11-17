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

package appctx

import (
	"context"

	"github.com/go-chi/chi/v5/middleware"
	rtrace "github.com/opencloud-eu/reva/v3/pkg/trace"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// WithLogger returns a context with an associated logger.
func WithLogger(ctx context.Context, l *zerolog.Logger) context.Context {
	return l.WithContext(ctx)
}

// GetLogger returns the logger associated with the given context
// or a disabled logger in case no logger is stored inside the context.
func GetLogger(ctx context.Context) *zerolog.Logger {
	logger := zerolog.Ctx(ctx)
	reqID := middleware.GetReqID(ctx)

	if reqID != "" {
		sublogger := logger.With().Str("request-id", reqID).Logger()
		logger = &sublogger
	}

	return logger
}

// WithTracerProvider returns a context with an associated TracerProvider
func WithTracerProvider(ctx context.Context, p trace.TracerProvider) context.Context {
	return rtrace.ContextSetTracerProvider(ctx, p)
}

// GetTracerProvider returns the TracerProvider associated with
// the given context. (Or the global default TracerProvider if there
// is no TracerProvider in the context)
func GetTracerProvider(ctx context.Context) trace.TracerProvider {
	return rtrace.ContextGetTracerProvider(ctx)
}
