package cs3

import (
	cs3user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	konnect "github.com/libregraph/lico"
)

type cs3User struct {
	u *cs3user.User
}

func newCS3User(u *cs3user.User) (*cs3User, error) {
	return &cs3User{
		u: u,
	}, nil
}

// Subject returns the cs3 users opaque id as sub
func (u *cs3User) Subject() string {
	return u.u.GetId().GetOpaqueId()
}

// Email returns the cs3 users email
func (u *cs3User) Email() string {
	return u.u.GetMail()
}

// EmailVerified returns the cs3 users email verified flag
func (u *cs3User) EmailVerified() bool {
	return u.u.GetMailVerified()
}

// Name returns the cs3 users displayname
func (u *cs3User) Name() string {
	return u.u.GetDisplayName()
}

// FamilyName always returns "" to fulfill the UserWithProfile interface
func (u *cs3User) FamilyName() string {
	return ""
}

// GivenName always returns "" to fulfill the UserWithProfile interface
func (u *cs3User) GivenName() string {
	return ""
}

// Username returns the cs3 users username
func (u *cs3User) Username() string {
	return u.u.GetUsername()
}

// UniqueID returns the cs3 users opaque id
func (u *cs3User) UniqueID() string {
	return u.u.GetId().GetOpaqueId()
}

// BackendClaims returns additional claims the cs3 users provides
func (u *cs3User) BackendClaims() map[string]any {
	claims := make(map[string]any)
	claims[konnect.IdentifiedUserIDClaim] = u.u.GetId().GetOpaqueId()

	return claims
}

// BackendScopes returns nil to fulfill the UserFromBackend interface
func (u *cs3User) BackendScopes() []string {
	return nil
}

// RequiredScopes returns nil to fulfill the UserFromBackend interface
func (u *cs3User) RequiredScopes() []string {
	return nil
}
