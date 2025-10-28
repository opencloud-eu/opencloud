package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/gofrs/uuid"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

type educationConfig struct {
	schoolBaseDN      string
	schoolFilter      string
	schoolObjectClass string
	schoolScope       int
	// memberOfSchoolAttribute defines the AttributeType on the user/group objects
	// which contains the school Id to which the user/group is assigned
	memberOfSchoolAttribute string
	schoolAttributeMap      schoolAttributeMap

	userObjectClass  string
	userAttributeMap educationUserAttributeMap

	classObjectClass  string
	classAttributeMap educationClassAttributeMap
}

type schoolAttributeMap struct {
	displayName     string
	schoolNumber    string
	id              string
	terminationDate string
}

type schoolUpdateOperation uint8

const (
	tooManyValues schoolUpdateOperation = iota
	schoolUnchanged
	schoolRenamed
	schoolPropertiesUpdated
)

var (
	errNotSet             = errors.New("attribute not set")
	errSchoolNameExists   = errorcode.New(errorcode.NameAlreadyExists, "A school with that name is already present")
	errSchoolNumberExists = errorcode.New(errorcode.NameAlreadyExists, "A school with that number is already present")
)

func defaultEducationConfig() educationConfig {
	return educationConfig{
		schoolObjectClass:       "openCloudEducationSchool",
		schoolScope:             ldap.ScopeWholeSubtree,
		memberOfSchoolAttribute: "openCloudMemberOfSchool",
		schoolAttributeMap:      newSchoolAttributeMap(),

		userObjectClass:  "openCloudEducationUser",
		userAttributeMap: newEducationUserAttributeMap(),

		classObjectClass:  "openCloudEducationClass",
		classAttributeMap: newEducationClassAttributeMap(),
	}
}

func newEducationConfig(config config.LDAP) (educationConfig, error) {
	if config.EducationResourcesEnabled {
		var err error
		eduCfg := defaultEducationConfig()
		eduCfg.schoolBaseDN = config.EducationConfig.SchoolBaseDN
		if config.EducationConfig.SchoolSearchScope != "" {
			if eduCfg.schoolScope, err = stringToScope(config.EducationConfig.SchoolSearchScope); err != nil {
				return educationConfig{}, fmt.Errorf("error configuring school search scope: %w", err)
			}
		}
		if config.EducationConfig.SchoolFilter != "" {
			eduCfg.schoolFilter = config.EducationConfig.SchoolFilter
		}
		if config.EducationConfig.SchoolObjectClass != "" {
			eduCfg.schoolObjectClass = config.EducationConfig.SchoolObjectClass
		}

		// Attribute mapping config
		if config.EducationConfig.SchoolNameAttribute != "" {
			eduCfg.schoolAttributeMap.displayName = config.EducationConfig.SchoolNameAttribute
		}
		if config.EducationConfig.SchoolNumberAttribute != "" {
			eduCfg.schoolAttributeMap.schoolNumber = config.EducationConfig.SchoolNumberAttribute
		}
		if config.EducationConfig.SchoolIDAttribute != "" {
			eduCfg.schoolAttributeMap.id = config.EducationConfig.SchoolIDAttribute
		}

		return eduCfg, nil
	}
	return educationConfig{}, nil
}

func newSchoolAttributeMap() schoolAttributeMap {
	return schoolAttributeMap{
		displayName:     "ou",
		schoolNumber:    "openCloudEducationSchoolNumber",
		id:              "openCloudUUID",
		terminationDate: "openCloudEducationSchoolTerminationTimestamp",
	}
}

