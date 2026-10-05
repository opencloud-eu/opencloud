// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package authguest

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	ocEvents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/jwt"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/pin"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/storage"
	storagemocks "github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/storage/mocks"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/token"
	revaevents "github.com/opencloud-eu/reva/v2/pkg/events"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	microevents "go-micro.dev/v4/events"
)

const testShareID = "e0123456-7890-abcd-ef01-234567890abc"

type testPublisher struct {
	published []any
	err       error
}

func (p *testPublisher) Publish(_ string, ev any, _ ...microevents.PublishOption) error {
	p.published = append(p.published, ev)
	return p.err
}

type gatewayTestSelector struct {
	client gateway.GatewayAPIClient
}

func (s gatewayTestSelector) Next(...pool.Option) (gateway.GatewayAPIClient, error) {
	return s.client, nil
}

func newGatewayTestSelector(client gateway.GatewayAPIClient) pool.Selectable[gateway.GatewayAPIClient] {
	return gatewayTestSelector{client: client}
}

func newGatewayMock(resp *collaboration.GetShareResponse) *cs3mocks.GatewayAPIClient {
	gwc := &cs3mocks.GatewayAPIClient{}
	gwc.On("Authenticate", mock.Anything, mock.Anything).
		Return(&gateway.AuthenticateResponse{
			Status: &rpc.Status{Code: rpc.Code_CODE_OK},
			Token:  "token",
		}, nil)
	gwc.On("GetShare", mock.Anything, mock.Anything).Return(resp, nil)
	return gwc
}

func newGuestShare(expiry time.Time) *collaboration.Share {
	return &collaboration.Share{
		Id:         &collaboration.ShareId{OpaqueId: testShareID},
		ResourceId: &provider.ResourceId{StorageId: "storage", OpaqueId: "item"},
		Expiration: utils.TimeToTS(expiry),
		Grantee: &provider.Grantee{
			Type: provider.GranteeType_GRANTEE_TYPE_USER,
			Id: &provider.Grantee_UserId{UserId: &user.UserId{
				OpaqueId: "guest@example.com",
				Type:     user.UserType_USER_TYPE_GUEST,
			}},
		},
	}
}

func newToken(t *testing.T) (string, storage.Record) {
	ts := token.NewTokenService()
	tok, err := ts.Generate(testShareID)
	require.NoError(t, err)

	rec := storage.Record{
		ShareID:     testShareID,
		ShareIDHash: tok.ShareIDHash,
		SecretHash:  tok.SecretHash(),
		Expiry:      time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
	}

	return tok.String(), rec
}

func newShareService(t *testing.T, gwc *cs3mocks.GatewayAPIClient) *AuthGuestService {
	t.Helper()
	return NewAuthGuestService(
		token.NewTokenService(),
		storagemocks.NewManager(t),
		GatewaySelector(newGatewayTestSelector(gwc)),
		ServiceAccount(config.ServiceAccount{ServiceAccountID: "sa-id", ServiceAccountSecret: "sa-secret"}),
	)
}

func newRedeemService(t *testing.T, store storage.Manager, gwc *cs3mocks.GatewayAPIClient) *AuthGuestService {
	t.Helper()
	return NewAuthGuestService(
		token.NewTokenService(),
		store,
		GatewaySelector(newGatewayTestSelector(gwc)),
		ServiceAccount(config.ServiceAccount{ServiceAccountID: "sa-id", ServiceAccountSecret: "sa-secret"}),
		JWT(jwt.NewJwtService("test-secret", time.Hour)),
	)
}

func TestCreateTokenPersistsRecord(t *testing.T) {
	store := storagemocks.NewManager(t)
	expiry := time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)
	gwc := newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share: &collaboration.Share{
			Id:         &collaboration.ShareId{OpaqueId: testShareID},
			Expiration: utils.TimeToTS(expiry),
		},
	})

	var added storage.Record
	store.On("Add", mock.Anything).Run(func(args mock.Arguments) {
		added = args.Get(0).(storage.Record)
	}).Return(nil)

	s := NewAuthGuestService(
		token.NewTokenService(),
		store,
		GatewaySelector(newGatewayTestSelector(gwc)),
		ServiceAccount(config.ServiceAccount{ServiceAccountID: "sa-id", ServiceAccountSecret: "sa-secret"}),
	)

	tok, err := s.CreateToken(context.Background(), testShareID)
	require.NoError(t, err)

	store.AssertCalled(t, "Add", mock.Anything)
	assert.Equal(t, testShareID, added.ShareID)
	assert.Equal(t, tok.ShareIDHash, added.ShareIDHash)
	assert.Equal(t, tok.SecretHash(), added.SecretHash)
	assert.WithinDuration(t, time.Now().Add(guestLinkTokenTTL), added.Expiry, time.Minute)
	assert.False(t, added.Redeemed)
}

