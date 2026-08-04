package data

import "time"

// RawActivity represents an activity as it is stored in the activitylog store
type RawActivity struct {
	EventID   string    `json:"event_id"`
	Depth     int       `json:"depth"`
	Timestamp time.Time `json:"timestamp"`
}
