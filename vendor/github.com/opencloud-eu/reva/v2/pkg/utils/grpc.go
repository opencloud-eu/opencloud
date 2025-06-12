package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	invitev1beta1 "github.com/cs3org/go-cs3apis/cs3/ocm/invite/v1beta1"
	permissions "github.com/cs3org/go-cs3apis/cs3/permissions/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	storageprovider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	ctxpkg "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"google.golang.org/grpc/metadata"
)

// SpaceRole defines the user role on space
type SpaceRole func(*storageprovider.ResourcePermissions) bool

// Possible roles in spaces
var (
	AllRole     SpaceRole = func(perms *storageprovider.ResourcePermissions) bool { return true }
	ViewerRole  SpaceRole = func(perms *storageprovider.ResourcePermissions) bool { return perms.Stat }
	EditorRole  SpaceRole = func(perms *storageprovider.ResourcePermissions) bool { return perms.InitiateFileUpload }
	ManagerRole SpaceRole = func(perms *storageprovider.ResourcePermissions) bool { return perms.DenyGrant }
)

var _errStatusCodeTmpl = "unexpected status code while %s: %v"

// Package error checkers
var (
	IsErrNotFound         = func(err error) bool { return IsStatusCodeError(err, rpc.Code_CODE_NOT_FOUND) }
	IsErrPermissionDenied = func(err error) bool { return IsStatusCodeError(err, rpc.Code_CODE_PERMISSION_DENIED) }
)

// GetServiceUserContext returns an authenticated context of the given service user
// Deprecated: Use GetServiceUserContextWithContext()
func GetServiceUserContext(serviceUserID string, gwc gateway.GatewayAPIClient, serviceUserSecret string) (context.Context, error) {
	return GetServiceUserContextWithContext(context.Background(), gwc, serviceUserID, serviceUserSecret)
}

// GetServiceUserContextWithContext returns an authenticated context of the given service user
func GetServiceUserContextWithContext(ctx context.Context, gwc gateway.GatewayAPIClient, serviceUserID string, serviceUserSecret string) (context.Context, error) {
	token, err := GetServiceUserToken(ctx, gwc, serviceUserID, serviceUserSecret)
	if err != nil {
		return nil, err
	}

	return metadata.AppendToOutgoingContext(ctx, ctxpkg.TokenHeader, token), nil
}

// GetServiceUserToken returns a reva authentication token for the given service user
func GetServiceUserToken(ctx context.Context, gwc gateway.GatewayAPIClient, serviceUserID string, serviceUserSecret string) (string, error) {
	authRes, err := gwc.Authenticate(ctx, &gateway.AuthenticateRequest{
		Type:         "serviceaccounts",
		ClientId:     serviceUserID,
		ClientSecret: serviceUserSecret,
	})
	if err != nil {
		return "", err
	}

	if err := checkStatusCode("authenticating service user", authRes.GetStatus().GetMessage(), authRes.GetStatus().GetCode()); err != nil {
		return "", err
	}

	return authRes.Token, nil
}

// GetUser gets the specified user
func GetUser(ctx context.Context, userID *user.UserId, gwc gateway.GatewayAPIClient) (*user.User, error) {
	return getUser(ctx, userID, false, gwc)
}

// GetUserNoGroups gets the specified user without expanding groupmemberships
func GetUserNoGroups(ctx context.Context, userID *user.UserId, gwc gateway.GatewayAPIClient) (*user.User, error) {
	return getUser(ctx, userID, true, gwc)
}

// getUser gets the specified user
func getUser(ctx context.Context, userID *user.UserId, skipGroups bool, gwc gateway.GatewayAPIClient) (*user.User, error) {
	getUserResponse, err := gwc.GetUser(ctx, &user.GetUserRequest{
		UserId:                 userID,
		SkipFetchingUserGroups: skipGroups,
	})
	if err != nil {
		return nil, err
	}

	if err := checkStatusCode("getting user", getUserResponse.GetStatus().GetMessage(), getUserResponse.GetStatus().GetCode()); err != nil {
		return nil, err

	}
	return getUserResponse.GetUser(), nil
}

