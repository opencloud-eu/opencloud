package userstate

import (
	"fmt"
	"time"
)

const (
	_ = iota
	UserStateUnspecified
	UserStateEnabled
	UserStateDisabled
	UserStateSoftDeleted
	UserStateHardDeleted
)

// UserState represents the state of a user account.
// Note: This does not reflect state changes, these need to be red from the audit logs.
type UserState struct {
	UserId          string        `json:"userid"`
	State           uint8         `json:"state"`
	TimeStamp       time.Time     `json:"timestamp,omitempty"`
	RetentionPeriod time.Duration `json:"retentionPeriod,omitempty"`
	Reason          string        `json:"reason,omitempty,omitempty"`
}

func IsValidUserState(us *UserState) (bool, error) {
	if us.State == UserStateSoftDeleted {
		if us.RetentionPeriod <= 0 {
			return false, fmt.Errorf("retention period must be greater than 0 for soft deleted users")
		}
		if us.Reason == "" {
			return false, fmt.Errorf("reason must be provided for soft deleted users")
		}
	}
	return true, nil
}
