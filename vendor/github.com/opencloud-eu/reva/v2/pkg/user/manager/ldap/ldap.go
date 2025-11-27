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
	"strings"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/user"
	"github.com/opencloud-eu/reva/v2/pkg/user/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	ldapIdentity "github.com/opencloud-eu/reva/v2/pkg/utils/ldap"
	"go.opentelemetry.io/otel/attribute"
)

func init() {
	registry.Register("ldap", New)
}

const tracerName = "pkg/user/manager/ldap"

type manager struct {
	c          *config
	ldapClient ldap.Client
}

type config struct {
	utils.LDAPConn `mapstructure:",squash"`
	LDAPIdentity   ldapIdentity.Identity `mapstructure:",squash"`
	Idp            string                `mapstructure:"idp"`
	// Nobody specifies the fallback uid number for users that don't have a uidNumber set in LDAP
	Nobody int64 `mapstructure:"nobody"`
}

func parseConfig(m map[string]interface{}) (*config, error) {
	c := config{
		LDAPIdentity: ldapIdentity.New(),
	}
	if err := mapstructure.Decode(m, &c); err != nil {
		return nil, fmt.Errorf("error decoding conf: %w", err)
	}

	return &c, nil
}

// New returns a user manager implementation that connects to a LDAP server to provide user metadata.
func New(m map[string]interface{}) (user.Manager, error) {
	mgr := &manager{}
	err := mgr.Configure(m)
	if err != nil {
		return nil, err
	}

	mgr.ldapClient, err = utils.GetLDAPClientWithReconnect(&mgr.c.LDAPConn)
	return mgr, err
}

// Configure initializes the configuration of the user manager from the supplied config map
func (m *manager) Configure(ml map[string]interface{}) error {
	c, err := parseConfig(ml)
	if err != nil {
		return err
	}
	if c.Nobody == 0 {
		c.Nobody = 99
	}

	if err = c.LDAPIdentity.Setup(); err != nil {
		return fmt.Errorf("error setting up Identity config: %w", err)
	}
	m.c = c
	return nil
}

// GetUser implements the user.Manager interface. Looks up a user by Id and return the user
func (m *manager) GetUser(ctx context.Context, uid *userpb.UserId, skipFetchingGroups bool) (*userpb.User, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GetUser")
	defer span.End()

	span.SetAttributes(
		attribute.Stringer("parameter.userid", uid),
		attribute.Bool("parameter.skipFetchingGroups", skipFetchingGroups),
	)

	log := appctx.GetLogger(ctx)
	log.Debug().Interface("id", uid).Msg("GetUser")
	// If the Idp value in the uid does not match our config, we can't answer this request
	if uid.Idp != "" && uid.Idp != m.c.Idp {
		return nil, errtypes.NotFound("idp mismatch")
	}

	userEntry, err := m.c.LDAPIdentity.GetLDAPUserByID(ctx, m.ldapClient, uid)
	if err != nil {
		return nil, err
	}

	log.Debug().Interface("entry", userEntry).Msg("entries")

	u, err := m.ldapEntryToUser(userEntry)
	if err != nil {
		return nil, err
	}

	if skipFetchingGroups {
		return u, nil
	}

	groups, err := m.c.LDAPIdentity.GetLDAPUserGroups(ctx, m.ldapClient, userEntry)
	if err != nil {
		return nil, err
	}

	u.Groups = groups
	return u, nil
}

// GetUserByClaim implements the user.Manager interface. Looks up a user by
// claim ('mail', 'username', 'userid') and returns the user.
func (m *manager) GetUserByClaim(ctx context.Context, claim, value, tenantID string, skipFetchingGroups bool) (*userpb.User, error) {
	log := appctx.GetLogger(ctx)
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GetUserByClaim")
	defer span.End()

	span.SetAttributes(
		attribute.String("parameter.claim", claim),
		attribute.String("paramter.value", value),
		attribute.Bool("parameter.skipFetchingGroups", skipFetchingGroups),
	)
	log.Debug().Str("claim", claim).Str("value", value).Msg("GetUserByClaim")
	userEntry, err := m.c.LDAPIdentity.GetLDAPUserByAttribute(ctx, m.ldapClient, claim, value, tenantID)
	if err != nil {
		log.Debug().Err(err).Msg("GetUserByClaim")
		return nil, err
	}

	log.Debug().Interface("entry", userEntry).Msg("entries")

	u, err := m.ldapEntryToUser(userEntry)
	if err != nil {
		return nil, err
	}

	if m.c.LDAPIdentity.IsLDAPUserInDisabledGroup(ctx, m.ldapClient, userEntry) {
		return nil, errtypes.NotFound("user is locally disabled")
	}

	if skipFetchingGroups {
		return u, nil
	}

	groups, err := m.c.LDAPIdentity.GetLDAPUserGroups(ctx, m.ldapClient, userEntry)
	if err != nil {
		return nil, err
	}

	u.Groups = groups

	return u, nil
}

