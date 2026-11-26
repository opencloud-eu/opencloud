package eventsmiddleware

import (
	"context"
	"path"
	"slices"
	"time"

	groupv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	link "github.com/cs3org/go-cs3apis/cs3/sharing/link/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	typesv1beta1 "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/opencloud-eu/reva/v3/pkg/audit"
	"github.com/opencloud-eu/reva/v3/pkg/conversions"
	ctxpkg "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/opencloud-eu/reva/v3/pkg/storage/utils/grants"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
)

// The keys of share, space member and public link changes. A share or member
// change always names the recipient, a role change the new role.
const (
	keyRecipientType = "recipient_type"
	keyRecipient     = "recipient"
	keyRole          = "role"
	keyPermissions   = "permissions"
	keyExpiration    = "expiration"
	keyPassword      = "password"
)

// roleCustom is the role of permissions that match no known role.
const roleCustom = "custom"

// recipient adds the grantee to m.
func recipient(m map[string]any, g *provider.Grantee) {
	switch {
	case g.GetUserId() != nil:
		m[keyRecipientType], m[keyRecipient] = "user", g.GetUserId().GetOpaqueId()
	case g.GetGroupId() != nil:
		m[keyRecipientType], m[keyRecipient] = "group", g.GetGroupId().GetOpaqueId()
	}
}

// knownRole is a reva role a unified role is built from. The names are set
// here, some constructors share a name, e.g. the file editor is "editor".
type knownRole struct {
	name  string
	perms func() *conversions.Role
	space bool
}

// roles are matched in this order. viewer-list-grants and spaceviewer,
// editor-list-grants and spaceeditor-without-versions have the same
// permissions; on a space root the space roles win, everywhere else the others.
var roles = []knownRole{
	{conversions.RoleViewer, conversions.NewViewerRole, false},
	{conversions.RoleViewerWithVersions, conversions.NewViewerWithVersionsRole, false},
	{conversions.RoleViewerListGrants, conversions.NewViewerListGrantsRole, false},
	{conversions.RoleEditor, conversions.NewEditorRole, false},
	{conversions.RoleEditorWithVersions, conversions.NewEditorWithVersionsRole, false},
	{conversions.RoleEditorListGrants, conversions.NewEditorListGrantsRole, false},
	{conversions.RoleEditorListGrantsWithVersions, conversions.NewEditorListGrantsWithVersionsRole, false},
	{conversions.RoleFileEditor, conversions.NewFileEditorRole, false},
	{conversions.RoleFileEditorWithVersions, conversions.NewFileEditorWithVersionsRole, false},
	{conversions.RoleFileEditorListGrants, conversions.NewFileEditorListGrantsRole, false},
	{conversions.RoleFileEditorListGrantsWithVersions, conversions.NewFileEditorListGrantsWithVersionsRole, false},
	{conversions.RoleEditorLite, conversions.NewEditorLiteRole, false},
	{conversions.RoleUploader, conversions.NewUploaderRole, false},
	{conversions.RoleSecureViewer, conversions.NewSecureViewerRole, false},
	{conversions.RoleCoowner, conversions.NewCoownerRole, false},
	{conversions.RoleSpaceViewer, conversions.NewSpaceViewerRole, true},
	{conversions.RoleSpaceViewerWithVersions, conversions.NewSpaceViewerWithVersionsRole, true},
	{conversions.RoleSpaceEditor, conversions.NewSpaceEditorRole, true},
	{conversions.RoleSpaceEditorWithoutVersions, conversions.NewSpaceEditorWithoutVersionsRole, true},
	{conversions.RoleSpaceEditorWithoutTrashbin, conversions.NewSpaceEditorWithoutTrashbinRole, true},
	{conversions.RoleSpaceEditorWithoutVersionsWithoutTrashbin, conversions.NewSpaceEditorWithoutVersionsWithoutTrashbinRole, true},
	{conversions.RoleManager, conversions.NewManagerRole, true},
}

