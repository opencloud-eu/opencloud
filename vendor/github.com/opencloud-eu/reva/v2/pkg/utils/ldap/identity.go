// Copyright 2022 CERN
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
	"strings"

	identityUser "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/opencloud-eu/reva/v2/pkg/appctx"
	"github.com/opencloud-eu/reva/v2/pkg/errtypes"
	"github.com/opencloud-eu/reva/v2/pkg/sharedconf"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Identity provides methods to query users and groups from an LDAP server
type Identity struct {
	User  userConfig  `mapstructure:",squash"`
	Group groupConfig `mapstructure:",squash"`
}

const tracerName = "pkg/utils/ldap"

type userConfig struct {
	BaseDN              string `mapstructure:"user_base_dn"`
	Scope               string `mapstructure:"user_search_scope"`
	scopeVal            int
	Filter              string     `mapstructure:"user_filter"`
	Objectclass         string     `mapstructure:"user_objectclass"`
	DisableMechanism    string     `mapstructure:"user_disable_mechanism"`
	EnabledProperty     string     `mapstructure:"user_enabled_property"`
	UserTypeProperty    string     `mapstructure:"user_type_property"`
	Schema              userSchema `mapstructure:"user_schema"`
	SubstringFilterType string     `mapstructure:"user_substring_filter_type"`
	substringFilterVal  int
}

type groupConfig struct {
	BaseDN              string `mapstructure:"group_base_dn"`
	Scope               string `mapstructure:"group_search_scope"`
	scopeVal            int
	Filter              string      `mapstructure:"group_filter"`
	Objectclass         string      `mapstructure:"group_objectclass"`
	Schema              groupSchema `mapstructure:"group_schema"`
	SubstringFilterType string      `mapstructure:"group_substring_filter_type"`
	substringFilterVal  int
	// LocalDisabledDN contains the full DN of a group that contains disabled users.
	LocalDisabledDN string `mapstructure:"group_local_disabled_dn"`
}

type groupSchema struct {
	// GID is an immutable group id, see https://docs.microsoft.com/en-us/azure/active-directory/hybrid/plan-connect-design-concepts
	ID              string `mapstructure:"id"`
	IDIsOctetString bool   `mapstructure:"idIsOctetString"`
	// CN is the group name, typically `cn`, `gid` or `samaccountname`
	Groupname string `mapstructure:"groupName"`
	// Mail is the email address of a group
	Mail string `mapstructure:"mail"`
	// Displayname is the Human readable name, e.g. `Database Admins`
	DisplayName string `mapstructure:"displayName"`
	// GIDNumber is a numeric id that maps to a filesystem gid, eg. 654321
	GIDNumber string `mapstructure:"gidNumber"`
	Member    string `mapstructure:"member"`
}

type userSchema struct {
	// UID is an immutable user id, see https://docs.microsoft.com/en-us/azure/active-directory/hybrid/plan-connect-design-concepts
	ID string `mapstructure:"id"`
	// UIDIsOctetString set this to true i the values of the UID attribute are returned as OCTET STRING values (binary byte sequences)
	// by the Directory Service. This is e.g. the case for the 'objectGUID' and	'ms-DS-ConsistencyGuid' Attributes in AD
	IDIsOctetString bool `mapstructure:"idIsOctetString"`
	// Name is the username, typically `cn`, `uid` or `samaccountname`
	Username string `mapstructure:"userName"`
	// Mail is the email address of a user
	Mail string `mapstructure:"mail"`
	// Displayname is the Human readable name, e.g. `Albert Einstein`
	DisplayName string `mapstructure:"displayName"`
	// UIDNumber is a numeric id that maps to a filesystem uid, eg. 123546
	UIDNumber string `mapstructure:"uidNumber"`
	// GIDNumber is a numeric id that maps to a filesystem gid, eg. 654321
	GIDNumber string `mapstructure:"gidNumber"`
	// TenantID is the tenant id of the user, if applicable.
	TenantID string `mapstructure:"tenantId"`
}

