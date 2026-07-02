package mapping

import (
	"fmt"
	"reflect"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Deserialize builds a *T from bleve's flat hit.Fields map (json-tag keys,
// "parent.child" for nested pointers). Used to rebuild a search Resource from a
// hit. Fail-soft: an unparseable field stays at its zero value.
func Deserialize[T any](fields map[string]any) *T {
	t := reflect.TypeFor[T]()
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("mapping: Deserialize requires a struct type, got %v", t))
	}
	out := reflect.New(t)
	fillStruct(out.Elem(), fields, "", setValue)
	return out.Interface().(*T)
}

// DeserializeAt is Deserialize scoped to a dotted prefix, used to rebuild one
// search-result facet (e.g. "audio"). Returns nil when nothing matched, so the
// caller can leave the enclosing pointer nil.
func DeserializeAt[T any](fields map[string]any, prefix string) *T {
	t := reflect.TypeFor[T]()
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("mapping: DeserializeAt requires a struct type, got %v", t))
	}
	out := reflect.New(t)
	if !fillStruct(out.Elem(), fields, prefix, setValue) {
		return nil
	}
	return out.Interface().(*T)
}

// fillStruct walks v's exported fields, reading values from the flat fields map
// (json-tag keys, "parent.child" for nested pointers). setLeaf converts each raw
// value into a leaf, which is what lets the any-valued and string-valued
// deserializers share one walker. Returns true if any leaf was populated.
func fillStruct[V any](v reflect.Value, fields map[string]V, prefix string, setLeaf func(reflect.Value, V) error) bool {
	t := v.Type()
	touched := false
	for i := 0; i < t.NumField(); i++ {
		fi := resolveField(t.Field(i))
		if fi.Skip {
			continue
		}
		fv := v.Field(i)

		if fi.Embedded {
			// Embedded *struct: allocate, recurse, keep the pointer only if set.
			if fv.Kind() == reflect.Ptr {
				if fv.Type().Elem().Kind() != reflect.Struct || !fv.CanSet() {
					continue
				}
				alloc := reflect.New(fv.Type().Elem())
				if fillStruct(alloc.Elem(), fields, prefix, setLeaf) {
					fv.Set(alloc)
					touched = true
				}
				continue
			}
			if fv.Kind() == reflect.Struct && fillStruct(fv, fields, prefix, setLeaf) {
				touched = true
			}
			continue
		}

		key := fi.Name
		if prefix != "" {
			key = prefix + "." + fi.Name
		}

		// Pointer to a nested (non-time) struct: recurse, keep the pointer only
		// if a field was populated.
		if fv.Kind() == reflect.Ptr {
			if elem := fv.Type().Elem(); elem.Kind() == reflect.Struct && elem != timeType && elem != timestampType {
				alloc := reflect.New(elem)
				if fillStruct(alloc.Elem(), fields, key, setLeaf) {
					fv.Set(alloc)
					touched = true
				}
				continue
			}
		}

		if raw, ok := fields[key]; ok && setLeaf(fv, raw) == nil {
			touched = true
		}
	}
	return touched
}

// setValue writes raw (an any value from a bleve hit) into v, converting to the
// field's type. Returns an error on nil or a type mismatch.
func setValue(v reflect.Value, raw any) error {
	if v.Kind() == reflect.Ptr {
		alloc := reflect.New(v.Type().Elem())
		if err := setValue(alloc.Elem(), raw); err != nil {
			return err
		}
		v.Set(alloc)
		return nil
	}
	if v.Type() == timeType || v.Type() == timestampType {
		t, ok := parseTime(raw)
		if !ok {
			return fmt.Errorf("not an RFC3339 time: %v", raw)
		}
		setParsedTime(v, t)
		return nil
	}
	if v.Kind() == reflect.Slice {
		return setSlice(v, raw)
	}
	rv := reflect.ValueOf(raw)
	if !rv.IsValid() {
		return fmt.Errorf("nil value for %s", v.Type())
	}
	if !rv.Type().ConvertibleTo(v.Type()) {
		return fmt.Errorf("cannot convert %s to %s", rv.Type(), v.Type())
	}
	v.Set(rv.Convert(v.Type()))
	return nil
}

func setSlice(v reflect.Value, raw any) error {
	items, ok := raw.([]any)
	if !ok {
		// bleve unwraps single-element slices; re-wrap here.
		items = []any{raw}
	}
	// Compact in place with a single MakeSlice: unparseable elements are
	// dropped, Slice(0, j) trims the tail.
	out := reflect.MakeSlice(v.Type(), len(items), len(items))
	j := 0
	for _, item := range items {
		if setValue(out.Index(j), item) == nil {
			j++
		}
	}
	if j == 0 {
		return fmt.Errorf("no slice elements set from %T", raw)
	}
	v.Set(out.Slice(0, j))
	return nil
}

func parseTime(raw any) (time.Time, bool) {
	s, ok := raw.(string)
	if !ok {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// setParsedTime writes t into v, which must be a time.Time or
// timestamppb.Timestamp field.
func setParsedTime(v reflect.Value, t time.Time) {
	if v.Type() == timestampType {
		v.Set(reflect.ValueOf(*timestamppb.New(t)))
		return
	}
	v.Set(reflect.ValueOf(t))
}
