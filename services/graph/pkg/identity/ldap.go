package identity

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/CiscoM31/godata"
	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/libregraph/idm/pkg/ldapdn"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/odata"
)

const (
	givenNameAttribute  = "givenname"
	surNameAttribute    = "sn"
	identitiesAttribute = "openCloudExternalIdentity"
	lastSignAttribute   = "openCloudLastSignInTimestamp"
	ldapDateFormat      = "20060102150405Z0700"
)

// DisableUserMechanismType is used instead of directly using the string values from the configuration.
type DisableUserMechanismType int64

// The different DisableMechanism* constants are used for managing the enabling/disabling of users.
const (
	DisableMechanismNone DisableUserMechanismType = iota
	DisableMechanismAttribute
	DisableMechanismGroup
)

var mechanismMap = map[string]DisableUserMechanismType{
	"":          DisableMechanismNone,
	"none":      DisableMechanismNone,
	"attribute": DisableMechanismAttribute,
	"group":     DisableMechanismGroup,
}

type LDAP struct {
	useServerUUID   bool
	writeEnabled    bool
	refintEnabled   bool
	usePwModifyExOp bool

	userBaseDN          string
	userFilter          string
	userObjectClass     string
	userIDisOctetString bool
	userScope           int
	userAttributeMap    userAttributeMap

	disableUserMechanism    DisableUserMechanismType
	localUserDisableGroupDN string

	groupBaseDN          string
	groupCreateBaseDN    string
	groupFilter          string
	groupObjectClass     string
	groupIDisOctetString bool
	groupScope           int
	groupAttributeMap    groupAttributeMap

	educationConfig educationConfig

	logger *log.Logger
	conn   ldap.Client
}

type userAttributeMap struct {
	displayName    string
	id             string
	mail           string
	userName       string
	givenName      string
	surname        string
	accountEnabled string
	userType       string
	identities     string
	lastSignIn     string
}

type ldapAttributeValues map[string][]string

type ldapResultToErrMap map[uint16]errorcode.Error

const ldapGenericErr = math.MaxUint16

// ParseDisableMechanismType checks that the configuration option for how to disable users is correct.
func ParseDisableMechanismType(disableMechanism string) (DisableUserMechanismType, error) {
	disableMechanism = strings.ToLower(disableMechanism)
	t, ok := mechanismMap[disableMechanism]
	if !ok {
		return -1, errors.New("invalid configuration option for disable user mechanism")
	}

	return t, nil
}

func NewLDAPBackend(lc ldap.Client, config config.LDAP, logger *log.Logger) (*LDAP, error) {
	if config.UserDisplayNameAttribute == "" || config.UserIDAttribute == "" ||
		config.UserEmailAttribute == "" || config.UserNameAttribute == "" {
		return nil, errors.New("invalid user attribute mappings")
	}
	uam := userAttributeMap{
		displayName:    config.UserDisplayNameAttribute,
		id:             config.UserIDAttribute,
		mail:           config.UserEmailAttribute,
		userName:       config.UserNameAttribute,
		accountEnabled: config.UserEnabledAttribute,
		givenName:      givenNameAttribute,
		surname:        surNameAttribute,
		userType:       config.UserTypeAttribute,
		identities:     identitiesAttribute,
		lastSignIn:     lastSignAttribute,
	}

	if config.GroupNameAttribute == "" || config.GroupIDAttribute == "" {
		return nil, errors.New("invalid group attribute mappings")
	}
	gam := groupAttributeMap{
		name:   config.GroupNameAttribute,
		id:     config.GroupIDAttribute,
		member: config.GroupMemberAttribute,
	}

	var userScope, groupScope int
	var err error
	if userScope, err = stringToScope(config.UserSearchScope); err != nil {
		return nil, fmt.Errorf("error configuring user scope: %w", err)
	}

	if groupScope, err = stringToScope(config.GroupSearchScope); err != nil {
		return nil, fmt.Errorf("error configuring group scope: %w", err)
	}

	var educationConfig educationConfig
	if educationConfig, err = newEducationConfig(config); err != nil {
		return nil, fmt.Errorf("error setting up education resource config: %w", err)
	}

	disableMechanismType, err := ParseDisableMechanismType(config.DisableUserMechanism)
	if err != nil {
		return nil, fmt.Errorf("error configuring disable user mechanism: %w", err)
	}

	return &LDAP{
		useServerUUID:           config.UseServerUUID,
		usePwModifyExOp:         config.UsePasswordModExOp,
		userBaseDN:              config.UserBaseDN,
		userFilter:              config.UserFilter,
		userObjectClass:         config.UserObjectClass,
		userIDisOctetString:     config.UserIDIsOctetString,
		userScope:               userScope,
		userAttributeMap:        uam,
		groupBaseDN:             config.GroupBaseDN,
		groupCreateBaseDN:       config.GroupCreateBaseDN,
		groupFilter:             config.GroupFilter,
		groupObjectClass:        config.GroupObjectClass,
		groupIDisOctetString:    config.GroupIDIsOctetString,
		groupScope:              groupScope,
		groupAttributeMap:       gam,
		educationConfig:         educationConfig,
		disableUserMechanism:    disableMechanismType,
		localUserDisableGroupDN: config.LdapDisabledUsersGroupDN,
		logger:                  logger,
		conn:                    lc,
		writeEnabled:            config.WriteEnabled,
		refintEnabled:           config.RefintEnabled,
	}, nil
}

