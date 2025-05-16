package svc

import (
	"strings"

	libregraph "github.com/opencloud-eu/libre-graph-api-go"
)

// lessSpacesByLastModifiedDateTime reports whether the element i
// must sort before the element j.
func lessSpacesByLastModifiedDateTime(i, j *libregraph.Drive) bool {
	// compare the items when both dates are set
	if i.LastModifiedDateTime != nil && j.LastModifiedDateTime != nil {
		return i.LastModifiedDateTime.Before(*j.LastModifiedDateTime)
	}
	// an item without a timestamp is considered "less than" an item with a timestamp
	if i.LastModifiedDateTime == nil && j.LastModifiedDateTime != nil {
		return true
	}
	// an item without a timestamp is considered "less than" an item with a timestamp
	if i.LastModifiedDateTime != nil && j.LastModifiedDateTime == nil {
		return false
	}
	// fallback to name if no dateTime is set on both items
	return strings.ToLower(i.Name) < strings.ToLower(j.Name)
}

func reverse(less func(i, j int) bool) func(i, j int) bool {
	return func(i, j int) bool {
		return !less(i, j)
	}
}
