package collaboration

import (
	"context"
	"fmt"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	permissionsapi "github.com/cs3org/go-cs3apis/cs3/permissions/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
)

type Permission string

const (
	PermissionCollaborationManageFonts Permission = "Collaboration.Fonts.Manage"
)

func CheckPermissions(gatewayClient gateway.GatewayAPIClient, ctx context.Context, permission Permission) (*userpb.User, bool, error) {
	user, ok := revactx.ContextGetUser(ctx)
	if !ok {
		return nil, false, fmt.Errorf("could not get user from context")
	}

	rsp, err := gatewayClient.CheckPermission(ctx, &permissionsapi.CheckPermissionRequest{
		Permission: string(permission),
		SubjectRef: &permissionsapi.SubjectReference{
			Spec: &permissionsapi.SubjectReference_UserId{
				UserId: user.GetId(),
			},
		},
	})
	if err != nil {
		return user, false, fmt.Errorf("could not check permissions: %w", err)
	}

	return user, rsp.GetStatus().GetCode() == rpc.Code_CODE_OK, nil
}
