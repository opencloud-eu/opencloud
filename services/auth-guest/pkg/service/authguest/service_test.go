// Copyright 2026 OpenCloud GmbH <mail@opencloud.eu>
// SPDX-License-Identifier: Apache-2.0

package authguest

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gateway "github.com/cs3org/go-cs3apis/cs3/gateway/v1beta1"
	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	rpc "github.com/cs3org/go-cs3apis/cs3/rpc/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	ocEvents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/config"
	"github.com/opencloud-eu/opencloud/services/auth-guest/pkg/service/guard"
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

// rewriteStore wraps a Manager and runs onGet once after the first Get, which
// lets tests simulate a concurrent modification between a read and a later
// write.
type rewriteStore struct {
	storage.Manager
	onGet func()
}

func (s *rewriteStore) Get(shareIDHash string) (*storage.Record, error) {
	rec, err := s.Manager.Get(shareIDHash)
	if s.onGet != nil {
		onGet := s.onGet
		s.onGet = nil
		onGet()
	}
	return rec, err
}

// mutatingPublisher wraps testPublisher and runs onPublish once before
// publishing, which lets tests simulate a concurrent modification during a
// failed publish.
type mutatingPublisher struct {
	testPublisher
	onPublish func()
}

func (p *mutatingPublisher) Publish(name string, ev any, opts ...microevents.PublishOption) error {
	if p.onPublish != nil {
		onPublish := p.onPublish
		p.onPublish = nil
		onPublish()
	}
	return p.testPublisher.Publish(name, ev, opts...)
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

func newTokenFor(t *testing.T, shareID string) (string, storage.Record) {
	t.Helper()
	ts := token.NewTokenService()
	tok, err := ts.Generate(shareID)
	require.NoError(t, err)

	rec := storage.Record{
		ShareID:     shareID,
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

func newAuthGuestService(t *testing.T, store storage.Manager, gwc *cs3mocks.GatewayAPIClient) *AuthGuestService {
	t.Helper()
	return NewAuthGuestService(
		token.NewTokenService(),
		store,
		GatewaySelector(newGatewayTestSelector(gwc)),
		ServiceAccount(config.ServiceAccount{ServiceAccountID: "sa-id", ServiceAccountSecret: "sa-secret"}),
		JWT(jwt.NewJwtService("test-secret", time.Hour)),
		PinAndNonceTTL(time.Minute),
		BackoffLimiter(config.BackoffLimiter{
			MaxAttempts: 5,
			BackoffBase: time.Second,
			BackoffCap:  30 * time.Second,
		}),
		QuotaLimiter(config.QuotaLimiter{
			Cooldown: time.Minute,
			Window:   24 * time.Hour,
			DailyCap: 20,
		}),
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
			store.On("Get", rec.ShareIDHash).Return(&rec, nil)

			got, err := s.verifyToken(tok)
			if tt.wantErr != nil {
				var ge *GuestError
				require.ErrorAs(t, err, &ge)
				assert.ErrorIs(t, ge.ErrorType, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, rec, *got)
		})
	}
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
				var ge *GuestError
				require.ErrorAs(t, err, &ge)
				assert.ErrorIs(t, ge.ErrorType, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.response.GetShare(), got)
		})
	}
}

func TestVerifyTokenSuccess(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tok, rec := newToken(t)
	require.NoError(t, store.Add(rec))

	share := &collaboration.Share{Id: &collaboration.ShareId{OpaqueId: testShareID}}
	s := newAuthGuestService(t, store, newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  share,
	}))

	result, err := s.VerifyToken(context.Background(), tok)
	require.NoError(t, err)
	require.NotEmpty(t, result.SessionToken)
	assert.WithinDuration(t, time.Now().Add(time.Hour), result.ExpiresAt, 2*time.Second)
	assert.Equal(t, testShareID, result.ShareID)

	got, err := store.Get(rec.ShareIDHash)
	require.NoError(t, err)
	assert.True(t, got.Redeemed)
}

