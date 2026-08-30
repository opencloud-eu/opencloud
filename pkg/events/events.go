package events

import (
	"encoding/json"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

type ResourceMention struct {
	Executant *user.UserId
	UserIDs   []*user.UserId
	Ref       *provider.Reference
	Timestamp time.Time
}

func (ResourceMention) Unmarshal(v []byte) (interface{}, error) {
	e := ResourceMention{}
	err := json.Unmarshal(v, &e)
	return e, err
}

type ProfilePictureSyncRequested struct {
	Executant  *user.UserId
	PictureURL string `json:",omitempty"`
	Timestamp  *types.Timestamp
}

func (ProfilePictureSyncRequested) Unmarshal(v []byte) (interface{}, error) {
	e := ProfilePictureSyncRequested{}
	err := json.Unmarshal(v, &e)
	return e, err
}

// UserProfilePictureUpdated  can be consumed by frontend-facing services to refresh the avatar without a page reload.
type UserProfilePictureUpdated struct {
	Executant *user.UserId
	Timestamp *types.Timestamp
}

func (UserProfilePictureUpdated) Unmarshal(v []byte) (interface{}, error) {
	e := UserProfilePictureUpdated{}
	err := json.Unmarshal(v, &e)
	return e, err
}
