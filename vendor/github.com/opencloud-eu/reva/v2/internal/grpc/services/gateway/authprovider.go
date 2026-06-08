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

package gateway

import (
	"context"
	"fmt"

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	registry "github.com/cs3org/go-cs3apis/cs3/auth/registry/v1beta1"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func (s *svc) Authenticate(ctx context.Context, req *gateway.AuthenticateRequest) (*gateway.AuthenticateResponse, error) {
	log := appctx.GetLogger(ctx)

	// find auth provider
	c, err := s.findAuthProvider(ctx, req.Type)
	if err != nil {
		log.Err(err).Str("type", req.Type).Msg("error getting auth provider client")
		return &gateway.AuthenticateResponse{
			Status: status.NewInternal(ctx, "error getting auth provider client"),
		}, nil
	}

	authProviderReq := &authpb.AuthenticateRequest{
		ClientId:     req.ClientId,
		ClientSecret: req.ClientSecret,
	}
	res, err := c.Authenticate(ctx, authProviderReq)
	switch {
	case err != nil:
		return &gateway.AuthenticateResponse{
			Status: status.NewInternal(ctx, fmt.Sprintf("gateway: error calling Authenticate for type: %s", req.Type)),
		}, nil
	case res.Status.Code == rpc.Code_CODE_PERMISSION_DENIED:
		fallthrough
	case res.Status.Code == rpc.Code_CODE_UNAUTHENTICATED:
		fallthrough
	case res.Status.Code == rpc.Code_CODE_NOT_FOUND:
		// normal failures, no need to log
		return &gateway.AuthenticateResponse{
			Status: res.Status,
		}, nil
	case res.Status.Code != rpc.Code_CODE_OK:
		return &gateway.AuthenticateResponse{
			Status: status.NewInternal(ctx, fmt.Sprintf("error authenticating credentials to auth provider for type: %s", req.Type)),
		}, nil
	}

	// validate valid userId
	if res.User == nil {
		err := errtypes.NotFound("gateway: user after Authenticate is nil")
		log.Err(err).Msg("user is nil")
		return &gateway.AuthenticateResponse{
			Status: status.NewInternal(ctx, "user is nil"),
		}, nil
	}

	if res.User.Id == nil {
		err := errtypes.NotFound("gateway: uid after Authenticate is nil")
		log.Err(err).Msg("user id is nil")
		return &gateway.AuthenticateResponse{
			Status: status.NewInternal(ctx, "user id is nil"),
		}, nil
	}

	u := proto.Clone(res.User).(*userpb.User)
	if sharedconf.SkipUserGroupsInToken() {
		u.Groups = []string{}
	}

	token, err := s.tokenmgr.MintToken(ctx, u, res.TokenScope)
	if err != nil {
		err = errors.Wrap(err, "authsvc: error in MintToken")
		res := &gateway.AuthenticateResponse{
			Status: status.NewUnauthenticated(ctx, err, "error creating access token"),
		}
		return res, nil
	}

	if scope, ok := res.TokenScope["user"]; s.c.DisableHomeCreationOnLogin || !ok || scope.Role != authpb.Role_ROLE_OWNER || res.User.Id.Type == userpb.UserType_USER_TYPE_FEDERATED {
		gwRes := &gateway.AuthenticateResponse{
			Status: status.NewOK(ctx),
			User:   res.User,
			Token:  token,
		}
		return gwRes, nil
	}

	// we need to pass the token to authenticate the CreateHome request.
	// TODO(labkode): appending to existing context will not pass the token.
	ctx = ctxpkg.ContextSetToken(ctx, token)
	ctx = ctxpkg.ContextSetUser(ctx, res.User)
	ctx = metadata.AppendToOutgoingContext(ctx, ctxpkg.TokenHeader, token) // TODO(jfd): hardcoded metadata key. use  PerRPCCredentials?

	// create home directory
	createHomeRes, err := s.CreateHome(ctx, &storageprovider.CreateHomeRequest{})
	if err != nil {
		log.Err(err).Msg("error calling CreateHome")
		return &gateway.AuthenticateResponse{
			Status: status.NewInternal(ctx, "error creating user home"),
		}, nil
	}

	if createHomeRes.Status.Code != rpc.Code_CODE_OK && createHomeRes.Status.Code != rpc.Code_CODE_ALREADY_EXISTS {
		err := status.NewErrorFromCode(createHomeRes.Status.Code, "gateway")
		log.Err(err).Msg("error calling Createhome")
		return &gateway.AuthenticateResponse{
			Status: status.NewInternal(ctx, "error creating user home"),
		}, nil
	}

	gwRes := &gateway.AuthenticateResponse{
		Status: status.NewOK(ctx),
		User:   res.User,
		Token:  token,
	}
	return gwRes, nil
}

func (s *svc) WhoAmI(ctx context.Context, req *gateway.WhoAmIRequest) (*gateway.WhoAmIResponse, error) {
	u, _, err := s.tokenmgr.DismantleToken(ctx, req.Token)
	if err != nil {
		err = errors.Wrap(err, "gateway: error getting user from token")
		return &gateway.WhoAmIResponse{
			Status: status.NewUnauthenticated(ctx, err, "error dismantling token"),
		}, nil
	}

	if sharedconf.SkipUserGroupsInToken() {
		groupsRes, err := s.GetUserGroups(ctx, &userpb.GetUserGroupsRequest{UserId: u.Id})
		if err != nil {
			return nil, err
		}
		u.Groups = groupsRes.Groups
	}

	res := &gateway.WhoAmIResponse{
		Status: status.NewOK(ctx),
		User:   u,
	}
	return res, nil
}

func (s *svc) findAuthProvider(ctx context.Context, authType string) (authpb.ProviderAPIClient, error) {
	c, err := pool.GetAuthRegistryServiceClient(s.c.AuthRegistryEndpoint)
	if err != nil {
		err = errors.Wrap(err, "gateway: error getting auth registry client")
		return nil, err
	}

	res, err := c.GetAuthProviders(ctx, &registry.GetAuthProvidersRequest{
		Type: authType,
	})

	if err != nil {
		err = errors.Wrap(err, "gateway: error calling GetAuthProvider")
		return nil, err
	}

	if res.Status.Code == rpc.Code_CODE_OK && res.Providers != nil && len(res.Providers) > 0 {
		// TODO(labkode): check for capabilities here
		c, err := pool.GetAuthProviderServiceClient(res.Providers[0].Address)
		if err != nil {
			err = errors.Wrap(err, "gateway: error getting an auth provider client")
			return nil, err
		}

		return c, nil
	}

	if res.Status.Code == rpc.Code_CODE_NOT_FOUND {
		return nil, errtypes.NotFound("gateway: auth provider not found for type:" + authType)
	}

	return nil, errtypes.InternalError("gateway: error finding an auth provider for type: " + authType)
}
