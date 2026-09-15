package utils

import (
	"encoding/base64"
	"strings"

	grouppb "github.com/cs3org/go-cs3apis/cs3/identity/group/v1beta1"
	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"google.golang.org/protobuf/proto"
)

type FilenameEncoder interface {
	SafeFilename() string
}

type FSSafeUserID struct {
	id *userpb.UserId
}

func NewFSSafeUserID(id *userpb.UserId) FSSafeUserID {
	return FSSafeUserID{id: id}
}

func (id FSSafeUserID) SafeFilename() string {
	opaqueID := id.id.GetOpaqueId()
	if id.id.GetType() == userpb.UserType_USER_TYPE_GUEST {
		// Guest opaqueID is an email address, which is why we choose to make
		// it lowercase: RFC 5321 does specify that email address local-parts
		// are case sensitive but, in practice, it's a de-facto standard that
		// email providers consider them to be case insensitive:
		return base64.RawURLEncoding.EncodeToString([]byte(strings.ToLower(opaqueID)))
	}
	return opaqueID
}

// Decode returns a copy of the wrapped user ID with the filename restored as its opaque ID.
// The decoding decision is done based on the Type attribute of the Receiver id
func (id FSSafeUserID) Decode(filename string) (*userpb.UserId, error) {
	opaqueID := filename
	if id.id.GetType() == userpb.UserType_USER_TYPE_GUEST {
		decoded, err := base64.RawURLEncoding.DecodeString(filename)
		if err != nil {
			return nil, err
		}
		opaqueID = string(decoded)
	}

	decodedID := &userpb.UserId{}
	if id.id != nil {
		decodedID = proto.Clone(id.id).(*userpb.UserId)
	}
	decodedID.OpaqueId = opaqueID
	return decodedID, nil
}

type FSSafeGroupID struct {
	ID *grouppb.GroupId
}

func (id FSSafeGroupID) SafeFilename() string {
	return id.ID.GetOpaqueId()
}
