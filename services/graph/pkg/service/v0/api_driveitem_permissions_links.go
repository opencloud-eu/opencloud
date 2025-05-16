package svc

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/linktype"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

func (s DriveItemPermissionsService) CreateLink(ctx context.Context, driveItemID *storageprovider.ResourceId, createLink libregraph.DriveItemCreateLink) (libregraph.Permission, error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		s.logger.Error().Err(err).Msg("could not select next gateway client")
		return libregraph.Permission{}, errorcode.New(errorcode.GeneralException, err.Error())
	}

	statResp, err := gatewayClient.Stat(
		ctx,
		&storageprovider.StatRequest{
			Ref: &storageprovider.Reference{
				ResourceId: driveItemID,
				Path:       ".",
			},
		})
	if err != nil {
		s.logger.Error().Err(err).Msg("transport error, could not stat resource")
		return libregraph.Permission{}, errorcode.New(errorcode.GeneralException, err.Error())
	}
	if code := statResp.GetStatus().GetCode(); code != rpc.Code_CODE_OK {
		s.logger.Debug().Interface("itemID", driveItemID).Msg(statResp.GetStatus().GetMessage())
		return libregraph.Permission{}, errorcode.New(cs3StatusToErrCode(code), statResp.GetStatus().GetMessage())
	}
	permissions, err := linktype.CS3ResourcePermissionsFromSharingLink(createLink, statResp.GetInfo().GetType())
	if err != nil {
		s.logger.Debug().Interface("createLink", createLink).Msg(err.Error())
		return libregraph.Permission{}, errorcode.New(errorcode.InvalidRequest, "invalid link type")
	}
	if createLink.GetType() == libregraph.INTERNAL && len(createLink.GetPassword()) > 0 {
		return libregraph.Permission{}, errorcode.New(errorcode.InvalidRequest, "password is redundant for the internal link")
	}
	req := link.CreatePublicShareRequest{
		ResourceInfo: statResp.GetInfo(),
		Grant: &link.Grant{
			Permissions: &link.PublicSharePermissions{
				Permissions: permissions,
			},
			Password: createLink.GetPassword(),
		},
	}
	expirationDate, isSet := createLink.GetExpirationDateTimeOk()
	if isSet {
		expireTime := parseAndFillUpTime(expirationDate)
		if expireTime == nil {
			s.logger.Debug().Interface("createLink", createLink).Send()
			return libregraph.Permission{}, errorcode.New(errorcode.InvalidRequest, "invalid expiration date")
		}
		req.GetGrant().Expiration = expireTime
	}

	// set displayname and password protected as arbitrary metadata
	req.ResourceInfo.ArbitraryMetadata = &storageprovider.ArbitraryMetadata{
		Metadata: map[string]string{
			"name":      createLink.GetDisplayName(),
			"quicklink": strconv.FormatBool(createLink.GetLibreGraphQuickLink()),
		},
	}
	createResp, err := gatewayClient.CreatePublicShare(ctx, &req)
	if err != nil {
		s.logger.Error().Err(err).Msg("transport error, could not create link")
		return libregraph.Permission{}, errorcode.New(errorcode.GeneralException, err.Error())
	}
	if statusCode := createResp.GetStatus().GetCode(); statusCode != rpc.Code_CODE_OK {
		return libregraph.Permission{}, errorcode.New(cs3StatusToErrCode(statusCode), createResp.Status.Message)
	}
	link := createResp.GetShare()
	perm, err := s.libreGraphPermissionFromCS3PublicShare(link)
	if err != nil {
		return libregraph.Permission{}, errorcode.New(errorcode.GeneralException, err.Error())
	}
	return *perm, nil
}

func (s DriveItemPermissionsService) CreateSpaceRootLink(ctx context.Context, driveID *storageprovider.ResourceId, createLink libregraph.DriveItemCreateLink) (libregraph.Permission, error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return libregraph.Permission{}, err
	}
	space, err := utils.GetSpace(ctx, storagespace.FormatResourceID(driveID), gatewayClient)
	if err != nil {
		return libregraph.Permission{}, errorcode.FromUtilsStatusCodeError(err)
	}

	if space.SpaceType != _spaceTypeProject {
		return libregraph.Permission{}, errorcode.New(errorcode.InvalidRequest, "unsupported space type")
	}

	rootResourceID := space.GetRoot()
	return s.CreateLink(ctx, rootResourceID, createLink)
}

