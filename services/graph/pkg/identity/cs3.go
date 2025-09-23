package identity

import (
	"context"
	"net/url"
	"time"

	"github.com/CiscoM31/godata"
	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	cs3group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	cs3user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	cs3rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	libregraph "github.com/opencloud-eu/libre-graph-api-go"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/odata"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
)

var (
	errNotImplemented = errorcode.New(errorcode.NotSupported, "not implemented")
)

type CS3 struct {
	Config          *shared.Reva
	Logger          *log.Logger
	GatewaySelector pool.Selectable[gateway.GatewayAPIClient]
}

// CreateUser implements the Backend Interface. It's currently not supported for the CS3 backend
func (i *CS3) CreateUser(ctx context.Context, user libregraph.User) (*libregraph.User, error) {
	return nil, errNotImplemented
}

// DeleteUser implements the Backend Interface. It's currently not supported for the CS3 backend
func (i *CS3) DeleteUser(ctx context.Context, nameOrID string) error {
	return errNotImplemented
}

// UpdateUser implements the Backend Interface. It's currently not supported for the CS3 backend
func (i *CS3) UpdateUser(ctx context.Context, nameOrID string, user libregraph.UserUpdate) (*libregraph.User, error) {
	return nil, errNotImplemented
}

// GetUser implements the Backend Interface.
func (i *CS3) GetUser(ctx context.Context, nameOrId string, _ *godata.GoDataRequest) (*libregraph.User, error) {
	logger := i.Logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "cs3").Msg("GetUser")
	gatewayClient, err := i.GatewaySelector.Next()
	if err != nil {
		logger.Error().Str("backend", "cs3").Err(err).Msg("could not get gatewayClient")
		return nil, errorcode.New(errorcode.ServiceNotAvailable, err.Error())
	}

	// Try to get the user by username first
	res, err := gatewayClient.GetUserByClaim(ctx, &cs3user.GetUserByClaimRequest{
		Claim: "username", // FIXME add consts to reva
		Value: nameOrId,
	})

	switch {
	case err != nil:
		logger.Error().Str("backend", "cs3").Err(err).Str("nameOrId", nameOrId).Msg("error sending get user by claim id grpc request: transport error")
		return nil, errorcode.New(errorcode.ServiceNotAvailable, err.Error())
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_OK:
		return CreateUserModelFromCS3(res.GetUser()), nil
	case res.GetStatus().GetCode() == cs3rpc.Code_CODE_NOT_FOUND:
		// If the user was not found by username, try to get it by user ID
	default:
		logger.Debug().Str("backend", "cs3").Err(err).Str("nameOrId", nameOrId).Msg("error sending get user by claim id grpc request")
		return nil, errorcode.New(errorcode.GeneralException, res.GetStatus().GetMessage())

	}

	// If the user was not found by username, try to get it by user ID
	res, err = gatewayClient.GetUserByClaim(ctx, &cs3user.GetUserByClaimRequest{
		Claim: "userid", // FIXME add consts to reva
		Value: nameOrId,
	})
	switch {
	case err != nil:
		logger.Error().Str("backend", "cs3").Err(err).Str("nameOrId", nameOrId).Msg("error sending get user by claim id grpc request: transport error")
		return nil, errorcode.New(errorcode.ServiceNotAvailable, err.Error())
	case res.GetStatus().GetCode() != cs3rpc.Code_CODE_OK:
		if res.GetStatus().GetCode() == cs3rpc.Code_CODE_NOT_FOUND {
			return nil, errorcode.New(errorcode.ItemNotFound, res.GetStatus().GetMessage())
		}
		logger.Debug().Str("backend", "cs3").Err(err).Str("nameOrId", nameOrId).Msg("error sending get user by claim id grpc request")
		return nil, errorcode.New(errorcode.GeneralException, res.GetStatus().GetMessage())
	}

	return CreateUserModelFromCS3(res.GetUser()), nil
}

// GetUsers implements the Backend Interface.
func (i *CS3) GetUsers(ctx context.Context, oreq *godata.GoDataRequest) ([]*libregraph.User, error) {
	logger := i.Logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "cs3").Msg("GetUsers")
	gatewayClient, err := i.GatewaySelector.Next()
	if err != nil {
		logger.Error().Str("backend", "cs3").Err(err).Msg("could not get gatewayClient")
		return nil, errorcode.New(errorcode.ServiceNotAvailable, err.Error())
	}

	search, err := odata.GetSearchValues(oreq.Query)
	if err != nil {
		return nil, err
	}

	res, err := gatewayClient.FindUsers(ctx, &cs3user.FindUsersRequest{
		// FIXME presence match is currently not implemented, an empty search currently leads to
		// Unwilling To Perform": Search Error: error parsing filter: (&(objectclass=posixAccount)(|(cn=*)(displayname=*)(mail=*))), error: Present filter match for cn not implemented
		Query: search,
	})
	switch {
	case err != nil:
		logger.Error().Str("backend", "cs3").Err(err).Str("search", search).Msg("error sending find users grpc request: transport error")
		return nil, errorcode.New(errorcode.ServiceNotAvailable, err.Error())
	case res.GetStatus().GetCode() != cs3rpc.Code_CODE_OK:
		if res.GetStatus().GetCode() == cs3rpc.Code_CODE_NOT_FOUND {
			return nil, errorcode.New(errorcode.ItemNotFound, res.GetStatus().GetMessage())
		}
		logger.Debug().Str("backend", "cs3").Err(err).Str("search", search).Msg("error sending find users grpc request")
		return nil, errorcode.New(errorcode.GeneralException, res.GetStatus().GetMessage())
	}

	users := make([]*libregraph.User, 0, len(res.GetUsers()))

	for _, user := range res.GetUsers() {
		users = append(users, CreateUserModelFromCS3(user))
	}

	return users, nil
}