// role adds the role of the permissions to m: the reva role with exactly
// these permissions, the one a unified role is built from, or "custom" with
// the permission names. space tells a space member from a share.
func role(m map[string]any, p *provider.ResourcePermissions, isLink, space bool) {
	if p == nil {
		return
	}
	if grants.PermissionsEqual(p, &provider.ResourcePermissions{}) {
		// empty permissions deny access, on a link they make an internal link
		m[keyRole] = conversions.RoleDenied
		if isLink {
			m[keyRole] = "internal"
		}
		return
	}
	for _, preferred := range []bool{true, false} {
		for _, r := range roles {
			if (r.space == space) == preferred && grants.PermissionsEqual(p, r.perms().CS3ResourcePermissions()) {
				m[keyRole] = r.name
				return
			}
		}
	}
	m[keyRole] = roleCustom
	m[keyPermissions] = permissionNames(p)
}

// permissionNames lists the granted permissions, e.g. stat, list_container.
func permissionNames(p *provider.ResourcePermissions) []string {
	names := []string{}
	p.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.Kind() == protoreflect.BoolKind && v.Bool() {
			names = append(names, string(fd.Name()))
		}
		return true
	})
	return names
}

// expiration adds the expiration date to m, an empty string when it was removed.
func expiration(m map[string]any, ts *typesv1beta1.Timestamp) {
	if ts == nil {
		m[keyExpiration] = ""
		return
	}
	m[keyExpiration] = time.Unix(int64(ts.GetSeconds()), int64(ts.GetNanos())).UTC().Format(time.RFC3339)
}

// shareCreated describes a new share or space member. The response carries the
// share; a failed request only has the grant that was asked for.
func shareCreated(r *collaboration.CreateShareRequest, res *collaboration.CreateShareResponse) *audit.Change {
	g, p, exp := r.GetGrant().GetGrantee(), r.GetGrant().GetPermissions().GetPermissions(), r.GetGrant().GetExpiration()
	if s := res.GetShare(); s != nil {
		g, p, exp = s.GetGrantee(), s.GetPermissions().GetPermissions(), s.GetExpiration()
	}
	after := map[string]any{}
	recipient(after, g)
	role(after, p, false, utils.ExistsInOpaque(res.GetOpaque(), "spacegrant"))
	if exp != nil {
		expiration(after, exp)
	}
	return &audit.Change{After: after}
}

// shareUpdated describes what an update changed: the fields of the update
// mask, or of the deprecated field.
func shareUpdated(r *collaboration.UpdateShareRequest, res *collaboration.UpdateShareResponse) *audit.Change {
	s := res.GetShare()
	if s == nil {
		s = r.GetShare()
	}
	space := utils.ExistsInOpaque(res.GetOpaque(), "spacegrant")
	after := map[string]any{}
	recipient(after, s.GetGrantee())
	for _, p := range r.GetUpdateMask().GetPaths() {
		switch p {
		case "permissions":
			role(after, s.GetPermissions().GetPermissions(), false, space)
		case "expiration":
			expiration(after, s.GetExpiration())
		}
	}
	if p := r.GetField().GetPermissions(); p != nil {
		role(after, p.GetPermissions(), false, space)
	}
	return &audit.Change{After: after}
}

// shareRemoved describes the removed share from what the share provider put
// into the response.
func shareRemoved(o *typesv1beta1.Opaque) *audit.Change {
	before := map[string]any{}
	var u *userpb.UserId
	var g *groupv1beta1.GroupId
	switch {
	case utils.ReadJSONFromOpaque(o, "granteeuserid", &u) == nil && u != nil:
		recipient(before, &provider.Grantee{Type: provider.GranteeType_GRANTEE_TYPE_USER, Id: &provider.Grantee_UserId{UserId: u}})
	case utils.ReadJSONFromOpaque(o, "granteegroupid", &g) == nil && g != nil:
		recipient(before, &provider.Grantee{Type: provider.GranteeType_GRANTEE_TYPE_GROUP, Id: &provider.Grantee_GroupId{GroupId: g}})
	}
	var p *provider.ResourcePermissions
	if utils.ReadJSONFromOpaque(o, "permissions", &p) == nil {
		role(before, p, false, utils.ExistsInOpaque(o, "spacegrant"))
	}
	if len(before) == 0 {
		return nil
	}
	return &audit.Change{Before: before}
}

