package service_test

import (
	"context"
	"time"

	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	"github.com/opencloud-eu/reva/v2/pkg/store"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	settingsmocks "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0/mocks"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config/defaults"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/channels"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/service"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

var _ = Describe("Notifications", func() {
	var (
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
		vs              *settingsmocks.ValueService
		sharer          = &user.User{
			Id: &user.UserId{
				OpaqueId: "sharer",
			},
			Mail:        "sharer@opencloud.eu",
			DisplayName: "Dr. S. Harer",
		}
		sharee = &user.User{
			Id: &user.UserId{
				OpaqueId: "sharee",
			},
			Mail:        "sharee@opencloud.eu",
			DisplayName: "Eric Expireling",
		}
		resourceid = &provider.ResourceId{
			StorageId: "storageid",
			SpaceId:   "spaceid",
			OpaqueId:  "itemid",
		}
	)

	BeforeEach(func() {
		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector = pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)

		gatewayClient.On("GetUser", mock.Anything, mock.MatchedBy(func(req *user.GetUserRequest) bool {
			return req.GetUserId().GetOpaqueId() == sharer.GetId().GetOpaqueId()
		})).Return(&user.GetUserResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, User: sharer}, nil)
		gatewayClient.On("GetUser", mock.Anything, mock.MatchedBy(func(req *user.GetUserRequest) bool {
			return req.GetUserId().GetOpaqueId() == sharee.GetId().GetOpaqueId()
		})).Return(&user.GetUserResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, User: sharee}, nil)
		gatewayClient.On("Authenticate", mock.Anything, mock.Anything).Return(&gateway.AuthenticateResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, User: sharer}, nil)
		gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, Info: &provider.ResourceInfo{Name: "secrets of the board", Space: &provider.StorageSpace{Name: "secret space"}}}, nil)
		vs = &settingsmocks.ValueService{}
		vs.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything).Return(&settingssvc.GetValueResponse{
			Value: &settingsmsg.ValueWithIdentifier{
				Value: &settingsmsg.Value{
					Value: &settingsmsg.Value_CollectionValue{
						CollectionValue: &settingsmsg.CollectionValue{
							Values: []*settingsmsg.CollectionOption{
								{
									Key:    "mail",
									Option: &settingsmsg.CollectionOption_BoolValue{BoolValue: true},
								},
							},
						},
					},
				},
			},
		}, nil)
	})

	DescribeTable("Sending userEventIds",
		func(tc testChannel, ev events.Event) {
			cfg := defaults.FullDefaultConfig()
			cfg.GRPCClientTLS = &shared.GRPCClientTLS{}
			ch := make(chan events.Event)
			evts := service.NewEventsNotifier(ch, tc, log.NewLogger(), gatewaySelector, vs, "",
				"", "", "", "", "", "",
				store.Create(), nil, nil)
			go evts.Run()

			ch <- ev
			select {
			case <-tc.done:
				// finished
			case <-time.Tick(3 * time.Second):
				Fail("timeout waiting for notification")
			}
		},

		Entry("Share Created", testChannel{
			expectedReceipients: []string{sharee.GetMail()},
			expectedSubject:     "Dr. S. Harer shared 'secrets of the board' with you",
			expectedTextBody: `Hello Eric Expireling

Dr. S. Harer has shared "secrets of the board" with you.

Click here to view it: files/shares/with-me


---
OpenCloud - a safe home for all your data
https://opencloud.eu
`,
			expectedSender: sharer.GetDisplayName(),
			done:           make(chan struct{}),
		}, events.Event{
			Event: events.ShareCreated{
				Sharer:        sharer.GetId(),
				GranteeUserID: sharee.GetId(),
				CTime:         utils.TimeToTS(time.Date(2023, 4, 17, 16, 42, 0, 0, time.UTC)),
				ItemID:        resourceid,
			},
		}),

		Entry("Share Expired", testChannel{
			expectedReceipients: []string{sharee.GetMail()},
			expectedSubject:     "Share to 'secrets of the board' expired at 2023-04-17 16:42:00",
			expectedTextBody: `Hello Eric Expireling,

Your share to secrets of the board has expired at 2023-04-17 16:42:00

Even though this share has been revoked you still might have access through other shares and/or space memberships.


---
OpenCloud - a safe home for all your data
https://opencloud.eu
`,
			expectedSender: sharer.GetDisplayName(),
			done:           make(chan struct{}),
		}, events.Event{
			Event: events.ShareExpired{
				ShareOwner:    sharer.GetId(),
				GranteeUserID: sharee.GetId(),
				ExpiredAt:     time.Date(2023, 4, 17, 16, 42, 0, 0, time.UTC),
				ItemID:        resourceid,
			},
		}),

		Entry("Share Removed", testChannel{
			expectedReceipients: []string{sharee.GetMail()},
			expectedSubject:     "Dr. S. Harer unshared 'secrets of the board' with you",
			expectedTextBody: `Hello Eric Expireling,

Dr. S. Harer has unshared 'secrets of the board' with you.

Even though this share has been revoked you still might have access through other shares and/or space memberships.


---
OpenCloud - a safe home for all your data
https://opencloud.eu
`,
			expectedSender: sharer.GetDisplayName(),
			done:           make(chan struct{}),
		}, events.Event{
			Event: events.ShareRemoved{
				Executant:     sharer.GetId(),
				GranteeUserID: sharee.GetId(),
				ItemID:        resourceid,
			},
		}),

		Entry("Added to Space", testChannel{
			expectedReceipients: []string{sharee.GetMail()},
			expectedSubject:     "Dr. S. Harer invited you to join secret space",
			expectedTextBody: `Hello Eric Expireling,

Dr. S. Harer has invited you to join "secret space".

Click here to view it: f/spaceid


---
OpenCloud - a safe home for all your data
https://opencloud.eu
`,
			expectedSender: sharer.GetDisplayName(),
			done:           make(chan struct{}),
		}, events.Event{
			Event: events.SpaceShared{
				Executant:     sharer.GetId(),
				Creator:       sharer.GetId(),
				GranteeUserID: sharee.GetId(),
				ID:            &provider.StorageSpaceId{OpaqueId: "spaceid"},
			},
		}),

		Entry("Removed from Space", testChannel{
			expectedReceipients: []string{sharee.GetMail()},
			expectedSubject:     "Dr. S. Harer removed you from secret space",
			expectedTextBody: `Hello Eric Expireling,

Dr. S. Harer has removed you from "secret space".

You might still have access through your other groups or direct membership.

Click here to check it: f/spaceid


---
OpenCloud - a safe home for all your data
https://opencloud.eu
`,
			expectedSender: sharer.GetDisplayName(),
			done:           make(chan struct{}),
		}, events.Event{
			Event: events.SpaceUnshared{
				Executant:     sharer.GetId(),
				GranteeUserID: sharee.GetId(),
				ID:            &provider.StorageSpaceId{OpaqueId: "spaceid"},
			},
		}),

		Entry("Space Expired", testChannel{
			expectedReceipients: []string{sharee.GetMail()},
			expectedSubject:     "Membership of 'secret space' expired at 2023-04-17 16:42:00",
			expectedTextBody: `Hello Eric Expireling,

Your membership of space secret space has expired at 2023-04-17 16:42:00

Even though this membership has expired you still might have access through other shares and/or space memberships


---
OpenCloud - a safe home for all your data
https://opencloud.eu
`,
			expectedSender: "",
			done:           make(chan struct{}),
		}, events.Event{
			Event: events.SpaceMembershipExpired{
				SpaceOwner:    sharer.GetId(),
				GranteeUserID: sharee.GetId(),
				SpaceID:       &provider.StorageSpaceId{OpaqueId: "spaceid"},
				SpaceName:     "secret space",
				ExpiredAt:     time.Date(2023, 4, 17, 16, 42, 0, 0, time.UTC),
			},
		}),
	)
})