// CreateUser implements the Backend Interface. It converts the libregraph.User into an
// LDAP User Entry (using the inetOrgPerson LDAP Objectclass) add adds that to the
// configured LDAP server
func (i *LDAP) CreateUser(ctx context.Context, user libregraph.User) (*libregraph.User, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("CreateUser")
	if !i.writeEnabled {
		return nil, ErrReadOnly
	}

	if user.AccountEnabled != nil && i.disableUserMechanism == DisableMechanismNone {
		return nil, errors.New("accountEnabled option not compatible with backend disable user mechanism")
	}

	ar, err := i.userToAddRequest(user)
	if err != nil {
		return nil, err
	}

	if err := i.conn.Add(ar); err != nil {
		msg := "failed to add user"
		logger.Error().Err(err).Msg(msg)
		errMap := ldapResultToErrMap{
			ldap.LDAPResultEntryAlreadyExists:       errorcode.New(errorcode.NameAlreadyExists, "a user with that name already exists"),
			ldap.LDAPResultUnwillingToPerform:       errorcode.New(errorcode.NotAllowed, msg),
			ldap.LDAPResultInsufficientAccessRights: errorcode.New(errorcode.NotAllowed, msg),
			ldapGenericErr:                          errorcode.New(errorcode.GeneralException, msg),
		}
		return nil, i.mapLDAPError(err, errMap)
	}

	if i.usePwModifyExOp && user.PasswordProfile != nil && user.PasswordProfile.Password != nil {
		if err := i.updateUserPassword(ctx, ar.DN, user.PasswordProfile.GetPassword()); err != nil {
			return nil, err
		}
	}

	// Read	back user from LDAP to get the generated UUID
	e, err := i.getUserByDN(ar.DN, "")
	if err != nil {
		return nil, err
	}
	return i.createUserModelFromLDAP(e), nil
}

// DeleteUser implements the Backend Interface. It permanently deletes a User identified
// by name or id from the LDAP server
func (i *LDAP) DeleteUser(ctx context.Context, nameOrID string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("DeleteUser")
	if !i.writeEnabled {
		return ErrReadOnly
	}
	e, err := i.getLDAPUserByNameOrID(nameOrID)
	if err != nil {
		return err
	}
	dr := ldap.DelRequest{DN: e.DN}
	if err = i.conn.Del(&dr); err != nil {
		msg := "error deleting user"
		logger.Error().Err(err).Msg(msg)
		errMap := ldapResultToErrMap{
			ldap.LDAPResultNoSuchObject:             errorcode.New(errorcode.ItemNotFound, "user not found"),
			ldap.LDAPResultUnwillingToPerform:       errorcode.New(errorcode.NotAllowed, msg),
			ldap.LDAPResultInsufficientAccessRights: errorcode.New(errorcode.NotAllowed, msg),
			ldapGenericErr:                          errorcode.New(errorcode.GeneralException, msg),
		}
		return i.mapLDAPError(err, errMap)
	}

	if !i.refintEnabled {
		// Find all the groups that this user was a member of and remove it from there
		groupEntries, err := i.getLDAPGroupsByFilter(fmt.Sprintf("(%s=%s)", i.groupAttributeMap.member, e.DN), true, false)
		if err != nil {
			return err
		}
		for _, group := range groupEntries {
			logger.Debug().Str("group", group.DN).Str("user", e.DN).Msg("Cleaning up group membership")

			if err := i.removeEntryByDNAndAttributeFromEntry(group, e.DN, i.groupAttributeMap.member); err != nil {
				// Errors when deleting the memberships are only logged as warnings but not returned
				// to the user as we already successfully deleted the users itself
				logger.Warn().Str("group", group.DN).Str("user", e.DN).Err(err).Msg("failed to remove member")
			}
		}
	}
	return nil
}

// UpdateUser implements the Backend Interface for the LDAP Backend
func (i *LDAP) UpdateUser(ctx context.Context, nameOrID string, user libregraph.UserUpdate) (*libregraph.User, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("UpdateUser")
	if !i.writeEnabled {
		// still allow to enable/disable user when using DisableMechanismGroup
		if i.disableUserMechanism == DisableMechanismGroup && isUserEnabledUpdate(user) {
			logger.Error().Str("backend", "ldap").Msg("Allowing accountEnabled Update on read-only backend")
		} else {
			return nil, ErrReadOnly
		}
	}
	e, err := i.getLDAPUserByNameOrID(nameOrID)
	if err != nil {
		return nil, err
	}

	var updateNeeded bool

	// Don't allow updates of the ID
	if user.GetId() != "" {
		id, err := i.ldapUUIDtoString(e, i.userAttributeMap.id, i.userIDisOctetString)
		if err != nil {
			i.logger.Warn().Str("dn", e.DN).Str(i.userAttributeMap.id, e.GetEqualFoldAttributeValue(i.userAttributeMap.id)).Msg("Invalid User. Cannot convert UUID")
			return nil, errorcode.New(errorcode.GeneralException, "error converting uuid")
		}
		if id != user.GetId() {
			return nil, errorcode.New(errorcode.NotAllowed, "changing the UserId is not allowed")
		}
	}
	if user.GetOnPremisesSamAccountName() != "" {
		if eu := e.GetEqualFoldAttributeValue(i.userAttributeMap.userName); eu != user.GetOnPremisesSamAccountName() {
			e, err = i.changeUserName(ctx, e.DN, eu, user.GetOnPremisesSamAccountName())
			if err != nil {
				return nil, err
			}
		}
	}

	mr := ldap.ModifyRequest{DN: e.DN}
	properties := map[string]string{
		i.userAttributeMap.displayName: user.GetDisplayName(),
		i.userAttributeMap.mail:        user.GetMail(),
		i.userAttributeMap.surname:     user.GetSurname(),
		i.userAttributeMap.givenName:   user.GetGivenName(),
		i.userAttributeMap.userType:    user.GetUserType(),
	}

	for attribute, value := range properties {
		if value != "" {
			if e.GetEqualFoldAttributeValue(attribute) != value {
				mr.Replace(attribute, []string{value})
				updateNeeded = true
			}
		}
	}

	if user.PasswordProfile != nil && user.PasswordProfile.GetPassword() != "" {
		if i.usePwModifyExOp {
			if err := i.updateUserPassword(ctx, e.DN, user.PasswordProfile.GetPassword()); err != nil {
				msg := "error updating user password"
				logger.Error().Err(err).Msg(msg)
				errMap := ldapResultToErrMap{
					ldap.LDAPResultNoSuchObject:             errorcode.New(errorcode.ItemNotFound, "user not found"),
					ldap.LDAPResultUnwillingToPerform:       errorcode.New(errorcode.NotAllowed, msg),
					ldap.LDAPResultInsufficientAccessRights: errorcode.New(errorcode.NotAllowed, msg),
					ldapGenericErr:                          errorcode.New(errorcode.GeneralException, msg),
				}
				return nil, i.mapLDAPError(err, errMap)
			}
		} else {
			// password are hashed server side there is no need to check if the new password
			// is actually different from the old one.
			mr.Replace("userPassword", []string{user.PasswordProfile.GetPassword()})
			updateNeeded = true
		}
	}

	if identities, ok := user.GetIdentitiesOk(); ok {
		attrValues := make([]string, 0, len(identities))
		for _, identity := range identities {
			identityStr, err := i.identityToLDAPAttrValue(identity)
			if err != nil {
				return nil, err
			}
			attrValues = append(attrValues, identityStr)
		}
		mr.Replace(i.userAttributeMap.identities, attrValues)
		updateNeeded = true
	}

	// Behavior of enabling/disabling of users depends on the "disableUserMechanism" config option:
	//
	// "attribute": For the upstream user management service which modifies accountEnabled on the user entry
	// "group": Makes it possible for local admins to disable users by adding them to a special group
	if user.AccountEnabled != nil {
		un, err := i.updateAccountEnabledState(logger, user.GetAccountEnabled(), e, &mr)

		if err != nil {
			return nil, err
		}

		if un {
			updateNeeded = true
		}
	}

	if updateNeeded {
		if err := i.conn.Modify(&mr); err != nil {
			msg := "error updating user"
			logger.Error().Err(err).Msg(msg)
			errMap := ldapResultToErrMap{
				ldap.LDAPResultNoSuchObject:             errorcode.New(errorcode.ItemNotFound, "user not found"),
				ldap.LDAPResultUnwillingToPerform:       errorcode.New(errorcode.NotAllowed, msg),
				ldap.LDAPResultInsufficientAccessRights: errorcode.New(errorcode.NotAllowed, msg),
				ldapGenericErr:                          errorcode.New(errorcode.GeneralException, msg),
			}
			return nil, i.mapLDAPError(err, errMap)
		}
	}

	// Read	back user from LDAP to get the generated UUID
	e, err = i.getUserByDN(e.DN, "")
	if err != nil {
		return nil, err
	}

	returnUser := i.createUserModelFromLDAP(e)

	// To avoid a ldap lookup for group membership, set the enabled flag to same as input value
	// since this would have been updated with group membership from the input anyway.
	if user.AccountEnabled != nil && i.disableUserMechanism == DisableMechanismGroup {
		returnUser.AccountEnabled = user.AccountEnabled
	}

	return returnUser, nil
}

