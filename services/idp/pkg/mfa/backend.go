package mfa

import (
	"context"
	"errors"
	"fmt"

	"github.com/libregraph/lico/identifier/backends"
)

// passwordVerifier checks the password of a user without the second factor.
type passwordVerifier func(ctx context.Context, username, password string) (userID string, ok bool, err error)

// loginState carries the one-time code of a logon request to the backend and the result
// of the second factor check back to the HTTP layer.
type loginState struct {
	code   string
	result LoginResult
}

type loginStateKey struct{}

func withLoginState(ctx context.Context, state *loginState) context.Context {
	return context.WithValue(ctx, loginStateKey{}, state)
}

func loginStateFromContext(ctx context.Context) *loginState {
	state, _ := ctx.Value(loginStateKey{}).(*loginState)
	return state
}

// backend wraps an identifier backend and requires the second factor on logon.
type backend struct {
	backends.Backend
	svc *Service
}

// WrapBackend returns an identifier backend that only lets users with two-factor
// authentication log on with a valid code. The check happens in the backend, after the
// password was verified, so no code path of the identifier can skip it. The code is
// expected in the context, see Handler.LogonMiddleware. Without one in the context the
// logon of users with two-factor authentication fails.
func (s *Service) WrapBackend(b backends.Backend) backends.Backend {
	s.passwordVerifier = func(ctx context.Context, username, password string) (string, bool, error) {
		ok, userID, sessionRef, _, err := b.Logon(ctx, "", username, password)
		if err != nil || !ok || userID == nil {
			return "", false, err
		}
		if sessionRef != nil {
			// only the password was checked, the session is not needed
			_ = b.DestroySession(ctx, sessionRef)
		}
		return *userID, true, nil
	}

	return &backend{Backend: b, svc: s}
}

// verifyPassword checks the password of the logged in user with the identifier backend.
func (s *Service) verifyPassword(ctx context.Context, user User, password string) (bool, error) {
	if s.passwordVerifier == nil {
		return false, errors.New("mfa: the identifier backend is not wrapped")
	}
	if password == "" {
		return false, nil
	}

	userID, ok, err := s.passwordVerifier(ctx, user.Username, password)
	if err != nil {
		return false, err
	}
	// make sure the password belongs to the logged in user
	return ok && userID == user.ID, nil
}

// Logon implements the Backend interface. It delegates the password check to the wrapped
// backend and checks the second factor afterwards.
func (b *backend) Logon(ctx context.Context, audience, username, password string) (bool, *string, *string, backends.UserFromBackend, error) {
	ok, userID, sessionRef, user, err := b.Backend.Logon(ctx, audience, username, password)
	if err != nil || !ok || userID == nil || user == nil {
		return ok, userID, sessionRef, user, err
	}

	state := loginStateFromContext(ctx)
	code := ""
	if state != nil {
		code = state.code
	}

	result, err := b.svc.VerifyLogin(User{ID: *userID, Username: user.Username()}, code)
	if state != nil {
		state.result = result
	}
	if err == nil && result.Passed() {
		return ok, userID, sessionRef, user, nil
	}

	if sessionRef != nil {
		_ = b.Backend.DestroySession(ctx, sessionRef)
	}
	if err != nil {
		// fail closed, the logon is rejected when the second factor cannot be checked
		return false, nil, nil, nil, fmt.Errorf("two-factor authentication check failed: %w", err)
	}
	return false, nil, nil, nil, nil
}