func TestVerifyTokenAlreadyRedeemed(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tok, rec := newToken(t)
	rec.Redeemed = true
	require.NoError(t, store.Add(rec))

	share := &collaboration.Share{Id: &collaboration.ShareId{OpaqueId: testShareID}}
	s := newAuthGuestService(t, store, newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  share,
	}))

	_, err := s.VerifyToken(context.Background(), tok)
	var ge *GuestError
	require.ErrorAs(t, err, &ge)
	assert.ErrorIs(t, ge.ErrorType, ErrAlreadyRedeemed)
}

func TestVerifyTokenConflictIsAlreadyRedeemed(t *testing.T) {
	base := storage.NewFileManager(t.TempDir())
	tok, rec := newToken(t)
	require.NoError(t, base.Add(rec))

	store := &rewriteStore{Manager: base}
	store.onGet = func() {
		require.NoError(t, base.Update(rec.ShareIDHash, func(r *storage.Record) error {
			return nil
		}))
	}

	share := &collaboration.Share{Id: &collaboration.ShareId{OpaqueId: testShareID}}
	s := newAuthGuestService(t, store, newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  share,
	}))

	_, err := s.VerifyToken(context.Background(), tok)
	var ge *GuestError
	require.ErrorAs(t, err, &ge)
	assert.ErrorIs(t, ge.ErrorType, ErrAlreadyRedeemed)

	got, err := base.Get(rec.ShareIDHash)
	require.NoError(t, err)
	assert.False(t, got.Redeemed)
}

func newRenewService(t *testing.T, store storage.Manager, gwc *cs3mocks.GatewayAPIClient, publisher revaevents.Publisher) *AuthGuestService {
	t.Helper()
	return newRenewServiceWithLimiter(t, store, gwc, publisher, config.QuotaLimiter{
		Window:   24 * time.Hour,
		DailyCap: 100,
	})
}

func newRenewServiceWithLimiter(t *testing.T, store storage.Manager, gwc *cs3mocks.GatewayAPIClient, publisher revaevents.Publisher, quota config.QuotaLimiter) *AuthGuestService {
	t.Helper()
	return NewAuthGuestService(
		token.NewTokenService(),
		store,
		GatewaySelector(newGatewayTestSelector(gwc)),
		ServiceAccount(config.ServiceAccount{ServiceAccountID: "sa-id", ServiceAccountSecret: "sa-secret"}),
		JWT(jwt.NewJwtService("test-secret", 24*time.Hour)),
		EventsPublisher(publisher),
		PinAndNonceTTL(time.Hour),
		BackoffLimiter(config.BackoffLimiter{
			MaxAttempts: 5,
			BackoffBase: time.Second,
			BackoffCap:  30 * time.Second,
		}),
		QuotaLimiter(quota),
	)
}

func existingRecord() storage.Record {
	return storage.Record{
		ShareID:     testShareID,
		ShareIDHash: token.Hash(testShareID),
	}
}

// pinRecord returns a record holding a freshly issued nonce, together with the
// nonce to pass to VerifyPin.
func pinRecord(t *testing.T, pinHash string, pinExpiry time.Time, backoff guard.BackoffLimiterState, nonceExpiry time.Time) (storage.Record, string) {
	t.Helper()

	var state guard.NonceState
	nonce, err := guard.NewNonce(time.Hour).Issue(&state, time.Now())
	require.NoError(t, err)
	if !nonceExpiry.IsZero() {
		state.Expiry = nonceExpiry
	}

	return storage.Record{
		ShareID:             testShareID,
		ShareIDHash:         token.Hash(testShareID),
		PinHash:             pinHash,
		PinExpiry:           pinExpiry,
		NonceState:          state,
		BackoffLimiterState: backoff,
	}, nonce
}

func validShareResponse() *collaboration.GetShareResponse {
	return &collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  newGuestShare(time.Now().Add(time.Hour)),
	}
}

