package eventsmiddleware

import (
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/audit"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
)

// spaceEvent audits the calls on spaces. item has the name of the space
// before the call changed it.
func spaceEvent(req, res interface{}, item *storagespace.ItemSlot) (audit.Event, bool) {
	var ev audit.Event
	switch v := res.(type) {
	case *provider.CreateStorageSpaceResponse:
		r := req.(*provider.CreateStorageSpaceRequest)
		// the proxy calls CreateHome on every request, for an existing
		// personal space nothing was created
		if r.GetType() == "personal" && v.GetStatus().GetCode() == rpc.Code_CODE_ALREADY_EXISTS {
			return ev, false
		}
		ev = audit.Event{Definition: audit.SpaceCreate, Target: spaceTarget(v.GetStorageSpace().GetRoot())}
		ev.Target.Name = r.GetName()
		if ev.Target.ID == "" {
			ev.Target.ID = r.GetName()
		}
	case *provider.UpdateStorageSpaceResponse:
		r := req.(*provider.UpdateStorageSpaceRequest)
		ev.Target = spaceTarget(r.GetStorageSpace().GetRoot())
		if ev.Target.ID == "" {
			ev.Target.ID = r.GetStorageSpace().GetId().GetOpaqueId()
		}
		ev.Target.Name = item.Name
		if utils.ExistsInOpaque(r.GetOpaque(), "restore") {
			ev.Definition = audit.SpaceEnable
			break
		}
		// graph sends all changes of a drive PATCH in one request, a quota
		// change is only its own event when nothing else changed
		ev.Change = spaceChange(r.GetStorageSpace())
		ev.Definition = audit.SpaceUpdate
		if c := ev.Change; c != nil && len(c.After) == 1 {
			if _, quota := c.After["quota"]; quota {
				ev.Definition = audit.SpaceQuotaChange
			}
		}
	case *provider.DeleteStorageSpaceResponse:
		r := req.(*provider.DeleteStorageSpaceRequest)
		ev = audit.Event{Definition: audit.SpaceDisable, Target: audit.Target{ID: r.GetId().GetOpaqueId(), Name: item.Name}}
		if rid, err := storagespace.ParseID(r.GetId().GetOpaqueId()); err == nil {
			ev.Target.SpaceID = storagespace.FormatStorageID(rid.GetStorageId(), rid.GetSpaceId())
		}
		if utils.ExistsInOpaque(r.GetOpaque(), "purge") {
			ev.Definition = audit.SpaceDelete
		}
	default:
		return ev, false
	}
	return ev, true
}

func spaceTarget(rid *provider.ResourceId) audit.Target {
	if rid == nil {
		return audit.Target{}
	}
	id := storagespace.FormatStorageID(rid.GetStorageId(), rid.GetSpaceId())
	return audit.Target{ID: id, SpaceID: id}
}

// spaceAttributes are the space attributes graph puts into the opaque of an
// update, with their keys in the record. image and readme are the IDs of the
// files.
var spaceAttributes = []struct{ opaque, key string }{
	{"description", "description"},
	{"spaceAlias", "alias"},
	{"image", "image"},
	{"readme", "readme"},
}

// spaceChange lists what a space update changes. An attribute set to an
// empty value was removed.
func spaceChange(s *provider.StorageSpace) *audit.Change {
	after := map[string]any{}
	if n := s.GetName(); n != "" {
		after["name"] = n
	}
	if q := s.GetQuota(); q != nil {
		after["quota"] = q.GetQuotaMaxBytes()
	}
	for _, a := range spaceAttributes {
		if utils.ExistsInOpaque(s.GetOpaque(), a.opaque) {
			after[a.key] = utils.ReadPlainFromOpaque(s.GetOpaque(), a.opaque)
		}
	}
	if len(after) == 0 {
		return nil
	}
	return &audit.Change{After: after}
}
