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

// Package oidc  verifies an OIDC token against the configured OIDC provider
// and obtains the necessary claims to obtain user information.
package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/juliangruber/go-intersect"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
	"github.com/opencloud-eu/reva/v2/pkg/auth/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/rhttp"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

func init() {
	registry.Register("oidc", New)
}

type mgr struct {
	provider         *oidc.Provider // cached on first request
	c                *config
	oidcUsersMapping map[string]*oidcUserMapping
}

type config struct {
	Insecure     bool   `mapstructure:"insecure" docs:"false;Whether to skip certificate checks when sending requests."`
	Issuer       string `mapstructure:"issuer" docs:";The issuer of the OIDC token."`
	IDClaim      string `mapstructure:"id_claim" docs:"sub;The claim containing the ID of the user."`
	UIDClaim     string `mapstructure:"uid_claim" docs:";The claim containing the UID of the user."`
	GIDClaim     string `mapstructure:"gid_claim" docs:";The claim containing the GID of the user."`
	GatewaySvc   string `mapstructure:"gatewaysvc" docs:";The endpoint at which the GRPC gateway is exposed."`
	UsersMapping string `mapstructure:"users_mapping" docs:"; The optional OIDC users mapping file path"`
	GroupClaim   string `mapstructure:"group_claim" docs:"; The group claim to be looked up to map the user (default to 'groups')."`
}

type oidcUserMapping struct {
	OIDCIssuer string `mapstructure:"oidc_issuer" json:"oidc_issuer"`
	OIDCGroup  string `mapstructure:"oidc_group" json:"oidc_group"`
	Username   string `mapstructure:"username" json:"username"`
}