func (i *LDAP) getUserByDN(dn, searchTerm string) (*ldap.Entry, error) {
	baseFilter := fmt.Sprintf("(objectClass=%s)", i.userObjectClass)

	userFilter := ""
	if i.userFilter != "" {
		userFilter = fmt.Sprintf("(%s)", i.userFilter)
	}
	searchFilter := ""
	if searchTerm != "" {
		searchTerm = ldap.EscapeFilter(searchTerm)
		searchFilter = fmt.Sprintf(
			"(|(%s=*%s*)(%s=*%s*)(%s=*%s*))",
			i.userAttributeMap.userName, searchTerm,
			i.userAttributeMap.mail, searchTerm,
			i.userAttributeMap.displayName, searchTerm,
		)
	}

	filter := baseFilter
	if userFilter != "" || searchFilter != "" {
		filter = fmt.Sprintf("(&%s%s%s)", baseFilter, userFilter, searchFilter)
	}

	return i.getEntryByDN(dn, i.getUserAttrTypesForSearch(), filter)
}

func (i *LDAP) getEntryByDN(dn string, attrs []string, filter string) (*ldap.Entry, error) {
	if filter == "" {
		filter = "(objectclass=*)"
	}

	searchRequest := ldap.NewSearchRequest(
		dn, ldap.ScopeBaseObject, ldap.NeverDerefAliases, 1, 0, false,
		filter,
		attrs,
		nil,
	)

	i.logger.Debug().Str("backend", "ldap").
		Str("base", searchRequest.BaseDN).
		Str("filter", searchRequest.Filter).
		Int("scope", searchRequest.Scope).
		Int("sizelimit", searchRequest.SizeLimit).
		Interface("attributes", searchRequest.Attributes).
		Msg("getEntryByDN")
	res, err := i.conn.Search(searchRequest)
	if err != nil {
		i.logger.Error().Err(err).Str("backend", "ldap").Str("dn", dn).Msg("Search ldap by DN failed")
		return nil, errorcode.New(errorcode.ItemNotFound, "user lookup failed")
	}
	if len(res.Entries) == 0 {
		return nil, ErrNotFound
	}

	return res.Entries[0], nil
}

func (i *LDAP) searchLDAPEntryByFilter(basedn string, attrs []string, filter string) (*ldap.Entry, error) {
	if filter == "" {
		filter = "(objectclass=*)"
	}

	searchRequest := ldap.NewSearchRequest(
		basedn,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases, 1, 0, false,
		filter,
		attrs,
		nil,
	)

	i.logger.Debug().Str("backend", "ldap").
		Str("base", searchRequest.BaseDN).
		Str("filter", searchRequest.Filter).
		Int("scope", searchRequest.Scope).
		Int("sizelimit", searchRequest.SizeLimit).
		Interface("attributes", searchRequest.Attributes).
		Msg("getEntryByFilter")
	res, err := i.conn.Search(searchRequest)
	if err != nil {
		i.logger.Error().Err(err).Str("backend", "ldap").Str("dn", basedn).Str("filter", filter).Msg("Search user by filter failed")
		return nil, errorcode.New(errorcode.ItemNotFound, "user search failed")
	}
	if len(res.Entries) == 0 {
		return nil, ErrNotFound
	}

	return res.Entries[0], nil
}

func filterEscapeUUID(binary bool, id string) (string, error) {
	var escaped string
	if binary {
		pid, err := uuid.Parse(id)
		if err != nil {
			err := fmt.Errorf("error parsing id '%s' as UUID: %w", id, err)
			return "", err
		}
		for _, b := range pid {
			escaped = fmt.Sprintf("%s\\%02x", escaped, b)
		}
	} else {
		escaped = ldap.EscapeFilter(id)
	}
	return escaped, nil
}

func (i *LDAP) getLDAPUserByID(id string) (*ldap.Entry, error) {
	idString, err := filterEscapeUUID(i.userIDisOctetString, id)
	if err != nil {
		return nil, fmt.Errorf("invalid User id: %w", err)
	}
	filter := fmt.Sprintf("(%s=%s)", i.userAttributeMap.id, idString)
	return i.getLDAPUserByFilter(filter)
}

