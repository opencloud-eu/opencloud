package svc

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/CiscoM31/godata"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// GetEducationClasses implements the Service interface.
func (g Graph) GetEducationClasses(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Interface("query", r.URL.Query()).Msg("calling GetEducationClasses")
	sanitizedPath := strings.TrimPrefix(r.URL.Path, "/graph/v1.0/")
	odataReq, err := godata.ParseRequest(r.Context(), sanitizedPath, r.URL.Query())
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg(
			"could not get educationClasses: query error")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	classes, err := g.identityEducationBackend.GetEducationClasses(r.Context())
	if err != nil {
		logger.Debug().Err(err).Msg("could not get classes: backend error")
		errorcode.RenderError(w, r, err)
		return
	}

	classes, err = sortClasses(odataReq, classes)
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("cannot get classes: could not sort classes according to query")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: classes})
}

// PostEducationClass implements the Service interface.
func (g Graph) PostEducationClass(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Msg("calling post EducationClass")
	class := libregraph.NewEducationClassWithDefaults()
	err := StrictJSONUnmarshal(r.Body, class)
	if err != nil {
		logger.Debug().Err(err).Interface("body", r.Body).Msg("could not create education class: invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	if _, ok := class.GetDisplayNameOk(); !ok {
		logger.Debug().Err(err).Interface("class", class).Msg("could not create class: missing required attribute")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "Missing Required Attribute")
		return
	}

	// Disallow user-supplied IDs. It's supposed to be readonly. We're either
	// generating them in the backend ourselves or rely on the Backend's
	// storage (e.g. LDAP) to provide a unique ID.
	if _, ok := class.GetIdOk(); ok {
		logger.Debug().Msg("could not create class: id is a read-only attribute")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "class id is a read-only attribute")
		return
	}

	if class, err = g.identityEducationBackend.CreateEducationClass(r.Context(), *class); err != nil {
		logger.Debug().Interface("class", class).Msg("could not create class: backend error")
		errorcode.RenderError(w, r, err)
		return
	}

	/* TODO requires reva changes
	if class != nil && class.Id != nil {
		currentUser := revactx.ContextMustGetUser(r.Context())
		g.publishEvent(events.EducationClassCreated{Executant: currentUser.Id, EducationClassID: *class.Id})
	}
	*/
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, class)
}

// PatchEducationClass implements the Service interface.
func (g Graph) PatchEducationClass(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	classID := chi.URLParam(r, "classID")
	logger.Info().Str("classID", classID).Msg("calling patch education class")
	classID, err := url.PathUnescape(classID)
	if err != nil {
		logger.Debug().Str("id", classID).Msg("could not change class: unescaping class id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping class id failed")
		return
	}

	if classID == "" {
		logger.Debug().Msg("could not change class: missing class id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing class id")
		return
	}
	changes := libregraph.NewEducationClassWithDefaults()
	err = StrictJSONUnmarshal(r.Body, changes)
	if err != nil {
		logger.Debug().Err(err).Interface("body", r.Body).Msg("could not change class: invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	var features []events.GroupFeature
	if displayName, ok := changes.GetDisplayNameOk(); ok {
		features = append(features, events.GroupFeature{Name: "displayname", Value: *displayName})
	}

	if externalID, ok := changes.GetExternalIdOk(); ok {
		features = append(features, events.GroupFeature{Name: "externalid", Value: *externalID})
	}

	_, err = g.identityEducationBackend.UpdateEducationClass(r.Context(), classID, *changes)
	if err != nil {
		logger.Error().
			Err(err).
			Str("classID", classID).
			Msg("could not update class")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if memberRefs, ok := changes.GetMembersodataBindOk(); ok {
		// The spec defines a limit of 20 members maxium per Request
		if len(memberRefs) > g.config.API.GroupMembersPatchLimit {
			logger.Debug().
				Int("number", len(memberRefs)).
				Int("limit", g.config.API.GroupMembersPatchLimit).
				Msg("could not add group members, exceeded members limit")
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest,
				fmt.Sprintf("Request is limited to %d members", g.config.API.GroupMembersPatchLimit))
			return
		}
		memberIDs := make([]string, 0, len(memberRefs))
		for _, memberRef := range memberRefs {
			memberType, id, err := g.parseMemberRef(memberRef)
			if err != nil {
				logger.Debug().
					Str("memberref", memberRef).
					Msg("could not change class: Error parsing member@odata.bind values")
				errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "Error parsing member@odata.bind values")
				return
			}
			logger.Debug().Str("membertype", memberType).Str("memberid", id).Msg("add class member")
			// The MS Graph spec allows "directoryObject", "user", "class" and "organizational Contact"
			// we restrict this to users for now. Might add Classes as members later
			if memberType != memberTypeUsers {
				logger.Debug().
					Str("type", memberType).
					Msg("could not change class: could not add member, only user type is allowed")
				errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "Only user are allowed as class members")
				return
			}
			memberIDs = append(memberIDs, id)
		}
		err = g.identityBackend.AddMembersToGroup(r.Context(), classID, memberIDs)
	}

	if err != nil {
		logger.Debug().Err(err).Msg("could not change class: backend could not add members")
		errorcode.RenderError(w, r, err)
		return
	}

	if len(features) > 0 {
		e := events.GroupFeatureChanged{
			GroupID:  classID,
			Features: features,
		}

		if currentUser, ok := revactx.ContextGetUser(r.Context()); ok {
			e.Executant = currentUser.GetId()
		}
		g.publishEvent(r.Context(), e)

	}

	render.Status(r, http.StatusNoContent) // TODO StatusNoContent when prefer=minimal is used, otherwise OK and the resource in the body
	render.NoContent(w, r)
}