// Default userConfig (somewhat inspired by Active Directory)
var userDefaults = userConfig{
	Scope:       "sub",
	Objectclass: "posixAccount",
	Schema: userSchema{
		ID:              "ms-DS-ConsistencyGuid",
		IDIsOctetString: false,
		Username:        "cn",
		Mail:            "mail",
		DisplayName:     "displayName",
		UIDNumber:       "uidNumber",
		GIDNumber:       "gidNumber",
	},
	SubstringFilterType: "initial",
}

// Default groupConfig (Active Directory)
var groupDefaults = groupConfig{
	Scope:       "sub",
	Objectclass: "posixGroup",
	Schema: groupSchema{
		ID:              "objectGUID",
		IDIsOctetString: false,
		Groupname:       "cn",
		Mail:            "mail",
		DisplayName:     "cn",
		GIDNumber:       "gidNumber",
		Member:          "memberUid",
	},
	SubstringFilterType: "initial",
}

// New initializes the default config
func New() Identity {
	return Identity{
		User:  userDefaults,
		Group: groupDefaults,
	}
}

// Setup initialzes some properties that can't be initialized from the
// mapstructure based config. Currently it just converts the LDAP search scope
// strings from the config to the integer constants expected by the ldap API
func (i *Identity) Setup() error {
	var err error
	if i.User.scopeVal, err = stringToScope(i.User.Scope); err != nil {
		return fmt.Errorf("error configuring user scope: %w", err)
	}

	if i.Group.scopeVal, err = stringToScope(i.Group.Scope); err != nil {
		return fmt.Errorf("error configuring group scope: %w", err)
	}

	if i.User.substringFilterVal, err = stringToFilterType(i.User.SubstringFilterType); err != nil {
		return fmt.Errorf("error configuring user substring filter type: %w", err)
	}

	if i.Group.substringFilterVal, err = stringToFilterType(i.Group.SubstringFilterType); err != nil {
		return fmt.Errorf("error configuring group substring filter type: %w", err)
	}

	switch i.User.DisableMechanism {
	case "group":
		if i.Group.LocalDisabledDN == "" {
			return fmt.Errorf("error configuring disable mechanism, disabled group DN not set")
		}
	case "attribute":
		if i.User.EnabledProperty == "" {
			return fmt.Errorf("error configuring disable mechanism, enabled property not set")
		}
	case "", "none":
	default:
		return fmt.Errorf("invalid disable mechanism setting: %s", i.User.DisableMechanism)
	}

	if sharedconf.MultiTenantEnabled() {
		if i.User.Schema.TenantID == "" {
			return fmt.Errorf("Invalid configuration: a 'tenantId' user schema attribute must be defined for multi-tenant setups")
		}
	} else {
		if i.User.Schema.TenantID != "" {
			return fmt.Errorf("Invalid configuration: Superfluous 'tenantId' user schema attribute defined for single-tenant setups")
		}
	}

	return nil
}

// GetLDAPUserByID looks up a user by the supplied Id. Returns the corresponding
// ldap.Entry
func (i *Identity) GetLDAPUserByID(ctx context.Context, lc ldap.Client, id *identityUser.UserId) (*ldap.Entry, error) {
	var filter string
	var err error
	if filter, err = i.getUserFilter(id); err != nil {
		return nil, err
	}
	return i.GetLDAPUserByFilter(ctx, lc, filter)
}

// GetLDAPUserByAttribute looks up a single user by attribute (can be "mail",
// "uid", "gid", "username" or "userid"). Returns the corresponding ldap.Entry
func (i *Identity) GetLDAPUserByAttribute(ctx context.Context, lc ldap.Client, attribute, value, tenantID string) (*ldap.Entry, error) {
	var filter string
	var err error
	if filter, err = i.getUserAttributeFilter(attribute, value, tenantID); err != nil {
		return nil, err
	}
	return i.GetLDAPUserByFilter(ctx, lc, filter)
}

