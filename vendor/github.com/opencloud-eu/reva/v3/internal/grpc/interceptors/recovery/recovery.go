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

package recovery

import (
	"context"

	"runtime/debug"

	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewUnary returns a server interceptor that adds telemetry to
// grpc calls.
func NewUnary() grpc.UnaryServerInterceptor {
	interceptor := grpc_recovery.UnaryServerInterceptor(grpc_recovery.WithRecoveryHandlerContext(recoveryFunc))
	return interceptor
}

// NewStream returns a streaming server interceptor that adds telemetry to
// streaming grpc calls.
func NewStream() grpc.StreamServerInterceptor {
	interceptor := grpc_recovery.StreamServerInterceptor(grpc_recovery.WithRecoveryHandlerContext(recoveryFunc))
	return interceptor
}

func recoveryFunc(ctx context.Context, p interface{}) (err error) {
	debug.PrintStack()
	log := appctx.GetLogger(ctx)
	log.Error().Msgf("%+v; stack: %s", p, debug.Stack())
	return status.Errorf(codes.Internal, "%s", p)
}
