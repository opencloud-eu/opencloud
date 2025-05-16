package svc

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	ocm "github.com/cs3org/go-cs3apis/cs3/sharing/ocm/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/render"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

const (
	_fieldMaskPathState      = "state"
	_fieldMaskPathMountPoint = "mount_point"
	_fieldMaskPathHidden     = "hidden"
)

var (
	// ErrNoUpdates is returned when no updates are provided
	ErrNoUpdates = errors.New("no updates")

	// ErrNoUpdater is returned when no updater is provided
	ErrNoUpdater = errors.New("no updater")

	// ErrAbsoluteNamePath is returned when the name is an absolute path
	ErrAbsoluteNamePath = errors.New("name cannot be an absolute path")

	// ErrCode errors

	// ErrNotAShareJail is returned when the driveID does not belong to a share jail
	ErrNotAShareJail = errorcode.New(errorcode.InvalidRequest, "id does not belong to a share jail")

	// ErrInvalidDriveIDOrItemID is returned when the driveID or itemID is invalid
	ErrInvalidDriveIDOrItemID = errorcode.New(errorcode.InvalidRequest, "invalid driveID or itemID")

	// ErrInvalidRequestBody is returned when the request body is invalid
	ErrInvalidRequestBody = errorcode.New(errorcode.InvalidRequest, "invalid request body")

	// ErrUnmountShare is returned when unmounting a share fails
	ErrUnmountShare = errorcode.New(errorcode.InvalidRequest, "unmounting share failed")

	// ErrMountShare is returned when mounting a share fails
	ErrMountShare = errorcode.New(errorcode.InvalidRequest, "mounting share failed")

	// ErrUpdateShares is returned when updating shares fails
	ErrUpdateShares = errorcode.New(errorcode.InvalidRequest, "failed to update share")

	// ErrInvalidID is returned when the id is invalid
	ErrInvalidID = errorcode.New(errorcode.InvalidRequest, "invalid id")

	// ErrDriveItemConversion is returned when converting to drive items fails
	ErrDriveItemConversion = errorcode.New(errorcode.InvalidRequest, "converting to drive items failed")

	// ErrNoShares is returned when no shares are found
	ErrNoShares = errorcode.New(errorcode.ItemNotFound, "no shares found")

	// ErrAlreadyMounted is returned when all shares are already mounted
	ErrAlreadyMounted = errorcode.New(errorcode.NameAlreadyExists, "shares already mounted")

	// ErrAlreadyUnmounted is returned when all shares are already unmounted
	ErrAlreadyUnmounted = errorcode.New(errorcode.NameAlreadyExists, "shares already unmounted")
)

type (
	// UpdateShareClosure is a closure that injects required updates into the update request
	UpdateShareClosure func(share *collaboration.ReceivedShare, request *collaboration.UpdateReceivedShareRequest)

	// DrivesDriveItemProvider is the interface that needs to be implemented by the individual space service
	DrivesDriveItemProvider interface {
		// MountShare mounts a share
		MountShare(ctx context.Context, resourceID *storageprovider.ResourceId, name string) ([]*collaboration.ReceivedShare, error)

		// MountOCMShare mounts an OCM share
		MountOCMShare(ctx context.Context, resourceID *storageprovider.ResourceId /*, name string*/) ([]*ocm.ReceivedShare, error)

		// UnmountShare unmounts a share
		UnmountShare(ctx context.Context, shareID *collaboration.ShareId) error

		// UpdateShares updates multiple shares
		UpdateShares(ctx context.Context, shares []*collaboration.ReceivedShare, updater UpdateShareClosure) ([]*collaboration.ReceivedShare, error)

		// GetShare returns the share
		GetShare(ctx context.Context, shareID *collaboration.ShareId) (*collaboration.ReceivedShare, error)

		// GetSharesForResource returns all shares for a given resourceID
		GetSharesForResource(ctx context.Context, resourceID *storageprovider.ResourceId, filters []*collaboration.Filter) ([]*collaboration.ReceivedShare, error)
	}
)

// DrivesDriveItemService contains the production business logic for everything that relates to drives
type DrivesDriveItemService struct {
	logger          log.Logger
	gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
}

