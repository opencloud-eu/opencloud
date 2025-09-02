package svc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/CiscoM31/godata"
	invitepb "github.com/cs3org/go-cs3apis/cs3/ocm/invite/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	v1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/identity"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/odata"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/userstate"
	ocsettingssvc "github.com/opencloud-eu/opencloud/services/settings/pkg/service/v0"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/store/defaults"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/status"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
)

// GetMe implements the Service interface.
func (g Graph) GetMe(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Debug().Msg("calling get user in /me")
	sanitizedPath := strings.TrimPrefix(r.URL.Path, "/graph/v1.0/")

	odataReq, err := godata.ParseRequest(r.Context(), sanitizedPath, r.URL.Query())
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get users: query error")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	u, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		logger.Debug().Msg("could not get user: user not in context")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "user not in context")
		return
	}

	exp, err := odata.GetExpandValues(odataReq.Query)
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get users: $expand error")
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

func (g Graph) fetchAppRoleAssignments(ctx context.Context, accountuuid string) ([]libregraph.AppRoleAssignment, error) {
	lrar, err := g.roleService.ListRoleAssignments(ctx, &settingssvc.ListRoleAssignmentsRequest{
		AccountUuid: accountuuid,
	})
	if err != nil {
		return []libregraph.AppRoleAssignment{}, err
	}

	values := make([]libregraph.AppRoleAssignment, 0, len(lrar.Assignments))
	for _, assignment := range lrar.GetAssignments() {
		values = append(values, g.assignmentToAppRoleAssignment(assignment))
	}
	return values, nil
}

// GetUserDrive implements the Service interface.
func (g Graph) GetUserDrive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := g.logger.SubloggerWithRequestID(ctx).With().Interface("query", r.URL.Query()).Logger()
	log.Debug().Msg("calling get user drive")

	webDavBaseURL, err := g.getWebDavBaseURL()
	if err != nil {
		log.Error().Err(err).Msg("could not get personal drive: error parsing webdav base url")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	log = log.With().Str("url", webDavBaseURL.String()).Logger()

	userID, err := url.PathUnescape(chi.URLParam(r, "userID"))
	if err != nil {
		log.Debug().Err(err).Str("userID", chi.URLParam(r, "userID")).Msg("could not get drive: unescaping drive id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping user id failed")
		return
	}

	log = log.With().Str("userID", userID).Logger()

	_, expandPermissions, err := parseDriveRequest(r)
	if err != nil {
		log.Debug().Err(err).Msg("could not get drives: error parsing odata request")
		errorcode.RenderError(w, r, err)
		return
	}

	if userID == "" {
		u, ok := revactx.ContextGetUser(ctx)
		if !ok {
			log.Debug().Msg("could not get user: user not in context")
			errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "user not in context")
			return
		}
		userID = u.GetId().GetOpaqueId()
	}

	log.Debug().Msg("calling list storage spaces with user and personal filter")

	filters := []*storageprovider.ListStorageSpacesRequest_Filter{listStorageSpacesTypeFilter("personal"), listStorageSpacesUserFilter(userID)}
	res, err := g.ListStorageSpacesWithFilters(ctx, filters, true)
	switch {
	case err != nil:
		log.Error().Err(err).Msg("could not get drive: transport error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	case res.Status.Code != cs3rpc.Code_CODE_OK:
		if res.Status.Code == cs3rpc.Code_CODE_NOT_FOUND {
			// the client is doing a lookup for a specific space, therefore we need to return
			// not found to the caller
			log.Debug().Msg("could not get personal drive for user: not found")
			errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "drive not found")
			return
		}
		log.Debug().
			Str("grpcmessage", res.GetStatus().GetMessage()).
			Msg("could not get personal drive for user: grpc error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, res.Status.Message)
		return
	}

	spaces, err := g.formatDrives(ctx, webDavBaseURL, res.StorageSpaces, APIVersion_1, expandPermissions, nil)
	if err != nil {
		log.Debug().Err(err).Msg("could not get personal drive: error parsing grpc response")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	switch num := len(spaces); {
	case num == 0:
		log.Debug().Msg("could not get personal drive: no drive returned from storage")
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "no drive returned from storage")
		return
	case num == 1:
		render.Status(r, http.StatusOK)
		render.JSON(w, r, spaces[0])
	default:
		log.Debug().Int("number", num).Msg("could not get personal drive: expected to find a single drive but fetched more")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not get personal drive: expected to find a single drive but fetched more")
		return
	}
}

func (g Graph) contextUserHasFullAccountPerms(reqctx context.Context) bool {
	// mostly copied from the canCreateSpace method
	pr, err := g.permissionsService.GetPermissionByID(reqctx, &settingssvc.GetPermissionByIDRequest{
		PermissionId: defaults.AccountManagementPermission(0).Id,
	})
	if err != nil || pr.Permission == nil {
		return false
	}

	if pr.Permission.Constraint != defaults.All {
		return false
	}
	return true
}

