package svc

import (
	"mime"
	"net/http"
	"path"
	"strings"

	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"

	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	graphm "github.com/opencloud-eu/opencloud/services/graph/pkg/middleware"
)

// RecycleBinSpecialFolderName addresses a drive's trash as a special folder
const RecycleBinSpecialFolderName = "recyclebin"

// GetDriveSpecial returns the driveItem of a special folder. In the colon path
// form (special/recyclebin:/{key}) it returns the trashed item with that key.
// $expand=children embeds the listing for folders.
func (g Graph) GetDriveSpecial(w http.ResponseWriter, r *http.Request) {
	g.logger.Debug().Msg("Calling GetDriveSpecial")

	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}
	if chi.URLParam(r, "specialName") != RecycleBinSpecialFolderName {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unknown special folder")
		return
	}

	key := specialFolderKey(r)
	driveItem, ok := g.getSpecialDriveItemForKey(w, r, &driveID, key)
	if !ok {
		return
	}

	if driveItem.Folder != nil && driveItemRelationExpanded(r, _expandChildren) {
		children, ok := g.listRecycleChildren(w, r, &driveID, key)
		if !ok {
			return
		}
		driveItem.Children = children
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, driveItem)
}

// getSpecialDriveItemForKey returns the trash root for an empty key, else the trashed item with that key
func (g Graph) getSpecialDriveItemForKey(w http.ResponseWriter, r *http.Request, driveID *storageprovider.ResourceId, key string) (*libregraph.DriveItem, bool) {
	if key == "" {
		return recycleBinDriveItem(driveID), true
	}
	item, ok := g.findRecycleItem(w, r, driveID, key)
	if !ok {
		return nil, false
	}
	return recycleItemToDriveItem(driveID, item), true
}

// findRecycleItem returns the trashed item with the given key. A top-level key lists itself,
// a nested key (key/rel/path) only shows up in its parent's listing.
func (g Graph) findRecycleItem(w http.ResponseWriter, r *http.Request, driveID *storageprovider.ResourceId, key string) (*storageprovider.RecycleItem, bool) {
	listKey := key
	if strings.Contains(key, "/") {
		listKey = path.Dir(key) + "/"
	}
	items, ok := g.listRecycle(w, r, driveID, listKey)
	if !ok {
		return nil, false
	}
	for _, item := range items {
		if item.GetKey() == key {
			return item, true
		}
	}
	errorcode.ItemNotFound.Render(w, r, http.StatusNotFound, "item not found")
	return nil, false
}

// ListDriveSpecialChildren lists the children of a special folder. For the
// recycle bin this is the trash listing; the colon path form
// (special/recyclebin:/{key}:/children) lists inside a trashed folder.
func (g Graph) ListDriveSpecialChildren(w http.ResponseWriter, r *http.Request) {
	g.logger.Debug().Msg("Calling ListDriveSpecialChildren")

	driveID, err := parseIDParam(r, "driveID")
	if err != nil {
		errorcode.RenderError(w, r, err)
		return
	}
	if chi.URLParam(r, "specialName") != RecycleBinSpecialFolderName {
		errorcode.InvalidRequest.Render(w, r, http.StatusBadRequest, "unknown special folder")
		return
	}

	files, ok := g.listRecycleChildren(w, r, &driveID, specialFolderKey(r))
	if !ok {
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &ListResponse{Value: files})
}

// listRecycleChildren lists the trash root for an empty key, else the children of the trashed folder with that key
func (g Graph) listRecycleChildren(w http.ResponseWriter, r *http.Request, driveID *storageprovider.ResourceId, key string) ([]libregraph.DriveItem, bool) {
	// the trailing slash asks reva for the children of the key instead of the key itself
	listKey := ""
	if key != "" {
		listKey = key + "/"
	}
	items, ok := g.listRecycle(w, r, driveID, listKey)
	if !ok {
		return nil, false
	}

	files := make([]libregraph.DriveItem, 0, len(items))
	for _, item := range items {
		files = append(files, *recycleItemToDriveItem(driveID, item))
	}
	return files, true
}

// specialFolderKey returns the recycle key addressed by the colon path form, "" for the trash root
func specialFolderKey(r *http.Request) string {
	p, _ := graphm.SpecialFolderPath(r.Context())
	return strings.Trim(p, "/")
}

