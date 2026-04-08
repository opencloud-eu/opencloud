package permissions

import (
	"context"

	userv1beta1 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	cs3permissions "github.com/cs3org/go-cs3apis/cs3/permissions/v1beta1"
	v1beta11 "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/storage/pkg/decomposedfs/node"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

var (
	tracer trace.Tracer
)

func init() {
	tracer = otel.Tracer("github.com/cs3org/reva/pkg/storage/utils/decomposedfs/permissions")
}

const (
	_spaceTypePersonal = "personal"
	_spaceTypeProject  = "project"
)

// PermissionsChecker defines an interface for checking permissions on a Node
type PermissionsChecker interface {
	AssemblePermissions(ctx context.Context, n *node.Node) (ap *provider.ResourcePermissions, err error)
	AssembleTrashPermissions(ctx context.Context, n *node.Node) (ap *provider.ResourcePermissions, err error)
}

// CS3PermissionsClient defines an interface for checking permissions against the CS3 permissions service
type CS3PermissionsClient interface {
	CheckPermission(ctx context.Context, in *cs3permissions.CheckPermissionRequest, opts ...grpc.CallOption) (*cs3permissions.CheckPermissionResponse, error)
}

// Permissions manages permissions
type Permissions struct {
	item                PermissionsChecker                                   // handles item permissions
	permissionsSelector pool.Selectable[cs3permissions.PermissionsAPIClient] // handlers space permissions
}

// NewPermissions returns a new Permissions instance
func NewPermissions(item PermissionsChecker, permissionsSelector pool.Selectable[cs3permissions.PermissionsAPIClient]) Permissions {
	return Permissions{item: item, permissionsSelector: permissionsSelector}
}

// AssemblePermissions is used to assemble file permissions
func (p Permissions) AssemblePermissions(ctx context.Context, n *node.Node) (*provider.ResourcePermissions, error) {
	ctx, span := tracer.Start(ctx, "AssemblePermissions")
	defer span.End()
	return p.item.AssemblePermissions(ctx, n)
}

// AssembleTrashPermissions is used to assemble file permissions
func (p Permissions) AssembleTrashPermissions(ctx context.Context, n *node.Node) (*provider.ResourcePermissions, error) {
	_, span := tracer.Start(ctx, "AssembleTrashPermissions")
	defer span.End()
	return p.item.AssembleTrashPermissions(ctx, n)
}

// CreateSpace returns true when the user is allowed to create the space
func (p Permissions) CreateSpace(ctx context.Context, spaceid string) bool {
	return p.checkPermission(ctx, "Drives.Create", spaceRef(spaceid))
}

// SetSpaceQuota returns true when the user is allowed to change the spaces quota
func (p Permissions) SetSpaceQuota(ctx context.Context, spaceid string, spaceType string) bool {
	switch spaceType {
	default:
		return false // only quotas of personal and project space may be changed
	case _spaceTypePersonal:
		return p.checkPermission(ctx, "Drives.ReadWritePersonalQuota", spaceRef(spaceid))
	case _spaceTypeProject:
		return p.checkPermission(ctx, "Drives.ReadWriteProjectQuota", spaceRef(spaceid))
	}
}

// ManageSpaceProperties returns true when the user is allowed to change space properties (name/subtitle)
func (p Permissions) ManageSpaceProperties(ctx context.Context, spaceid string) bool {
	return p.checkPermission(ctx, "Drives.ReadWrite", spaceRef(spaceid))
}

// SpaceAbility returns true when the user is allowed to enable/disable the space
func (p Permissions) SpaceAbility(ctx context.Context, spaceid string) bool {
	return p.checkPermission(ctx, "Drives.ReadWriteEnabled", spaceRef(spaceid))
}

// ListAllSpaces returns true when the user is allowed to list all spaces
func (p Permissions) ListAllSpaces(ctx context.Context) bool {
	return p.checkPermission(ctx, "Drives.List", nil)
}

// ListSpacesOfUser returns true when the user is allowed to list the spaces of the given user
func (p Permissions) ListSpacesOfUser(ctx context.Context, userid *userv1beta1.UserId) bool {
	switch {
	case userid == nil:
		// there is no filter
		return true // TODO: is `true` actually correct here? Shouldn't we check for ListAllSpaces too?
	case utils.UserIDEqual(ctxpkg.ContextMustGetUser(ctx).GetId(), userid):
		return true
	default:
		return p.ListAllSpaces(ctx)
	}
}

// DeleteAllSpaces returns true when the user is allowed to delete all spaces
func (p Permissions) DeleteAllSpaces(ctx context.Context) bool {
	return p.checkPermission(ctx, "Drives.DeleteProject", nil)
}

// DeleteAllHomeSpaces returns true when the user is allowed to delete all home spaces
func (p Permissions) DeleteAllHomeSpaces(ctx context.Context) bool {
	return p.checkPermission(ctx, "Drives.DeletePersonal", nil)
}

// checkPermission is used to check a users space permissions
func (p Permissions) checkPermission(ctx context.Context, perm string, ref *provider.Reference) bool {
	permissionsClient, err := p.permissionsSelector.Next()
	if err != nil {
		return false
	}

	user := ctxpkg.ContextMustGetUser(ctx)
	checkRes, err := permissionsClient.CheckPermission(ctx, &cs3permissions.CheckPermissionRequest{
		Permission: perm,
		SubjectRef: &cs3permissions.SubjectReference{
			Spec: &cs3permissions.SubjectReference_UserId{
				UserId: user.Id,
			},
		},
		Ref: ref,
	})
	if err != nil {
		return false
	}

	return checkRes.Status.Code == v1beta11.Code_CODE_OK
}

// IsManager returns true if the given resource permissions evaluate the user as "manager"
func IsManager(rp *provider.ResourcePermissions) bool {
	return rp.RemoveGrant
}

// IsEditor returns true if the given resource permissions evaluate the user as "editor"
func IsEditor(rp *provider.ResourcePermissions) bool {
	return rp.InitiateFileUpload
}

// IsViewer returns true if the given resource permissions evaluate the user as "viewer"
func IsViewer(rp *provider.ResourcePermissions) bool {
	return rp.Stat
}

func spaceRef(spaceid string) *provider.Reference {
	return &provider.Reference{
		ResourceId: &provider.ResourceId{
			StorageId: spaceid,
			// OpaqueId is the same, no need to transfer it
		},
	}
}
