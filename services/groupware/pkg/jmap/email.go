package jmap

import "time"

type Email struct {
	From           string
	Subject        string
	HasAttachments bool
	Received       time.Time
}
