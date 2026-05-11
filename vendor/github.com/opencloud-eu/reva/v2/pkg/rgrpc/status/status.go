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

// Package status contains helpers functions
// to create grpc Status with contextual information,
// like traces.
package status

import (
	"context"
	"errors"
	"net/http"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewOK returns a Status with CODE_OK.
func NewOK(ctx context.Context) *rpc.Status {
	return &rpc.Status{
		Code:  rpc.Code_CODE_OK,
		Trace: getTrace(ctx),
	}
}

// NewNotFound returns a Status with CODE_NOT_FOUND.
func NewNotFound(ctx context.Context, msg string) *rpc.Status {
	return &rpc.Status{
		Code:    rpc.Code_CODE_NOT_FOUND,
		Message: msg,
		Trace:   getTrace(ctx),
	}
}

// NewInvalid returns a Status with CODE_INVALID_ARGUMENT.
func NewInvalid(ctx context.Context, msg string) *rpc.Status {
	return &rpc.Status{
		Code:    rpc.Code_CODE_INVALID_ARGUMENT,
		Message: msg,
		Trace:   getTrace(ctx),
	}
}

// NewInternal returns a Status with CODE_INTERNAL.
func NewInternal(ctx context.Context, msg string) *rpc.Status {
	return &rpc.Status{
		Code:    rpc.Code_CODE_INTERNAL,
		Message: msg,
		Trace:   getTrace(ctx),
	}
}

// NewUnavailable returns a Status with CODE_UNAVAILABLE.
func NewUnavailable(ctx context.Context, msg string) *rpc.Status {
	return &rpc.Status{
		Code:    rpc.Code_CODE_UNAVAILABLE,
		Message: msg,
		Trace:   getTrace(ctx),
	}
}

// NewUnauthenticated returns a Status with CODE_UNAUTHENTICATED.
func NewUnauthenticated(ctx context.Context, err error, msg string) *rpc.Status {
	return &rpc.Status{
		Code:    rpc.Code_CODE_UNAUTHENTICATED,
		Message: msg,
		Trace:   getTrace(ctx),
	}
}

// NewPermissionDenied returns a Status with PERMISSION_DENIED.
func NewPermissionDenied(ctx context.Context, err error, msg string) *rpc.Status {
	return &rpc.Status{
		Code:    rpc.Code_CODE_PERMISSION_DENIED,
		Message: msg,
		Trace:   getTrace(ctx),
	}
}

// NewAborted returns a Status with ABORTED.
func NewAborted(ctx context.Context, err error, msg string) *rpc.Status {
	return &rpc.Status{
		Code:    rpc.Code_CODE_ABORTED,
		Message: msg,
		Trace:   getTrace(ctx),
	}
}

// NewFailedPrecondition returns a Status with FAILED_PRECONDITION.
func NewFailedPrecondition(ctx context.Context, err error, msg string) *rpc.Status {
	return &rpc.Status{
		Code:    rpc.Code_CODE_FAILED_PRECONDITION,
		Message: msg,
		Trace:   getTrace(ctx),
	}
}

// NewInsufficientStorage returns a Status with INSUFFICIENT_STORAGE.
func NewInsufficientStorage(ctx context.Context, err error, msg string) *rpc.Status {
	return &rpc.Status{
		Code:    rpc.Code_CODE_INSUFFICIENT_STORAGE,
		Message: msg,
		Trace:   getTrace(ctx),
	}
}

// NewUnimplemented returns a Status with CODE_UNIMPLEMENTED.
func NewUnimplemented(ctx context.Context, err error, msg string) *rpc.Status {
	return &rpc.Status{
		Code:    rpc.Code_CODE_UNIMPLEMENTED,
		Message: msg,
		Trace:   getTrace(ctx),
	}
}

// NewAlreadyExists returns a Status with CODE_ALREADY_EXISTS.
func NewAlreadyExists(ctx context.Context, err error, msg string) *rpc.Status {
	return &rpc.Status{
		Code:    rpc.Code_CODE_ALREADY_EXISTS,
		Message: msg,
		Trace:   getTrace(ctx),
	}
}

// NewInvalidArg returns a Status with CODE_INVALID_ARGUMENT.
func NewInvalidArg(ctx context.Context, msg string) *rpc.Status {
	return &rpc.Status{Code: rpc.Code_CODE_INVALID_ARGUMENT,
		Message: msg,
		Trace:   getTrace(ctx),
	}
}

// NewConflict returns a Status with Code_CODE_ABORTED.
//
// Deprecated: NewConflict exists for historical compatibility
// and should not be used. To create a Status with code ABORTED,
// use NewAborted.
func NewConflict(ctx context.Context, err error, msg string) *rpc.Status {
	return &rpc.Status{
		Code:    rpc.Code_CODE_ABORTED,
		Message: msg,
		Trace:   getTrace(ctx),
	}
}

// NewLocked returns a status Code_CODE_LOCKED
func NewLocked(ctx context.Context, msg string) *rpc.Status {
	return &rpc.Status{
		Code:    rpc.Code_CODE_LOCKED,
		Message: msg,
		Trace:   getTrace(ctx),
	}
}

// NewStatusFromErrType returns a status that corresponds to the given errtype
func NewStatusFromErrType(ctx context.Context, msg string, err error) *rpc.Status {
	switch e := err.(type) {
	case nil:
		return NewOK(ctx)
	case errtypes.NotFound:
		return NewNotFound(ctx, msg+": "+err.Error())
	case errtypes.IsNotFound:
		return NewNotFound(ctx, msg+": "+err.Error())
	case errtypes.AlreadyExists:
		return NewAlreadyExists(ctx, err, msg+": "+err.Error())
	case errtypes.InvalidCredentials:
		return NewPermissionDenied(ctx, e, msg+": "+err.Error())
	case errtypes.IsInvalidCredentials:
		// TODO this maps badly
		return NewUnauthenticated(ctx, err, msg+": "+err.Error())
	case errtypes.PermissionDenied:
		return NewPermissionDenied(ctx, e, msg+": "+err.Error())
	case errtypes.Locked:
		// FIXME a locked error returns the current lockid
		// FIXME use NewAborted as per the rpc code docs
		return NewLocked(ctx, msg+": "+err.Error())
	case errtypes.Aborted:
		return NewAborted(ctx, e, msg+": "+err.Error())
	case errtypes.PreconditionFailed:
		return NewFailedPrecondition(ctx, e, msg+": "+err.Error())
	case errtypes.IsNotSupported:
		return NewUnimplemented(ctx, err, msg+":"+err.Error())
	case errtypes.BadRequest:
		return NewInvalid(ctx, msg+":"+err.Error())
	case errtypes.Unavailable:
		return NewUnavailable(ctx, msg+": "+err.Error())
	case errtypes.IsUnavailable:
		return NewUnavailable(ctx, msg+": "+err.Error())
	}

	// map GRPC status codes coming from the auth middleware
	grpcErr := err
	for {
		st, ok := status.FromError(grpcErr)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				return NewNotFound(ctx, msg+": "+err.Error())
			case codes.Unauthenticated:
				return NewUnauthenticated(ctx, err, msg+": "+err.Error())
			case codes.PermissionDenied:
				return NewPermissionDenied(ctx, err, msg+": "+err.Error())
			case codes.Unimplemented:
				return NewUnimplemented(ctx, err, msg+": "+err.Error())
			}
		}
		// the actual error can be wrapped multiple times
		grpcErr = errors.Unwrap(grpcErr)
		if grpcErr == nil {
			break
		}
	}

	return NewInternal(ctx, msg+":"+err.Error())
}