func TestRenewPersistsRecordAndPublishesEvent(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tok, rec := newToken(t)
	require.NoError(t, store.Add(rec))

	publisher := &testPublisher{}
	gwc := newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  newGuestShare(time.Now().Add(time.Hour)),
	})

	s := newRenewService(t, store, gwc, publisher)

	nonce, err := s.Renew(context.Background(), testShareID, tok)
	require.NoError(t, err)

	replaced, err := store.Get(token.Hash(testShareID))
	require.NoError(t, err)
	assert.Equal(t, rec.Revision+1, replaced.Revision)
	assert.Equal(t, testShareID, replaced.ShareID)
	assert.Equal(t, token.Hash(testShareID), replaced.ShareIDHash)
	assert.NotEmpty(t, replaced.SecretHash)
	assert.True(t, strings.HasPrefix(replaced.PinHash, "$argon2id$"))
	assert.WithinDuration(t, time.Now().Add(guestLinkTokenTTL), replaced.Expiry, time.Minute)
	assert.WithinDuration(t, time.Now().Add(time.Hour), replaced.PinExpiry, time.Minute)
	assert.Equal(t, token.Hash(nonce), replaced.NonceState.Hash)
	assert.NoError(t, guard.NewNonce(time.Hour).Verify(&replaced.NonceState, nonce, time.Now()))
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

func TestRenewLinkTokenIgnoresRedeemedAndExpiry(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tok, rec := newToken(t)
	rec.Redeemed = true
	rec.Expiry = time.Now().Add(-time.Hour)
	require.NoError(t, store.Add(rec))

	s := newRenewService(t, store, newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  newGuestShare(time.Now().Add(time.Hour)),
	}), &testPublisher{})

	_, err := s.Renew(context.Background(), testShareID, tok)
	require.NoError(t, err)
}

// conflictOnceStore returns ErrConflict on the first UpdateFrom, simulating a
// concurrent modification, then delegates to the wrapped Manager.
type conflictOnceStore struct {
	storage.Manager
	updates int
}

func (s *conflictOnceStore) UpdateFrom(seen storage.Record, fn func(*storage.Record) error) (*storage.Record, error) {
	s.updates++
	if s.updates == 1 {
		return nil, storage.ErrConflict
	}
	return s.Manager.UpdateFrom(seen, fn)
}

func TestRenewRetriesOnConflict(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tok, rec := newToken(t)
	require.NoError(t, store.Add(rec))

	publisher := &testPublisher{}
	gwc := newGatewayMock(validShareResponse())
	flaky := &conflictOnceStore{Manager: store}
	s := newRenewService(t, flaky, gwc, publisher)

	nonce, err := s.Renew(context.Background(), testShareID, tok)
	require.NoError(t, err)
	assert.NotEmpty(t, nonce)
	assert.Equal(t, 2, flaky.updates)

	renewed, err := store.Get(token.Hash(testShareID))
	require.NoError(t, err)
	assert.Equal(t, rec.Revision+1, renewed.Revision)
	assert.NoError(t, guard.NewNonce(time.Hour).Verify(&renewed.NonceState, nonce, time.Now()))
	require.Len(t, publisher.published, 1)

	// The remote share is validated only once; the retry re-reads the record only.
	gwc.AssertNumberOfCalls(t, "GetShare", 1)
}

func TestRenewInvalidCredentials(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	s := newRenewService(t, store, newGatewayMock(&collaboration.GetShareResponse{}), &testPublisher{})

	otherToken, _ := newTokenFor(t, "other-share")

	tests := []struct {
		name    string
		shareID string
		token   string
		wantErr error
	}{
		{name: "empty token", shareID: testShareID, wantErr: token.ErrInvalidToken},
		{name: "invalid token", shareID: testShareID, token: "garbage", wantErr: token.ErrInvalidToken},
		{name: "token for another share", shareID: testShareID, token: otherToken, wantErr: ErrPermissionMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Renew(context.Background(), tt.shareID, tt.token)

			var ge *GuestError
			require.ErrorAs(t, err, &ge)
			assert.ErrorIs(t, ge.ErrorType, tt.wantErr)
		})
	}
}

