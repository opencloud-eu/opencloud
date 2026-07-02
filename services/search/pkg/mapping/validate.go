package mapping

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Validate returns an error if any override key does not match a known field
// name in t. Top-level fields are identified by their json-tag name; nested
// named struct fields are reachable as "parent.child". Embedded (anonymous)
// structs are flattened, so their fields sit at the parent level (as with
// encoding/json).
func Validate(t reflect.Type, overrides map[string]FieldOpts) error {
	if len(overrides) == 0 {
		return nil
	}
	names := collectNames(t, "")
	var unknown []string
	for k := range overrides {
		if _, ok := names[k]; !ok {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return fmt.Errorf("mapping: unknown override keys: %s", strings.Join(unknown, ", "))
}

func collectNames(t reflect.Type, prefix string) map[string]struct{} {
	out := map[string]struct{}{}
	_ = walkFields(t, func(fi fieldInfo) error {
		key := fi.Name
		if prefix != "" {
			key = prefix + "." + fi.Name
		}
		out[key] = struct{}{}
		if sub := structType(fi.GoField.Type); sub != nil {
			for k := range collectNames(sub, key) {
				out[k] = struct{}{}
			}
		}
		return nil
	})
	return out
}