func (i *LDAP) getLDAPUserByNameOrID(nameOrID string) (*ldap.Entry, error) {
	idString, err := filterEscapeUUID(i.userIDisOctetString, nameOrID)
	// err != nil just means that this is not an uuid, so we can skip the uuid filter part
	// and just filter by name
	var filter string
	if err == nil {
		filter = fmt.Sprintf("(|(%s=%s)(%s=%s))", i.userAttributeMap.userName, ldap.EscapeFilter(nameOrID), i.userAttributeMap.id, idString)
	} else {
		filter = fmt.Sprintf("(%s=%s)", i.userAttributeMap.userName, ldap.EscapeFilter(nameOrID))
	}

	return i.getLDAPUserByFilter(filter)
}

func (i *LDAP) getLDAPUserByFilter(filter string) (*ldap.Entry, error) {
	filter = fmt.Sprintf("(&%s(objectClass=%s)%s)", i.userFilter, i.userObjectClass, filter)
	return i.searchLDAPEntryByFilter(i.userBaseDN, i.getUserAttrTypesForSearch(), filter)
}

// GetUser implements the Backend Interface.
func (i *LDAP) GetUser(ctx context.Context, nameOrID string, oreq *godata.GoDataRequest) (*libregraph.User, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("GetUser")

	e, err := i.getLDAPUserByNameOrID(nameOrID)
	if err != nil {
		return nil, err
	}

	u := i.createUserModelFromLDAP(e)
	if u == nil {
		return nil, ErrNotFound
	}

	if i.disableUserMechanism != DisableMechanismNone {
		userEnabled, err := i.UserEnabled(e)
		if err == nil {
			u.AccountEnabled = &userEnabled
		}
	}

	exp, err := odata.GetExpandValues(oreq.Query)
	if err != nil {
		return nil, err
	}

	if slices.Contains(exp, "memberOf") {
		userGroups, err := i.getGroupsForUser(e.DN)
		if err != nil {
			return nil, err
		}
		u.MemberOf = i.groupsFromLDAPEntries(userGroups)
	}
	return u, nil
}

// GetUsers implements the Backend Interface.
func (i *LDAP) GetUsers(ctx context.Context, oreq *godata.GoDataRequest) ([]*libregraph.User, error) {
	return i.FilterUsers(ctx, oreq, nil)
}

// FilterUsers implements the Backend Interface.
func (i *LDAP) FilterUsers(ctx context.Context, oreq *godata.GoDataRequest, filter *godata.ParseNode) ([]*libregraph.User, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("GetUsers")

	queryFilter, err := i.oDataFilterToLDAPFilter(filter)
	if err != nil {
		return nil, err
	}

	search, err := odata.GetSearchValues(oreq.Query)
	if err != nil {
		return nil, err
	}

	exp, err := odata.GetExpandValues(oreq.Query)
	if err != nil {
		return nil, err
	}

	var userFilter string
	if search != "" {
		search = ldap.EscapeFilter(search)
		userFilter = fmt.Sprintf(
			"(|(%s=*%s*)(%s=*%s*)(%s=*%s*))",
			i.userAttributeMap.userName, search,
			i.userAttributeMap.mail, search,
			i.userAttributeMap.displayName, search,
		)
	}
	userFilter = fmt.Sprintf("(&%s(objectClass=%s)%s%s)", i.userFilter, i.userObjectClass, queryFilter, userFilter)
	searchRequest := ldap.NewSearchRequest(
		i.userBaseDN, i.userScope, ldap.NeverDerefAliases, 0, 0, false,
		userFilter,
		i.getUserAttrTypesForSearch(),
		nil,
	)
	logger.Debug().Str("backend", "ldap").
		Str("base", searchRequest.BaseDN).
		Str("filter", searchRequest.Filter).
		Int("scope", searchRequest.Scope).
		Int("sizelimit", searchRequest.SizeLimit).
		Interface("attributes", searchRequest.Attributes).
		Msg("GetUsers")
	res, err := i.conn.Search(searchRequest)
	if err != nil {
		msg := "error listing users"
		logger.Error().Err(err).Msg(msg)
		errMap := ldapResultToErrMap{
			ldap.LDAPResultInsufficientAccessRights: errorcode.New(errorcode.AccessDenied, msg),
			ldapGenericErr:                          errorcode.New(errorcode.GeneralException, msg),
		}
		return nil, i.mapLDAPError(err, errMap)
	}

	return i.usersFromLDAPEntries(res.Entries, exp)
}

func (i *LDAP) usersFromLDAPEntries(entries []*ldap.Entry, exp []string) ([]*libregraph.User, error) {
	usersEnabledState, err := i.usersEnabledState(entries)
	if err != nil {
		return nil, err
	}
	users := make([]*libregraph.User, 0, len(entries))
	for _, e := range entries {
		u := i.createUserModelFromLDAP(e)
		// Skip invalid LDAP users
		if u == nil {
			continue
		}

		if i.disableUserMechanism != DisableMechanismNone {
			userEnabled := usersEnabledState[e.DN]
			u.AccountEnabled = &userEnabled
		}

		if slices.Contains(exp, "memberOf") {
			userGroups, err := i.getGroupsForUser(e.DN)
			if err != nil {
				return nil, err
			}
			u.MemberOf = i.groupsFromLDAPEntries(userGroups)
		}
		users = append(users, u)
	}
	return users, nil
}

