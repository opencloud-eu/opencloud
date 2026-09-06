package svc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-chi/render"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/publicshare"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/tags"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/unifiedrole"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

// opt-in driveItem instance annotations, returned only when requested via $select
const (
	_selectAllowedValues = "@libre.graph.permissions.actions.allowedValues"
	_selectShareTypes    = "@libre.graph.shareTypes"
)

// without it the provider leaves the share-types opaque empty
var shareTypesFieldMask = &fieldmaskpb.FieldMask{Paths: []string{"share-types"}}

// opt-in driveItem relations, returned only when requested via $expand
const _expandChildren = "children"

func odataListContains(r *http.Request, parameter, value string) bool {
	for _, values := range r.URL.Query()[parameter] {
		for _, v := range strings.Split(values, ",") {
			if v == value {
				return true
			}
		}
	}
	return false
}

// driveItemInDrive reports whether an item id may be addressed below a drive.
// Items below the public share drive keep their real ids, so any id is
// acceptable there; the token scope enforces access, not this routing check.
func driveItemInDrive(driveID, driveItemID *storageprovider.ResourceId) bool {
	if driveID.GetStorageId() == utils.PublicStorageProviderID && driveID.GetSpaceId() == utils.PublicStorageSpaceID {
		return true
	}
	return driveID.GetStorageId() == driveItemID.GetStorageId() && driveID.GetSpaceId() == driveItemID.GetSpaceId()
}

// publicDriveRequest reports whether the request addresses the public share drive.
func publicDriveRequest(r *http.Request) bool {
	driveID, err := parseIDParam(r, "driveID")
	return err == nil &&
		driveID.GetStorageId() == utils.PublicStorageProviderID &&
		driveID.GetSpaceId() == utils.PublicStorageSpaceID
}

// sanitizePublicDriveInfos applies the publicstorageprovider's reduction to
// infos that bypassed it (navigation by id): paths relative to the share root,
// permissions cut to the link grant.
func (g Graph) sanitizePublicDriveInfos(ctx context.Context, r *http.Request, infos ...*storageprovider.ResourceInfo) error {
	shareRoot, grant, err := g.publicLinkOfRequest(ctx, r)
	if err != nil {
		return err
	}
	for _, info := range infos {
		if info != nil {
			publicshare.FilterResourceInfo(info, shareRoot, grant)
		}
	}
	return nil
}

// publicLinkOfRequest resolves the request's public link into the share root
// info and the granted permissions; the token is the public drive's opaque id.
func (g Graph) publicLinkOfRequest(ctx context.Context, r *http.Request) (*storageprovider.ResourceInfo, *storageprovider.ResourcePermissions, error) {
	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		return nil, nil, err
	}
	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		return nil, nil, err
	}
	shareResp, err := gatewayClient.GetPublicShare(ctx, &link.GetPublicShareRequest{
		Ref: &link.PublicShareReference{
			Spec: &link.PublicShareReference_Token{Token: driveID.GetOpaqueId()},
		},
	})
	if err := errorcode.FromCS3Status(shareResp.GetStatus(), err); err != nil {
		g.logger.Error().Err(err).Msg("could not resolve the public link of the request")
		return nil, nil, err
	}
	statResp, err := gatewayClient.Stat(ctx, &storageprovider.StatRequest{
		Ref: &storageprovider.Reference{ResourceId: shareResp.GetShare().GetResourceId()},
	})
	if err := errorcode.FromCS3Status(statResp.GetStatus(), err); err != nil {
		g.logger.Error().Err(err).Msg("could not stat the public link root")
		return nil, nil, err
	}
	return statResp.GetInfo(), shareResp.GetShare().GetPermissions().GetPermissions(), nil
}

// driveItemPropertySelected reports whether the given opt-in property was requested via $select
func driveItemPropertySelected(r *http.Request, property string) bool {
	return odataListContains(r, "$select", property)
}

// driveItemRelationExpanded reports whether the relation was requested via $expand
func driveItemRelationExpanded(r *http.Request, relation string) bool {
	return odataListContains(r, "$expand", relation)
}