// NewDrivesDriveItemService creates a new DrivesDriveItemService
func NewDrivesDriveItemService(logger log.Logger, gatewaySelector pool.Selectable[gateway.GatewayAPIClient]) (DrivesDriveItemService, error) {
	return DrivesDriveItemService{
		logger:          log.Logger{Logger: logger.With().Str("graph api", "DrivesDriveItemService").Logger()},
		gatewaySelector: gatewaySelector,
	}, nil
}

func (s DrivesDriveItemService) GetShare(ctx context.Context, shareID *collaboration.ShareId) (*collaboration.ReceivedShare, error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	// Now, find out the resourceID of the shared resource
	getReceivedShareResponse, err := gatewayClient.GetReceivedShare(ctx,
		&collaboration.GetReceivedShareRequest{
			Ref: &collaboration.ShareReference{
				Spec: &collaboration.ShareReference_Id{
					Id: shareID,
				},
			},
		},
	)

	return getReceivedShareResponse.GetShare(), errorcode.FromCS3Status(getReceivedShareResponse.GetStatus(), err)
}

// GetSharesForResource returns all shares for a given resourceID
func (s DrivesDriveItemService) GetSharesForResource(ctx context.Context, resourceID *storageprovider.ResourceId, filters []*collaboration.Filter) ([]*collaboration.ReceivedShare, error) {
	// Find all accepted shares for this resource
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	receivedSharesResponse, err := gatewayClient.ListReceivedShares(ctx, &collaboration.ListReceivedSharesRequest{
		Filters: append([]*collaboration.Filter{
			{
				Type: collaboration.Filter_TYPE_RESOURCE_ID,
				Term: &collaboration.Filter_ResourceId{
					ResourceId: resourceID,
				},
			},
		}, filters...),
	})
	switch {
	case err != nil:
		return nil, err
	case len(receivedSharesResponse.GetShares()) == 0:
		return nil, ErrNoShares
	default:
		return receivedSharesResponse.GetShares(), errorcode.FromCS3Status(receivedSharesResponse.GetStatus(), err)
	}
}

// UpdateShares updates multiple shares;
// it could happen that some shares are updated and some are not,
// this will return a list of updated shares and a list of errors;
// there is no guarantee that all updates are successful
func (s DrivesDriveItemService) UpdateShares(ctx context.Context, shares []*collaboration.ReceivedShare, updater UpdateShareClosure) ([]*collaboration.ReceivedShare, error) {
	errs := make([]error, 0, len(shares))
	updatedShares := make([]*collaboration.ReceivedShare, 0, len(shares))

	for _, share := range shares {
		updatedShare, err := s.UpdateShare(
			ctx,
			share,
			updater,
		)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		updatedShares = append(updatedShares, updatedShare)
	}

	return updatedShares, errors.Join(errs...)
}

