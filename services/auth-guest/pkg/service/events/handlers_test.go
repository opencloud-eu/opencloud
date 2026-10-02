// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"testing"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	ocEvents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/authguest/mocks"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/token"
	"github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	microevents "go-micro.dev/v4/events"
)

const testShareID = "e0123456-7890-abcd-ef01-234567890abc"

type testBus struct {
	published []any
}

func (tb *testBus) Publish(_ string, ev any, _ ...microevents.PublishOption) error {
	tb.published = append(tb.published, ev)
	return nil
}

func (tb *testBus) Consume(_ string, _ ...microevents.ConsumeOption) (<-chan microevents.Event, error) {
	return nil, nil
}

func newConsumer(t *testing.T, authGuest authguest.AuthGuest) (*EventConsumer, *testBus) {
	t.Helper()
	bus := &testBus{}
	consumer, err := NewEventConsumer(bus, AuthGuestService(authGuest))
	require.NoError(t, err)

	return consumer, bus
}

func TestHandleShareCreated(t *testing.T) {
	tok, err := token.NewTokenService().Generate(testShareID)
	require.NoError(t, err)

	svcMock := mocks.NewAuthGuest(t)
	svcMock.On("CreateToken", mock.Anything, testShareID).Return(tok, nil)

	svc, bus := newConsumer(t, svcMock)

	ev := events.ShareCreated{
		ShareID:       &collaboration.ShareId{OpaqueId: testShareID},
		Sharer:        &user.UserId{OpaqueId: "sharer"},
		ItemID:        &provider.ResourceId{StorageId: "storage", OpaqueId: "item"},
		ResourceName:  "resource",
		GranteeUserID: &user.UserId{OpaqueId: "guest@example.org", Type: user.UserType_USER_TYPE_GUEST},
	}

	require.NoError(t, svc.handleShareCreated(context.Background(), ev))

	svcMock.AssertCalled(t, "CreateToken", mock.Anything, testShareID)

	require.Len(t, bus.published, 1)
	published, ok := bus.published[0].(ocEvents.GuestTokenCreated)
	require.True(t, ok)
	assert.Equal(t, testShareID, published.ShareID.GetOpaqueId())
	assert.Equal(t, ev.Sharer, published.Sharer)
	assert.Equal(t, "guest@example.org", published.GranteeEmail)
	assert.Equal(t, ev.ItemID, published.ItemID)
	assert.Equal(t, ev.ResourceName, published.ResourceName)
	assert.Equal(t, tok.String(), published.Token)
}

func TestHandleShareCreatedSkipsNonGuest(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)
	svc, bus := newConsumer(t, svcMock)

	ev := events.ShareCreated{
		ShareID:       &collaboration.ShareId{OpaqueId: testShareID},
		GranteeUserID: &user.UserId{OpaqueId: "user", Type: user.UserType_USER_TYPE_PRIMARY},
	}

	require.NoError(t, svc.handleShareCreated(context.Background(), ev))

	svcMock.AssertNotCalled(t, "CreateToken", mock.Anything, mock.Anything)
	assert.Empty(t, bus.published)
}

func TestHandleShareRemoved(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)
	svcMock.On("CleanupShare", testShareID).Return(nil)
	svc, _ := newConsumer(t, svcMock)

	ev := events.ShareRemoved{
		ShareID:       &collaboration.ShareId{OpaqueId: testShareID},
		GranteeUserID: &user.UserId{OpaqueId: "guest@example.org", Type: user.UserType_USER_TYPE_GUEST},
	}

	require.NoError(t, svc.handleShareRemoved(context.Background(), ev))

	svcMock.AssertCalled(t, "CleanupShare", testShareID)
}

func TestHandleShareRemovedSkipsNonGuest(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)
	svc, _ := newConsumer(t, svcMock)

	ev := events.ShareRemoved{
		ShareID:       &collaboration.ShareId{OpaqueId: testShareID},
		GranteeUserID: &user.UserId{OpaqueId: "user", Type: user.UserType_USER_TYPE_PRIMARY},
	}

	require.NoError(t, svc.handleShareRemoved(context.Background(), ev))

	svcMock.AssertNotCalled(t, "CleanupShare", mock.Anything)
}

func TestHandleShareExpired(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)
	svcMock.On("CleanupShare", testShareID).Return(nil)
	svc, _ := newConsumer(t, svcMock)

	ev := events.ShareExpired{
		ShareID:       &collaboration.ShareId{OpaqueId: testShareID},
		GranteeUserID: &user.UserId{OpaqueId: "guest@example.org", Type: user.UserType_USER_TYPE_GUEST},
	}

	require.NoError(t, svc.handleShareExpired(context.Background(), ev))

	svcMock.AssertCalled(t, "CleanupShare", testShareID)
}

func TestHandleShareExpiredSkipsNonGuest(t *testing.T) {
	svcMock := mocks.NewAuthGuest(t)
	svc, _ := newConsumer(t, svcMock)

	ev := events.ShareExpired{
		ShareID:       &collaboration.ShareId{OpaqueId: testShareID},
		GranteeUserID: &user.UserId{OpaqueId: "user", Type: user.UserType_USER_TYPE_PRIMARY},
	}

	require.NoError(t, svc.handleShareExpired(context.Background(), ev))

	svcMock.AssertNotCalled(t, "CleanupShare", mock.Anything)
}
