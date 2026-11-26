package datatx

import (
	"context"
	"path"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	"github.com/opencloud-eu/reva/v3/pkg/appctx"
	"github.com/opencloud-eu/reva/v3/pkg/audit"
	ctxpkg "github.com/opencloud-eu/reva/v3/pkg/ctx"
	"github.com/opencloud-eu/reva/v3/pkg/storage"
	"github.com/opencloud-eu/reva/v3/pkg/storagespace"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
)

// UploadAudit describes a finished upload for the audit log.
type UploadAudit struct {
	// Header is the X-Audit-Context of the request that finished the upload.
	// Without it the upload was not sent by a client and is not reported.
	Header string
	// Initial is the request context of the request that created the upload,
	// for what the finishing request lacks, e.g. the auth method of a tus
	// upload whose data is sent with a transfer token.
	Initial audit.RequestContext
	// Executant is the user of the upload session, the actor when the
	// context has no user, as for tus uploads.
	Executant *userv1beta1.User
	// PublicShareID is the public link the upload was created through, when
	// the context doesn't tell.
	PublicShareID string
	// Ref is where the file was uploaded to, ID the node.
	Ref  *provider.Reference
	ID   *provider.ResourceId
	Name string
	// FS looks up the path of the node in its space. The path of the upload
	// session is cut off above what the uploader may see, e.g. in a share.
	FS storage.FS
}

// EmitUploadAudit reports a finished upload. Only uploads a client sent are
// reported: the proxy marks them with the audit header, writes by OpenCloud
// itself, e.g. for a copy, don't carry it. With audit off nothing is looked up.
func EmitUploadAudit(ctx context.Context, e audit.Emitter, u UploadAudit) {
	if !audit.Enabled(e) || u.Header == "" {
		return
	}
	ctx = audit.ContextSetRequest(ctx, audit.ParseRequestContext(u.Header).Merge(u.Initial))

	target := storagespace.FormatResourceID(u.ID)
	if u.ID.GetOpaqueId() == "" {
		target, _ = storagespace.FormatReference(u.Ref)
	}
	ev := audit.Event{
		Definition: audit.FileUpload,
		Target:     audit.Target{ID: target, Path: u.Ref.GetPath(), Name: u.Name},
	}
	if rid := u.Ref.GetResourceId(); rid != nil {
		ev.Target.SpaceID = storagespace.FormatStorageID(rid.GetStorageId(), rid.GetSpaceId())
	}
	if ev.Target.Name == "" && u.Ref.GetPath() != "" && u.Ref.GetPath() != "." {
		ev.Target.Name = path.Base(u.Ref.GetPath())
	}

	if _, ok := ctxpkg.ContextGetUser(ctx); !ok && u.Executant != nil {
		// the emitter takes name, type and tenant from the user
		ctx = ctxpkg.ContextSetUser(ctx, u.Executant)
	}
	if p := pathInSpace(ctx, u.FS, u.ID); p != "" {
		ev.Target.Path = p
	}
	shareID := audit.PublicShareID(ctx)
	if shareID == "" {
		shareID = u.PublicShareID
	}
	if shareID != "" {
		// through a public link the user is the owner of the link
		ev.Actor = &audit.Actor{Type: audit.ActorPublicLink, ID: shareID}
	}

	if err := e.Emit(ctx, ev); err != nil {
		appctx.GetLogger(ctx).Error().Err(err).Msg("audit event not emitted")
	}
}

// pathInSpace returns the path of the node in its space. A stat by ID has the
// whole path, a stat by path only what the user may see.
func pathInSpace(ctx context.Context, fs storage.FS, id *provider.ResourceId) string {
	if fs == nil || id.GetOpaqueId() == "" {
		return ""
	}
	info, err := fs.GetMD(ctx, &provider.Reference{ResourceId: id}, nil, []string{"path"})
	if err != nil || info.GetPath() == "" || info.GetPath() == "." {
		return ""
	}
	return utils.MakeRelativePath(info.GetPath())
}