func TestRenewRecordNotFound(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tok, _ := newToken(t)

	publisher := &testPublisher{}
	s := newRenewService(t, store, newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  newGuestShare(time.Now().Add(time.Hour)),
	}), publisher)

	_, err := s.Renew(context.Background(), testShareID, tok)
	require.Error(t, err)

	var ge *GuestError
	require.ErrorAs(t, err, &ge)
	assert.ErrorIs(t, ge.ErrorType, storage.ErrNotFound)
	assert.Equal(t, testShareID, ge.ShareID)

	assert.Empty(t, publisher.published)
}

func TestRenewWithoutPublisher(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	require.NoError(t, store.Add(existingRecord()))

	s := newRenewService(t, store, newGatewayMock(&collaboration.GetShareResponse{}), nil)

	_, err := s.Renew(context.Background(), testShareID, "irrelevant")
	require.ErrorIs(t, err, ErrEventsNotConfigured)

	got, err := store.Get(token.Hash(testShareID))
	require.NoError(t, err)
	assert.Empty(t, got.SecretHash)
	assert.Empty(t, got.PinHash)
}

func TestRenewRollsBackOnPublishFailure(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tok, rec := newToken(t)
	require.NoError(t, store.Add(rec))

	publisher := &testPublisher{err: errors.New("publish failed")}
	gwc := newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  newGuestShare(time.Now().Add(time.Hour)),
	})

	s := newRenewService(t, store, gwc, publisher)

	_, err := s.Renew(context.Background(), testShareID, tok)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish failed")

	got, err := store.Get(token.Hash(testShareID))
	require.NoError(t, err)
	assert.Equal(t, rec.SecretHash, got.SecretHash)
	assert.Empty(t, got.PinHash)
	assert.True(t, got.PinExpiry.IsZero())
	assert.False(t, got.Redeemed)
}

func TestRenewRollbackConflictKeepsConcurrentChange(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tok, rec := newToken(t)
	require.NoError(t, store.Add(rec))

	publisher := &mutatingPublisher{testPublisher: testPublisher{err: errors.New("publish failed")}}
	publisher.onPublish = func() {
		require.NoError(t, store.Update(rec.ShareIDHash, func(r *storage.Record) error {
			r.SecretHash = "concurrent"
			return nil
		}))
	}

	s := newRenewService(t, store, newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  newGuestShare(time.Now().Add(time.Hour)),
	}), publisher)

	_, err := s.Renew(context.Background(), testShareID, tok)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publish failed")
	assert.Contains(t, err.Error(), "rollback failed")

	got, err := store.Get(rec.ShareIDHash)
	require.NoError(t, err)
	assert.Equal(t, "concurrent", got.SecretHash)
}

func TestRenewShareNotFound(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tok, rec := newToken(t)
	require.NoError(t, store.Add(rec))

	s := newRenewService(t, store, newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_NOT_FOUND},
	}), &testPublisher{})

	_, err := s.Renew(context.Background(), testShareID, tok)
	require.Error(t, err)

	var ge *GuestError
	require.ErrorAs(t, err, &ge)
	assert.ErrorIs(t, ge.ErrorType, ErrShareNotFound)
	assert.Equal(t, testShareID, ge.ShareID)
}

func TestRenewShareExpired(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tok, rec := newToken(t)
	require.NoError(t, store.Add(rec))

	s := newRenewService(t, store, newGatewayMock(&collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  newGuestShare(time.Now().Add(-time.Hour)),
	}), &testPublisher{})

	_, err := s.Renew(context.Background(), testShareID, tok)
	require.Error(t, err)

	var ge *GuestError
	require.ErrorAs(t, err, &ge)
	assert.ErrorIs(t, ge.ErrorType, ErrShareExpired)
	assert.Equal(t, testShareID, ge.ShareID)
}

func TestRenewRecordsCooldown(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tok, rec := newToken(t)
	require.NoError(t, store.Add(rec))

	s := newRenewServiceWithLimiter(t, store, newGatewayMock(validShareResponse()), &testPublisher{}, config.QuotaLimiter{
		Cooldown: time.Minute,
		Window:   24 * time.Hour,
		DailyCap: 20,
	})

	_, err := s.Renew(context.Background(), testShareID, tok)
	require.NoError(t, err)

	got, err := store.Get(token.Hash(testShareID))
	require.NoError(t, err)
	assert.Equal(t, 1, got.QuotaLimiterState.Count)
	assert.False(t, got.QuotaLimiterState.WindowStart.IsZero())
	assert.WithinDuration(t, time.Now().Add(time.Minute), got.QuotaLimiterState.NotBefore, time.Second)
}