// GetUsers implements the Service interface.
func (g Graph) GetUsers(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Debug().Interface("query", r.URL.Query()).Msg("calling get users")
	sanitizedPath := strings.TrimPrefix(r.URL.Path, "/graph/v1.0/")
	odataReq, err := godata.ParseRequest(r.Context(), sanitizedPath, r.URL.Query())
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get users: query error")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	ctxHasFullPerms := g.contextUserHasFullAccountPerms(r.Context())
	searchHasAcceptableLength := false
	if odataReq.Query != nil && odataReq.Query.Search != nil {
		minSearchLength := g.config.API.IdentitySearchMinLength
		if strings.HasPrefix(odataReq.Query.Search.RawValue, "\"") {
			// if search starts with double quotes then it must finish with double quotes
			// add +2 to the minimum search length in this case
			minSearchLength += 2
		}
		searchHasAcceptableLength = len(odataReq.Query.Search.RawValue) >= minSearchLength
	}
	if !ctxHasFullPerms && !searchHasAcceptableLength {
		// for regular user the search term must have a minimum length
		logger.Debug().Interface("query", r.URL.Query()).Msgf("search with less than %d chars for a regular user", g.config.API.IdentitySearchMinLength)
		errorcode.AccessDenied.Render(w, r, http.StatusForbidden, "search term too short")
		return
	}

	if !ctxHasFullPerms && odataReq.Query.Filter != nil {
		// regular users are allowed to filter only by userType
		filter := odataReq.Query.Filter
		switch {
		case filter.Tree.Token.Type != godata.ExpressionTokenLogical:
			fallthrough
		case filter.Tree.Token.Value != "eq":
			fallthrough
		case filter.Tree.Children[0].Token.Value != "userType":
			logger.Debug().Interface("query", r.URL.Query()).Msg("forbidden filter for a regular user")
			errorcode.AccessDenied.Render(w, r, http.StatusForbidden, "filter has forbidden elements for regular users")
			return
		}
	}
	if !ctxHasFullPerms && (odataReq.Query.Apply != nil || odataReq.Query.Expand != nil || odataReq.Query.Compute != nil) {
		// regular users can't use filter, apply, expand and compute
		logger.Debug().Interface("query", r.URL.Query()).Msg("forbidden query elements for a regular user")
		errorcode.AccessDenied.Render(w, r, http.StatusForbidden, "query has forbidden elements for regular users")
		return
	}

	logger.Debug().Interface("query", r.URL.Query()).Msg("calling get users on backend")

	var users []*libregraph.User

	if odataReq.Query.Filter != nil {
		users, err = g.applyUserFilter(r.Context(), odataReq, nil)
	} else {
		users, err = g.identityBackend.GetUsers(r.Context(), odataReq)
	}

	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get users from backend")
		var errcode errorcode.Error
		var godataerr *godata.GoDataError
		switch {
		case errors.As(err, &errcode):
			errcode.Render(w, r)
		case errors.As(err, &godataerr):
			errorcode.GeneralException.Render(w, r, godataerr.ResponseCode, err.Error())
		default:
			errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// If the user isn't admin, we'll show just the minimum user attibutes
	if !ctxHasFullPerms {
		finalUsers := make([]*libregraph.User, len(users))
		for i, u := range users {
			finalUsers[i] = &libregraph.User{
				Id:          u.Id,
				DisplayName: u.DisplayName,
				UserType:    u.UserType,
				Identities:  u.Identities,
			}

			if g.config.API.ShowUserEmailInResults {
				finalUsers[i].Mail = u.Mail
			}
		}
		users = finalUsers
	}

	exp, err := odata.GetExpandValues(odataReq.Query)
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get users: $expand error")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}
	expandAppRoleAssignments := slices.Contains(exp, appRoleAssignments)
	expandMemberOf := slices.Contains(exp, "memberOf")
	for _, u := range users {
		if expandAppRoleAssignments && u.AppRoleAssignments == nil {
			u.AppRoleAssignments, err = g.fetchAppRoleAssignments(r.Context(), u.GetId())
			if err != nil {
				// TODO I think we should not continue here, see http://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_SystemQueryOptionexpand
				// > The $expand system query option indicates the related entities and stream values that MUST be represented inline. The service MUST return the specified content, and MAY choose to return additional information.
				logger.Debug().Err(err).Str("userid", u.GetId()).Msg("could not get appRoleAssignments when listing user")
			}
		}
		if expandMemberOf {
			if u.MemberOf == nil {
				u.MemberOf = []libregraph.Group{}
			}
		}
	}

	users, err = sortUsers(odataReq, users)
	if err != nil {
		logger.Debug().Interface("query", odataReq).Msg("error while sorting users according to query")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: users})
}

