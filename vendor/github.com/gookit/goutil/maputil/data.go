package maputil

import (
	"strings"

	"github.com/gookit/goutil/arrutil"
	"github.com/gookit/goutil/internal/comfunc"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/strutil"
)

// Data an map data type
type Data map[string]any

// Map alias of Data
type Map = Data

// Has value on the data map
func (d Data) Has(key string) bool {
	_, ok := d.GetByPath(key)
	return ok
}

// IsEmpty if the data map
func (d Data) IsEmpty() bool {
	return len(d) == 0
}

//
// endregion
// region T: set value(s)
//

// Set value to the data map
func (d Data) Set(key string, val any) {
	d[key] = val
}

// SetByPath sets a value in the map.
// Supports dot syntax to set deep values.
//
// Example:
//
//	d.SetByPath("name.first", "Mat")
func (d Data) SetByPath(path string, value any) error {
	if path == "" {
		return nil
	}
	return d.SetByKeys(strings.Split(path, KeySepStr), value)
}

// SetByKeys sets a value in the map by path keys.
// Supports dot syntax to set deep values.
//
// Example:
//
//	d.SetByKeys([]string{"name", "first"}, "Mat")
func (d Data) SetByKeys(keys []string, value any) error {
	kln := len(keys)
	if kln == 0 {
		return nil
	}

	// special handle d is empty.
	if len(d) == 0 {
		if kln == 1 {
			d.Set(keys[0], value)
		} else {
			d.Set(keys[0], MakeByKeys(keys[1:], value))
		}
		return nil
	}

	return SetByKeys((*map[string]any)(&d), keys, value)
	// It's ok, but use `func (d *Data)`
	// return SetByKeys((*map[string]any)(d), keys, value)
}

//
// endregion
// region T: read value(s)
//

// Value get from the data map
func (d Data) Value(key string) (any, bool) {
	val, ok := d.GetByPath(key)
	return val, ok
}

// Get value from the data map.
// Supports dot syntax to get deep values. eg: top.sub
func (d Data) Get(key string) any {
	if val, ok := d.GetByPath(key); ok {
		return val
	}
	return nil
}

// One get value from the data by multi paths. will return first founded value
func (d Data) One(keys ...string) any {
	if val, ok := d.TryOne(keys...); ok {
		return val
	}
	return nil
}

// TryOne get value from the data by multi paths. will return first founded value
func (d Data) TryOne(keys ...string) (any, bool) {
	for _, path := range keys {
		if val, ok := d.GetByPath(path); ok {
			return val, true
		}
	}
	return nil, false
}

// GetByPath get value from the data map by path. eg: top.sub
// Supports dot syntax to get deep values.
func (d Data) GetByPath(path string) (any, bool) {
	if val, ok := d[path]; ok {
		return val, true
	}

	// is a key path.
	if strings.ContainsRune(path, '.') {
		val, ok := GetByPath(path, d)
		if ok {
			return val, true
		}
	}
	return nil, false
}

// Default get value from the data map with default value
func (d Data) Default(key string, def any) any {
	if val, ok := d.GetByPath(key); ok {
		return val
	}
	return def
}

// Int value get
func (d Data) Int(key string) int {
	if val, ok := d.GetByPath(key); ok {
		return mathutil.QuietInt(val)
	}
	return 0
}

// Int64 value get
func (d Data) Int64(key string) int64 {
	if val, ok := d.GetByPath(key); ok {
		return mathutil.QuietInt64(val)
	}
	return 0
}

// Uint value get
func (d Data) Uint(key string) uint {
	if val, ok := d.GetByPath(key); ok {
		return mathutil.QuietUint(val)
	}
	return 0
}

// Uint64 value get
func (d Data) Uint64(key string) uint64 {
	if val, ok := d.GetByPath(key); ok {
		return mathutil.QuietUint64(val)
	}
	return 0
}

// Str value gets by key
func (d Data) Str(key string) string {
	if val, ok := d.GetByPath(key); ok {
		return strutil.SafeString(val)
	}
	return ""
}

// StrOne value gets by multi keys, will return first value
func (d Data) StrOne(keys ...string) string {
	for _, key := range keys {
		if val, ok := d.GetByPath(key); ok {
			return strutil.SafeString(val)
		}
	}
	return ""
}

// Bool value get
func (d Data) Bool(key string) bool {
	val, ok := d.GetByPath(key)
	if !ok {
		return false
	}
	return comfunc.Bool(val)
}

// BoolOne value gets from multi keys, return first value
func (d Data) BoolOne(keys ...string) bool {
	for _, key := range keys {
		if val, ok := d.GetByPath(key); ok {
			return comfunc.Bool(val)
		}
	}
	return false
}

// StringsOne get []string value by multi keys, return first founded value
func (d Data) StringsOne(keys ...string) []string {
	for _, key := range keys {
		if val, ok := d.GetByPath(key); ok {
			return arrutil.AnyToStrings(val)
		}
	}
	return nil
}

// Strings get []string value by key
func (d Data) Strings(key string) []string {
	if val, ok := d.GetByPath(key); ok {
		return arrutil.AnyToStrings(val)
	}
	return nil
}

// StrSplit get strings by split string value
func (d Data) StrSplit(key, sep string) []string {
	if val, ok := d.GetByPath(key); ok {
		return strings.Split(strutil.SafeString(val), sep)
	}
	return nil
}

// StringsByStr value gets by key, will split string value by ","
func (d Data) StringsByStr(key string) []string {
	return d.StrSplit(key, ",")
}

// StrMap get map[string]string value
func (d Data) StrMap(key string) map[string]string {
	return d.StringMap(key)
}

// StringMap get map[string]string value
func (d Data) StringMap(key string) map[string]string {
	val, ok := d.GetByPath(key)
	if !ok {
		return nil
	}

	switch tv := val.(type) {
	case map[string]string:
		return tv
	case map[string]any:
		return ToStringMap(tv)
	default:
		return nil
	}
}

// Sub get sub value(map[string]any) as new Data
func (d Data) Sub(key string) Data {
	if val, ok := d.GetByPath(key); ok {
		return d.toAnyMap(val)
	}
	return nil
}

// AnyMap get sub value as map[string]any
func (d Data) AnyMap(key string) map[string]any {
	if val, ok := d.GetByPath(key); ok {
		return d.toAnyMap(val)
	}
	return nil
}

// AnyMap get sub value as map[string]any
func (d Data) toAnyMap(val any) map[string]any {
	switch tv := val.(type) {
	case map[string]string:
		return ToAnyMap(tv)
	case map[string]any:
		return tv
	default:
		return nil
	}
}

// Slice get []any value from data map
func (d Data) Slice(key string) ([]any, error) {
	val, ok := d.GetByPath(key)
	if !ok {
		return nil, nil
	}
	return arrutil.AnyToSlice(val)
}

// Keys of the data map
func (d Data) Keys() []string {
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	return keys
}

// ToStringMap convert to map[string]string
func (d Data) ToStringMap() map[string]string {
	return ToStringMap(d)
}

// String data to string
func (d Data) String() string {
	return ToString(d)
}

// Load other data to current data map
func (d Data) Load(sub map[string]any) {
	for name, val := range sub {
		d[name] = val
	}
}

// LoadSMap to data
func (d Data) LoadSMap(smp map[string]string) {
	for name, val := range smp {
		d[name] = val
	}
}