func TestRenewBlockedDuringCooldown(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tok, rec := newToken(t)
	rec.QuotaLimiterState = guard.QuotaLimiterState{
		Count:       1,
		NotBefore:   time.Now().Add(time.Minute),
		WindowStart: time.Now(),
	}
	require.NoError(t, store.Add(rec))

	gwc := newGatewayMock(validShareResponse())
	s := newRenewServiceWithLimiter(t, store, gwc, &testPublisher{}, config.QuotaLimiter{
		Cooldown: time.Minute,
		Window:   24 * time.Hour,
		DailyCap: 20,
	})

	_, err := s.Renew(context.Background(), testShareID, tok)
	require.Error(t, err)

	var ge *GuestError
	require.ErrorAs(t, err, &ge)
	assert.ErrorIs(t, ge.ErrorType, ErrTooManyAttempts)
	assert.Greater(t, ge.RetryAfter, time.Duration(0))

	gwc.AssertNotCalled(t, "GetShare", mock.Anything, mock.Anything)
}

func TestRenewDailyCap(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tok, rec := newToken(t)
	rec.QuotaLimiterState = guard.QuotaLimiterState{
		Count:       20,
		NotBefore:   time.Now().Add(time.Minute),
		WindowStart: time.Now(),
	}
	require.NoError(t, store.Add(rec))

	gwc := newGatewayMock(validShareResponse())
	s := newRenewServiceWithLimiter(t, store, gwc, &testPublisher{}, config.QuotaLimiter{
		Cooldown: time.Minute,
		Window:   24 * time.Hour,
		DailyCap: 20,
	})

	_, err := s.Renew(context.Background(), testShareID, tok)
	require.Error(t, err)

	var ge *GuestError
	require.ErrorAs(t, err, &ge)
	assert.ErrorIs(t, ge.ErrorType, ErrTooManyAttempts)
	assert.Greater(t, ge.RetryAfter, time.Duration(0))

	gwc.AssertNotCalled(t, "GetShare", mock.Anything, mock.Anything)
}

func TestRotateGuestTokenBlockedReturnsRetryAfter(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tokStr, rec := newToken(t)
	rec.QuotaLimiterState = guard.QuotaLimiterState{
		Count:       1,
		NotBefore:   time.Now().Add(time.Minute),
		WindowStart: time.Now(),
	}
	require.NoError(t, store.Add(rec))

	s := newRenewServiceWithLimiter(t, store, newGatewayMock(validShareResponse()), &testPublisher{}, config.QuotaLimiter{
		Cooldown: time.Minute,
		Window:   24 * time.Hour,
		DailyCap: 20,
	})

	tok, err := s.tokenSvc.Parse(tokStr)
	require.NoError(t, err)

	_, _, _, err = s.rotateGuestToken(token.Hash(testShareID), tok, "pin-hash", time.Now())
	require.Error(t, err)

	var ge *GuestError
	require.ErrorAs(t, err, &ge)
	assert.ErrorIs(t, ge.ErrorType, ErrTooManyAttempts)
	assert.Greater(t, ge.RetryAfter, time.Duration(0))
}

func TestVerifyPinSuccess(t *testing.T) {
	pinHash, err := pin.Hash("123456")
	require.NoError(t, err)

	rec, nonce := pinRecord(t, pinHash, time.Now().Add(time.Minute), guard.BackoffLimiterState{}, time.Time{})
	store := storage.NewFileManager(t.TempDir())
	require.NoError(t, store.Add(rec))

	s := newAuthGuestService(t, store, newGatewayMock(validShareResponse()))

	result, err := s.VerifyPin(context.Background(), testShareID, "123456", nonce)
	require.NoError(t, err)
	assert.NotEmpty(t, result.SessionToken)
	assert.WithinDuration(t, time.Now().Add(time.Hour), result.ExpiresAt, 2*time.Second)
	assert.Equal(t, testShareID, result.ShareID)

	got, err := store.Get(token.Hash(testShareID))
	require.NoError(t, err)
	assert.Empty(t, got.PinHash)
	assert.True(t, got.PinExpiry.IsZero())
	assert.Empty(t, got.NonceState.Hash)
}

