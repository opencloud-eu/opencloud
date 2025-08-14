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

package eventsmiddleware

import (
	"context"
	"fmt"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocmcore "github.com/cs3org/go-cs3apis/cs3/ocm/core/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/events/stream"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"google.golang.org/grpc"
)

const (
	defaultPriority = 200
)

func init() {
	rgrpc.RegisterUnaryInterceptor("eventsmiddleware", NewUnary)
}

// NewUnary returns a new unary interceptor that emits events when needed
// no lint because of the switch statement that should be extendable
//
//nolint:gocritic
func NewUnary(m map[string]interface{}) (grpc.UnaryServerInterceptor, int, error) {
	publisher, err := publisherFromConfig(m)
	if err != nil {
		return nil, 0, err
	}

	interceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Register a channel in the context to receive the space owner id from the handler(s) further down the stack
		var ownerID *user.UserId
		sendOwnerChan := make(chan *user.UserId)
		ctx = storagespace.ContextRegisterSendOwnerChan(ctx, sendOwnerChan)

		res, err := handler(ctx, req)
		if err != nil {
			return res, err
		}

		// Read the space owner id from the channel
		select {
		case ownerID = <-sendOwnerChan:
		default:
		}

		executant, _ := revactx.ContextGetUser(ctx)

		// The MoveResponse event is moved to the decomposedfs
		var ev interface{}
		switch v := res.(type) {
		case *collaboration.CreateShareResponse:
			if isSuccess(v) {
				ev = ShareCreated(v, executant)
			}
		case *collaboration.RemoveShareResponse:
			if isSuccess(v) {
				ev = ShareRemoved(v, req.(*collaboration.RemoveShareRequest), executant)
			}
		case *collaboration.UpdateShareResponse:
			if isSuccess(v) {
				ev = ShareUpdated(v, req.(*collaboration.UpdateShareRequest), executant)
			}
		case *collaboration.UpdateReceivedShareResponse:
			if isSuccess(v) {
				ev = ReceivedShareUpdated(v, executant)
			}
		case *link.CreatePublicShareResponse:
			if isSuccess(v) {
				ev = LinkCreated(v, executant)
			}
		case *link.UpdatePublicShareResponse:
			if isSuccess(v) {
				ev = LinkUpdated(v, req.(*link.UpdatePublicShareRequest), executant)
			}
		case *link.RemovePublicShareResponse:
			if isSuccess(v) {
				ev = LinkRemoved(v, req.(*link.RemovePublicShareRequest), executant)
			}
		case *link.GetPublicShareByTokenResponse:
			if isSuccess(v) {
				ev = LinkAccessed(v, executant)
			} else {
				ev = LinkAccessFailed(v, req.(*link.GetPublicShareByTokenRequest), executant)
			}
		case *ocmcore.CreateOCMCoreShareResponse:
			if isSuccess(v) {
				ev = OCMCoreShareCreated(v, req.(*ocmcore.CreateOCMCoreShareRequest), executant)
			}
		case *provider.AddGrantResponse:
			// TODO: update CS3 APIs
			// FIXME these should be part of the RemoveGrantRequest object
			// https://github.com/owncloud/ocis/issues/4312
			r := req.(*provider.AddGrantRequest)
			if isSuccess(v) && utils.ExistsInOpaque(r.Opaque, "spacegrant") {
				ev = SpaceShared(v, r, executant)
			}
		case *provider.UpdateGrantResponse:
			r := req.(*provider.UpdateGrantRequest)
			if isSuccess(v) && utils.ExistsInOpaque(r.Opaque, "spacegrant") {
				ev = SpaceShareUpdated(v, r, executant)
			}
		case *provider.RemoveGrantResponse:
			r := req.(*provider.RemoveGrantRequest)
			if isSuccess(v) && utils.ExistsInOpaque(r.Opaque, "spacegrant") {
				ev = SpaceUnshared(v, req.(*provider.RemoveGrantRequest), executant)
			}
		case *provider.CreateContainerResponse:
			if isSuccess(v) {
				ev = ContainerCreated(v, req.(*provider.CreateContainerRequest), ownerID, executant)
			}
		case *provider.InitiateFileDownloadResponse:
			if isSuccess(v) {
				ev = FileDownloaded(v, req.(*provider.InitiateFileDownloadRequest), executant)
			}
		case *provider.DeleteResponse:
			if isSuccess(v) {
				ev = ItemTrashed(v, req.(*provider.DeleteRequest), ownerID, executant)
			}
		case *provider.PurgeRecycleResponse:
			if isSuccess(v) {
				id := req.(*provider.PurgeRecycleRequest).GetRef().GetResourceId()
				if id != nil && id.OpaqueId == id.SpaceId && req.(*provider.PurgeRecycleRequest).Key == "" {
					// this is a purge of the whole trashbin, NOT the whole space
					ev = TrashbinPurged(v, req.(*provider.PurgeRecycleRequest), executant)
				} else {
					ev = ItemPurged(v, req.(*provider.PurgeRecycleRequest), executant)
				}
			}
		case *provider.RestoreRecycleItemResponse:
			if isSuccess(v) {
				ev = ItemRestored(v, req.(*provider.RestoreRecycleItemRequest), ownerID, executant)
			}
		case *provider.RestoreFileVersionResponse:
			if isSuccess(v) {
				ev = FileVersionRestored(v, req.(*provider.RestoreFileVersionRequest), ownerID, executant)
			}
		case *provider.CreateStorageSpaceResponse:
			if isSuccess(v) && v.StorageSpace != nil { // TODO: Why are there CreateStorageSpaceResponses with nil StorageSpace?
				ev = SpaceCreated(v, executant)
			}
		case *provider.UpdateStorageSpaceResponse:
			if isSuccess(v) {
				r := req.(*provider.UpdateStorageSpaceRequest)
				if r.StorageSpace.Name != "" {
					ev = SpaceRenamed(v, r, executant)
				} else if utils.ExistsInOpaque(r.Opaque, "restore") {
					ev = SpaceEnabled(v, r, executant)
				} else {
					ev = SpaceUpdated(v, r, executant)
				}
			}
		case *provider.DeleteStorageSpaceResponse:
			if isSuccess(v) {
				r := req.(*provider.DeleteStorageSpaceRequest)
				if utils.ExistsInOpaque(r.Opaque, "purge") {
					ev = SpaceDeleted(v, r, executant)
				} else {
					ev = SpaceDisabled(v, r, executant)
				}
			}
		case *provider.TouchFileResponse:
			if isSuccess(v) {
				ev = FileTouched(v, req.(*provider.TouchFileRequest), ownerID, executant)
			}
		case *provider.SetLockResponse:
			if isSuccess(v) {
				ev = FileLocked(v, req.(*provider.SetLockRequest), ownerID, executant)
			}
		case *provider.UnlockResponse:
			if isSuccess(v) {
				ev = FileUnlocked(v, req.(*provider.UnlockRequest), ownerID, executant)
			}
		}

		if ev != nil {
			if err := events.Publish(ctx, publisher, ev); err != nil {
				appctx.GetLogger(ctx).Error().Err(err).Interface("event", ev).Msg("publishing event failed")
			}
		}

		return res, nil
	}
	return interceptor, defaultPriority, nil
}

// NewStream returns a new server stream interceptor
// that creates the application context.
func NewStream() grpc.StreamServerInterceptor {
	interceptor := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// TODO: Use ss.RecvMsg() and ss.SendMsg() to send events from a stream
		return handler(srv, ss)
	}
	return interceptor
}

// common interface to all responses
type su interface {
	GetStatus() *rpc.Status
}

func isSuccess(res su) bool {
	return res.GetStatus().Code == rpc.Code_CODE_OK
}

func publisherFromConfig(m map[string]interface{}) (events.Publisher, error) {
	typ := m["type"].(string)
	switch typ {
	default:
		return nil, fmt.Errorf("stream type '%s' not supported", typ)
	case "nats":
		var cfg stream.NatsConfig
		if err := mapstructure.Decode(m, &cfg); err != nil {
			return nil, err
		}
		name, _ := m["name"].(string)
		return stream.NatsFromConfig(name, false, cfg)
	}
}