// UpdateLastSignInDate implements the Backend Interface.
func (i *LDAP) UpdateLastSignInDate(ctx context.Context, userID string, timestamp time.Time) error {
	if !i.writeEnabled {
		i.logger.Debug().Str("backend", "ldap").Msg("The LDAP Server is readonly. Skipping update of last sign in date")
		return nil
	}
	e, err := i.getLDAPUserByID(userID)
	switch {
	case errors.Is(err, ErrNotFound):
		i.logger.Warn().Err(err).Str("userID", userID).Msg("Failed to update last sign in date for user")
		return nil
	case err != nil:
		return err
	}

	mr := ldap.ModifyRequest{DN: e.DN}
	mr.Replace(lastSignAttribute, []string{timestamp.UTC().Format(ldapDateFormat)})
	if err := i.conn.Modify(&mr); err != nil {
		msg := "error updating last sign in date for user"
		i.logger.Error().Err(err).Str("userid", userID).Msg(msg)
		errMap := ldapResultToErrMap{
			ldap.LDAPResultNoSuchObject:             errorcode.New(errorcode.ItemNotFound, msg),
			ldap.LDAPResultUnwillingToPerform:       errorcode.New(errorcode.NotAllowed, msg),
			ldap.LDAPResultInsufficientAccessRights: errorcode.New(errorcode.NotAllowed, msg),
			ldapGenericErr:                          errorcode.New(errorcode.GeneralException, msg),
		}
		return i.mapLDAPError(err, errMap)
	}

	return nil
}

func (i *LDAP) changeUserName(ctx context.Context, dn, originalUserName, newUserName string) (*ldap.Entry, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)

	attributeTypeAndValue := ldap.AttributeTypeAndValue{
		Type:  i.userAttributeMap.userName,
		Value: newUserName,
	}
	newDNString := attributeTypeAndValue.String()

	logger.Debug().Str("originalDN", dn).Str("newDN", newDNString).Msg("Modifying DN")
	mrdn := ldap.NewModifyDNRequest(dn, newDNString, true, "")

	if err := i.conn.ModifyDN(mrdn); err != nil {
		msg := "error renaming user"
		logger.Error().Err(err).Msg(msg)
		errMap := ldapResultToErrMap{
			ldap.LDAPResultEntryAlreadyExists:       errorcode.New(errorcode.NameAlreadyExists, "a user with that name already exists"),
			ldap.LDAPResultNoSuchObject:             errorcode.New(errorcode.ItemNotFound, msg),
			ldap.LDAPResultUnwillingToPerform:       errorcode.New(errorcode.NotAllowed, msg),
			ldap.LDAPResultInsufficientAccessRights: errorcode.New(errorcode.NotAllowed, msg),
			ldapGenericErr:                          errorcode.New(errorcode.GeneralException, msg),
		}
		return nil, i.mapLDAPError(err, errMap)
	}

	parsed, err := ldap.ParseDN(dn)
	if err != nil {
		return nil, err
	}

	newFullDN, err := replaceDN(parsed, newDNString)
	if err != nil {
		return nil, err
	}

	u, err := i.getUserByDN(newFullDN, "")
	if err != nil {
		return nil, err
	}

	if !i.refintEnabled {
		groups, err := i.getGroupsForUser(dn)
		if err != nil {
			return nil, err
		}
		for _, g := range groups {
			logger.Debug().Str("originalDN", dn).Str("newDN", u.DN).Str("group", g.DN).Msg("Changing member in group")
			err = i.renameMemberInGroup(ctx, g, dn, u.DN)
			// This could leave the groups in an inconsistent state, might be a good idea to
			// add a defer that changes everything back on error. Ideally, this entire function
			// should be atomic, but LDAP doesn't support that.
			if err != nil {
				return nil, err
			}
		}
	}

	return u, nil
}

func (i *LDAP) renameMemberInGroup(ctx context.Context, group *ldap.Entry, oldMember, newMember string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("oldMember", oldMember).Str("newMember", newMember).Msg("replacing group member")
	mr := ldap.NewModifyRequest(group.DN, nil)
	mr.Delete(i.groupAttributeMap.member, []string{oldMember})
	mr.Add(i.groupAttributeMap.member, []string{newMember})
	if err := i.conn.Modify(mr); err != nil {
		logger.Warn().Err(err).
			Str("oldMember", oldMember).
			Str("newMember", newMember).
			Str("groupDN", group.DN).
			Msg("Error renaming group members.")
		var lerr *ldap.Error
		if errors.As(err, &lerr) {
			// NoSuchObject means that the group no longer exists. Some other client might have deleted it. There is
			// not much we can do
			// NoSuchAttribute means that the old value is no longer present. We can't do much here either
			if lerr.ResultCode == ldap.LDAPResultNoSuchObject || lerr.ResultCode == ldap.LDAPResultNoSuchAttribute {
				return nil
			}
		}
		return err
	}
	return nil
}

func (i *LDAP) updateUserPassword(ctx context.Context, dn, password string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("updateUserPassword")
	pwMod := ldap.PasswordModifyRequest{
		UserIdentity: dn,
		NewPassword:  password,
	}
	// Note: We can ignore the result message here, as it were only relevant if we requested
	// the server to generate a new Password
	_, err := i.conn.PasswordModify(&pwMod)
	if err != nil {
		var lerr *ldap.Error
		logger.Debug().Err(err).Msg("error setting password for user")
		if errors.As(err, &lerr) {
			if lerr.ResultCode == ldap.LDAPResultEntryAlreadyExists {
				err = errorcode.New(errorcode.NameAlreadyExists, lerr.Error())
			}
		}
	}
	return err
}

func (i *LDAP) ldapUUIDtoString(e *ldap.Entry, attrType string, binary bool) (string, error) {
	if binary {
		rawValue := e.GetEqualFoldRawAttributeValue(attrType)
		value, err := uuid.FromBytes(rawValue)
		if err == nil {
			return value.String(), nil
		}
		return "", err
	}
	return e.GetEqualFoldAttributeValue(attrType), nil
}

