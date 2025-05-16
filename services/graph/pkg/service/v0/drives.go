package svc

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/CiscoM31/godata"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-chi/render"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/pkg/errors"
	merrors "go-micro.dev/v4/errors"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/opencloud-eu/opencloud/pkg/l10n"
	v0 "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	settingsServiceExt "github.com/opencloud-eu/opencloud/services/settings/pkg/store/defaults"
)

const (
	_spaceTypePersonal   = "personal"
	_spaceTypeProject    = "project"
	_spaceTypeVirtual    = "virtual"
	_spaceTypeMountpoint = "mountpoint"
	_spaceStateTrashed   = "trashed"

	_sortDescending = "desc"
)

var (
	_invalidSpaceNameCharacters = []string{`/`, `\`, `.`, `:`, `?`, `*`, `"`, `>`, `<`, `|`}
	_maxSpaceNameLength         = 255

	// ErrNameTooLong is thrown when the spacename is too long
	ErrNameTooLong = fmt.Errorf("spacename must be smaller than %d", _maxSpaceNameLength)

	// ErrNameEmpty is thrown when the spacename is empty
	ErrNameEmpty = errors.New("spacename must not be empty")

	// ErrForbiddenCharacter is thrown when the spacename contains an invalid character
	ErrForbiddenCharacter = fmt.Errorf("spacenames must not contain %v", _invalidSpaceNameCharacters)
)

// GetDrives serves as a factory method that returns the appropriate
// http.Handler function based on the specified API version.
func (g Graph) GetDrives(version APIVersion) http.HandlerFunc {
	switch version {
	case APIVersion_1:
		return g.GetDrivesV1
	case APIVersion_1_Beta_1:
		return g.GetDrivesV1Beta1
	default:
		return func(w http.ResponseWriter, r *http.Request) {
			errorcode.New(errorcode.NotSupported, "api version not supported").Render(w, r)
		}
	}
}

// GetDrivesV1 attempts to retrieve the current users drives;
// it lists all drives the current user has access to.
func (g Graph) GetDrivesV1(w http.ResponseWriter, r *http.Request) {
	spaces, errCode := g.getDrives(r, false, APIVersion_1)
	if errCode != nil {
		errorcode.RenderError(w, r, errCode)
		return
	}

	render.Status(r, http.StatusOK)

	switch {
	case spaces == nil && errCode == nil:
		render.JSON(w, r, nil)
	default:
		render.JSON(w, r, &ListResponse{Value: spaces})
	}

}

// GetDrivesV1Beta1 is the same as the GetDrivesV1 endpoint, expect:
// it includes the grantedtoV2 property
// it uses unified roles instead of the cs3 representations
func (g Graph) GetDrivesV1Beta1(w http.ResponseWriter, r *http.Request) {
	spaces, errCode := g.getDrives(r, false, APIVersion_1_Beta_1)
	if errCode != nil {
		errorcode.RenderError(w, r, errCode)
		return
	}

	render.Status(r, http.StatusOK)

	switch {
	case spaces == nil && errCode == nil:
		render.JSON(w, r, nil)
	default:
		render.JSON(w, r, &ListResponse{Value: spaces})
	}
}

// GetAllDrives serves as a factory method that returns the appropriate
// http.Handler function based on the specified API version.
func (g Graph) GetAllDrives(version APIVersion) http.HandlerFunc {
	switch version {
	case APIVersion_1:
		return g.GetAllDrivesV1
	case APIVersion_1_Beta_1:
		return g.GetAllDrivesV1Beta1
	default:
		return func(w http.ResponseWriter, r *http.Request) {
			errorcode.New(errorcode.NotSupported, "api version not supported").Render(w, r)
		}
	}
}

// GetAllDrivesV1 attempts to retrieve the current users drives;
// it includes another user's drives, if the current user has the permission.
func (g Graph) GetAllDrivesV1(w http.ResponseWriter, r *http.Request) {
	spaces, errCode := g.getDrives(r, true, APIVersion_1)
	if errCode != nil {
		errorcode.RenderError(w, r, errCode)
		return
	}

	render.Status(r, http.StatusOK)

	switch {
	case spaces == nil && errCode == nil:
		render.JSON(w, r, nil)
	default:
		render.JSON(w, r, &ListResponse{Value: spaces})
	}
}

// GetAllDrivesV1Beta1 is the same as the GetAllDrivesV1 endpoint, expect:
// it includes the grantedtoV2 property
// it uses unified roles instead of the cs3 representations
func (g Graph) GetAllDrivesV1Beta1(w http.ResponseWriter, r *http.Request) {
	drives, errCode := g.getDrives(r, true, APIVersion_1_Beta_1)
	if errCode != nil {
		errorcode.RenderError(w, r, errCode)
		return
	}

	render.Status(r, http.StatusOK)

	switch {
	case drives == nil && errCode == nil:
		render.JSON(w, r, nil)
	default:
		render.JSON(w, r, &ListResponse{Value: drives})
	}
}

// getDrives implements the Service interface.
func (g Graph) getDrives(r *http.Request, unrestricted bool, apiVersion APIVersion) ([]*libregraph.Drive, error) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().
		Interface("query", r.URL.Query()).
		Bool("unrestricted", unrestricted).
		Msg("calling get drives")
	sanitizedPath := strings.TrimPrefix(r.URL.Path, "/graph/v1.0/")
	// Parse the request with odata parser
	odataReq, err := godata.ParseRequest(r.Context(), sanitizedPath, r.URL.Query())
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get drives: query error")
		return nil, errorcode.New(errorcode.InvalidRequest, err.Error())
	}
	ctx := r.Context()

	filters, err := generateCs3Filters(odataReq)
	if err != nil {
		logger.Debug().Err(err).Interface("query", r.URL.Query()).Msg("could not get drives: error parsing filters")
		return nil, errorcode.New(errorcode.NotSupported, err.Error())
	}
	if !unrestricted {
		user, ok := revactx.ContextGetUser(r.Context())
		if !ok {
			logger.Debug().Msg("could not create drive: invalid user")
			return nil, errorcode.New(errorcode.AccessDenied, "invalid user")
		}
		filters = append(filters, &storageprovider.ListStorageSpacesRequest_Filter{
			Type: storageprovider.ListStorageSpacesRequest_Filter_TYPE_USER,
			Term: &storageprovider.ListStorageSpacesRequest_Filter_User{
				User: user.GetId(),
			},
		})
	}

	logger.Debug().
		Interface("filters", filters).
		Bool("unrestricted", unrestricted).
		Msg("calling list storage spaces on backend")
	res, err := g.ListStorageSpacesWithFilters(ctx, filters, unrestricted)
	switch {
	case err != nil:
		logger.Error().Err(err).Msg("could not get drives: transport error")
		return nil, errorcode.New(errorcode.GeneralException, err.Error())
	case res.Status.Code != cs3rpc.Code_CODE_OK:
		if res.Status.Code == cs3rpc.Code_CODE_NOT_FOUND {
			// ok, empty return
			return nil, nil
		}
		logger.Debug().Str("message", res.GetStatus().GetMessage()).Msg("could not get drives: grpc error")
		return nil, errorcode.New(errorcode.GeneralException, res.Status.Message)
	}

	webDavBaseURL, err := g.getWebDavBaseURL()
	if err != nil {
		logger.Error().Err(err).Str("url", webDavBaseURL.String()).Msg("could not get drives: error parsing url")
		return nil, errorcode.New(errorcode.GeneralException, err.Error())
	}

	spaces, err := g.formatDrives(ctx, webDavBaseURL, res.StorageSpaces, apiVersion)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get drives: error parsing grpc response")
		return nil, errorcode.New(errorcode.GeneralException, err.Error())
	}

	spaces, err = sortSpaces(odataReq, spaces)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get drives: error sorting the spaces list according to query")
		return nil, errorcode.New(errorcode.InvalidRequest, err.Error())
	}

	return spaces, nil
}

