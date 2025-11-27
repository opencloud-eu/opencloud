package user

import (
	"fmt"
	"net/url"
	"strings"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
)

// LocalUserFederatedID creates a federated id for local users by
// 1. stripping the protocol from the domain and
// 2. if the domain is different from the idp, add the idp to the opaque id
func LocalUserFederatedID(id *userpb.UserId, domain string) *userpb.UserId {
	if u, err := url.Parse(domain); err == nil && u.Host != "" {
		domain = u.Host
	}
	u := &userpb.UserId{
		Type:     userpb.UserType_USER_TYPE_FEDERATED,
		Idp:      id.Idp,
		OpaqueId: id.OpaqueId,
	}

	if id.Idp != "" && domain != "" && id.Idp != domain {
		u.OpaqueId = id.OpaqueId + "@" + id.Idp
		u.Idp = domain
	}
	return u
}

// DecodeRemoteUserFederatedID decodes opaque id into remote user's federated id by
// splitting the opaque id at the last @ to get the opaque id and the domain
func DecodeRemoteUserFederatedID(id *userpb.UserId) *userpb.UserId {
	remoteId := &userpb.UserId{
		Type:     userpb.UserType_USER_TYPE_PRIMARY,
		Idp:      id.Idp,
		OpaqueId: id.OpaqueId,
	}
	remote := id.OpaqueId
	last := strings.LastIndex(remote, "@")
	if last == -1 {
		return remoteId
	}
	remoteId.OpaqueId = remote[:last]
	remoteId.Idp = remote[last+1:]

	return remoteId
}

// FormatOCMUser formats a user id in the form of <opaque-id>@<idp> used by the OCM API in shareWith, owner and creator fields
func FormatOCMUser(u *userpb.UserId) string {
	if u.Idp == "" {
		return u.OpaqueId
	}
	return fmt.Sprintf("%s@%s", u.OpaqueId, u.Idp)
}