// CreateEducationSchool creates the supplied school in the identity backend.
func (i *LDAP) CreateEducationSchool(ctx context.Context, school libregraph.EducationSchool) (*libregraph.EducationSchool, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("CreateEducationSchool")
	if !i.writeEnabled {
		return nil, ErrReadOnly
	}

	// Check that the school number is not already used
	if school.HasSchoolNumber() {
		_, err := i.getSchoolByNumber(school.GetSchoolNumber())
		switch err {
		case nil:
			logger.Debug().Err(errSchoolNumberExists).Str("schoolNumber", school.GetSchoolNumber()).Msg("duplicate school number")
			return nil, errSchoolNumberExists
		case ErrNotFound:
			break
		default:
			logger.Error().Err(err).Str("schoolNumber", school.GetSchoolNumber()).Msg("error looking up school by number")
			return nil, errorcode.New(errorcode.GeneralException, "error looking up school by number")
		}
	}

	attributeTypeAndValue := ldap.AttributeTypeAndValue{
		Type:  i.educationConfig.schoolAttributeMap.displayName,
		Value: school.GetDisplayName(),
	}

	dn := fmt.Sprintf("%s,%s",
		attributeTypeAndValue.String(),
		i.educationConfig.schoolBaseDN,
	)
	ar := ldap.NewAddRequest(dn, nil)
	ar.Attribute(i.educationConfig.schoolAttributeMap.displayName, []string{school.GetDisplayName()})
	if school.HasSchoolNumber() {
		ar.Attribute(i.educationConfig.schoolAttributeMap.schoolNumber, []string{school.GetSchoolNumber()})
	}
	if !i.useServerUUID {
		ar.Attribute(i.educationConfig.schoolAttributeMap.id, []string{uuid.Must(uuid.NewV4()).String()})
	}
	objectClasses := []string{"organizationalUnit", i.educationConfig.schoolObjectClass, "top"}
	ar.Attribute("objectClass", objectClasses)

	if err := i.conn.Add(ar); err != nil {
		var lerr *ldap.Error
		logger.Debug().Err(err).Msg("error adding school")
		if errors.As(err, &lerr) {
			if lerr.ResultCode == ldap.LDAPResultEntryAlreadyExists {
				err = errSchoolNameExists
			}
		}
		return nil, err
	}

	// Read	back school from LDAP to get the generated UUID
	e, err := i.getSchoolByDN(ar.DN)
	if err != nil {
		return nil, err
	}
	return i.createSchoolModelFromLDAP(e), nil
}

// UpdateEducationSchoolOperation contains the logic for which update operation to apply to a school
func (i *LDAP) updateEducationSchoolOperation(
	schoolUpdate libregraph.EducationSchool,
	currentSchool libregraph.EducationSchool,
) schoolUpdateOperation {

	providedDisplayName, displayNameIsSet := schoolUpdate.GetDisplayNameOk()
	if displayNameIsSet {
		if *providedDisplayName == "" || *providedDisplayName == currentSchool.GetDisplayName() {
			// The school name hasn't changed
			displayNameIsSet = false
		}
	}

	var propertiesUpdated bool

	switch {
	case schoolUpdate.HasSchoolNumber():
		if schoolUpdate.GetSchoolNumber() != "" && schoolUpdate.GetSchoolNumber() != currentSchool.GetSchoolNumber() {
			propertiesUpdated = true
		}
	case schoolUpdate.HasTerminationDate():
		if schoolUpdate.GetTerminationDate() != currentSchool.GetTerminationDate() {
			propertiesUpdated = true
		}
	}

	if propertiesUpdated && displayNameIsSet {
		return tooManyValues
	}

	if displayNameIsSet {
		return schoolRenamed
	}

	if propertiesUpdated {
		return schoolPropertiesUpdated
	}

	return schoolUnchanged
}

