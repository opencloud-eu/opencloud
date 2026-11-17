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

package log

import (
	"context"
	"time"

	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// NewUnary returns a new unary interceptor
// that logs grpc calls.
func NewUnary() grpc.UnaryServerInterceptor {
	interceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		res, err := handler(ctx, req)
		code := status.Code(err)
		end := time.Now()
		diff := end.Sub(start).Nanoseconds()
		var fromAddress string
		if p, ok := peer.FromContext(ctx); ok {
			fromAddress = p.Addr.Network() + "://" + p.Addr.String()
		}
		userAgent, ok := ctxpkg.ContextGetUserAgentString(ctx)
		if !ok {
			userAgent = ""
		}

		log := appctx.GetLogger(ctx)
		var event *zerolog.Event
		var msg string
		if code != codes.OK {
			event = log.Error()
			msg = err.Error()
		} else {
			event = log.Debug()
			msg = "unary"
		}

		event.Str("user-agent", userAgent).
			Str("from", fromAddress).
			Str("uri", info.FullMethod).
			Str("start", start.Format("02/Jan/2006:15:04:05 -0700")).
			Str("end", end.Format("02/Jan/2006:15:04:05 -0700")).Int("time_ns", int(diff)).
			Str("code", code.String()).
			Msg(msg)

		return res, err
	}
	return interceptor
}

// NewStream returns a new server stream interceptor
// that adds trace information to the request.
func NewStream() grpc.StreamServerInterceptor {
	interceptor := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		start := time.Now()
		err := handler(srv, ss)
		end := time.Now()
		code := status.Code(err)
		diff := end.Sub(start).Nanoseconds()
		var fromAddress string
		if p, ok := peer.FromContext(ss.Context()); ok {
			fromAddress = p.Addr.Network() + "://" + p.Addr.String()
		}
		userAgent, ok := ctxpkg.ContextGetUserAgentString(ctx)
		if !ok {
			userAgent = ""
		}

		log := appctx.GetLogger(ss.Context())
		var event *zerolog.Event
		var msg string
		if code != codes.OK {
			event = log.Error()
			msg = err.Error()
		} else {
			event = log.Debug()
			msg = "stream"
		}

		event.Str("user-agent", userAgent).
			Str("from", fromAddress).
			Str("uri", info.FullMethod).
			Str("start", start.Format("02/Jan/2006:15:04:05 -0700")).
			Str("end", end.Format("02/Jan/2006:15:04:05 -0700")).Int("time_ns", int(diff)).
			Str("code", code.String()).
			Msg(msg)

		return err
	}
	return interceptor
}
