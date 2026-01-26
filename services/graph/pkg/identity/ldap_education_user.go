package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-ldap/ldap/v3"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

type educationUserAttributeMap struct {
	primaryRole string
}

func newEducationUserAttributeMap() educationUserAttributeMap {
	return educationUserAttributeMap{
		primaryRole: "userClass",
	}
}

// CreateEducationUser creates a given education user in the identity backend.
func (i *LDAP) CreateEducationUser(ctx context.Context, user libregraph.EducationUser) (*libregraph.EducationUser, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("CreateEducationUser")
	if !i.writeEnabled {
		return nil, ErrReadOnly
	}

	ar, err := i.educationUserToAddRequest(user)
	if err != nil {
		return nil, err
	}

	if err := i.conn.Add(ar); err != nil {
		var lerr *ldap.Error
		logger.Debug().Err(err).Msg("error adding user")
		if errors.As(err, &lerr) {
			if lerr.ResultCode == ldap.LDAPResultEntryAlreadyExists {
				err = errorcode.New(errorcode.NameAlreadyExists, lerr.Error())
			}
		}
		return nil, err
	}

	// Read	back user from LDAP to get the generated UUID
	e, err := i.getEducationUserByDN(ar.DN)
	if err != nil {
		return nil, err
	}
	return i.createEducationUserModelFromLDAP(e), nil
}

// DeleteEducationUser deletes a given education user, identified by username or id, from the backend
func (i *LDAP) DeleteEducationUser(ctx context.Context, nameOrID string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("DeleteEducationUser")
	if !i.writeEnabled {
		return ErrReadOnly
	}
	// TODO, implement a proper lookup for education Users here
	e, err := i.getEducationUserByNameOrID(nameOrID)
	if err != nil {
		return err
	}

	dr := ldap.DelRequest{DN: e.DN}
	if err = i.conn.Del(&dr); err != nil {
		return err
	}
	return nil
}