func (c *config) init() {
	if c.IDClaim == "" {
		// sub is stable and defined as unique. the user manager needs to take care of the sub to user metadata lookup
		c.IDClaim = "sub"
	}
	if c.GroupClaim == "" {
		c.GroupClaim = "groups"
	}
	if c.UIDClaim == "" {
		c.UIDClaim = "uid"
	}
	if c.GIDClaim == "" {
		c.GIDClaim = "gid"
	}

	c.GatewaySvc = sharedconf.GetGatewaySVC(c.GatewaySvc)
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

// New returns an auth manager implementation that verifies the oidc token and obtains the user claims.
func New(m map[string]any, _ *zerolog.Logger) (auth.Manager, error) {
	manager := &mgr{}
	err := manager.Configure(m)
	if err != nil {
		return nil, err
	}
	return manager, nil
}

func (am *mgr) Configure(m map[string]interface{}) error {
	c, err := parseConfig(m)
	if err != nil {
		return err
	}
	c.init()
	am.c = c

	am.oidcUsersMapping = map[string]*oidcUserMapping{}
	if c.UsersMapping == "" {
		// no mapping defined, leave the map empty and move on
		return nil
	}

	f, err := os.ReadFile(c.UsersMapping)
	if err != nil {
		return fmt.Errorf("oidc: error reading the users mapping file: +%v", err)
	}
	oidcUsers := []*oidcUserMapping{}
	err = json.Unmarshal(f, &oidcUsers)
	if err != nil {
		return fmt.Errorf("oidc: error unmarshalling the users mapping file: +%v", err)
	}
	for _, u := range oidcUsers {
		if _, found := am.oidcUsersMapping[u.OIDCGroup]; found {
			return fmt.Errorf("oidc: mapping error, group \"%s\" is mapped to multiple users", u.OIDCGroup)
		}
		am.oidcUsersMapping[u.OIDCGroup] = u
	}

	return nil
}

// The clientID would be empty as we only need to validate the clientSecret variable
// which contains the access token that we can use to contact the UserInfo endpoint
// and get the user claims.
func (am *mgr) Authenticate(ctx context.Context, clientID, clientSecret string) (*user.User, map[string]*authpb.Scope, error) {
	ctx = am.getOAuthCtx(ctx)
	log := appctx.GetLogger(ctx)

	oidcProvider, err := am.getOIDCProvider(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("oidc: error creating oidc provider: +%v", err)
	}

	oauth2Token := &oauth2.Token{
		AccessToken: clientSecret,
	}

	// query the oidc provider for user info
	userInfo, err := oidcProvider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		return nil, nil, fmt.Errorf("oidc: error getting userinfo: +%v", err)
	}

	// claims contains the standard OIDC claims like iss, iat, aud, ... and any other non-standard one.
	// TODO(labkode): make claims configuration dynamic from the config file so we can add arbitrary mappings from claims to user struct.
	// For now, only the group claim is dynamic.
	// TODO(labkode): may do like K8s does it: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/plugin/pkg/authenticator/token/oidc/oidc.go
	var claims map[string]interface{}
	if err := userInfo.Claims(&claims); err != nil {
		return nil, nil, fmt.Errorf("oidc: error unmarshaling userinfo claims: %v", err)
	}

	log.Debug().Interface("claims", claims).Interface("userInfo", userInfo).Msg("unmarshalled userinfo")

	if claims["iss"] == nil { // This is not set in simplesamlphp
		claims["iss"] = am.c.Issuer
	}
	if claims["email_verified"] == nil { // This is not set in simplesamlphp
		claims["email_verified"] = false
	}
	if claims["preferred_username"] == nil {
		claims["preferred_username"] = claims[am.c.IDClaim]
	}
	if claims["preferred_username"] == nil {
		claims["preferred_username"] = claims["email"]
	}
	if claims["name"] == nil {
		claims["name"] = claims[am.c.IDClaim]
	}
	if claims["name"] == nil {
		return nil, nil, fmt.Errorf("no \"name\" attribute found in userinfo: maybe the client did not request the oidc \"profile\"-scope")
	}
	if claims["email"] == nil {
		return nil, nil, fmt.Errorf("no \"email\" attribute found in userinfo: maybe the client did not request the oidc \"email\"-scope")
	}

	uid, _ := claims[am.c.UIDClaim].(float64)
	claims[am.c.UIDClaim] = int64(uid) // in case the uid claim is missing and a mapping is to be performed, resolveUser() will populate it
	// Note that if not, will silently carry a user with 0 uid, potentially problematic with storage providers
	gid, _ := claims[am.c.GIDClaim].(float64)
	claims[am.c.GIDClaim] = int64(gid)

	err = am.resolveUser(ctx, claims)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "oidc: error resolving username for external user '%v'", claims["email"])
	}

	userID := &user.UserId{
		OpaqueId: claims[am.c.IDClaim].(string), // a stable non reassignable id
		Idp:      claims["iss"].(string),        // in the scope of this issuer
		Type:     getUserType(claims[am.c.IDClaim].(string)),
	}

	gwc, err := pool.GetGatewayServiceClient(am.c.GatewaySvc)
	if err != nil {
		return nil, nil, errors.Wrap(err, "oidc: error getting gateway grpc client")
	}
	getGroupsResp, err := gwc.GetUserGroups(ctx, &user.GetUserGroupsRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "oidc: error getting user groups for '%+v'", userID)
	}
	if getGroupsResp.Status.Code != rpc.Code_CODE_OK {
		return nil, nil, status.NewErrorFromCode(getGroupsResp.Status.Code, "oidc")
	}

	u := &user.User{
		Id:           userID,
		Username:     claims["preferred_username"].(string),
		Groups:       getGroupsResp.Groups,
		Mail:         claims["email"].(string),
		MailVerified: claims["email_verified"].(bool),
		DisplayName:  claims["name"].(string),
		UidNumber:    claims[am.c.UIDClaim].(int64),
		GidNumber:    claims[am.c.GIDClaim].(int64),
	}

	var scopes map[string]*authpb.Scope
	if userID != nil && (userID.Type == user.UserType_USER_TYPE_LIGHTWEIGHT || userID.Type == user.UserType_USER_TYPE_FEDERATED) {
		scopes, err = scope.AddLightweightAccountScope(authpb.Role_ROLE_OWNER, nil)
		if err != nil {
			return nil, nil, err
		}
	} else {
		scopes, err = scope.AddOwnerScope(nil)
		if err != nil {
			return nil, nil, err
		}
	}

	return u, scopes, nil
}