// GetLDAPUserByFilter looks up a single user by the supplied LDAP filter
// returns the corresponding ldap.Entry
func (i *Identity) GetLDAPUserByFilter(ctx context.Context, lc ldap.Client, filter string) (*ldap.Entry, error) {
	log := appctx.GetLogger(ctx)
	_, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GetLDAPUserByFilter")
	defer span.End()
	searchRequest := ldap.NewSearchRequest(
		i.User.BaseDN, i.User.scopeVal, ldap.NeverDerefAliases, 1, 0, false,
		filter,
		i.getUserLDAPAttrTypes(),
		nil,
	)

	setLDAPSearchSpanAttributes(span, searchRequest)
	log.Debug().Str("backend", "ldap").Str("basedn", i.User.BaseDN).Str("filter", filter).Int("scope", i.User.scopeVal).Msg("LDAP Search")
	res, err := lc.Search(searchRequest)
	if err != nil {
		log.Debug().Str("backend", "ldap").Err(err).Str("userfilter", filter).Msg("Error looking up user by filter")
		var errmsg string
		if lerr, ok := err.(*ldap.Error); ok {
			if lerr.ResultCode == ldap.LDAPResultSizeLimitExceeded {
				errmsg = fmt.Sprintf("too many results searching for user '%s'", filter)
			}
		}
		span.SetAttributes(attribute.String("ldap.error", errmsg))
		span.SetStatus(codes.Error, errmsg)
		return nil, errtypes.NotFound(errmsg)
	}
	if len(res.Entries) == 0 {
		return nil, errtypes.NotFound(filter)
	}
	span.SetStatus(codes.Ok, "")

	return res.Entries[0], nil
}

// GetLDAPUserByDN looks up a single user by the supplied LDAP DN
// returns the corresponding ldap.Entry
func (i *Identity) GetLDAPUserByDN(ctx context.Context, lc ldap.Client, dn string) (*ldap.Entry, error) {
	log := appctx.GetLogger(ctx)
	_, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GetLDAPUserByDN")
	defer span.End()

	filter := fmt.Sprintf("(objectclass=%s)", i.User.Objectclass)
	if i.User.Filter != "" {
		filter = fmt.Sprintf("(&%s%s)", i.User.Filter, filter)
	}
	searchRequest := ldap.NewSearchRequest(
		dn, i.User.scopeVal, ldap.NeverDerefAliases, 1, 0, false,
		filter,
		i.getUserLDAPAttrTypes(),
		nil,
	)
	setLDAPSearchSpanAttributes(span, searchRequest)
	log.Debug().Str("backend", "ldap").Str("basedn", dn).Str("filter", filter).Int("scope", i.User.scopeVal).Msg("LDAP Search")
	res, err := lc.Search(searchRequest)
	if err != nil {
		log.Debug().Str("backend", "ldap").Err(err).Str("dn", dn).Msg("Error looking up user by DN")
		span.SetAttributes(attribute.String("ldap.error", err.Error()))
		span.SetStatus(codes.Error, "")
		return nil, errtypes.NotFound(dn)
	}
	span.SetStatus(codes.Ok, "")
	if len(res.Entries) == 0 {
		return nil, errtypes.NotFound(dn)
	}

	return res.Entries[0], nil
}

// GetLDAPUsers searches for users using a prefix-substring match on the user
// attributes. Returns a slice of matching ldap.Entries
func (i *Identity) GetLDAPUsers(ctx context.Context, lc ldap.Client, query, tenantID string) ([]*ldap.Entry, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GetLDAPUsers")
	defer span.End()
	log := appctx.GetLogger(ctx)
	filter := i.getUserFindFilter(query, tenantID)
	searchRequest := ldap.NewSearchRequest(
		i.User.BaseDN,
		i.User.scopeVal, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		i.getUserLDAPAttrTypes(),
		nil,
	)
	setLDAPSearchSpanAttributes(span, searchRequest)
	log.Debug().Str("backend", "ldap").Str("basedn", i.User.BaseDN).Str("filter", filter).Int("scope", i.User.scopeVal).Msg("LDAP Search")
	sr, err := lc.Search(searchRequest)
	if err != nil {
		log.Debug().Str("backend", "ldap").Err(err).Str("filter", filter).Msg("Error searching users")
		span.SetAttributes(attribute.String("ldap.error", err.Error()))
		span.SetStatus(codes.Error, "")
		return nil, errtypes.NotFound(query)
	}

	span.SetAttributes(attribute.Int("ldap.result_count", len(sr.Entries)))
	span.SetStatus(codes.Ok, "")

	return sr.Entries, nil
}

