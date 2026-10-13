package events

import (
	"encoding/json"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	collaboration "github.com/cs3org/go-cs3apis/cs3/sharing/collaboration/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
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

type GuestTokenCreated struct {
	ShareID      *collaboration.ShareId
	Sharer       *user.UserId
	GranteeEmail string
	ItemID       *provider.ResourceId
	ResourceName string
	Token        string
	Timestamp    time.Time
}

func (GuestTokenCreated) Unmarshal(v []byte) (interface{}, error) {
	e := GuestTokenCreated{}
	err := json.Unmarshal(v, &e)
	return e, err
}

type GuestTokenRenewed struct {
	ShareID      *collaboration.ShareId
	Sharer       *user.UserId
	GranteeEmail string
	ItemID       *provider.ResourceId
	Token        string
	Pin          string
	Timestamp    time.Time
}

func (GuestTokenRenewed) Unmarshal(v []byte) (interface{}, error) {
	e := GuestTokenRenewed{}
	err := json.Unmarshal(v, &e)
	return e, err
}
