package events

import (
	"encoding/json"
)

// SendEmailsEvent instructs the notification service to send grouped emails
type SendEmailsEvent struct {
	Interval string
}

// Unmarshal to fulfill umarshaller interface
func (SendEmailsEvent) Unmarshal(v []byte) (interface{}, error) {
	e := SendEmailsEvent{}
	err := json.Unmarshal(v, &e)
	return e, err
}
