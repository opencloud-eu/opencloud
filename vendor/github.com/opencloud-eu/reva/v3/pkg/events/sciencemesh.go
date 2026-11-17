package events

import (
	"encoding/json"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	types "github.com/cs3org/go-cs3apis/cs3/types/v1beta1"
)

// ScienceMeshInviteTokenGenerated is emitted when a sciencemesh token is generated
type ScienceMeshInviteTokenGenerated struct {
	Sharer        *user.UserId
	RecipientMail string
	Token         string
	Description   string
	Expiration    uint64
	InviteLink    string
	Timestamp     *types.Timestamp
}

// Unmarshal to fulfill unmarshaller interface
func (ScienceMeshInviteTokenGenerated) Unmarshal(v []byte) (interface{}, error) {
	e := ScienceMeshInviteTokenGenerated{}
	err := json.Unmarshal(v, &e)
	return e, err
}
