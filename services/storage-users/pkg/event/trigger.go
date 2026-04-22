package event

import (
	"encoding/json"
	"time"

	apiUser "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
)

// PurgeTrashBin wraps all needed information to purge a trash-bin
type PurgeTrashBin struct {
	ExecutantID   *apiUser.UserId
	ExecutionTime time.Time
}

// Unmarshal to fulfill umarshaller interface
func (PurgeTrashBin) Unmarshal(v []byte) (any, error) {
	e := PurgeTrashBin{}
	err := json.Unmarshal(v, &e)
	return e, err
}