// GetEducationClass implements the Service interface.
func (g Graph) GetEducationClass(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	classID := chi.URLParam(r, "classID")
	logger.Info().Str("classID", classID).Msg("calling get education class")
	classID, err := url.PathUnescape(classID)
	if err != nil {
		logger.Debug().Str("id", classID).Msg("could not get class: unescaping class id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping class id failed")
	}

	if classID == "" {
		logger.Debug().Msg("could not get class: missing class id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing class id")
		return
	}

	logger.Debug().
		Str("id", classID).
		Interface("query", r.URL.Query()).
		Msg("calling get class on backend")
	class, err := g.identityEducationBackend.GetEducationClass(r.Context(), classID)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get class: backend error")
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, class)
}

// DeleteEducationClass implements the Service interface.
func (g Graph) DeleteEducationClass(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	classID := chi.URLParam(r, "classID")
	logger.Info().Str("classID", classID).Msg("calling delete class")
	classID, err := url.PathUnescape(classID)
	if err != nil {
		logger.Debug().Err(err).Str("id", classID).Msg("could not delete class: unescaping class id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping class id failed")
		return
	}

	if classID == "" {
		logger.Debug().Msg("could not delete class: missing class id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing class id")
		return
	}

	logger.Debug().Str("id", classID).Msg("calling delete class on backend")
	err = g.identityEducationBackend.DeleteEducationClass(r.Context(), classID)

	if err != nil {
		logger.Debug().Err(err).Msg("could not delete class: backend error")
		errorcode.RenderError(w, r, err)
		return
	}

	/* TODO requires reva changes
	currentUser := revactx.ContextMustGetUser(r.Context())
	g.publishEvent(events.ClassDeleted{Executant: currentUser.Id, ClassID: classID})
	*/

	render.Status(r, http.StatusNoContent)
	render.NoContent(w, r)
}

// GetEducationClassMembers implements the Service interface.
func (g Graph) GetEducationClassMembers(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	classID := chi.URLParam(r, "classID")
	logger.Info().Str("classID", classID).Msg("calling get class members")
	classID, err := url.PathUnescape(classID)
	if err != nil {
		logger.Debug().Str("id", classID).Msg("could not get class members: unescaping class id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping class id failed")
		return
	}

	if classID == "" {
		logger.Debug().Msg("could not get class members: missing class id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing class id")
		return
	}

	logger.Debug().Str("id", classID).Msg("calling get class members on backend")
	members, err := g.identityEducationBackend.GetEducationClassMembers(r.Context(), classID)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get class members: backend error")
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, members)
}

