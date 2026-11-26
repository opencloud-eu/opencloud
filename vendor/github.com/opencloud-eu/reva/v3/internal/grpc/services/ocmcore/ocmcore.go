// Copyright 2018-2023 CERN
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

package ocmcore

import (
	"context"
	"errors"
	"fmt"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocmcore "github.com/cs3org/go-cs3apis/cs3/ocm/core/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	providerpb "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/errtypes"
	"github.com/opencloud-eu/reva/v3/pkg/events"
	"github.com/opencloud-eu/reva/v3/pkg/events/stream"
	"github.com/opencloud-eu/reva/v3/pkg/ocm/share"
	"github.com/opencloud-eu/reva/v3/pkg/ocm/share/repository/registry"
	ocmuser "github.com/opencloud-eu/reva/v3/pkg/ocm/user"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
	"github.com/opencloud-eu/reva/v3/pkg/utils/cfg"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func init() {
	rgrpc.Register("ocmcore", New)
}

// EventOptions are the configurable options for events
type EventOptions struct {
	Endpoint             string `mapstructure:"natsaddress"`
	Cluster              string `mapstructure:"natsclusterid"`
	TLSInsecure          bool   `mapstructure:"tlsinsecure"`
	TLSRootCACertificate string `mapstructure:"tlsrootcacertificate"`
	EnableTLS            bool   `mapstructure:"enabletls"`
	AuthUsername         string `mapstructure:"authusername"`
	AuthPassword         string `mapstructure:"authpassword"`
}

type config struct {
	Driver  string                            `mapstructure:"driver"`
	Drivers map[string]map[string]interface{} `mapstructure:"drivers"`
	Events  EventOptions                      `mapstructure:"events"`
}

type service struct {
	conf        *config
	repo        share.Repository
	eventStream events.Stream
	log         *zerolog.Logger
}

func (c *config) ApplyDefaults() {
	if c.Driver == "" {
		c.Driver = "json"
	}
}

func (s *service) Register(ss *grpc.Server) {
	ocmcore.RegisterOcmCoreAPIServer(ss, s)
}

func getShareRepository(c *config) (share.Repository, error) {
	if f, ok := registry.NewFuncs[c.Driver]; ok {
		return f(c.Drivers[c.Driver])
	}
	return nil, errtypes.NotFound(fmt.Sprintf("driver not found: %s", c.Driver))
}

// New creates a new ocm core svc.
func New(m map[string]interface{}, ss *grpc.Server, log *zerolog.Logger) (rgrpc.Service, error) {
	var c config
	if err := cfg.Decode(m, &c); err != nil {
		return nil, err
	}

	repo, err := getShareRepository(&c)
	if err != nil {
		return nil, err
	}

	service := &service{
		conf: &c,
		repo: repo,
		log:  log,
	}

	if c.Events.Endpoint != "" {
		es, err := stream.NatsFromConfig("ocmcore-handler", false, stream.NatsConfig(c.Events))
		if err != nil {
			return nil, err
		}
		service.eventStream = es
	}

	return service, nil
}

func (s *service) Close() error {
	return nil
}

func (s *service) UnprotectedEndpoints() []string {
	return []string{
		ocmcore.OcmCoreAPI_CreateOCMCoreShare_FullMethodName,
		ocmcore.OcmCoreAPI_UpdateOCMCoreShare_FullMethodName,
		ocmcore.OcmCoreAPI_DeleteOCMCoreShare_FullMethodName,
	}
}

// CreateOCMCoreShare is called when an OCM request comes into this reva instance from.
func (s *service) CreateOCMCoreShare(ctx context.Context, req *ocmcore.CreateOCMCoreShareRequest) (*ocmcore.CreateOCMCoreShareResponse, error) {
	if req.ShareType != ocm.ShareType_SHARE_TYPE_USER {
		return nil, errtypes.NotSupported("share type not supported")
	}

	now := &typespb.Timestamp{
		Seconds: uint64(time.Now().Unix()),
	}

	share, err := s.repo.StoreReceivedShare(ctx, &ocm.ReceivedShare{
		RemoteShareId: req.ResourceId,
		Name:          req.Name,
		Grantee: &providerpb.Grantee{
			Type: providerpb.GranteeType_GRANTEE_TYPE_USER,
			Id: &providerpb.Grantee_UserId{
				UserId: req.ShareWith,
			},
		},
		ResourceType: req.ResourceType,
		ShareType:    req.ShareType,
		Owner:        req.Owner,
		Creator:      req.Sender,
		Protocols:    req.Protocols,
		Ctime:        now,
		Mtime:        now,
		Expiration:   req.Expiration,
		State:        ocm.ShareState_SHARE_STATE_PENDING,
	})
	if err != nil {
		// TODO: identify errors
		return &ocmcore.CreateOCMCoreShareResponse{
			Status: status.NewInternal(ctx, err.Error()),
		}, nil
	}

	var permissions *providerpb.ResourcePermissions
	for _, p := range req.GetProtocols() {
		if p.GetWebdavOptions() != nil {
			permissions = p.GetWebdavOptions().GetPermissions().GetPermissions()
			break
		}
	}

	if s.eventStream != nil {
		if err := events.Publish(ctx, s.eventStream, events.OCMCoreShareCreated{
			ShareID:       share.Id.OpaqueId,
			Executant:     share.Creator,
			Sharer:        share.Creator,
			GranteeUserID: share.Grantee.GetUserId(),
			ItemID:        share.RemoteShareId,
			ResourceName:  share.Name,
			CTime:         share.Ctime,
			Permissions:   permissions,
		}); err != nil {
			s.log.Error().Err(err).
				Msg("failed to publish the ocmcore share created event")
		}
	}

	return &ocmcore.CreateOCMCoreShareResponse{
		Status:  status.NewOK(ctx),
		Id:      share.Id.OpaqueId,
		Created: share.Ctime,
	}, nil
}

func (s *service) UpdateOCMCoreShare(ctx context.Context, req *ocmcore.UpdateOCMCoreShareRequest) (*ocmcore.UpdateOCMCoreShareResponse, error) {
	grantee := utils.ReadPlainFromOpaque(req.GetOpaque(), "grantee")
	if grantee == "" {
		return nil, errtypes.UserRequired("missing remote user id in a metadata")
	}
	if req == nil || len(req.Protocols) == 0 {
		return nil, errtypes.PreconditionFailed("missing protocols in a request")
	}
	fileMask := &fieldmaskpb.FieldMask{Paths: []string{"protocols"}}

	user := &userpb.User{Id: ocmuser.DecodeRemoteUserFederatedID(&userpb.UserId{OpaqueId: grantee})}
	protocols, err := s.mergeWebDAVPermissions(ctx, user, req.GetOcmShareId(), req.Protocols)
	if err == nil {
		_, err = s.repo.UpdateReceivedShare(ctx, user, &ocm.ReceivedShare{
			Id: &ocm.ShareId{
				OpaqueId: req.GetOcmShareId(),
			},
			Protocols: protocols,
		}, fileMask)
	}
	res := &ocmcore.UpdateOCMCoreShareResponse{}
	if err == nil {
		res.Status = status.NewOK(ctx)
	} else {
		var notFound errtypes.NotFound
		if errors.As(err, &notFound) {
			res.Status = status.NewNotFound(ctx, "remote ocm share not found")
		} else {
			res.Status = status.NewInternal(ctx, "error deleting remote ocm share")
		}
	}
	return res, nil
}

// mergeWebDAVPermissions takes only the webdav permissions from a permission change notification,
// so a notification cannot replace the uri, the shared secret or other protocols of the stored share
func (s *service) mergeWebDAVPermissions(ctx context.Context, user *userpb.User, shareID string, notified []*ocm.Protocol) ([]*ocm.Protocol, error) {
	rs, err := s.repo.GetReceivedShare(ctx, user, &ocm.ShareReference{
		Spec: &ocm.ShareReference_Id{
			Id: &ocm.ShareId{
				OpaqueId: shareID,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	var permissions *ocm.SharePermissions
	for _, p := range notified {
		if wd := p.GetWebdavOptions(); wd != nil {
			permissions = wd.GetPermissions()
			break
		}
	}

	protocols := make([]*ocm.Protocol, 0, len(rs.GetProtocols()))
	for _, p := range rs.GetProtocols() {
		if p.GetWebdavOptions() != nil && permissions != nil {
			p = proto.Clone(p).(*ocm.Protocol)
			p.GetWebdavOptions().Permissions = permissions
		}
		protocols = append(protocols, p)
	}
	return protocols, nil
}

func (s *service) DeleteOCMCoreShare(ctx context.Context, req *ocmcore.DeleteOCMCoreShareRequest) (*ocmcore.DeleteOCMCoreShareResponse, error) {
	grantee := utils.ReadPlainFromOpaque(req.GetOpaque(), "grantee")
	if grantee == "" {
		return nil, errtypes.UserRequired("missing remote user id in a metadata")
	}

	share, err := s.repo.GetReceivedShare(ctx, &userpb.User{Id: ocmuser.DecodeRemoteUserFederatedID(&userpb.UserId{OpaqueId: grantee})}, &ocm.ShareReference{
		Spec: &ocm.ShareReference_Id{
			Id: &ocm.ShareId{
				OpaqueId: req.GetId(),
			},
		},
	})
	if err != nil {
		var notFound errtypes.NotFound
		if errors.As(err, &notFound) {
			return &ocmcore.DeleteOCMCoreShareResponse{
				Status: status.NewNotFound(ctx, "remote ocm share not found"),
			}, nil
		}
		return nil, errtypes.InternalError("unable to get share details")
	}

	granteeUser := &userpb.User{Id: ocmuser.DecodeRemoteUserFederatedID(&userpb.UserId{OpaqueId: grantee})}
	err = s.repo.DeleteReceivedShare(ctx, granteeUser, &ocm.ShareReference{
		Spec: &ocm.ShareReference_Id{
			Id: &ocm.ShareId{
				OpaqueId: req.GetId(),
			},
		},
	})

	res := &ocmcore.DeleteOCMCoreShareResponse{}
	if err == nil {
		res.Status = status.NewOK(ctx)

		if s.eventStream != nil {
			if err := events.Publish(ctx, s.eventStream, events.OCMCoreShareDelete{
				ShareID:      share.Id.OpaqueId,
				Sharer:       share.GetOwner(),
				Grantee:      ocmuser.DecodeRemoteUserFederatedID(&userpb.UserId{OpaqueId: grantee}),
				ResourceName: share.Name,
				CTime:        &typespb.Timestamp{Seconds: uint64(time.Now().Unix())},
			}); err != nil {
				s.log.Error().Err(err).
					Msg("failed to publish the ocmcore share deleted event")
			}
		}

	} else {
		var notFound errtypes.NotFound
		if errors.As(err, &notFound) {
			res.Status = status.NewNotFound(ctx, "remote ocm share not found")
		} else {
			res.Status = status.NewInternal(ctx, "error deleting remote ocm share")
		}
	}
	return res, nil
}