// linkCreated describes a new public link: role, expiration and whether it
// has a password, never the password itself.
func linkCreated(r *link.CreatePublicShareRequest, res *link.CreatePublicShareResponse) *audit.Change {
	after := map[string]any{}
	if s := res.GetShare(); s != nil {
		role(after, s.GetPermissions().GetPermissions(), true, false)
		after[keyPassword] = s.GetPasswordProtected()
		if s.GetExpiration() != nil {
			expiration(after, s.GetExpiration())
		}
		return &audit.Change{After: after}
	}
	role(after, r.GetGrant().GetPermissions().GetPermissions(), true, false)
	after[keyPassword] = r.GetGrant().GetPassword() != ""
	if r.GetGrant().GetExpiration() != nil {
		expiration(after, r.GetGrant().GetExpiration())
	}
	return &audit.Change{After: after}
}

// linkUpdated describes the one attribute a public link update changes.
func linkUpdated(r *link.UpdatePublicShareRequest, res *link.UpdatePublicShareResponse) *audit.Change {
	u := r.GetUpdate()
	s := res.GetShare()
	after := map[string]any{}
	switch u.GetType() {
	case link.UpdatePublicShareRequest_Update_TYPE_PERMISSIONS:
		p := u.GetGrant().GetPermissions().GetPermissions()
		if s != nil {
			p = s.GetPermissions().GetPermissions()
		}
		role(after, p, true, false)
	case link.UpdatePublicShareRequest_Update_TYPE_PASSWORD:
		// true: set or changed, false: removed
		after[keyPassword] = u.GetGrant().GetPassword() != ""
		if s != nil {
			after[keyPassword] = s.GetPasswordProtected()
		}
	case link.UpdatePublicShareRequest_Update_TYPE_EXPIRATION:
		expiration(after, u.GetGrant().GetExpiration())
	case link.UpdatePublicShareRequest_Update_TYPE_DISPLAYNAME:
		after["display_name"] = u.GetDisplayName()
	case link.UpdatePublicShareRequest_Update_TYPE_DESCRIPTION:
		after["description"] = u.GetDescription()
	case link.UpdatePublicShareRequest_Update_TYPE_NOTIFYUPLOADS:
		after["notify_uploads"] = u.GetNotifyUploads()
	case link.UpdatePublicShareRequest_Update_TYPE_NOTIFYUPLOADSEXTRARECIPIENTS:
		// email addresses, only whether there are any
		after["notify_uploads_extra_recipients"] = u.GetNotifyUploadsExtraRecipients() != ""
	default:
		return nil
	}
	return &audit.Change{After: after}
}

// linkRemoved describes the removed public link.
func linkRemoved(o *typesv1beta1.Opaque) *audit.Change {
	var p *provider.ResourcePermissions
	if utils.ReadJSONFromOpaque(o, "permissions", &p) != nil || p == nil {
		return nil
	}
	before := map[string]any{}
	role(before, p, true, false)
	return &audit.Change{Before: before}
}

