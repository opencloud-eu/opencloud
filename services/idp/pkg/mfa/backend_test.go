package mfa

import (
	"context"
	"errors"
	"testing"

	"github.com/libregraph/lico/identifier/backends"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	microstore "go-micro.dev/v4/store"
)

// fakeUser is a minimal backends.UserFromBackend.
type fakeUser struct{ id, username string }

func (u fakeUser) Subject() string               { return u.id }
func (u fakeUser) Username() string              { return u.username }
func (u fakeUser) BackendClaims() map[string]any { return nil }
func (u fakeUser) BackendScopes() []string       { return nil }
func (u fakeUser) RequiredScopes() []string      { return nil }

// fakeBackend knows the users alan and mary, both with the password "secret".
type fakeBackend struct {
	backends.Backend
	err               error
	destroyedSessions []string
}

func (b *fakeBackend) Logon(_ context.Context, _, username, password string) (bool, *string, *string, backends.UserFromBackend, error) {
	if b.err != nil {
		return false, nil, nil, nil, b.err
	}
	for _, u := range []User{alan, mary} {
		if u.Username == username && password == "secret" {
			id := u.ID
			session := "session-of-" + username
			return true, &id, &session, fakeUser{id: u.ID, username: u.Username}, nil
		}
	}
	return false, nil, nil, nil, nil
}

func (b *fakeBackend) DestroySession(_ context.Context, sessionRef *string) error {
	b.destroyedSessions = append(b.destroyedSessions, *sessionRef)
	return nil
}

func logon(b backends.Backend, username, password string, state *loginState) (bool, *string, error) {
	ctx := context.Background()
	if state != nil {
		ctx = withLoginState(ctx, state)
	}
	ok, id, session, user, err := b.Logon(ctx, "", username, password)
	if !ok && (id != nil || session != nil || user != nil) {
		panic("a rejected logon must not return a user")
	}
	return ok, id, err
}

func TestBackendLogon(t *testing.T) {
	svc, c, _ := newTestService(t)
	inner := &fakeBackend{}
	wrapped := svc.WrapBackend(inner)
	secret, codes := enroll(t, svc, c, alan)

	t.Run("users without second factor log on with the password", func(t *testing.T) {
		state := &loginState{}
		ok, id, err := logon(wrapped, "mary", "secret", state)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, mary.ID, *id)
		assert.Equal(t, LoginNotEnrolled, state.result)

		// also without the middleware in front
		ok, _, err = logon(wrapped, "mary", "secret", nil)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("a wrong password is rejected before the second factor is looked at", func(t *testing.T) {
		state := &loginState{code: codeFor(t, secret, c.now())}
		ok, _, err := logon(wrapped, "alan", "wrong", state)
		require.NoError(t, err)
		assert.False(t, ok)
		// nothing is revealed about the second factor
		assert.Equal(t, LoginNotEnrolled, state.result)

		// the code was not consumed
		state = &loginState{code: codeFor(t, secret, c.now())}
		ok, _, err = logon(wrapped, "alan", "secret", state)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, LoginOK, state.result)
	})

	t.Run("the password alone is not sufficient", func(t *testing.T) {
		inner.destroyedSessions = nil
		state := &loginState{}
		ok, _, err := logon(wrapped, "alan", "secret", state)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, LoginCodeRequired, state.result)
		assert.Equal(t, []string{"session-of-alan"}, inner.destroyedSessions, "the backend session must not survive")
	})

	t.Run("fails closed without the middleware in front", func(t *testing.T) {
		ok, _, err := logon(wrapped, "alan", "secret", nil)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("a wrong code is rejected", func(t *testing.T) {
		state := &loginState{code: "000000"}
		ok, _, err := logon(wrapped, "alan", "secret", state)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, LoginCodeInvalid, state.result)
	})

	t.Run("a recovery code is accepted", func(t *testing.T) {
		state := &loginState{code: codes[0]}
		ok, id, err := logon(wrapped, "alan", "secret", state)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, alan.ID, *id)
	})

	t.Run("errors of the backend are passed on", func(t *testing.T) {
		inner.err = errors.New("ldap is down")
		defer func() { inner.err = nil }()
		ok, _, err := logon(wrapped, "alan", "secret", &loginState{})
		assert.Error(t, err)
		assert.False(t, ok)
	})
}

// failingStore fails all reads, like a store that is not reachable.
type failingStore struct{ microstore.Store }

func (failingStore) Read(string, ...microstore.ReadOption) ([]*microstore.Record, error) {
	return nil, errors.New("store is down")
}

func TestBackendLogonFailsClosedWhenTheStoreIsDown(t *testing.T) {
	svc, _, st := newTestService(t)
	svc.store = failingStore{Store: st}
	wrapped := svc.WrapBackend(&fakeBackend{})

	// mary has no second factor, but that cannot be known without the store
	state := &loginState{}
	ok, _, err := logon(wrapped, "mary", "secret", state)
	assert.Error(t, err)
	assert.False(t, ok)
	assert.False(t, state.result.Passed())
}

func TestVerifyPassword(t *testing.T) {
	svc, _, _ := newTestService(t)

	_, err := svc.verifyPassword(context.Background(), alan, "secret")
	assert.Error(t, err, "needs a wrapped backend")

	inner := &fakeBackend{}
	svc.WrapBackend(inner)

	ok, err := svc.verifyPassword(context.Background(), alan, "secret")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, []string{"session-of-alan"}, inner.destroyedSessions)

	for _, password := range []string{"", "wrong"} {
		ok, err = svc.verifyPassword(context.Background(), alan, password)
		require.NoError(t, err)
		assert.False(t, ok)
	}

	// the password of another account with the same username does not count
	ok, err = svc.verifyPassword(context.Background(), User{ID: "another-id", Username: "alan"}, "secret")
	require.NoError(t, err)
	assert.False(t, ok)
}
