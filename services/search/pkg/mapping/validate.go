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
	fields := collectFields(t, "")
	var unknown, miscased, unbroken []string
	for k, opts := range overrides {
		goType, ok := fields[k]
		if !ok {
			unknown = append(unknown, k)
			continue
		}
		if opts.NoWordBreaker != nil && !*opts.NoWordBreaker && !effectivelyKeyword(opts, goType) {
			unbroken = append(unbroken, k)
		}
		// CaseInsensitive routes queries to a <field>_lowercase sibling, which is
		// only generated for keyword/path fields; on any other type the query
		// would target a non-existent field and silently match nothing. Use the
		// effective type (override, else the inferred Go type), since an override
		// with no explicit Type still infers keyword/numeric/... from the field.
		if opts.CaseInsensitive != nil && *opts.CaseInsensitive && !effectivelyCased(opts, goType) {
			miscased = append(miscased, k)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("mapping: unknown override keys: %s", strings.Join(unknown, ", "))
	}
	if len(miscased) > 0 {
		sort.Strings(miscased)
		return fmt.Errorf("mapping: CaseInsensitive is only valid on keyword/path fields: %s", strings.Join(miscased, ", "))
	}
	if len(unbroken) > 0 {
		sort.Strings(unbroken)
		return fmt.Errorf("mapping: NoWordBreaker is only valid on keyword fields: %s", strings.Join(unbroken, ", "))
	}
	return nil
}

// effectivelyCased reports whether a field is keyword/path (the only types that
// get a _lowercase sibling), from the override type or the inferred Go type.
func effectivelyCased(opts FieldOpts, goType reflect.Type) bool {
	eff := opts.Type
	if eff == "" && goType != nil {
		eff = inferType(goType)
	}
	return eff == TypeKeyword || eff == TypePath
}

// effectivelyKeyword reports whether a field is a keyword, the only type
// NoWordBreaker applies to.
func effectivelyKeyword(opts FieldOpts, goType reflect.Type) bool {
	eff := opts.Type
	if eff == "" && goType != nil {
		eff = inferType(goType)
	}
	return eff == TypeKeyword
}

// collectFields maps every known field name (nested as "parent.child") to its Go
// type. Embedded structs are flattened, matching encoding/json.
func collectFields(t reflect.Type, prefix string) map[string]reflect.Type {
	out := map[string]reflect.Type{}
	_ = walkFields(t, func(fi fieldInfo) error {
		key := fi.Name
		if prefix != "" {
			key = prefix + "." + fi.Name
		}
		out[key] = fi.GoField.Type
		if sub := structType(fi.GoField.Type); sub != nil {
			for k, v := range collectFields(sub, key) {
				out[k] = v
			}
		}
		return nil
	})
	return out
}