// NewErrorFromCode returns a standardized Error for a given RPC code.
func NewErrorFromCode(code rpc.Code, pkgname string) error {
	return errors.New(pkgname + ": grpc failed with code " + code.String())
}

// internal function to attach the trace to a context
func getTrace(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	return span.SpanContext().TraceID().String()
}

// a mapping from the CS3 status codes to http codes
var httpStatusCode = map[rpc.Code]int{
	rpc.Code_CODE_ABORTED:              http.StatusConflict, // webdav uses 412 PreconditionFailed for locks and etags
	rpc.Code_CODE_ALREADY_EXISTS:       http.StatusConflict,
	rpc.Code_CODE_CANCELLED:            499, // Client Closed Request
	rpc.Code_CODE_DATA_LOSS:            http.StatusInternalServerError,
	rpc.Code_CODE_DEADLINE_EXCEEDED:    http.StatusGatewayTimeout,
	rpc.Code_CODE_FAILED_PRECONDITION:  http.StatusPreconditionFailed,
	rpc.Code_CODE_INSUFFICIENT_STORAGE: http.StatusInsufficientStorage,
	rpc.Code_CODE_INTERNAL:             http.StatusInternalServerError,
	rpc.Code_CODE_INVALID:              http.StatusInternalServerError,
	rpc.Code_CODE_INVALID_ARGUMENT:     http.StatusBadRequest,
	rpc.Code_CODE_NOT_FOUND:            http.StatusNotFound,
	rpc.Code_CODE_OK:                   http.StatusOK,
	rpc.Code_CODE_OUT_OF_RANGE:         http.StatusBadRequest,
	rpc.Code_CODE_PERMISSION_DENIED:    http.StatusForbidden,
	rpc.Code_CODE_REDIRECTION:          http.StatusTemporaryRedirect, // or permanent?
	rpc.Code_CODE_RESOURCE_EXHAUSTED:   http.StatusTooManyRequests,
	rpc.Code_CODE_UNAUTHENTICATED:      http.StatusUnauthorized,
	rpc.Code_CODE_UNAVAILABLE:          http.StatusServiceUnavailable,
	rpc.Code_CODE_UNIMPLEMENTED:        http.StatusNotImplemented,
	rpc.Code_CODE_UNKNOWN:              http.StatusInternalServerError,
	rpc.Code_CODE_LOCKED:               http.StatusLocked,
	rpc.Code_CODE_TOO_EARLY:            http.StatusTooEarly,
}

// HTTPStatusFromCode returns an HTTP status code for the rpc code. It returns
// an internal server error (500) if the code is unknown
func HTTPStatusFromCode(code rpc.Code) int {
	if s, ok := httpStatusCode[code]; ok {
		return s
	}
	return http.StatusInternalServerError
}
