package svc

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/CiscoM31/godata"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// GetEducationUsers implements the Service interface.
func (g Graph) GetEducationUsers(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Interface("query", r.URL.Query()).Msg("calling get education users")
	sanitizedPath := strings.TrimPrefix(r.URL.Path, "/graph/v1.0/")
	odataReq, err := godata.ParseRequest(r.Context(), sanitizedPath, r.URL.Query())
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get education users: query error")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	logger.Debug().Interface("query", r.URL.Query()).Msg("calling get education users on backend")
	users, err := g.identityEducationBackend.GetEducationUsers(r.Context())
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get education users from backend")
		errorcode.RenderError(w, r, err)
		return
	}

	users, err = sortEducationUsers(odataReq, users)
	if err != nil {
		logger.Debug().Interface("query", odataReq).Msg("error while sorting education users according to query")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: users})
}

// PostEducationUser implements the Service interface.
func (g Graph) PostEducationUser(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Interface("body", r.Body).Msg("calling create education user")
	u := libregraph.NewEducationUser()
	err := StrictJSONUnmarshal(r.Body, u)
	if err != nil {
		logger.Debug().Err(err).Interface("body", r.Body).Msg("could not create education user: invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err.Error()))
		return
	}

	// Disallow user-supplied IDs. It's supposed to be readonly. We're either
	// generating them in the backend ourselves or rely on the Backend's
	// storage (e.g. LDAP) to provide a unique ID.
	if _, ok := u.GetIdOk(); ok {
		logger.Debug().Interface("user", u).Msg("could not create education user: id is a read-only attribute")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "education user id is a read-only attribute")
		return
	}

	if _, ok := u.GetDisplayNameOk(); !ok {
		logger.Debug().Err(err).Interface("user", u).Msg("could not create education user: missing required Attribute: 'displayName'")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing required Attribute: 'displayName'")
		return
	}

	identities, ok := u.GetIdentitiesOk()
	if !ok {
		logger.Debug().Err(err).Interface("user", u).Msg("could not create education user: missing required Collection: 'identities'")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing required Attribute: 'identities'")
		return
	}
	if len(identities) < 1 {
		logger.Debug().Err(err).Interface("user", u).Msg("could not create education user: missing entry in Collection: 'identities'")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing required Collection: 'identities'")
		return
	}
	for i, identity := range identities {
		if _, ok := identity.GetIssuerOk(); !ok {
			logger.Debug().Err(err).Interface("user", u).Msgf("could not create education user: missing Attribute in 'identities' Collection Entry %d: 'issuer'", i)
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("missing Attribute in 'identities' Collection Entry %d: 'issuer'", i))
			return
		}
		if _, ok := identity.GetIssuerAssignedIdOk(); !ok {
			logger.Debug().Err(err).Interface("user", u).Msgf("could not create education user: missing Attribute in 'identities' Collection Entry %d: 'issuerAssignedId'", i)
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("missing Attribute in 'identities' Collection Entry %d: 'issuerAssignedId'", i))
			return
		}
	}

	if accountName, ok := u.GetOnPremisesSamAccountNameOk(); ok {
		if !g.isValidUsername(*accountName) {
			logger.Debug().Str("username", *accountName).Msg("could not create education user: username must be at least the local part of an email")
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("username %s must be at least the local part of an email", *u.OnPremisesSamAccountName))
			return
		}
	} else {
		logger.Debug().Interface("user", u).Msg("could not create education user: missing required Attribute: 'onPremisesSamAccountName'")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing required Attribute: 'onPremisesSamAccountName'")
		return
	}

	if mail, ok := u.GetMailOk(); ok {
		if !isValidEmail(*mail) {
			logger.Debug().Str("mail", *u.Mail).Msg("could not create education user: invalid email address")
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("%v is not a valid email address", *u.Mail))
			return
		}
	}

	if u.HasUserType() {
		if !isValidUserType(*u.UserType) {
			logger.Debug().Interface("user", u).Msg("invalid userType attribute")
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid userType attribute, valid options are 'Member' or 'Guest'")
			return
		}
	} else {
		u.SetUserType("Member")
	}

	if _, ok := u.GetPrimaryRoleOk(); !ok {
		logger.Debug().Err(err).Interface("user", u).Msg("could not create education user: missing required Attribute: 'primaryRole'")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing required Attribute: 'primaryRole'")
		return
	}

	logger.Debug().Interface("user", u).Msg("calling create education user on backend")
	if u, err = g.identityEducationBackend.CreateEducationUser(r.Context(), *u); err != nil {
		logger.Debug().Err(err).Msg("could not create education user: backend error")
		errorcode.RenderError(w, r, err)
		return
	}

	e := events.UserCreated{UserID: *u.Id}
	if currentUser, ok := revactx.ContextGetUser(r.Context()); ok {
		e.Executant = currentUser.GetId()
	}
	g.publishEvent(r.Context(), e)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, u)
}