// GetUserWithContext gets the specified accepted user
func GetAcceptedUserWithContext(ctx context.Context, userID *user.UserId, gwc gateway.GatewayAPIClient) (*user.User, error) {
	getAcceptedUserResponse, err := gwc.GetAcceptedUser(ctx, &invitev1beta1.GetAcceptedUserRequest{RemoteUserId: userID})
	if err != nil {
		return nil, err
	}

	if err := checkStatusCode("getting accepted user", getAcceptedUserResponse.GetStatus().GetMessage(), getAcceptedUserResponse.GetStatus().GetCode()); err != nil {
		return nil, err
	}

	return getAcceptedUserResponse.GetRemoteUser(), nil
}

// GetSpace returns the given space
func GetSpace(ctx context.Context, spaceID string, gwc gateway.GatewayAPIClient) (*storageprovider.StorageSpace, error) {
	res, err := gwc.ListStorageSpaces(ctx, listStorageSpaceRequest(spaceID))
	if err != nil {
		return nil, err
	}

	if err := checkStatusCode("getting space", res.GetStatus().GetMessage(), res.GetStatus().GetCode()); err != nil {
		return nil, err
	}

	if len(res.StorageSpaces) == 0 {
		return nil, statusCodeError{"getting space", "", rpc.Code_CODE_NOT_FOUND}
	}

	return res.StorageSpaces[0], nil
}

// GetGroupMembers returns all members of the given group
func GetGroupMembers(ctx context.Context, groupID string, gwc gateway.GatewayAPIClient) ([]string, error) {
	r, err := gwc.GetGroup(ctx, &group.GetGroupRequest{GroupId: &group.GroupId{OpaqueId: groupID}})
	if err != nil {
		return nil, err
	}

	if err := checkStatusCode("getting group", r.GetStatus().GetMessage(), r.GetStatus().GetCode()); err != nil {
		return nil, err
	}

	users := make([]string, 0, len(r.GetGroup().GetMembers()))
	for _, u := range r.GetGroup().GetMembers() {
		users = append(users, u.GetOpaqueId())
	}

	return users, nil
}

// ResolveID returns either the given userID or all members of the given groupID (if userID is nil)
func ResolveID(ctx context.Context, userid *user.UserId, groupid *group.GroupId, gwc gateway.GatewayAPIClient) ([]string, error) {
	if userid != nil {
		return []string{userid.GetOpaqueId()}, nil
	}

	if ctx == nil {
		return nil, errors.New("need ctx to resolve group id")
	}

	return GetGroupMembers(ctx, groupid.GetOpaqueId(), gwc)
}

// GetSpaceMembers returns all members of the given space that have at least the given role. `nil` role will be interpreted as all
func GetSpaceMembers(ctx context.Context, spaceID string, gwc gateway.GatewayAPIClient, role SpaceRole) ([]string, error) {
	if ctx == nil {
		return nil, errors.New("need authenticated context to find space members")
	}

	space, err := GetSpace(ctx, spaceID, gwc)
	if err != nil {
		return nil, err
	}

	var users []string
	switch space.SpaceType {
	case "personal":
		users = append(users, space.GetOwner().GetId().GetOpaqueId())
	case "project":
		if users, err = gatherProjectSpaceMembers(ctx, space, gwc, role); err != nil {
			return nil, err
		}
	default:
		// TODO: shares? other space types?
		return nil, fmt.Errorf("unsupported space type: %s", space.SpaceType)
	}

	return users, nil
}

// GetResourceByID is a convenience method to get a resource by its resourceID
func GetResourceByID(ctx context.Context, resourceid *storageprovider.ResourceId, gwc gateway.GatewayAPIClient) (*storageprovider.ResourceInfo, error) {
	return GetResource(ctx, &storageprovider.Reference{ResourceId: resourceid}, gwc)
}

// GetResource returns a resource by reference
func GetResource(ctx context.Context, ref *storageprovider.Reference, gwc gateway.GatewayAPIClient) (*storageprovider.ResourceInfo, error) {
	res, err := gwc.Stat(ctx, &storageprovider.StatRequest{Ref: ref})
	if err != nil {
		return nil, err
	}

	if err := checkStatusCode("getting resource", res.GetStatus().GetMessage(), res.GetStatus().GetCode()); err != nil {
		return nil, err
	}

	return res.GetInfo(), nil
}

