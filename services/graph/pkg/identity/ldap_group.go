package identity

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/CiscoM31/godata"
	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/libregraph/idm/pkg/ldapdn"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/odata"
)

type groupAttributeMap struct {
	name   string
	id     string
	member string
}

// GetGroup implements the Backend Interface for the LDAP Backend
func (i *LDAP) GetGroup(ctx context.Context, nameOrID string, queryParam url.Values) (*libregraph.Group, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("GetGroup")
	e, err := i.getLDAPGroupByNameOrID(nameOrID, true)
	if err != nil {
		return nil, err
	}
	sel := strings.Split(queryParam.Get("$select"), ",")
	exp := strings.Split(queryParam.Get("$expand"), ",")
	var g *libregraph.Group
	if g = i.createGroupModelFromLDAP(e); g == nil {
		return nil, errorcode.New(errorcode.ItemNotFound, "not found")
	}
	if slices.Contains(sel, "members") || slices.Contains(exp, "members") {
		members, err := i.expandLDAPAttributeEntries(ctx, e, i.groupAttributeMap.member, "")
		if err != nil {
			return nil, err
		}
		g.Members = make([]libregraph.User, 0, len(members))
		if len(members) > 0 {
			for _, ue := range members {
				if u := i.createUserModelFromLDAP(ue); u != nil {
					g.Members = append(g.Members, *u)
				}
			}
		}
	}
	return g, nil
}

