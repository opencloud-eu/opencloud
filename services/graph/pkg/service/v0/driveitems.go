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
	"reflect"
	"strconv"
	"strings"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"github.com/go-chi/render"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"golang.org/x/crypto/sha3"

	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

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

	lRes, err := gatewayClient.ListContainer(ctx, &storageprovider.ListContainerRequest{
		Ref: &storageprovider.Reference{ResourceId: space.GetRoot()},
	})
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

	files, err := formatDriveItems(g.logger, lRes.GetInfos())
	if err != nil {
		g.logger.Error().Err(err).Msg("error encoding response as json")
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
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
	if driveID.GetStorageId() != driveItemID.GetStorageId() || driveID.GetSpaceId() != driveItemID.GetSpaceId() {
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
	driveItem, err := cs3ResourceToDriveItem(g.logger, res.GetInfo())
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &driveItem)
}

// GetDriveItemChildren lists the children of a driveItem
func (g Graph) GetDriveItemChildren(w http.ResponseWriter, r *http.Request) {
	g.logger.Info().Msg("Calling GetDriveItemChildren")
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
	if driveID.GetStorageId() != driveItemID.GetStorageId() || driveID.GetSpaceId() != driveItemID.GetSpaceId() {
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

	res, err := gatewayClient.ListContainer(ctx, &storageprovider.ListContainerRequest{
		Ref: &storageprovider.Reference{ResourceId: &driveItemID},
	})
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

	files, err := formatDriveItems(g.logger, res.GetInfos())
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: files})
}

