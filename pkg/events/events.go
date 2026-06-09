package events

import (
	"encoding/json"
	"time"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
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