// CreateUploadSession create an upload session to allow your app to upload files up to the maximum file size.
// An upload session allows your app to upload ranges of the file in sequential API requests, which allows the
// transfer to be resumed if a connection is dropped while the upload is in progress.
// ```json
//
//	{
//	  "@microsoft.graph.conflictBehavior": "fail (default) | replace | rename",
//	  "description": "description",
//	  "fileSize": 1234,
//	  "name": "filename.txt"
//	}
//
// ```
// From https://learn.microsoft.com/en-us/graph/api/driveitem-createuploadsession?view=graph-rest-1.0
func (g Graph) CreateUploadSession(w http.ResponseWriter, r *http.Request) {
	g.logger.Info().Msg("Calling CreateUploadSession")

	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}
	driveItemID, err := parseIDParam(r, "driveItemID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}
	// strict on purpose: uploading through the public drive addressing is not
	// part of the listing surface and stays untested for now
	if driveID.GetStorageId() != driveItemID.GetStorageId() || driveID.GetSpaceId() != driveItemID.GetSpaceId() {
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "Item does not exist")
		return
	}

	var cusr createUploadSessionRequest
	err = json.NewDecoder(r.Body).Decode(&cusr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("could not select next gateway client")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, "could not select next gateway client, aborting")
		return
	}

	ref := &storageprovider.Reference{
		ResourceId: &driveItemID,
	}
	if cusr.Item.Name != "" {
		ref.Path = utils.MakeRelativePath(cusr.Item.Name)
	}
	req := &storageprovider.InitiateFileUploadRequest{
		Ref:    ref,
		Opaque: utils.AppendPlainToOpaque(nil, "Upload-Length", strconv.FormatUint(uint64(cusr.Item.FileSize), 10)),
	}

	ctx := r.Context()
	res, err := gatewayClient.InitiateFileUpload(ctx, req)
	switch {
	case err != nil:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_OK:
		// ok
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_NOT_FOUND:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, res.GetStatus().GetMessage())
		return
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_PERMISSION_DENIED:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, res.GetStatus().GetMessage()) // do not leak existence? check what graph does
		return
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_UNAUTHENTICATED:
		errorcode.Unauthenticated.Render(w, r, http.StatusUnauthorized, res.GetStatus().GetMessage()) // do not leak existence? check what graph does
		return
	default:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, res.GetStatus().GetMessage())
		return
	}
	uploadSession := uploadSession{
		CS3Protocols: res.GetProtocols(),
	}
	for _, p := range res.GetProtocols() {
		if p.GetProtocol() == "simple" {
			uploadSession.UploadURL = p.GetUploadEndpoint() + "/" + p.GetToken()
		}
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, &uploadSession)
}

type createUploadSessionRequest struct {
	DeferCommit bool                          `json:"deferCommit"`
	Item        driveItemUploadableProperties `json:"item"`
}
type driveItemUploadableProperties struct {
	// ConflictBehavior "@microsoft.graph.conflictBehavior"
	//Description string
	FileSize int64 `json:"fileSize"`
	// fileSystemInfo
	Name string `json:"name"`
}
type uploadSession struct {
	UploadURL string
	//"expirationDateTime": "2015-01-29T09:21:55.523Z",
	//"nextExpectedRanges": ["0-"]
	CS3Protocols []*gateway.FileUploadProtocol
}

