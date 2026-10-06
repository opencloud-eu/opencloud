package svc

import (
	"path"
	"slices"
	"time"

	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"

	searchmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/search/v0"
	"github.com/opencloud-eu/opencloud/services/search/pkg/mapping"
)

func searchResourceID(id *searchmsg.ResourceID) string {
	return storagespace.FormatResourceID(&storageprovider.ResourceId{
		StorageId: id.GetStorageId(),
		SpaceId:   id.GetSpaceId(),
		OpaqueId:  id.GetOpaqueId(),
	})
}

func searchEntityToDriveItem(e *searchmsg.Entity, uid string) *libregraph.DriveItem {
	size := int64(e.GetSize())
	di := &libregraph.DriveItem{
		Id:   libregraph.PtrString(searchResourceID(e.GetId())),
		Name: libregraph.PtrString(e.GetName()),
		Size: &size,
	}
	if etag := e.GetEtag(); etag != "" {
		di.ETag = &etag
	}
	if mt := e.GetLastModifiedTime(); mt != nil {
		lm := time.Unix(mt.GetSeconds(), int64(mt.GetNanos())).UTC()
		di.LastModifiedDateTime = &lm
	}
	if e.GetType() == uint64(storageprovider.ResourceType_RESOURCE_TYPE_FILE) && e.GetMimeType() != "" {
		mt := e.GetMimeType()
		di.File = &libregraph.OpenGraphFile{MimeType: &mt}
	}
	if e.GetType() == uint64(storageprovider.ResourceType_RESOURCE_TYPE_CONTAINER) {
		di.Folder = &libregraph.Folder{}
	}
	if p := e.GetParentId(); p != nil {
		ref := libregraph.NewItemReference()
		ref.SetDriveId(storagespace.FormatStorageID(p.GetStorageId(), p.GetSpaceId()))
		ref.SetId(searchResourceID(p))
		if refPath := e.GetRef().GetPath(); refPath != "" {
			// the index keeps paths relative to the space root (./dir/file)
			parent := path.Dir(path.Join("/", refPath))
			ref.SetPath(parent)
			if parent != "/" {
				ref.SetName(path.Base(parent))
			}
		}
		di.ParentReference = ref
	}
	di.RemoteItem = searchEntityToRemoteItem(e)
	di.Audio = mapping.FromProto[libregraph.Audio](e.GetAudio())
	di.Image = mapping.FromProto[libregraph.Image](e.GetImage())
	di.Photo = mapping.FromProto[libregraph.Photo](e.GetPhoto())
	di.Location = mapping.FromProto[libregraph.GeoCoordinates](e.GetLocation())
	di.Video = mapping.FromProto[libregraph.Video](e.GetVideo())
	di.LibreGraphMotionPhoto = mapping.FromProto[libregraph.MotionPhoto](e.GetMotionPhoto())
	di.LibreGraphLivePhoto = mapping.FromProto[libregraph.LivePhoto](e.GetLivePhoto())
	if tags := e.GetTags(); len(tags) > 0 {
		di.LibreGraphTags = tags
	}
	if av := e.GetPermissionsActionsAllowedValues(); len(av) > 0 {
		di.LibreGraphPermissionsActionsAllowedValues = av
	}
	// the WebDAV report emits oc:favorite only when the caller favorited the item; mirror that
	if uid != "" && slices.Contains(e.GetFavorites(), uid) {
		di.LibreGraphMeFollowing = libregraph.PtrBool(true)
	}
	return di
}

// remoteItem carries the id in the owner's drive and the mountpoint it is
// reached through; absent for hits from the caller's own spaces
func searchEntityToRemoteItem(e *searchmsg.Entity) *libregraph.RemoteItem {
	id := e.GetRemoteItemId()
	if id == nil {
		return nil
	}

	item := libregraph.NewRemoteItem()
	item.SetId(searchResourceID(id))

	if root := e.GetShareRootName(); root != "" {
		item.SetPath(root)
		item.SetName(path.Base(root))
	}

	return item
}
