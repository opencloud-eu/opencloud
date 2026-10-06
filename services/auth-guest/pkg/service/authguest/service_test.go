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
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/jwt"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/storage"
	storagemocks "github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/storage/mocks"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/token"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils"
	cs3mocks "github.com/opencloud-eu/reva/v2/tests/cs3mocks/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testShareID = "e0123456-7890-abcd-ef01-234567890abc"

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