func (i *LDAP) createUserModelFromLDAP(e *ldap.Entry) *libregraph.User {
	if e == nil {
		return nil
	}

	opsan := e.GetEqualFoldAttributeValue(i.userAttributeMap.userName)
	id, err := i.ldapUUIDtoString(e, i.userAttributeMap.id, i.userIDisOctetString)
	if err != nil {
		i.logger.Warn().Str("dn", e.DN).Str(i.userAttributeMap.id, e.GetEqualFoldAttributeValue(i.userAttributeMap.id)).Msg("Invalid User. Cannot convert UUID")
	}
	surname := e.GetEqualFoldAttributeValue(i.userAttributeMap.surname)

	if id != "" && opsan != "" {
		user := &libregraph.User{
			DisplayName:              e.GetEqualFoldAttributeValue(i.userAttributeMap.displayName),
			Mail:                     pointerOrNil(e.GetEqualFoldAttributeValue(i.userAttributeMap.mail)),
			OnPremisesSamAccountName: opsan,
			Id:                       &id,
			GivenName:                pointerOrNil(e.GetEqualFoldAttributeValue(i.userAttributeMap.givenName)),
			Surname:                  &surname,
			AccountEnabled:           booleanOrNil(e.GetEqualFoldAttributeValue(i.userAttributeMap.accountEnabled)),
		}

		userType := e.GetEqualFoldAttributeValue(i.userAttributeMap.userType)
		if userType == "" {
			userType = UserTypeMember
		}
		user.SetUserType(userType)
		var identities []libregraph.ObjectIdentity
		for _, identityStr := range e.GetEqualFoldAttributeValues(i.userAttributeMap.identities) {
			parts := strings.SplitN(identityStr, "$", 3)
			identity := libregraph.NewObjectIdentity()
			identity.SetIssuer(strings.TrimSpace(parts[1]))
			identity.SetIssuerAssignedId(strings.TrimSpace(parts[2]))
			identities = append(identities, *identity)
		}
		if len(identities) > 0 {
			user.SetIdentities(identities)
		}

		lastSignIn, err := i.getLastSignTime(e)
		switch {
		case err == nil:
			signinActivity := libregraph.SignInActivity{
				LastSuccessfulSignInDateTime: lastSignIn,
			}
			user.SetSignInActivity(signinActivity)
		case !errors.Is(err, errNotSet):
			i.logger.Warn().Err(err).Str("dn", e.DN).Msg("Error getting last signin timestamp")
		}
		return user
	}
	i.logger.Warn().Str("dn", e.DN).Msg("Invalid User. Missing username or id attribute")
	return nil
}

func (i *LDAP) userToLDAPAttrValues(user libregraph.User) (map[string][]string, error) {
	attrs := map[string][]string{
		i.userAttributeMap.displayName: {user.GetDisplayName()},
		i.userAttributeMap.userName:    {user.GetOnPremisesSamAccountName()},
		i.userAttributeMap.mail:        {user.GetMail()},
		"objectClass":                  {"inetOrgPerson", "organizationalPerson", "person", "top", "openCloudUser"},
		"cn":                           {user.GetOnPremisesSamAccountName()},
		i.userAttributeMap.userType:    {user.GetUserType()},
	}

	if identities, ok := user.GetIdentitiesOk(); ok {
		for _, identity := range identities {
			identityStr, err := i.identityToLDAPAttrValue(identity)
			if err != nil {
				return nil, err
			}
			attrs[i.userAttributeMap.identities] = append(
				attrs[i.userAttributeMap.identities],
				identityStr,
			)
		}
	}

	if !i.useServerUUID {
		attrs[i.userAttributeMap.id] = []string{uuid.New().String()}
	}

	if user.AccountEnabled != nil {
		attrs[i.userAttributeMap.accountEnabled] = []string{
			strings.ToUpper(strconv.FormatBool(*user.AccountEnabled)),
		}
	}

	// inetOrgPerson requires "sn" to be set. Set it to the Username if
	// Surname is not set in the Request
	var sn string
	if user.Surname != nil && *user.Surname != "" {
		sn = *user.Surname
	} else {
		sn = user.OnPremisesSamAccountName
	}
	attrs[i.userAttributeMap.surname] = []string{sn}

	// When we get a givenName, we set the attribute.
	if givenName := user.GetGivenName(); givenName != "" {
		attrs[i.userAttributeMap.givenName] = []string{givenName}
	}

	if !i.usePwModifyExOp && user.PasswordProfile != nil && user.PasswordProfile.Password != nil {
		// Depending on the LDAP server implementation this might cause the
		// password to be stored in cleartext in the LDAP database. Using the
		// "Password Modify LDAP Extended Operation" is recommended.
		attrs["userPassword"] = []string{*user.PasswordProfile.Password}
	}
	return attrs, nil
}

func (i *LDAP) identityToLDAPAttrValue(identity libregraph.ObjectIdentity) (string, error) {
	// TODO add support for the "signInType" of objectIdentity
	if identity.GetIssuer() == "" || identity.GetIssuerAssignedId() == "" {
		return "", fmt.Errorf("missing Attribute for objectIdentity")
	}
	identityStr := fmt.Sprintf(" $ %s $ %s", identity.GetIssuer(), identity.GetIssuerAssignedId())
	return identityStr, nil
}

func (i *LDAP) getUserAttrTypesForSearch() []string {
	return []string{
		i.userAttributeMap.displayName,
		i.userAttributeMap.id,
		i.userAttributeMap.mail,
		i.userAttributeMap.userName,
		i.userAttributeMap.surname,
		i.userAttributeMap.givenName,
		i.userAttributeMap.accountEnabled,
		i.userAttributeMap.userType,
		i.userAttributeMap.identities,
		i.userAttributeMap.lastSignIn,
	}
}

func (i *LDAP) getUserAttrTypes() []string {
	return append(i.getUserAttrTypesForSearch(), "cn", "userPassword", "objectClass")
}

func (i *LDAP) getUserLDAPDN(user libregraph.User) string {
	attributeTypeAndValue := ldap.AttributeTypeAndValue{
		Type:  "uid",
		Value: user.OnPremisesSamAccountName,
	}
	return fmt.Sprintf("%s,%s", attributeTypeAndValue.String(), i.userBaseDN)
}

func (i *LDAP) userToAddRequest(user libregraph.User) (*ldap.AddRequest, error) {
	ar := ldap.NewAddRequest(i.getUserLDAPDN(user), nil)

	attrMap, err := i.userToLDAPAttrValues(user)
	if err != nil {
		return nil, err
	}

	for _, attrType := range i.getUserAttrTypes() {
		if values, ok := attrMap[attrType]; ok {
			ar.Attribute(attrType, values)
		}
	}
	return ar, nil
}

func pointerOrNil(val string) *string {
	if val == "" {
		return nil
	}
	return &val
}