// updateDisplayName updates the school OU in the identity backend
func (i *LDAP) updateDisplayName(ctx context.Context, dn string, providedDisplayName string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	attributeTypeAndValue := ldap.AttributeTypeAndValue{
		Type:  i.educationConfig.schoolAttributeMap.displayName,
		Value: providedDisplayName,
	}

	mrdn := ldap.NewModifyDNRequest(dn, attributeTypeAndValue.String(), true, "")
	i.logger.Debug().Str("backend", "ldap").
		Str("dn", mrdn.DN).
		Str("newrdn", mrdn.NewRDN).
		Msg("updateDisplayName")

	if err := i.conn.ModifyDN(mrdn); err != nil {
		var lerr *ldap.Error
		logger.Debug().Err(err).Msg("error updating school name")
		if errors.As(err, &lerr) {
			if lerr.ResultCode == ldap.LDAPResultEntryAlreadyExists {
				err = errSchoolNameExists
			}
		}
		return err
	}

	return nil
}

// updateSchoolProperties updates the properties (other that displayName) of a school.
// It checks if a school number is already taken, before updating the school number
func (i *LDAP) updateSchoolProperties(ctx context.Context, dn string, currentSchool, updatedSchool libregraph.EducationSchool) error {
	logger := i.logger.SubloggerWithRequestID(ctx)

	mr := ldap.NewModifyRequest(dn, nil)
	if updatedSchoolNumber, ok := updatedSchool.GetSchoolNumberOk(); ok {
		if *updatedSchoolNumber != "" && currentSchool.GetSchoolNumber() != *updatedSchoolNumber {
			_, err := i.getSchoolByNumber(*updatedSchoolNumber)
			if err == nil {
				return errSchoolNumberExists
			}
			mr.Replace(i.educationConfig.schoolAttributeMap.schoolNumber, []string{*updatedSchoolNumber})
		}
	}

	if updatedTerminationDate, ok := updatedSchool.GetTerminationDateOk(); ok {
		if updatedTerminationDate == nil && currentSchool.HasTerminationDate() {
			// Delete the termination date
			mr.Delete(i.educationConfig.schoolAttributeMap.terminationDate, []string{})
		}
		if updatedTerminationDate != nil && *updatedTerminationDate != currentSchool.GetTerminationDate() {
			ldapDateTime := updatedTerminationDate.UTC().Format(ldapDateFormat)
			mr.Replace(i.educationConfig.schoolAttributeMap.terminationDate, []string{ldapDateTime})
		}
	}

	if err := i.conn.Modify(mr); err != nil {
		logger.Debug().Err(err).Msg("error updating school number")
		return err
	}

	return nil
}

// UpdateEducationSchool updates the supplied school in the identity backend
func (i *LDAP) UpdateEducationSchool(ctx context.Context, numberOrID string, school libregraph.EducationSchool) (*libregraph.EducationSchool, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("UpdateEducationSchool")
	if !i.writeEnabled {
		return nil, ErrReadOnly
	}

	e, err := i.getSchoolByNumberOrID(numberOrID)
	if err != nil {
		return nil, err
	}

	currentSchool := i.createSchoolModelFromLDAP(e)
	switch i.updateEducationSchoolOperation(school, *currentSchool) {
	case tooManyValues:
		return nil, fmt.Errorf("school name and school number cannot be updated in the same request")
	case schoolUnchanged:
		logger.Debug().Str("backend", "ldap").Msg("UpdateEducationSchool: Nothing changed")
		return currentSchool, nil
	case schoolRenamed:
		if err := i.updateDisplayName(ctx, e.DN, school.GetDisplayName()); err != nil {
			return nil, err
		}
	case schoolPropertiesUpdated:
		if err := i.updateSchoolProperties(ctx, e.DN, *currentSchool, school); err != nil {
			return nil, err
		}
	}

	// Read	back school from LDAP
	e, err = i.getSchoolByNumberOrID(i.getID(e))
	if err != nil {
		return nil, err
	}
	return i.createSchoolModelFromLDAP(e), nil
}

// DeleteEducationSchool deletes a given school, identified by id
func (i *LDAP) DeleteEducationSchool(ctx context.Context, id string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("DeleteEducationSchool")
	if !i.writeEnabled {
		return ErrReadOnly
	}
	e, err := i.getSchoolByNumberOrID(id)
	if err != nil {
		return err
	}

	dr := ldap.DelRequest{DN: e.DN}
	if err = i.conn.Del(&dr); err != nil {
		return err
	}

	// TODO update any users that are member of this school
	return nil
}

