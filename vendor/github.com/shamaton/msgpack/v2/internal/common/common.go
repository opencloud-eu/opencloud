package common

import (
	"reflect"
	"strings"
)

// Common is used encoding/decoding
type Common struct {
}

// CheckField returns flag whether should encode/decode or not and field name
func (c *Common) CheckField(field reflect.StructField) (public, omit bool, name string) {
	// A to Z
	if !c.isPublic(field.Name) {
		return false, false, ""
	}
	tag := field.Tag.Get("msgpack")
	if tag == "" {
		return true, false, field.Name
	}

	parts := strings.Split(tag, ",")
	// check ignore
	if parts[0] == "-" {
		return false, false, ""
	}
	// check omitempty
	for _, part := range parts[1:] {
		if part == "omitempty" {
			omit = true
		}
	}
	// check name
	name = field.Name
	if parts[0] != "" {
		name = parts[0]
	}
	return true, omit, name
}

func (c *Common) isPublic(name string) bool {
	return 0x41 <= name[0] && name[0] <= 0x5a
}