// CheckPermission checks if the user role contains the given permission
func CheckPermission(ctx context.Context, perm string, gwc gateway.GatewayAPIClient) (bool, error) {
	user := ctxpkg.ContextMustGetUser(ctx)
	resp, err := gwc.CheckPermission(ctx, &permissions.CheckPermissionRequest{
		SubjectRef: &permissions.SubjectReference{
			Spec: &permissions.SubjectReference_UserId{
				UserId: user.Id,
			},
		},
		Permission: perm,
	})
	return resp.GetStatus().GetCode() == rpc.Code_CODE_OK, err
}

// IsStatusCodeError returns true if `err` was caused because of status code `code`
func IsStatusCodeError(err error, code rpc.Code) bool {
	sce, ok := err.(statusCodeError)
	if !ok {
		return false
	}
	return sce.code == code
}

// StatusCodeErrorToCS3Status translate the `statusCodeError` type to CS3 Status
// returns nil if `err` does not match to the `statusCodeError` type
func StatusCodeErrorToCS3Status(err error) *rpc.Status {
	var sce statusCodeError
	ok := errors.As(err, &sce)
	if !ok {
		return nil
	}
	if sce.message == "" {
		sce.message = sce.reason
	}
	return &rpc.Status{Message: sce.message, Code: sce.code}
}

// IsSpaceRoot checks if the given resource info is referring to a space root
func IsSpaceRoot(ri *storageprovider.ResourceInfo) bool {
	f := ri.GetId()
	s := ri.GetSpace().GetRoot()
	return f.GetOpaqueId() == s.GetOpaqueId() && f.GetSpaceId() == s.GetSpaceId()
}

func checkStatusCode(reason, message string, code rpc.Code) error {
	if code == rpc.Code_CODE_OK {
		return nil
	}
	return statusCodeError{reason, message, code}
}

func gatherProjectSpaceMembers(ctx context.Context, space *storageprovider.StorageSpace, gwc gateway.GatewayAPIClient, role SpaceRole) ([]string, error) {
	var permissionsMap map[string]*storageprovider.ResourcePermissions
	if err := ReadJSONFromOpaque(space.GetOpaque(), "grants", &permissionsMap); err != nil {
		return nil, err
	}

	groupsMap := make(map[string]struct{})
	if opaqueGroups, ok := space.Opaque.Map["groups"]; ok {
		_ = json.Unmarshal(opaqueGroups.GetValue(), &groupsMap)
	}

	if role == nil {
		role = AllRole
	}

	// we use a map to avoid duplicates
	usermap := make(map[string]struct{})
	for id, perm := range permissionsMap {
		if !role(perm) {
			continue
		}

		if _, isGroup := groupsMap[id]; !isGroup {
			usermap[id] = struct{}{}
			continue
		}

		usrs, err := GetGroupMembers(ctx, id, gwc)
		if err != nil {
			// TODO: continue?
			return nil, err
		}

		for _, u := range usrs {
			usermap[u] = struct{}{}
		}
	}

	users := make([]string, 0, len(usermap))
	for id := range usermap {
		users = append(users, id)
	}

	return users, nil
}

func listStorageSpaceRequest(spaceID string) *storageprovider.ListStorageSpacesRequest {
	return &storageprovider.ListStorageSpacesRequest{
		Opaque: AppendPlainToOpaque(nil, "unrestricted", "true"),
		Filters: []*storageprovider.ListStorageSpacesRequest_Filter{
			{
				Type: storageprovider.ListStorageSpacesRequest_Filter_TYPE_ID,
				Term: &storageprovider.ListStorageSpacesRequest_Filter_Id{
					Id: &storageprovider.StorageSpaceId{
						OpaqueId: spaceID,
					},
				},
			},
		},
	}
}

// statusCodeError is a helper struct to return errors
type statusCodeError struct {
	reason  string
	message string // represents the v1beta11.Status.Message
	code    rpc.Code
}

// Error implements error interface
func (sce statusCodeError) Error() string {
	if sce.reason != "" {
		return fmt.Sprintf(_errStatusCodeTmpl, sce.reason, sce.code)
	}
	return fmt.Sprintf(_errStatusCodeTmpl, sce.message, sce.code)
}