// GetEducationSchool implements the EducationBackend interface for the LDAP backend.
func (i *LDAP) GetEducationSchool(ctx context.Context, numberOrID string) (*libregraph.EducationSchool, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("GetEducationSchool")
	e, err := i.getSchoolByNumberOrID(numberOrID)
	if err != nil {
		return nil, err
	}

	return i.createSchoolModelFromLDAP(e), nil
}

// GetEducationSchools implements the EducationBackend interface for the LDAP backend.
func (i *LDAP) GetEducationSchools(ctx context.Context) ([]*libregraph.EducationSchool, error) {
	var filter string
	filter = fmt.Sprintf("(objectClass=%s)", i.educationConfig.schoolObjectClass)

	if i.educationConfig.schoolFilter != "" {
		filter = fmt.Sprintf("(&%s%s)", i.educationConfig.schoolFilter, filter)
	}

	searchRequest := ldap.NewSearchRequest(
		i.educationConfig.schoolBaseDN,
		i.educationConfig.schoolScope,
		ldap.NeverDerefAliases, 0, 0, false,
		filter,
		i.getEducationSchoolAttrTypes(),
		nil,
	)
	i.logger.Debug().Str("backend", "ldap").
		Str("base", searchRequest.BaseDN).
		Str("filter", searchRequest.Filter).
		Int("scope", searchRequest.Scope).
		Int("sizelimit", searchRequest.SizeLimit).
		Interface("attributes", searchRequest.Attributes).
		Msg("GetEducationSchools")
	res, err := i.conn.Search(searchRequest)
	if err != nil {
		return nil, errorcode.New(errorcode.ItemNotFound, err.Error())
	}

	schools := make([]*libregraph.EducationSchool, 0, len(res.Entries))
	for _, e := range res.Entries {
		school := i.createSchoolModelFromLDAP(e)
		// Skip invalid LDAP entries
		if school == nil {
			continue
		}
		schools = append(schools, school)
	}
	return schools, nil
}

// GetEducationSchoolUsers implements the EducationBackend interface for the LDAP backend.
func (i *LDAP) GetEducationSchoolUsers(ctx context.Context, schoolNumberOrID string) ([]*libregraph.EducationUser, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("GetEducationSchoolUsers")

	entries, err := i.getEducationSchoolEntries(
		schoolNumberOrID, i.userFilter, i.educationConfig.userObjectClass, i.userBaseDN, i.userScope, i.getEducationUserAttrTypes(), logger,
	)
	if err != nil {
		return nil, err
	}

	users := make([]*libregraph.EducationUser, 0, len(entries))

	for _, e := range entries {
		u := i.createEducationUserModelFromLDAP(e)
		// Skip invalid LDAP users
		if u == nil {
			continue
		}
		users = append(users, u)
	}
	return users, nil
}

// AddUsersToEducationSchool adds new members (reference by a slice of IDs) to supplied school in the identity backend.
func (i *LDAP) AddUsersToEducationSchool(ctx context.Context, schoolNumberOrID string, memberIDs []string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("AddUsersToEducationSchool")

	schoolEntry, err := i.getSchoolByNumberOrID(schoolNumberOrID)
	if err != nil {
		return err
	}

	if schoolEntry == nil {
		return ErrNotFound
	}

	schoolID := schoolEntry.GetEqualFoldAttributeValue(i.educationConfig.schoolAttributeMap.id)

	userEntries := make([]*ldap.Entry, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		user, err := i.getEducationUserByNameOrID(memberID)
		if err != nil {
			i.logger.Warn().Str("userid", memberID).Msg("User does not exist")
			return errorcode.New(errorcode.ItemNotFound, fmt.Sprintf("user '%s' not found", memberID))
		}
		userEntries = append(userEntries, user)
	}

	for _, userEntry := range userEntries {
		currentSchools := userEntry.GetEqualFoldAttributeValues(i.educationConfig.memberOfSchoolAttribute)
		found := false
		for _, currentSchool := range currentSchools {
			if currentSchool == schoolID {
				found = true
				break
			}
		}
		if !found {
			mr := ldap.ModifyRequest{DN: userEntry.DN}
			mr.Add(i.educationConfig.memberOfSchoolAttribute, []string{schoolID})
			if err := i.conn.Modify(&mr); err != nil {
				return err
			}
		}
	}

	return nil
}

