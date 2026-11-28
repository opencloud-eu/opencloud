package eventsmiddleware

import (
	"path"
	"strings"

	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/audit"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
)

// fileEvent audits the calls on files and folders, the trash bin and versions.
func fileEvent(req, res interface{}, item *storagespace.ItemSlot) (audit.Event, bool) {
	itemType := item.Type
	var ev audit.Event
	switch res.(type) {
	case *provider.MoveResponse:
		r := req.(*provider.MoveRequest)
		src, dst := r.GetSource(), r.GetDestination()
		d := pick(itemType, audit.FileMove, audit.FolderMove)
		if sameParent(src, dst) {
			d = pick(itemType, audit.FileRename, audit.FolderRename)
		}
		if d == nil {
			return ev, false
		}
		ev = audit.Event{Definition: d, Target: refTarget(src, item)}
		before, after := movePaths(src, dst, item)
		if before != "" {
			ev.Change = &audit.Change{Before: map[string]any{"path": before}}
		}
		if after != "" {
			if ev.Change == nil {
				ev.Change = &audit.Change{}
			}
			ev.Change.After = map[string]any{"path": after}
		}
	case *provider.DeleteResponse:
		d := pick(itemType, audit.FileDelete, audit.FolderDelete)
		if d == nil {
			return ev, false
		}
		ev = audit.Event{Definition: d, Target: refTarget(req.(*provider.DeleteRequest).GetRef(), item)}
	case *provider.RestoreRecycleItemResponse:
		d := pick(itemType, audit.FileRestore, audit.FolderRestore)
		if d == nil {
			return ev, false
		}
		r := req.(*provider.RestoreRecycleItemRequest)
		ev = audit.Event{Definition: d, Target: trashTarget(r.GetRef(), r.GetKey(), item)}
	case *provider.CreateContainerResponse:
		r := req.(*provider.CreateContainerRequest)
		ev = audit.Event{Definition: audit.FolderCreate, Target: refTarget(r.GetRef(), item)}
	case *provider.SetLockResponse:
		ev = audit.Event{Definition: audit.FileLock, Target: refTarget(req.(*provider.SetLockRequest).GetRef(), item)}
	case *provider.UnlockResponse:
		ev = audit.Event{Definition: audit.FileUnlock, Target: refTarget(req.(*provider.UnlockRequest).GetRef(), item)}
	case *provider.RestoreFileVersionResponse:
		ev = audit.Event{Definition: audit.FileVersionRestore, Target: refTarget(req.(*provider.RestoreFileVersionRequest).GetRef(), item)}
	case *provider.PurgeRecycleResponse:
		r := req.(*provider.PurgeRecycleRequest)
		id := r.GetRef().GetResourceId()
		if id != nil && id.GetOpaqueId() == id.GetSpaceId() && r.GetKey() == "" {
			ev = audit.Event{Definition: audit.TrashPurgeAll, Target: spaceTarget(id)}
			break
		}
		d := pick(itemType, audit.FilePurge, audit.FolderPurge)
		if d == nil {
			return ev, false
		}
		ev = audit.Event{Definition: d, Target: trashTarget(r.GetRef(), r.GetKey(), item)}
	default:
		return ev, false
	}
	return ev, true
}

// pick returns the file or the folder definition, nil when the type is unknown.
func pick(t provider.ResourceType, file, folder *audit.Definition) *audit.Definition {
	switch t {
	case provider.ResourceType_RESOURCE_TYPE_CONTAINER:
		return folder
	case provider.ResourceType_RESOURCE_TYPE_FILE, provider.ResourceType_RESOURCE_TYPE_SYMLINK, provider.ResourceType_RESOURCE_TYPE_REFERENCE:
		return file
	default:
		return nil
	}
}

// sameParent reports whether a move only changes the name.
func sameParent(src, dst *provider.Reference) bool {
	if src.GetPath() == "" || dst.GetPath() == "" {
		return false
	}
	return storagespace.FormatResourceID(src.GetResourceId()) == storagespace.FormatResourceID(dst.GetResourceId()) &&
		path.Dir(src.GetPath()) == path.Dir(dst.GetPath())
}

// trashTarget identifies a trash bin item in the space of ref: by the ID the
// item had before it was deleted, when the driver reported it, else by its
// trash key.
func trashTarget(ref *provider.Reference, key string, item *storagespace.ItemSlot) audit.Target {
	t := audit.Target{ID: key, Name: item.Name, Path: itemPath(item)}
	if rid := ref.GetResourceId(); rid != nil {
		t.SpaceID = storagespace.FormatStorageID(rid.GetStorageId(), rid.GetSpaceId())
	}
	if id := itemID(ref, item); id != "" {
		t.ID = id
	}
	return t
}

// refTarget identifies the item of ref by its ID, when the driver reported
// it. A call that failed before the item was found only has the reference.
func refTarget(ref *provider.Reference, item *storagespace.ItemSlot) audit.Target {
	t := audit.Target{ID: itemID(ref, item), Path: refPath(ref, item), Name: item.Name}
	if t.ID == "" {
		t.ID, _ = storagespace.FormatReference(ref)
	}
	if rid := ref.GetResourceId(); rid != nil {
		t.SpaceID = storagespace.FormatStorageID(rid.GetStorageId(), rid.GetSpaceId())
	}
	if p := ref.GetPath(); p != "" && p != "." {
		t.Name = path.Base(p)
	}
	return t
}

// refPath is the path of the item in its space. The path of a request is
// only that when the request starts at the space root; a request through a
// share, a link or by ID gets the path from the driver.
func refPath(ref *provider.Reference, item *storagespace.ItemSlot) string {
	if p := itemPath(item); p != "" {
		return p
	}
	if utils.IsSpaceRootedReference(ref) {
		return ref.GetPath()
	}
	return ""
}

// movePaths returns the paths in the space before and after a move. The
// driver knows the path of the source; a destination relative to the same
// node as the source is put under the same prefix.
func movePaths(src, dst *provider.Reference, item *storagespace.ItemSlot) (before, after string) {
	before = refPath(src, item)
	switch {
	case utils.IsSpaceRootedReference(dst):
		after = dst.GetPath()
	case before != "" && utils.IsRelativeReference(dst) && utils.ResourceIDEqual(src.GetResourceId(), dst.GetResourceId()):
		rel := path.Clean(src.GetPath())
		if rel == "." {
			break
		}
		if prefix, ok := strings.CutSuffix(before, rel); ok {
			after = utils.MakeRelativePath(path.Join(prefix, dst.GetPath()))
		}
	}
	return before, after
}

// itemPath is the path in the space the driver reported, like the paths of
// requests: "./folder/file".
func itemPath(item *storagespace.ItemSlot) string {
	if item.Path == "" || item.Path == "/" {
		return ""
	}
	return utils.MakeRelativePath(item.Path)
}

// itemID formats the ID the driver reported, with the storage ID of ref,
// which the driver doesn't know.
func itemID(ref *provider.Reference, item *storagespace.ItemSlot) string {
	if item.ID.GetOpaqueId() == "" {
		return ""
	}
	return storagespace.FormatResourceID(&provider.ResourceId{
		StorageId: ref.GetResourceId().GetStorageId(),
		SpaceId:   item.ID.GetSpaceId(),
		OpaqueId:  item.ID.GetOpaqueId(),
	})
}