// IsLDAPUserInDisabledGroup checkes if the user is in the disabled group.
func (i *Identity) IsLDAPUserInDisabledGroup(ctx context.Context, lc ldap.Client, userEntry *ldap.Entry) bool {
	_, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "IsLDAPUserInDisabledGroup")
	defer span.End()

	span.SetAttributes(attribute.String("identity.config.disable_mechanism", i.User.DisableMechanism))
	span.SetStatus(codes.Ok, "")

	// Check if we need to do this here because the configuration is local to Identity.
	if i.User.DisableMechanism != "group" {
		return false
	}

	filter := fmt.Sprintf("(&(objectClass=groupOfNames)(%s=%s))", i.Group.Schema.Member, userEntry.DN)
	searchRequest := ldap.NewSearchRequest(
		i.Group.LocalDisabledDN,
		i.Group.scopeVal,
		ldap.NeverDerefAliases, 0, 0, false,
		filter,
		[]string{i.Group.Schema.ID},
		nil,
	)
	setLDAPSearchSpanAttributes(span, searchRequest)

	log.Debug().Str("backend", "ldap").Str("basedn", i.Group.LocalDisabledDN).Str("filter", filter).Int("scope", i.Group.scopeVal).Msg("LDAP Search")
	sr, err := lc.Search(searchRequest)
	if err != nil {
		log.Error().Str("backend", "ldap").Err(err).Str("filter", filter).Msg("Error looking up error group")
		// Err on the side of caution.
		span.SetAttributes(attribute.String("ldap.error", err.Error()))
		span.SetStatus(codes.Error, "")
		return true
	}

	return len(sr.Entries) > 0
}

// GetLDAPUserGroups looks up the group member ship of the supplied LDAP user entry.
// Returns a slice of strings with groupids
func (i *Identity) GetLDAPUserGroups(ctx context.Context, lc ldap.Client, userEntry *ldap.Entry) ([]string, error) {
	var memberValue string

	log := appctx.GetLogger(ctx)

	_, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GetLDAPUserGroups")
	defer span.End()

	if strings.ToLower(i.Group.Objectclass) == "posixgroup" {
		// posixGroup usually means that the member attribute just contains the username
		memberValue = userEntry.GetEqualFoldAttributeValue(i.User.Schema.Username)
	} else {
		// In all other case we assume the member Attribute to contain full LDAP DNs
		memberValue = userEntry.DN
	}

	filter := i.getGroupMemberFilter(memberValue)
	searchRequest := ldap.NewSearchRequest(
		i.Group.BaseDN, i.Group.scopeVal,
		ldap.NeverDerefAliases, 0, 0, false,
		filter,
		[]string{i.Group.Schema.ID},
		nil,
	)

	log.Debug().Str("backend", "ldap").Str("basedn", i.Group.BaseDN).Str("filter", filter).Int("scope", i.Group.scopeVal).Msg("LDAP Search")
	setLDAPSearchSpanAttributes(span, searchRequest)
	sr, err := lc.Search(searchRequest)
	if err != nil {
		log.Debug().Str("backend", "ldap").Err(err).Str("filter", filter).Msg("Error looking up group memberships")
		span.SetAttributes(attribute.String("ldap.error", err.Error()))
		span.SetStatus(codes.Error, "")
		return []string{}, err
	}
	span.SetStatus(codes.Ok, "")
	span.SetAttributes(attribute.Int("ldap.result_count", len(sr.Entries)))

	groups := make([]string, 0, len(sr.Entries))
	for _, entry := range sr.Entries {
		// FIXME this makes the users groups use the cn, not an immutable id
		// FIXME 1. use the memberof or members attribute of a user to get the groups
		// FIXME 2. ook up the id for each group
		var groupID string
		if i.Group.Schema.IDIsOctetString {
			raw := entry.GetEqualFoldRawAttributeValue(i.Group.Schema.ID)
			value, err := uuid.FromBytes(raw)
			if err != nil {
				return nil, err
			}
			groupID = value.String()
		} else {
			groupID = entry.GetEqualFoldAttributeValue(i.Group.Schema.ID)
		}

		groups = append(groups, groupID)
	}
	return groups, nil
}