// GetRootDriveChildren implements the Service interface.
func (g Graph) GetRootDriveChildren(w http.ResponseWriter, r *http.Request) {
	g.logger.Info().Msg("Calling GetRootDriveChildren")
	ctx := r.Context()

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("could not select next gateway client")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, "could not select next gateway client, aborting")
		return
	}

	currentUser := revactx.ContextMustGetUser(r.Context())
	// do we need to list all or only the personal drive
	filters := []*storageprovider.ListStorageSpacesRequest_Filter{}
	filters = append(filters, listStorageSpacesUserFilter(currentUser.GetId().GetOpaqueId()))
	filters = append(filters, listStorageSpacesTypeFilter("personal"))

	res, err := gatewayClient.ListStorageSpaces(ctx, &storageprovider.ListStorageSpacesRequest{
		Filters: filters,
	})
	switch {
	case err != nil:
		g.logger.Error().Err(err).Msg("error making ListStorageSpaces grpc call")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	case res.GetStatus().GetCode() != cs3rpc.Code_CODE_OK:
		if res.GetStatus().GetCode() == cs3rpc.Code_CODE_NOT_FOUND {
			errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, res.GetStatus().GetMessage())
			return
		}
		g.logger.Error().Err(err).Msg("error sending ListStorageSpaces grpc request")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, res.GetStatus().GetMessage())
		return
	}

	var space *storageprovider.StorageSpace
	for _, s := range res.GetStorageSpaces() {
		if utils.UserIDEqual(currentUser.GetId(), s.GetOwner().GetId()) {
			space = s
		}
	}

	listRequest := &storageprovider.ListContainerRequest{
		Ref: &storageprovider.Reference{ResourceId: space.GetRoot()},
	}
	if driveItemPropertySelected(r, _selectShareTypes) {
		listRequest.FieldMask = shareTypesFieldMask
	}

	lRes, err := gatewayClient.ListContainer(ctx, listRequest)
	switch {
	case err != nil:
		g.logger.Error().Err(err).Msg("error making ListContainer grpc call")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	case lRes.GetStatus().GetCode() != cs3rpc.Code_CODE_OK:
		if lRes.GetStatus().GetCode() == cs3rpc.Code_CODE_NOT_FOUND {
			errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, lRes.GetStatus().GetMessage())
			return
		}
		if lRes.GetStatus().GetCode() == cs3rpc.Code_CODE_PERMISSION_DENIED {
			// TODO check if we should return 404 to not disclose existing items
			errorcode.AccessDenied.Render(w, r, http.StatusForbidden, lRes.GetStatus().GetMessage())
			return
		}
		g.logger.Error().Err(err).Msg("error sending list container grpc request")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, res.GetStatus().GetMessage())
		return
	}

	files, err := formatDriveItems(g.logger, g.publicBaseURL, lRes.GetInfos())
	if err != nil {
		g.logger.Error().Err(err).Msg("error encoding response as json")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	if driveItemPropertySelected(r, _selectAllowedValues) {
		for i, info := range lRes.GetInfos() {
			files[i].LibreGraphPermissionsActionsAllowedValues = unifiedrole.CS3ResourcePermissionsToLibregraphActions(info.GetPermissionSet())
		}
	}

	if driveItemPropertySelected(r, _selectShareTypes) {
		g.addShareTypes(ctx, files, lRes.GetInfos())
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: files})
}

// GetDriveItem returns a driveItem
func (g Graph) GetDriveItem(w http.ResponseWriter, r *http.Request) {
	g.logger.Info().Msg("Calling GetDriveItem")
	ctx := r.Context()

	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}
	driveItemID, err := parseIDParam(r, "driveItemID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}
	if !driveItemInDrive(&driveID, &driveItemID) {
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "Item does not exist")
		return
	}
	/*
		sanitizedPath := strings.TrimPrefix(r.URL.Path, "/graph/v1.0/")
		// Parse the request with odata parser
		odataReq, err := godata.ParseRequest(ctx, sanitizedPath, r.URL.Query())
		if err != nil {
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
			return
		}
	*/

	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	res, err := gatewayClient.Stat(ctx, &storageprovider.StatRequest{Ref: &storageprovider.Reference{ResourceId: &driveItemID}})
	switch {
	case err != nil:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_OK:
		if publicDriveRequest(r) {
			if err := g.sanitizePublicDriveInfos(ctx, r, res.GetInfo()); err != nil {
				errorcode.RenderError(w, r, err)
				return
			}
		}
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_NOT_FOUND:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, res.GetStatus().GetMessage())
		return
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_PERMISSION_DENIED:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, res.GetStatus().GetMessage()) // do not leak existence? check what graph does
		return
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_UNAUTHENTICATED:
		errorcode.Unauthenticated.Render(w, r, http.StatusUnauthorized, res.GetStatus().GetMessage()) // do not leak existence? check what graph does
		return
	default:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, res.GetStatus().GetMessage())
		return
	}
	driveItem, err := cs3ResourceToDriveItem(g.logger, g.publicBaseURL, res.GetInfo())
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	if driveItemPropertySelected(r, _selectAllowedValues) {
		driveItem.LibreGraphPermissionsActionsAllowedValues = unifiedrole.CS3ResourcePermissionsToLibregraphActions(res.GetInfo().GetPermissionSet())
	}

	// only containers have children
	if res.GetInfo().GetType() == storageprovider.ResourceType_RESOURCE_TYPE_CONTAINER && driveItemRelationExpanded(r, _expandChildren) {
		children, ok := g.listDriveItemChildren(w, r, &driveItemID)
		if !ok {
			return
		}
		driveItem.Children = children
	}

	if driveItemPropertySelected(r, _selectShareTypes) {
		infos := []*storageprovider.ResourceInfo{res.GetInfo()}
		driveItem.LibreGraphShareTypes = shareTypesOf(res.GetInfo(), g.listLinkShares(ctx, infos))
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &driveItem)
}