var _ = Describe("Notifications X-Site Scripting", func() {
	var (
		gatewayClient   *cs3mocks.GatewayAPIClient
		gatewaySelector pool.Selectable[gateway.GatewayAPIClient]
		vs              *settingsmocks.ValueService
		sharer          = &user.User{
			Id: &user.UserId{
				OpaqueId: "sharer",
			},
			Mail:        "sharer@opencloud.eu",
			DisplayName: "Dr. O'reilly",
		}
		sharee = &user.User{
			Id: &user.UserId{
				OpaqueId: "sharee",
			},
			Mail:        "sharee@opencloud.eu",
			DisplayName: "<script>alert('Eric Expireling');</script>",
		}
		resourceid = &provider.ResourceId{
			StorageId: "storageid",
			SpaceId:   "spaceid",
			OpaqueId:  "itemid",
		}
	)

	BeforeEach(func() {
		pool.RemoveSelector("GatewaySelector" + "eu.opencloud.api.gateway")
		gatewayClient = &cs3mocks.GatewayAPIClient{}
		gatewaySelector = pool.GetSelector[gateway.GatewayAPIClient](
			"GatewaySelector",
			"eu.opencloud.api.gateway",
			func(cc grpc.ClientConnInterface) gateway.GatewayAPIClient {
				return gatewayClient
			},
		)

		gatewayClient.On("GetUser", mock.Anything, mock.MatchedBy(func(req *user.GetUserRequest) bool {
			return req.GetUserId().GetOpaqueId() == sharer.GetId().GetOpaqueId()
		})).Return(&user.GetUserResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, User: sharer}, nil)
		gatewayClient.On("GetUser", mock.Anything, mock.MatchedBy(func(req *user.GetUserRequest) bool {
			return req.GetUserId().GetOpaqueId() == sharee.GetId().GetOpaqueId()
		})).Return(&user.GetUserResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, User: sharee}, nil)
		gatewayClient.On("Authenticate", mock.Anything, mock.Anything).Return(&gateway.AuthenticateResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, User: sharer}, nil)
		gatewayClient.On("Stat", mock.Anything, mock.Anything).Return(&provider.StatResponse{
			Status: &rpc.Status{Code: rpc.Code_CODE_OK},
			Info: &provider.ResourceInfo{
				Name:  "<script>alert('secrets of the board');</script>",
				Space: &provider.StorageSpace{Name: "<script>alert('secret space');</script>"}},
		}, nil)
		vs = &settingsmocks.ValueService{}
		vs.On("GetValueByUniqueIdentifiers", mock.Anything, mock.Anything).Return(&settingssvc.GetValueResponse{
			Value: &settingsmsg.ValueWithIdentifier{
				Value: &settingsmsg.Value{
					Value: &settingsmsg.Value_CollectionValue{
						CollectionValue: &settingsmsg.CollectionValue{
							Values: []*settingsmsg.CollectionOption{
								{
									Key:    "mail",
									Option: &settingsmsg.CollectionOption_BoolValue{BoolValue: true},
								},
							},
						},
					},
				},
			},
		}, nil)
	})

	DescribeTable("Sending userEventIds",
		func(tc testChannel, ev events.Event) {
			cfg := defaults.FullDefaultConfig()
			cfg.GRPCClientTLS = &shared.GRPCClientTLS{}
			ch := make(chan events.Event)
			evts := service.NewEventsNotifier(ch, tc, log.NewLogger(), gatewaySelector, vs, "",
				"", "", "", "", "", "",
				store.Create(), nil, nil)
			go evts.Run()

			ch <- ev
			select {
			case <-tc.done:
				// finished
			case <-time.Tick(3 * time.Second):
				Fail("timeout waiting for notification")
			}
		},

		Entry("Share Created", testChannel{
			expectedReceipients: []string{sharee.GetMail()},
			expectedSubject:     "Dr. O'reilly shared '<script>alert('secrets of the board');</script>' with you",
			expectedTextBody: `Hello <script>alert('Eric Expireling');</script>

Dr. O'reilly has shared "<script>alert('secrets of the board');</script>" with you.

Click here to view it: files/shares/with-me


---
OpenCloud - a safe home for all your data
https://opencloud.eu
`,
			expectedHTMLBody: `<!DOCTYPE html>
<html>
<body>
<table cellspacing="0" cellpadding="0" border="0" width="100%">
    <tr>
        <td>
            <table cellspacing="0" cellpadding="0" border="0" width="600px">
                <tr>
                    <td width="20px">&nbsp;</td>
                    <td style="font-weight:normal; font-size:0.8em; line-height:1.2em; font-family:verdana,'arial',sans;">
                        Hello &lt;script&gt;alert(&#39;Eric Expireling&#39;);&lt;/script&gt;
                        <br><br>
                        Dr. O&#39;reilly has shared "&lt;script&gt;alert(&#39;secrets of the board&#39;);&lt;/script&gt;" with you.
                        <br><br>
                        Click here to view it: <a href="files/shares/with-me">files/shares/with-me</a>
                    </td>
                </tr>
                <tr>
                    <td colspan="2">&nbsp;</td>
                </tr>
                <tr>
                    <td width="20px">&nbsp;</td>
                    <td style="font-weight:normal; font-size:0.8em; line-height:1.2em; font-family:verdana,'arial',sans;">
                        <footer>
                            <br>
                            <br>
                            --- <br>
                            OpenCloud - a safe home for all your data<br>
                            <a href="https://opencloud.eu">https://opencloud.eu</a>
                        </footer>
                    </td>
                </tr>
                <tr>
                    <td colspan="2">&nbsp;</td>
                </tr>
            </table>
        </td>
    </tr>
</table>
</body>
</html>
`,
			expectedSender: sharer.GetDisplayName(),

			done: make(chan struct{}),
		}, events.Event{
			Event: events.ShareCreated{
				Sharer:        sharer.GetId(),
				GranteeUserID: sharee.GetId(),
				CTime:         utils.TimeToTS(time.Date(2023, 4, 17, 16, 42, 0, 0, time.UTC)),
				ItemID:        resourceid,
			},
		}),

		Entry("Added to Space", testChannel{
			expectedReceipients: []string{sharee.GetMail()},
			expectedSubject:     "Dr. O'reilly invited you to join <script>alert('secret space');</script>",
			expectedTextBody: `Hello <script>alert('Eric Expireling');</script>,

Dr. O'reilly has invited you to join "<script>alert('secret space');</script>".

Click here to view it: f/spaceid


---
OpenCloud - a safe home for all your data
https://opencloud.eu
`,
			expectedSender: sharer.GetDisplayName(),
			expectedHTMLBody: `<!DOCTYPE html>
<html>
<body>
<table cellspacing="0" cellpadding="0" border="0" width="100%">
    <tr>
        <td>
            <table cellspacing="0" cellpadding="0" border="0" width="600px">
                <tr>
                    <td width="20px">&nbsp;</td>
                    <td style="font-weight:normal; font-size:0.8em; line-height:1.2em; font-family:verdana,'arial',sans;">
                        Hello &lt;script&gt;alert(&#39;Eric Expireling&#39;);&lt;/script&gt;,
                        <br><br>
                        Dr. O&#39;reilly has invited you to join "&lt;script&gt;alert(&#39;secret space&#39;);&lt;/script&gt;".
                        <br><br>
                        Click here to view it: <a href="f/spaceid">f/spaceid</a>
                    </td>
                </tr>
                <tr>
                    <td colspan="2">&nbsp;</td>
                </tr>
                <tr>
                    <td width="20px">&nbsp;</td>
                    <td style="font-weight:normal; font-size:0.8em; line-height:1.2em; font-family:verdana,'arial',sans;">
                        <footer>
                            <br>
                            <br>
                            --- <br>
                            OpenCloud - a safe home for all your data<br>
                            <a href="https://opencloud.eu">https://opencloud.eu</a>
                        </footer>
                    </td>
                </tr>
                <tr>
                    <td colspan="2">&nbsp;</td>
                </tr>
            </table>
        </td>
    </tr>
</table>
</body>
</html>
`,
			done: make(chan struct{}),
		}, events.Event{
			Event: events.SpaceShared{
				Executant:     sharer.GetId(),
				Creator:       sharer.GetId(),
				GranteeUserID: sharee.GetId(),
				ID:            &provider.StorageSpaceId{OpaqueId: "spaceid"},
			},
		}),
	)
})

// NOTE: This is explictitly not testing the message itself. Should we?
type testChannel struct {
	expectedReceipients []string
	expectedSubject     string
	expectedTextBody    string
	expectedHTMLBody    string
	expectedSender      string
	done                chan struct{}
}

func (tc testChannel) SendMessage(ctx context.Context, m *channels.Message) error {
	defer GinkgoRecover()

	Expect(tc.expectedReceipients).To(Equal(m.Recipient))
	Expect(tc.expectedSubject).To(Equal(m.Subject))
	Expect(tc.expectedTextBody).To(Equal(m.TextBody))
	Expect(tc.expectedSender).To(Equal(m.Sender))
	if tc.expectedHTMLBody != "" {
		Expect(tc.expectedHTMLBody).To(Equal(m.HTMLBody))
	}
	tc.done <- struct{}{}
	return nil
}
