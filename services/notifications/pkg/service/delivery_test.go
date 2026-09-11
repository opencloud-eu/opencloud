package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	group "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	ocEvents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/pkg/log"
	ehmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/eventhistory/v0"
	settingsmsg "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/messages/settings/v0"
	ehsvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/eventhistory/v0"
	settingssvc "github.com/opencloud-eu/opencloud/protogen/gen/opencloud/services/settings/v0"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/channels"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/store/defaults"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v4/client"
	"google.golang.org/grpc"
)

// These doubles replace remote RPCs and SMTP. Recipient selection, rendering,
// interval splitting and the user event store run through the production code.
type deliveryGateway struct {
	gateway.GatewayAPIClient
	recipient *user.User
	disabled  bool
	lookupErr error
	authErr   error
	response  *user.GetUserByClaimResponse
	members   []*user.UserId
}

func (g *deliveryGateway) GetUser(_ context.Context, req *user.GetUserRequest, _ ...grpc.CallOption) (*user.GetUserResponse, error) {
	u := g.recipient
	if req.GetUserId().GetOpaqueId() == "alice" {
		u = &user.User{Id: req.UserId, DisplayName: "Alice", Mail: "alice@example.org"}
	}
	if req.GetUserId().GetOpaqueId() == "carol" {
		u = &user.User{Id: req.UserId, DisplayName: "Carol", Mail: "carol@example.org"}
	}
	return &user.GetUserResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, User: u}, nil
}

func (g *deliveryGateway) GetUserByClaim(_ context.Context, req *user.GetUserByClaimRequest, _ ...grpc.CallOption) (*user.GetUserByClaimResponse, error) {
	if req.GetClaim() == "userid" && req.GetValue() == "carol" {
		return &user.GetUserByClaimResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, User: &user.User{Id: &user.UserId{OpaqueId: "carol"}, Mail: "carol@example.org"}}, nil
	}
	if g.response != nil {
		return g.response, nil
	}
	if g.lookupErr != nil {
		return nil, g.lookupErr
	}
	if req.GetClaim() != "userid" || req.GetValue() != "brian" {
		return nil, errors.New("unexpected recipient lookup")
	}
	if g.disabled {
		return &user.GetUserByClaimResponse{Status: &rpc.Status{Code: rpc.Code_CODE_NOT_FOUND}}, nil
	}
	return &user.GetUserByClaimResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, User: g.recipient}, nil
}

func (g *deliveryGateway) Authenticate(context.Context, *gateway.AuthenticateRequest, ...grpc.CallOption) (*gateway.AuthenticateResponse, error) {
	if g.authErr != nil {
		return nil, g.authErr
	}
	return &gateway.AuthenticateResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, Token: "service-token", User: &user.User{Id: &user.UserId{OpaqueId: "service"}}}, nil
}

func (g *deliveryGateway) Stat(context.Context, *provider.StatRequest, ...grpc.CallOption) (*provider.StatResponse, error) {
	return &provider.StatResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, Info: &provider.ResourceInfo{
		Id: &provider.ResourceId{StorageId: "storage", SpaceId: "space", OpaqueId: "file"}, Name: "file.txt", Space: &provider.StorageSpace{Name: "Project"},
	}}, nil
}

func (g *deliveryGateway) GetGroup(context.Context, *group.GetGroupRequest, ...grpc.CallOption) (*group.GetGroupResponse, error) {
	if g.members != nil {
		return &group.GetGroupResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, Group: &group.Group{Members: g.members}}, nil
	}
	return &group.GetGroupResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, Group: &group.Group{Members: []*user.UserId{{OpaqueId: "brian"}}}}, nil
}

type deliverySelector struct {
	client gateway.GatewayAPIClient
	err    error
}

func (s deliverySelector) Next(...pool.Option) (gateway.GatewayAPIClient, error) {
	return s.client, s.err
}

type deliverySettings struct {
	settingssvc.ValueService
	interval string
	optOut   bool
}

func (s *deliverySettings) GetValueByUniqueIdentifiers(_ context.Context, req *settingssvc.GetValueByUniqueIdentifiersRequest, _ ...client.CallOption) (*settingssvc.GetValueResponse, error) {
	v := &settingsmsg.Value{}
	switch req.GetSettingId() {
	case defaults.SettingUUIDProfileEmailSendingInterval:
		v.Value = &settingsmsg.Value_StringValue{StringValue: s.interval}
	case defaults.SettingUUIDProfileDisableNotifications:
		v.Value = &settingsmsg.Value_BoolValue{BoolValue: s.optOut}
	default:
		v.Value = &settingsmsg.Value_CollectionValue{CollectionValue: &settingsmsg.CollectionValue{Values: []*settingsmsg.CollectionOption{{Key: "mail", Option: &settingsmsg.CollectionOption_BoolValue{BoolValue: true}}}}}
	}
	return &settingssvc.GetValueResponse{Value: &settingsmsg.ValueWithIdentifier{Value: v}}, nil
}

type deliveryChannel struct {
	messages []*channels.Message
	err      error
}