// GetLDAPGroupByID looks up a group by the supplied Id. Returns the corresponding
// ldap.Entry
func (i *Identity) GetLDAPGroupByID(ctx context.Context, lc ldap.Client, id string) (*ldap.Entry, error) {
	var filter string
	var err error
	if filter, err = i.getGroupFilter(id); err != nil {
		return nil, err
	}
	return i.GetLDAPGroupByFilter(ctx, lc, filter)
}

// GetLDAPGroupByAttribute looks up a single group by attribute (can be "mail", "gid_number",
// "display_name", "group_name", "group_id"). Returns the corresponding ldap.Entry
func (i *Identity) GetLDAPGroupByAttribute(ctx context.Context, lc ldap.Client, attribute, value string) (*ldap.Entry, error) {
	var filter string
	var err error
	if filter, err = i.getGroupAttributeFilter(attribute, value); err != nil {
		return nil, err
	}
	return i.GetLDAPGroupByFilter(ctx, lc, filter)
}

// GetLDAPGroupByFilter looks up a single group by the supplied LDAP filter
// returns the corresponding ldap.Entry
func (i *Identity) GetLDAPGroupByFilter(ctx context.Context, lc ldap.Client, filter string) (*ldap.Entry, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GetLDAPGroupByFilter")
	defer span.End()
	log := appctx.GetLogger(ctx)
	searchRequest := ldap.NewSearchRequest(
		i.Group.BaseDN, i.Group.scopeVal, ldap.NeverDerefAliases, 1, 0, false,
		filter,
		[]string{
			i.Group.Schema.DisplayName,
			i.Group.Schema.ID,
			i.Group.Schema.Mail,
			i.Group.Schema.Groupname,
			i.Group.Schema.Member,
			i.Group.Schema.GIDNumber,
		},
		nil,
	)
	setLDAPSearchSpanAttributes(span, searchRequest)

	log.Debug().Str("backend", "ldap").Str("basedn", i.Group.BaseDN).Str("filter", filter).Int("scope", i.Group.scopeVal).Msg("LDAP Search")
	res, err := lc.Search(searchRequest)
	if err != nil {
		log.Debug().Str("backend", "ldap").Err(err).Str("filter", filter).Msg("Error looking up group by filter")
		var errmsg string
		if lerr, ok := err.(*ldap.Error); ok {
			if lerr.ResultCode == ldap.LDAPResultSizeLimitExceeded {
				errmsg = fmt.Sprintf("too many results searching for group '%s'", filter)
			}
		}
		span.SetAttributes(attribute.String("ldap.error", errmsg))
		span.SetStatus(codes.Error, "")
		return nil, errtypes.NotFound(errmsg)
	}
	if len(res.Entries) == 0 {
		return nil, errtypes.NotFound(filter)
	}
	span.SetStatus(codes.Ok, "")
	return res.Entries[0], nil
}