// PostEducationClassMember implements the Service interface.
func (g Graph) PostEducationClassMember(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())

	classID := chi.URLParam(r, "classID")
	logger.Info().Str("classID", classID).Msg("Calling post class member")
	classID, err := url.PathUnescape(classID)
	if err != nil {
		logger.Debug().
			Err(err).
			Str("id", classID).
			Msg("could not add member to class: unescaping class id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping class id failed")
		return
	}

	if classID == "" {
		logger.Debug().Msg("could not add class member: missing class id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing class id")
		return
	}
	memberRef := libregraph.NewMemberReference()
	err = StrictJSONUnmarshal(r.Body, memberRef)
	if err != nil {
		logger.Debug().
			Err(err).
			Interface("body", r.Body).
			Msg("could not add class member: invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	memberRefURL, ok := memberRef.GetOdataIdOk()
	if !ok {
		logger.Debug().Msg("could not add class member: @odata.id reference is missing")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "@odata.id reference is missing")
		return
	}
	memberType, id, err := g.parseMemberRef(*memberRefURL)
	if err != nil {
		logger.Debug().Err(err).Msg("could not add class member: error parsing @odata.id url")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "Error parsing @odata.id url")
		return
	}
	// The MS Graph spec allows "directoryObject", "user", "class" and "organizational Contact"
	// we restrict this to users for now. Might add EducationClass as members later
	if memberType != memberTypeUsers {
		logger.Debug().Str("type", memberType).Msg("could not add class member: Only users are allowed as class members")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "Only users are allowed as class members")
		return
	}

	logger.Debug().Str("memberType", memberType).Str("id", id).Msg("calling add member on backend")
	err = g.identityBackend.AddMembersToGroup(r.Context(), classID, []string{id})

	if err != nil {
		logger.Debug().Err(err).Msg("could not add class member: backend error")
		errorcode.RenderError(w, r, err)
		return
	}

	/* TODO requires reva changes
	currentUser := revactx.ContextMustGetUser(r.Context())
	g.publishEvent(events.EducationClassMemberAdded{Executant: currentUser.Id, EducationClassID: classID, UserID: id})
	*/
	render.Status(r, http.StatusNoContent)
	render.NoContent(w, r)
}

// DeleteEducationClassMember implements the Service interface.
func (g Graph) DeleteEducationClassMember(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())

	classID := chi.URLParam(r, "classID")
	memberID := chi.URLParam(r, "memberID")
	logger.Info().Str("classID", classID).Str("memberID", memberID).Msg("calling delete class member")
	classID, err := url.PathUnescape(classID)
	if err != nil {
		logger.Debug().Err(err).Str("id", classID).Msg("could not delete class member: unescaping class id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping class id failed")
		return
	}

	if classID == "" {
		logger.Debug().Msg("could not delete class member: missing class id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing class id")
		return
	}

	memberID, err = url.PathUnescape(memberID)
	if err != nil {
		logger.Debug().Err(err).Str("id", memberID).Msg("could not delete class member: unescaping member id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping member id failed")
		return
	}

	if memberID == "" {
		logger.Debug().Msg("could not delete class member: missing member id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing member id")
		return
	}
	logger.Debug().Str("classID", classID).Str("memberID", memberID).Msg("calling delete member on backend")
	err = g.identityBackend.RemoveMemberFromGroup(r.Context(), classID, memberID)

	if err != nil {
		logger.Debug().Err(err).Msg("could not delete class member: backend error")
		errorcode.RenderError(w, r, err)
		return
	}
	/* TODO requires reva changes
	currentUser := revactx.ContextMustGetUser(r.Context())
	g.publishEvent(events.EducationClassMemberRemoved{Executant: currentUser.Id, EducationClassID: classID, UserID: memberID})
	*/
	render.Status(r, http.StatusNoContent)
	render.NoContent(w, r)
}

// GetEducationClassTeachers implements the Service interface.
func (g Graph) GetEducationClassTeachers(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	classID := chi.URLParam(r, "classID")
	logger.Info().Str("classID", classID).Msg("calling get class teachers")
	classID, err := url.PathUnescape(classID)
	if err != nil {
		logger.Debug().Str("id", classID).Msg("could not get class teachers: unescaping class id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping class id failed")
		return
	}

	if classID == "" {
		logger.Debug().Msg("could not get class teachers: missing class id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing class id")
		return
	}

	logger.Debug().Str("id", classID).Msg("calling get class teachers on backend")
	teachers, err := g.identityEducationBackend.GetEducationClassTeachers(r.Context(), classID)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get class teachers: backend error")
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, teachers)
}

