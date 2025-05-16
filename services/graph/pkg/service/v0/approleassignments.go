package svc

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	merrors "go-micro.dev/v4/errors"
)

const principalTypeUser = "User"

// ListAppRoleAssignments implements the Service interface.
func (g Graph) ListAppRoleAssignments(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Interface("query", r.URL.Query()).Msg("calling list appRoleAssignments")

	userID := chi.URLParam(r, "userID")

	lrar, err := g.roleService.ListRoleAssignments(r.Context(), &settingssvc.ListRoleAssignmentsRequest{
		AccountUuid: userID,
	})
	if err != nil {
		// TODO check the error type and return proper error code
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	values := make([]libregraph.AppRoleAssignment, 0, len(lrar.GetAssignments()))
	for _, assignment := range lrar.GetAssignments() {
		values = append(values, g.assignmentToAppRoleAssignment(assignment))
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: values})
}

// CreateAppRoleAssignment implements the Service interface.
func (g Graph) CreateAppRoleAssignment(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Interface("query", r.URL.Query()).Msg("calling create appRoleAssignment")

	appRoleAssignment := libregraph.NewAppRoleAssignmentWithDefaults()
	err := StrictJSONUnmarshal(r.Body, appRoleAssignment)
	if err != nil {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err.Error()))
		return
	}

	userID := chi.URLParam(r, "userID")

	// We can ignore the error, in the worst case the old role will be empty
	oldRoles, _ := g.roleService.ListRoleAssignments(r.Context(), &settingssvc.ListRoleAssignmentsRequest{
		AccountUuid: userID,
	})

	if appRoleAssignment.GetPrincipalId() != userID {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("user id %s does not match principal id %v", userID, appRoleAssignment.GetPrincipalId()))
		return
	}
	if appRoleAssignment.GetResourceId() != g.config.Application.ID {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("resource id %s does not match expected application id %v", userID, g.config.Application.ID))
		return
	}

	artur, err := g.roleService.AssignRoleToUser(r.Context(), &settingssvc.AssignRoleToUserRequest{
		AccountUuid: userID,
		RoleId:      appRoleAssignment.AppRoleId,
	})
	if err != nil {
		if merr, ok := merrors.As(err); ok && merr.Code == http.StatusForbidden {
			errorcode.NotAllowed.Render(w, r, http.StatusForbidden, err.Error())
			return
		}
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	var role string
	roles := oldRoles.GetAssignments()
	if len(roles) > 0 {
		role = roles[0].GetRoleId()
	}

	e := events.UserFeatureChanged{
		UserID: userID,
		Features: []events.UserFeature{
			{
				Name:     "roleChanged",
				Value:    appRoleAssignment.AppRoleId,
				OldValue: &role,
			},
		},
		Timestamp: utils.TSNow(),
	}
	if currentUser, ok := revactx.ContextGetUser(r.Context()); ok {
		e.Executant = currentUser.GetId()
	}
	g.publishEvent(r.Context(), e)
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, g.assignmentToAppRoleAssignment(artur.GetAssignment()))
}

// DeleteAppRoleAssignment implements the Service interface.
func (g Graph) DeleteAppRoleAssignment(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Interface("body", r.Body).Msg("calling delete appRoleAssignment")

	userID := chi.URLParam(r, "userID")

	// check assignment belongs to the user
	lrar, err := g.roleService.ListRoleAssignments(r.Context(), &settingssvc.ListRoleAssignmentsRequest{
		AccountUuid: userID,
	})
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	appRoleAssignmentID := chi.URLParam(r, "appRoleAssignmentID")

	assignmentFound := false
	for _, roleAssignment := range lrar.GetAssignments() {
		if roleAssignment.Id == appRoleAssignmentID {
			assignmentFound = true
		}
	}
	if !assignmentFound {
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, fmt.Sprintf("appRoleAssignment %v not found for user %v", appRoleAssignmentID, userID))
		return
	}

	_, err = g.roleService.RemoveRoleFromUser(r.Context(), &settingssvc.RemoveRoleFromUserRequest{
		Id: appRoleAssignmentID,
	})
	if err != nil {
		if merr, ok := merrors.As(err); ok && merr.Code == http.StatusForbidden {
			errorcode.NotAllowed.Render(w, r, http.StatusForbidden, err.Error())
			return
		}
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	render.NoContent(w, r)
}

func (g Graph) assignmentToAppRoleAssignment(assignment *settingsmsg.UserRoleAssignment) libregraph.AppRoleAssignment {
	appRoleAssignment := libregraph.NewAppRoleAssignmentWithDefaults()
	appRoleAssignment.SetId(assignment.Id)
	appRoleAssignment.SetAppRoleId(assignment.RoleId)
	appRoleAssignment.SetPrincipalType(principalTypeUser) // currently always assigned to the user
	appRoleAssignment.SetResourceId(g.config.Application.ID)
	appRoleAssignment.SetResourceDisplayName(g.config.Application.DisplayName)
	appRoleAssignment.SetPrincipalId(assignment.AccountUuid)
	// appRoleAssignment.SetPrincipalDisplayName() // TODO fetch and cache
	return *appRoleAssignment
}