// RemoveUserFromEducationSchool removes a single member (by ID) from a school
func (i *LDAP) RemoveUserFromEducationSchool(ctx context.Context, schoolNumberOrID string, memberID string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("RemoveUserFromEducationSchool")

	schoolEntry, err := i.getSchoolByNumberOrID(schoolNumberOrID)
	if err != nil {
		return err
	}

	if schoolEntry == nil {
		return ErrNotFound
	}

	schoolID := schoolEntry.GetEqualFoldAttributeValue(i.educationConfig.schoolAttributeMap.id)
	user, err := i.getEducationUserByNameOrID(memberID)
	if err != nil {
		i.logger.Warn().Str("userid", memberID).Msg("User does not exist")
		return err
	}
	currentSchools := user.GetEqualFoldAttributeValues(i.educationConfig.memberOfSchoolAttribute)
	for _, currentSchool := range currentSchools {
		if currentSchool == schoolID {
			mr := ldap.ModifyRequest{DN: user.DN}
			mr.Delete(i.educationConfig.memberOfSchoolAttribute, []string{schoolID})
			if err := i.conn.Modify(&mr); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

// GetEducationSchoolClasses implements the EducationBackend interface for the LDAP backend.
func (i *LDAP) GetEducationSchoolClasses(ctx context.Context, schoolNumberOrID string) ([]*libregraph.EducationClass, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("GetEducationSchoolClasses")

	entries, err := i.getEducationSchoolEntries(
		schoolNumberOrID, i.groupFilter, i.educationConfig.classObjectClass, i.groupBaseDN, i.groupScope, i.getEducationClassAttrTypes(false), logger,
	)
	if err != nil {
		return nil, err
	}

	classes := make([]*libregraph.EducationClass, 0, len(entries))

	for _, e := range entries {
		class := i.createEducationClassModelFromLDAP(e)
		// Skip invalid LDAP classes
		if class == nil {
			continue
		}
		classes = append(classes, class)
	}
	return classes, nil
}

func (i *LDAP) getEducationSchoolEntries(
	schoolNumberOrID, filter, objectClass, baseDN string,
	scope int,
	attributes []string,
	logger log.Logger,
) ([]*ldap.Entry, error) {
	schoolEntry, err := i.getSchoolByNumberOrID(schoolNumberOrID)
	if err != nil {
		return nil, err
	}

	if schoolEntry == nil {
		return nil, ErrNotFound
	}

	schoolID := schoolEntry.GetEqualFoldAttributeValue(i.educationConfig.schoolAttributeMap.id)
	schoolID = ldap.EscapeFilter(schoolID)
	idFilter := fmt.Sprintf("(%s=%s)", i.educationConfig.memberOfSchoolAttribute, schoolID)
	searchFilter := fmt.Sprintf("(&%s(objectClass=%s)%s)", filter, objectClass, idFilter)

	searchRequest := ldap.NewSearchRequest(
		baseDN,
		scope,
		ldap.NeverDerefAliases, 0, 0, false,
		searchFilter,
		attributes,
		nil,
	)
	logger.Debug().Str("backend", "ldap").
		Str("base", searchRequest.BaseDN).
		Str("filter", searchRequest.Filter).
		Int("scope", searchRequest.Scope).
		Int("sizelimit", searchRequest.SizeLimit).
		Interface("attributes", searchRequest.Attributes).
		Msg("GetEducationClasses")
	res, err := i.conn.Search(searchRequest)
	if err != nil {
		return nil, errorcode.New(errorcode.ItemNotFound, err.Error())
	}
	return res.Entries, nil
}

// AddClassesToEducationSchool adds new members (reference by a slice of IDs) to supplied school in the identity backend.
func (i *LDAP) AddClassesToEducationSchool(ctx context.Context, schoolNumberOrID string, memberIDs []string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("AddClassesToEducationSchool")

	schoolEntry, err := i.getSchoolByNumberOrID(schoolNumberOrID)
	if err != nil {
		return err
	}

	if schoolEntry == nil {
		return ErrNotFound
	}

	schoolID := schoolEntry.GetEqualFoldAttributeValue(i.educationConfig.schoolAttributeMap.id)

	classEntries := make([]*ldap.Entry, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		class, err := i.getEducationClassByID(memberID, false)
		if err != nil {
			i.logger.Warn().Str("userid", memberID).Msg("Class does not exist")
			return err
		}
		classEntries = append(classEntries, class)
	}

	for _, classEntry := range classEntries {
		currentSchools := classEntry.GetEqualFoldAttributeValues(i.educationConfig.memberOfSchoolAttribute)
		found := false
		for _, currentSchool := range currentSchools {
			if currentSchool == schoolID {
				found = true
				break
			}
		}
		if !found {
			mr := ldap.ModifyRequest{DN: classEntry.DN}
			mr.Add(i.educationConfig.memberOfSchoolAttribute, []string{schoolID})
			if err := i.conn.Modify(&mr); err != nil {
				return err
			}
		}
	}

	return nil
}

// RemoveClassFromEducationSchool removes a single member (by ID) from a school
func (i *LDAP) RemoveClassFromEducationSchool(ctx context.Context, schoolNumberOrID string, memberID string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("RemoveClassFromEducationSchool")

	schoolEntry, err := i.getSchoolByNumberOrID(schoolNumberOrID)
	if err != nil {
		return err
	}

	if schoolEntry == nil {
		return ErrNotFound
	}

	schoolID := schoolEntry.GetEqualFoldAttributeValue(i.educationConfig.schoolAttributeMap.id)
	class, err := i.getEducationClassByID(memberID, false)
	if err != nil {
		i.logger.Warn().Str("userid", memberID).Msg("Class does not exist")
		return err
	}
	currentSchools := class.GetEqualFoldAttributeValues(i.educationConfig.memberOfSchoolAttribute)
	for _, currentSchool := range currentSchools {
		if currentSchool == schoolID {
			mr := ldap.ModifyRequest{DN: class.DN}
			mr.Delete(i.educationConfig.memberOfSchoolAttribute, []string{schoolID})
			if err := i.conn.Modify(&mr); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (i *LDAP) getSchoolByDN(dn string) (*ldap.Entry, error) {
	filter := fmt.Sprintf("(objectClass=%s)", i.educationConfig.schoolObjectClass)

	if i.educationConfig.schoolFilter != "" {
		filter = fmt.Sprintf("(&%s(%s))", filter, i.educationConfig.schoolFilter)
	}
	return i.getEntryByDN(dn, i.getEducationSchoolAttrTypes(), filter)
}

func (i *LDAP) getSchoolByNumberOrID(numberOrID string) (*ldap.Entry, error) {
	numberOrID = ldap.EscapeFilter(numberOrID)
	filter := fmt.Sprintf(
		"(|(%s=%s)(%s=%s))",
		i.educationConfig.schoolAttributeMap.id,
		numberOrID,
		i.educationConfig.schoolAttributeMap.schoolNumber,
		numberOrID,
	)
	return i.getSchoolByFilter(filter)
}

func (i *LDAP) getSchoolByNumber(schoolNumber string) (*ldap.Entry, error) {
	schoolNumber = ldap.EscapeFilter(schoolNumber)
	filter := fmt.Sprintf(
		"(%s=%s)",
		i.educationConfig.schoolAttributeMap.schoolNumber,
		schoolNumber,
	)
	return i.getSchoolByFilter(filter)
}

func (i *LDAP) getSchoolByFilter(filter string) (*ldap.Entry, error) {
	filter = fmt.Sprintf("(&%s(objectClass=%s)%s)",
		i.educationConfig.schoolFilter,
		i.educationConfig.schoolObjectClass,
		filter,
	)
	searchRequest := ldap.NewSearchRequest(
		i.educationConfig.schoolBaseDN,
		i.educationConfig.schoolScope,
		ldap.NeverDerefAliases, 1, 0, false,
		filter,
		i.getEducationSchoolAttrTypes(),
		nil,
	)
	i.logger.Debug().Str("backend", "ldap").
		Str("base", searchRequest.BaseDN).
		Str("filter", searchRequest.Filter).
		Int("scope", searchRequest.Scope).
		Int("sizelimit", searchRequest.SizeLimit).
		Interface("attributes", searchRequest.Attributes).
		Msg("getSchoolByFilter")
	res, err := i.conn.Search(searchRequest)
	if err != nil {
		var errmsg string
		if lerr, ok := err.(*ldap.Error); ok {
			if lerr.ResultCode == ldap.LDAPResultSizeLimitExceeded {
				errmsg = fmt.Sprintf("too many results searching for school '%s'", filter)
				i.logger.Debug().Str("backend", "ldap").Err(lerr).
					Str("schoolfilter", filter).Msg("too many results searching for school")
			}
		}
		return nil, errorcode.New(errorcode.ItemNotFound, errmsg)
	}
	if len(res.Entries) == 0 {
		return nil, ErrNotFound
	}

	return res.Entries[0], nil
}

func (i *LDAP) createSchoolModelFromLDAP(e *ldap.Entry) *libregraph.EducationSchool {
	if e == nil {
		return nil
	}

	displayName := i.getDisplayName(e)
	id := i.getID(e)
	schoolNumber := i.getSchoolNumber(e)

	t, err := i.getTerminationDate(e)
	if err != nil && !errors.Is(err, errNotSet) {
		i.logger.Error().Err(err).Str("dn", e.DN).Msg("Error reading termination date for LDAP entry")
	}

	if id == "" || displayName == "" {
		i.logger.Warn().Str("dn", e.DN).Str("id", id).Str("displayName", displayName).Msg("Invalid School. Missing required attribute")
		return nil
	}

	school := libregraph.NewEducationSchool()
	school.SetDisplayName(displayName)
	school.SetId(id)
	if schoolNumber != "" {
		school.SetSchoolNumber(schoolNumber)
	}
	if t != nil {
		school.SetTerminationDate(*t)
	}
	return school
}

func (i *LDAP) getSchoolNumber(e *ldap.Entry) string {
	schoolNumber := e.GetEqualFoldAttributeValue(i.educationConfig.schoolAttributeMap.schoolNumber)
	return schoolNumber
}

func (i *LDAP) getID(e *ldap.Entry) string {
	id := e.GetEqualFoldAttributeValue(i.educationConfig.schoolAttributeMap.id)
	return id
}

func (i *LDAP) getDisplayName(e *ldap.Entry) string {
	displayName := e.GetEqualFoldAttributeValue(i.educationConfig.schoolAttributeMap.displayName)
	return displayName
}

func (i *LDAP) getTerminationDate(e *ldap.Entry) (*time.Time, error) {
	dateString := e.GetEqualFoldAttributeValue(i.educationConfig.schoolAttributeMap.terminationDate)
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

func (i *LDAP) getEducationSchoolAttrTypes() []string {
	return []string{
		i.educationConfig.schoolAttributeMap.displayName,
		i.educationConfig.schoolAttributeMap.id,
		i.educationConfig.schoolAttributeMap.schoolNumber,
		i.educationConfig.schoolAttributeMap.terminationDate,
	}
}