func spacePermissionIdToCS3Grantee(permissionID string) (storageprovider.Grantee, error) {
	// the permission ID for space permission is made of two parts
	// the grantee type ('u' or user, 'g' for group) and the user or group id
	var grantee storageprovider.Grantee
	parts := strings.SplitN(permissionID, ":", 2)
	if len(parts) != 2 {
		return grantee, errorcode.New(errorcode.InvalidRequest, "invalid space permission id")
	}
	switch parts[0] {
	case "u":
		grantee.Type = storageprovider.GranteeType_GRANTEE_TYPE_USER
		grantee.Id = &storageprovider.Grantee_UserId{
			UserId: &userpb.UserId{
				OpaqueId: parts[1],
			},
		}
	case "g":
		grantee.Type = storageprovider.GranteeType_GRANTEE_TYPE_GROUP
		grantee.Id = &storageprovider.Grantee_GroupId{
			GroupId: &grouppb.GroupId{
				OpaqueId: parts[1],
			},
		}
	default:
		return grantee, errorcode.New(errorcode.InvalidRequest, "invalid space permission id")
	}
	return grantee, nil
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

func formatDriveItems(logger *log.Logger, mds []*storageprovider.ResourceInfo) ([]*libregraph.DriveItem, error) {
	responses := make([]*libregraph.DriveItem, 0, len(mds))
	for i := range mds {
		res, err := cs3ResourceToDriveItem(logger, mds[i])
		if err != nil {
			return nil, err
		}
		responses = append(responses, res)
	}

	return responses, nil
}

func cs3TimestampToTime(t *types.Timestamp) time.Time {
	return time.Unix(int64(t.GetSeconds()), int64(t.GetNanos()))
}

func cs3ResourceToDriveItem(logger *log.Logger, res *storageprovider.ResourceInfo) (*libregraph.DriveItem, error) {
	size := new(int64)
	*size = int64(res.GetSize()) // TODO lurking overflow: make size of libregraph drive item use uint64

	driveItem := &libregraph.DriveItem{
		Id:   libregraph.PtrString(storagespace.FormatResourceID(res.GetId())),
		Size: size,
	}

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
	if res.GetType() == storageprovider.ResourceType_RESOURCE_TYPE_FILE && res.GetMimeType() != "" {
		// We cannot use a libregraph.File here because the openapi codegenerator autodetects 'File' as a go type ...
		driveItem.File = &libregraph.OpenGraphFile{
			MimeType: &res.MimeType,
		}
	}
	if res.GetType() == storageprovider.ResourceType_RESOURCE_TYPE_CONTAINER {
		driveItem.Folder = &libregraph.Folder{}
	}

	if res.GetArbitraryMetadata() != nil {
		driveItem.Audio = cs3ResourceToDriveItemAudioFacet(logger, res)
		driveItem.Image = cs3ResourceToDriveItemImageFacet(logger, res)
		driveItem.Location = cs3ResourceToDriveItemLocationFacet(logger, res)
		driveItem.Photo = cs3ResourceToDriveItemPhotoFacet(logger, res)
	}

	return driveItem, nil
}

func cs3ResourceToDriveItemAudioFacet(logger *log.Logger, res *storageprovider.ResourceInfo) *libregraph.Audio {
	if !strings.HasPrefix(res.GetMimeType(), "audio/") {
		return nil
	}

	k := res.GetArbitraryMetadata().GetMetadata()
	if k == nil {
		return nil
	}

	var audio = &libregraph.Audio{}
	if ok := unmarshalStringMap(logger, audio, k, "libre.graph.audio."); ok {
		return audio
	}

	return nil
}

func cs3ResourceToDriveItemImageFacet(logger *log.Logger, res *storageprovider.ResourceInfo) *libregraph.Image {
	k := res.GetArbitraryMetadata().GetMetadata()
	if k == nil {
		return nil
	}

	var image = &libregraph.Image{}
	if ok := unmarshalStringMap(logger, image, k, "libre.graph.image."); ok {
		return image
	}

	return nil
}

func cs3ResourceToDriveItemLocationFacet(logger *log.Logger, res *storageprovider.ResourceInfo) *libregraph.GeoCoordinates {
	k := res.GetArbitraryMetadata().GetMetadata()
	if k == nil {
		return nil
	}

	var location = &libregraph.GeoCoordinates{}
	if ok := unmarshalStringMap(logger, location, k, "libre.graph.location."); ok {
		return location
	}

	return nil
}

func cs3ResourceToDriveItemPhotoFacet(logger *log.Logger, res *storageprovider.ResourceInfo) *libregraph.Photo {
	k := res.GetArbitraryMetadata().GetMetadata()
	if k == nil {
		return nil
	}

	var photo = &libregraph.Photo{}
	if ok := unmarshalStringMap(logger, photo, k, "libre.graph.photo."); ok {
		return photo
	}

	return nil
}

func getFieldName(structField reflect.StructField) string {
	tag := structField.Tag.Get("json")
	if tag == "" {
		return structField.Name
	}

	return strings.Split(tag, ",")[0]
}

func unmarshalStringMap(logger *log.Logger, out any, flatMap map[string]string, prefix string) bool {
	nonEmpty := false
	obj := reflect.ValueOf(out).Elem()
	timeKind := reflect.TypeOf(&time.Time{}).Elem().Kind()
	for i := 0; i < obj.NumField(); i++ {
		field := obj.Field(i)
		structField := obj.Type().Field(i)
		mapKey := prefix + getFieldName(structField)

		if value, ok := flatMap[mapKey]; ok {
			if field.Kind() == reflect.Ptr {
				newValue := reflect.New(field.Type().Elem())
				var tmp any
				var err error
				switch t := newValue.Type().Elem().Kind(); t {
				case reflect.String:
					tmp = value
				case reflect.Int32:
					tmp, err = strconv.ParseInt(value, 10, 32)
				case reflect.Int64:
					tmp, err = strconv.ParseInt(value, 10, 64)
				case reflect.Float32:
					tmp, err = strconv.ParseFloat(value, 32)
				case reflect.Float64:
					tmp, err = strconv.ParseFloat(value, 64)
				case reflect.Bool:
					tmp, err = strconv.ParseBool(value)
				case timeKind:
					tmp, err = time.Parse(time.RFC3339, value)
				default:
					err = errors.New("unsupported type")
					logger.Error().Err(err).Str("type", t.String()).Str("mapKey", mapKey).Msg("target field type for value of mapKey is not supported")
				}
				if err != nil {
					logger.Error().Err(err).Str("mapKey", mapKey).Msg("unmarshalling failed")
					continue
				}
				newValue.Elem().Set(reflect.ValueOf(tmp).Convert(field.Type().Elem()))
				field.Set(newValue)
				nonEmpty = true
			}
		}
	}

	return nonEmpty
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