// GetDriveItemChildren lists the children of a driveItem
func (g Graph) GetDriveItemChildren(w http.ResponseWriter, r *http.Request) {
	g.logger.Info().Msg("Calling GetDriveItemChildren")

	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}
	driveItemID, err := parseIDParam(r, "driveItemID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}
	if !driveItemInDrive(&driveID, &driveItemID) {
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "Item does not exist")
		return
	}
	/*
		sanitizedPath := strings.TrimPrefix(r.URL.Path, "/graph/v1.0/")
		// Parse the request with odata parser
		odataReq, err := godata.ParseRequest(ctx, sanitizedPath, r.URL.Query())
		if err != nil {
			errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, err.Error())
			return
		}
	*/

	files, ok := g.listDriveItemChildren(w, r, &driveItemID)
	if !ok {
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: files})
}

func (g Graph) listDriveItemChildren(w http.ResponseWriter, r *http.Request, driveItemID *storageprovider.ResourceId) ([]libregraph.DriveItem, bool) {
	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return nil, false
	}

	childrenRequest := &storageprovider.ListContainerRequest{
		Ref: &storageprovider.Reference{ResourceId: driveItemID},
	}
	if driveItemPropertySelected(r, _selectShareTypes) {
		childrenRequest.FieldMask = shareTypesFieldMask
	}

	res, err := gatewayClient.ListContainer(r.Context(), childrenRequest)
	switch {
	case err != nil:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return nil, false
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_OK:
		// ok
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_NOT_FOUND:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, res.GetStatus().GetMessage())
		return nil, false
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_PERMISSION_DENIED:
		errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, res.GetStatus().GetMessage()) // do not leak existence? check what graph does
		return nil, false
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_UNAUTHENTICATED:
		errorcode.Unauthenticated.Render(w, r, http.StatusUnauthorized, res.GetStatus().GetMessage()) // do not leak existence? check what graph does
		return nil, false
	default:
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, res.GetStatus().GetMessage())
		return nil, false
	}

	if publicDriveRequest(r) {
		if err := g.sanitizePublicDriveInfos(r.Context(), r, res.GetInfos()...); err != nil {
			errorcode.RenderError(w, r, err)
			return nil, false
		}
	}

	files, err := formatDriveItems(g.logger, g.publicBaseURL, res.GetInfos())
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return nil, false
	}

	if driveItemPropertySelected(r, _selectShareTypes) {
		g.addShareTypes(r.Context(), files, res.GetInfos())
	}

	return files, true
}

func (g Graph) getRemoteItem(ctx context.Context, root *storageprovider.ResourceId, baseURL *url.URL) (*libregraph.RemoteItem, error) {
	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		return nil, err
	}

	ref := &storageprovider.Reference{
		ResourceId: root,
	}
	res, err := gatewayClient.Stat(ctx, &storageprovider.StatRequest{Ref: ref})
	if err != nil {
		return nil, err
	}
	if res.GetStatus().GetCode() != cs3rpc.Code_CODE_OK {
		// Only log this, there could be mountpoints which have no grant
		g.logger.Debug().Msg(res.GetStatus().GetMessage())
		return nil, errors.New("could not fetch grant resource for the mountpoint")
	}
	item, err := cs3ResourceToRemoteItem(res.GetInfo())
	if err != nil {
		return nil, err
	}

	if baseURL != nil && res.GetInfo() != nil && res.GetInfo().GetSpace() != nil {
		// TODO read from StorageSpace ... needs Opaque for now
		// TODO how do we build the url?
		// for now: read from request
		item.Name = libregraph.PtrString(res.GetInfo().GetName())
		if res.GetInfo().GetSpace().GetRoot() != nil {
			webDavURL := *baseURL
			relativePath := res.GetInfo().GetPath()
			webDavURL.Path = path.Join(webDavURL.Path, storagespace.FormatResourceID(res.GetInfo().GetSpace().GetRoot()), relativePath)
			item.WebDavUrl = libregraph.PtrString(webDavURL.String())
		}
	}
	return item, nil
}

