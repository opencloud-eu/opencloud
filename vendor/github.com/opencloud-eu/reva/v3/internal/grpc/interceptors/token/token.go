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

package token

import (
	"context"

	ctxpkg "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// NewUnary returns a new unary interceptor that adds
// the token to the context.
func NewUnary() grpc.UnaryServerInterceptor {
	interceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok && md != nil {
			if val, ok := md[ctxpkg.TokenHeader]; ok {
				if len(val) > 0 && val[0] != "" {
					tkn := val[0]
					ctx = ctxpkg.ContextSetToken(ctx, tkn)
					ctx = metadata.AppendToOutgoingContext(ctx, ctxpkg.TokenHeader, tkn)
				}
			}

			if val, ok := md[ctxpkg.InitiatorHeader]; ok {
				if len(val) > 0 && val[0] != "" {
					initiatorID := val[0]
					ctx = ctxpkg.ContextSetInitiator(ctx, initiatorID)
					ctx = metadata.AppendToOutgoingContext(ctx, ctxpkg.InitiatorHeader, initiatorID)
				}
			}
		}

		return handler(ctx, req)
	}
	return interceptor
}

// NewStream returns a new server stream interceptor
// that adds trace information to the request.
func NewStream() grpc.StreamServerInterceptor {
	interceptor := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		md, ok := metadata.FromIncomingContext(ss.Context())
		if ok && md != nil {
			if val, ok := md[ctxpkg.TokenHeader]; ok {
				if len(val) > 0 && val[0] != "" {
					tkn := val[0]
					ctx = ctxpkg.ContextSetToken(ctx, tkn)
					ctx = metadata.AppendToOutgoingContext(ctx, ctxpkg.TokenHeader, tkn)
				}
			}

			if val, ok := md[ctxpkg.InitiatorHeader]; ok {
				if len(val) > 0 && val[0] != "" {
					initiatorID := val[0]
					ctx = ctxpkg.ContextSetInitiator(ctx, initiatorID)
					ctx = metadata.AppendToOutgoingContext(ctx, ctxpkg.InitiatorHeader, initiatorID)
				}
			}
		}

		wrapped := newWrappedServerStream(ctx, ss)
		return handler(srv, wrapped)
	}
	return interceptor
}

func newWrappedServerStream(ctx context.Context, ss grpc.ServerStream) *wrappedServerStream {
	return &wrappedServerStream{ServerStream: ss, newCtx: ctx}
}

type wrappedServerStream struct {
	grpc.ServerStream
	newCtx context.Context
}

func (ss *wrappedServerStream) Context() context.Context {
	return ss.newCtx
}