func (am *mgr) getOAuthCtx(ctx context.Context) context.Context {
	// Sometimes for testing we need to skip the TLS check, that's why we need a
	// custom HTTP client.
	customHTTPClient := rhttp.GetHTTPClient(
		rhttp.Context(ctx),
		rhttp.Timeout(time.Second*10),
		rhttp.Insecure(am.c.Insecure),
		// Fixes connection fd leak which might be caused by provider-caching
		rhttp.DisableKeepAlive(true),
	)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, customHTTPClient)
	return ctx
}

// getOIDCProvider returns a singleton OIDC provider
func (am *mgr) getOIDCProvider(ctx context.Context) (*oidc.Provider, error) {
	ctx = am.getOAuthCtx(ctx)
	log := appctx.GetLogger(ctx)

	if am.provider != nil {
		return am.provider, nil
	}

	// Initialize a provider by specifying the issuer URL.
	// Once initialized this is a singleton that is reused for further requests.
	// The provider is responsible to verify the token sent by the client
	// against the security keys oftentimes available in the .well-known endpoint.
	provider, err := oidc.NewProvider(ctx, am.c.Issuer)

	if err != nil {
		log.Error().Err(err).Msg("oidc: error creating a new oidc provider")
		return nil, fmt.Errorf("oidc: error creating a new oidc provider: %+v", err)
	}

	am.provider = provider
	return am.provider, nil
}

func (am *mgr) resolveUser(ctx context.Context, claims map[string]interface{}) error {
	if len(am.oidcUsersMapping) > 0 {
		var username string

		// map and discover the user's username when a mapping is defined
		if claims[am.c.GroupClaim] == nil {
			// we are required to perform a user mapping but the group claim is not available
			return fmt.Errorf("no \"%s\" claim found in userinfo to map user", am.c.GroupClaim)
		}
		mappings := make([]string, 0, len(am.oidcUsersMapping))
		for _, m := range am.oidcUsersMapping {
			if m.OIDCIssuer == claims["iss"] {
				mappings = append(mappings, m.OIDCGroup)
			}
		}

		intersection := intersect.Simple(claims[am.c.GroupClaim], mappings)
		if len(intersection) > 1 {
			// multiple mappings are not implemented as we cannot decide which one to choose
			return errtypes.PermissionDenied("more than one user mapping entry exists for the given group claims")
		}
		if len(intersection) == 0 {
			return errtypes.PermissionDenied("no user mapping found for the given group claim(s)")
		}
		for _, m := range intersection {
			username = am.oidcUsersMapping[m.(string)].Username
		}

		upsc, err := pool.GetUserProviderServiceClient(am.c.GatewaySvc)
		if err != nil {
			return errors.Wrap(err, "error getting user provider grpc client")
		}
		getUserByClaimResp, err := upsc.GetUserByClaim(ctx, &user.GetUserByClaimRequest{
			Claim: "username",
			Value: username,
		})
		if err != nil {
			return errors.Wrapf(err, "error getting user by username '%v'", username)
		}
		if getUserByClaimResp.Status.Code != rpc.Code_CODE_OK {
			return status.NewErrorFromCode(getUserByClaimResp.Status.Code, "oidc")
		}

		// take the properties of the mapped target user to override the claims
		claims["preferred_username"] = username
		claims[am.c.IDClaim] = getUserByClaimResp.GetUser().GetId().OpaqueId
		claims["iss"] = getUserByClaimResp.GetUser().GetId().Idp
		claims[am.c.UIDClaim] = getUserByClaimResp.GetUser().UidNumber
		claims[am.c.GIDClaim] = getUserByClaimResp.GetUser().GidNumber
		appctx.GetLogger(ctx).Debug().Str("username", username).Interface("claims", claims).Msg("resolveUser: claims overridden from mapped user")
	}
	return nil
}

func getUserType(upn string) user.UserType {
	var t user.UserType
	switch {
	case strings.HasPrefix(upn, "guest"):
		t = user.UserType_USER_TYPE_LIGHTWEIGHT
	case strings.Contains(upn, "@"):
		t = user.UserType_USER_TYPE_FEDERATED
	default:
		t = user.UserType_USER_TYPE_PRIMARY
	}
	return t
}
