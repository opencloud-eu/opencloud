// Package locks provides functionality to parse lockIDs.
//
// It can be used to bridge requests from different clients that send lockIDs in different formats.
// For example, Microsoft Office Online sends the lockID in a JSON string,
// while other clients send the lockID as a plain string.
package locks

import (
	"encoding/json"
)

// LockParser is the interface that wraps the ParseLock method
type LockParser interface {
	ParseLock(id string) string
}

// LegacyLockParser is a lock parser that can extract the lockID from a JSON string
type LegacyLockParser struct{}

// NoopLockParser is a lock parser that does not change the lockID
type NoopLockParser struct{}

// ParseLock will return the lockID as is
func (*NoopLockParser) ParseLock(id string) string {
	return id
}

// ParseLock extracts the lockID from a JSON string.
// For Microsoft Office Online we need to extract the lockID from the JSON string
// that is sent by the WOPI client.
// The JSON string is expected to have the following format:
//
//	{
//	  "L": "12345678",
//	  "F": 4,
//	  "E": 2,
//	  "C": "",
//	  "P": "3453345345346",
//	  "M": "12345678"
//	}
//
// or
//
//	{
//	  "S": "12345678",
//	  "F": 4,
//	  "E": 2,
//	  "C": "",
//	  "P": "3453345345346",
//	  "M": "12345678"
//	}
//
// If the JSON string is not in the expected format, the original lockID will be returned.
func (*LegacyLockParser) ParseLock(id string) string {
	var decodedValues map[string]any
	err := json.Unmarshal([]byte(id), &decodedValues)
	if err != nil || len(decodedValues) == 0 {
		return id
	}
	if v, ok := decodedValues["L"]; ok {
		if idString, ok := v.(string); ok {
			return idString
		}
	}
	if v, ok := decodedValues["S"]; ok {
		if idString, ok := v.(string); ok {
			return idString
		}
	}
	return id
}
