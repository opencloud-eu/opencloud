package svc

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

	"github.com/opencloud-eu/opencloud/pkg/l10n"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
)

// GetRoleDefinitions a list of permission roles than can be used when sharing with users or groups
func (g Graph) GetRoleDefinitions(w http.ResponseWriter, r *http.Request) {
	loc := r.Header.Get(l10n.HeaderAcceptLanguage)
	w.Header().Add("Content-Language", loc)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, unifiedrole.LocalizeRoles(g.availableRoles, loc))
}

// GetRoleDefinition a permission role than can be used when sharing with users or groups
func (g Graph) GetRoleDefinition(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	roleID, err := url.PathUnescape(chi.URLParam(r, "roleID"))
	if err != nil {
		logger.Debug().Err(err).Str("roleID", chi.URLParam(r, "roleID")).Msg("could not get roleID: unescaping is failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping role id failed")
		return
	}
	role, err := unifiedrole.GetRole(unifiedrole.RoleFilterIDs(roleID))
	if err != nil {
		logger.Debug().Str("roleID", roleID).Msg("could not get role: not found")
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, err.Error())
		return
	}
	loc := r.Header.Get(l10n.HeaderAcceptLanguage)
	w.Header().Add("Content-Language", loc)
	render.Status(r, http.StatusOK)
	render.JSON(w, r, unifiedrole.LocalizeRole(role, loc))
}
