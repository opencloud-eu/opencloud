package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"github.com/libregraph/idm/pkg/ldapdn"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

type educationClassAttributeMap struct {
	externalID     string
	classification string
	teachers       string
}

func newEducationClassAttributeMap() educationClassAttributeMap {
	return educationClassAttributeMap{
		externalID:     "openCloudEducationExternalId",
		classification: "openCloudEducationClassType",
		teachers:       "openCloudEducationTeacherMember",
	}
}

// GetEducationClasses implements the EducationBackend interface for the LDAP backend.
func (i *LDAP) GetEducationClasses(ctx context.Context) ([]*libregraph.EducationClass, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("GetEducationClasses")

	classFilter := fmt.Sprintf("(&%s(objectClass=%s))", i.groupFilter, i.educationConfig.classObjectClass)

	classAttrs := i.getEducationClassAttrTypes(false)

	searchRequest := ldap.NewSearchRequest(
		i.groupBaseDN, i.groupScope, ldap.NeverDerefAliases, 0, 0, false,
		classFilter,
		classAttrs,
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

	classes := make([]*libregraph.EducationClass, 0, len(res.Entries))

	var c *libregraph.EducationClass
	for _, e := range res.Entries {
		if c = i.createEducationClassModelFromLDAP(e); c == nil {
			continue
		}
		classes = append(classes, c)
	}
	return classes, nil
}

// CreateEducationClass implements the EducationBackend interface for the LDAP backend.
// An EducationClass is mapped to an LDAP entry of the "groupOfNames" structural ObjectClass.
// With a few additional Attributes added on top via the "openCloudEducationClass" auxiliary ObjectClass.
func (i *LDAP) CreateEducationClass(ctx context.Context, class libregraph.EducationClass) (*libregraph.EducationClass, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("create educationClass")
	if !i.writeEnabled {
		return nil, errorcode.New(errorcode.NotAllowed, "server is configured read-only")
	}
	ar, err := i.educationClassToAddRequest(class)
	if err != nil {
		return nil, err
	}

	if err := i.conn.Add(ar); err != nil {
		var lerr *ldap.Error
		logger.Debug().Err(err).Msg("error adding class")
		if errors.As(err, &lerr) {
			if lerr.ResultCode == ldap.LDAPResultEntryAlreadyExists {
				err = errorcode.New(errorcode.NameAlreadyExists, lerr.Error())
			}
		}
		return nil, err
	}

	// Read	back group from LDAP to get the generated UUID
	e, err := i.getEducationClassByDN(ar.DN)
	if err != nil {
		return nil, err
	}
	return i.createEducationClassModelFromLDAP(e), nil
}

// GetEducationClass implements the EducationBackend interface for the LDAP backend.
func (i *LDAP) GetEducationClass(ctx context.Context, id string) (*libregraph.EducationClass, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("GetEducationClass")
	e, err := i.getEducationClassByID(id, false)
	if err != nil {
		return nil, err
	}
	var class *libregraph.EducationClass
	if class = i.createEducationClassModelFromLDAP(e); class == nil {
		return nil, errorcode.New(errorcode.ItemNotFound, "not found")
	}
	return class, nil
}

// DeleteEducationClass implements the EducationBackend interface for the LDAP backend.
func (i *LDAP) DeleteEducationClass(ctx context.Context, id string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("DeleteEducationClass")
	if !i.writeEnabled {
		return ErrReadOnly
	}
	e, err := i.getEducationClassByID(id, false)
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

// UpdateEducationClass implements the EducationBackend interface for the LDAP backend.
// Only the displayName and externalID are supported to change at this point.
func (i *LDAP) UpdateEducationClass(ctx context.Context, id string, class libregraph.EducationClass) (*libregraph.EducationClass, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("UpdateEducationClass")
	if !i.writeEnabled {
		return nil, ErrReadOnly
	}

	g, err := i.getLDAPGroupByID(id, false)
	if err != nil {
		return nil, err
	}

	var updateNeeded bool

	if class.GetId() != "" {
		id, err := i.ldapUUIDtoString(g, i.groupAttributeMap.id, i.groupIDisOctetString)
		if err != nil {
			i.logger.Warn().Str("dn", g.DN).Str(i.userAttributeMap.id, g.GetEqualFoldAttributeValue(i.userAttributeMap.id)).Msg("Invalid class. Cannot convert UUID")
			return nil, errorcode.New(errorcode.GeneralException, "error converting uuid")
		}
		if id != class.GetId() {
			return nil, errorcode.New(errorcode.NotAllowed, "changing the GroupID is not allowed")
		}
	}

	if class.GetDescription() != "" {
		return nil, errorcode.New(errorcode.NotSupported, "changing the description is currently not supported")
	}

	if len(class.GetMembers()) != 0 {
		return nil, errorcode.New(errorcode.NotSupported, "changing the members is currently not supported")
	}

	if class.GetClassification() != "" {
		return nil, errorcode.New(errorcode.NotSupported, "changing the classification is currently not supported")
	}

	dn := g.DN

	if eID := class.GetExternalId(); eID != "" {
		if g.GetEqualFoldAttributeValue(i.educationConfig.classAttributeMap.externalID) != eID {
			dn, err = i.updateClassExternalID(ctx, dn, eID)
			if err != nil {
				return nil, err
			}
		}
	}

	mr := ldap.ModifyRequest{DN: dn}

	if dName := class.GetDisplayName(); dName != "" {
		if g.GetEqualFoldAttributeValue(i.groupAttributeMap.name) != dName {
			mr.Replace(i.groupAttributeMap.name, []string{dName})
			updateNeeded = true
		}
	}

	if updateNeeded {
		if err := i.conn.Modify(&mr); err != nil {
			return nil, err
		}
	}

	g, err = i.getEducationClassByDN(dn)
	if err != nil {
		return nil, err
	}

	return i.createEducationClassModelFromLDAP(g), nil
}

func (i *LDAP) updateClassExternalID(ctx context.Context, dn, externalID string) (string, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	newDN := fmt.Sprintf("openCloudEducationExternalId=%s", externalID)

	mrdn := ldap.NewModifyDNRequest(dn, newDN, true, "")
	i.logger.Debug().Str("Backend", "ldap").
		Str("dn", mrdn.DN).
		Str("newrdn", mrdn.NewRDN).
		Msg("updating class external ID")

	if err := i.conn.ModifyDN(mrdn); err != nil {
		var lerr *ldap.Error
		logger.Debug().Err(err).Msg("error updating class external ID")
		if errors.As(err, &lerr) {
			if lerr.ResultCode == ldap.LDAPResultEntryAlreadyExists {
				err = errorcode.New(errorcode.NameAlreadyExists, lerr.Error())
			}
		}
		return "", err
	}

	return fmt.Sprintf("%s,%s", newDN, i.groupBaseDN), nil
}

// GetEducationClassMembers implements the EducationBackend interface for the LDAP backend.
func (i *LDAP) GetEducationClassMembers(ctx context.Context, id string) ([]*libregraph.EducationUser, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "ldap").Msg("GetEducationClassMembers")
	e, err := i.getEducationClassByID(id, true)
	if err != nil {
		return nil, err
	}

	memberEntries, err := i.expandLDAPAttributeEntries(ctx, e, i.groupAttributeMap.member, "")
	result := make([]*libregraph.EducationUser, 0, len(memberEntries))
	if err != nil {
		return nil, err
	}
	for _, member := range memberEntries {
		if u := i.createEducationUserModelFromLDAP(member); u != nil {
			result = append(result, u)
		}
	}

	return result, nil
}

func (i *LDAP) educationClassToAddRequest(class libregraph.EducationClass) (*ldap.AddRequest, error) {
	plainGroup := i.educationClassToGroup(class)
	ldapAttrs, err := i.groupToLDAPAttrValues(*plainGroup)
	if err != nil {
		return nil, err
	}
	ldapAttrs, err = i.educationClassToLDAPAttrValues(class, ldapAttrs)
	if err != nil {
		return nil, err
	}

	ar := ldap.NewAddRequest(i.getEducationClassLDAPDN(class), nil)

	for attrType, values := range ldapAttrs {
		ar.Attribute(attrType, values)
	}
	return ar, nil
}

func (i *LDAP) educationClassToGroup(class libregraph.EducationClass) *libregraph.Group {
	group := libregraph.NewGroup()
	group.SetDisplayName(class.DisplayName)

	return group
}

func (i *LDAP) educationClassToLDAPAttrValues(class libregraph.EducationClass, attrs ldapAttributeValues) (ldapAttributeValues, error) {
	if externalID, ok := class.GetExternalIdOk(); ok {
		attrs[i.educationConfig.classAttributeMap.externalID] = []string{*externalID}
	}
	if classification, ok := class.GetClassificationOk(); ok {
		attrs[i.educationConfig.classAttributeMap.classification] = []string{*classification}
	}
	attrs["objectClass"] = append(attrs["objectClass"], i.educationConfig.classObjectClass)
	return attrs, nil
}

func (i *LDAP) getEducationClassAttrTypes(requestMembers bool) []string {
	attrs := []string{
		i.groupAttributeMap.name,
		i.groupAttributeMap.id,
		i.educationConfig.classAttributeMap.classification,
		i.educationConfig.classAttributeMap.externalID,
		i.educationConfig.memberOfSchoolAttribute,
		i.educationConfig.classAttributeMap.teachers,
	}
	if requestMembers {
		attrs = append(attrs, i.groupAttributeMap.member)
	}
	return attrs
}

func (i *LDAP) getEducationClassByDN(dn string) (*ldap.Entry, error) {
	filter := fmt.Sprintf("(objectClass=%s)", i.educationConfig.classObjectClass)

	if i.groupFilter != "" {
		filter = fmt.Sprintf("(&%s(%s))", filter, i.groupFilter)
	}

	return i.getEntryByDN(dn, i.getEducationClassAttrTypes(false), filter)
}

func (i *LDAP) createEducationClassModelFromLDAP(e *ldap.Entry) *libregraph.EducationClass {
	group := i.createGroupModelFromLDAP(e)
	return i.groupToEducationClass(*group, e)
}

func (i *LDAP) groupToEducationClass(group libregraph.Group, e *ldap.Entry) *libregraph.EducationClass {
	class := libregraph.NewEducationClass(group.GetDisplayName(), "")
	class.SetId(group.GetId())

	if e != nil {
		// Set the education User specific Attributes from the supplied LDAP Entry
		if externalID := e.GetEqualFoldAttributeValue(i.educationConfig.classAttributeMap.externalID); externalID != "" {
			class.SetExternalId(externalID)
		}
		if classification := e.GetEqualFoldAttributeValue(i.educationConfig.classAttributeMap.classification); classification != "" {
			class.SetClassification(classification)
		}
	}

	return class
}

func (i *LDAP) getEducationClassLDAPDN(class libregraph.EducationClass) string {
	attributeTypeAndValue := ldap.AttributeTypeAndValue{
		Type:  "openCloudEducationExternalId",
		Value: class.GetExternalId(),
	}
	return fmt.Sprintf("%s,%s", attributeTypeAndValue.String(), i.groupBaseDN)
}

func (i *LDAP) getEducationClassByID(nameOrID string, requestMembers bool) (*ldap.Entry, error) {
	return i.getEducationObjectByNameOrID(
		nameOrID,
		i.userAttributeMap.id,
		i.educationConfig.classAttributeMap.externalID,
		i.groupFilter,
		i.educationConfig.classObjectClass,
		i.groupBaseDN,
		i.getEducationClassAttrTypes(requestMembers),
	)
}

// GetEducationClassTeachers returns the EducationUser teachers for an EducationClass
func (i *LDAP) GetEducationClassTeachers(ctx context.Context, classID string) ([]*libregraph.EducationUser, error) {
	logger := i.logger.SubloggerWithRequestID(ctx)
	class, err := i.getEducationClassByID(classID, false)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get class: backend error")
		return nil, err
	}

	teacherEntries, err := i.expandLDAPAttributeEntries(ctx, class, i.educationConfig.classAttributeMap.teachers, "")
	result := make([]*libregraph.EducationUser, 0, len(teacherEntries))
	if err != nil {
		return nil, err
	}
	for _, teacher := range teacherEntries {
		if u := i.createEducationUserModelFromLDAP(teacher); u != nil {
			result = append(result, u)
		}
	}

	return result, nil

}

