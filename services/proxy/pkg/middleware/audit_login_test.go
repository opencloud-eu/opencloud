package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/golang-jwt/jwt/v5"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/router"
	"github.com/opencloud-eu/reva/v2/pkg/audit"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingEmitter keeps the records the emitter would publish.
type recordingEmitter struct {
	mu      sync.Mutex
	records []*audit.Record
	builder *audit.NatsEmitter
}

func newRecordingEmitter() *recordingEmitter {
	return &recordingEmitter{builder: audit.NewEmitter(nil, audit.Options{Service: audit.Service{Name: "proxy"}, Mode: audit.ModeStrict})}
}

func (e *recordingEmitter) Emit(ctx context.Context, ev audit.Event) error {
	rec, err := e.builder.Record(ctx, ev)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.records = append(e.records, rec)
	return nil
}

// fakeAuthenticator rejects every request with the given auth method.
type fakeAuthenticator struct {
	method string
	result AuthenticationResult
}

func (f fakeAuthenticator) Authenticate(*http.Request) AuthenticationResult { return f.result }
func (f fakeAuthenticator) AuthMethod() string                              { return f.method }

func TestFailedLoginIsAuditedOncePerRequest(t *testing.T) {
	emitter := newRecordingEmitter()
	auths := []Authenticator{
		fakeAuthenticator{method: AuthMethodBasic, result: Failed()},
		fakeAuthenticator{method: AuthMethodAppToken, result: Failed()},
	}
	h := Authentication(auths, AuditEmitter(emitter))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("must not be reached")
	}))

	r := httptest.NewRequest("PROPFIND", "/remote.php/dav/files/alice", nil)
	r.RemoteAddr = "203.0.113.7"
	r.SetBasicAuth("alice", "wrong")
	r = r.WithContext(router.SetRoutingInfo(r.Context(), router.RoutingInfo{}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	require.Len(t, emitter.records, 1)
	rec := emitter.records[0]
	assert.Equal(t, "opencloud.auth.login", rec.EventName)
	assert.Equal(t, "A001F", rec.EventCode)
	assert.Equal(t, audit.OutcomeFailure, rec.Outcome)
	assert.Equal(t, "invalid_credentials", rec.OutcomeReason)
	assert.Equal(t, audit.ActorUser, rec.ActorType)
	assert.Equal(t, "alice", rec.ActorName)
	assert.Equal(t, "203.0.113.7", rec.ClientAddress)
	assert.Equal(t, AuthMethodBasic, rec.AuthMethod)
}

func TestLoginFailures(t *testing.T) {
	for _, tc := range []struct {
		name    string
		method  string
		err     error
		audited bool
		reason  string
		actor   audit.ActorType
	}{
		{name: "invalid OIDC token", method: AuthMethodOIDC, err: errors.New("signature is invalid"), audited: true, reason: "invalid_token", actor: audit.ActorAnonymous},
		{name: "expired OIDC token", method: AuthMethodOIDC, err: jwt.ErrTokenExpired},
		{name: "public link", method: AuthMethodPublicLink},
		{name: "signed URL", method: AuthMethodSignedURL, audited: true, reason: "invalid_signature", actor: audit.ActorAnonymous},
		{name: "guest link", method: AuthMethodGuestLink, audited: true, reason: "invalid_credentials", actor: audit.ActorGuest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			emitter := newRecordingEmitter()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			auditLoginFailure(r, emitter, fakeAuthenticator{method: tc.method}, FailedWithErr(tc.err))

			if !tc.audited {
				assert.Empty(t, emitter.records)
				return
			}
			require.Len(t, emitter.records, 1)
			assert.Equal(t, tc.reason, emitter.records[0].OutcomeReason)
			assert.Equal(t, tc.actor, emitter.records[0].ActorType)
			assert.Equal(t, tc.method, emitter.records[0].AuthMethod)
		})
	}
}

func TestLoginSuccess(t *testing.T) {
	emitter := newRecordingEmitter()
	u := &userpb.User{Id: &userpb.UserId{OpaqueId: "u-1", TenantId: "t-1"}, Username: "alice", DisplayName: "Alice"}
	ctx := audit.ContextSetRequest(context.Background(), audit.RequestContext{ClientAddress: "203.0.113.7", AuthMethod: AuthMethodOIDC, SessionID: "s-1"})

	auditLogin(revactx.ContextSetUser(ctx, u), emitter, u)

	require.Len(t, emitter.records, 1)
	rec := emitter.records[0]
	assert.Equal(t, "A001S", rec.EventCode)
	assert.Equal(t, "u-1", rec.ActorID)
	assert.Equal(t, "t-1", rec.TenantID)
	assert.Equal(t, "u-1", rec.TargetID)
	assert.Equal(t, "s-1", rec.SessionID)
}
