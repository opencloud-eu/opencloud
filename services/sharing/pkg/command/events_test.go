package command

import (
	"errors"
	"testing"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/opencloud-eu/opencloud/pkg/conversions"
	"github.com/opencloud-eu/opencloud/pkg/log"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	settingsmocks "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0/mocks"
	"github.com/opencloud-eu/opencloud/services/sharing/pkg/config"
)

type gatewayTestSelector struct {
	client gateway.GatewayAPIClient
}

func (s gatewayTestSelector) Next(...pool.Option) (gateway.GatewayAPIClient, error) {
	return s.client, nil
}

func newGatewayTestSelector(client gateway.GatewayAPIClient) pool.Selectable[gateway.GatewayAPIClient] {
	return gatewayTestSelector{client: client}
}

func boolSettingResponse(value bool) *settingssvc.GetValueResponse {
	return &settingssvc.GetValueResponse{
		Value: &settingsmsg.ValueWithIdentifier{
			Value: &settingsmsg.Value{
				Value: &settingsmsg.Value_BoolValue{BoolValue: value},
			},
		},
	}
}

func newGatewayMock() *cs3mocks.GatewayAPIClient {
	gwc := &cs3mocks.GatewayAPIClient{}
	gwc.On("Authenticate", mock.Anything, mock.Anything, mock.Anything).
		Return(&gateway.AuthenticateResponse{
			Status: &rpc.Status{Code: rpc.Code_CODE_OK},
			Token:  "token",
		}, nil)
	gwc.On("UpdateReceivedShare", mock.Anything, mock.Anything, mock.Anything).
		Return(&collaboration.UpdateReceivedShareResponse{
			Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		}, nil)
	return gwc
}

func TestAutoAcceptShares(t *testing.T) {
	primaryUser := &user.UserId{OpaqueId: "user", Type: user.UserType_USER_TYPE_PRIMARY}
	guestUser := &user.UserId{OpaqueId: "guest@example.org", Type: user.UserType_USER_TYPE_GUEST}

	testCases := []struct {
		name              string
		granteeUser       *user.UserId
		granteeGroup      *group.GroupId
		groupMembers      []*user.UserId
		defaultAccept     bool
		settingValue      *bool
		expectAccepted    int
		expectSettingCall int
	}{
		{
			name:              "user setting enables auto accept",
			granteeUser:       primaryUser,
			defaultAccept:     false,
			settingValue:      conversions.ToPointer(true),
			expectAccepted:    1,
			expectSettingCall: 1,
		},
		{
			name:              "user setting disables auto accept",
			granteeUser:       primaryUser,
			defaultAccept:     true,
			settingValue:      conversions.ToPointer(false),
			expectAccepted:    0,
			expectSettingCall: 1,
		},
		{
			name:              "missing user setting falls back to default true",
			granteeUser:       primaryUser,
			defaultAccept:     true,
			expectAccepted:    1,
			expectSettingCall: 1,
		},
		{
			name:              "missing user setting falls back to default false",
			granteeUser:       primaryUser,
			defaultAccept:     false,
			expectAccepted:    0,
			expectSettingCall: 1,
		},
		{
			name:              "guest user grantee is skipped",
			granteeUser:       guestUser,
			defaultAccept:     true,
			settingValue:      conversions.ToPointer(true),
			expectAccepted:    0,
			expectSettingCall: 0,
		},
		{
			name:              "group grantee skips guest members",
			granteeGroup:      &group.GroupId{OpaqueId: "group"},
			groupMembers:      []*user.UserId{primaryUser, guestUser},
			defaultAccept:     true,
			settingValue:      conversions.ToPointer(true),
			expectAccepted:    1,
			expectSettingCall: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gwc := newGatewayMock()
			if tc.granteeGroup != nil {
				gwc.On("GetGroup", mock.Anything, mock.Anything, mock.Anything).
					Return(&group.GetGroupResponse{
						Status: &rpc.Status{Code: rpc.Code_CODE_OK},
						Group:  &group.Group{Members: tc.groupMembers},
					}, nil)
			}

			valueService := &settingsmocks.ValueService{}
			if tc.settingValue != nil {
				valueService.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything, mock.Anything).
					Return(boolSettingResponse(*tc.settingValue), nil)
			} else {
				valueService.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("setting not found"))
			}

			ev := events.ShareCreated{
				ShareID:        &collaboration.ShareId{OpaqueId: "share-id"},
				GranteeUserID:  tc.granteeUser,
				GranteeGroupID: tc.granteeGroup,
			}

			AutoAcceptShares(ev, tc.defaultAccept, log.NopLogger(), newGatewayTestSelector(gwc), valueService, config.ServiceAccount{
				ServiceAccountID:     "service-account",
				ServiceAccountSecret: "service-account-secret",
			}, 1)

			gwc.AssertNumberOfCalls(t, "UpdateReceivedShare", tc.expectAccepted)
			valueService.AssertNumberOfCalls(t, "GetValueByUniqueIdentifiers", tc.expectSettingCall)
			if tc.expectAccepted > 0 {
				var req *collaboration.UpdateReceivedShareRequest
				for _, call := range gwc.Calls {
					if call.Method == "UpdateReceivedShare" {
						req, _ = call.Arguments.Get(1).(*collaboration.UpdateReceivedShareRequest)
					}
				}
				assert.Equal(t, collaboration.ShareState_SHARE_STATE_ACCEPTED, req.GetShare().GetState())
			}
		})
	}
}
