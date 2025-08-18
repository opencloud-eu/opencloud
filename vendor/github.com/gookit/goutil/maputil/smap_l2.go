package maputil

import "github.com/gookit/goutil/strutil"

// L2StrMap is alias of map[string]map[string]string
type L2StrMap map[string]map[string]string

// Load data, merge new data to old
func (m L2StrMap) Load(mp map[string]map[string]string) {
	for k, v := range mp {
		if oldV, ok := m[k]; ok {
			for k1, v1 := range v {
				oldV[k1] = v1
			}
			m[k] = oldV
		} else {
			m[k] = v
		}
	}
}

// Value get by key path. eg: "top.sub"
func (m L2StrMap) Value(key string) (val string, ok bool) {
	top, sub, found := strutil.Cut(key, KeySepStr)
	if !found {
		return "", false
	}

	if vals, ok1 := m[top]; ok1 {
		val, ok = vals[sub]
		return
	}
	return "", false
}

// Get value by key path. eg: "top.sub"
func (m L2StrMap) Get(key string) string {
	val, _ := m.Value(key)
	return val
}

// Exists check key path exists. eg: "top.sub"
func (m L2StrMap) Exists(key string) bool {
	_, ok := m.Value(key)
	return ok
}

// StrMap get by top key. eg: "top"
func (m L2StrMap) StrMap(top string) StrMap {
	return m[top]
}