// GetEducationUser implements the Service interface.
func (g Graph) GetEducationUser(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	userID := chi.URLParam(r, "userID")
	logger.Info().Str("userID", userID).Msg("calling get education user")
	userID, err := url.PathUnescape(userID)
	if err != nil {
		logger.Debug().Err(err).Str("id", userID).Msg("could not get education user: unescaping education user id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping education user id failed")
		return
	}

	if userID == "" {
		logger.Debug().Msg("could not get user: missing education user id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing education user id")
		return
	}

	logger.Debug().Str("id", userID).Msg("calling get education user from backend")
	user, err := g.identityEducationBackend.GetEducationUser(r.Context(), userID)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get education user: error fetching education user from backend")
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, user)
}

// DeleteEducationUser implements the Service interface.
func (g Graph) DeleteEducationUser(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	userID := chi.URLParam(r, "userID")
	logger.Info().Str("userID", userID).Msg("calling delete education user")
	userID, err := url.PathUnescape(userID)
	if err != nil {
		logger.Debug().Err(err).Str("id", userID).Msg("could not delete education user: unescaping education user id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping education user id failed")
		return
	}

	if userID == "" {
		logger.Debug().Msg("could not delete education user: missing education user id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing education user id")
		return
	}
	logger.Debug().Str("id", userID).Msg("calling get education user on user backend")
	user, err := g.identityEducationBackend.GetEducationUser(r.Context(), userID)
	if err != nil {
		logger.Debug().Err(err).Str("userID", userID).Msg("failed to get education user from backend")
		errorcode.RenderError(w, r, err)
		return
	}

	e := events.UserDeleted{UserID: user.GetId()}
	if currentUser, ok := revactx.ContextGetUser(r.Context()); ok {
		if currentUser.GetId().GetOpaqueId() == user.GetId() {
			logger.Debug().Msg("could not delete education user: self deletion forbidden")
			errorcode.NotAllowed.Render(w, r, http.StatusForbidden, "self deletion forbidden")
			return
		}
		e.Executant = currentUser.GetId()
	}

	if g.gatewaySelector != nil {
		logger.Debug().
			Str("user", user.GetId()).
			Msg("calling list spaces with user filter to fetch the personal space for deletion")
		opaque := utils.AppendPlainToOpaque(nil, "unrestricted", "T")
		f := listStorageSpacesUserFilter(user.GetId())
		client, err := g.gatewaySelector.Next()
		if err != nil {
			logger.Error().Err(err).Msg("could not select next gateway client")
			errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, "could not select next gateway client, aborting")
			return
		}
		lspr, err := client.ListStorageSpaces(r.Context(), &storageprovider.ListStorageSpacesRequest{
			Opaque:  opaque,
			Filters: []*storageprovider.ListStorageSpacesRequest_Filter{f},
		})
		if err != nil {
			// transport error, log as error
			logger.Error().Err(err).Msg("could not fetch spaces: transport error")
			errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not fetch spaces for deletion, aborting")
			return
		}
		for _, sp := range lspr.GetStorageSpaces() {
			if !(sp.SpaceType == _spaceTypePersonal && sp.Owner.Id.OpaqueId == user.GetId()) {
				continue
			}
			// TODO: check if request contains a homespace and if, check if requesting user has the privilege to
			// delete it and make sure it is not deleting its own homespace
			// needs modification of the cs3api

			// Deleting a space a two step process (1. disabling/trashing, 2. purging)
			// Do the "disable/trash" step only if the space is not marked as trashed yet:
			if _, ok := sp.Opaque.Map[_spaceStateTrashed]; !ok {
				_, err := client.DeleteStorageSpace(r.Context(), &storageprovider.DeleteStorageSpaceRequest{
					Id: &storageprovider.StorageSpaceId{
						OpaqueId: sp.Id.OpaqueId,
					},
				})
				if err != nil {
					logger.Error().Err(err).Msg("could not disable homespace: transport error")
					errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not disable homespace, aborting")
					return
				}
			}
			purgeFlag := utils.AppendPlainToOpaque(nil, "purge", "")
			_, err := client.DeleteStorageSpace(r.Context(), &storageprovider.DeleteStorageSpaceRequest{
				Opaque: purgeFlag,
				Id: &storageprovider.StorageSpaceId{
					OpaqueId: sp.Id.OpaqueId,
				},
			})
			if err != nil {
				// transport error, log as error
				logger.Error().Err(err).Msg("could not delete homespace: transport error")
				errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not delete homespace, aborting")
				return
			}
			break
		}
	}

	logger.Debug().Str("id", user.GetId()).Msg("calling delete education user on backend")
	err = g.identityEducationBackend.DeleteEducationUser(r.Context(), user.GetId())

	if err != nil {
		logger.Debug().Err(err).Msg("could not delete education user: backend error")
		errorcode.RenderError(w, r, err)
		return
	}

	g.publishEvent(r.Context(), e)

	render.Status(r, http.StatusNoContent)
	render.NoContent(w, r)
}