// UpdateShare updates a single share
func (s DrivesDriveItemService) UpdateShare(ctx context.Context, share *collaboration.ReceivedShare, updater UpdateShareClosure) (*collaboration.ReceivedShare, error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	updateReceivedShareRequest := &collaboration.UpdateReceivedShareRequest{
		Share: &collaboration.ReceivedShare{
			Share: &collaboration.Share{
				Id: share.GetShare().GetId(),
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{}},
	}

	switch updater {
	case nil:
		return nil, ErrNoUpdater
	default:
		updater(share, updateReceivedShareRequest)
	}

	if len(updateReceivedShareRequest.GetUpdateMask().GetPaths()) == 0 {
		return nil, ErrNoUpdates
	}

	updateReceivedShareResponse, err := gatewayClient.UpdateReceivedShare(ctx, updateReceivedShareRequest)
	return updateReceivedShareResponse.GetShare(), errorcode.FromCS3Status(updateReceivedShareResponse.GetStatus(), err)
}

// UnmountShare unmounts a share
func (s DrivesDriveItemService) UnmountShare(ctx context.Context, shareID *collaboration.ShareId) error {
	share, err := s.GetShare(ctx, shareID)
	if err != nil {
		return err
	}

	shares, err := s.GetSharesForResource(ctx, share.GetShare().GetResourceId(), []*collaboration.Filter{
		{
			Type: collaboration.Filter_TYPE_STATE,
			Term: &collaboration.Filter_State{
				State: collaboration.ShareState_SHARE_STATE_ACCEPTED,
			},
		},
		{
			Type: collaboration.Filter_TYPE_STATE,
			Term: &collaboration.Filter_State{
				State: collaboration.ShareState_SHARE_STATE_REJECTED,
			},
		},
	})
	if err != nil {
		return err
	}
	availableShares := make([]*collaboration.ReceivedShare, 0, 1)
	rejectedShares := make([]*collaboration.ReceivedShare, 0, 1)
	for _, v := range shares {
		switch v.GetState() {
		case collaboration.ShareState_SHARE_STATE_ACCEPTED:
			availableShares = append(availableShares, v)
		case collaboration.ShareState_SHARE_STATE_REJECTED:
			rejectedShares = append(rejectedShares, v)
		}
	}
	if len(availableShares) == 0 {
		if len(rejectedShares) > 0 {
			return ErrAlreadyUnmounted
		}
		return ErrNoShares
	}

	_, err = s.UpdateShares(ctx, availableShares, func(_ *collaboration.ReceivedShare, request *collaboration.UpdateReceivedShareRequest) {
		request.Share.State = collaboration.ShareState_SHARE_STATE_REJECTED
		request.UpdateMask.Paths = append(request.UpdateMask.Paths, _fieldMaskPathState)
	})

	return err
}

// MountShare mounts a share, there is no guarantee that all siblings will be mounted
// in some rare cases it could happen that none of the siblings could be mounted,
// then the error will be returned
func (s DrivesDriveItemService) MountShare(ctx context.Context, resourceID *storageprovider.ResourceId, name string) ([]*collaboration.ReceivedShare, error) {
	if filepath.IsAbs(name) {
		return nil, ErrAbsoluteNamePath
	}

	if name != "" {
		name = filepath.Clean(name)
	}

	shares, err := s.GetSharesForResource(ctx, resourceID, nil)
	if err != nil {
		return nil, err
	}

	availableShares := make([]*collaboration.ReceivedShare, 0, len(shares))
	mountedShares := make([]*collaboration.ReceivedShare, 0, 1)
	for _, v := range shares {
		switch v.GetState() {
		case collaboration.ShareState_SHARE_STATE_ACCEPTED:
			mountedShares = append(mountedShares, v)
		case collaboration.ShareState_SHARE_STATE_PENDING, collaboration.ShareState_SHARE_STATE_REJECTED:
			availableShares = append(availableShares, v)
		}
	}
	if len(availableShares) == 0 {
		if len(mountedShares) > 0 {
			return nil, ErrAlreadyMounted
		}
		return nil, ErrNoShares
	}

	updatedShares, err := s.UpdateShares(ctx, availableShares, func(share *collaboration.ReceivedShare, request *collaboration.UpdateReceivedShareRequest) {
		request.Share.State = collaboration.ShareState_SHARE_STATE_ACCEPTED
		request.UpdateMask.Paths = append(request.UpdateMask.Paths, _fieldMaskPathState)

		// only update if mountPoint name is not empty and the path has changed
		if name != "" {
			mountPoint := share.GetMountPoint()
			if mountPoint == nil {
				mountPoint = &storageprovider.Reference{}
			}

			if filepath.Clean(mountPoint.GetPath()) != name {
				mountPoint.Path = name
				request.Share.MountPoint = mountPoint
				request.UpdateMask.Paths = append(request.UpdateMask.Paths, _fieldMaskPathMountPoint)
			}
		}
	})

	errs, ok := err.(interface{ Unwrap() []error })
	if ok && len(errs.Unwrap()) == len(availableShares) {
		// none of the received shares could be accepted.
		// this is an error, return it.
		return nil, err
	}

	return updatedShares, nil
}

// DrivesDriveItemApi is the api that registers the http endpoints which expose needed operation to the graph api.
// the business logic is delegated to the space service and further down to the cs3 client.
type DrivesDriveItemApi struct {
	logger                 log.Logger
	drivesDriveItemService DrivesDriveItemProvider
	baseGraphService       BaseGraphProvider
}

// NewDrivesDriveItemApi creates a new DrivesDriveItemApi
func NewDrivesDriveItemApi(drivesDriveItemService DrivesDriveItemProvider, baseGraphService BaseGraphProvider, logger log.Logger) (DrivesDriveItemApi, error) {
	return DrivesDriveItemApi{
		logger:                 log.Logger{Logger: logger.With().Str("graph api", "DrivesDriveItemApi").Logger()},
		drivesDriveItemService: drivesDriveItemService,
		baseGraphService:       baseGraphService,
	}, nil
}

// DeleteDriveItem deletes a drive item
func (api DrivesDriveItemApi) DeleteDriveItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	driveID, itemID, err := GetDriveAndItemIDParam(r, &api.logger)
	if err != nil {
		api.logger.Debug().Err(err).Msg(ErrInvalidDriveIDOrItemID.Error())
		ErrInvalidDriveIDOrItemID.Render(w, r)
		return
	}

	if !IsShareJail(driveID) {
		api.logger.Debug().Interface("driveID", driveID).Msg(ErrNotAShareJail.Error())
		ErrNotAShareJail.Render(w, r)
		return
	}

	shareID := ExtractShareIdFromResourceId(itemID)
	if err := api.drivesDriveItemService.UnmountShare(ctx, shareID); err != nil {
		api.logger.Debug().Err(err).Msg(ErrUnmountShare.Error())
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusNoContent)
	render.NoContent(w, r)
}