func (g Graph) listRecycle(w http.ResponseWriter, r *http.Request, driveID *storageprovider.ResourceId, key string) ([]*storageprovider.RecycleItem, bool) {
	gatewayClient, err := g.gatewaySelector.Next()
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return nil, false
	}

	res, err := gatewayClient.ListRecycle(r.Context(), &storageprovider.ListRecycleRequest{
		Ref: &storageprovider.Reference{ResourceId: spaceRootID(driveID)},
		Key: key,
	})
	if err != nil {
		errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, err.Error())
		return nil, false
	}
	// like listDriveItemChildren, a denied listing does not disclose existence
	if !renderTrashStatus(w, r, res.GetStatus(), true) {
		return nil, false
	}
	return res.GetRecycleItems(), true
}

func spaceRootID(driveID *storageprovider.ResourceId) *storageprovider.ResourceId {
	return &storageprovider.ResourceId{
		StorageId: driveID.GetStorageId(),
		SpaceId:   driveID.GetSpaceId(),
		OpaqueId:  driveID.GetSpaceId(),
	}
}

// recycleBinDriveItem is the synthetic folder for the trash root. It has no node in the storage.
func recycleBinDriveItem(driveID *storageprovider.ResourceId) *libregraph.DriveItem {
	item := libregraph.NewDriveItem()
	item.SetId(storagespace.FormatResourceID(&storageprovider.ResourceId{
		StorageId: driveID.GetStorageId(),
		SpaceId:   driveID.GetSpaceId(),
		OpaqueId:  RecycleBinSpecialFolderName,
	}))
	item.SetName(RecycleBinSpecialFolderName)
	item.SetFolder(libregraph.Folder{})
	item.SetSpecialFolder(libregraph.SpecialFolder{Name: libregraph.PtrString(RecycleBinSpecialFolderName)})

	parentRef := libregraph.NewItemReference()
	parentRef.SetDriveId(storagespace.FormatStorageID(driveID.GetStorageId(), driveID.GetSpaceId()))
	parentRef.SetId(storagespace.FormatResourceID(spaceRootID(driveID)))
	item.SetParentReference(*parentRef)
	return item
}

// recycleItemToDriveItem maps a trashed item. The id carries the recycle key, which is the
// node id for top-level entries and key/relative/path inside a trashed folder.
func recycleItemToDriveItem(driveID *storageprovider.ResourceId, ri *storageprovider.RecycleItem) *libregraph.DriveItem {
	origin := ri.GetRef().GetPath()
	name := path.Base(origin)
	if name == "." || name == "/" {
		name = path.Base(ri.GetKey())
	}

	item := libregraph.NewDriveItem()
	item.SetId(storagespace.FormatResourceID(&storageprovider.ResourceId{
		StorageId: driveID.GetStorageId(),
		SpaceId:   driveID.GetSpaceId(),
		OpaqueId:  ri.GetKey(),
	}))
	item.SetName(name)
	item.SetSize(int64(ri.GetSize())) // TODO lurking overflow, see cs3ResourceToDriveItem

	switch ri.GetType() {
	case storageprovider.ResourceType_RESOURCE_TYPE_FILE:
		mimeType := mime.TypeByExtension(path.Ext(name))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		item.SetFile(libregraph.OpenGraphFile{MimeType: &mimeType})
	case storageprovider.ResourceType_RESOURCE_TYPE_CONTAINER:
		item.SetFolder(libregraph.Folder{})
	}

	trash := libregraph.NewTrash()
	if ri.GetDeletionTime() != nil {
		trash.SetTrashedDateTime(utils.TSToTime(ri.GetDeletionTime()).UTC())
	}
	item.SetTrash(*trash)

	if origin != "" {
		// the original location; the parent may itself be trashed or gone, so no id
		dir := path.Dir(origin)
		parentRef := libregraph.NewItemReference()
		parentRef.SetDriveId(storagespace.FormatStorageID(driveID.GetStorageId(), driveID.GetSpaceId()))
		parentRef.SetPath(dir)
		if base := path.Base(dir); base != "/" && base != "." {
			parentRef.SetName(base)
		}
		item.SetParentReference(*parentRef)
	}

	return item
}