// GetLDAPGroups searches for groups using a prefix-substring match on the group
// attributes. Returns a slice of matching ldap.Entries
func (i *Identity) GetLDAPGroups(ctx context.Context, lc ldap.Client, query string) ([]*ldap.Entry, error) {
	ctx, span := appctx.GetTracerProvider(ctx).Tracer(tracerName).Start(ctx, "GetLDAPGroups")
	defer span.End()
	log := appctx.GetLogger(ctx)
	searchRequest := ldap.NewSearchRequest(
		i.Group.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		i.getGroupFindFilter(query),
		[]string{
			i.Group.Schema.DisplayName,
			i.Group.Schema.ID,
			i.Group.Schema.Mail,
			i.Group.Schema.Groupname,
			i.Group.Schema.GIDNumber,
		},
		nil,
	)
	setLDAPSearchSpanAttributes(span, searchRequest)
	sr, err := lc.Search(searchRequest)
	if err != nil {
		span.SetAttributes(attribute.String("ldap.error", err.Error()))
		span.SetStatus(codes.Error, "")
		log.Debug().Str("backend", "ldap").Err(err).Str("query", query).Msg("Error search for groups")
		return nil, errtypes.NotFound(query)
	}
	span.SetStatus(codes.Ok, "")
	return sr.Entries, nil
}

// GetLDAPGroupMembers looks up all members of the supplied LDAP group entry and returns the
// corresponding LDAP user entries
func (i *Identity) GetLDAPGroupMembers(ctx context.Context, lc ldap.Client, group *ldap.Entry) ([]*ldap.Entry, error) {
	log := appctx.GetLogger(ctx)
	members := group.GetEqualFoldAttributeValues(i.Group.Schema.Member)
	log.Debug().Str("dn", group.DN).Interface("member", members).Msg("Get Group members")
	memberEntries := make([]*ldap.Entry, 0, len(members))
	for _, member := range members {
		var e *ldap.Entry
		var err error
		if strings.ToLower(i.Group.Objectclass) == "posixgroup" {
			e, err = i.GetLDAPUserByAttribute(ctx, lc, "username", member, "")
		} else {
			e, err = i.GetLDAPUserByDN(ctx, lc, member)
		}
		if err != nil {
			log.Warn().Err(err).Interface("member", member).Msg("Failed read user entry for member")
			continue
		}
		memberEntries = append(memberEntries, e)
	}

	return memberEntries, nil
}

func filterEscapeBinaryUUID(value uuid.UUID) string {
	filtered := ""
	for _, b := range value {
		filtered = fmt.Sprintf("%s\\%02x", filtered, b)
	}
	return filtered
}

func (i *Identity) getUserFilter(uid *identityUser.UserId) (string, error) {
	var escapedUUID string
	if i.User.Schema.IDIsOctetString {
		id, err := uuid.Parse(uid.GetOpaqueId())
		if err != nil {
			err := errors.Wrap(err, fmt.Sprintf("error parsing OpaqueID '%s' as UUID", uid))
			return "", err
		}
		escapedUUID = filterEscapeBinaryUUID(id)
	} else {
		escapedUUID = ldap.EscapeFilter(uid.GetOpaqueId())
	}
	return fmt.Sprintf("(&%s(objectclass=%s)%s(%s=%s))",
		i.User.Filter,
		i.User.Objectclass,
		i.tenantFilter(uid.GetTenantId()),
		i.User.Schema.ID,
		escapedUUID,
	), nil
}

func (i *Identity) getUserAttributeFilter(attribute, value, tenantID string) (string, error) {
	switch attribute {
	case "mail":
		attribute = i.User.Schema.Mail
	case "uid":
		attribute = i.User.Schema.UIDNumber
	case "gid":
		attribute = i.User.Schema.GIDNumber
	case "username":
		attribute = i.User.Schema.Username
	case "userid":
		attribute = i.User.Schema.ID
	case "tenantid":
		attribute = i.User.Schema.TenantID
	default:
		return "", errors.New("ldap: invalid field " + attribute)
	}
	if attribute == i.User.Schema.ID && i.User.Schema.IDIsOctetString {
		id, err := uuid.Parse(value)
		if err != nil {
			err := errors.Wrap(err, fmt.Sprintf("error parsing OpaqueID '%s' as UUID", value))
			return "", err
		}
		value = filterEscapeBinaryUUID(id)
	} else {
		value = ldap.EscapeFilter(value)
	}
	return fmt.Sprintf("(&%s(objectclass=%s)(%s=%s)%s%s)",
		i.User.Filter,
		i.User.Objectclass,
		attribute,
		value,
		i.tenantFilter(tenantID),
		i.disabledFilter(),
	), nil
}

