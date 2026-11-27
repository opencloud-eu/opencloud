package staticroutes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/golang-jwt/jwt/v5"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/oidc"
	oidcmocks "github.com/opencloud-eu/opencloud/pkg/oidc/mocks"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	bcl "github.com/opencloud-eu/opencloud/services/proxy/pkg/staticroutes/backchannellogout"
	backendmocks "github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend/mocks"
	"github.com/opencloud-eu/reva/v3/pkg/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
	microevents "go-micro.dev/v4/events"
	microstore "go-micro.dev/v4/store"
)

type recordingEmitter struct{ events []audit.Event }

func (r *recordingEmitter) Emit(_ context.Context, ev audit.Event) error {
	r.events = append(r.events, ev)
	return nil
}

type nopPublisher struct{}

func (nopPublisher) Publish(string, interface{}, ...microevents.PublishOption) error { return nil }

func logoutRequest() *http.Request {
	form := url.Values{"logout_token": {"token"}}
	r := httptest.NewRequest(http.MethodPost, "/backchannel_logout", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestBackchannelLogoutIsAudited(t *testing.T) {
	store := microstore.NewMemoryStore()
	key, err := bcl.NewKey("subject", "s-1")
	require.NoError(t, err)
	claims, err := msgpack.Marshal(bcl.TokenRecord{Claims: map[string]any{"preferred_username": "alice"}})
	require.NoError(t, err)
	require.NoError(t, store.Write(&microstore.Record{Key: key, Value: []byte("claims-key")}))
	require.NoError(t, store.Write(&microstore.Record{Key: "claims-key", Value: claims}))

	oidcClient := oidcmocks.NewOIDCClient(t)
	oidcClient.EXPECT().VerifyLogoutToken(mock.Anything, "token").Return(&oidc.LogoutToken{
		RegisteredClaims: jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(time.Now())},
		SessionId:        "s-1",
	}, nil)
	users := backendmocks.NewUserBackend(t)
	users.On("GetUserByClaims", mock.Anything, "username", "alice").Return(&userpb.User{
		Id:       &userpb.UserId{OpaqueId: "u-1", TenantId: "t-1"},
		Username: "alice",
	}, "", nil)

	emitter := &recordingEmitter{}
	s := &StaticRouteHandler{
		Logger:          log.NopLogger(),
		Config:          config.Config{UserOIDCClaim: "preferred_username", UserCS3Claim: "username"},
		OidcClient:      oidcClient,
		UserInfoCache:   store,
		LogoutMarkers:   bcl.NewLogoutMarkers(microstore.NewMemoryStore(), time.Hour),
		UserProvider:    users,
		EventsPublisher: nopPublisher{},
		AuditEmitter:    emitter,
	}

	w := httptest.NewRecorder()
	s.backchannelLogout(w, logoutRequest())

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, emitter.events, 1)
	ev := emitter.events[0]
	assert.Equal(t, audit.AuthBackchannelLogout, ev.Definition)
	assert.Equal(t, audit.OutcomeSuccess, ev.Outcome)
	assert.Equal(t, "u-1", ev.Target.ID)
	assert.Equal(t, "t-1", ev.Tenant)
}

// undeletableStore cannot remove records.
type undeletableStore struct{ microstore.Store }

func (undeletableStore) Delete(string, ...microstore.DeleteOption) error {
	return errors.New("delete failed")
}

// The marker has ended the session, so the logout is audited even if the cached
// claims cannot be removed.
func TestBackchannelLogoutIsAuditedWhenTheCleanupFails(t *testing.T) {
	store := undeletableStore{microstore.NewMemoryStore()}
	key, err := bcl.NewKey("subject", "s-1")
	require.NoError(t, err)
	claims, err := msgpack.Marshal(bcl.TokenRecord{Claims: map[string]any{"preferred_username": "alice"}})
	require.NoError(t, err)
	require.NoError(t, store.Write(&microstore.Record{Key: key, Value: []byte("claims-key")}))
	require.NoError(t, store.Write(&microstore.Record{Key: "claims-key", Value: claims}))

	oidcClient := oidcmocks.NewOIDCClient(t)
	oidcClient.EXPECT().VerifyLogoutToken(mock.Anything, "token").Return(&oidc.LogoutToken{
		RegisteredClaims: jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(time.Now())},
		SessionId:        "s-1",
	}, nil)
	users := backendmocks.NewUserBackend(t)
	users.On("GetUserByClaims", mock.Anything, "username", "alice").Return(&userpb.User{
		Id: &userpb.UserId{OpaqueId: "u-1"},
	}, "", nil)

	emitter := &recordingEmitter{}
	s := &StaticRouteHandler{
		Logger:          log.NopLogger(),
		Config:          config.Config{UserOIDCClaim: "preferred_username", UserCS3Claim: "username"},
		OidcClient:      oidcClient,
		UserInfoCache:   store,
		LogoutMarkers:   bcl.NewLogoutMarkers(microstore.NewMemoryStore(), time.Hour),
		UserProvider:    users,
		EventsPublisher: nopPublisher{},
		AuditEmitter:    emitter,
	}

	w := httptest.NewRecorder()
	s.backchannelLogout(w, logoutRequest())

	assert.Equal(t, http.StatusOK, w.Code)
	_, err = store.Read("claims-key")
	require.NoError(t, err, "the cached claims are still there")
	require.Len(t, emitter.events, 1)
	assert.Equal(t, audit.OutcomeSuccess, emitter.events[0].Outcome)
	assert.Equal(t, "u-1", emitter.events[0].Target.ID)
}

func TestInvalidLogoutTokenIsAudited(t *testing.T) {
	oidcClient := oidcmocks.NewOIDCClient(t)
	oidcClient.EXPECT().VerifyLogoutToken(mock.Anything, "token").Return(nil, errors.New("bad signature"))

	emitter := &recordingEmitter{}
	s := &StaticRouteHandler{
		Logger:        log.NopLogger(),
		OidcClient:    oidcClient,
		UserInfoCache: microstore.NewMemoryStore(),
		AuditEmitter:  emitter,
	}

	w := httptest.NewRecorder()
	s.backchannelLogout(w, logoutRequest())

	assert.Equal(t, http.StatusBadRequest, w.Code)
	require.Len(t, emitter.events, 1)
	assert.Equal(t, audit.OutcomeFailure, emitter.events[0].Outcome)
	assert.Equal(t, "invalid_token", emitter.events[0].Reason)
}
