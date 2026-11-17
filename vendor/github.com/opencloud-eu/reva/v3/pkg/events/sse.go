package events

import (
	"encoding/json"
)

// SendSSE instructs the sse service to send one or multiple notifications
type SendSSE struct {
	UserIDs []string
	Type    string
	Message []byte
}

// Unmarshal to fulfill umarshaller interface
func (SendSSE) Unmarshal(v []byte) (interface{}, error) {
	e := SendSSE{}
	err := json.Unmarshal(v, &e)
	return e, err
}
