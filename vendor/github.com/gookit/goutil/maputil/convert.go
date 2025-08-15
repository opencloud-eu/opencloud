package maputil

import (
	"errors"
	"reflect"
	"strings"

	"github.com/gookit/goutil/arrutil"
	"github.com/gookit/goutil/comdef"
	"github.com/gookit/goutil/reflects"
	"github.com/gookit/goutil/strutil"
)

// alias functions
var (
	// ToStrMap convert map[string]any to map[string]string
	ToStrMap = ToStringMap
	// ToL2StrMap convert map[string]any to map[string]map[string]string
	ToL2StrMap = ToL2StringMap
)

// KeyToLower convert keys to lower case.
func KeyToLower(src map[string]string) map[string]string {
	if len(src) == 0 {
		return src
	}

	newMp := make(map[string]string, len(src))
	for k, v := range src {
		k = strings.ToLower(k)
		newMp[k] = v
	}
	return newMp
}

// AnyToStrMap try convert any(map[string]any, map[string]string) to map[string]string
func AnyToStrMap(src any) map[string]string {
	if src == nil {
		return nil
	}

	if m, ok := src.(map[string]string); ok {
		return m
	}
	if m, ok := src.(map[string]any); ok {
		return ToStringMap(m)
	}
	return nil
}

// ToStringMap simple convert map[string]any to map[string]string
func ToStringMap(src map[string]any) map[string]string {
	strMp := make(map[string]string, len(src))
	for k, v := range src {
		strMp[k] = strutil.SafeString(v)
	}
	return strMp
}

// ToL2StringMap convert map[string]any to map[string]map[string]string
func ToL2StringMap(groupsMap map[string]any) map[string]map[string]string {
	if len(groupsMap) == 0 {
		return nil
	}

	l2sMap := make(map[string]map[string]string, len(groupsMap))

	for k, v := range groupsMap {
		if mp, ok := v.(map[string]any); ok {
			l2sMap[k] = ToStringMap(mp)
		} else if smp, ok := v.(map[string]string); ok {
			l2sMap[k] = smp
		}
	}
	return l2sMap
}

// CombineToSMap combine two string-slices to SMap(map[string]string)
func CombineToSMap(keys, values []string) SMap {
	return arrutil.CombineToSMap(keys, values)
}

// CombineToMap combine two any slice to map[K]V. alias of arrutil.CombineToMap
func CombineToMap[K comdef.SortedType, V any](keys []K, values []V) map[K]V {
	return arrutil.CombineToMap(keys, values)
}

// SliceToSMap convert string k-v pairs slice to map[string]string
//  - eg: []string{k1,v1,k2,v2} -> map[string]string{k1:v1, k2:v2}
func SliceToSMap(kvPairs ...string) map[string]string {
	ln := len(kvPairs)
	// check kvPairs length must be even
	if ln == 0 || ln%2 != 0 {
		return nil
	}

	sMap := make(map[string]string, ln/2)
	for i := 0; i < ln; i += 2 {
		sMap[kvPairs[i]] = kvPairs[i+1]
	}
	return sMap
}

// SliceToMap convert any k-v pairs slice to map[string]any
func SliceToMap(kvPairs ...any) map[string]any {
	ln := len(kvPairs)
	// check kvPairs length must be even
	if ln == 0 || ln%2 != 0 {
		return nil
	}

	mp := make(map[string]any, ln/2)
	for i := 0; i < ln; i += 2 {
		kStr := strutil.SafeString(kvPairs[i])
		mp[kStr] = kvPairs[i+1]
	}
	return mp
}

// SliceToTypeMap convert k-v pairs slice to map[string]T
func SliceToTypeMap[T any](valFunc func(any) T, kvPairs ...any) map[string]T {
	ln := len(kvPairs)
	// check kvPairs length must be even
	if ln == 0 || ln%2 != 0 {
		return nil
	}

	mp := make(map[string]T, ln/2)
	for i := 0; i < ln; i += 2 {
		kStr := strutil.SafeString(kvPairs[i])
		mp[kStr] = valFunc(kvPairs[i+1])
	}
	return mp
}