func TestVerifyPinErrors(t *testing.T) {
	pinHash, err := pin.Hash("123456")
	require.NoError(t, err)

	validShare := validShareResponse()
	shareNotFound := &collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_NOT_FOUND},
	}
	shareExpired := &collaboration.GetShareResponse{
		Status: &rpc.Status{Code: rpc.Code_CODE_OK},
		Share:  newGuestShare(time.Now().Add(-time.Hour)),
	}

	freshRecord := func(t *testing.T) (storage.Record, string) {
		return pinRecord(t, pinHash, time.Now().Add(time.Minute), guard.BackoffLimiterState{}, time.Time{})
	}

	tests := []struct {
		name     string
		pin      string
		build    func(t *testing.T) (storage.Record, string)
		missing  bool
		response *collaboration.GetShareResponse
		wantErr  error
	}{
		{
			name: "invalid pin",
			pin:  "000000",
			build: func(t *testing.T) (storage.Record, string) {
				return freshRecord(t)
			},
			response: validShare,
			wantErr:  ErrPinInvalid,
		},
		{
			name: "expired pin",
			pin:  "123456",
			build: func(t *testing.T) (storage.Record, string) {
				return pinRecord(t, pinHash, time.Now().Add(-time.Minute), guard.BackoffLimiterState{}, time.Time{})
			},
			response: validShare,
			wantErr:  ErrPinExpired,
		},
		{
			name: "expired nonce",
			pin:  "123456",
			build: func(t *testing.T) (storage.Record, string) {
				return pinRecord(t, pinHash, time.Now().Add(time.Minute), guard.BackoffLimiterState{}, time.Now().Add(-time.Minute))
			},
			response: validShare,
			wantErr:  ErrNonceExpired,
		},
		{
			name: "no pin set",
			pin:  "123456",
			build: func(t *testing.T) (storage.Record, string) {
				return pinRecord(t, "", time.Time{}, guard.BackoffLimiterState{}, time.Time{})
			},
			response: validShare,
			wantErr:  ErrPinInvalid,
		},
		{
			name:     "record not found",
			pin:      "123456",
			missing:  true,
			response: validShare,
			wantErr:  ErrNonceInvalid,
		},
		{
			name: "share not found",
			pin:  "123456",
			build: func(t *testing.T) (storage.Record, string) {
				return freshRecord(t)
			},
			response: shareNotFound,
			wantErr:  ErrShareNotFound,
		},
		{
			name: "share expired",
			pin:  "123456",
			build: func(t *testing.T) (storage.Record, string) {
				return freshRecord(t)
			},
			response: shareExpired,
			wantErr:  ErrShareExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				rec   storage.Record
				nonce string
			)
			if !tt.missing {
				rec, nonce = tt.build(t)
			}

			store := storage.NewFileManager(t.TempDir())
			if !tt.missing {
				require.NoError(t, store.Add(rec))
			}

			s := newAuthGuestService(t, store, newGatewayMock(tt.response))

			_, err := s.VerifyPin(context.Background(), testShareID, tt.pin, nonce)
			require.Error(t, err)

			var ge *GuestError
			require.ErrorAs(t, err, &ge)
			assert.ErrorIs(t, ge.ErrorType, tt.wantErr)
			assert.Equal(t, testShareID, ge.ShareID)

			if !tt.missing {
				got, err := store.Get(token.Hash(testShareID))
				require.NoError(t, err)
				assert.Equal(t, rec.PinHash, got.PinHash)
				assert.WithinDuration(t, rec.PinExpiry, got.PinExpiry, time.Second)
				assert.Equal(t, rec.Redeemed, got.Redeemed)
			}
		})
	}
}

