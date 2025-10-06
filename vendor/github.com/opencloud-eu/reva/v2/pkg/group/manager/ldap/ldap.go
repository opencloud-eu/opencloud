// Copyright 2018-2020 CERN
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

	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/group"
	"github.com/opencloud-eu/reva/v2/pkg/group/manager/registry"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	ldapIdentity "github.com/opencloud-eu/reva/v2/pkg/utils/ldap"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
)

func init() {
	registry.Register("ldap", New)
}

type manager struct {
	c          *config
	ldapClient ldap.Client
}

type config struct {
	utils.LDAPConn `mapstructure:",squash"`
	LDAPIdentity   ldapIdentity.Identity `mapstructure:",squash"`
	Idp            string                `mapstructure:"idp"`
	// Nobody specifies the fallback gid number for groups that don't have a gidNumber set in LDAP
	Nobody int64 `mapstructure:"nobody"`
}

const tracerName = "pkg/group/manager/ldap"

func parseConfig(m map[string]interface{}) (*config, error) {
	c := config{
		LDAPIdentity: ldapIdentity.New(),
	}
	if err := mapstructure.Decode(m, &c); err != nil {
		err = errors.Wrap(err, "error decoding conf")
		return nil, err
	}

	return &c, nil
}

// New returns a group manager implementation that connects to a LDAP server to provide group metadata.
func New(m map[string]interface{}) (group.Manager, error) {
	if sharedconf.MultiTenantEnabled() {
		return nil, errtypes.NotSupported("ldap group manager does not support multi-tenancy")
	}

	mgr := &manager{}
	err := mgr.Configure(m)
	if err != nil {
		return nil, err
	}

	mgr.ldapClient, err = utils.GetLDAPClientWithReconnect(&mgr.c.LDAPConn)
	if err != nil {
		return nil, err
	}
	return mgr, nil
}

// Configure initializes the configuration of the group manager from the supplied config map
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

// GetGroup implements the group.Manager interface. Looks up a group by Id and return the group
func (m *manager) GetGroup(ctx context.Context, gid *grouppb.GroupId, skipFetchingMembers bool) (*grouppb.Group, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GetGroup")
	defer span.End()

	span.SetAttributes(
		attribute.Stringer("group_id", gid),
		attribute.Bool("skip_fetching_members", skipFetchingMembers),
	)

	log := appctx.GetLogger(ctx)
	if gid.Idp != "" && gid.Idp != m.c.Idp {
		return nil, errtypes.NotFound("idp mismatch")
	}

	groupEntry, err := m.c.LDAPIdentity.GetLDAPGroupByID(ctx, m.ldapClient, gid.OpaqueId)
	if err != nil {
		return nil, err
	}

	log.Debug().Interface("entry", groupEntry).Msg("entries")

	g, err := m.ldapEntryToGroup(groupEntry)
	if err != nil {
		return nil, err
	}

	if skipFetchingMembers {
		return g, nil
	}

	members, err := m.c.LDAPIdentity.GetLDAPGroupMembers(ctx, m.ldapClient, groupEntry)
	if err != nil {
		return nil, err
	}

	memberIDs := make([]*userpb.UserId, 0, len(members))
	for _, member := range members {
		userid, err := m.ldapEntryToUserID(member)
		if err != nil {
			log.Warn().Err(err).Interface("member", member).Msg("Failed convert member entry to userid")
			continue
		}
		memberIDs = append(memberIDs, userid)
	}

	g.Members = memberIDs

	return g, nil
}

// GetGroupByClaim implements the group.Manager interface. Looks up a group by
// claim ('group_name', 'group_id', 'display_name') and returns the group.
func (m *manager) GetGroupByClaim(ctx context.Context, claim, value string, skipFetchingMembers bool) (*grouppb.Group, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GetGroupByClaim")
	defer span.End()

	span.SetAttributes(
		attribute.String("claim", claim),
		attribute.String("value", value),
		attribute.Bool("skip_fetching_members", skipFetchingMembers),
	)

	log := appctx.GetLogger(ctx)
	groupEntry, err := m.c.LDAPIdentity.GetLDAPGroupByAttribute(ctx, m.ldapClient, claim, value)
	if err != nil {
		log.Debug().Err(err).Msg("GetGroupByClaim")
		return nil, err
	}

	log.Debug().Interface("entry", groupEntry).Msg("entries")

	g, err := m.ldapEntryToGroup(groupEntry)
	if err != nil {
		return nil, err
	}

	if skipFetchingMembers {
		return g, nil
	}

	members, err := m.c.LDAPIdentity.GetLDAPGroupMembers(ctx, m.ldapClient, groupEntry)
	if err != nil {
		return nil, err
	}

	memberIDs := make([]*userpb.UserId, 0, len(members))
	for _, member := range members {
		userid, err := m.ldapEntryToUserID(member)
		if err != nil {
			log.Warn().Err(err).Interface("member", member).Msg("Failed convert member entry to userid")
			continue
		}
		memberIDs = append(memberIDs, userid)
	}

	g.Members = memberIDs

	return g, nil
}