func (s DriveItemPermissionsService) SetPublicLinkPassword(ctx context.Context, driveItemId *storageprovider.ResourceId, permissionID string, password string) (libregraph.Permission, error) {
	publicShare, err := s.getCS3PublicShareByID(ctx, permissionID)
	if err != nil {
		return libregraph.Permission{}, err
	}

	// The resourceID of the shared resource need to match the item ID from the Request Path
	// otherwise this is an invalid Request.
	if !utils.ResourceIDEqual(publicShare.GetResourceId(), driveItemId) {
		s.logger.Debug().Msg("resourceID of shared does not match itemID")
		return libregraph.Permission{}, errorcode.New(errorcode.InvalidRequest, "permissionID and itemID do not match")
	}

	permission, err := s.updatePublicLinkPassword(ctx, permissionID, password)
	if err != nil {
		return libregraph.Permission{}, err
	}
	return *permission, nil
}

func (s DriveItemPermissionsService) SetPublicLinkPasswordOnSpaceRoot(ctx context.Context, driveID *storageprovider.ResourceId, permissionID string, password string) (libregraph.Permission, error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return libregraph.Permission{}, err
	}
	space, err := utils.GetSpace(ctx, storagespace.FormatResourceID(driveID), gatewayClient)
	if err != nil {
		return libregraph.Permission{}, errorcode.FromUtilsStatusCodeError(err)
	}

	if space.SpaceType != _spaceTypeProject {
		return libregraph.Permission{}, errorcode.New(errorcode.InvalidRequest, "unsupported space type")
	}
	rootResourceID := space.GetRoot()
	return s.SetPublicLinkPassword(ctx, rootResourceID, permissionID, password)
}

// CreateLink creates a public link on the cs3 api
func (api DriveItemPermissionsApi) CreateLink(w http.ResponseWriter, r *http.Request) {
	logger := api.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Msg("calling create link")

	_, driveItemID, err := GetDriveAndItemIDParam(r, &logger)
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	var createLink libregraph.DriveItemCreateLink
	if err = StrictJSONUnmarshal(r.Body, &createLink); err != nil {
		logger.Error().Err(err).Interface("body", r.Body).Msg("could not create link: invalid body schema definition")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid body schema definition")
		return
	}

	perm, err := api.driveItemPermissionsService.CreateLink(r.Context(), driveItemID, createLink)
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, perm)
}

func (api DriveItemPermissionsApi) CreateSpaceRootLink(w http.ResponseWriter, r *http.Request) {
	logger := api.logger.SubloggerWithRequestID(r.Context())
	logger.Info().Msg("calling create link")

	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		msg := "could not parse driveID"
		api.logger.Debug().Err(err).Msg(msg)
		errorcode.InvalidRequest.Render(w, r, http.StatusUnprocessableEntity, msg)
		return
	}

	var createLink libregraph.DriveItemCreateLink
	if err = StrictJSONUnmarshal(r.Body, &createLink); err != nil {
		logger.Error().Err(err).Interface("body", r.Body).Msg("could not create link: invalid body schema definition")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid body schema definition")
		return
	}

	perm, err := api.driveItemPermissionsService.CreateSpaceRootLink(r.Context(), &driveID, createLink)
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, perm)
}

// SetLinkPassword sets public link password on the cs3 api
func (api DriveItemPermissionsApi) SetLinkPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, itemID, err := GetDriveAndItemIDParam(r, &api.logger)
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	permissionID, err := url.PathUnescape(chi.URLParam(r, "permissionID"))
	if err != nil {
		api.logger.Debug().Err(err).Msg("could not parse permissionID")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid permissionID")
		return
	}

	password := &libregraph.SharingLinkPassword{}
	if err = StrictJSONUnmarshal(r.Body, password); err != nil {
		api.logger.Debug().Err(err).Interface("Body", r.Body).Msg("failed unmarshalling request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	newPermission, err := api.driveItemPermissionsService.SetPublicLinkPassword(ctx, itemID, permissionID, password.GetPassword())
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, newPermission)
}

func (api DriveItemPermissionsApi) SetSpaceRootLinkPassword(w http.ResponseWriter, r *http.Request) {
	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		msg := "could not parse driveID"
		api.logger.Debug().Err(err).Msg(msg)
		errorcode.InvalidRequest.Render(w, r, http.StatusUnprocessableEntity, msg)
		return
	}

	permissionID, err := url.PathUnescape(chi.URLParam(r, "permissionID"))
	if err != nil {
		api.logger.Debug().Err(err).Msg("could not parse permissionID")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid permissionID")
		return
	}

	password := &libregraph.SharingLinkPassword{}
	if err = StrictJSONUnmarshal(r.Body, password); err != nil {
		api.logger.Debug().Err(err).Interface("Body", r.Body).Msg("failed unmarshalling request body")
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := r.Context()
	newPermission, err := api.driveItemPermissionsService.SetPublicLinkPasswordOnSpaceRoot(ctx, &driveID, permissionID, password.GetPassword())
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, newPermission)
}