// GetGroups implements the Backend Interface for the LDAP Backend
func (i *LDAP) GetGroups(ctx context.Context, oreq *godata.GoDataRequest) ([]*libregraph.Group, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("GetGroups")

	search, err := odata.GetSearchValues(oreq.Query)
	if err != nil {
		return nil, err
	}

	var expandMembers bool
	exp, err := odata.GetExpandValues(oreq.Query)
	if err != nil {
		return nil, err
	}
	sel, err := odata.GetSelectValues(oreq.Query)
	if err != nil {
		return nil, err
	}

	if slices.Contains(exp, "members") || slices.Contains(sel, "members") {
		expandMembers = true
	}

	var groupFilter string
	if search != "" {
		search = ldap.EscapeFilter(search)
		groupFilter = fmt.Sprintf(
			"(%s=*%s*)",
			i.groupAttributeMap.name, search,
		)
	}
	groupFilter = fmt.Sprintf("(&%s(objectClass=%s)%s)", i.groupFilter, i.groupObjectClass, groupFilter)

	groupAttrs := []string{
		i.groupAttributeMap.name,
		i.groupAttributeMap.id,
	}
	if expandMembers {
		groupAttrs = append(groupAttrs, i.groupAttributeMap.member)
	}

	searchRequest := ldap.NewSearchRequest(
		i.groupBaseDN, i.groupScope, ldap.NeverDerefAliases, 0, 0, false,
		groupFilter,
		groupAttrs,
		nil,
	)
	logger.Debug().Str("backend", "ldap").
		Str("base", searchRequest.BaseDN).
		Str("filter", searchRequest.Filter).
		Int("scope", searchRequest.Scope).
		Int("sizelimit", searchRequest.SizeLimit).
		Interface("attributes", searchRequest.Attributes).
		Msg("GetGroups")
	res, err := i.conn.Search(searchRequest)
	if err != nil {
		return nil, errorcode.New(errorcode.ItemNotFound, err.Error())
	}

	groups := make([]*libregraph.Group, 0, len(res.Entries))

	var g *libregraph.Group
	for _, e := range res.Entries {
		if g = i.createGroupModelFromLDAP(e); g == nil {
			continue
		}
		if expandMembers {
			members, err := i.expandLDAPAttributeEntries(ctx, e, i.groupAttributeMap.member, "")
			if err != nil {
				return nil, err
			}
			g.Members = make([]libregraph.User, 0, len(members))
			if len(members) > 0 {
				for _, ue := range members {
					if u := i.createUserModelFromLDAP(ue); u != nil {
						g.Members = append(g.Members, *u)
					}
				}
			}
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// GetGroupMembers implements the Backend Interface for the LDAP Backend
func (i *LDAP) GetGroupMembers(ctx context.Context, groupID string, req *godata.GoDataRequest) ([]*libregraph.User, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("GetGroupMembers")

	exp, err := odata.GetExpandValues(req.Query)
	if err != nil {
		return nil, err
	}

	e, err := i.getLDAPGroupByNameOrID(groupID, true)
	if err != nil {
		return nil, err
	}

	searchTerm, err := odata.GetSearchValues(req.Query)
	if err != nil {
		return nil, err
	}

	memberEntries, err := i.expandLDAPAttributeEntries(ctx, e, i.groupAttributeMap.member, searchTerm)
	result := make([]*libregraph.User, 0, len(memberEntries))
	if err != nil {
		return nil, err
	}
	for _, member := range memberEntries {
		if u := i.createUserModelFromLDAP(member); u != nil {
			if slices.Contains(exp, "memberOf") {
				userGroups, err := i.getGroupsForUser(member.DN)
				if err != nil {
					return nil, err
				}
				u.MemberOf = i.groupsFromLDAPEntries(userGroups)
			}
			result = append(result, u)
		}
	}

	return result, nil
}

// CreateGroup implements the Backend Interface for the LDAP Backend
// It is currently restricted to managing groups based on the "groupOfNames" ObjectClass.
// As "groupOfNames" requires a "member" Attribute to be present. Empty Groups (groups
// without a member) a represented by adding an empty DN as the single member.
func (i *LDAP) CreateGroup(ctx context.Context, group libregraph.Group) (*libregraph.Group, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("create group")
	if !i.writeEnabled && i.groupCreateBaseDN == i.groupBaseDN {
		return nil, errorcode.New(errorcode.NotAllowed, "server is configured read-only")
	}
	ar, err := i.groupToAddRequest(group)
	if err != nil {
		return nil, err
	}

	if err := i.conn.Add(ar); err != nil {
		var lerr *ldap.Error
		logger.Debug().Str("backend", "ldap").Str("dn", group.GetDisplayName()).Err(err).Msg("Failed to create group")
		if errors.As(err, &lerr) {
			if lerr.ResultCode == ldap.LDAPResultEntryAlreadyExists {
				err = errorcode.New(errorcode.NameAlreadyExists, "group already exists")
			}
		}
		return nil, err
	}

	// Read	back group from LDAP to get the generated UUID
	e, err := i.getGroupByDN(ar.DN)
	if err != nil {
		return nil, err
	}
	return i.createGroupModelFromLDAP(e), nil
}

// DeleteGroup implements the Backend Interface.
func (i *LDAP) DeleteGroup(ctx context.Context, id string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("DeleteGroup")
	if !i.writeEnabled && i.groupCreateBaseDN == i.groupBaseDN {
		return errorcode.New(errorcode.NotAllowed, "server is configured read-only")
	}

	e, err := i.getLDAPGroupByID(id, false)
	if err != nil {
		return err
	}

	if i.isLDAPGroupReadOnly(e) {
		return errorcode.New(errorcode.NotAllowed, "group is read-only")
	}

	dr := ldap.DelRequest{DN: e.DN}
	if err = i.conn.Del(&dr); err != nil {
		return err
	}
	return nil
}

// UpdateGroupName implements the Backend Interface.
func (i *LDAP) UpdateGroupName(ctx context.Context, groupID string, groupName string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("AddMembersToGroup")
	if !i.writeEnabled && i.groupCreateBaseDN == i.groupBaseDN {
		return errorcode.New(errorcode.NotAllowed, "server is configured read-only")
	}

	ge, err := i.getLDAPGroupByID(groupID, true)
	if err != nil {
		return err
	}

	if i.isLDAPGroupReadOnly(ge) {
		return errorcode.New(errorcode.NotAllowed, "group is read-only")
	}

	if ge.GetEqualFoldAttributeValue(i.groupAttributeMap.name) == groupName {
		return nil
	}

	attributeTypeAndValue := ldap.AttributeTypeAndValue{
		Type:  i.groupAttributeMap.name,
		Value: groupName,
	}
	newDNString := attributeTypeAndValue.String()

	logger.Debug().Str("originalDN", ge.DN).Str("newDN", newDNString).Msg("Modifying DN")
	mrdn := ldap.NewModifyDNRequest(ge.DN, newDNString, true, "")

	if err := i.conn.ModifyDN(mrdn); err != nil {
		var lerr *ldap.Error
		logger.Debug().Str("originalDN", ge.DN).Str("newDN", newDNString).Err(err).Msg("Failed to modify DN")
		if errors.As(err, &lerr) {
			if lerr.ResultCode == ldap.LDAPResultEntryAlreadyExists {
				err = errorcode.New(errorcode.NameAlreadyExists, "Group name already in use")
			}
		}
		return err
	}

	return nil
}

// AddMembersToGroup implements the Backend Interface for the LDAP backend.
// Currently, it is limited to adding Users as Group members. Adding other groups
// as members is not yet implemented
func (i *LDAP) AddMembersToGroup(ctx context.Context, groupID string, memberIDs []string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("AddMembersToGroup")
	if !i.writeEnabled && i.groupCreateBaseDN == i.groupBaseDN {
		return errorcode.New(errorcode.NotAllowed, "server is configured read-only")
	}
	ge, err := i.getLDAPGroupByNameOrID(groupID, true)
	if err != nil {
		return err
	}

	if i.isLDAPGroupReadOnly(ge) {
		return errorcode.New(errorcode.NotAllowed, "group is read-only")
	}

	mr := ldap.ModifyRequest{DN: ge.DN}
	// Handle empty groups (using the empty member attribute)
	current := ge.GetEqualFoldAttributeValues(i.groupAttributeMap.member)
	if len(current) == 1 && current[0] == "" {
		mr.Delete(i.groupAttributeMap.member, []string{""})
	}

	// Create a Set of current members for faster lookups
	currentSet := make(map[string]struct{}, len(current))
	for _, currentMember := range current {
		// We can ignore any empty member value here
		if currentMember == "" {
			continue
		}
		nCurrentMember, err := ldapdn.ParseNormalize(currentMember)
		if err != nil {
			// We couldn't parse the member value as a DN. Let's skip it, but log a warning
			logger.Warn().Str("memberDN", currentMember).Err(err).Msg("Couldn't parse DN")
			continue
		}
		currentSet[nCurrentMember] = struct{}{}
	}

	var newMemberDN []string
	for _, memberID := range memberIDs {
		me, err := i.getLDAPUserByID(memberID)
		if err != nil {
			return err
		}
		nDN, err := ldapdn.ParseNormalize(me.DN)
		if err != nil {
			logger.Error().Str("new member", me.DN).Err(err).Msg("Couldn't parse DN")
			return err
		}
		if _, present := currentSet[nDN]; !present {
			newMemberDN = append(newMemberDN, me.DN)
		} else {
			logger.Debug().Str("memberDN", me.DN).Msg("Member already present in group. Skipping")
		}
	}

	if len(newMemberDN) > 0 {
		// Small retry loop. It might be that, when reading the group we found the empty group member ("",
		// line 289 above). Our modify operation tries to delete that value. However, another go-routine
		// might have done that in parallel. In that case
		// (LDAPResultNoSuchAttribute) we need to retry the modification
		// without to delete.
		for j := 0; j < 2; j++ {
			mr.Add(i.groupAttributeMap.member, newMemberDN)
			if err := i.conn.Modify(&mr); err != nil {
				if lerr, ok := err.(*ldap.Error); ok {
					switch lerr.ResultCode {
					case ldap.LDAPResultAttributeOrValueExists:
						err = fmt.Errorf("duplicate member entries in request")
					case ldap.LDAPResultNoSuchAttribute:
						if len(mr.Changes) == 2 {
							// We tried the special case for adding the first group member, but some
							// other request running in parallel did that already. Retry with a "normal"
							// modification
							logger.Debug().Err(err).
								Msg("Failed to add first group member. Retrying once, without deleting the empty member value.")
							mr.Changes = make([]ldap.Change, 0, 1)
							continue
						}
					default:
						logger.Info().Err(err).Msg("Failed to modify group member entries on PATCH group")
						err = fmt.Errorf("unknown error when trying to modify group member entries")
					}
				}
				return err
			}
			// succeeded
			break
		}
	}
	return nil
}

// RemoveMemberFromGroup implements the Backend Interface.
func (i *LDAP) RemoveMemberFromGroup(ctx context.Context, groupID string, memberID string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("RemoveMemberFromGroup")
	if !i.writeEnabled && i.groupCreateBaseDN == i.groupBaseDN {
		return errorcode.New(errorcode.NotAllowed, "server is configured read-only")
	}

	ge, err := i.getLDAPGroupByID(groupID, true)
	if err != nil {
		logger.Debug().Str("backend", "ldap").Str("groupID", groupID).Msg("Error looking up group")
		return err
	}

	if i.isLDAPGroupReadOnly(ge) {
		return errorcode.New(errorcode.NotAllowed, "group is read-only")
	}

	me, err := i.getLDAPUserByID(memberID)
	if err != nil {
		logger.Debug().Str("backend", "ldap").Str("memberID", memberID).Msg("Error looking up group member")
		return err
	}

	logger.Debug().Str("backend", "ldap").Str("groupdn", ge.DN).Str("member", me.DN).Msg("remove member")

	if err = i.removeEntryByDNAndAttributeFromEntry(ge, me.DN, i.groupAttributeMap.member); err != nil {
		logger.Error().Err(err).Str("backend", "ldap").Str("group", groupID).Str("member", memberID).Msg("Failed to remove member from group.")
	}
	return err
}

func (i *LDAP) groupToAddRequest(group libregraph.Group) (*ldap.AddRequest, error) {
	ar := ldap.NewAddRequest(i.getGroupCreateLDAPDN(group), nil)

	attrMap, err := i.groupToLDAPAttrValues(group)
	if err != nil {
		return nil, err
	}
	for attrType, values := range attrMap {
		ar.Attribute(attrType, values)
	}
	return ar, nil
}

func (i *LDAP) getGroupCreateLDAPDN(group libregraph.Group) string {
	attributeTypeAndValue := ldap.AttributeTypeAndValue{
		Type:  "cn",
		Value: group.GetDisplayName(),
	}
	return fmt.Sprintf("%s,%s", attributeTypeAndValue.String(), i.groupCreateBaseDN)
}

func (i *LDAP) groupToLDAPAttrValues(group libregraph.Group) (map[string][]string, error) {
	attrs := map[string][]string{
		i.groupAttributeMap.name: {group.GetDisplayName()},
		"objectClass":            {"groupOfNames", "top"},
		// This is a crutch to allow groups without members for LDAP servers
		// that apply strict Schema checking. The RFCs define "member/uniqueMember"
		// as required attribute for groupOfNames/groupOfUniqueNames. So we
		// add an empty string (which is a valid DN) as the initial member.
		// It will be replaced once real members are added.
		// We might want to use the newer, but not so broadly used "groupOfMembers"
		// objectclass (RFC2307bis-02) where "member" is optional.
		i.groupAttributeMap.member: {""},
	}

	if !i.useServerUUID {
		attrs["openCloudUUID"] = []string{uuid.NewString()}
		attrs["objectClass"] = append(attrs["objectClass"], "openCloudObject")
	}
	return attrs, nil
}

func (i *LDAP) getLDAPGroupByID(id string, requestMembers bool) (*ldap.Entry, error) {
	idString, err := filterEscapeUUID(i.groupIDisOctetString, id)
	if err != nil {
		return nil, fmt.Errorf("invalid group id: %w", err)
	}
	filter := fmt.Sprintf("(%s=%s)", i.groupAttributeMap.id, idString)
	return i.getLDAPGroupByFilter(filter, requestMembers)
}

func (i *LDAP) getLDAPGroupByNameOrID(nameOrID string, requestMembers bool) (*ldap.Entry, error) {
	idString, err := filterEscapeUUID(i.groupIDisOctetString, nameOrID)
	// err != nil just means that this is not an uuid, so we can skip the uuid filter part
	// and just filter by name
	filter := ""
	if err == nil {
		filter = fmt.Sprintf("(|(%s=%s)(%s=%s))", i.groupAttributeMap.name, ldap.EscapeFilter(nameOrID), i.groupAttributeMap.id, idString)
	} else {
		filter = fmt.Sprintf("(%s=%s)", i.userAttributeMap.userName, ldap.EscapeFilter(nameOrID))
	}
	return i.getLDAPGroupByFilter(filter, requestMembers)
}

func (i *LDAP) getLDAPGroupByFilter(filter string, requestMembers bool) (*ldap.Entry, error) {
	e, err := i.getLDAPGroupsByFilter(filter, requestMembers, true)
	if err != nil {
		return nil, err
	}
	if len(e) == 0 {
		return nil, errorcode.New(errorcode.ItemNotFound, "not found")
	}

	return e[0], nil
}

// Search for LDAP Groups matching the specified filter, if requestMembers is true the groupMemberShip
// attribute will be part of the result attributes. The LDAP filter is combined with the configured groupFilter
// resulting in a filter like "(&(LDAP.groupFilter)(objectClass=LDAP.groupObjectClass)(<filter_from_args>))"
func (i *LDAP) getLDAPGroupsByFilter(filter string, requestMembers, single bool) ([]*ldap.Entry, error) {
	attrs := []string{
		i.groupAttributeMap.name,
		i.groupAttributeMap.id,
	}

	if requestMembers {
		attrs = append(attrs, i.groupAttributeMap.member)
	}

	sizelimit := 0
	if single {
		sizelimit = 1
	}
	searchRequest := ldap.NewSearchRequest(
		i.groupBaseDN, i.groupScope, ldap.NeverDerefAliases, sizelimit, 0, false,
		fmt.Sprintf("(&%s(objectClass=%s)%s)", i.groupFilter, i.groupObjectClass, filter),
		attrs,
		nil,
	)
	i.logger.Debug().Str("backend", "ldap").
		Str("base", searchRequest.BaseDN).
		Str("filter", searchRequest.Filter).
		Int("scope", searchRequest.Scope).
		Int("sizelimit", searchRequest.SizeLimit).
		Interface("attributes", searchRequest.Attributes).
		Msg("getLDAPGroupsByFilter")
	res, err := i.conn.Search(searchRequest)
	if err != nil {
		var errmsg string
		if lerr, ok := err.(*ldap.Error); ok {
			if lerr.ResultCode == ldap.LDAPResultSizeLimitExceeded {
				errmsg = fmt.Sprintf("too many results searching for group '%s'", filter)
				i.logger.Debug().Str("backend", "ldap").Err(lerr).Msg(errmsg)
			}
		}
		return nil, errorcode.New(errorcode.ItemNotFound, errmsg)
	}
	return res.Entries, nil
}

func (i *LDAP) getGroupByDN(dn string) (*ldap.Entry, error) {
	attrs := []string{
		i.groupAttributeMap.id,
		i.groupAttributeMap.name,
	}
	filter := fmt.Sprintf("(objectClass=%s)", i.groupObjectClass)

	if i.groupFilter != "" {
		filter = fmt.Sprintf("(&%s(%s))", filter, i.groupFilter)
	}
	return i.getEntryByDN(dn, attrs, filter)
}

func (i *LDAP) getGroupsForUser(dn string) ([]*ldap.Entry, error) {
	groupFilter := fmt.Sprintf(
		"(%s=%s)",
		i.groupAttributeMap.member, ldap.EscapeFilter(dn),
	)
	userGroups, err := i.getLDAPGroupsByFilter(groupFilter, false, false)
	if err != nil {
		return nil, err
	}
	return userGroups, nil
}

func (i *LDAP) createGroupModelFromLDAP(e *ldap.Entry) *libregraph.Group {
	name := e.GetEqualFoldAttributeValue(i.groupAttributeMap.name)
	id, err := i.ldapUUIDtoString(e, i.groupAttributeMap.id, i.groupIDisOctetString)
	if err != nil {
		i.logger.Warn().Str("dn", e.DN).Str(i.groupAttributeMap.id, e.GetEqualFoldAttributeValue(i.groupAttributeMap.id)).Msg("Invalid User. Cannot convert UUID")
	}
	groupTypes := []string{}

	if i.isLDAPGroupReadOnly(e) {
		groupTypes = []string{"ReadOnly"}
	}

	if id != "" && name != "" {
		return &libregraph.Group{
			DisplayName: &name,
			Id:          &id,
			GroupTypes:  groupTypes,
		}
	}
	i.logger.Warn().Str("dn", e.DN).Msg("Group is missing name or id")
	return nil
}

func (i *LDAP) isLDAPGroupReadOnly(e *ldap.Entry) bool {
	groupDN, err := ldap.ParseDN(e.DN)
	if err != nil {
		i.logger.Warn().Err(err).Str("dn", e.DN).Msg("Failed to parse DN")
		return false
	}

	baseDN, err := ldap.ParseDN(i.groupCreateBaseDN)
	if err != nil {
		i.logger.Warn().Err(err).Str("dn", i.groupCreateBaseDN).Msg("Failed to parse DN")
		return false
	}

	return !baseDN.AncestorOfFold(groupDN)
}

func (i *LDAP) groupsFromLDAPEntries(e []*ldap.Entry) []libregraph.Group {
	groups := make([]libregraph.Group, 0, len(e))
	for _, g := range e {
		if grp := i.createGroupModelFromLDAP(g); grp != nil {
			groups = append(groups, *grp)
		}
	}
	return groups
}