// PostUser implements the Service interface.
func (g Graph) PostUser(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Debug().Msg("calling create user")
	u := libregraph.NewUser("", "")
	err := StrictJSONUnmarshal(r.Body, u)
	if err != nil {
		logger.Info().Err(err).Msg("could not create user: invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err.Error()))
		return
	}

	if u.GetDisplayName() == "" {
		logger.Info().Err(err).Interface("user", u).Msg("could not create user: empty displayname")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "Empty displayname")
		return
	}
	if username := u.GetOnPremisesSamAccountName(); !g.isValidUsername(username) {
		logger.Info().Str("username", username).Msg("could not create user: invalid username")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "Invalid username")
		return
	}
	if mail, ok := u.GetMailOk(); ok {
		if !isValidEmail(*mail) {
			logger.Info().Str("mail", *u.Mail).Msg("could not create user: invalid email address")
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid email address")
			return
		}
	}

	// Disallow user-supplied IDs. It's supposed to be readonly. We're either
	// generating them in the backend ourselves or rely on the Backend's
	// storage (e.g. LDAP) to provide a unique ID.
	if u.HasId() {
		logger.Info().Interface("user", u).Msg("could not create user: user id is a read-only attribute")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "user id is a read-only attribute")
		return
	}

	// treat memberOf as a read-only attribute.
	if u.HasMemberOf() {
		logger.Info().Interface("user", u).Msg("could not create user: memberOf id is a read-only attribute")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "memberOf id is a read-only attribute")
		return
	}

	// treat userType as a read-only attribute.
	if u.HasUserType() {
		logger.Info().Interface("user", u).Msg("could not create user: userType is a read-only attribute")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "userType is a read-only attribute")
		return
	}
	u.SetUserType("Member")

	logger.Debug().Interface("user", u).Msg("calling create user on backend")
	if u, err = g.identityBackend.CreateUser(r.Context(), *u); err != nil {
		logger.Error().Err(err).Msg("could not create user: backend error")
		errorcode.RenderError(w, r, err)
		return
	}

	// assign roles if possible
	if g.roleService != nil && g.config.API.AssignDefaultUserRole {
		// All users get the user role by default currently.
		// to all new users for now, as create Account request does not have any role field
		if _, err = g.roleService.AssignRoleToUser(r.Context(), &settingssvc.AssignRoleToUserRequest{
			AccountUuid: *u.Id,
			RoleId:      ocsettingssvc.BundleUUIDRoleUser,
		}); err != nil {
			// log as error, admin eventually needs to do something
			logger.Error().Err(err).Str("id", *u.Id).Str("role", ocsettingssvc.BundleUUIDRoleUser).Msg("could not create user: role assignment failed")
			errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "role assignment failed")
			return
		}
	}

	e := events.UserCreated{UserID: *u.Id}
	if currentUser, ok := revactx.ContextGetUser(r.Context()); ok {
		e.Executant = currentUser.GetId()
	}
	g.publishEvent(r.Context(), e)

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, u)
}

