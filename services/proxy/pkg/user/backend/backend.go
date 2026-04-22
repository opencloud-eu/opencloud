package backend

import (
	"context"
	"errors"

	cs3 "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
)

var (
	// ErrAccountNotFound account not found
	ErrAccountNotFound = errors.New("user not found")
	// ErrAccountDisabled account disabled
	ErrAccountDisabled = errors.New("account disabled")
	// ErrNotSupported operation not supported by user-backend
	ErrNotSupported = errors.New("operation not supported")
)

// UserBackend allows the proxy to retrieve users from different user-backends (accounts-service, CS3)
type UserBackend interface {
	GetUserByClaims(ctx context.Context, claim, value string) (*cs3.User, string, error)
	Authenticate(ctx context.Context, username string, password string) (*cs3.User, string, error)
	CreateUserFromClaims(ctx context.Context, claims map[string]any) (*cs3.User, error)
	UpdateUserIfNeeded(ctx context.Context, user *cs3.User, claims map[string]any) error
	SyncGroupMemberships(ctx context.Context, user *cs3.User, claims map[string]any) error
}