// shareEvent audits the calls on shares and space members. The sharing
// service marks space grants in its responses, they are space member events.
func shareEvent(req, res interface{}, _ *storagespace.ItemSlot) (audit.Event, bool) {
	var ev audit.Event
	switch v := res.(type) {
	case *collaboration.CreateShareResponse:
		if utils.ExistsInOpaque(v.GetOpaque(), "noevent") {
			return ev, false
		}
		r := req.(*collaboration.CreateShareRequest)
		rid := r.GetResourceInfo().GetId()
		// a failed space grant therefore shows up as a failed share.create
		if utils.ExistsInOpaque(v.GetOpaque(), "spacegrant") {
			ev = audit.Event{Definition: audit.SpaceMemberAdd, Target: spaceTarget(rid)}
			ev.Target.Name = utils.ReadPlainFromOpaque(v.GetOpaque(), "resourcename")
		} else {
			ev = audit.Event{Definition: audit.ShareCreate, Target: shareTarget(v.GetShare().GetId().GetOpaqueId(), rid, r.GetResourceInfo().GetName())}
		}
		ev.Change = shareCreated(r, v)
	case *collaboration.UpdateShareResponse:
		r := req.(*collaboration.UpdateShareRequest)
		if utils.ExistsInOpaque(v.GetOpaque(), "spacegrant") {
			ev = audit.Event{Definition: audit.SpaceMemberUpdate, Target: spaceTarget(v.GetShare().GetResourceId())}
			ev.Target.Name = utils.ReadPlainFromOpaque(v.GetOpaque(), "resourcename")
		} else {
			ev = audit.Event{Definition: audit.ShareUpdate, Target: shareTarget(shareID(r.GetShare().GetId(), r.GetRef()), v.GetShare().GetResourceId(), utils.ReadPlainFromOpaque(v.GetOpaque(), "resourcename"))}
		}
		ev.Change = shareUpdated(r, v)
	case *collaboration.RemoveShareResponse:
		r := req.(*collaboration.RemoveShareRequest)
		var rid *provider.ResourceId
		_ = utils.ReadJSONFromOpaque(v.GetOpaque(), "resourceid", &rid)
		if utils.ExistsInOpaque(v.GetOpaque(), "spacegrant") {
			ev = audit.Event{Definition: audit.SpaceMemberRemove, Target: spaceTarget(rid)}
			ev.Target.Name = utils.ReadPlainFromOpaque(v.GetOpaque(), "resourcename")
		} else {
			ev = audit.Event{Definition: audit.ShareDelete, Target: shareTarget(shareID(nil, r.GetRef()), rid, utils.ReadPlainFromOpaque(v.GetOpaque(), "resourcename"))}
		}
		ev.Change = shareRemoved(v.GetOpaque())
	case *collaboration.UpdateReceivedShareResponse:
		// hiding a share or moving its mount point leaves the state as it is,
		// some clients send the state along, the provider adds the one before
		if r, ok := req.(*collaboration.UpdateReceivedShareRequest); !ok || !slices.Contains(r.GetUpdateMask().GetPaths(), "state") {
			return ev, false
		}
		rs := v.GetShare()
		if utils.ReadPlainFromOpaque(v.GetOpaque(), "previousstate") == rs.GetState().String() {
			return ev, false
		}
		switch rs.GetState() {
		case collaboration.ShareState_SHARE_STATE_ACCEPTED:
			ev = audit.Event{Definition: audit.ShareAccept}
		case collaboration.ShareState_SHARE_STATE_REJECTED:
			ev = audit.Event{Definition: audit.ShareDecline}
		default:
			return ev, false
		}
		ev.Target = shareTarget(rs.GetShare().GetId().GetOpaqueId(), rs.GetShare().GetResourceId(), path.Base(rs.GetMountPoint().GetPath()))
		if ev.Target.Name == "." || ev.Target.Name == "/" {
			ev.Target.Name = ""
		}
	default:
		return ev, false
	}
	return ev, true
}

// autoAccepted reports a share accepted by a service account, e.g. the
// sharing service accepting new shares automatically. Nobody decided anything,
// the share.create of the sharer is the record that counts.
func autoAccepted(ctx context.Context, res interface{}) bool {
	if _, ok := res.(*collaboration.UpdateReceivedShareResponse); !ok {
		return false
	}
	u, ok := ctxpkg.ContextGetUser(ctx)
	return ok && u.GetId().GetType() == userpb.UserType_USER_TYPE_SERVICE
}

func shareTarget(id string, rid *provider.ResourceId, name string) audit.Target {
	t := audit.Target{ID: id, Name: name}
	if rid != nil {
		t.SpaceID = storagespace.FormatStorageID(rid.GetStorageId(), rid.GetSpaceId())
	}
	return t
}

func shareID(id *collaboration.ShareId, ref *collaboration.ShareReference) string {
	if id != nil {
		return id.GetOpaqueId()
	}
	return ref.GetId().GetOpaqueId()
}