// AddTeacherToEducationClass adds a teacher (by ID) to class in the identity backend.
func (i *LDAP) AddTeacherToEducationClass(ctx context.Context, classID string, teacherID string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	class, err := i.getEducationClassByID(classID, false)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get class: backend error")
		return err
	}

	logger.Debug().Str("classDn", class.DN).Msg("got a class")
	teacher, err := i.getEducationUserByNameOrID(teacherID)

	if err != nil {
		logger.Debug().Err(err).Msg("could not get education user: error fetching education user from backend")
		return err
	}

	logger.Debug().Str("userDn", teacher.DN).Msg("got a user")

	mr := ldap.ModifyRequest{DN: class.DN}
	// Handle empty teacher list
	current := class.GetEqualFoldAttributeValues(i.educationConfig.classAttributeMap.teachers)
	if len(current) == 1 && current[0] == "" {
		mr.Delete(i.educationConfig.classAttributeMap.teachers, []string{""})
	}

	// Create a Set of current teachers
	currentSet := make(map[string]struct{}, len(current))
	for _, currentTeacher := range current {
		if currentTeacher == "" {
			continue
		}
		nCurrentTeacher, err := ldapdn.ParseNormalize(currentTeacher)
		if err != nil {
			// Couldn't parse teacher value as a DN, skipping
			logger.Warn().Str("teacherDN", currentTeacher).Err(err).Msg("Couldn't parse DN")
			continue
		}
		currentSet[nCurrentTeacher] = struct{}{}
	}

	var newTeacherDN []string
	nDN, err := ldapdn.ParseNormalize(teacher.DN)
	if err != nil {
		logger.Error().Str("new teacher", teacher.DN).Err(err).Msg("Couldn't parse DN")
		return err
	}
	if _, present := currentSet[nDN]; !present {
		newTeacherDN = append(newTeacherDN, teacher.DN)
	} else {
		logger.Debug().Str("teacherDN", teacher.DN).Msg("Member already present in group. Skipping")
	}

	if len(newTeacherDN) > 0 {
		mr.Add(i.educationConfig.classAttributeMap.teachers, newTeacherDN)

		if err := i.conn.Modify(&mr); err != nil {
			return err
		}
	}

	return nil
}

// RemoveTeacherFromEducationClass removes teacher (by ID) from a class
func (i *LDAP) RemoveTeacherFromEducationClass(ctx context.Context, classID string, teacherID string) error {
	logger := i.logger.SubloggerWithRequestID(ctx)
	class, err := i.getEducationClassByID(classID, false)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get class: backend error")
		return err
	}

	teacher, err := i.getEducationUserByNameOrID(teacherID)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get education user: error fetching education user from backend")
		return err
	}

	return i.removeEntryByDNAndAttributeFromEntry(class, teacher.DN, i.educationConfig.classAttributeMap.teachers)
}