// FindUser implements the user.Manager interface. Searches for users using a prefix-substring search on
// the user attributes ('mail', 'username', 'displayname', 'userid') and returns the users.
func (m *manager) FindUsers(ctx context.Context, query, tenantID string, skipFetchingGroups bool) ([]*userpb.User, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "FindUsers")
	defer span.End()
	span.SetAttributes(
		attribute.String("parameter.query", query),
		attribute.String("parameter.tenantID", tenantID),
		attribute.Bool("parameter.skipFetchingGroups", skipFetchingGroups),
	)

	entries, err := m.c.LDAPIdentity.GetLDAPUsers(ctx, m.ldapClient, query, tenantID)
	if err != nil {
		return nil, err
	}
	users := []*userpb.User{}

	for _, entry := range entries {
		u, err := m.ldapEntryToUser(entry)
		if err != nil {
			return nil, err
		}

		if !skipFetchingGroups {
			groups, err := m.c.LDAPIdentity.GetLDAPUserGroups(ctx, m.ldapClient, entry)
			if err != nil {
				return nil, err
			}
			u.Groups = groups
		}

		users = append(users, u)
	}

	return users, nil
}

// GetUserGroups implements the user.Manager interface. Looks up all group membership of
// the user with the supplied Id. Returns a string slice with the group ids
func (m *manager) GetUserGroups(ctx context.Context, uid *userpb.UserId) ([]string, error) {
	log := appctx.GetLogger(ctx)
	if uid.Idp != "" && uid.Idp != m.c.Idp {
		log.Debug().Str("useridp", uid.Idp).Str("configured idp", m.c.Idp).Msg("IDP mismatch")
		return nil, errtypes.NotFound("idp mismatch")
	}
	userEntry, err := m.c.LDAPIdentity.GetLDAPUserByID(ctx, m.ldapClient, uid)
	if err != nil {
		log.Debug().Err(err).Interface("userid", uid).Msg("Failed to lookup user")
		return []string{}, err
	}
	return m.c.LDAPIdentity.GetLDAPUserGroups(ctx, m.ldapClient, userEntry)
}

func (m *manager) ldapEntryToUser(entry *ldap.Entry) (*userpb.User, error) {
	id, err := m.ldapEntryToUserID(entry)
	if err != nil {
		return nil, err
	}

	gidNumber := m.c.Nobody
	gidValue := entry.GetEqualFoldAttributeValue(m.c.LDAPIdentity.User.Schema.GIDNumber)
	if gidValue != "" {
		gidNumber, err = strconv.ParseInt(gidValue, 10, 64)
		if err != nil {
			return nil, err
		}
	}
	uidNumber := m.c.Nobody
	uidValue := entry.GetEqualFoldAttributeValue(m.c.LDAPIdentity.User.Schema.UIDNumber)
	if uidValue != "" {
		uidNumber, err = strconv.ParseInt(uidValue, 10, 64)
		if err != nil {
			return nil, err
		}
	}
	u := &userpb.User{
		Id:          id,
		Username:    entry.GetEqualFoldAttributeValue(m.c.LDAPIdentity.User.Schema.Username),
		Mail:        entry.GetEqualFoldAttributeValue(m.c.LDAPIdentity.User.Schema.Mail),
		DisplayName: entry.GetEqualFoldAttributeValue(m.c.LDAPIdentity.User.Schema.DisplayName),
		GidNumber:   gidNumber,
		UidNumber:   uidNumber,
	}
	return u, nil
}

func (m *manager) ldapEntryToUserID(entry *ldap.Entry) (*userpb.UserId, error) {
	var uid string
	if m.c.LDAPIdentity.User.Schema.IDIsOctetString {
		attribute := m.c.LDAPIdentity.User.Schema.ID
		rawValue := entry.GetEqualFoldRawAttributeValue(attribute)
		if strings.EqualFold(attribute, "objectguid") {
			rawValue = ldapIdentity.SwapObjectGUIDBytes(rawValue)
		}
		if value, err := uuid.FromBytes(rawValue); err == nil {
			uid = value.String()
		} else {
			return nil, err
		}
	} else {
		uid = entry.GetEqualFoldAttributeValue(m.c.LDAPIdentity.User.Schema.ID)
	}

	return &userpb.UserId{
		Idp:      m.c.Idp,
		OpaqueId: uid,
		TenantId: entry.GetEqualFoldAttributeValue(m.c.LDAPIdentity.User.Schema.TenantID),
		Type:     m.c.LDAPIdentity.GetUserType(entry),
	}, nil
}