// GetDriveItem get a drive item
func (api DrivesDriveItemApi) GetDriveItem(w http.ResponseWriter, r *http.Request) {
	driveID, itemID, err := GetDriveAndItemIDParam(r, &api.logger)
	if err != nil {
		api.logger.Debug().Err(err).Msg(ErrInvalidDriveIDOrItemID.Error())
		ErrInvalidDriveIDOrItemID.Render(w, r)
		return
	}

	if !IsShareJail(driveID) {
		api.logger.Debug().Interface("driveID", driveID).Msg(ErrNotAShareJail.Error())
		ErrNotAShareJail.Render(w, r)
		return
	}

	shareID := ExtractShareIdFromResourceId(itemID)
	share, err := api.drivesDriveItemService.GetShare(r.Context(), shareID)
	if err != nil {
		api.logger.Debug().Err(err).Msg(ErrNoShares.Error())
		ErrNoShares.Render(w, r)
		return
	}

	availableShares, err := api.drivesDriveItemService.GetSharesForResource(r.Context(), share.GetShare().GetResourceId(), nil)
	if err != nil {
		api.logger.Debug().Err(err).Msg(ErrNoShares.Error())
		ErrNoShares.Render(w, r)
		return
	}

	driveItems, err := api.baseGraphService.CS3ReceivedSharesToDriveItems(r.Context(), availableShares)
	switch {
	case err != nil:
		break
	case len(driveItems) != 1:
		err = ErrDriveItemConversion
	}
	if err != nil {
		api.logger.Debug().Err(err).Msg(ErrDriveItemConversion.Error())
		ErrDriveItemConversion.Render(w, r)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, driveItems[0])
}

// UpdateDriveItem updates a drive item, currently only the visibility of the share is updated
func (api DrivesDriveItemApi) UpdateDriveItem(w http.ResponseWriter, r *http.Request) {
	driveID, itemID, err := GetDriveAndItemIDParam(r, &api.logger)
	if err != nil {
		api.logger.Debug().Err(err).Msg(ErrInvalidDriveIDOrItemID.Error())
		ErrInvalidDriveIDOrItemID.Render(w, r)
		return
	}

	if !IsShareJail(driveID) {
		api.logger.Debug().Interface("driveID", driveID).Msg(ErrNotAShareJail.Error())
		ErrNotAShareJail.Render(w, r)
		return
	}

	shareID := ExtractShareIdFromResourceId(itemID)
	requestDriveItem := libregraph.DriveItem{}
	if err := StrictJSONUnmarshal(r.Body, &requestDriveItem); err != nil {
		api.logger.Debug().Err(err).Msg(ErrInvalidRequestBody.Error())
		ErrInvalidRequestBody.Render(w, r)
		return
	}

	share, err := api.drivesDriveItemService.GetShare(r.Context(), shareID)
	if err != nil {
		api.logger.Debug().Err(err).Msg(ErrNoShares.Error())
		ErrNoShares.Render(w, r)
		return
	}

	availableShares, err := api.drivesDriveItemService.GetSharesForResource(r.Context(), share.GetShare().GetResourceId(), nil)
	if err != nil {
		api.logger.Debug().Err(err).Msg(ErrNoShares.Error())
		ErrNoShares.Render(w, r)
		return
	}

	updatedShares, err := api.drivesDriveItemService.UpdateShares(
		r.Context(),
		availableShares,
		func(_ *collaboration.ReceivedShare, request *collaboration.UpdateReceivedShareRequest) {
			request.GetShare().Hidden = requestDriveItem.GetUIHidden()
			request.UpdateMask.Paths = append(request.UpdateMask.Paths, _fieldMaskPathHidden)
		},
	)
	if err != nil {
		api.logger.Debug().Err(err).Msg(ErrUpdateShares.Error())
		ErrUpdateShares.Render(w, r)
		return
	}

	driveItems, err := api.baseGraphService.CS3ReceivedSharesToDriveItems(r.Context(), updatedShares)
	switch {
	case err != nil:
		break
	case len(driveItems) != 1:
		err = ErrDriveItemConversion
	}
	if err != nil {
		api.logger.Debug().Err(err).Msg(ErrDriveItemConversion.Error())
		ErrDriveItemConversion.Render(w, r)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, driveItems[0])
}

