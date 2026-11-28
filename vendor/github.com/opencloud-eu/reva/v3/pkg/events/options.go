package events

import (
	"time"
)

// ConsumeOptions contains all the options which can be provided when consuming events
type ConsumeOptions struct {
	Group   string
	AutoAck bool
	AckWait time.Duration
}