func TestVerifyToken(t *testing.T) {
	tests := []struct {
		name     string
		expired  bool
		redeemed bool
		wantErr  error
	}{
		{name: "valid"},
		{name: "expired", expired: true, wantErr: ErrExpired},
		{name: "already redeemed", redeemed: true, wantErr: ErrAlreadyRedeemed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := storagemocks.NewManager(t)
			s := NewAuthGuestService(token.NewTokenService(), store)
			tok, rec := newToken(t)
			if tt.expired {
				rec.Expiry = time.Now().Add(-time.Hour)
			}
			if tt.redeemed {
				rec.Redeemed = true
			}
			store.On("Get", rec.ShareIDHash).Return(rec, nil)

			got, err := s.verifyToken(tok)
			if tt.wantErr != nil {
				var re *RedeemError
				require.ErrorAs(t, err, &re)
				assert.ErrorIs(t, re.ErrorType, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, rec, *got)
		})
	}
}

func TestVerifyTokenDoesNotDiscloseShare(t *testing.T) {
	tok, rec := newToken(t)
	// flip the last character of the secret
	last := "x"
	if strings.HasSuffix(tok, last) {
		last = "y"
	}
	tamperedSecret := tok[:len(tok)-1] + last

	tests := []struct {
		name  string
		token string
		setup func(store *storagemocks.Manager)
	}{
		{
			name:  "malformed token",
			token: "not-a-token",
		},
		{
			name:  "unknown record",
			token: tok,
			setup: func(store *storagemocks.Manager) {
				store.On("Get", rec.ShareIDHash).Return(storage.Record{}, storage.ErrNotFound)
			},
		},
		{
			name:  "invalid hash",
			token: "v1.ab.secret",
			setup: func(store *storagemocks.Manager) {
				store.On("Get", "ab").Return(storage.Record{}, storage.ErrInvalidHash)
			},
		},
		{
			name:  "wrong secret",
			token: tamperedSecret,
			setup: func(store *storagemocks.Manager) {
				store.On("Get", rec.ShareIDHash).Return(rec, nil)
			},
		},
		{
			name:  "wrong secret on expired and redeemed record",
			token: tamperedSecret,
			setup: func(store *storagemocks.Manager) {
				r := rec
				r.Expiry = time.Now().Add(-time.Hour)
				r.Redeemed = true
				store.On("Get", rec.ShareIDHash).Return(r, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := storagemocks.NewManager(t)
			if tt.setup != nil {
				tt.setup(store)
			}
			s := NewAuthGuestService(token.NewTokenService(), store)

			_, err := s.verifyToken(tt.token)
			var re *RedeemError
			require.ErrorAs(t, err, &re)
			assert.ErrorIs(t, re.ErrorType, token.ErrInvalidToken)
			assert.Empty(t, re.ShareID)
		})
	}
}

func TestVerifyTokenStorageFailure(t *testing.T) {
	tok, rec := newToken(t)
	store := storagemocks.NewManager(t)
	store.On("Get", rec.ShareIDHash).Return(storage.Record{}, errors.New("disk on fire"))
	s := NewAuthGuestService(token.NewTokenService(), store)

	_, err := s.verifyToken(tok)
	require.Error(t, err)
	var re *RedeemError
	assert.False(t, errors.As(err, &re), "storage failures must surface as internal errors")
}

func TestValidateShare(t *testing.T) {
	share := &collaboration.Share{Id: &collaboration.ShareId{OpaqueId: testShareID}}
	notExpiredShare := &collaboration.Share{
		Id:         &collaboration.ShareId{OpaqueId: testShareID},
		Expiration: utils.TimeToTS(time.Now().Add(time.Hour)),
	}
	expiredShare := &collaboration.Share{
		Id:         &collaboration.ShareId{OpaqueId: testShareID},
		Expiration: utils.TimeToTS(time.Now().Add(-time.Hour)),
	}

	tests := []struct {
		name     string
		response *collaboration.GetShareResponse
		wantErr  error
	}{
		{
			name:     "valid",
			response: &collaboration.GetShareResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, Share: share},
		},
		{
			name:     "not expired",
			response: &collaboration.GetShareResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, Share: notExpiredShare},
		},
		{
			name:     "expired",
			response: &collaboration.GetShareResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}, Share: expiredShare},
			wantErr:  ErrShareExpired,
		},
		{
			name:     "not found",
			response: &collaboration.GetShareResponse{Status: &rpc.Status{Code: rpc.Code_CODE_NOT_FOUND}},
			wantErr:  ErrShareNotFound,
		},
		{
			name:     "nil share",
			response: &collaboration.GetShareResponse{Status: &rpc.Status{Code: rpc.Code_CODE_OK}},
			wantErr:  ErrShareNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newShareService(t, newGatewayMock(tt.response))

			got, err := s.validateShare(context.Background(), testShareID)
			if tt.wantErr != nil {
				var re *RedeemError
				require.ErrorAs(t, err, &re)
				assert.ErrorIs(t, re.ErrorType, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.response.GetShare(), got)
		})
	}
}

