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

package ocmshareprovider

import (
	"context"
	"net/url"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	ocmprovider "github.com/cs3org/go-cs3apis/cs3/ocm/provider/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	providerpb "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typespb "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/opencloud-eu/reva/v2/internal/http/services/ocmd"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/ocm/client"
	"github.com/opencloud-eu/reva/v2/pkg/ocm/share"
	"github.com/opencloud-eu/reva/v2/pkg/ocm/share/repository/registry"
	ocmuser "github.com/opencloud-eu/reva/v2/pkg/ocm/user"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/walker"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"github.com/opencloud-eu/reva/v2/pkg/utils/cfg"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

func init() {
	rgrpc.Register("ocmshareprovider", New)
}

type config struct {
	Driver         string                            `mapstructure:"driver"`
	Drivers        map[string]map[string]interface{} `mapstructure:"drivers"`
	ClientTimeout  int                               `mapstructure:"client_timeout"`
	ClientInsecure bool                              `mapstructure:"client_insecure"`
	GatewaySVC     string                            `mapstructure:"gatewaysvc"      validate:"required"`
	ProviderDomain string                            `mapstructure:"provider_domain" validate:"required" docs:"The same domain registered in the provider authorizer"`
	WebDAVEndpoint string                            `mapstructure:"webdav_endpoint" validate:"required"`
	WebappTemplate string                            `mapstructure:"webapp_template"`
}

type service struct {
	conf            *config
	repo            share.Repository
	client          *client.OCMClient
	gatewaySelector *pool.Selector[gateway.GatewayAPIClient]
	webappTmpl      *template.Template
	walker          walker.Walker
}

func (c *config) ApplyDefaults() {
	if c.Driver == "" {
		c.Driver = "json"
	}
	if c.ClientTimeout == 0 {
		c.ClientTimeout = 10
	}
	if c.WebappTemplate == "" {
		c.WebappTemplate = "https://cernbox.cern.ch/external/sciencemesh/{{.Token}}{relative-path-to-shared-resource}"
	}

	c.GatewaySVC = sharedconf.GetGatewaySVC(c.GatewaySVC)
}

func (s *service) Register(ss *grpc.Server) {
	ocm.RegisterOcmAPIServer(ss, s)
}

func getShareRepository(c *config) (share.Repository, error) {
	if f, ok := registry.NewFuncs[c.Driver]; ok {
		return f(c.Drivers[c.Driver])
	}
	return nil, errtypes.NotFound("driver not found: " + c.Driver)
}

// New creates a new ocm share provider svc.
func New(m map[string]interface{}, ss *grpc.Server, _ *zerolog.Logger) (rgrpc.Service, error) {
	var c config
	if err := cfg.Decode(m, &c); err != nil {
		return nil, err
	}

	repo, err := getShareRepository(&c)
	if err != nil {
		return nil, err
	}

	client := client.New(&client.Config{
		Timeout:  time.Duration(c.ClientTimeout) * time.Second,
		Insecure: c.ClientInsecure,
	})

	gatewaySelector, err := pool.GatewaySelector(c.GatewaySVC)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("webapp_template").Parse(c.WebappTemplate)
	if err != nil {
		return nil, err
	}
	walker := walker.NewWalker(gatewaySelector)

	service := &service{
		conf:            &c,
		repo:            repo,
		client:          client,
		gatewaySelector: gatewaySelector,
		webappTmpl:      tpl,
		walker:          walker,
	}

	return service, nil
}

func (s *service) Close() error {
	return nil
}

func (s *service) UnprotectedEndpoints() []string {
	return []string{"/cs3.sharing.ocm.v1beta1.OcmAPI/GetOCMShareByToken"}
}

func getOCMEndpoint(originProvider *ocmprovider.ProviderInfo) (string, error) {
	for _, s := range originProvider.Services {
		if s.Endpoint.Type.Name == "OCM" {
			return s.Endpoint.Path, nil
		}
	}
	return "", errors.New("ocm endpoint not specified for mesh provider")
}

func getResourceType(info *providerpb.ResourceInfo) string {
	switch info.Type {
	case providerpb.ResourceType_RESOURCE_TYPE_FILE:
		return "file"
	case providerpb.ResourceType_RESOURCE_TYPE_CONTAINER:
		return "folder"
	}
	return "unknown"
}

func (s *service) webdavURL(_ context.Context, share *ocm.Share) string {
	// the url is in the form of https://cernbox.cern.ch/remote.php/dav/ocm/token
	p, _ := url.JoinPath(s.conf.WebDAVEndpoint, "/dav/ocm", share.GetId().GetOpaqueId())
	return p
}

func (s *service) getWebdavProtocol(ctx context.Context, share *ocm.Share, m *ocm.AccessMethod_WebdavOptions) *ocmd.WebDAV {
	var perms []string
	if m.WebdavOptions.Permissions.InitiateFileDownload {
		perms = append(perms, "read")
	}
	if m.WebdavOptions.Permissions.InitiateFileUpload {
		perms = append(perms, "write")
	}

	return &ocmd.WebDAV{
		Permissions:  perms,
		URI:          s.webdavURL(ctx, share),
		SharedSecret: share.Token,
	}
}

func (s *service) getWebappProtocol(share *ocm.Share) *ocmd.Webapp {
	var b strings.Builder
	if err := s.webappTmpl.Execute(&b, share); err != nil {
		return nil
	}
	return &ocmd.Webapp{
		URITemplate: b.String(),
	}
}

func (s *service) getDataTransferProtocol(ctx context.Context, share *ocm.Share) *ocmd.Datatx {
	var size uint64

	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil
	}
	// get the path of the share
	statRes, err := gatewayClient.Stat(ctx, &providerpb.StatRequest{
		Ref: &providerpb.Reference{
			ResourceId: share.ResourceId,
		},
	})
	if err != nil {
		return nil
	}

	err = s.walker.Walk(ctx, statRes.GetInfo().GetId(), func(path string, info *providerpb.ResourceInfo, err error) error {
		if info.Type == providerpb.ResourceType_RESOURCE_TYPE_FILE {
			size += info.Size
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return &ocmd.Datatx{
		SourceURI: s.webdavURL(ctx, share),
		Size:      size,
	}
}

func (s *service) getProtocols(ctx context.Context, share *ocm.Share) ocmd.Protocols {
	var p ocmd.Protocols
	for _, m := range share.AccessMethods {
		var newProtocol ocmd.Protocol
		switch t := m.Term.(type) {
		case *ocm.AccessMethod_WebdavOptions:
			newProtocol = s.getWebdavProtocol(ctx, share, t)
		case *ocm.AccessMethod_WebappOptions:
			newProtocol = s.getWebappProtocol(share)
		case *ocm.AccessMethod_TransferOptions:
			newProtocol = s.getDataTransferProtocol(ctx, share)
		}
		if newProtocol != nil {
			p = append(p, newProtocol)
		}
	}
	return p
}

func (s *service) CreateOCMShare(ctx context.Context, req *ocm.CreateOCMShareRequest) (*ocm.CreateOCMShareResponse, error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	statRes, err := gatewayClient.Stat(ctx, &providerpb.StatRequest{
		Ref: &providerpb.Reference{
			ResourceId: req.ResourceId,
		},
	})
	if err != nil {
		return nil, err
	}

	if statRes.Status.Code != rpc.Code_CODE_OK {
		if statRes.Status.Code == rpc.Code_CODE_NOT_FOUND {
			return &ocm.CreateOCMShareResponse{
				Status: status.NewNotFound(ctx, statRes.Status.Message),
			}, nil
		}
		return &ocm.CreateOCMShareResponse{
			Status: status.NewInternal(ctx, statRes.Status.Message),
		}, nil
	}

	info := statRes.Info
	user := ctxpkg.ContextMustGetUser(ctx)
	tkn := utils.RandString(32)
	now := time.Now().UnixNano()
	ts := &typespb.Timestamp{
		Seconds: uint64(now / 1000000000),
		Nanos:   uint32(now % 1000000000),
	}

	// 1. persist the share in the repository
	ocmshare := &ocm.Share{
		Token:         tkn,
		Name:          filepath.Base(info.Path),
		ResourceId:    req.ResourceId,
		Grantee:       req.Grantee,
		ShareType:     ocm.ShareType_SHARE_TYPE_USER,
		Owner:         info.Owner,
		Creator:       user.Id,
		Ctime:         ts,
		Mtime:         ts,
		Expiration:    req.Expiration,
		AccessMethods: req.AccessMethods,
	}

	ocmshare, err = s.repo.StoreShare(ctx, ocmshare)
	if err != nil {
		if errors.Is(err, share.ErrShareAlreadyExisting) {
			return &ocm.CreateOCMShareResponse{
				Status: status.NewAlreadyExists(ctx, err, "share already exists"),
			}, nil
		}
		return &ocm.CreateOCMShareResponse{
			Status: status.NewInternal(ctx, err.Error()),
		}, nil
	}

	// 2. create the share on the remote provider
	// 2.a get the ocm endpoint of the remote provider
	ocmEndpoint, err := getOCMEndpoint(req.RecipientMeshProvider)
	if err != nil {
		return &ocm.CreateOCMShareResponse{
			Status: status.NewInvalidArg(ctx, "the selected provider does not have an OCM endpoint"),
		}, nil
	}

	// 2.b replace outgoing user ids with ocm user ids
	// unpack the federated user id
	shareWith := ocmuser.FormatOCMUser(ocmuser.LocalUserFederatedID(req.GetGrantee().GetUserId(), ""))

	// wrap the local user id in a local federated user id
	owner := ocmuser.FormatOCMUser(ocmuser.LocalUserFederatedID(info.Owner, s.conf.ProviderDomain))
	sender := ocmuser.FormatOCMUser(ocmuser.LocalUserFederatedID(user.Id, s.conf.ProviderDomain))

	newShareReq := &client.NewShareRequest{
		ShareWith:         shareWith,
		Name:              ocmshare.Name,
		ProviderID:        ocmshare.Id.OpaqueId,
		Owner:             owner,
		Sender:            sender,
		SenderDisplayName: user.DisplayName,
		ShareType:         "user",
		ResourceType:      getResourceType(info),
		Protocols:         s.getProtocols(ctx, ocmshare),
	}

	if req.Expiration != nil {
		newShareReq.Expiration = req.Expiration.Seconds
	}

	// 2.c make POST /shares request
	newShareRes, err := s.client.NewShare(ctx, ocmEndpoint, newShareReq)
	if err != nil {
		err2 := s.repo.DeleteShare(ctx, user, &ocm.ShareReference{Spec: &ocm.ShareReference_Id{Id: ocmshare.Id}})
		if err2 != nil {
			appctx.GetLogger(ctx).Error().Err(err2).Str("shareid", ocmshare.GetId().GetOpaqueId()).Msg("could not delete local ocm share")
		}
		// TODO remove the share from the local storage
		switch {
		case errors.Is(err, client.ErrInvalidParameters):
			return &ocm.CreateOCMShareResponse{
				Status: status.NewInvalidArg(ctx, err.Error()),
			}, nil
		case errors.Is(err, client.ErrServiceNotTrusted):
			return &ocm.CreateOCMShareResponse{
				Status: status.NewInvalidArg(ctx, err.Error()),
			}, nil
		default:
			return &ocm.CreateOCMShareResponse{
				Status: status.NewInternal(ctx, err.Error()),
			}, nil
		}
	}

	res := &ocm.CreateOCMShareResponse{
		Status:               status.NewOK(ctx),
		Share:                ocmshare,
		RecipientDisplayName: newShareRes.RecipientDisplayName,
	}
	return res, nil
}

func (s *service) RemoveOCMShare(ctx context.Context, req *ocm.RemoveOCMShareRequest) (*ocm.RemoveOCMShareResponse, error) {
	// TODO (gdelmont): notify the remote provider using the /notification ocm endpoint
	// https://cs3org.github.io/OCM-API/docs.html?branch=develop&repo=OCM-API&user=cs3org#/paths/~1notifications/post
	user := ctxpkg.ContextMustGetUser(ctx)
	if err := s.repo.DeleteShare(ctx, user, req.Ref); err != nil {
		if errors.Is(err, share.ErrShareNotFound) {
			return &ocm.RemoveOCMShareResponse{
				Status: status.NewNotFound(ctx, "share does not exist"),
			}, nil
		}
		return &ocm.RemoveOCMShareResponse{
			Status: status.NewInternal(ctx, "error removing share"),
		}, nil
	}

	return &ocm.RemoveOCMShareResponse{
		Status: status.NewOK(ctx),
	}, nil
}

func (s *service) GetOCMShare(ctx context.Context, req *ocm.GetOCMShareRequest) (*ocm.GetOCMShareResponse, error) {
	// if the request is by token, the user does not need to be in the ctx
	var user *userpb.User
	if req.Ref.GetToken() == "" {
		user = ctxpkg.ContextMustGetUser(ctx)
	}
	ocmshare, err := s.repo.GetShare(ctx, user, req.Ref)
	if err != nil {
		if errors.Is(err, share.ErrShareNotFound) {
			return &ocm.GetOCMShareResponse{
				Status: status.NewNotFound(ctx, "share does not exist"),
			}, nil
		}
		return &ocm.GetOCMShareResponse{
			Status: status.NewInternal(ctx, "error getting share"),
		}, nil
	}

	return &ocm.GetOCMShareResponse{
		Status: status.NewOK(ctx),
		Share:  ocmshare,
	}, nil
}

func (s *service) GetOCMShareByToken(ctx context.Context, req *ocm.GetOCMShareByTokenRequest) (*ocm.GetOCMShareByTokenResponse, error) {
	ocmshare, err := s.repo.GetShare(ctx, nil, &ocm.ShareReference{
		Spec: &ocm.ShareReference_Token{
			Token: req.Token,
		},
	})
	if err != nil {
		if errors.Is(err, share.ErrShareNotFound) {
			return &ocm.GetOCMShareByTokenResponse{
				Status: status.NewNotFound(ctx, "share does not exist"),
			}, nil
		}
		return &ocm.GetOCMShareByTokenResponse{
			Status: status.NewInternal(ctx, "error getting share"),
		}, nil
	}

	return &ocm.GetOCMShareByTokenResponse{
		Status: status.NewOK(ctx),
		Share:  ocmshare,
	}, nil
}

func (s *service) ListOCMShares(ctx context.Context, req *ocm.ListOCMSharesRequest) (*ocm.ListOCMSharesResponse, error) {
	user := ctxpkg.ContextMustGetUser(ctx)
	shares, err := s.repo.ListShares(ctx, user, req.Filters)
	if err != nil {
		return &ocm.ListOCMSharesResponse{
			Status: status.NewInternal(ctx, "error listing shares"),
		}, nil
	}

	res := &ocm.ListOCMSharesResponse{
		Status: status.NewOK(ctx),
		Shares: shares,
	}
	return res, nil
}

func (s *service) UpdateOCMShare(ctx context.Context, req *ocm.UpdateOCMShareRequest) (*ocm.UpdateOCMShareResponse, error) {
	user := ctxpkg.ContextMustGetUser(ctx)
	if len(req.Field) == 0 {
		return &ocm.UpdateOCMShareResponse{
			Status: status.NewOK(ctx),
		}, nil
	}
	_, err := s.repo.UpdateShare(ctx, user, req.Ref, req.Field...)
	if err != nil {
		if errors.Is(err, share.ErrShareNotFound) {
			return &ocm.UpdateOCMShareResponse{
				Status: status.NewNotFound(ctx, "share does not exist"),
			}, nil
		}
		return &ocm.UpdateOCMShareResponse{
			Status: status.NewInternal(ctx, "error updating share"),
		}, nil
	}

	res := &ocm.UpdateOCMShareResponse{
		Status: status.NewOK(ctx),
	}
	return res, nil
}

func (s *service) ListReceivedOCMShares(ctx context.Context, req *ocm.ListReceivedOCMSharesRequest) (*ocm.ListReceivedOCMSharesResponse, error) {
	user := ctxpkg.ContextMustGetUser(ctx)
	shares, err := s.repo.ListReceivedShares(ctx, user)
	if err != nil {
		return &ocm.ListReceivedOCMSharesResponse{
			Status: status.NewInternal(ctx, "error listing received shares"),
		}, nil
	}

	res := &ocm.ListReceivedOCMSharesResponse{
		Status: status.NewOK(ctx),
		Shares: shares,
	}
	return res, nil
}

func (s *service) UpdateReceivedOCMShare(ctx context.Context, req *ocm.UpdateReceivedOCMShareRequest) (*ocm.UpdateReceivedOCMShareResponse, error) {
	user := ctxpkg.ContextMustGetUser(ctx)
	_, err := s.repo.UpdateReceivedShare(ctx, user, req.Share, req.UpdateMask)
	if err != nil {
		if errors.Is(err, share.ErrShareNotFound) {
			return &ocm.UpdateReceivedOCMShareResponse{
				Status: status.NewNotFound(ctx, "share does not exist"),
			}, nil
		}
		return &ocm.UpdateReceivedOCMShareResponse{
			Status: status.NewInternal(ctx, "error updating received share"),
		}, nil
	}

	res := &ocm.UpdateReceivedOCMShareResponse{
		Status: status.NewOK(ctx),
	}
	return res, nil
}

func (s *service) GetReceivedOCMShare(ctx context.Context, req *ocm.GetReceivedOCMShareRequest) (*ocm.GetReceivedOCMShareResponse, error) {
	user := ctxpkg.ContextMustGetUser(ctx)
	if user.Id.GetType() == userpb.UserType_USER_TYPE_SERVICE {
		var uid userpb.UserId
		_ = utils.ReadJSONFromOpaque(req.Opaque, "userid", &uid)
		user = &userpb.User{
			Id: &uid,
		}
	}

	ocmshare, err := s.repo.GetReceivedShare(ctx, user, req.Ref)
	if err != nil {
		if errors.Is(err, share.ErrShareNotFound) {
			return &ocm.GetReceivedOCMShareResponse{
				Status: status.NewNotFound(ctx, "share does not exist"),
			}, nil
		}
		return &ocm.GetReceivedOCMShareResponse{
			Status: status.NewInternal(ctx, "error getting received share: "+err.Error()),
		}, nil
	}

	res := &ocm.GetReceivedOCMShareResponse{
		Status: status.NewOK(ctx),
		Share:  ocmshare,
	}
	return res, nil
}