// GetUser implements the Service interface.
func (g Graph) GetUser(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Debug().Msg("calling get user")

	sanitizedPath := strings.TrimPrefix(r.URL.Path, "/graph/v1.0/")

	userID := chi.URLParam(r, "userID")
	userID, err := url.PathUnescape(userID)
	if err != nil {
		logger.Debug().Err(err).Str("id", userID).Msg("could not get user: unescaping user id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping user id failed")
		return
	}

	if userID == "" {
		logger.Debug().Msg("could not get user: missing user id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing user id")
		return
	}

	odataReq, err := godata.ParseRequest(r.Context(), sanitizedPath, r.URL.Query())
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get users: query error")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	exp, err := odata.GetExpandValues(odataReq.Query)
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get users: $expand error")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	logger.Debug().Str("id", userID).Msg("calling get user from backend")
	user, err := g.identityBackend.GetUser(r.Context(), userID, odataReq)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get user: error fetching user from backend")
		errorcode.RenderError(w, r, err)
		return
	}

	listDrives := slices.Contains(exp, "drives")
	listDrive := slices.Contains(exp, "drive")
	expandDrivePermissions := slices.Contains(exp, "drive.root.permissions")
	expandDrivesPermissions := slices.Contains(exp, "drives.root.permissions")

	// do we need to list all or only the personal drive
	filters := []*storageprovider.ListStorageSpacesRequest_Filter{}
	if listDrives || listDrive {
		filters = append(filters, listStorageSpacesUserFilter(user.GetId()))
		if !listDrives {
			// TODO filter by owner when decomposedfs supports the OWNER filter
			filters = append(filters, listStorageSpacesTypeFilter("personal"))
		}
	}

	if len(filters) > 0 {
		wdu, err := g.getWebDavBaseURL()
		if err != nil {
			// log error, wrong configuration
			logger.Error().
				Err(err).
				Str("webdav_base", g.config.Spaces.WebDavBase).
				Str("webdav_path", g.config.Spaces.WebDavPath).
				Msg("error parsing webdav URL")
			render.Status(r, http.StatusInternalServerError)
			return
		}
		logger.Debug().Str("id", user.GetId()).Msg("calling list storage spaces with filter")

		// use the unrestricted flag to get all possible spaces
		// users with the canListAllSpaces permission should see all spaces

		client, err := g.gatewaySelector.Next()
		if err != nil {
			logger.Error().Err(err).Msg("error selecting next gateway client")
			render.Status(r, http.StatusInternalServerError)
			return
		}
		opaque := utils.AppendPlainToOpaque(nil, "unrestricted", "T")
		lspr, err := client.ListStorageSpaces(r.Context(), &storageprovider.ListStorageSpacesRequest{
			Opaque:  opaque,
			Filters: filters,
		})
		if err != nil {
			// transport error, needs to be fixed by admin
			logger.Error().Err(err).Interface("query", r.URL.Query()).Msg("error getting storages: transport error")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, user)
			return
		}
		if lspr.GetStatus().GetCode() != cs3rpc.Code_CODE_OK {
			logger.Debug().Str("grpc", lspr.GetStatus().GetMessage()).Msg("could not list drives for user")
			// in case of NOT_OK, we can just return the user object with empty drives
			render.Status(r, status.HTTPStatusFromCode(http.StatusOK))
			render.JSON(w, r, user)
			return
		}
		if listDrives {
			user.Drives = make([]libregraph.Drive, 0, len(lspr.GetStorageSpaces()))
		}
		if listDrive {
			user.Drive = &libregraph.Drive{}
		}
		for _, sp := range lspr.GetStorageSpaces() {
			expandPermissions := false
			if sp.GetSpaceType() == "personal" && sp.GetOwner().GetId().GetOpaqueId() != user.GetId() {
				expandPermissions = expandDrivePermissions
			} else {
				expandPermissions = expandDrivesPermissions
			}
			d, err := g.cs3StorageSpaceToDrive(r.Context(), wdu, sp, APIVersion_1, expandPermissions, nil)
			if err != nil {
				logger.Debug().Err(err).Interface("id", sp.Id).Msg("error converting space to drive")
				continue
			}
			quota, err := g.getDriveQuota(r.Context(), sp)
			if err != nil {
				logger.Debug().Err(err).Interface("id", sp.Id).Msg("error calling get quota on drive")
			}
			d.Quota = &quota
			if d.GetDriveType() == "personal" && sp.GetOwner().GetId().GetOpaqueId() == user.GetId() {
				if listDrive {
					user.Drive = d
				}
			} else {
				if listDrives {
					user.Drives = append(user.Drives, *d)
				}
			}
		}
	}

	// expand appRoleAssignments if requested
	if slices.Contains(exp, appRoleAssignments) {
		user.AppRoleAssignments, err = g.fetchAppRoleAssignments(r.Context(), user.GetId())
		if err != nil {
			logger.Debug().Err(err).Str("userid", user.GetId()).Msg("could not get appRoleAssignments for user")
			errorcode.RenderError(w, r, err)
			return
		}
	}

	ctxHasFullPerms := g.contextUserHasFullAccountPerms(r.Context())
	if !ctxHasFullPerms && !g.config.API.ShowUserEmailInResults {
		user.Mail = nil
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, user)
}

// getUserLanguage returns the language of the user in the context.
func getUserLanguage(ctx context.Context, valueService settingssvc.ValueService, userID string) (string, string, error) {
	gvr, err := valueService.GetValueByUniqueIdentifiers(ctx, &settingssvc.GetValueByUniqueIdentifiersRequest{
		AccountUuid: userID,
		SettingId:   defaults.SettingUUIDProfileLanguage,
	})
	if err != nil {
		return "", "", err
	}

	langVal := gvr.GetValue().GetValue().GetListValue().GetValues()
	if len(langVal) > 0 && langVal[0] != nil {
		return langVal[0].GetStringValue(), gvr.GetValue().GetValue().GetId(), nil
	}
	return "", "", errors.New("no language value found")
}