func TestRedeem(t *testing.T) {
	store := storagemocks.NewManager(t)
	tok, rec := newToken(t)
	store.On("Get", rec.ShareIDHash).Return(rec, nil)
	store.On("Redeem", rec.ShareIDHash).Return(nil)

	share := &collaboration.Share{Id: &collaboration.ShareId{OpaqueId: testShareID}}
	s := newRedeemService(t, store, newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  share,
	}))

	result, err := s.Redeem(context.Background(), tok)
	require.NoError(t, err)
	require.NotEmpty(t, result.SessionToken)
	assert.Equal(t, testShareID, result.ShareID)

	store.AssertCalled(t, "Redeem", rec.ShareIDHash)
}

func TestRedeemAlreadyRedeemed(t *testing.T) {
	store := storagemocks.NewManager(t)
	tok, rec := newToken(t)
	store.On("Get", rec.ShareIDHash).Return(rec, nil)
	store.On("Redeem", rec.ShareIDHash).Return(storage.ErrAlreadyRedeemed)

	share := &collaboration.Share{Id: &collaboration.ShareId{OpaqueId: testShareID}}
	s := newRedeemService(t, store, newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  share,
	}))

	_, err := s.Redeem(context.Background(), tok)
	var re *RedeemError
	require.ErrorAs(t, err, &re)
	assert.ErrorIs(t, re.ErrorType, ErrAlreadyRedeemed)
}

func newRenewService(t *testing.T, store storage.Manager, gwc *cs3mocks.GatewayAPIClient, publisher revaevents.Publisher) *AuthGuestService {
	t.Helper()
	return NewAuthGuestService(
		token.NewTokenService(),
		store,
		GatewaySelector(newGatewayTestSelector(gwc)),
		ServiceAccount(config.ServiceAccount{ServiceAccountID: "sa-id", ServiceAccountSecret: "sa-secret"}),
		EventsPublisher(publisher),
	)
}

func existingRecord() storage.Record {
	return storage.Record{
		ShareID:     testShareID,
		ShareIDHash: token.Hash(testShareID),
	}
}