// FindGroups implements the group.Manager interface. Searches for groups using
// a prefix-substring search on the group attributes ('group_name',
// 'display_name', 'group_id') and returns the groups. FindGroups does NOT expand the
// members of the Groups.
func (m *manager) FindGroups(ctx context.Context, query string, skipFetchingMembers bool) ([]*grouppb.Group, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "FindGroups")
	defer span.End()

	span.SetAttributes(
		attribute.String("query", query),
		attribute.Bool("skip_fetching_members", skipFetchingMembers),
	)

	entries, err := m.c.LDAPIdentity.GetLDAPGroups(ctx, m.ldapClient, query)
	if err != nil {
		return nil, err
	}

	groups := make([]*grouppb.Group, 0, len(entries))

	for _, entry := range entries {
		g, err := m.ldapEntryToGroup(entry)
		if err != nil {
			return nil, err
		}

		groups = append(groups, g)
	}

	return groups, nil
}

// GetMembers implements the group.Manager interface. It returns all the userids of the members
// of the group identified by the supplied id.
func (m *manager) GetMembers(ctx context.Context, gid *grouppb.GroupId) ([]*userpb.UserId, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GetMembers")
	defer span.End()

	span.SetAttributes(
		attribute.Stringer("group_id", gid),
	)
	log := appctx.GetLogger(ctx)
	if gid.Idp != "" && gid.Idp != m.c.Idp {
		return nil, errtypes.NotFound("idp mismatch")
	}

	groupEntry, err := m.c.LDAPIdentity.GetLDAPGroupByID(ctx, m.ldapClient, gid.OpaqueId)
	if err != nil {
		return nil, err
	}

	log.Debug().Interface("entry", groupEntry).Msg("entries")

	members, err := m.c.LDAPIdentity.GetLDAPGroupMembers(ctx, m.ldapClient, groupEntry)
	if err != nil {
		return nil, err
	}

	memberIDs := make([]*userpb.UserId, 0, len(members))
	for _, member := range members {
		userid, err := m.ldapEntryToUserID(member)
		if err != nil {
			log.Warn().Err(err).Interface("member", member).Msg("Failed convert member entry to userid")
			continue
		}
		memberIDs = append(memberIDs, userid)
	}

	return memberIDs, nil
}

// HasMember implements the group.Member interface. Checks whether the supplied userid is a member
// of the supplied groupid.
func (m *manager) HasMember(ctx context.Context, gid *grouppb.GroupId, uid *userpb.UserId) (bool, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "HasMember")
	defer span.End()

	span.SetAttributes(
		attribute.Stringer("group_id", gid),
		attribute.Stringer("user_id", uid),
	)
	// It might be possible to do a somewhat more clever LDAP search here. (First lookup the user and then
	// search for (&(objectclass=<groupoc>)(<groupid>=gid)(member=<username/userdn>)
	// The GetMembers call used below can be quiet ineffecient for large groups
	members, err := m.GetMembers(ctx, gid)
	if err != nil {
		return false, err
	}

	for _, u := range members {
		if u.OpaqueId == uid.OpaqueId && u.Idp == uid.Idp {
			return true, nil
		}
	}
	return false, nil
}

func (m *manager) ldapEntryToGroup(entry *ldap.Entry) (*grouppb.Group, error) {
	id, err := m.ldapEntryToGroupID(entry)
	if err != nil {
		return nil, err
	}

	gidNumber := m.c.Nobody
	gidValue := entry.GetEqualFoldAttributeValue(m.c.LDAPIdentity.Group.Schema.GIDNumber)
	if gidValue != "" {
		gidNumber, err = strconv.ParseInt(gidValue, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	g := &grouppb.Group{
		Id:          id,
		GroupName:   entry.GetEqualFoldAttributeValue(m.c.LDAPIdentity.Group.Schema.Groupname),
		Mail:        entry.GetEqualFoldAttributeValue(m.c.LDAPIdentity.Group.Schema.Mail),
		DisplayName: entry.GetEqualFoldAttributeValue(m.c.LDAPIdentity.Group.Schema.DisplayName),
		GidNumber:   gidNumber,
	}

	return g, nil
}

func (m *manager) ldapEntryToGroupID(entry *ldap.Entry) (*grouppb.GroupId, error) {
	var id string
	if m.c.LDAPIdentity.Group.Schema.IDIsOctetString {
		rawValue := entry.GetEqualFoldRawAttributeValue(m.c.LDAPIdentity.Group.Schema.ID)
		if value, err := uuid.FromBytes(rawValue); err == nil {
			id = value.String()
		} else {
			return nil, err
		}
	} else {
		id = entry.GetEqualFoldAttributeValue(m.c.LDAPIdentity.Group.Schema.ID)
	}

	return &grouppb.GroupId{
		Idp:      m.c.Idp,
		OpaqueId: id,
	}, nil
}

func (m *manager) ldapEntryToUserID(entry *ldap.Entry) (*userpb.UserId, error) {
	var uid string
	if m.c.LDAPIdentity.User.Schema.IDIsOctetString {
		rawValue := entry.GetEqualFoldRawAttributeValue(m.c.LDAPIdentity.User.Schema.ID)
		var value uuid.UUID
		var err error
		if value, err = uuid.FromBytes(rawValue); err != nil {
			return nil, err
		}
		uid = value.String()
	} else {
		uid = entry.GetEqualFoldAttributeValue(m.c.LDAPIdentity.User.Schema.ID)
	}

	return &userpb.UserId{
		Idp:      m.c.Idp,
		OpaqueId: uid,
		Type:     userpb.UserType_USER_TYPE_PRIMARY,
	}, nil
}
