package jmap

import "time"

type Email struct {
	From           string
	Subject        string
	HasAttachments bool
	Received       time.Time
}

func NewEmail(elem map[string]any) Email {
	fromList := elem["from"].([]any)
	from := fromList[0].(map[string]any)
	var subject string
	var value any = elem["subject"]
	if value != nil {
		subject = value.(string)
	} else {
		subject = ""
	}
	var hasAttachments bool
	hasAttachmentsAny := elem["hasAttachments"]
	if hasAttachmentsAny != nil {
		hasAttachments = hasAttachmentsAny.(bool)
	} else {
		hasAttachments = false
	}

	received, receivedErr := time.ParseInLocation(time.RFC3339, elem["receivedAt"].(string), time.UTC)
	if receivedErr != nil {
		panic(receivedErr)
	}

	return Email{
		From:           from["email"].(string),
		Subject:        subject,
		HasAttachments: hasAttachments,
		Received:       received,
	}
}
