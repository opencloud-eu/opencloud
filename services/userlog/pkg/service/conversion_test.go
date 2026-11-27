package service_test

import (
	"context"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	invitepb "github.com/cs3org/go-cs3apis/cs3/ocm/invite/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/reva/v3/pkg/events"
	"github.com/opencloud-eu/reva/v3/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v3/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v3/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	"github.com/opencloud-eu/opencloud/services/userlog/pkg/service"
)

var _ = Describe("Converter", func() {
	var (
		gatewayClient *cs3mocks.GatewayAPIClient
		converter     *service.Converter

		// what ocmd makes of the sender "<uuid>@<issuer>@<provider domain>" of an incoming share
		sharer = &user.UserId{
			Type:     user.UserType_USER_TYPE_FEDERATED,
			OpaqueId: "4c510ada-c86b-4815-8820-42cdf82c3d51@https://remote.example.org",
			Idp:      "remote.example.org",
		}
		grantee = &user.UserId{OpaqueId: "f7fbf8c8-139b-4376-b307-cf0a8c2d0d9c", Idp: "https://local.example.org"}
		ctime   = &types.Timestamp{Seconds: 1700000000}
	)

	BeforeEach(func() {
		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		selector := pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)
		converter = service.NewConverter(context.Background(), "en", selector, "userlog", "", "en")
	})

	Describe("ocm shares", func() {
		It("converts a received share and resolves the sharer for the grantee", func() {
			gatewayClient.On("GetAcceptedUser", mock.Anything, mock.MatchedBy(func(req *invitepb.GetAcceptedUserRequest) bool {
				var filter user.UserId
				if err := utils.UnmarshalJSONToProtoV1(req.GetOpaque().GetMap()["user-filter"].GetValue(), &filter); err != nil {
					return false
				}
				return req.GetRemoteUserId().GetOpaqueId() == sharer.GetOpaqueId() && filter.GetOpaqueId() == grantee.GetOpaqueId()
			})).Return(&invitepb.GetAcceptedUserResponse{
				Status:     &rpc.Status{Code: rpc.Code_CODE_OK},
				RemoteUser: &user.User{Id: sharer, Username: "einstein", DisplayName: "Albert Einstein"},
			}, nil)

			n, err := converter.ConvertEvent("eventid", events.OCMCoreShareCreated{
				ShareID:       "shareid",
				Sharer:        sharer,
				GranteeUserID: grantee,
				ItemID:        "remote-item-id",
				ResourceName:  "relativity",
				CTime:         ctime,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(n.Subject).To(Equal("Resource shared"))
			Expect(n.Message).To(Equal("Albert Einstein shared relativity with you"))
			Expect(n.UserName).To(Equal("einstein"))
			Expect(n.ResourceID).To(Equal("shareid"))
			Expect(n.ResourceType).To(Equal("share"))
			Expect(n.MessageDetails["share"]).To(Equal(map[string]string{"id": "shareid"}))
			Expect(n.MessageDetails["resource"]).To(Equal(map[string]string{"name": "relativity"}))
		})

		It("names the provider domain when the sharer cannot be resolved", func() {
			gatewayClient.On("GetAcceptedUser", mock.Anything, mock.Anything).Return(&invitepb.GetAcceptedUserResponse{
				Status: &rpc.Status{Code: rpc.Code_CODE_NOT_FOUND},
			}, nil)

			n, err := converter.ConvertEvent("eventid", events.OCMCoreShareDelete{
				ShareID:      "shareid",
				Sharer:       sharer,
				Grantee:      grantee,
				ResourceName: "relativity",
				CTime:        ctime,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(n.Subject).To(Equal("Resource unshared"))
			Expect(n.Message).To(Equal("Someone at remote.example.org unshared relativity with you"))
			Expect(n.MessageRaw).To(Equal("Someone at remote.example.org unshared {resource} with you"))
			Expect(n.ResourceID).To(Equal("shareid"))
			Expect(n.MessageDetails).ToNot(HaveKey("user"))
			Expect(n.MessageDetails["resource"]).To(Equal(map[string]string{"name": "relativity"}))
		})
	})
})