func formatDriveItems(logger *log.Logger, publicBaseURL *url.URL, mds []*storageprovider.ResourceInfo) ([]libregraph.DriveItem, error) {
	responses := make([]libregraph.DriveItem, 0, len(mds))
	for i := range mds {
		res, err := cs3ResourceToDriveItem(logger, publicBaseURL, mds[i])
		if err != nil {
			return nil, err
		}
		responses = append(responses, *res)
	}

	return responses, nil
}

func cs3TimestampToTime(t *types.Timestamp) time.Time {
	return time.Unix(int64(t.GetSeconds()), int64(t.GetNanos()))
}

func cs3ResourceToDriveItem(logger *log.Logger, publicBaseURL *url.URL, res *storageprovider.ResourceInfo) (*libregraph.DriveItem, error) {
	size := new(int64)
	*size = int64(res.GetSize()) // TODO lurking overflow: make size of libregraph drive item use uint64

	driveItem := &libregraph.DriveItem{
		Id:   libregraph.PtrString(storagespace.FormatResourceID(res.GetId())),
		Size: size,
	}

	webURL := *publicBaseURL
	webURL.Path = path.Join(webURL.Path, "f", storagespace.FormatResourceID(res.GetId()))
	driveItem.WebUrl = libregraph.PtrString(webURL.String())

	if name := path.Base(res.GetPath()); name != "" {
		driveItem.Name = &name
	}
	if res.GetEtag() != "" {
		driveItem.ETag = &res.Etag
	}
	if res.GetMtime() != nil {
		lastModified := cs3TimestampToTime(res.GetMtime())
		driveItem.LastModifiedDateTime = &lastModified
	}
	if res.GetParentId() != nil {
		parentRef := libregraph.NewItemReference()
		parentRef.SetDriveType(res.GetSpace().GetSpaceType())
		parentRef.SetDriveId(storagespace.FormatStorageID(res.GetParentId().GetStorageId(), res.GetParentId().GetSpaceId()))
		parentRef.SetId(storagespace.FormatResourceID(res.GetParentId()))
		parentRef.SetName(path.Base(path.Dir(res.GetPath())))
		parentRef.SetPath(path.Dir(res.GetPath()))
		driveItem.ParentReference = parentRef
	}
	switch res.GetType() {
	case storageprovider.ResourceType_RESOURCE_TYPE_FILE:
		mimeType := res.GetMimeType()
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		// We cannot use a libregraph.File here because the openapi codegenerator autodetects 'File' as a go type ...
		driveItem.File = &libregraph.OpenGraphFile{
			MimeType: &mimeType,
		}
	case storageprovider.ResourceType_RESOURCE_TYPE_CONTAINER:
		if IsSpaceRoot(res.GetId()) {
			driveItem.SetRoot(map[string]any{})
		} else {
			driveItem.SetFolder(libregraph.Folder{})
		}
	}

	if metadata := res.GetArbitraryMetadata().GetMetadata(); metadata != nil {
		driveItem.Audio = metadataToFacet[libregraph.Audio](metadata, "audio")
		driveItem.Image = metadataToFacet[libregraph.Image](metadata, "image")
		driveItem.Location = metadataToFacet[libregraph.GeoCoordinates](metadata, "location")
		driveItem.Photo = metadataToFacet[libregraph.Photo](metadata, "photo")
		driveItem.Video = metadataToFacet[libregraph.Video](metadata, "video")
		driveItem.LibreGraphMotionPhoto = metadataToFacet[libregraph.MotionPhoto](metadata, "motionPhoto")
		driveItem.LibreGraphLivePhoto = metadataToFacet[libregraph.LivePhoto](metadata, "livePhoto")
		driveItem.LibreGraphMeFollowing = libregraph.PtrBool(metadata[_favoriteMetadataKey] == "1")
		if t := metadata["tags"]; t != "" {
			driveItem.LibreGraphTags = tags.New(t).AsSlice()
		}
	}

	driveItem.LockInfo = lockToFacet(res.GetLock())

	if utils.IsProcessing(res) {
		// queuedDateTime stays absent, we do not track when postprocessing started
		driveItem.PendingOperations = &libregraph.PendingOperations{
			PendingContentUpdate: &libregraph.PendingOperationsPendingContentUpdate{},
		}
	}

	return driveItem, nil
}