// PostEducationClassTeacher implements the Service interface.
func (g Graph) PostEducationClassTeacher(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())

	classID := chi.URLParam(r, "classID")
	logger.Info().Str("classID", classID).Msg("Calling post class teacher")
	classID, err := url.PathUnescape(classID)
	if err != nil {
		logger.Debug().
			Err(err).
			Str("id", classID).
			Msg("could not add teacher to class: unescaping class id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping class id failed")
		return
	}

	if classID == "" {
		logger.Debug().Msg("could not add class teacher: missing class id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing class id")
		return
	}
	memberRef := libregraph.NewMemberReference()
	err = StrictJSONUnmarshal(r.Body, memberRef)
	if err != nil {
		logger.Debug().
			Err(err).
			Interface("body", r.Body).
			Msg("could not add class teacher: invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	memberRefURL, ok := memberRef.GetOdataIdOk()
	if !ok {
		logger.Debug().Msg("could not add class teacher: @odata.id reference is missing")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "@odata.id reference is missing")
		return
	}
	memberType, id, err := g.parseMemberRef(*memberRefURL)
	if err != nil {
		logger.Debug().Err(err).Msg("could not add class teacher: error parsing @odata.id url")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "Error parsing @odata.id url")
		return
	}
	// The MS Graph spec allows "directoryObject", "user", "class" and "organizational Contact"
	// we restrict this to users for now. Might add EducationClass as members later
	if memberType != memberTypeUsers {
		logger.Debug().Str("type", memberType).Msg("could not add class member: Only users are allowed as class teachers")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "Only users are allowed as class teachers")
		return
	}

	logger.Debug().Str("memberType", memberType).Str("id", id).Msg("calling add teacher on backend")
	err = g.identityEducationBackend.AddTeacherToEducationClass(r.Context(), classID, id)

	if err != nil {
		logger.Debug().Err(err).Msg("could not add class teacher: backend error")
		errorcode.RenderError(w, r, err)
		return
	}

	/* TODO requires reva changes
	currentUser := revactx.ContextMustGetUser(r.Context())
	g.publishEvent(events.EducationClassTeacherAdded{Executant: currentUser.Id, EducationClassID: classID, UserID: id})
	*/
	render.Status(r, http.StatusNoContent)
	render.NoContent(w, r)
}

// DeleteEducationClassTeacher implements the Service interface.
func (g Graph) DeleteEducationClassTeacher(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())

	classID := chi.URLParam(r, "classID")
	teacherID := chi.URLParam(r, "teacherID")
	logger.Info().Str("classID", classID).Str("teacherID", teacherID).Msg("calling delete class teacher")
	classID, err := url.PathUnescape(classID)
	if err != nil {
		logger.Debug().Err(err).Str("id", classID).Msg("could not delete class teacher: unescaping class id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping class id failed")
		return
	}

	if classID == "" {
		logger.Debug().Msg("could not delete class teacher: missing class id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing class id")
		return
	}

	teacherID, err = url.PathUnescape(teacherID)
	if err != nil {
		logger.Debug().Err(err).Str("id", teacherID).Msg("could not delete class teacher: unescaping teacher id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping teacher id failed")
		return
	}

	if teacherID == "" {
		logger.Debug().Msg("could not delete class teacher: missing teacher id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing teacher id")
		return
	}
	logger.Debug().Str("classID", classID).Str("teacherID", teacherID).Msg("calling delete teacher on backend")
	err = g.identityEducationBackend.RemoveTeacherFromEducationClass(r.Context(), classID, teacherID)

	if err != nil {
		logger.Debug().Err(err).Msg("could not delete class teacher: backend error")
		errorcode.RenderError(w, r, err)
		return
	}
	/* TODO requires reva changes
	currentUser := revactx.ContextMustGetUser(r.Context())
	g.publishEvent(events.EducationClassTeacherRemoved{Executant: currentUser.Id, EducationClassID: classID, UserID: teacherID})
	*/
	render.Status(r, http.StatusNoContent)
	render.NoContent(w, r)
}

func sortClasses(req *godata.GoDataRequest, classes []*libregraph.EducationClass) ([]*libregraph.EducationClass, error) {
	if req.Query.OrderBy == nil || len(req.Query.OrderBy.OrderByItems) != 1 {
		return classes, nil
	}
	var less func(i, j int) bool

	switch req.Query.OrderBy.OrderByItems[0].Field.Value {
	case displayNameAttr:
		less = func(i, j int) bool {
			return strings.ToLower(classes[i].GetDisplayName()) < strings.ToLower(classes[j].GetDisplayName())
		}
	default:
		return nil, fmt.Errorf("we do not support <%s> as a order parameter", req.Query.OrderBy.OrderByItems[0].Field.Value)
	}

	if req.Query.OrderBy.OrderByItems[0].Order == _sortDescending {
		sort.Slice(classes, reverse(less))
	} else {
		sort.Slice(classes, less)
	}

	return classes, nil
}