func TestRenewPersistsRecordAndPublishesEvent(t *testing.T) {
	store := storagemocks.NewManager(t)
	store.On("Get", token.Hash(testShareID)).Return(existingRecord(), nil)

	var replaced storage.Record
	store.On("Replace", mock.Anything).Run(func(args mock.Arguments) {
		replaced = args.Get(0).(storage.Record)
	}).Return(nil)

	publisher := &testPublisher{}
	gwc := newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  newGuestShare(time.Now().Add(time.Hour)),
	})

	s := newRenewService(t, store, gwc, publisher)

	require.NoError(t, s.Renew(context.Background(), testShareID))

	store.AssertCalled(t, "Replace", mock.Anything)
	assert.Equal(t, testShareID, replaced.ShareID)
	assert.Equal(t, token.Hash(testShareID), replaced.ShareIDHash)
	assert.NotEmpty(t, replaced.SecretHash)
	assert.True(t, strings.HasPrefix(replaced.PinHash, "$argon2id$"))
	assert.WithinDuration(t, time.Now().Add(guestLinkTokenTTL), replaced.Expiry, time.Minute)
	assert.WithinDuration(t, time.Now().Add(guestPinTTL), replaced.PinExpiry, time.Minute)
	assert.False(t, replaced.Redeemed)

	require.Len(t, publisher.published, 1)
	ev, ok := publisher.published[0].(ocEvents.GuestTokenRenewed)
	require.True(t, ok)
	assert.Equal(t, testShareID, ev.ShareID.GetOpaqueId())
	assert.Equal(t, "guest@example.com", ev.GranteeEmail)
	assert.Equal(t, "item", ev.ItemID.GetOpaqueId())
	assert.NotEmpty(t, ev.Token)
	assert.NotEmpty(t, ev.Pin)

	published, err := token.NewTokenService().Parse(ev.Token)
	require.NoError(t, err)
	assert.Equal(t, replaced.SecretHash, published.SecretHash())

	pinOK, err := pin.Verify(ev.Pin, replaced.PinHash)
	require.NoError(t, err)
	assert.True(t, pinOK)
}

func TestRenewRecordNotFound(t *testing.T) {
	store := storagemocks.NewManager(t)
	store.On("Get", token.Hash(testShareID)).Return(storage.Record{}, storage.ErrNotFound)

	publisher := &testPublisher{}
	s := newRenewService(t, store, newGatewayMock(&collaboration.GetShareResponse{}), publisher)

	err := s.Renew(context.Background(), testShareID)
	require.Error(t, err)

	var re *RedeemError
	require.ErrorAs(t, err, &re)
	assert.ErrorIs(t, re.ErrorType, storage.ErrNotFound)
	assert.Equal(t, testShareID, re.ShareID)

	store.AssertNotCalled(t, "Replace", mock.Anything)
	assert.Empty(t, publisher.published)
}

func TestRenewWithoutPublisher(t *testing.T) {
	store := storagemocks.NewManager(t)
	store.On("Get", token.Hash(testShareID)).Return(existingRecord(), nil)

	s := newRenewService(t, store, newGatewayMock(&collaboration.GetShareResponse{}), nil)

	err := s.Renew(context.Background(), testShareID)
	require.ErrorIs(t, err, ErrEventsNotConfigured)
	store.AssertNotCalled(t, "Replace", mock.Anything)
}

func TestRenewPublishFailure(t *testing.T) {
	store := storagemocks.NewManager(t)
	store.On("Get", token.Hash(testShareID)).Return(existingRecord(), nil)
	store.On("Replace", mock.Anything).Return(nil)

	publisher := &testPublisher{err: errors.New("publish failed")}
	gwc := newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  newGuestShare(time.Now().Add(time.Hour)),
	})

	s := newRenewService(t, store, gwc, publisher)

	err := s.Renew(context.Background(), testShareID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish failed")
}

func TestRenewShareNotFound(t *testing.T) {
	store := storagemocks.NewManager(t)
	store.On("Get", token.Hash(testShareID)).Return(existingRecord(), nil)

	s := newRenewService(t, store, newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_NOT_FOUND},
	}), &testPublisher{})

	err := s.Renew(context.Background(), testShareID)
	require.Error(t, err)

	var re *RedeemError
	require.ErrorAs(t, err, &re)
	assert.ErrorIs(t, re.ErrorType, ErrShareNotFound)
	assert.Equal(t, testShareID, re.ShareID)
}

func TestRenewShareExpired(t *testing.T) {
	store := storagemocks.NewManager(t)
	store.On("Get", token.Hash(testShareID)).Return(existingRecord(), nil)

	s := newRenewService(t, store, newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  newGuestShare(time.Now().Add(-time.Hour)),
	}), &testPublisher{})

	err := s.Renew(context.Background(), testShareID)
	require.Error(t, err)

	var re *RedeemError
	require.ErrorAs(t, err, &re)
	assert.ErrorIs(t, re.ErrorType, ErrShareExpired)
	assert.Equal(t, testShareID, re.ShareID)
}