// ToAnyMap convert map[TYPE1]TYPE2 to map[string]any
func ToAnyMap(mp any) map[string]any {
	amp, _ := TryAnyMap(mp)
	return amp
}

// TryAnyMap convert map[TYPE1]TYPE2 to map[string]any
func TryAnyMap(mp any) (map[string]any, error) {
	if aMp, ok := mp.(map[string]any); ok {
		return aMp, nil
	}
	if sMp, ok := mp.(map[string]string); ok {
		anyMp := make(map[string]any, len(sMp))
		for k, v := range sMp {
			anyMp[k] = v
		}
		return anyMp, nil
	}

	rv := reflect.Indirect(reflect.ValueOf(mp))
	if rv.Kind() != reflect.Map {
		return nil, errors.New("input is not a map value type")
	}

	anyMp := make(map[string]any, rv.Len())
	for _, key := range rv.MapKeys() {
		keyStr := strutil.SafeString(key.Interface())
		anyMp[keyStr] = rv.MapIndex(key).Interface()
	}
	return anyMp, nil
}

// HTTPQueryString convert map[string]any data to http query string.
func HTTPQueryString(data map[string]any) string {
	ss := make([]string, 0, len(data))
	for k, v := range data {
		ss = append(ss, k+"="+strutil.QuietString(v))
	}

	return strings.Join(ss, "&")
}

// StringsMapToAnyMap convert map[string][]string to map[string]any
//
//	Example:
//	{"k1": []string{"v1", "v2"}, "k2": []string{"v3"}}
//	=>
//	{"k": []string{"v1", "v2"}, "k2": "v3"}
//
//	mp := StringsMapToAnyMap(httpReq.Header)
func StringsMapToAnyMap(ssMp map[string][]string) map[string]any {
	if len(ssMp) == 0 {
		return nil
	}

	anyMp := make(map[string]any, len(ssMp))
	for k, v := range ssMp {
		if len(v) == 1 {
			anyMp[k] = v[0]
			continue
		}
		anyMp[k] = v
	}
	return anyMp
}

// ToString simple and quickly convert map[string]any to string.
func ToString(mp map[string]any) string {
	if mp == nil {
		return ""
	}
	if len(mp) == 0 {
		return "{}"
	}

	buf := make([]byte, 0, len(mp)*16)
	buf = append(buf, '{')

	for k, val := range mp {
		buf = append(buf, k...)
		buf = append(buf, ':')

		str := strutil.QuietString(val)
		buf = append(buf, str...)
		buf = append(buf, ',', ' ')
	}

	// remove last ', '
	buf = append(buf[:len(buf)-2], '}')
	return strutil.Byte2str(buf)
}

// ToString2 simple and quickly convert a map to string.
func ToString2(mp any) string { return NewFormatter(mp).Format() }

// FormatIndent format map data to string with newline and indent.
func FormatIndent(mp any, indent string) string {
	return NewFormatter(mp).WithIndent(indent).Format()
}

/*************************************************************
 * Flat convert tree map to flatten key-value map.
 *************************************************************/

// Flatten convert tree map to flat key-value map.
//
// Examples:
//
//	{"top": {"sub": "value", "sub2": "value2"} }
//	->
//	{"top.sub": "value", "top.sub2": "value2" }
func Flatten(mp map[string]any) map[string]any {
	if mp == nil {
		return nil
	}

	flatMp := make(map[string]any, len(mp)*2)
	reflects.FlatMap(reflect.ValueOf(mp), func(path string, val reflect.Value) {
		flatMp[path] = val.Interface()
	})

	return flatMp
}

// FlatWithFunc flat a tree-map with custom collect handle func
func FlatWithFunc(mp map[string]any, fn reflects.FlatFunc) {
	if mp == nil || fn == nil {
		return
	}
	reflects.FlatMap(reflect.ValueOf(mp), fn)
}