// UpdateEducationUser applies changes to given education user, identified by username or id
func (i *LDAP) UpdateEducationUser(ctx context.Context, nameOrID string, user libregraph.EducationUser) (*libregraph.EducationUser, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("UpdateEducationUser")
	if !i.writeEnabled {
		return nil, ErrReadOnly
	}
	e, err := i.getEducationUserByNameOrID(nameOrID)
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
			e, err = i.getEducationUserByDN(e.DN)
			if err != nil {
				return nil, err
			}
		}
	}

	mr := ldap.ModifyRequest{DN: e.DN}
	properties := map[string]string{
		i.userAttributeMap.displayName:                 user.GetDisplayName(),
		i.userAttributeMap.mail:                        user.GetMail(),
		i.userAttributeMap.surname:                     user.GetSurname(),
		i.userAttributeMap.givenName:                   user.GetGivenName(),
		i.userAttributeMap.userType:                    user.GetUserType(),
		i.educationConfig.userAttributeMap.primaryRole: user.GetPrimaryRole(),
	}

	for attribute, value := range properties {
		if value != "" {
			if e.GetEqualFoldAttributeValue(attribute) != value {
				mr.Replace(attribute, []string{value})
				updateNeeded = true
			}
		}
	}

	if user.AccountEnabled != nil {
		un, err := i.updateAccountEnabledState(logger, user.GetAccountEnabled(), e, &mr)

		if err != nil {
			return nil, err
		}

		if un {
			updateNeeded = true
		}
	}
	if user.PasswordProfile != nil && user.PasswordProfile.GetPassword() != "" {
		if i.usePwModifyExOp {
			if err := i.updateUserPassword(ctx, e.DN, user.PasswordProfile.GetPassword()); err != nil {
				return nil, err
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

	if updateNeeded {
		if err := i.conn.Modify(&mr); err != nil {
			return nil, err
		}
	}

	// Read	back user from LDAP to get the generated UUID
	e, err = i.getEducationUserByDN(e.DN)
	if err != nil {
		return nil, err
	}

	returnUser := i.createEducationUserModelFromLDAP(e)

	// To avoid a ldap lookup for group membership, set the enabled flag to same as input value
	// since this would have been updated with group membership from the input anyway.
	if user.AccountEnabled != nil && i.disableUserMechanism == DisableMechanismGroup {
		returnUser.AccountEnabled = user.AccountEnabled
	}

	return returnUser, nil
}

// GetEducationUser implements the EducationBackend interface for the LDAP backend.
func (i *LDAP) GetEducationUser(ctx context.Context, nameOrID string) (*libregraph.EducationUser, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("GetEducationUser")
	e, err := i.getEducationUserByNameOrID(nameOrID)
	if err != nil {
		return nil, err
	}
	u := i.createEducationUserModelFromLDAP(e)
	if u == nil {
		return nil, ErrNotFound
	}
	return u, nil
}

// GetEducationUsers implements the EducationBackend interface for the LDAP backend.
func (i *LDAP) GetEducationUsers(ctx context.Context) ([]*libregraph.EducationUser, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("GetEducationUsers")

	var userFilter string

	if i.userFilter == "" {
		userFilter = fmt.Sprintf("(objectClass=%s)", i.educationConfig.userObjectClass)
	} else {
		userFilter = fmt.Sprintf("(&%s(objectClass=%s))", i.userFilter, i.educationConfig.userObjectClass)
	}

	searchRequest := ldap.NewSearchRequest(
		i.userBaseDN,
		i.userScope,
		ldap.NeverDerefAliases, 0, 0, false,
		userFilter,
		i.getEducationUserAttrTypes(),
		nil,
	)
	logger.Debug().Str("backend", "ldap").
		Str("base", searchRequest.BaseDN).
		Str("filter", searchRequest.Filter).
		Int("scope", searchRequest.Scope).
		Int("sizelimit", searchRequest.SizeLimit).
		Interface("attributes", searchRequest.Attributes).
		Msg("GetEducationUsers")
	res, err := i.conn.Search(searchRequest)
	if err != nil {
		return nil, errorcode.New(errorcode.ItemNotFound, err.Error())
	}

	users := make([]*libregraph.EducationUser, 0, len(res.Entries))

	for _, e := range res.Entries {
		u := i.createEducationUserModelFromLDAP(e)
		// Skip invalid LDAP users
		if u == nil {
			continue
		}
		users = append(users, u)
	}
	return users, nil
}

func (i *LDAP) educationUserToUser(eduUser libregraph.EducationUser) *libregraph.User {
	user := libregraph.NewUser(*eduUser.DisplayName, *eduUser.OnPremisesSamAccountName)
	user.Surname = eduUser.Surname
	user.AccountEnabled = eduUser.AccountEnabled
	user.GivenName = eduUser.GivenName
	user.Mail = eduUser.Mail
	user.UserType = eduUser.UserType
	user.Identities = eduUser.Identities

	return user
}

func (i *LDAP) userToEducationUser(user libregraph.User, e *ldap.Entry) *libregraph.EducationUser {
	eduUser := libregraph.NewEducationUser()
	eduUser.Id = user.Id
	eduUser.OnPremisesSamAccountName = &user.OnPremisesSamAccountName
	eduUser.Surname = user.Surname
	eduUser.AccountEnabled = user.AccountEnabled
	eduUser.GivenName = user.GivenName
	eduUser.DisplayName = &user.DisplayName
	eduUser.Mail = user.Mail
	eduUser.UserType = user.UserType

	if e != nil {
		// Set the education User specific Attributes from the supplied LDAP Entry
		if primaryRole := e.GetEqualFoldAttributeValue(i.educationConfig.userAttributeMap.primaryRole); primaryRole != "" {
			eduUser.SetPrimaryRole(primaryRole)
		}
	}

	return eduUser
}

func (i *LDAP) educationUserToLDAPAttrValues(user libregraph.EducationUser, attrs ldapAttributeValues) (ldapAttributeValues, error) {
	if role, ok := user.GetPrimaryRoleOk(); ok {
		attrs[i.educationConfig.userAttributeMap.primaryRole] = []string{*role}
	}
	attrs["objectClass"] = append(attrs["objectClass"], i.educationConfig.userObjectClass)
	return attrs, nil
}

func (i *LDAP) educationUserToAddRequest(user libregraph.EducationUser) (*ldap.AddRequest, error) {
	plainUser := i.educationUserToUser(user)
	ldapAttrs, err := i.userToLDAPAttrValues(*plainUser)
	if err != nil {
		return nil, err
	}
	ldapAttrs, err = i.educationUserToLDAPAttrValues(user, ldapAttrs)
	if err != nil {
		return nil, err
	}

	ar := ldap.NewAddRequest(i.getUserLDAPDN(*plainUser), nil)

	for attrType, values := range ldapAttrs {
		ar.Attribute(attrType, values)
	}
	return ar, nil
}

func (i *LDAP) createEducationUserModelFromLDAP(e *ldap.Entry) *libregraph.EducationUser {
	user := i.createUserModelFromLDAP(e)
	return i.userToEducationUser(*user, e)
}

func (i *LDAP) getEducationUserAttrTypes() []string {
	return []string{
		i.userAttributeMap.displayName,
		i.userAttributeMap.id,
		i.userAttributeMap.externalId,
		i.userAttributeMap.mail,
		i.userAttributeMap.userName,
		i.userAttributeMap.surname,
		i.userAttributeMap.givenName,
		i.userAttributeMap.accountEnabled,
		i.userAttributeMap.userType,
		i.userAttributeMap.identities,
		i.educationConfig.userAttributeMap.primaryRole,
		i.educationConfig.memberOfSchoolAttribute,
	}
}

func (i *LDAP) getEducationUserByDN(dn string) (*ldap.Entry, error) {
	filter := fmt.Sprintf("(objectClass=%s)", i.educationConfig.userObjectClass)

	if i.userFilter != "" {
		filter = fmt.Sprintf("(&%s(%s))", filter, i.userFilter)
	}

	return i.getEntryByDN(dn, i.getEducationUserAttrTypes(), filter)
}

func (i *LDAP) getEducationUserByNameOrID(nameOrID string) (*ldap.Entry, error) {
	return i.getEducationObjectByNameOrID(
		nameOrID,
		i.userAttributeMap.userName,
		i.userAttributeMap.id,
		i.userAttributeMap.externalId,
		i.userFilter,
		i.educationConfig.userObjectClass,
		i.userBaseDN,
		i.getEducationUserAttrTypes(),
	)
}

func (i *LDAP) getEducationObjectByNameOrID(nameOrID, nameAttribute, idAttribute, externalIdAttribute, objectFilter, objectClass, baseDN string, attributes []string) (*ldap.Entry, error) {
	nameOrID = ldap.EscapeFilter(nameOrID)
	filter := fmt.Sprintf("(|(%s=%s)(%s=%s)(%s=%s)", nameAttribute, nameOrID, idAttribute, nameOrID, externalIdAttribute, nameOrID)
	return i.getEducationObjectByFilter(filter, baseDN, objectFilter, objectClass, attributes)
}

func (i *LDAP) getEducationObjectByFilter(filter, baseDN, objectFilter, objectClass string, attributes []string) (*ldap.Entry, error) {
	filter = fmt.Sprintf("(&%s(objectClass=%s)%s)", objectFilter, objectClass, filter)
	return i.searchLDAPEntryByFilter(baseDN, attributes, filter)
}