// DeleteUser implements the Service interface.
func (g Graph) DeleteUser(w http.ResponseWriter, r *http.Request) {
	purgeUser := r.Header.Get("Prefer") == "purge"

	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Debug().Msg("calling delete user")
	sanitizedPath := strings.TrimPrefix(r.URL.Path, "/graph/v1.0/")
	userID := chi.URLParam(r, "userID")
	userID, err := url.PathUnescape(userID)
	if err != nil {
		logger.Debug().Err(err).Str("id", userID).Msg("could not delete user: unescaping user id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping user id failed")
		return
	}

	if userID == "" {
		logger.Debug().Msg("could not delete user: missing user id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing user id")
		return
	}

	odataReq, err := godata.ParseRequest(r.Context(), sanitizedPath, r.URL.Query())
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get users: query error")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	logger.Debug().Str("id", userID).Msg("calling get user on user backend")
	user, err := g.identityBackend.GetUser(r.Context(), userID, odataReq)
	if err != nil {
		logger.Debug().Err(err).Str("userID", userID).Msg("failed to get user from backend")
		errorcode.RenderError(w, r, err)
		return
	}

	us, err := g.getUserStateFromNatsKeyValue(r.Context(), userID)
	if err != nil {
		logger.Error().Err(err).Str("id", userID).Msg("could not get user state")
		us = userstate.UserState{
			UserId: userID,
			State:  userstate.UserStateUnspecified,
		}
	}

	if us.State == userstate.UserStateHardDeleted {
		logger.Debug().Str("id", userID).Msg("could not delete user: user already hard deleted")
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "user not found")
		return
	}

	if us.State == userstate.UserStateUnspecified {
		if user.GetAccountEnabled() {
			us.State = userstate.UserStateEnabled
		} else {
			us.State = userstate.UserStateSoftDeleted
		}
	}

	if g.config.UserSoftDeleteRetentionTime > 0 && purgeUser && us.State == userstate.UserStateEnabled {
		logger.Debug().Msg("could not delete user: purgeUser is set but user is still enabled")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "user should be hard deleted, but is still enabled, please soft delete first")
		return
	}

	currentUser, ok := revactx.ContextGetUser(r.Context())
	if !ok {
		logger.Debug().Msg("could not delete user: user not in context")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "user not in context")
		return
	}

	if currentUser.GetId().GetOpaqueId() == user.GetId() {
		logger.Debug().Msg("could not delete user: self deletion forbidden")
		errorcode.NotAllowed.Render(w, r, http.StatusForbidden, "self deletion forbidden")
		return
	}

	if g.gatewaySelector != nil {
		logger.Debug().
			Str("user", user.GetId()).
			Msg("calling list spaces with user filter to fetch the personal space for deletion")
		opaque := utils.AppendPlainToOpaque(nil, "unrestricted", "T")
		f := listStorageSpacesUserFilter(user.GetId())
		client, err := g.gatewaySelector.Next()
		if err != nil {
			logger.Error().Err(err).Msg("error selecting next gateway client")
			errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, "error selecting next gateway client, aborting")
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
			// the space will if the system does not have a UserSoftDeleteRetentionTime configured, e.g. SoftDelete disabled
			if g.config.UserSoftDeleteRetentionTime == 0 || (purgeUser && us.State == userstate.UserStateSoftDeleted) {
				purgeSpaceFlag := utils.AppendPlainToOpaque(nil, "purge", "")
				_, err := client.DeleteStorageSpace(r.Context(), &storageprovider.DeleteStorageSpaceRequest{
					Opaque: purgeSpaceFlag,
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
			}
			break
		}
	}

	if g.config.UserSoftDeleteRetentionTime == 0 || (purgeUser && us.State == userstate.UserStateSoftDeleted) {
		logger.Debug().Str("id", user.GetId()).Msg("calling delete user on backend")
		err = g.identityBackend.DeleteUser(r.Context(), user.GetId())
		if err != nil {
			logger.Debug().Err(err).Msg("could not delete user: backend error")
			errorcode.RenderError(w, r, err)
			return
		}

		us.State = userstate.UserStateHardDeleted
		err = g.setUserStateToNatsKeyValue(r.Context(), userID, us)
		if err != nil {
			logger.Error().Err(err).Str("id", userID).Msg("could not set user state")
			errorcode.RenderError(w, r, err)
		}
	} else {
		logger.Debug().Str("id", user.GetId()).Msg("calling soft delete user on backend")
		userUpdate := *libregraph.NewUserUpdate()
		userUpdate.AccountEnabled = libregraph.PtrBool(false)
		us.State = userstate.UserStateSoftDeleted
		us.RetentionPeriod = g.config.UserSoftDeleteRetentionTime
		us.Reason = "User soft deleted via Graph API" // TODO: this needs a proper implementation through the request
		us.TimeStamp = time.Now()
		err = g.setUserStateToNatsKeyValue(r.Context(), userID, us)
		if err != nil {
			logger.Error().Err(err).Str("id", userID).Msg("could not set user state")
			errorcode.RenderError(w, r, err)
			return
		}
		g.identityBackend.UpdateUser(r.Context(), user.GetId(), userUpdate)
	}

	if g.config.UserSoftDeleteRetentionTime == 0 ||
		(g.config.UserSoftDeleteRetentionTime > 0 && purgeUser && us.State == userstate.UserStateSoftDeleted) {
		e := events.UserDeleted{UserID: user.GetId()}
		e.Executant = currentUser.GetId()
		g.publishEvent(r.Context(), e)
	} else {
		e := events.UserSoftDeleted{
			UserID:        user.GetId(),
			RetentionTime: g.config.UserSoftDeleteRetentionTime,
			Timestamp: &v1beta1.Timestamp{
				Seconds: uint64(time.Now().Unix()),
				Nanos:   uint32(time.Now().Nanosecond()),
			},
			Reason: "User deleted via Graph API", // TODO: this needs a proper implementation through the request
		}
		e.Executant = currentUser.GetId()
		g.publishEvent(r.Context(), e)
	}
	render.Status(r, http.StatusNoContent)
	render.NoContent(w, r)
}