// GetSingleDrive does a lookup of a single space by spaceId
func (g Graph) GetSingleDrive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := g.logger.SubloggerWithRequestID(ctx)
	logger.Info().Interface("query", r.URL.Query()).Msg("calling get drive")

	rid, err := parseIDParam(r, "driveID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}
	log := logger.With().Str("storage", rid.StorageId).Str("space", rid.SpaceId).Str("node", rid.OpaqueId).Logger()

	log.Debug().Msg("calling list storage spaces with id filter")

	filters := []*storageprovider.ListStorageSpacesRequest_Filter{
		listStorageSpacesIDFilter(storagespace.FormatResourceID(&rid)),
	}
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
			log.Debug().Msg("could not get drive: not found")
			errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "drive not found")
			return
		}
		log.Debug().
			Str("grpcmessage", res.GetStatus().GetMessage()).
			Msg("could not get drive: grpc error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, res.Status.Message)
		return
	}

	webDavBaseURL, err := g.getWebDavBaseURL()
	if err != nil {
		log.Error().Err(err).Str("url", webDavBaseURL.String()).Msg("could not get drive: error parsing webdav base url")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	spaces, err := g.formatDrives(ctx, webDavBaseURL, res.StorageSpaces, APIVersion_1)
	if err != nil {
		log.Debug().Err(err).Msg("could not get drive: error parsing grpc response")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	switch num := len(spaces); {
	case num == 0:
		log.Debug().Msg("could not get drive: no drive returned from storage")
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "no drive returned from storage")
		return
	case num == 1:
		render.Status(r, http.StatusOK)
		render.JSON(w, r, spaces[0])
	default:
		log.Debug().Int("number", num).Msg("could not get drive: expected to find a single drive but fetched more")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not get drive: expected to find a single drive but fetched more")
		return
	}
}

