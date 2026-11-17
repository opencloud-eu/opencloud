// Copyright 2018-2022 CERN
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

package sharees

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/internal/http/services/owncloud/ocdav/spacelookup"
	"github.com/opencloud-eu/reva/v3/internal/http/services/owncloud/ocs/response"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/conversions"
	ctxpkg "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
	"google.golang.org/grpc/metadata"
)

// TokenInfo handles http requests regarding tokens
func (h *Handler) TokenInfo(protected bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		log := appctx.GetLogger(r.Context())
		tkn := path.Base(r.URL.Path)
		_, pw, _ := r.BasicAuth()

		c, err := pool.GetGatewayServiceClient(h.gatewayAddr)
		if err != nil {
			// endpoint public - don't exponse information
			log.Error().Err(err).Msg("error getting gateway client")
			response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "", nil)
			return
		}

		t, err := handleGetToken(r.Context(), tkn, pw, c, protected)
		if err != nil {
			// endpoint public - don't exponse information
			log.Error().Err(err).Msg("error while handling GET TokenInfo")
			response.WriteOCSError(w, r, response.MetaServerError.StatusCode, "", nil)
			return
		}

		response.WriteOCSSuccess(w, r, t)
	}
}

func handleGetToken(ctx context.Context, tkn string, pw string, c gateway.GatewayAPIClient, protected bool) (conversions.TokenInfo, error) {
	user, token, passwordProtected, err := getInfoForToken(tkn, pw, c)
	if err != nil {
		return conversions.TokenInfo{}, err
	}

	t, err := buildTokenInfo(user, tkn, token, passwordProtected, c)
	if err != nil {
		return t, err
	}

	if protected && !t.PasswordProtected {
		space, status, err := spacelookup.LookUpStorageSpaceByID(ctx, c, storagespace.FormatResourceID(&provider.ResourceId{StorageId: t.StorageID, SpaceId: t.SpaceID, OpaqueId: t.OpaqueID}))
		// add info only if user is able to stat
		if err == nil && status.Code == rpc.Code_CODE_OK {
			t.SpacePath = utils.ReadPlainFromOpaque(space.Opaque, "path")
			t.SpaceAlias = utils.ReadPlainFromOpaque(space.Opaque, "spaceAlias")
			t.SpaceURL = path.Join(t.SpaceAlias, t.OpaqueID, t.Path)
			t.SpaceType = space.SpaceType
		}

	}

	return t, nil
}

func buildTokenInfo(owner *user.User, tkn string, token string, passProtected bool, c gateway.GatewayAPIClient) (conversions.TokenInfo, error) {
	t := conversions.TokenInfo{Token: tkn, LinkURL: "/s/" + tkn}
	if passProtected {
		t.PasswordProtected = true
		return t, nil
	}

	ctx := ctxpkg.ContextSetToken(context.TODO(), token)
	ctx = ctxpkg.ContextSetUser(ctx, owner)
	ctx = metadata.AppendToOutgoingContext(ctx, ctxpkg.TokenHeader, token)

	sRes, err := getPublicShare(ctx, c, tkn)
	if err != nil || sRes.Status.Code != rpc.Code_CODE_OK {
		return t, fmt.Errorf("can't stat resource. %+v %s", sRes, err)
	}

	t.ID = storagespace.FormatResourceID(sRes.Share.GetResourceId())
	t.StorageID = sRes.Share.ResourceId.GetStorageId()
	t.SpaceID = sRes.Share.ResourceId.GetSpaceId()
	t.OpaqueID = sRes.Share.ResourceId.GetOpaqueId()

	role := conversions.RoleFromResourcePermissions(sRes.Share.Permissions.GetPermissions(), true)
	t.Aliaslink = role.OCSPermissions() == 0

	return t, nil
}

func getInfoForToken(tkn string, pw string, c gateway.GatewayAPIClient) (owner *user.User, token string, passwordProtected bool, err error) {
	ctx := context.Background()

	res, err := handleBasicAuth(ctx, c, tkn, pw)
	if err != nil {
		return
	}

	switch res.Status.Code {
	case rpc.Code_CODE_OK:
		// nothing to do
	case rpc.Code_CODE_PERMISSION_DENIED:
		if res.Status.Message != "wrong password" {
			err = errors.New("not found")
			return
		}

		passwordProtected = true
		return
	default:
		err = fmt.Errorf("authentication returned unsupported status code '%d'", res.Status.Code)
		return
	}

	return res.User, res.Token, false, nil
}

func handleBasicAuth(ctx context.Context, c gateway.GatewayAPIClient, token, pw string) (*gateway.AuthenticateResponse, error) {
	authenticateRequest := gateway.AuthenticateRequest{
		Type:         "publicshares",
		ClientId:     token,
		ClientSecret: "password|" + pw,
	}

	return c.Authenticate(ctx, &authenticateRequest)
}

func getPublicShare(ctx context.Context, client gateway.GatewayAPIClient, token string) (*link.GetPublicShareResponse, error) {
	return client.GetPublicShare(ctx, &link.GetPublicShareRequest{
		Ref: &link.PublicShareReference{
			Spec: &link.PublicShareReference_Token{
				Token: token,
			},
		}})
}