// PatchEducationUser implements the Service Interface. Updates the specified attributes of an
// ExistingUser
func (g Graph) PatchEducationUser(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	nameOrID := chi.URLParam(r, "userID")
	logger.Info().Str("userID", nameOrID).Msg("calling patch education user")
	nameOrID, err := url.PathUnescape(nameOrID)
	if err != nil {
		logger.Debug().Err(err).Str("id", nameOrID).Msg("could not update education user: unescaping education user id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping education user id failed")
		return
	}

	if nameOrID == "" {
		logger.Debug().Msg("could not update education user: missing education user id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing education user id")
		return
	}
	changes := libregraph.NewEducationUser()
	err = StrictJSONUnmarshal(r.Body, changes)
	if err != nil {
		logger.Debug().Err(err).Interface("body", r.Body).Msg("could not update education user: invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest,
			fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}

	if accountName, ok := changes.GetOnPremisesSamAccountNameOk(); ok {
		if !g.isValidUsername(*accountName) {
			logger.Debug().Str("username", *accountName).Msg("could not update education user: username must be at least the local part of an email")
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("username %s must be at least the local part of an email", *changes.OnPremisesSamAccountName))
			return
		}
	}

	var features []events.UserFeature
	if mail, ok := changes.GetMailOk(); ok {
		if !isValidEmail(*mail) {
			logger.Debug().Str("mail", *mail).Msg("could not update education user: email is not a valid email address")
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest,
				fmt.Sprintf("'%s' is not a valid email address", *mail))
			return
		}
		features = append(features, events.UserFeature{Name: "email", Value: *mail})
	}

	if name, ok := changes.GetDisplayNameOk(); ok {
		features = append(features, events.UserFeature{Name: "displayname", Value: *name})
	}

	if changes.HasUserType() {
		if !isValidUserType(*changes.UserType) {
			logger.Debug().Interface("user", changes).Msg("invalid userType attribute")
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid userType attribute, valid options are 'Member' or 'Guest'")
			return
		}
	}

	logger.Debug().Str("nameid", nameOrID).Interface("changes", *changes).Msg("calling update education user on backend")
	u, err := g.identityEducationBackend.UpdateEducationUser(r.Context(), nameOrID, *changes)
	if err != nil {
		logger.Debug().Err(err).Str("id", nameOrID).Msg("could not update education user: backend error")
		errorcode.RenderError(w, r, err)
		return
	}

	e := events.UserFeatureChanged{
		UserID:    nameOrID,
		Features:  features,
		Timestamp: utils.TSNow(),
	}
	if currentUser, ok := revactx.ContextGetUser(r.Context()); ok {
		e.Executant = currentUser.GetId()
	}
	g.publishEvent(r.Context(), e)

	render.Status(r, http.StatusOK) // TODO StatusNoContent when prefer=minimal is used
	render.JSON(w, r, u)
}

func sortEducationUsers(req *godata.GoDataRequest, users []*libregraph.EducationUser) ([]*libregraph.EducationUser, error) {
	if req.Query.OrderBy == nil || len(req.Query.OrderBy.OrderByItems) != 1 {
		return users, nil
	}
	var less func(i, j int) bool

	switch req.Query.OrderBy.OrderByItems[0].Field.Value {
	case displayNameAttr:
		less = func(i, j int) bool {
			return strings.ToLower(users[i].GetDisplayName()) < strings.ToLower(users[j].GetDisplayName())
		}
	case "mail":
		less = func(i, j int) bool {
			return strings.ToLower(users[i].GetMail()) < strings.ToLower(users[j].GetMail())
		}
	case "onPremisesSamAccountName":
		less = func(i, j int) bool {
			return strings.ToLower(users[i].GetOnPremisesSamAccountName()) < strings.ToLower(users[j].GetOnPremisesSamAccountName())
		}
	default:
		return nil, fmt.Errorf("we do not support <%s> as a order parameter", req.Query.OrderBy.OrderByItems[0].Field.Value)
	}

	if req.Query.OrderBy.OrderByItems[0].Order == _sortDescending {
		sort.Slice(users, reverse(less))
	} else {
		sort.Slice(users, less)
	}
	return users, nil
}