func (i *Identity) tenantFilter(tenantID string) string {
	if tenantID != "" && i.User.Schema.TenantID != "" {
		return fmt.Sprintf("(%s=%s)", i.User.Schema.TenantID, ldap.EscapeFilter(tenantID))
	}
	return ""
}
func (i *Identity) disabledFilter() string {
	if i.User.DisableMechanism == "attribute" {
		return fmt.Sprintf("(!(%s=FALSE))", i.User.EnabledProperty)
	}
	return ""
}

// getUserFindFilter construct a LDAP filter to perform a prefix-substring
// search for users.
func (i *Identity) getUserFindFilter(query, tenantID string) string {
	searchAttrs := []string{
		i.User.Schema.Mail,
		i.User.Schema.DisplayName,
		i.User.Schema.Username,
	}
	var filter, squery string
	switch i.User.substringFilterVal {
	case ldap.FilterSubstringsInitial:
		squery = fmt.Sprintf("%s*", ldap.EscapeFilter(query))
	case ldap.FilterSubstringsAny:
		squery = fmt.Sprintf("*%s*", ldap.EscapeFilter(query))
	case ldap.FilterSubstringsFinal:
		squery = fmt.Sprintf("*%s", ldap.EscapeFilter(query))
	}
	for _, attr := range searchAttrs {
		filter = fmt.Sprintf("%s(%s=%s)", filter, attr, squery)
	}
	// substring search for UUID is not possible
	filter = fmt.Sprintf("(|%s(%s=%s))", filter, i.User.Schema.ID, ldap.EscapeFilter(query))

	return fmt.Sprintf("(&%s(objectclass=%s)%s%s)",
		i.User.Filter,
		i.User.Objectclass,
		i.tenantFilter(tenantID),
		filter,
	)
}

// getGroupFindFilter construct a LDAP filter to perform a prefix-substring
// search for groups.
func (i *Identity) getGroupFindFilter(query string) string {
	searchAttrs := []string{
		i.Group.Schema.Mail,
		i.Group.Schema.DisplayName,
		i.Group.Schema.Groupname,
	}
	var filter, squery string
	switch i.Group.substringFilterVal {
	case ldap.FilterSubstringsInitial:
		squery = fmt.Sprintf("%s*", ldap.EscapeFilter(query))
	case ldap.FilterSubstringsAny:
		squery = fmt.Sprintf("*%s*", ldap.EscapeFilter(query))
	case ldap.FilterSubstringsFinal:
		squery = fmt.Sprintf("*%s", ldap.EscapeFilter(query))
	}
	for _, attr := range searchAttrs {
		filter = fmt.Sprintf("%s(%s=%s)", filter, attr, squery)
	}
	// substring search for UUID is not possible
	filter = fmt.Sprintf("%s(%s=%s)", filter, i.Group.Schema.ID, ldap.EscapeFilter(query))

	return fmt.Sprintf("(&%s(objectclass=%s)(|%s))",
		i.Group.Filter,
		i.Group.Objectclass,
		filter,
	)
}

func stringToScope(scope string) (int, error) {
	var s int
	switch scope {
	case "sub":
		s = ldap.ScopeWholeSubtree
	case "one":
		s = ldap.ScopeSingleLevel
	case "base":
		s = ldap.ScopeBaseObject
	default:
		return 0, fmt.Errorf("invalid Scope '%s'", scope)
	}
	return s, nil
}

func stringToFilterType(t string) (int, error) {
	var s int
	switch t {
	case "initial":
		s = ldap.FilterSubstringsInitial
	case "any":
		s = ldap.FilterSubstringsAny
	case "final":
		s = ldap.FilterSubstringsFinal
	default:
		return 0, fmt.Errorf("invalid filter type '%s'", t)
	}
	return s, nil
}