func (s DriveItemPermissionsService) updatePublicLinkPermission(ctx context.Context, permissionID string, itemID *storageprovider.ResourceId, newPermission *libregraph.Permission) (perm *libregraph.Permission, err error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		s.logger.Error().Err(err).Msg("could not select next gateway client")
		return nil, errorcode.New(errorcode.GeneralException, err.Error())
	}

	statResp, err := gatewayClient.Stat(
		ctx,
		&storageprovider.StatRequest{
			Ref: &storageprovider.Reference{
				ResourceId: itemID,
				Path:       ".",
			},
		})

	if err := errorcode.FromCS3Status(statResp.GetStatus(), err); err != nil {
		return nil, err
	}

	if newPermission.HasExpirationDateTime() {
		expirationDate := newPermission.GetExpirationDateTime()
		update := &link.UpdatePublicShareRequest_Update{
			Type:  link.UpdatePublicShareRequest_Update_TYPE_EXPIRATION,
			Grant: &link.Grant{Expiration: parseAndFillUpTime(&expirationDate)},
		}
		perm, err = s.updatePublicLink(ctx, permissionID, update)
		if err != nil {
			return nil, err
		}
	}

	if newPermission.HasLink() && newPermission.Link.HasLibreGraphDisplayName() {
		changedLink := newPermission.GetLink()
		update := &link.UpdatePublicShareRequest_Update{
			Type:        link.UpdatePublicShareRequest_Update_TYPE_DISPLAYNAME,
			DisplayName: changedLink.GetLibreGraphDisplayName(),
		}
		perm, err = s.updatePublicLink(ctx, permissionID, update)
		if err != nil {
			return nil, err
		}
	}

	if newPermission.HasLink() && newPermission.Link.HasType() {
		changedLink := newPermission.Link.GetType()
		permissions, err := linktype.CS3ResourcePermissionsFromSharingLink(
			libregraph.DriveItemCreateLink{
				Type: &changedLink,
			},
			statResp.GetInfo().GetType(),
		)
		if err != nil {
			return nil, err
		}
		update := &link.UpdatePublicShareRequest_Update{
			Type: link.UpdatePublicShareRequest_Update_TYPE_PERMISSIONS,
			Grant: &link.Grant{
				Permissions: &link.PublicSharePermissions{Permissions: permissions},
			},
		}
		perm, err = s.updatePublicLink(ctx, permissionID, update)
		if err != nil {
			return nil, err
		}
		// reset the password for the internal link
		if changedLink == libregraph.INTERNAL {
			perm, err = s.updatePublicLinkPassword(ctx, permissionID, "")
			if err != nil {
				return nil, err
			}
		}
	}

	return perm, err
}

func (s DriveItemPermissionsService) updatePublicLinkPassword(ctx context.Context, permissionID string, password string) (*libregraph.Permission, error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	changeLinkRes, err := gatewayClient.UpdatePublicShare(ctx, &link.UpdatePublicShareRequest{
		Update: &link.UpdatePublicShareRequest_Update{
			Type: link.UpdatePublicShareRequest_Update_TYPE_PASSWORD,
			Grant: &link.Grant{
				Password: password,
			},
		},
		Ref: &link.PublicShareReference{
			Spec: &link.PublicShareReference_Id{
				Id: &link.PublicShareId{
					OpaqueId: permissionID,
				},
			},
		},
	})
	if err := errorcode.FromCS3Status(changeLinkRes.GetStatus(), err); err != nil {
		return nil, err
	}
	permission, err := s.libreGraphPermissionFromCS3PublicShare(changeLinkRes.GetShare())
	if err != nil {
		return nil, err
	}
	return permission, nil
}

func (s DriveItemPermissionsService) updatePublicLink(ctx context.Context, permissionID string, update *link.UpdatePublicShareRequest_Update) (*libregraph.Permission, error) {
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	changeLinkRes, err := gatewayClient.UpdatePublicShare(ctx, &link.UpdatePublicShareRequest{
		Update: update,
		Ref: &link.PublicShareReference{
			Spec: &link.PublicShareReference_Id{
				Id: &link.PublicShareId{
					OpaqueId: permissionID,
				},
			},
		},
	})

	if err := errorcode.FromCS3Status(changeLinkRes.GetStatus(), err); err != nil {
		return nil, err
	}

	permission, err := s.libreGraphPermissionFromCS3PublicShare(changeLinkRes.GetShare())
	if err != nil {
		return nil, err
	}
	return permission, nil
}

func parseAndFillUpTime(t *time.Time) *types.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}

	// the link needs to be valid for the whole day
	tLink := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	tLink = tLink.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	final := tLink.UnixNano()

	return &types.Timestamp{
		Seconds: uint64(final / 1000000000),
		Nanos:   uint32(final % 1000000000),
	}
}