// PatchMe implements the Service Interface. Updates the specified attributes of the current user
func (g Graph) PatchMe(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Debug().Msg("calling patch me")
	userID := revactx.ContextMustGetUser(r.Context()).GetId().GetOpaqueId()
	if userID == "" {
		logger.Debug().Msg("could not update user: missing user id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing user id")
		return
	}
	changes := libregraph.NewUserUpdate()
	err := StrictJSONUnmarshal(r.Body, changes)
	if err != nil {
		logger.Debug().Err(err).Interface("body", r.Body).Msg("could not update user: invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest,
			fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if _, ok := changes.GetDisplayNameOk(); ok {
		logger.Info().Interface("user", changes).Msg("could not update user: user is not allowed to change own displayname")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "user is not allowed to change own displayname")
		return
	}
	if _, ok := changes.GetMailOk(); ok {
		logger.Info().Interface("user", changes).Msg("could not update user: user is not allowed to change own mail")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "user is not allowed to change own mail")
		return
	}
	g.patchUser(w, r, userID, changes)
}

// PatchUser implements the Service Interface. Updates the specified attributes of an
// ExistingUser
func (g Graph) PatchUser(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	nameOrID := chi.URLParam(r, "userID")
	nameOrID, err := url.PathUnescape(nameOrID)
	if err != nil {
		logger.Debug().Err(err).Str("id", nameOrID).Msg("could not update user: unescaping user id failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unescaping user id failed")
		return
	}
	changes := libregraph.NewUserUpdate()
	err = StrictJSONUnmarshal(r.Body, changes)
	if err != nil {
		logger.Debug().Err(err).Interface("body", r.Body).Msg("could not update user: invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest,
			fmt.Sprintf("invalid request body: %s", err.Error()))
		return
	}
	if _, ok := changes.GetPreferredLanguageOk(); ok {
		logger.Info().Interface("user", changes).Msg("could not update user: user is not allowed to change other users language")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "user is not allowed to change other users language")
		return
	}

	g.patchUser(w, r, nameOrID, changes)
}

