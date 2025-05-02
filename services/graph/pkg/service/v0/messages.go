package svc

import (
	"net/http"
	"slices"
	"strings"

	"github.com/CiscoM31/godata"
	"github.com/go-chi/render"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	libregraph "github.com/owncloud/libre-graph-api-go"
)

// GetMessages implements the Service interface.
func (g Graph) GetMessages(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Debug().Msg("calling get messages in /me")
	sanitizedPath := strings.TrimPrefix(r.URL.Path, "/graph/v1.0/")

	odataReq, err := godata.ParseRequest(r.Context(), sanitizedPath, r.URL.Query())
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get messages: query error")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	u, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		logger.Debug().Msg("could not get messages: user not in context")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "user not in context")
		return
	}

	exp, err := identity.GetExpandValues(odataReq.Query)
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get messages: $expand error")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	var me *libregraph.User
	// We can just return the user from context unless we need to expand the group memberships
	if !slices.Contains(exp, "memberOf") {
		me = identity.CreateUserModelFromCS3(u)
	} else {
		var err error
		logger.Debug().Msg("calling get user on backend")
		me, err = g.identityBackend.GetUser(r.Context(), u.GetId().GetOpaqueId(), odataReq)
		if err != nil {
			logger.Debug().Err(err).Interface("user", u).Msg("could not get user from backend")
			errorcode.RenderError(w, r, err)
			return
		}
		if me.MemberOf == nil {
			me.MemberOf = []libregraph.Group{}
		}
	}

	// expand appRoleAssignments if requested
	if slices.Contains(exp, appRoleAssignments) {
		var err error
		me.AppRoleAssignments, err = g.fetchAppRoleAssignments(r.Context(), me.GetId())
		if err != nil {
			logger.Debug().Err(err).Str("userid", me.GetId()).Msg("could not get appRoleAssignments for self")
			errorcode.RenderError(w, r, err)
			return
		}
	}

	preferedLanguage, _, err := getUserLanguage(r.Context(), g.valueService, me.GetId())
	if err != nil {
		logger.Error().Err(err).Msg("could not get user language")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not get user language")
		return
	}

	me.PreferredLanguage = &preferedLanguage

	render.Status(r, http.StatusOK)
	render.JSON(w, r, me)
}
