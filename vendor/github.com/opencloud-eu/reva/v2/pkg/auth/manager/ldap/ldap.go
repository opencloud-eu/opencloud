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

package ldap

import (
	"context"
	"fmt"
	"strconv"

	authpb "github.com/cs3org/go-cs3apis/cs3/auth/provider/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/auth"
	"github.com/opencloud-eu/reva/v2/pkg/auth/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/auth/scope"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	ldapIdentity "github.com/opencloud-eu/reva/v2/pkg/utils/ldap"
	"github.com/pkg/errors"
)

func init() {
	registry.Register("ldap", New)
}

type mgr struct {
	c          *config
	ldapClient ldap.Client
}

type config struct {
	utils.LDAPConn  `mapstructure:",squash"`
	LDAPIdentity    ldapIdentity.Identity `mapstructure:",squash"`
	Idp             string                `mapstructure:"idp"`
	GatewaySvc      string                `mapstructure:"gatewaysvc"`
	Nobody          int64                 `mapstructure:"nobody"`
	LoginAttributes []string              `mapstructure:"login_attributes"`
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := &config{
		LDAPIdentity:    ldapIdentity.New(),
		LoginAttributes: []string{"cn"},
	}
	if err := mapstructure.Decode(m, c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}
	return c, nil
}

// New returns an auth manager implementation that connects to a LDAP server to validate the user.
func New(m map[string]interface{}) (auth.Manager, error) {
	manager := &mgr{}
	err := manager.Configure(m)
	if err != nil {
		return nil, err
	}
	manager.ldapClient, err = utils.GetLDAPClientWithReconnect(&manager.c.LDAPConn)
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

	if c.Nobody == 0 {
		c.Nobody = 99
	}

	if err = c.LDAPIdentity.Setup(); err != nil {
		return fmt.Errorf("error setting up Identity config: %w", err)
	}
	c.GatewaySvc = sharedconf.GetGatewaySVC(c.GatewaySvc)
	am.c = c
	return nil
}

func (am *mgr) Authenticate(ctx context.Context, clientID, clientSecret string) (*user.User, map[string]*authpb.Scope, error) {
	log := appctx.GetLogger(ctx)

	filter := am.getLoginFilter(clientID)

	userEntry, err := am.c.LDAPIdentity.GetLDAPUserByFilter(ctx, am.ldapClient, filter)

	if err != nil {
		return nil, nil, err
	}

	// Bind as the user to verify their password
	la, err := utils.GetLDAPClientForAuth(ctx, &am.c.LDAPConn)
	if err != nil {
		return nil, nil, err
	}
	defer la.Close()
	err = la.Bind(userEntry.DN, clientSecret)
	switch {
	case err == nil:
		break
	case ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials):
		return nil, nil, errtypes.InvalidCredentials(clientID)
	default:
		log.Debug().Err(err).Interface("userdn", userEntry.DN).Msg("bind with user credentials failed")
		return nil, nil, err
	}

	var uid string
	if am.c.LDAPIdentity.User.Schema.IDIsOctetString {
		rawValue := userEntry.GetEqualFoldRawAttributeValue(am.c.LDAPIdentity.User.Schema.ID)
		if value, err := uuid.FromBytes(rawValue); err == nil {
			uid = value.String()
		}
	} else {
		uid = userEntry.GetEqualFoldAttributeValue(am.c.LDAPIdentity.User.Schema.ID)
	}

	userID := &user.UserId{
		Idp:      am.c.Idp,
		OpaqueId: uid,
		Type:     am.c.LDAPIdentity.GetUserType(userEntry),
	}
	gwc, err := pool.GetGatewayServiceClient(am.c.GatewaySvc)
	if err != nil {
		return nil, nil, errors.Wrap(err, "ldap: error getting gateway grpc client")
	}
	getGroupsResp, err := gwc.GetUserGroups(ctx, &user.GetUserGroupsRequest{
		UserId: userID,
	})
	if err != nil {
		log.Warn().Err(err).Msg("error getting user groups")
		return nil, nil, errors.Wrap(err, "ldap: error getting user groups")
	}
	if getGroupsResp.Status.Code != rpc.Code_CODE_OK {
		log.Warn().Err(err).Str("msg", getGroupsResp.Status.Message).Msg("grpc getting user groups failed")
		return nil, nil, fmt.Errorf("ldap: grpc getting user groups failed: '%s'", getGroupsResp.Status.Message)
	}
	gidNumber := am.c.Nobody
	gidValue := userEntry.GetEqualFoldAttributeValue(am.c.LDAPIdentity.User.Schema.GIDNumber)
	if gidValue != "" {
		gidNumber, err = strconv.ParseInt(gidValue, 10, 64)
		if err != nil {
			return nil, nil, err
		}
	}
	uidNumber := am.c.Nobody
	uidValue := userEntry.GetEqualFoldAttributeValue(am.c.LDAPIdentity.User.Schema.UIDNumber)
	if uidValue != "" {
		uidNumber, err = strconv.ParseInt(uidValue, 10, 64)
		if err != nil {
			return nil, nil, err
		}
	}
	u := &user.User{
		Id: userID,
		// TODO add more claims from the StandardClaims, eg EmailVerified
		Username: userEntry.GetEqualFoldAttributeValue(am.c.LDAPIdentity.User.Schema.Username),
		// TODO groups
		Groups:      getGroupsResp.Groups,
		Mail:        userEntry.GetEqualFoldAttributeValue(am.c.LDAPIdentity.User.Schema.Mail),
		DisplayName: userEntry.GetEqualFoldAttributeValue(am.c.LDAPIdentity.User.Schema.DisplayName),
		UidNumber:   uidNumber,
		GidNumber:   gidNumber,
	}

	var scopes map[string]*authpb.Scope
	if userID != nil && userID.Type == user.UserType_USER_TYPE_LIGHTWEIGHT {
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

	log.Debug().Interface("entry", userEntry).Interface("user", u).Msg("authenticated user")

	return u, scopes, nil
}

func (am *mgr) getLoginFilter(login string) string {
	var filter string
	for _, attr := range am.c.LoginAttributes {
		filter = fmt.Sprintf("%s(%s=%s)", filter, attr, ldap.EscapeFilter(login))
	}

	return fmt.Sprintf("(&%s(objectclass=%s)(|%s))",
		am.c.LDAPIdentity.User.Filter,
		am.c.LDAPIdentity.User.Objectclass,
		filter,
	)
}