// FilterUsers implements the Backend Interface. It's currently not supported for the CS3 backend
func (i *CS3) FilterUsers(_ context.Context, _ *godata.GoDataRequest, _ *godata.ParseNode) ([]*libregraph.User, error) {
	return nil, errNotImplemented
}

// UpdateLastSignInDate implements the Backend Interface. It's currently not supported for the CS3 backend
func (i *CS3) UpdateLastSignInDate(ctx context.Context, userID string, timestamp time.Time) error {
	return errNotImplemented
}

// GetGroups implements the Backend Interface.
func (i *CS3) GetGroups(ctx context.Context, oreq *godata.GoDataRequest) ([]*libregraph.Group, error) {
	logger := i.Logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "cs3").Msg("GetGroups")
	gatewayClient, err := i.GatewaySelector.Next()
	if err != nil {
		logger.Error().Str("backend", "cs3").Err(err).Msg("could not get gatewayClient")
		return nil, errorcode.New(errorcode.ServiceNotAvailable, err.Error())
	}

	search, err := odata.GetSearchValues(oreq.Query)
	if err != nil {
		return nil, err
	}

	res, err := gatewayClient.FindGroups(ctx, &cs3group.FindGroupsRequest{
		// FIXME presence match is currently not implemented, an empty search currently leads to
		// Unwilling To Perform": Search Error: error parsing filter: (&(objectclass=posixAccount)(|(cn=*)(displayname=*)(mail=*))), error: Present filter match for cn not implemented
		Filter: search,
	})

	switch {
	case err != nil:
		logger.Error().Str("backend", "cs3").Err(err).Str("search", search).Msg("error sending find groups grpc request: transport error")
		return nil, errorcode.New(errorcode.ServiceNotAvailable, err.Error())
	case res.Status.Code != cs3rpc.Code_CODE_OK:
		if res.Status.Code == cs3rpc.Code_CODE_NOT_FOUND {
			return nil, errorcode.New(errorcode.ItemNotFound, res.Status.Message)
		}
		logger.Debug().Str("backend", "cs3").Err(err).Str("search", search).Msg("error sending find groups grpc request")
		return nil, errorcode.New(errorcode.GeneralException, res.Status.Message)
	}

	groups := make([]*libregraph.Group, 0, len(res.GetGroups()))

	for _, group := range res.GetGroups() {
		groups = append(groups, CreateGroupModelFromCS3(group))
	}

	return groups, nil
}

// CreateGroup implements the Backend Interface. It's currently not supported for the CS3 backend
func (i *CS3) CreateGroup(ctx context.Context, group libregraph.Group) (*libregraph.Group, error) {
	return nil, errNotImplemented
}

// GetGroup implements the Backend Interface.
func (i *CS3) GetGroup(ctx context.Context, groupID string, queryParam url.Values) (*libregraph.Group, error) {
	logger := i.Logger.SubloggerWithRequestID(ctx)
	logger.Debug().Str("backend", "cs3").Msg("GetGroup")
	gatewayClient, err := i.GatewaySelector.Next()
	if err != nil {
		logger.Error().Str("backend", "cs3").Err(err).Msg("could not get gatewayClient")
		return nil, errorcode.New(errorcode.ServiceNotAvailable, err.Error())
	}

	res, err := gatewayClient.GetGroupByClaim(ctx, &cs3group.GetGroupByClaimRequest{
		Claim: "group_id", // FIXME add consts to reva
		Value: groupID,
	})

	switch {
	case err != nil:
		logger.Error().Str("backend", "cs3").Err(err).Str("groupid", groupID).Msg("error sending get group by claim id grpc request: transport error")
		return nil, errorcode.New(errorcode.ServiceNotAvailable, err.Error())
	case res.Status.Code != cs3rpc.Code_CODE_OK:
		if res.Status.Code == cs3rpc.Code_CODE_NOT_FOUND {
			return nil, errorcode.New(errorcode.ItemNotFound, res.Status.Message)
		}
		logger.Debug().Str("backend", "cs3").Err(err).Str("groupid", groupID).Msg("error sending get group by claim id grpc request")
		return nil, errorcode.New(errorcode.GeneralException, res.Status.Message)
	}

	return CreateGroupModelFromCS3(res.GetGroup()), nil
}

// DeleteGroup implements the Backend Interface. It's currently not supported for the CS3 backend
func (i *CS3) DeleteGroup(ctx context.Context, id string) error {
	return errNotImplemented
}

// UpdateGroupName implements the Backend Interface. It's currently not supported for the CS3 backend
func (i *CS3) UpdateGroupName(ctx context.Context, groupID string, groupName string) error {
	return errNotImplemented
}

// GetGroupMembers implements the Backend Interface. It's currently not supported for the CS3 backend
func (i *CS3) GetGroupMembers(ctx context.Context, groupID string, _ *godata.GoDataRequest) ([]*libregraph.User, error) {
	return nil, errNotImplemented
}

// AddMembersToGroup implements the Backend Interface. It's currently not supported for the CS3 backend
func (i *CS3) AddMembersToGroup(ctx context.Context, groupID string, memberID []string) error {
	return errNotImplemented
}

// RemoveMemberFromGroup implements the Backend Interface. It's currently not supported for the CS3 backend
func (i *CS3) RemoveMemberFromGroup(ctx context.Context, groupID string, memberID string) error {
	return errNotImplemented
}