// CreateDriveItem creates a drive item
func (api DrivesDriveItemApi) CreateDriveItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		api.logger.Debug().Err(err).Msg(ErrInvalidDriveIDOrItemID.Error())
		ErrInvalidDriveIDOrItemID.Render(w, r)
		return
	}

	if !IsShareJail(&driveID) {
		api.logger.Debug().Interface("driveID", driveID).Msg(ErrNotAShareJail.Error())
		ErrNotAShareJail.Render(w, r)
		return
	}

	requestDriveItem := libregraph.DriveItem{}
	if err := StrictJSONUnmarshal(r.Body, &requestDriveItem); err != nil {
		api.logger.Debug().Err(err).Msg(ErrInvalidRequestBody.Error())
		ErrInvalidRequestBody.Render(w, r)
		return

	}

	remoteItem := requestDriveItem.GetRemoteItem()
	resourceId, err := storagespace.ParseID(remoteItem.GetId())
	if err != nil {
		api.logger.Debug().Err(err).Msg(ErrInvalidID.Error())
		ErrInvalidID.Render(w, r)
		return
	}

	var driveItems []libregraph.DriveItem
	switch {
	case resourceId.GetStorageId() == utils.OCMStorageProviderID:
		var mountedOcmShares []*ocm.ReceivedShare
		mountedOcmShares, err = api.drivesDriveItemService.MountOCMShare(ctx, &resourceId /*, requestDriveItem.GetName()*/)
		if err != nil {
			api.logger.Debug().Err(err).Msg(ErrMountShare.Error())

			switch e, ok := errorcode.ToError(err); {
			case ok && e.GetOrigin() == errorcode.ErrorOriginCS3 && e.GetCode() == errorcode.ItemNotFound:
				ErrDriveItemConversion.Render(w, r)
			default:
				errorcode.RenderError(w, r, err)
			}

			return
		}
		driveItems, err = api.baseGraphService.CS3ReceivedOCMSharesToDriveItems(ctx, mountedOcmShares)
	default:
		var mountedShares []*collaboration.ReceivedShare
		// Get all shares that the user has received for this resource. There might be multiple
		mountedShares, err = api.drivesDriveItemService.MountShare(ctx, &resourceId, requestDriveItem.GetName())
		if err != nil {
			api.logger.Debug().Err(err).Msg(ErrMountShare.Error())

			switch e, ok := errorcode.ToError(err); {
			case ok && e.GetOrigin() == errorcode.ErrorOriginCS3 && e.GetCode() == errorcode.ItemNotFound:
				ErrDriveItemConversion.Render(w, r)
			default:
				errorcode.RenderError(w, r, err)
			}

			return
		}
		driveItems, err = api.baseGraphService.CS3ReceivedSharesToDriveItems(ctx, mountedShares)
	}

	switch {
	case err != nil:
		break
	case len(driveItems) != 1:
		err = ErrDriveItemConversion
	}
	if err != nil {
		api.logger.Debug().Err(err).Msg(ErrDriveItemConversion.Error())
		ErrDriveItemConversion.Render(w, r)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, driveItems[0])
}
