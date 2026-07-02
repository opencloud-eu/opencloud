package mapping

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// DeserializeStringsAt is DeserializeAt for a string-valued map (e.g. CS3
// ArbitraryMetadata), parsing each string into the field's Go type via strconv/
// time.Parse. Used to build a graph DriveItem facet. Returns nil when nothing
// under the prefix matched.
func DeserializeStringsAt[T any](fields map[string]string, prefix string) *T {
	t := reflect.TypeFor[T]()
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("mapping: DeserializeStringsAt requires a struct type, got %v", t))
	}
	out := reflect.New(t)
	// Callers pass a flat-key prefix with a trailing dot (e.g.
	// "libre.graph.audio."); fillStruct joins segments with ".", so drop it.
	if !fillStruct(out.Elem(), fields, strings.TrimSuffix(prefix, "."), setValueFromString) {
		return nil
	}
	return out.Interface().(*T)
}

// setValueFromString parses the string raw into v's Go type via strconv/
// time.Parse, returning a descriptive error on failure.
func setValueFromString(v reflect.Value, raw string) error {
	if v.Kind() == reflect.Ptr {
		alloc := reflect.New(v.Type().Elem())
		if err := setValueFromString(alloc.Elem(), raw); err != nil {
			return err
		}
		v.Set(alloc)
		return nil
	}
	if v.Type() == timeType || v.Type() == timestampType {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return fmt.Errorf("parse time %q: %w", raw, err)
		}
		setParsedTime(v, t)
		return nil
	}
	switch v.Kind() {
	case reflect.String:
		v.SetString(raw)
		return nil
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("parse bool %q: %w", raw, err)
		}
		v.SetBool(b)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse int %q: %w", raw, err)
		}
		v.SetInt(n)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse uint %q: %w", raw, err)
		}
		v.SetUint(n)
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse float %q: %w", raw, err)
		}
		v.SetFloat(f)
		return nil
	}
	return fmt.Errorf("unsupported target kind %s", v.Kind())
}
