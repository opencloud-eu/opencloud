package svc

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

// ListApplications implements the Service interface.
func (g Graph) ListApplications(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Interface("query", r.URL.Query()).Msg("calling list applications")

	lbr, err := g.roleService.ListRoles(r.Context(), &settingssvc.ListBundlesRequest{})
	if err != nil {
		logger.Error().Err(err).Msg("could not list roles: transport error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	roles := make([]libregraph.AppRole, 0, len(lbr.Bundles))
	for _, bundle := range lbr.GetBundles() {
		role := libregraph.NewAppRole(bundle.GetId())
		role.SetDisplayName(bundle.GetDisplayName())
		roles = append(roles, *role)
	}

	application := libregraph.NewApplication(g.config.Application.ID)
	application.SetDisplayName(g.config.Application.DisplayName)
	application.SetAppRoles(roles)

	applications := []*libregraph.Application{
		application,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: applications})
}

// GetApplication implements the Service interface.
func (g Graph) GetApplication(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Interface("query", r.URL.Query()).Msg("calling get application")

	applicationID := chi.URLParam(r, "applicationID")

	if applicationID != g.config.Application.ID {
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, fmt.Sprintf("requested id %s does not match expected application id %v", applicationID, g.config.Application.ID))
		return
	}

	lbr, err := g.roleService.ListRoles(r.Context(), &settingssvc.ListBundlesRequest{})
	if err != nil {
		logger.Error().Err(err).Msg("could not list roles: transport error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	roles := make([]libregraph.AppRole, 0, len(lbr.Bundles))
	for _, bundle := range lbr.GetBundles() {
		role := libregraph.NewAppRole(bundle.GetId())
		role.SetDisplayName(bundle.GetDisplayName())
		roles = append(roles, *role)
	}

	application := libregraph.NewApplication(applicationID)
	application.SetDisplayName(g.config.Application.DisplayName)
	application.SetAppRoles(roles)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, application)
}