func (i *Identity) getGroupMemberFilter(memberName string) string {
	return fmt.Sprintf("(&%s(objectclass=%s)(%s=%s))",
		i.Group.Filter,
		i.Group.Objectclass,
		i.Group.Schema.Member,
		ldap.EscapeFilter(memberName),
	)
}

func (i *Identity) getGroupFilter(id string) (string, error) {
	var escapedUUID string
	if i.Group.Schema.IDIsOctetString {
		id, err := uuid.Parse(id)
		if err != nil {
			err := errors.Wrap(err, fmt.Sprintf("error parsing OpaqueID '%s' as UUID", id))
			return "", err
		}
		escapedUUID = filterEscapeBinaryUUID(id)
	} else {
		escapedUUID = ldap.EscapeFilter(id)
	}

	return fmt.Sprintf("(&%s(objectclass=%s)(%s=%s))",
		i.Group.Filter,
		i.Group.Objectclass,
		i.Group.Schema.ID,
		escapedUUID,
	), nil
}

func (i *Identity) getGroupAttributeFilter(attribute, value string) (string, error) {
	switch attribute {
	case "mail":
		attribute = i.Group.Schema.Mail
	case "gid_number":
		attribute = i.Group.Schema.GIDNumber
	case "display_name":
		attribute = i.Group.Schema.DisplayName
	case "group_name":
		attribute = i.Group.Schema.Groupname
	case "group_id":
		attribute = i.Group.Schema.ID
	default:
		return "", errors.New("ldap: invalid field " + attribute)
	}
	if attribute == i.Group.Schema.ID && i.Group.Schema.IDIsOctetString {
		id, err := uuid.Parse(value)
		if err != nil {
			err := errors.Wrap(err, fmt.Sprintf("error parsing OpaqueID '%s' as UUID", value))
			return "", err
		}
		value = filterEscapeBinaryUUID(id)
	} else {
		value = ldap.EscapeFilter(value)
	}
	return fmt.Sprintf("(&%s(objectclass=%s)(%s=%s))",
		i.Group.Filter,
		i.Group.Objectclass,
		attribute,
		value,
	), nil
}

// GetUserType is used to get the proper UserType from ldap entry string
func (i *Identity) GetUserType(userEntry *ldap.Entry) identityUser.UserType {
	userTypeString := userEntry.GetEqualFoldAttributeValue(i.User.UserTypeProperty)
	switch strings.ToLower(userTypeString) {
	case "member":
		return identityUser.UserType_USER_TYPE_PRIMARY
	case "guest":
		return identityUser.UserType_USER_TYPE_GUEST
	default:
		return identityUser.UserType_USER_TYPE_PRIMARY
	}
}

func (i *Identity) getUserLDAPAttrTypes() []string {
	// The are the attributes we request unconditionally when looking up users
	// as they are needed to populate a user object
	attrs := []string{
		i.User.Schema.ID,
		i.User.Schema.Username,
		i.User.Schema.Mail,
		i.User.Schema.DisplayName,
	}

	// Only add optional attributes if they are configured
	if i.User.Schema.UIDNumber != "" {
		attrs = append(attrs, i.User.Schema.UIDNumber)
	}
	if i.User.Schema.GIDNumber != "" {
		attrs = append(attrs, i.User.Schema.GIDNumber)
	}
	if i.User.Schema.TenantID != "" {
		attrs = append(attrs, i.User.Schema.TenantID)
	}
	if i.User.EnabledProperty != "" {
		attrs = append(attrs, i.User.EnabledProperty)
	}
	if i.User.UserTypeProperty != "" {
		attrs = append(attrs, i.User.UserTypeProperty)
	}
	return attrs
}
func setLDAPSearchSpanAttributes(span trace.Span, request *ldap.SearchRequest) {
	span.SetAttributes(
		attribute.String("ldap.basedn", request.BaseDN),
		attribute.String("ldap.filter", request.Filter),
		attribute.Int("ldap.scope", request.Scope),
		attribute.Int("ldap.size_limit", request.SizeLimit),
	)
}