func (g Graph) patchUser(w http.ResponseWriter, r *http.Request, nameOrID string, changes *libregraph.UserUpdate) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Debug().Msg("calling patch user")

	sanitizedPath := strings.TrimPrefix(r.URL.Path, "/graph/v1.0/")

	odataReq, err := godata.ParseRequest(r.Context(), sanitizedPath, r.URL.Query())
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get users: query error")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
		return
	}

	oldUserValues, err := g.identityBackend.GetUser(r.Context(), nameOrID, odataReq)
	if err != nil {
		logger.Debug().Err(err).Str("id", nameOrID).Msg("could not get user: backend error")
		errorcode.RenderError(w, r, err)
		return
	}

	if nameOrID == "" {
		logger.Debug().Msg("could not update user: missing user id")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "missing user id")
		return
	}

	if reflect.ValueOf(*changes).IsZero() {
		logger.Debug().Interface("body", r.Body).Msg("ignoring empty request body")
		render.Status(r, http.StatusNoContent)
		render.NoContent(w, r)
		return
	}

	if changes.HasId() {
		logger.Debug().Msg("could not update user: user id is a read-only attribute")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "user id is a read-only attribute")
		return
	}

	if changes.HasMemberOf() {
		logger.Debug().Msg("could not update user: memberOf id is a read-only attribute")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "memberOf id is a read-only attribute")
		return
	}

	if changes.HasUserType() {
		logger.Debug().Msg("could not update user: userType is a read-only attribute")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "userType is a read-only attribute")
		return
	}

	if accountName, ok := changes.GetOnPremisesSamAccountNameOk(); ok {
		if !g.isValidUsername(*accountName) {
			logger.Info().Str("username", *accountName).Msg("could not update user: invalid username")
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "Invalid username")
			return
		}
	}

	var features []events.UserFeature
	addfeature := func(name string, value string, oldvalue *string) {
		features = append(features, events.UserFeature{
			Name:     name,
			Value:    value,
			OldValue: oldvalue,
		})
	}

	preferredLanguage, ok := changes.GetPreferredLanguageOk()
	if ok {
		_, vID, err := getUserLanguage(r.Context(), g.valueService, oldUserValues.GetId())
		if err != nil {
			logger.Error().Err(err).Msg("could not get user language")
			tvID, err := uuid.NewUUID()
			if err != nil {
				logger.Error().Err(err).Msg("could not create user: error generating uuid")
				errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "error generating uuid")
				return
			}
			vID = tvID.String()
		}
		_, err = g.valueService.SaveValue(r.Context(), &settingssvc.SaveValueRequest{
			Value: &settingsmsg.Value{
				Id:          vID,
				BundleId:    defaults.BundleUUIDProfile,
				SettingId:   defaults.SettingUUIDProfileLanguage,
				AccountUuid: oldUserValues.GetId(),
				Resource: &settingsmsg.Resource{
					Type: settingsmsg.Resource_TYPE_USER,
				},
				Value: &settingsmsg.Value_ListValue{
					ListValue: &settingsmsg.ListValue{Values: []*settingsmsg.ListOptionValue{
						{
							Option: &settingsmsg.ListOptionValue_StringValue{
								StringValue: *preferredLanguage,
							},
						},
					}},
				},
			},
		})
		if err != nil {
			logger.Error().Err(err).Msg("could not update user: error saving language setting")
			errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "error saving language setting")
			return
		}

		addfeature("preferredLanguage", *preferredLanguage, nil) // do we need to get the old value?

		// if there are no more changes we can exit here
		m, _ := changes.ToMap() // always returns nil error
		if len(m) == 1 {
			oldUserValues.PreferredLanguage = preferredLanguage
			g.patchUserResponse(w, r, oldUserValues, features)
			return
		}
	}

	if mail, ok := changes.GetMailOk(); ok {
		if !isValidEmail(*mail) {
			logger.Debug().Str("mail", *mail).Msg("could not update user: email is not a valid email address")
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest,
				fmt.Sprintf("'%s' is not a valid email address", *mail))
			return
		}
		addfeature("email", *mail, oldUserValues.Mail)
	}

	if name, ok := changes.GetDisplayNameOk(); ok {
		addfeature("displayname", *name, &oldUserValues.DisplayName)
	}

	if accEnabled, ok := changes.GetAccountEnabledOk(); ok {
		oldAccVal := strconv.FormatBool(oldUserValues.GetAccountEnabled())
		addfeature("accountEnabled", strconv.FormatBool(*accEnabled), &oldAccVal)
	}

	if changes.HasPasswordProfile() {
		addfeature("passwordChanged", "", nil)
	}

	logger.Debug().Str("nameid", nameOrID).Interface("changes", *changes).Msg("calling update user on backend")
	u, err := g.identityBackend.UpdateUser(r.Context(), nameOrID, *changes)
	if err != nil {
		logger.Debug().Err(err).Str("id", nameOrID).Msg("could not update user: backend error")
		errorcode.RenderError(w, r, err)
		return
	}

	u.PreferredLanguage = preferredLanguage
	g.patchUserResponse(w, r, u, features)
}

func (g *Graph) patchUserResponse(w http.ResponseWriter, r *http.Request, user *libregraph.User, features []events.UserFeature) {
	e := events.UserFeatureChanged{
		UserID:    *user.Id,
		Features:  features,
		Timestamp: utils.TSNow(),
	}

	if currentUser, ok := revactx.ContextGetUser(r.Context()); ok {
		e.Executant = currentUser.GetId()
	}

	g.publishEvent(r.Context(), e)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, user)
}

const (
	usernameMatchDefault = "default"
	usernameMatchNone    = "none"
)

var usernameRegexes = map[string]*regexp.Regexp{
	// We want to allow email addresses as usernames so they show up when using them in ACLs on storages that allow integration with our glauth LDAP service
	// so we are adding a few restrictions from https://stackoverflow.com/questions/6949667/what-are-the-real-rules-for-linux-usernames-on-centos-6-and-rhel-6
	// names should not start with numbers
	usernameMatchDefault: regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]*(@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*)*$"),

	// In some cases users will be provisioned from an existing system, which may or may not have strange usernames. Because of this we want to "trust" the
	// upstream system and allow weird usernames, so relying on the used identity provider to complain if a username is violating its restrictions.
	usernameMatchNone: regexp.MustCompile(".*"),
}