func (c *deliveryChannel) SendMessage(_ context.Context, m *channels.Message) error {
	if c.err != nil {
		return c.err
	}
	c.messages = append(c.messages, m)
	return nil
}

type deliveryHistory struct {
	ehsvc.EventHistoryService
	event *ehmsg.Event
}

func (h deliveryHistory) GetEvents(context.Context, *ehsvc.GetEventsRequest, ...client.CallOption) (*ehsvc.GetEventsResponse, error) {
	return &ehsvc.GetEventsResponse{Events: []*ehmsg.Event{h.event}}, nil
}

func newDeliveryFixture() (eventsNotifier, *deliveryGateway, *deliverySettings, *deliveryChannel) {
	g := &deliveryGateway{recipient: &user.User{Id: &user.UserId{OpaqueId: "brian"}, DisplayName: "Brian", Mail: "brian@example.org"}}
	v := &deliverySettings{interval: "instant"}
	c := &deliveryChannel{}
	l := log.NewLogger()
	n := eventsNotifier{logger: l, channel: c, gatewaySelector: deliverySelector{client: g}, valueService: v, openCloudURL: "https://cloud.example.org",
		filter: newNotificationFilter(l, v), splitter: newIntervalSplitter(l, v), userEventStore: newUserEventStore(l, store.Create(), nil)}
	return n, g, v, c
}

func TestDeliverySkipsDisabledUsersAcrossHandlers(t *testing.T) {
	alice, brian := &user.UserId{OpaqueId: "alice"}, &user.UserId{OpaqueId: "brian"}
	space := &provider.StorageSpaceId{OpaqueId: "storage!space"}
	file := &provider.ResourceId{StorageId: "storage", SpaceId: "space", OpaqueId: "file"}
	handlers := map[string]func(eventsNotifier){
		"SpaceShared": func(n eventsNotifier) {
			n.handleSpaceShared(events.SpaceShared{Executant: alice, GranteeUserID: brian, ID: space}, "event")
		},
		"SpaceShared/group": func(n eventsNotifier) {
			n.handleSpaceShared(events.SpaceShared{Executant: alice, GranteeGroupID: &group.GroupId{OpaqueId: "group"}, ID: space}, "event")
		},
		"SpaceUnshared": func(n eventsNotifier) {
			n.handleSpaceUnshared(events.SpaceUnshared{Executant: alice, GranteeUserID: brian, ID: space}, "event")
		},
		"SpaceMembershipExpired": func(n eventsNotifier) {
			n.handleSpaceMembershipExpired(events.SpaceMembershipExpired{GranteeUserID: brian, SpaceID: space}, "event")
		},
		"ShareCreated": func(n eventsNotifier) {
			n.handleShareCreated(events.ShareCreated{Sharer: alice, GranteeUserID: brian, ItemID: file}, "event")
		},
		"ShareExpired": func(n eventsNotifier) {
			n.handleShareExpired(events.ShareExpired{ShareOwner: alice, GranteeUserID: brian, ItemID: file}, "event")
		},
		"ShareRemoved": func(n eventsNotifier) {
			n.handleShareRemoved(events.ShareRemoved{Executant: alice, GranteeUserID: brian, ItemID: file}, "event")
		},
		"ResourceMention": func(n eventsNotifier) {
			n.handleResourceMention(ocEvents.ResourceMention{Executant: alice, UserIDs: []*user.UserId{brian}, Ref: &provider.Reference{ResourceId: file}}, "event")
		},
	}
	for name, handle := range handlers {
		t.Run(name, func(t *testing.T) {
			for _, disabled := range []bool{false, true} {
				n, g, _, ch := newDeliveryFixture()
				g.disabled = disabled
				handle(n)
				if disabled {
					require.Empty(t, ch.messages, "disabled user must not receive email")
				} else {
					require.Len(t, ch.messages, 1, "active user must receive email")
					require.Equal(t, []string{"brian@example.org"}, ch.messages[0].Recipient)
				}
			}
		})
	}
}

func TestGroupedEmailRechecksRecipientAfterQueueing(t *testing.T) {
	for _, interval := range []string{"daily", "weekly"} {
		for _, disabled := range []bool{false, true} {
			t.Run(interval+"/"+map[bool]string{false: "active", true: "disabled"}[disabled], func(t *testing.T) {
				n, g, settings, ch := newDeliveryFixture()
				settings.interval = interval
				e := events.SpaceMembershipExpired{GranteeUserID: g.recipient.Id, SpaceID: &provider.StorageSpaceId{OpaqueId: "storage!space"}, SpaceName: "Project", ExpiredAt: time.Now()}
				body, err := json.Marshal(e)
				require.NoError(t, err)
				n.registeredEvents = map[string]events.Unmarshaller{"events.SpaceMembershipExpired": events.SpaceMembershipExpired{}}
				n.userEventStore.historyClient = deliveryHistory{event: &ehmsg.Event{Type: "events.SpaceMembershipExpired", Event: body}}
				n.handleSpaceMembershipExpired(e, "event")
				require.Empty(t, ch.messages, "grouped email must wait for its job")
				keys, err := n.userEventStore.listKeys(interval)
				require.NoError(t, err)
				require.Len(t, keys, 1)
				g.disabled = disabled
				g.recipient = &user.User{Id: g.recipient.Id, Mail: "new-address@example.org", DisplayName: "Brian"}
				n.createGroupedMail(context.Background(), n.logger.Logger, keys[0])
				if disabled {
					require.Empty(t, ch.messages)
				} else {
					require.Len(t, ch.messages, 1)
					require.Equal(t, []string{"new-address@example.org"}, ch.messages[0].Recipient, "email must use the address returned by the recipient lookup")
				}
			})
		}
	}
}

