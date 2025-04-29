package maputil

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/gookit/goutil/comdef"
	"github.com/gookit/goutil/reflects"
)

// some consts for separators
const (
	Wildcard = "*"
	PathSep  = "."
)

// DeepGet value by key path. eg "top" "top.sub"
func DeepGet(mp map[string]any, path string) (val any) {
	val, _ = GetByPath(path, mp)
	return
}

// QuietGet value by key path. eg "top" "top.sub"
func QuietGet(mp map[string]any, path string) (val any) {
	val, _ = GetByPath(path, mp)
	return
}

// GetFromAny get value by key path from any(map,slice) data. eg "top" "top.sub"
func GetFromAny(path string, data any) (val any, ok bool) {
	// empty data
	if data == nil {
		return nil, false
	}
	if len(path) == 0 {
		return data, true
	}

	return getByPathKeys(data, strings.Split(path, "."))
}

// GetByPath get value by key path from a map(map[string]any). eg "top" "top.sub"
func GetByPath(path string, mp map[string]any) (val any, ok bool) {
	if len(path) == 0 {
		return mp, true
	}
	if val, ok := mp[path]; ok {
		return val, true
	}

	// no sub key
	if len(mp) == 0 || strings.IndexByte(path, '.') < 1 {
		return nil, false
	}

	// key is path. eg: "top.sub"
	return GetByPathKeys(mp, strings.Split(path, "."))
}

// GetByPathKeys get value by path keys from a map(map[string]any). eg "top" "top.sub"
//
// Example:
//
//	mp := map[string]any{
//		"top": map[string]any{
//			"sub": "value",
//		},
//	}
//	val, ok := GetByPathKeys(mp, []string{"top", "sub"}) // return "value", true
func GetByPathKeys(mp map[string]any, keys []string) (val any, ok bool) {
	kl := len(keys)
	if kl == 0 {
		return mp, true
	}

	// find top item data use top key
	var item any
	topK := keys[0]
	if item, ok = mp[topK]; !ok {
		return
	}

	// find sub item data use sub key
	return getByPathKeys(item, keys[1:])
}

func getByPathKeys(item any, keys []string) (val any, ok bool) {
	kl := len(keys)

	for i, k := range keys {
		switch tData := item.(type) {
		case map[string]string: // is string map
			if item, ok = tData[k]; !ok {
				return
			}
		case map[string]any: // is map(decode from toml/json/yaml)
			if item, ok = tData[k]; !ok {
				return
			}
		case map[any]any: // is map(decode from yaml.v2)
			if item, ok = tData[k]; !ok {
				return
			}
		case []map[string]any: // is an any-map slice
			if k == Wildcard {
				if kl == i+1 { // * is last key
					return tData, true
				}

				// * is not last key, find sub item data
				sl := make([]any, 0, len(tData))
				for _, v := range tData {
					if val, ok = getByPathKeys(v, keys[i+1:]); ok {
						sl = append(sl, val)
					}
				}

				if len(sl) > 0 {
					return sl, true
				}
				return nil, false
			}

			// k is index number
			idx, err := strconv.Atoi(k)
			if err != nil || idx >= len(tData) {
				return nil, false
			}
			item = tData[idx]
		default:
			if k == Wildcard && kl == i+1 { // * is last key
				return tData, true
			}

			rv := reflect.ValueOf(tData)
			// check is slice
			if rv.Kind() == reflect.Slice {
				if k == Wildcard {
					// * is not last key, find sub item data
					sl := make([]any, 0, rv.Len())
					for si := 0; si < rv.Len(); si++ {
						el := reflects.Indirect(rv.Index(si))
						if el.Kind() != reflect.Map {
							return nil, false
						}

						// el is map value.
						if val, ok = getByPathKeys(el.Interface(), keys[i+1:]); ok {
							sl = append(sl, val)
						}
					}

					if len(sl) > 0 {
						return sl, true
					}
					return nil, false
				}

				// check k is index number
				ii, err := strconv.Atoi(k)
				if err != nil || ii >= rv.Len() {
					return nil, false
				}

				item = rv.Index(ii).Interface()
				continue
			}

			// as error
			return nil, false
		}

		// next is last key and it is *
		if kl == i+2 && keys[i+1] == Wildcard {
			return item, true
		}
	}

	return item, true
}

// Keys get all keys of the given map.
func Keys(mp any) (keys []string) {
	rftVal := reflect.Indirect(reflect.ValueOf(mp))
	if rftVal.Kind() != reflect.Map {
		return
	}

	keys = make([]string, 0, rftVal.Len())
	for _, key := range rftVal.MapKeys() {
		keys = append(keys, key.String())
	}
	return
}

// TypedKeys get all keys of the given typed map.
func TypedKeys[K comdef.SimpleType, V any](mp map[K]V) (keys []K) {
	for key := range mp {
		keys = append(keys, key)
	}
	return
}

// Values get all values from the given map.
func Values(mp any) (values []any) {
	rv := reflect.Indirect(reflect.ValueOf(mp))
	if rv.Kind() != reflect.Map {
		return
	}

	values = make([]any, 0, rv.Len())
	for _, key := range rv.MapKeys() {
		values = append(values, rv.MapIndex(key).Interface())
	}
	return
}

// TypedValues get all values from the given typed map.
func TypedValues[K comdef.SimpleType, V any](mp map[K]V) (values []V) {
	for _, val := range mp {
		values = append(values, val)
	}
	return
}

// EachAnyMap iterates the given map and calls the given function for each item.
func EachAnyMap(mp any, fn func(key string, val any)) {
	rv := reflect.Indirect(reflect.ValueOf(mp))
	if rv.Kind() != reflect.Map {
		panic("not a map value")
	}

	for _, key := range rv.MapKeys() {
		fn(key.String(), rv.MapIndex(key).Interface())
	}
}

// EachTypedMap iterates the given map and calls the given function for each item.
func EachTypedMap[K comdef.SimpleType, V any](mp map[K]V, fn func(key K, val V)) {
	for key, val := range mp {
		fn(key, val)
	}
}