func TestVerifyPinNonceMismatch(t *testing.T) {
	pinHash, err := pin.Hash("123456")
	require.NoError(t, err)

	tests := []struct {
		name  string
		nonce string
	}{
		{name: "missing nonce", nonce: ""},
		{name: "wrong nonce", nonce: "wrong-nonce"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, _ := pinRecord(t, pinHash, time.Now().Add(time.Minute), guard.BackoffLimiterState{}, time.Time{})

			store := storage.NewFileManager(t.TempDir())
			require.NoError(t, store.Add(rec))

			s := newAuthGuestService(t, store, newGatewayMock(validShareResponse()))

			_, err := s.VerifyPin(context.Background(), testShareID, "123456", tt.nonce)
			require.Error(t, err)

			var ge *GuestError
			require.ErrorAs(t, err, &ge)
			assert.ErrorIs(t, ge.ErrorType, ErrNonceInvalid)

			got, err := store.Get(token.Hash(testShareID))
			require.NoError(t, err)
			assert.Equal(t, rec.PinHash, got.PinHash)
			assert.NotEmpty(t, got.NonceState.Hash)
		})
	}
}

func TestVerifyPinNonceRejectedBeforeGateway(t *testing.T) {
	pinHash, err := pin.Hash("123456")
	require.NoError(t, err)

	tests := []struct {
		name     string
		override string
		expiry   time.Time
		wantErr  error
	}{
		{name: "wrong nonce", override: "wrong-nonce", wantErr: ErrNonceInvalid},
		{name: "expired nonce", expiry: time.Now().Add(-time.Minute), wantErr: ErrNonceExpired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, issued := pinRecord(t, pinHash, time.Now().Add(time.Minute), guard.BackoffLimiterState{}, tt.expiry)

			store := storage.NewFileManager(t.TempDir())
			require.NoError(t, store.Add(rec))

			gwc := newGatewayMock(validShareResponse())
			s := newAuthGuestService(t, store, gwc)

			nonce := tt.override
			if nonce == "" {
				nonce = issued
			}

			_, err := s.VerifyPin(context.Background(), testShareID, "123456", nonce)
			require.Error(t, err)

			var ge *GuestError
			require.ErrorAs(t, err, &ge)
			assert.ErrorIs(t, ge.ErrorType, tt.wantErr)

			gwc.AssertNotCalled(t, "GetShare", mock.Anything, mock.Anything)
			gwc.AssertNotCalled(t, "Authenticate", mock.Anything, mock.Anything)
		})
	}
}

func TestVerifyPinFailedAttemptSetsBackoff(t *testing.T) {
	pinHash, err := pin.Hash("123456")
	require.NoError(t, err)

	rec, nonce := pinRecord(t, pinHash, time.Now().Add(time.Minute), guard.BackoffLimiterState{}, time.Time{})
	store := storage.NewFileManager(t.TempDir())
	require.NoError(t, store.Add(rec))

	s := newAuthGuestService(t, store, newGatewayMock(validShareResponse()))

	_, err = s.VerifyPin(context.Background(), testShareID, "000000", nonce)
	require.Error(t, err)

	var ge *GuestError
	require.ErrorAs(t, err, &ge)
	assert.ErrorIs(t, ge.ErrorType, ErrPinInvalid)

	got, err := store.Get(token.Hash(testShareID))
	require.NoError(t, err)
	assert.Equal(t, 1, got.BackoffLimiterState.Count)
	assert.WithinDuration(t, time.Now().Add(time.Second), got.BackoffLimiterState.NotBefore, time.Second)
	assert.NotEmpty(t, got.PinHash, "PIN must not be invalidated before MaxAttempts")
}

func TestVerifyPinBlockedDuringBackoff(t *testing.T) {
	pinHash, err := pin.Hash("123456")
	require.NoError(t, err)

	rec, nonce := pinRecord(t, pinHash, time.Now().Add(time.Minute), guard.BackoffLimiterState{
		Count:     1,
		NotBefore: time.Now().Add(time.Minute),
	}, time.Time{})

	store := storage.NewFileManager(t.TempDir())
	require.NoError(t, store.Add(rec))

	gwc := newGatewayMock(validShareResponse())
	s := newAuthGuestService(t, store, gwc)

	_, err = s.VerifyPin(context.Background(), testShareID, "123456", nonce)
	require.Error(t, err)

	var ge *GuestError
	require.ErrorAs(t, err, &ge)
	assert.ErrorIs(t, ge.ErrorType, ErrTooManyAttempts)
	assert.Greater(t, ge.RetryAfter, time.Duration(0))

	gwc.AssertNotCalled(t, "GetShare", mock.Anything, mock.Anything)
}