func TestScienceMeshInvitationDoesNotLookUpRecipient(t *testing.T) {
	n, g, _, ch := newDeliveryFixture()
	g.lookupErr = errors.New("recipient lookup must not be called")
	n.handleScienceMeshInviteTokenGenerated(events.ScienceMeshInviteTokenGenerated{Sharer: &user.UserId{OpaqueId: "alice"}, RecipientMail: "external@example.net", Token: "invitation"})
	require.Len(t, ch.messages, 1)
	require.Equal(t, []string{"external@example.net"}, ch.messages[0].Recipient)
}

func TestDeliveryKeepsActiveGroupMember(t *testing.T) {
	for _, failedLookup := range []bool{false, true} {
		n, g, _, ch := newDeliveryFixture()
		g.disabled = true
		if failedLookup {
			g.lookupErr = errors.New("brian lookup failed")
		}
		g.members = []*user.UserId{{OpaqueId: "brian"}, {OpaqueId: "carol"}}
		n.handleSpaceShared(events.SpaceShared{Executant: &user.UserId{OpaqueId: "alice"}, GranteeGroupID: &group.GroupId{OpaqueId: "group"}, ID: &provider.StorageSpaceId{OpaqueId: "storage!space"}}, "event")
		require.Len(t, ch.messages, 1)
		require.Equal(t, []string{"carol@example.org"}, ch.messages[0].Recipient)
	}
}

func TestDeliveryRejectsUnavailableOrInvalidRecipient(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*deliveryGateway, *deliverySettings)
	}{
		{"global opt out", func(_ *deliveryGateway, s *deliverySettings) { s.optOut = true }},
		{"empty address", func(g *deliveryGateway, _ *deliverySettings) { g.recipient.Mail = "  " }},
		{"blocked status", func(g *deliveryGateway, _ *deliverySettings) {
			g.recipient.Status = user.UserStatus_USER_STATUS_BLOCKED
		}},
		{"transport error", func(g *deliveryGateway, _ *deliverySettings) { g.lookupErr = errors.New("offline") }},
		{"missing user", func(g *deliveryGateway, _ *deliverySettings) {
			g.response = &user.GetUserByClaimResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}}
		}},
		{"missing status", func(g *deliveryGateway, _ *deliverySettings) {
			g.response = &user.GetUserByClaimResponse{User: g.recipient}
		}},
		{"provider error", func(g *deliveryGateway, _ *deliverySettings) {
			g.response = &user.GetUserByClaimResponse{Status: &rpc.Status{Code: rpc.Code_CODE_INTERNAL}}
		}},
		{"wrong user", func(g *deliveryGateway, _ *deliverySettings) { g.recipient.Id = &user.UserId{OpaqueId: "other"} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			n, g, s, ch := newDeliveryFixture()
			test.change(g, s)
			n.send(context.Background(), n.logger.Logger, []recipientMessage{{recipient: &user.UserId{OpaqueId: "brian"}, message: &channels.Message{Recipient: []string{"stale@example.org"}}}})
			require.Empty(t, ch.messages)
		})
	}
}

func TestDeliveryChecksRecipientIdentity(t *testing.T) {
	for _, test := range []struct {
		name      string
		requested *user.UserId
		resolved  *user.UserId
		wantMail  bool
	}{
		{"matching full identity", &user.UserId{OpaqueId: "brian", Idp: "idp", TenantId: "tenant"}, &user.UserId{OpaqueId: "brian", Idp: "idp", TenantId: "tenant"}, true},
		{"wrong IDP", &user.UserId{OpaqueId: "brian", Idp: "idp", TenantId: "tenant"}, &user.UserId{OpaqueId: "brian", Idp: "other", TenantId: "tenant"}, false},
		{"wrong tenant", &user.UserId{OpaqueId: "brian", Idp: "idp", TenantId: "tenant"}, &user.UserId{OpaqueId: "brian", Idp: "idp", TenantId: "other"}, false},
		{"unspecified IDP and tenant", &user.UserId{OpaqueId: "brian"}, &user.UserId{OpaqueId: "brian", Idp: "idp", TenantId: "tenant"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			n, g, _, ch := newDeliveryFixture()
			g.recipient.Id = test.resolved
			n.send(context.Background(), n.logger.Logger, []recipientMessage{{recipient: test.requested, message: &channels.Message{}}})
			if test.wantMail {
				require.Len(t, ch.messages, 1)
				require.Equal(t, []string{"brian@example.org"}, ch.messages[0].Recipient)
			} else {
				require.Empty(t, ch.messages)
			}
		})
	}
}