// lockToFacet maps a CS3 lock; WebDAV renders the same state as d:lockdiscovery.
// Only exclusive locks are reported, that is all OpenCloud issues.
func lockToFacet(lock *storageprovider.Lock) *libregraph.LockInfo {
	switch lock.GetType() {
	case storageprovider.LockType_LOCK_TYPE_EXCL, storageprovider.LockType_LOCK_TYPE_WRITE:
	default:
		return nil
	}

	info := libregraph.NewLockInfo()
	info.SetLockType("exclusive")

	if appName := lock.GetAppName(); appName != "" {
		info.SetLibreGraphAppName(appName)
	}

	if user := lock.GetUser(); user != nil {
		info.SetOwners([]libregraph.Identity{{
			Id:          libregraph.PtrString(user.GetOpaqueId()),
			DisplayName: utils.ReadPlainFromOpaque(lock.GetOpaque(), "lockownername"),
		}})
	}

	if lockTime, err := time.Parse(time.RFC3339, utils.ReadPlainFromOpaque(lock.GetOpaque(), "locktime")); err == nil {
		info.SetCreatedDateTime(lockTime.UTC())
	}

	if expiration := lock.GetExpiration(); expiration != nil {
		info.SetExpirationDateTime(time.Unix(int64(expiration.GetSeconds()), 0).UTC())
	}

	return info
}

// addShareTypes reads user and group shares off the grants, links from the share
// manager. Links are not stored as grants, so they need one filter per item,
// which is why the whole annotation is only built when it is selected.
func (g Graph) addShareTypes(ctx context.Context, items []libregraph.DriveItem, infos []*storageprovider.ResourceInfo) {
	linkShares := g.listLinkShares(ctx, infos)

	for i, info := range infos {
		if i >= len(items) {
			break
		}

		items[i].LibreGraphShareTypes = shareTypesOf(info, linkShares)
	}
}

// shareTypesOf maps the grants on a resource, plus a hit in the link lookup, to
// the annotation values.
func shareTypesOf(info *storageprovider.ResourceInfo, linkShares map[string]struct{}) []string {
	var types []string
	for _, grant := range strings.Split(utils.ReadPlainFromOpaque(info.GetOpaque(), "share-types"), ",") {
		switch grant {
		case strconv.Itoa(int(storageprovider.GranteeType_GRANTEE_TYPE_USER)):
			types = append(types, "user")
		case strconv.Itoa(int(storageprovider.GranteeType_GRANTEE_TYPE_GROUP)):
			types = append(types, "group")
		}
	}

	if _, ok := linkShares[info.GetId().GetOpaqueId()]; ok {
		types = append(types, "link")
	}

	return types
}

// listLinkShares returns the resource ids that carry a public link. A failed
// lookup is logged and treated as "no links", same as the WebDAV PROPFIND.
func (g Graph) listLinkShares(ctx context.Context, infos []*storageprovider.ResourceInfo) map[string]struct{} {
	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		g.logger.Error().Err(err).Msg("could not select gateway client for public shares")
		return nil
	}

	filters := make([]*link.ListPublicSharesRequest_Filter, 0, len(infos))
	for _, info := range infos {
		filters = append(filters, publicshare.ResourceIDFilter(info.GetId()))
	}

	res, err := gatewayClient.ListPublicShares(ctx, &link.ListPublicSharesRequest{Filters: filters})
	if err != nil || res.GetStatus().GetCode() != cs3rpc.Code_CODE_OK {
		g.logger.Error().Err(err).Msg("could not list public shares")
		return nil
	}

	linkShares := make(map[string]struct{}, len(res.GetShare()))
	for _, share := range res.GetShare() {
		linkShares[share.GetResourceId().GetOpaqueId()] = struct{}{}
	}

	return linkShares
}