func TestVerifyPinAttemptBlockedReturnsRetryAfter(t *testing.T) {
	rec, nonce := pinRecord(t, "", time.Now().Add(time.Minute), guard.BackoffLimiterState{
		Count:     1,
		NotBefore: time.Now().Add(time.Minute),
	}, time.Time{})
	s := newAuthGuestService(t, storage.NewFileManager(t.TempDir()), newGatewayMock(validShareResponse()))

	ok, err := s.verifyPinAttempt(&rec, "123456", nonce)
	require.False(t, ok)

	var ge *GuestError
	require.ErrorAs(t, err, &ge)
	assert.ErrorIs(t, ge.ErrorType, ErrTooManyAttempts)
	assert.Greater(t, ge.RetryAfter, time.Duration(0))
}

func TestVerifyPinInvalidatedAfterMaxAttempts(t *testing.T) {
	pinHash, err := pin.Hash("123456")
	require.NoError(t, err)

	rec, nonce := pinRecord(t, pinHash, time.Now().Add(time.Minute), guard.BackoffLimiterState{
		Count:     4,
		NotBefore: time.Now().Add(-time.Second),
	}, time.Time{})

	store := storage.NewFileManager(t.TempDir())
	require.NoError(t, store.Add(rec))

	s := newAuthGuestService(t, store, newGatewayMock(validShareResponse()))

	_, err = s.VerifyPin(context.Background(), testShareID, "000000", nonce)
	require.Error(t, err)

	var ge *GuestError
	require.ErrorAs(t, err, &ge)
	assert.ErrorIs(t, ge.ErrorType, ErrPinInvalid)

	got, err := store.Get(token.Hash(testShareID))
	require.NoError(t, err)
	assert.Empty(t, got.PinHash)
	assert.Empty(t, got.NonceState.Hash)

	// The cleared nonce rejects the next attempt.
	_, err = s.VerifyPin(context.Background(), testShareID, "123456", nonce)
	require.Error(t, err)

	var ge2 *GuestError
	require.ErrorAs(t, err, &ge2)
	assert.ErrorIs(t, ge2.ErrorType, ErrNonceInvalid)
}

func TestRenewResetsPinAttempts(t *testing.T) {
	store := storage.NewFileManager(t.TempDir())
	tok, rec := newToken(t)
	rec.BackoffLimiterState = guard.BackoffLimiterState{Count: 3, NotBefore: time.Now().Add(time.Minute)}
	require.NoError(t, store.Add(rec))

	s := newRenewService(t, store, newGatewayMock(validShareResponse()), &testPublisher{})

	_, err := s.Renew(context.Background(), testShareID, tok)
	require.NoError(t, err)

	got, err := store.Get(token.Hash(testShareID))
	require.NoError(t, err)
	assert.Zero(t, got.BackoffLimiterState.Count)
	assert.True(t, got.BackoffLimiterState.NotBefore.IsZero())
}

func TestVerifyPinSingleUseConcurrent(t *testing.T) {
	pinHash, err := pin.Hash("123456")
	require.NoError(t, err)

	rec, nonce := pinRecord(t, pinHash, time.Now().Add(time.Minute), guard.BackoffLimiterState{}, time.Time{})
	store := storage.NewFileManager(t.TempDir())
	require.NoError(t, store.Add(rec))

	s := newAuthGuestService(t, store, newGatewayMock(validShareResponse()))

	const workers = 10
	var (
		wg      sync.WaitGroup
		success atomic.Int32
	)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.VerifyPin(context.Background(), testShareID, "123456", nonce); err == nil {
				success.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), success.Load())
}