func booleanOrNil(val string) *bool {
	boolValue, err := strconv.ParseBool(val)
	if err != nil {
		return nil
	}

	return &boolValue
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

// removeEntryByDNAndAttributeFromEntry creates a request to remove a single member entry by attribute and DN from a ldap entry
func (i *LDAP) removeEntryByDNAndAttributeFromEntry(entry *ldap.Entry, dn string, attribute string) error {
	nOldDN, err := ldapdn.ParseNormalize(dn)
	if err != nil {
		return err
	}

	currentValues := entry.GetEqualFoldAttributeValues(attribute)
	i.logger.Debug().Interface("members", currentValues).Msg("current values")
	found := false
	for _, currentValue := range currentValues {
		if currentValue == "" {
			continue
		}
		if normalizedCurrentValue, err := ldapdn.ParseNormalize(currentValue); err != nil {
			// We couldn't parse the entry value as a DN. Let's keep it
			// as it is but log a warning
			i.logger.Warn().Str("member", currentValue).Err(err).Msg("Couldn't parse DN")
			continue
		} else {
			if normalizedCurrentValue == nOldDN {
				found = true
			}
		}
	}
	if !found {
		i.logger.Error().Str("backend", "ldap").Str("entry", entry.DN).Str("target", dn).
			Msg("The target value is not present in the attribute list")
		return ErrNotFound
	}

	mr := &ldap.ModifyRequest{DN: entry.DN}
	if len(currentValues) == 1 {
		mr.Add(attribute, []string{""})
	}
	mr.Delete(attribute, []string{dn})

	err = i.conn.Modify(mr)
	var lerr *ldap.Error
	if err != nil && errors.As(err, &lerr) {
		if lerr.ResultCode == ldap.LDAPResultObjectClassViolation {
			// objectclass "groupOfName" requires at least one member to be present, some other go-routine
			// must have removed the 2nd last member from the group after we read the group. We adapt the
			// modification request to replace the last member with an empty member and re-try.
			i.logger.Debug().Err(err).
				Msg("Failed to remove last group member. Retrying once. Replacing last group member with an empty member value.")
			mr.Add(attribute, []string{""})
			err = i.conn.Modify(mr)
		}
	}

	if err != nil {
		i.logger.Error().Err(err).Str("entry", entry.DN).Str("attribute", attribute).Str("target value", dn).
			Msg("Failed to remove dn attribute from entry")
		msg := "failed to update object"
		errMap := ldapResultToErrMap{
			ldap.LDAPResultNoSuchObject:             errorcode.New(errorcode.ItemNotFound, "object does not exists"),
			ldap.LDAPResultUnwillingToPerform:       errorcode.New(errorcode.NotAllowed, msg),
			ldap.LDAPResultInsufficientAccessRights: errorcode.New(errorcode.NotAllowed, msg),
			ldapGenericErr:                          errorcode.New(errorcode.GeneralException, msg),
		}
		return i.mapLDAPError(err, errMap)
	}

	return nil
}

// expandLDAPAttributeEntries reads an attribute from a ldap entry and expands to users
func (i *LDAP) expandLDAPAttributeEntries(ctx context.Context, e *ldap.Entry, attribute, searchTerm string) ([]*ldap.Entry, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("ExpandLDAPAttributeEntries")
	result := []*ldap.Entry{}

	for _, entryDN := range e.GetEqualFoldAttributeValues(attribute) {
		if entryDN == "" {
			continue
		}
		logger.Debug().Str("entryDN", entryDN).Msg("lookup")
		ue, err := i.getUserByDN(entryDN, searchTerm)
		if err != nil {
			// Ignore errors when reading a specific entry fails, just log them and continue
			logger.Debug().Err(err).Str("entry", entryDN).Msg("error reading attribute member entry")
			continue
		}
		result = append(result, ue)
	}

	return result, nil
}

func replaceDN(fullDN *ldap.DN, newDN string) (string, error) {
	if len(fullDN.RDNs) == 0 {
		return "", fmt.Errorf("can't operate on an empty dn")
	}

	if len(fullDN.RDNs) == 1 {
		return newDN, nil
	}

	for _, part := range fullDN.RDNs[1:] {
		newDN += "," + part.String()
	}

	return newDN, nil
}

// CreateLDAPGroupByDN is a helper method specifically intended for creating a "system" group
// for managing locally disabled users on service startup
func (i *LDAP) CreateLDAPGroupByDN(dn string) error {
	ar := ldap.NewAddRequest(dn, nil)

	attrs := map[string][]string{
		"objectClass": {"groupOfNames", "top"},
		"member":      {""},
	}

	for attrType, values := range attrs {
		ar.Attribute(attrType, values)
	}

	return i.conn.Add(ar)
}

func (i *LDAP) addUserToDisableGroup(logger log.Logger, userDN string) (err error) {
	groupFilter := fmt.Sprintf("(objectClass=%s)", i.groupObjectClass)
	group, err := i.getEntryByDN(i.localUserDisableGroupDN, []string{i.groupAttributeMap.member}, groupFilter)

	if err != nil {
		return err
	}

	mr := ldap.ModifyRequest{DN: group.DN}
	mr.Add(i.groupAttributeMap.member, []string{userDN})

	err = i.conn.Modify(&mr)
	var lerr *ldap.Error
	if errors.As(err, &lerr) {
		// If the user is already in the group, just log a message and return
		if lerr.ResultCode == ldap.LDAPResultAttributeOrValueExists {
			logger.Info().Msg("User already in group for disabled users")
			return nil
		}
	}

	return err
}

func (i *LDAP) removeUserFromDisableGroup(logger log.Logger, userDN string) (err error) {
	groupFilter := fmt.Sprintf("(objectClass=%s)", i.groupObjectClass)
	group, err := i.getEntryByDN(i.localUserDisableGroupDN, []string{i.groupAttributeMap.member}, groupFilter)

	if err != nil {
		return err
	}

	mr := ldap.ModifyRequest{DN: group.DN}
	mr.Delete(i.groupAttributeMap.member, []string{userDN})

	err = i.conn.Modify(&mr)
	var lerr *ldap.Error
	if errors.As(err, &lerr) {
		// If the user is not in the group, just log a message and return
		if lerr.ResultCode == ldap.LDAPResultNoSuchAttribute {
			logger.Info().Msg("User was not in group for disabled users")
			return nil
		}
	}

	return err
}

func (i *LDAP) userEnabledByAttribute(user *ldap.Entry) bool {
	enabledAttribute := booleanOrNil(user.GetEqualFoldAttributeValue(i.userAttributeMap.accountEnabled))

	if enabledAttribute == nil {
		return true
	}

	return *enabledAttribute
}

func (i *LDAP) usersEnabledStateFromGroup(users []string) (usersEnabledState map[string]bool, err error) {
	groupFilter := fmt.Sprintf("(objectClass=%s)", i.groupObjectClass)
	group, err := i.getEntryByDN(i.localUserDisableGroupDN, []string{i.groupAttributeMap.member}, groupFilter)

	if err != nil {
		return nil, err
	}

	usersEnabledState = make(map[string]bool, len(users))
	for _, user := range users {
		usersEnabledState[user] = true
	}

	for _, memberDN := range group.GetEqualFoldAttributeValues(i.groupAttributeMap.member) {
		usersEnabledState[memberDN] = false
	}

	return usersEnabledState, err
}

// UserEnabled returns if a user is enabled. This can depend on a flag in the user entry or group membership
func (i *LDAP) UserEnabled(user *ldap.Entry) (bool, error) {
	usersEnabledState, err := i.usersEnabledState([]*ldap.Entry{user})

	if err != nil {
		return false, err
	}

	return usersEnabledState[user.DN], nil
}

func (i *LDAP) usersEnabledState(users []*ldap.Entry) (usersEnabledState map[string]bool, err error) {
	usersEnabledState = make(map[string]bool, len(users))
	keys := make([]string, len(users))
	for index, user := range users {
		usersEnabledState[user.DN] = true
		keys[index] = user.DN
	}

	switch i.disableUserMechanism {
	case DisableMechanismAttribute:
		for _, user := range users {
			usersEnabledState[user.DN] = i.userEnabledByAttribute(user)
		}

	case DisableMechanismGroup:
		userDisabledGroupState, err := i.usersEnabledStateFromGroup(keys)

		if err != nil {
			return nil, err
		}

		for _, user := range keys {
			usersEnabledState[user] = userDisabledGroupState[user]
		}
	}

	return usersEnabledState, nil
}

// Behavior of enabling/disabling of users depends on the "disableUserMechanism" config option:
//
// "attribute": For the upstream user management service which modifies accountEnabled on the user entry
// "group": Makes it possible for local admins to disable users by adding them to a special group
func (i *LDAP) updateAccountEnabledState(logger log.Logger, accountEnabled bool, e *ldap.Entry, mr *ldap.ModifyRequest) (updateNeeded bool, err error) {
	switch i.disableUserMechanism {
	case DisableMechanismNone:
		err = errors.New("accountEnabled option not compatible with backend disable user mechanism")
	case DisableMechanismAttribute:
		boolString := strings.ToUpper(strconv.FormatBool(accountEnabled))
		ldapValue := e.GetEqualFoldAttributeValue(i.userAttributeMap.accountEnabled)
		if ldapValue != "" {
			mr.Replace(i.userAttributeMap.accountEnabled, []string{boolString})
		} else {
			mr.Add(i.userAttributeMap.accountEnabled, []string{boolString})
		}
		updateNeeded = true
	case DisableMechanismGroup:
		if accountEnabled {
			err = i.removeUserFromDisableGroup(logger, e.DN)
		} else {
			err = i.addUserToDisableGroup(logger, e.DN)
		}
		updateNeeded = false
	}

	return updateNeeded, err
}

func (i *LDAP) mapLDAPError(err error, errmap ldapResultToErrMap) errorcode.Error {
	var lerr *ldap.Error
	if errors.As(err, &lerr) {
		if res, ok := errmap[lerr.ResultCode]; ok {
			return res
		}
	}
	if res, ok := errmap[ldapGenericErr]; ok {
		return res
	}
	return errorcode.New(errorcode.GeneralException, err.Error())
}

func (i *LDAP) getLastSignTime(e *ldap.Entry) (*time.Time, error) {
	dateString := e.GetEqualFoldAttributeValue(i.userAttributeMap.lastSignIn)
	if dateString == "" {
		return nil, errNotSet
	}
	t, err := time.Parse(ldapDateFormat, dateString)
	if err != nil {
		err = fmt.Errorf("error parsing LDAP date: '%s': %w", dateString, err)
		return nil, err
	}
	return &t, nil
}

func (i *LDAP) oDataFilterToLDAPFilter(filter *godata.ParseNode) (string, error) {
	if filter == nil {
		return "", nil
	}

	if filter.Token.Type != godata.ExpressionTokenLogical {
		return "", ErrUnsupportedFilter
	}

	if filter.Token.Value != "le" {
		return "", ErrUnsupportedFilter
	}

	if !isLastSuccessFullSignInDateTimeFilter(filter.Children[0]) {
		return "", ErrUnsupportedFilter
	}

	if filter.Children[1].Token.Type != godata.ExpressionTokenDateTime {
		return "", ErrUnsupportedFilter
	}
	parsed, err := time.Parse(time.RFC3339, filter.Children[1].Token.Value)
	if err != nil {
		return "", godata.BadRequestError("invalid date format")
	}

	ldapDateTime := parsed.UTC().Format(ldapDateFormat)
	return fmt.Sprintf("(%s<=%s)", i.userAttributeMap.lastSignIn, ldap.EscapeFilter(ldapDateTime)), nil
}

func isLastSuccessFullSignInDateTimeFilter(node *godata.ParseNode) bool {
	if node.Token.Type != godata.ExpressionTokenNav {
		return false
	}

	if len(node.Children) != 2 {
		return false
	}

	if node.Children[0].Token.Value != "signInActivity" {
		return false
	}

	if node.Children[1].Token.Value != "lastSuccessfulSignInDateTime" {
		return false
	}

	return true
}

func isUserEnabledUpdate(user libregraph.UserUpdate) bool {
	switch {
	case user.Id != nil, user.DisplayName != nil,
		user.Drive != nil, user.Mail != nil, user.OnPremisesSamAccountName != nil,
		user.PasswordProfile != nil, user.Surname != nil, user.GivenName != nil,
		user.UserType != nil:
		return false
	case len(user.AppRoleAssignments) != 0,
		len(user.MemberOf) != 0,
		len(user.Identities) != 0,
		len(user.Drives) != 0:
		return false
	}
	return user.AccountEnabled != nil
}