// metadataToFacet builds a DriveItem facet *T from CS3 arbitrary metadata under
// the "libre.graph.<facet>." key prefix. Nil when no such keys are present.
func metadataToFacet[T any](metadata map[string]string, facet string) *T {
	return mapping.DeserializeStringsAt[T](metadata, "libre.graph."+facet+".")
}

func cs3ResourceToRemoteItem(res *storageprovider.ResourceInfo) (*libregraph.RemoteItem, error) {
	size := new(int64)
	*size = int64(res.GetSize()) // TODO lurking overflow: make size of libregraph drive item use uint64

	remoteItem := &libregraph.RemoteItem{
		Id:   libregraph.PtrString(storagespace.FormatResourceID(res.GetId())),
		Size: size,
	}

	if res.GetPath() != "" {
		remoteItem.Path = libregraph.PtrString(path.Clean(res.GetPath()))
	}
	if res.GetEtag() != "" {
		remoteItem.ETag = &res.Etag
	}
	if res.GetMtime() != nil {
		lastModified := cs3TimestampToTime(res.GetMtime())
		remoteItem.LastModifiedDateTime = &lastModified
	}
	if res.GetType() == storageprovider.ResourceType_RESOURCE_TYPE_FILE && res.GetMimeType() != "" {
		// We cannot use a libregraph.File here because the openapi codegenerator autodetects 'File' as a go type ...
		remoteItem.File = &libregraph.OpenGraphFile{
			MimeType: &res.MimeType,
		}
	}
	if res.GetType() == storageprovider.ResourceType_RESOURCE_TYPE_CONTAINER {
		remoteItem.Folder = &libregraph.Folder{}
	}
	if res.GetSpace() != nil && res.GetSpace().GetRoot() != nil {
		remoteItem.RootId = libregraph.PtrString(storagespace.FormatResourceID(res.GetSpace().GetRoot()))
		grantSpaceAlias := utils.ReadPlainFromOpaque(res.GetSpace().GetOpaque(), "spaceAlias")
		if grantSpaceAlias != "" {
			remoteItem.DriveAlias = libregraph.PtrString(grantSpaceAlias)
		}
	}
	return remoteItem, nil
}

func (g Graph) getPathForResource(ctx context.Context, id storageprovider.ResourceId) (string, error) {
	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		return "", err
	}

	res, err := gatewayClient.GetPath(ctx, &storageprovider.GetPathRequest{ResourceId: &id})
	if err != nil {
		return "", err
	}
	if res.GetStatus().GetCode() != cs3rpc.Code_CODE_OK {
		return "", fmt.Errorf("could not stat %v: %s", id, res.GetStatus().GetMessage())
	}
	return res.GetPath(), err
}

// getSpecialDriveItems reads properties from the opaque and transforms them into driveItems
func (g Graph) getSpecialDriveItems(ctx context.Context, baseURL *url.URL, space *storageprovider.StorageSpace) []libregraph.DriveItem {
	if space.GetRoot().GetStorageId() == utils.ShareStorageProviderID {
		return nil // no point in stating the ShareStorageProvider
	}
	if space.GetOpaque() == nil {
		return nil
	}

	imageNode := utils.ReadPlainFromOpaque(space.GetOpaque(), SpaceImageSpecialFolderName)
	readmeNode := utils.ReadPlainFromOpaque(space.GetOpaque(), ReadmeSpecialFolderName)

	cachekey := spaceRootStatKey(space.GetRoot(), imageNode, readmeNode)
	// if the root is older or equal to our cache we can reuse the cached extended spaces properties
	if entry := g.specialDriveItemsCache.Get(cachekey); entry != nil {
		if cached, ok := entry.Value().(specialDriveItemEntry); ok {
			if cached.rootMtime != nil && space.GetMtime() != nil {
				// beware, LaterTS does not handle equalness. it returns t1 if t1 > t2, else t2, so a >= check looks like this
				if utils.LaterTS(space.GetMtime(), cached.rootMtime) == cached.rootMtime {
					return cached.specialDriveItems
				}
			}
		}
	}

	var spaceItems []libregraph.DriveItem
	var err error
	doCache := true

	spaceItems, err = g.fetchSpecialDriveItem(ctx, spaceItems, SpaceImageSpecialFolderName, imageNode, space, baseURL)
	if err != nil {
		doCache = false
		g.logger.Debug().Err(err).Str("ID", imageNode).Msg("Could not get space image")
	}
	spaceItems, err = g.fetchSpecialDriveItem(ctx, spaceItems, ReadmeSpecialFolderName, readmeNode, space, baseURL)
	if err != nil {
		doCache = false
		g.logger.Debug().Err(err).Str("ID", imageNode).Msg("Could not get space readme")
	}

	// cache properties
	spacePropertiesEntry := specialDriveItemEntry{
		specialDriveItems: spaceItems,
		rootMtime:         space.GetMtime(),
	}

	if doCache {
		g.specialDriveItemsCache.Set(cachekey, spacePropertiesEntry, time.Duration(g.config.Spaces.ExtendedSpacePropertiesCacheTTL))
	}

	return spaceItems
}