func (g Graph) canCreateSpace(ctx context.Context, ownPersonalHome bool) bool {
	pr, err := g.permissionsService.GetPermissionByID(ctx, &settingssvc.GetPermissionByIDRequest{
		PermissionId: settingsServiceExt.CreateSpacesPermission(0).Id,
	})
	if err != nil || pr.Permission == nil {
		return false
	}
	// TODO @C0rby shouldn't the permissions service check this? aka shouldn't we call CheckPermission?
	if pr.Permission.Constraint == v0.Permission_CONSTRAINT_OWN && !ownPersonalHome {
		return false
	}
	return true
}

// CreateDrive creates a storage drive (space).
func (g Graph) CreateDrive(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Msg("calling create drive")

	ctx := r.Context()

	us, ok := revactx.ContextGetUser(ctx)
	if !ok {
		logger.Debug().Msg("could not create drive: invalid user")
		errorcode.NotAllowed.Render(w, r, http.StatusUnauthorized, "invalid user")
		return
	}

	// TODO determine if the user tries to create his own personal space and pass that as a boolean
	canCreateSpace := g.canCreateSpace(ctx, false)
	if !canCreateSpace {
		logger.Debug().Bool("cancreatespace", canCreateSpace).Msg("could not create drive: insufficient permissions")
		// if the permission is not existing for the user in context we can assume we don't have it. Return 401.
		errorcode.NotAllowed.Render(w, r, http.StatusForbidden, "insufficient permissions to create a space.")
		return
	}

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		logger.Error().Err(err).Msg("could not select next gateway client")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, "could not select next gateway client, aborting")
		return
	}

	drive := libregraph.Drive{}
	if err := StrictJSONUnmarshal(r.Body, &drive); err != nil {
		logger.Debug().Err(err).Interface("body", r.Body).Msg("could not create drive: invalid body schema definition")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid body schema definition")
		return
	}
	spaceName := strings.TrimSpace(drive.Name)
	if err := validateSpaceName(spaceName); err != nil {
		logger.Debug().Str("name", spaceName).Err(err).Msg("could not create drive: name validation failed")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("invalid spacename: %s", err.Error()))
		return
	}

	var driveType string
	if drive.DriveType != nil {
		driveType = *drive.DriveType
	}
	switch driveType {
	case "", _spaceTypeProject:
		driveType = _spaceTypeProject
	default:
		logger.Debug().Str("type", driveType).Msg("could not create drive: drives of this type cannot be created via this api")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "drives of this type cannot be created via this api")
		return
	}

	csr := storageprovider.CreateStorageSpaceRequest{
		Type:  driveType,
		Name:  spaceName,
		Quota: getQuota(drive.Quota, g.config.Spaces.DefaultQuota),
	}

	if drive.Description != nil {
		csr.Opaque = utils.AppendPlainToOpaque(csr.Opaque, "description", *drive.Description)
	}

	if drive.DriveAlias != nil {
		csr.Opaque = utils.AppendPlainToOpaque(csr.Opaque, "spaceAlias", *drive.DriveAlias)
	}

	if driveType == _spaceTypePersonal {
		csr.Owner = us
	}

	resp, err := gatewayClient.CreateStorageSpace(ctx, &csr)
	if err != nil {
		logger.Error().Err(err).Msg("could not create drive: transport error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	if resp.GetStatus().GetCode() != cs3rpc.Code_CODE_OK {
		if resp.GetStatus().GetCode() == cs3rpc.Code_CODE_PERMISSION_DENIED {
			logger.Debug().Str("grpcmessage", resp.GetStatus().GetMessage()).Msg("could not create drive: permission denied")
			errorcode.NotAllowed.Render(w, r, http.StatusForbidden, "permission denied")
			return
		}
		if resp.GetStatus().GetCode() == cs3rpc.Code_CODE_INVALID_ARGUMENT {
			logger.Debug().Str("grpcmessage", resp.GetStatus().GetMessage()).Msg("could not create drive: bad request")
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, resp.GetStatus().GetMessage())
			return
		}
		logger.Debug().Interface("grpcmessage", csr).Str("grpc", resp.GetStatus().GetMessage()).Msg("could not create drive: grpc error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, resp.GetStatus().GetMessage())
		return
	}

	webDavBaseURL, err := g.getWebDavBaseURL()
	if err != nil {
		logger.Error().Str("url", webDavBaseURL.String()).Err(err).Msg("could not create drive: error parsing webdav base url")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	space := resp.GetStorageSpace()
	if t := r.URL.Query().Get(TemplateParameter); t != "" && driveType == _spaceTypeProject {
		loc := l10n.MustGetUserLocale(ctx, us.GetId().GetOpaqueId(), r.Header.Get(HeaderAcceptLanguage), g.valueService)
		if err := g.applySpaceTemplate(ctx, gatewayClient, space.GetRoot(), t, loc); err != nil {
			logger.Error().Err(err).Msg("could not apply template to space")
			errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
			return
		}

		// refetch the drive to get quota information - should we calculate this ourselves to avoid the extra call?
		space, err = utils.GetSpace(ctx, space.GetId().GetOpaqueId(), gatewayClient)
		if err != nil {
			logger.Error().Err(err).Msg("could not refetch space")
			errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}

	spaces, err := g.formatDrives(ctx, webDavBaseURL, []*storageprovider.StorageSpace{space}, APIVersion_1)
	if err != nil {
		logger.Debug().Err(err).Msg("could not get drive: error parsing grpc response")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if len(spaces) == 0 {
		logger.Error().Msg("could not convert space")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "could not convert space")
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, spaces[0])
}

// UpdateDrive updates the properties of a storage drive (space).
func (g Graph) UpdateDrive(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Msg("calling update drive")

	rid, err := parseIDParam(r, "driveID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	drive := libregraph.DriveUpdate{}
	if err = StrictJSONUnmarshal(r.Body, &drive); err != nil {
		logger.Debug().Err(err).Interface("body", r.Body).Msg("could not update drive, invalid request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, fmt.Sprintf("invalid request body: error: %v", err.Error()))
		return
	}

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("could not select next gateway client")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, "could not select next gateway client, aborting")
		return
	}

	root := &rid
	updateSpaceRequest := &storageprovider.UpdateStorageSpaceRequest{
		// Prepare the object to apply the diff from. The properties on StorageSpace will overwrite
		// the original storage space.
		StorageSpace: &storageprovider.StorageSpace{
			Id: &storageprovider.StorageSpaceId{
				OpaqueId: storagespace.FormatResourceID(&rid),
			},
			Root: root,
		},
	}

	// Note: this is the Opaque prop of the request
	if restore, _ := strconv.ParseBool(r.Header.Get("restore")); restore {
		updateSpaceRequest.Opaque = &types.Opaque{
			Map: map[string]*types.OpaqueEntry{
				"restore": {
					Decoder: "plain",
					Value:   []byte("true"),
				},
			},
		}
	}

	if drive.Description != nil {
		updateSpaceRequest.StorageSpace.Opaque = utils.AppendPlainToOpaque(updateSpaceRequest.StorageSpace.Opaque, "description", *drive.Description)
	}

	if drive.DriveAlias != nil {
		updateSpaceRequest.StorageSpace.Opaque = utils.AppendPlainToOpaque(updateSpaceRequest.StorageSpace.Opaque, "spaceAlias", *drive.DriveAlias)
	}

	for _, special := range drive.Special {
		if special.Id != nil {
			updateSpaceRequest.StorageSpace.Opaque = utils.AppendPlainToOpaque(updateSpaceRequest.StorageSpace.Opaque, *special.SpecialFolder.Name, *special.Id)
		}
	}

	if drive.GetName() != "" {
		spacename := strings.TrimSpace(drive.GetName())
		if err := validateSpaceName(spacename); err != nil {
			logger.Info().Err(err).Msg("could not update drive: spacename invalid")
			errorcode.GeneralException.Render(w, r, http.StatusBadRequest, err.Error())
			return
		}

		updateSpaceRequest.StorageSpace.Name = spacename
	}

	if drive.Quota.HasTotal() {
		user := revactx.ContextMustGetUser(r.Context())

		// NOTE: a space admin cannot get a space by ID. We need to fetch all spaces and search for it
		dt := _spaceTypePersonal
		filters := []*storageprovider.ListStorageSpacesRequest_Filter{listStorageSpacesTypeFilter(_spaceTypeProject)}
		res, err := g.ListStorageSpacesWithFilters(r.Context(), filters, true)
		if err == nil && res.GetStatus().GetCode() == cs3rpc.Code_CODE_OK {
			for _, sp := range res.StorageSpaces {
				id, _ := storagespace.ParseID(sp.GetId().GetOpaqueId())
				if id.GetSpaceId() == rid.GetSpaceId() {
					dt = _spaceTypeProject
				}
			}
		}

		canSetSpaceQuota, err := g.canSetSpaceQuota(r.Context(), user, dt)
		if err != nil {
			logger.Error().Err(err).Msg("could not update drive: failed to check if the user can set space quota")
			errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		if !canSetSpaceQuota {
			logger.Debug().
				Bool("cansetspacequota", canSetSpaceQuota).
				Msg("could not update drive: user is not allowed to set the space quota")
			errorcode.NotAllowed.Render(w, r, http.StatusForbidden, "user is not allowed to set the space quota")
			return
		}
		updateSpaceRequest.StorageSpace.Quota = &storageprovider.Quota{
			QuotaMaxBytes: uint64(*drive.Quota.Total),
		}
	}

	logger.Debug().Interface("payload", updateSpaceRequest).Msg("calling update space on backend")
	resp, err := gatewayClient.UpdateStorageSpace(r.Context(), updateSpaceRequest)
	if err != nil {
		logger.Error().Err(err).Msg("could not update drive: transport error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "transport error")
		return
	}

	if resp.GetStatus().GetCode() != cs3rpc.Code_CODE_OK {
		switch resp.Status.GetCode() {
		case cs3rpc.Code_CODE_NOT_FOUND:
			logger.Debug().Interface("id", rid).Msg("could not update drive: drive not found")
			errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "drive not found")
			return
		case cs3rpc.Code_CODE_PERMISSION_DENIED:
			logger.Debug().Interface("id", rid).Msg("could not update drive, permission denied")
			errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "drive not found")
			return
		case cs3rpc.Code_CODE_INVALID_ARGUMENT:
			logger.Debug().Interface("id", rid).Msg("could not update drive, invalid argument")
			errorcode.NotAllowed.Render(w, r, http.StatusBadRequest, resp.GetStatus().GetMessage())
			return
		case cs3rpc.Code_CODE_UNIMPLEMENTED:
			logger.Debug().Interface("id", rid).Msg("could not delete drive: delete not implemented for this type of drive")
			errorcode.NotAllowed.Render(w, r, http.StatusMethodNotAllowed, "drive cannot be updated")
			return
		default:
			logger.Debug().Interface("id", rid).Str("grpc", resp.GetStatus().GetMessage()).Msg("could not update drive: grpc error")
			errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "grpc error")
			return
		}
	}

	webDavBaseURL, err := g.getWebDavBaseURL()
	if err != nil {
		logger.Error().Err(err).Interface("url", webDavBaseURL.String()).Msg("could not update drive: error parsing url")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	spaces, err := g.formatDrives(r.Context(), webDavBaseURL, []*storageprovider.StorageSpace{resp.StorageSpace}, APIVersion_1)
	if err != nil {
		logger.Debug().Err(err).Msg("could not update drive: error parsing grpc response")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, spaces[0])
}

func (g Graph) formatDrives(ctx context.Context, baseURL *url.URL, storageSpaces []*storageprovider.StorageSpace, apiVersion APIVersion) ([]*libregraph.Drive, error) {
	errg, ctx := errgroup.WithContext(ctx)
	work := make(chan *storageprovider.StorageSpace, len(storageSpaces))
	results := make(chan *libregraph.Drive, len(storageSpaces))

	// Distribute work
	errg.Go(func() error {
		defer close(work)
		for _, space := range storageSpaces {
			select {
			case work <- space:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	// Spawn workers that'll concurrently work the queue
	numWorkers := 20
	if len(storageSpaces) < numWorkers {
		numWorkers = len(storageSpaces)
	}
	for i := 0; i < numWorkers; i++ {
		errg.Go(func() error {
			for storageSpace := range work {
				if storageSpace.GetRoot().GetStorageId() == utils.OCMStorageProviderID {
					// skip OCM shares they are no supposed to show up in the drives list
					continue
				}
				res, err := g.cs3StorageSpaceToDrive(ctx, baseURL, storageSpace, apiVersion)
				if err != nil {
					return err
				}

				// can't access disabled space
				if utils.ReadPlainFromOpaque(storageSpace.Opaque, _spaceStateTrashed) != _spaceStateTrashed {
					res.Special = g.getSpecialDriveItems(ctx, baseURL, storageSpace)
					if storageSpace.SpaceType != _spaceTypeMountpoint && storageSpace.SpaceType != _spaceTypeVirtual {
						quota, err := g.getDriveQuota(ctx, storageSpace)
						res.Quota = &quota
						if err != nil {
							return err
						}
					} else {
						res.Quota = nil
					}
				}
				select {
				case results <- res:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	}

	// Wait for things to settle down, then close results chan
	go func() {
		_ = errg.Wait() // error is checked later
		close(results)
	}()

	responses := make([]*libregraph.Drive, 0, len(storageSpaces))
	for r := range results {
		responses = append(responses, r)
	}

	if err := errg.Wait(); err != nil {
		return nil, err
	}

	return responses, nil
}

// ListStorageSpacesWithFilters List Storage Spaces using filters
func (g Graph) ListStorageSpacesWithFilters(ctx context.Context, filters []*storageprovider.ListStorageSpacesRequest_Filter, unrestricted bool) (*storageprovider.ListStorageSpacesResponse, error) {

	_, err := g.permissionsService.GetPermissionByID(ctx, &settingssvc.GetPermissionByIDRequest{
		PermissionId: settingsServiceExt.ListSpacesPermission(0).Id,
	})

	permissions := make(map[string]struct{}, 1)
	// No error means the user has the permission
	if err == nil {
		permissions[settingsServiceExt.ListSpacesPermission(0).Name] = struct{}{}
	}
	value, err := json.Marshal(permissions)
	if err != nil {
		return nil, err
	}
	lReq := &storageprovider.ListStorageSpacesRequest{
		Opaque: &types.Opaque{Map: map[string]*types.OpaqueEntry{
			"permissions": {
				Decoder: "json",
				Value:   value,
			},
			"unrestricted": {
				Decoder: "plain",
				Value:   []byte(strconv.FormatBool(unrestricted)),
			},
		}},
		Filters: filters,
	}

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}
	res, err := gatewayClient.ListStorageSpaces(ctx, lReq)
	return res, err
}

func (g Graph) cs3StorageSpaceToDrive(ctx context.Context, baseURL *url.URL, space *storageprovider.StorageSpace, apiVersion APIVersion) (*libregraph.Drive, error) {
	logger := g.logger.SubloggerWithRequestID(ctx)
	if space.Root == nil {
		logger.Error().Msg("unable to parse space: space has no root")
		return nil, errors.New("space has no root")
	}
	spaceRid := proto.Clone(space.Root).(*storageprovider.ResourceId)
	if space.Root.GetSpaceId() == space.Root.GetOpaqueId() {
		spaceRid.OpaqueId = ""
	}
	spaceID := storagespace.FormatResourceID(spaceRid)

	permissions := g.cs3SpacePermissionsToLibreGraph(ctx, space, apiVersion)

	drive := &libregraph.Drive{
		Id:   libregraph.PtrString(spaceID),
		Name: space.Name,
		//"createdDateTime": "string (timestamp)", // TODO read from StorageSpace ... needs Opaque for now
		DriveType: &space.SpaceType,
		Root: &libregraph.DriveItem{
			Id:          libregraph.PtrString(storagespace.FormatResourceID(spaceRid)),
			Permissions: permissions,
		},
	}
	if space.SpaceType == _spaceTypeMountpoint {
		var remoteItem *libregraph.RemoteItem
		grantID := storageprovider.ResourceId{
			StorageId: utils.ReadPlainFromOpaque(space.Opaque, "grantStorageID"),
			SpaceId:   utils.ReadPlainFromOpaque(space.Opaque, "grantSpaceID"),
			OpaqueId:  utils.ReadPlainFromOpaque(space.Opaque, "grantOpaqueID"),
		}
		if grantID.SpaceId != "" && grantID.OpaqueId != "" {
			var err error
			remoteItem, err = g.getRemoteItem(ctx, &grantID, baseURL)
			if err != nil {
				logger.Debug().Err(err).Interface("id", grantID).Msg("could not fetch remote item for space, continue")
			}
		}
		if remoteItem != nil {
			drive.Root.RemoteItem = remoteItem
		}
	}

	if space.Opaque != nil {
		if description, ok := space.Opaque.Map["description"]; ok {
			drive.Description = libregraph.PtrString(string(description.Value))
		}

		if alias, ok := space.Opaque.Map["spaceAlias"]; ok {
			drive.DriveAlias = libregraph.PtrString(string(alias.Value))
		}

		if v, ok := space.Opaque.Map[_spaceStateTrashed]; ok {
			deleted := &libregraph.Deleted{}
			deleted.SetState(string(v.Value))
			drive.Root.Deleted = deleted
		}

		if entry, ok := space.Opaque.Map["etag"]; ok {
			drive.Root.ETag = libregraph.PtrString(string(entry.Value))
		}
	}

	if baseURL != nil {
		webDavURL := *baseURL
		webDavURL.Path = path.Join(webDavURL.Path, spaceID)
		drive.Root.WebDavUrl = libregraph.PtrString(webDavURL.String())
	}

	webURL, err := url.Parse(g.config.Spaces.WebDavBase)
	if err != nil {
		logger.Error().
			Err(err).
			Str("url", g.config.Spaces.WebDavBase).
			Msg("failed to parse webURL base url")
		return nil, err
	}

	webURL.Path = path.Join(webURL.Path, "f", storagespace.FormatResourceID(spaceRid))
	drive.WebUrl = libregraph.PtrString(webURL.String())

	if space.Owner != nil && space.Owner.Id != nil {
		drive.Owner = &libregraph.IdentitySet{
			User: &libregraph.Identity{
				Id: &space.Owner.Id.OpaqueId,
				// DisplayName: , TODO read and cache from users provider
			},
		}
	}
	if space.Mtime != nil {
		lastModified := cs3TimestampToTime(space.Mtime)
		drive.LastModifiedDateTime = &lastModified
	}

	drive.Quota = &libregraph.Quota{}
	if space.Quota != nil {
		var t int64
		if space.Quota.QuotaMaxBytes > math.MaxInt64 {
			t = math.MaxInt64
		} else {
			t = int64(space.Quota.QuotaMaxBytes)
		}
		drive.Quota.Total = &t
	}

	return drive, nil
}

func (g Graph) getDriveQuota(ctx context.Context, space *storageprovider.StorageSpace) (libregraph.Quota, error) {
	logger := g.logger.SubloggerWithRequestID(ctx)

	noQuotaInOpaque := true
	remainingUnknown := true
	var remaining, used, total int64
	if space.Opaque != nil {
		m := space.Opaque.Map
		if e, ok := m["quota.remaining"]; ok {
			noQuotaInOpaque = false
			remainingUnknown = false
			remaining, _ = strconv.ParseInt(string(e.Value), 10, 64)
		}
		if e, ok := m["quota.used"]; ok {
			noQuotaInOpaque = false
			used, _ = strconv.ParseInt(string(e.Value), 10, 64)
		}
		if e, ok := m["quota.total"]; ok {
			noQuotaInOpaque = false
			total, _ = strconv.ParseInt(string(e.Value), 10, 64)
		}

	}

	if noQuotaInOpaque {
		// we have to make a trip to the storage
		// TODO only if quota property was requested
		gatewayClient, err := g.gatewaySelector.Next()
		if err != nil {
			return libregraph.Quota{}, err
		}

		req := &gateway.GetQuotaRequest{
			Ref: &storageprovider.Reference{
				ResourceId: space.Root,
				Path:       ".",
			},
		}

		res, err := gatewayClient.GetQuota(ctx, req)
		switch {
		case err != nil:
			logger.Error().Err(err).Interface("ref", req.Ref).Msg("could not call GetQuota: transport error")
			return libregraph.Quota{}, nil
		case res.GetStatus().GetCode() == cs3rpc.Code_CODE_UNIMPLEMENTED:
			logger.Debug().Msg("get quota is not implemented on the storage driver")
			return libregraph.Quota{}, nil
		case res.GetStatus().GetCode() != cs3rpc.Code_CODE_OK:
			logger.Debug().Str("grpc", res.GetStatus().GetMessage()).Msg("error sending get quota grpc request")
			return libregraph.Quota{}, errors.New(res.GetStatus().GetMessage())
		}

		if res.Opaque != nil {
			m := res.Opaque.Map
			if e, ok := m["remaining"]; ok {
				remainingUnknown = false
				remaining, _ = strconv.ParseInt(string(e.Value), 10, 64)
			}
		}

		used = int64(res.UsedBytes)
		total = int64(res.TotalBytes)
	}

	qta := libregraph.Quota{
		Used:  &used,
		Total: &total,
	}

	if !remainingUnknown {
		qta.Remaining = &remaining
	}

	var t int64
	switch {
	case total != 0:
		t = total
	case remainingUnknown:
		t = math.MaxInt64
	default:
		// Quota was not set
		// Use remaining bytes to calculate state
		t = remaining
	}
	state := calculateQuotaState(t, used)
	qta.State = &state

	return qta, nil
}

func calculateQuotaState(total int64, used int64) (state string) {
	percent := (float64(used) / float64(total)) * 100

	switch {
	case percent <= float64(75):
		return "normal"
	case percent <= float64(90):
		return "nearing"
	case percent <= float64(99):
		return "critical"
	default:
		return "exceeded"
	}
}

func getQuota(quota *libregraph.Quota, defaultQuota string) *storageprovider.Quota {
	switch {
	case quota != nil && quota.Total != nil:
		if q := *quota.Total; q >= 0 {
			return &storageprovider.Quota{QuotaMaxBytes: uint64(q)}
		}
		fallthrough
	case defaultQuota != "":
		if q, err := strconv.ParseInt(defaultQuota, 10, 64); err == nil && q >= 0 {
			return &storageprovider.Quota{QuotaMaxBytes: uint64(q)}
		}
		fallthrough
	default:
		return nil
	}
}

func (g Graph) canSetSpaceQuota(ctx context.Context, _ *userv1beta1.User, typ string) (bool, error) {
	permID := settingsServiceExt.SetPersonalSpaceQuotaPermission(0).Id
	if typ == _spaceTypeProject {
		permID = settingsServiceExt.SetProjectSpaceQuotaPermission(0).Id
	}
	_, err := g.permissionsService.GetPermissionByID(ctx, &settingssvc.GetPermissionByIDRequest{PermissionId: permID})
	if err != nil {
		merror := merrors.FromError(err)
		if merror.Status == http.StatusText(http.StatusNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func generateCs3Filters(request *godata.GoDataRequest) ([]*storageprovider.ListStorageSpacesRequest_Filter, error) {
	var filters []*storageprovider.ListStorageSpacesRequest_Filter
	if request.Query.Filter != nil {
		if request.Query.Filter.Tree.Token.Value == "eq" {
			switch request.Query.Filter.Tree.Children[0].Token.Value {
			case "driveType":
				filters = append(filters, listStorageSpacesTypeFilter(strings.Trim(request.Query.Filter.Tree.Children[1].Token.Value, "'")))
			case "id":
				filters = append(filters, listStorageSpacesIDFilter(strings.Trim(request.Query.Filter.Tree.Children[1].Token.Value, "'")))
			}
		} else {
			err := errors.Errorf("unsupported filter operand: %s", request.Query.Filter.Tree.Token.Value)
			return nil, err
		}
	}
	return filters, nil
}

func listStorageSpacesIDFilter(id string) *storageprovider.ListStorageSpacesRequest_Filter {
	return &storageprovider.ListStorageSpacesRequest_Filter{
		Type: storageprovider.ListStorageSpacesRequest_Filter_TYPE_ID,
		Term: &storageprovider.ListStorageSpacesRequest_Filter_Id{
			Id: &storageprovider.StorageSpaceId{
				OpaqueId: id,
			},
		},
	}
}

func listStorageSpacesUserFilter(id string) *storageprovider.ListStorageSpacesRequest_Filter {
	return &storageprovider.ListStorageSpacesRequest_Filter{
		Type: storageprovider.ListStorageSpacesRequest_Filter_TYPE_USER,
		Term: &storageprovider.ListStorageSpacesRequest_Filter_User{
			User: &userv1beta1.UserId{
				OpaqueId: id,
			},
		},
	}
}

func listStorageSpacesTypeFilter(spaceType string) *storageprovider.ListStorageSpacesRequest_Filter {
	return &storageprovider.ListStorageSpacesRequest_Filter{
		Type: storageprovider.ListStorageSpacesRequest_Filter_TYPE_SPACE_TYPE,
		Term: &storageprovider.ListStorageSpacesRequest_Filter_SpaceType{
			SpaceType: spaceType,
		},
	}
}

// DeleteDrive deletes a storage drive (space).
func (g Graph) DeleteDrive(w http.ResponseWriter, r *http.Request) {
	logger := g.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Msg("calling delete drive")
	rid, err := parseIDParam(r, "driveID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	purge := parsePurgeHeader(r.Header)

	var opaque *types.Opaque
	if purge {
		opaque = &types.Opaque{
			Map: map[string]*types.OpaqueEntry{
				"purge": {},
			},
		}
	}
	gatewayClient, _ := g.gatewaySelector.Next()
	dRes, err := gatewayClient.DeleteStorageSpace(r.Context(), &storageprovider.DeleteStorageSpaceRequest{
		Opaque: opaque,
		Id: &storageprovider.StorageSpaceId{
			OpaqueId: storagespace.FormatResourceID(&rid),
		},
	})
	if err != nil {
		logger.Error().Err(err).Str("id", rid.String()).Msg("could not delete drive: transport error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "transport error")
		return
	}

	switch dRes.GetStatus().GetCode() {
	case cs3rpc.Code_CODE_OK:
		w.WriteHeader(http.StatusNoContent)
		return
	case cs3rpc.Code_CODE_INVALID_ARGUMENT:
		logger.Debug().Interface("id", rid).Str("grpc", dRes.GetStatus().GetMessage()).Msg("could not delete drive: invalid argument")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, dRes.Status.Message)
		return
	case cs3rpc.Code_CODE_PERMISSION_DENIED:
		logger.Debug().Interface("id", rid).Msg("could not delete drive: permission denied")
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "drive not found")
		return
	case cs3rpc.Code_CODE_NOT_FOUND:
		logger.Debug().Interface("id", rid).Msg("could not delete drive: drive not found")
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "drive not found")
		return
	case cs3rpc.Code_CODE_UNIMPLEMENTED:
		logger.Debug().Interface("id", rid).Msg("could not delete drive: delete not implemented for this type of drive")
		errorcode.NotAllowed.Render(w, r, http.StatusMethodNotAllowed, "drive cannot be deleted")
		return
	// don't expose internal error codes to the outside world
	default:
		logger.Debug().Str("grpc", dRes.GetStatus().GetMessage()).Interface("id", rid).Msg("could not delete drive: grpc error")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "grpc error")
		return
	}
}

func sortSpaces(req *godata.GoDataRequest, spaces []*libregraph.Drive) ([]*libregraph.Drive, error) {
	if req.Query.OrderBy == nil || len(req.Query.OrderBy.OrderByItems) != 1 {
		return spaces, nil
	}
	var less func(i, j int) bool

	switch req.Query.OrderBy.OrderByItems[0].Field.Value {
	case "name":
		less = func(i, j int) bool {
			return strings.ToLower(spaces[i].GetName()) < strings.ToLower(spaces[j].GetName())
		}
	case "lastModifiedDateTime":
		less = func(i, j int) bool {
			return lessSpacesByLastModifiedDateTime(spaces[i], spaces[j])
		}
	default:
		return nil, errors.Errorf("we do not support <%s> as a order parameter", req.Query.OrderBy.OrderByItems[0].Field.Value)
	}

	if req.Query.OrderBy.OrderByItems[0].Order == _sortDescending {
		sort.Slice(spaces, reverse(less))
	} else {
		sort.Slice(spaces, less)
	}
	return spaces, nil
}

func validateSpaceName(name string) error {
	if name == "" {
		return ErrNameEmpty
	}

	if len(name) > _maxSpaceNameLength {
		return ErrNameTooLong
	}

	for _, c := range _invalidSpaceNameCharacters {
		if strings.Contains(name, c) {
			return ErrForbiddenCharacter
		}
	}

	return nil
}