func (g Graph) isValidUsername(e string) bool {
	if len(e) < 1 && len(e) > 254 {
		return false
	}

	return usernameRegexes[g.config.API.UsernameMatch].MatchString(e)
}

// regex from https://www.w3.org/TR/2016/REC-html51-20161101/sec-forms.html#valid-e-mail-address
var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func isValidEmail(e string) bool {
	if len(e) < 3 && len(e) > 254 {
		return false
	}
	return emailRegex.MatchString(e)
}

func sortUsers(req *godata.GoDataRequest, users []*libregraph.User) ([]*libregraph.User, error) {
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

func isValidUserType(userType string) bool {
	userType = strings.ToLower(userType)

	for _, value := range []string{"member", "guest"} {
		if userType == value {
			return true
		}
	}

	return false
}

func (g Graph) searchOCMAcceptedUsers(ctx context.Context, odataReq *godata.GoDataRequest) ([]*libregraph.User, error) {
	logger := g.logger.SubloggerWithRequestID(ctx)
	users := []*libregraph.User{}

	if !g.config.IncludeOCMSharees {
		logger.Debug().Msg("OCM sharing is disabled")
		return users, nil
	}

	gwc, err := g.gatewaySelector.Next()
	if err != nil {
		return nil, errorcode.New(errorcode.GeneralException, err.Error())
	}
	term, err := odata.GetSearchValues(odataReq.Query)
	if err != nil {
		return nil, errorcode.New(errorcode.InvalidRequest, err.Error())
	}

	remoteUsersRes, err := gwc.FindAcceptedUsers(ctx, &invitepb.FindAcceptedUsersRequest{Filter: term})
	if err != nil {
		logger.Error().Err(err).Msg("FindAcceptedUsers call failed")
		return nil, errorcode.New(errorcode.GeneralException, err.Error())
	}
	if remoteUsersRes.GetStatus().GetCode() != cs3rpc.Code_CODE_OK {
		return nil, errorcode.FromCS3Status(remoteUsersRes.Status, nil)
	}

	cs3Users := remoteUsersRes.GetAcceptedUsers()
	users = make([]*libregraph.User, 0, len(cs3Users))
	for _, user := range cs3Users {
		users = append(users, identity.CreateUserModelFromCS3(user))
	}
	return users, nil
}

// getUserStateFromNatsKeyValue gets the user state from the nats key value store.
func (g Graph) getUserStateFromNatsKeyValue(ctx context.Context, userID string) (userstate.UserState, error) {
	logger := g.logger.SubloggerWithRequestID(ctx)
	if g.natskv == nil {
		logger.Debug().Msg("nats connection or user state key value store not configured")
		return userstate.UserState{
			UserId: userID,
			State:  userstate.UserStateUnspecified,
		}, nil
	}

	entry, err := g.natskv.Get(userID)
	if err != nil {
		if errors.Is(err, nats.ErrKeyNotFound) {
			logger.Debug().Str("userid", userID).Msg("no user state found in nats key value store")
			return userstate.UserState{
				UserId: userID,
				State:  userstate.UserStateUnspecified,
			}, nil
		}
		logger.Error().Err(err).Str("userid", userID).Msg("error getting user state from nats key value store")
		return userstate.UserState{
			State: userstate.UserStateUnspecified,
		}, err
	}

	userState := userstate.UserState{}
	if err := json.Unmarshal(entry.Value(), userState); err != nil {
		logger.Error().Err(err).Str("userid", userID).Msg("error unmarshalling user state from nats key value store")
		return userstate.UserState{
			UserId: userID,
			State:  userstate.UserStateUnspecified,
		}, err
	}

	return userState, nil
}

// setUserStateToNatsKeyValue sets the user state in the nats key value store.
func (g Graph) setUserStateToNatsKeyValue(ctx context.Context, userID string, us userstate.UserState) error {
	logger := g.logger.SubloggerWithRequestID(ctx)

	if ok, err := userstate.IsValidUserState(&us); !ok {
		logger.Debug().Str("userid", userID).Msg("invalid user state")
		return fmt.Errorf("invalid user state: %w", err)
	}

	if g.natskv == nil {
		logger.Debug().Msg("nats connection or user state key value store not configured")
		return nil
	}

	data, err := json.Marshal(us)
	if err != nil {
		logger.Error().Err(err).Str("userid", userID).Msg("error marshalling user state to nats key value store")
		return err
	}

	if _, err := g.natskv.Put(userID, data); err != nil {
		logger.Error().Err(err).Str("userid", userID).Msg("error putting user state to nats key value store")
		return err
	}

	return nil
}