func (g Graph) fetchSpecialDriveItem(ctx context.Context, spaceItems []libregraph.DriveItem, itemName string, itemNode string, space *storageprovider.StorageSpace, baseURL *url.URL) ([]libregraph.DriveItem, error) {
	var ref *storageprovider.Reference
	if itemNode != "" {
		rid, _ := storagespace.ParseID(itemNode)

		rid.StorageId = space.GetRoot().GetStorageId()
		ref = &storageprovider.Reference{
			ResourceId: &rid,
		}
		spaceItem, err := g.getSpecialDriveItem(ctx, ref, itemName, baseURL, space)
		if err != nil {
			return spaceItems, err
		}
		if spaceItem != nil {
			spaceItems = append(spaceItems, *spaceItem)
		}
	}
	return spaceItems, nil
}

// generates a space root stat cache key used to detect changes in a space
// takes into account the special nodes because changing metadata does not affect the etag / mtime
func spaceRootStatKey(id *storageprovider.ResourceId, imagenode, readmeNode string) string {
	if id == nil {
		return ""
	}
	shakeHash := sha3.NewShake256()
	_, _ = shakeHash.Write([]byte(id.GetStorageId()))
	_, _ = shakeHash.Write([]byte(id.GetSpaceId()))
	_, _ = shakeHash.Write([]byte(id.GetOpaqueId()))
	_, _ = shakeHash.Write([]byte(imagenode))
	_, _ = shakeHash.Write([]byte(readmeNode))
	h := make([]byte, 64)
	_, _ = shakeHash.Read(h)
	return hex.EncodeToString(h)
}

type specialDriveItemEntry struct {
	specialDriveItems []libregraph.DriveItem
	rootMtime         *types.Timestamp
}

func (g Graph) getSpecialDriveItem(ctx context.Context, ref *storageprovider.Reference, itemName string, baseURL *url.URL, space *storageprovider.StorageSpace) (*libregraph.DriveItem, error) {
	var spaceItem *libregraph.DriveItem
	if ref.GetResourceId().GetSpaceId() == "" && ref.GetResourceId().GetOpaqueId() == "" {
		return nil, nil
	}

	// FIXME we should send a fieldmask 'path' and return it as the Path property to save an additional call to the storage.
	// To do that we need to align the useg of ResourceInfo.Name vs ResourceInfo.Path. By default, only the name should be set
	// and Path should always be relative to the space root OR the resource the current user can access ...
	spaceItem, err := g.getDriveItem(ctx, ref)
	if err != nil {
		return nil, err
	}
	itemPath := ref.GetPath()
	if itemPath == "" {
		// lookup by id
		itemPath, err = g.getPathForResource(ctx, *ref.GetResourceId())
		if err != nil {
			return nil, err
		}
	}
	spaceItem.SpecialFolder = &libregraph.SpecialFolder{Name: libregraph.PtrString(itemName)}
	webdavURL := *baseURL
	webdavURL.Path = path.Join(webdavURL.Path, space.GetId().GetOpaqueId(), itemPath)
	spaceItem.WebDavUrl = libregraph.PtrString(webdavURL.String())

	return spaceItem, nil
}
