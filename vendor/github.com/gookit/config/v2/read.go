package config

import (
	"strconv"
	"strings"
	"time"

	"github.com/gookit/goutil/envutil"
	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/strutil"
)

// Exists key exists check
func Exists(key string, findByPath ...bool) bool { return dc.Exists(key, findByPath...) }

// Exists key exists check
func (c *Config) Exists(key string, findByPath ...bool) (ok bool) {
	sep := c.opts.Delimiter
	if key = formatKey(key, string(sep)); key == "" {
		return
	}

	if _, ok = c.data[key]; ok {
		return
	}

	// disable find by path.
	if len(findByPath) > 0 && !findByPath[0] {
		return
	}

	// has sub key? eg. "lang.dir"
	if strings.IndexByte(key, sep) == -1 {
		return
	}

	keys := strings.Split(key, string(sep))
	topK := keys[0]

	// find top item data based on top key
	var item any
	if item, ok = c.data[topK]; !ok {
		return
	}
	for _, k := range keys[1:] {
		switch typeData := item.(type) {
		case map[string]int: // is map(from Set)
			if item, ok = typeData[k]; !ok {
				return
			}
		case map[string]string: // is map(from Set)
			if item, ok = typeData[k]; !ok {
				return
			}
		case map[string]any: // is map(decode from toml/json/yaml.v3)
			if item, ok = typeData[k]; !ok {
				return
			}
		case map[any]any: // is map(decode from yaml.v2)
			if item, ok = typeData[k]; !ok {
				return
			}
		case []int: // is array(is from Set)
			i, err := strconv.Atoi(k)

			// check slice index
			if err != nil || len(typeData) < i {
				return false
			}
		case []string: // is array(is from Set)
			i, err := strconv.Atoi(k)
			if err != nil || len(typeData) < i {
				return false
			}
		case []any: // is array(load from file)
			i, err := strconv.Atoi(k)
			if err != nil || len(typeData) < i {
				return false
			}
		default: // error
			return false
		}
	}
	return true
}

/*************************************************************
 * read config data
 *************************************************************/

// Data return all config data
func Data() map[string]any { return dc.Data() }

// Data get all config data.
//
// Note: will don't apply any options, like ParseEnv
func (c *Config) Data() map[string]any {
	return c.data
}

// Sub return a map config data by key
func Sub(key string) map[string]any { return dc.Sub(key) }

// Sub get a map config data by key
//
// Note: will don't apply any options, like ParseEnv
func (c *Config) Sub(key string) map[string]any {
	if mp, ok := c.GetValue(key); ok {
		if mmp, ok := mp.(map[string]any); ok {
			return mmp
		}
	}
	return nil
}

// Keys return all config data
func Keys() []string { return dc.Keys() }

// Keys get all config data
func (c *Config) Keys() []string {
	keys := make([]string, 0, len(c.data))
	for key := range c.data {
		keys = append(keys, key)
	}
	return keys
}

// Get config value by key string, support get sub-value by key path(eg. 'map.key'),
func Get(key string, findByPath ...bool) any { return dc.Get(key, findByPath...) }

// Get config value by key, findByPath default is true.
func (c *Config) Get(key string, findByPath ...bool) any {
	val, _ := c.GetValue(key, findByPath...)
	return val
}

// GetValue get value by given key string. findByPath default is true.
func GetValue(key string, findByPath ...bool) (any, bool) {
	return dc.GetValue(key, findByPath...)
}

// GetValue get value by given key string. findByPath default is true.
//
// Return:
//   - ok is true, find value from config
//   - ok is false, not found or error
func (c *Config) GetValue(key string, findByPath ...bool) (value any, ok bool) {
	sep := c.opts.Delimiter
	if key = formatKey(key, string(sep)); key == "" {
		c.addError(ErrKeyIsEmpty)
		return
	}

	// if not is readonly
	if !c.opts.Readonly {
		c.lock.RLock()
		defer c.lock.RUnlock()
	}

	// is top key
	if value, ok = c.data[key]; ok {
		return
	}

	// disable find by path.
	if len(findByPath) > 0 && !findByPath[0] {
		// c.addError(ErrNotFound)
		return
	}

	// has sub key? eg. "lang.dir"
	if strings.IndexByte(key, sep) == -1 {
		// c.addError(ErrNotFound)
		return
	}

	keys := strings.Split(key, string(sep))
	topK := keys[0]

	// find top item data based on top key
	var item any
	if item, ok = c.data[topK]; !ok {
		// c.addError(ErrNotFound)
		return
	}

	// find child
	// NOTICE: don't merge case, will result in an error.
	// e.g. case []int, []string
	// OR
	// case []int:
	// case []string:
	for _, k := range keys[1:] {
		switch typeData := item.(type) {
		case map[string]int: // is map(from Set)
			if item, ok = typeData[k]; !ok {
				return
			}
		case map[string]string: // is map(from Set)
			if item, ok = typeData[k]; !ok {
				return
			}
		case map[string]any: // is map(decode from toml/json)
			if item, ok = typeData[k]; !ok {
				return
			}
		case map[any]any: // is map(decode from yaml)
			if item, ok = typeData[k]; !ok {
				return
			}
		case []int: // is array(is from Set)
			i, err := strconv.Atoi(k)

			// check slice index
			if err != nil || len(typeData) < i {
				ok = false
				c.addError(err)
				return
			}

			item = typeData[i]
		case []string: // is array(is from Set)
			i, err := strconv.Atoi(k)
			if err != nil || len(typeData) < i {
				ok = false
				c.addError(err)
				return
			}

			item = typeData[i]
		case []any: // is array(load from file)
			i, err := strconv.Atoi(k)
			if err != nil || len(typeData) < i {
				ok = false
				c.addError(err)
				return
			}

			item = typeData[i]
		default: // error
			ok = false
			c.addErrorf("cannot get value of the key '%s'", key)
			return
		}
	}

	return item, true
}

/*************************************************************
 * read config (basic data type)
 *************************************************************/

// String get a string by key
func String(key string, defVal ...string) string { return dc.String(key, defVal...) }

// String get a string by key, if not found return default value
func (c *Config) String(key string, defVal ...string) string {
	value, ok := c.getString(key)

	if !ok && len(defVal) > 0 { // give default value
		value = defVal[0]
	}
	return value
}

// MustString get a string by key, will panic on empty or not exists
func MustString(key string) string { return dc.MustString(key) }

// MustString get a string by key, will panic on empty or not exists
func (c *Config) MustString(key string) string {
	value, ok := c.getString(key)
	if !ok {
		panic("config: string value not found, key: " + key)
	}
	return value
}

func (c *Config) getString(key string) (value string, ok bool) {
	// find from cache
	if c.opts.EnableCache && len(c.strCache) > 0 {
		value, ok = c.strCache[key]
		if ok {
			return
		}
	}

	val, ok := c.GetValue(key)
	if !ok {
		return
	}

	switch typVal := val.(type) {
	// from json `int` always is float64
	case string:
		value = typVal
		if c.opts.ParseEnv {
			value = envutil.ParseEnvValue(value)
		}
	default:
		var err error
		value, err = strutil.AnyToString(val, false)
		if err != nil {
			return "", false
		}
	}

	// add cache
	if ok && c.opts.EnableCache {
		if c.strCache == nil {
			c.strCache = make(map[string]string)
		}
		c.strCache[key] = value
	}
	return
}

// Int get an int by key
func Int(key string, defVal ...int) int { return dc.Int(key, defVal...) }

// Int get a int value, if not found return default value
func (c *Config) Int(key string, defVal ...int) (value int) {
	i64, exist := c.tryInt64(key)

	if exist {
		value = int(i64)
	} else if len(defVal) > 0 {
		value = defVal[0]
	}
	return
}

// Uint get a uint value, if not found return default value
func Uint(key string, defVal ...uint) uint { return dc.Uint(key, defVal...) }

// Uint get a int value, if not found return default value
func (c *Config) Uint(key string, defVal ...uint) (value uint) {
	i64, exist := c.tryInt64(key)

	if exist {
		value = uint(i64)
	} else if len(defVal) > 0 {
		value = defVal[0]
	}
	return
}

// Int64 get a int value, if not found return default value
func Int64(key string, defVal ...int64) int64 { return dc.Int64(key, defVal...) }

// Int64 get a int value, if not found return default value
func (c *Config) Int64(key string, defVal ...int64) (value int64) {
	value, exist := c.tryInt64(key)

	if !exist && len(defVal) > 0 {
		value = defVal[0]
	}
	return
}

// try to get an int64 value by given key
func (c *Config) tryInt64(key string) (value int64, ok bool) {
	strVal, ok := c.getString(key)
	if !ok {
		return
	}

	value, err := strconv.ParseInt(strVal, 10, 0)
	if err != nil {
		c.addError(err)
	}
	return
}

// Duration get a time.Duration type value. if not found return default value
func Duration(key string, defVal ...time.Duration) time.Duration { return dc.Duration(key, defVal...) }

// Duration get a time.Duration type value. if not found return default value
func (c *Config) Duration(key string, defVal ...time.Duration) time.Duration {
	value, exist := c.tryInt64(key)

	if !exist && len(defVal) > 0 {
		return defVal[0]
	}
	return time.Duration(value)
}

// Float get a float64 value, if not found return default value
func Float(key string, defVal ...float64) float64 { return dc.Float(key, defVal...) }

// Float get a float64 by key
func (c *Config) Float(key string, defVal ...float64) (value float64) {
	str, ok := c.getString(key)
	if !ok {
		if len(defVal) > 0 {
			value = defVal[0]
		}
		return
	}

	value, err := strconv.ParseFloat(str, 64)
	if err != nil {
		c.addError(err)
	}
	return
}

// Bool get a bool value, if not found return default value
func Bool(key string, defVal ...bool) bool { return dc.Bool(key, defVal...) }

// Bool looks up a value for a key in this section and attempts to parse that value as a boolean,
// along with a boolean result similar to a map lookup.
//
// of following(case insensitive):
//   - true
//   - yes
//   - false
//   - no
//   - 1
//   - 0
//
// The `ok` boolean will be false in the event that the value could not be parsed as a bool
func (c *Config) Bool(key string, defVal ...bool) (value bool) {
	rawVal, ok := c.getString(key)
	if !ok {
		if len(defVal) > 0 {
			return defVal[0]
		}
		return
	}

	lowerCase := strings.ToLower(rawVal)
	switch lowerCase {
	case "", "0", "false", "no":
		value = false
	case "1", "true", "yes":
		value = true
	default:
		c.addErrorf("the value '%s' cannot be convert to bool", lowerCase)
	}
	return
}

/*************************************************************
 * read config (complex data type)
 *************************************************************/

// Ints get config data as an int slice/array
func Ints(key string) []int { return dc.Ints(key) }

// Ints get config data as an int slice/array
func (c *Config) Ints(key string) (arr []int) {
	rawVal, ok := c.GetValue(key)
	if !ok {
		return
	}

	switch typeData := rawVal.(type) {
	case []int:
		arr = typeData
	case []any:
		for _, v := range typeData {
			iv, err := mathutil.ToInt(v)
			// iv, err := strconv.Atoi(fmt.Sprintf("%v", v))
			if err != nil {
				c.addError(err)
				arr = arr[0:0] // reset
				return
			}

			arr = append(arr, iv)
		}
	default:
		c.addErrorf("value cannot be convert to []int, key is '%s'", key)
	}
	return
}

// IntMap get config data as a map[string]int
func IntMap(key string) map[string]int { return dc.IntMap(key) }

// IntMap get config data as a map[string]int
func (c *Config) IntMap(key string) (mp map[string]int) {
	rawVal, ok := c.GetValue(key)
	if !ok {
		return
	}

	switch typeData := rawVal.(type) {
	case map[string]int: // from Set
		mp = typeData
	case map[string]any: // decode from json,toml
		mp = make(map[string]int)
		for k, v := range typeData {
			// iv, err := strconv.Atoi(fmt.Sprintf("%v", v))
			iv, err := mathutil.ToInt(v)
			if err != nil {
				c.addError(err)
				mp = map[string]int{} // reset
				return
			}
			mp[k] = iv
		}
	case map[any]any: // if decode from yaml
		mp = make(map[string]int)
		for k, v := range typeData {
			// iv, err := strconv.Atoi(fmt.Sprintf( "%v", v))
			iv, err := mathutil.ToInt(v)
			if err != nil {
				c.addError(err)
				mp = map[string]int{} // reset
				return
			}

			// sk := fmt.Sprintf("%v", k)
			sk, _ := strutil.AnyToString(k, false)
			mp[sk] = iv
		}
	default:
		c.addErrorf("value cannot be convert to map[string]int, key is '%s'", key)
	}
	return
}

// Strings get strings by key
func Strings(key string) []string { return dc.Strings(key) }

// Strings get config data as a string slice/array
func (c *Config) Strings(key string) (arr []string) {
	var ok bool
	// find from cache
	if c.opts.EnableCache && len(c.sArrCache) > 0 {
		arr, ok = c.sArrCache[key]
		if ok {
			return
		}
	}

	rawVal, ok := c.GetValue(key)
	if !ok {
		return
	}

	switch typeData := rawVal.(type) {
	case []string:
		arr = typeData
	case []any:
		for _, v := range typeData {
			// arr = append(arr, fmt.Sprintf("%v", v))
			arr = append(arr, strutil.MustString(v))
		}
	default:
		c.addErrorf("value cannot be convert to []string, key is '%s'", key)
		return
	}

	// add cache
	if c.opts.EnableCache {
		if c.sArrCache == nil {
			c.sArrCache = make(map[string]strArr)
		}
		c.sArrCache[key] = arr
	}
	return
}

// StringsBySplit get []string by split a string value.
func StringsBySplit(key, sep string) []string { return dc.StringsBySplit(key, sep) }

// StringsBySplit get []string by split a string value.
func (c *Config) StringsBySplit(key, sep string) (ss []string) {
	if str, ok := c.getString(key); ok {
		ss = strutil.Split(str, sep)
	}
	return
}

// StringMap get config data as a map[string]string
func StringMap(key string) map[string]string { return dc.StringMap(key) }

// StringMap get config data as a map[string]string
func (c *Config) StringMap(key string) (mp map[string]string) {
	var ok bool

	// find from cache
	if c.opts.EnableCache && len(c.sMapCache) > 0 {
		mp, ok = c.sMapCache[key]
		if ok {
			return
		}
	}

	rawVal, ok := c.GetValue(key)
	if !ok {
		return
	}

	switch typeData := rawVal.(type) {
	case map[string]string: // from Set
		mp = typeData
	case map[string]any: // decode from json,toml,yaml.v3
		mp = make(map[string]string, len(typeData))

		for k, v := range typeData {
			switch tv := v.(type) {
			case string:
				if c.opts.ParseEnv {
					mp[k] = envutil.ParseEnvValue(tv)
				} else {
					mp[k] = tv
				}
			default:
				mp[k] = strutil.QuietString(v)
			}
		}
	case map[any]any: // decode from yaml v2
		mp = make(map[string]string, len(typeData))

		for k, v := range typeData {
			sk := strutil.QuietString(k)

			switch typVal := v.(type) {
			case string:
				if c.opts.ParseEnv {
					mp[sk] = envutil.ParseEnvValue(typVal)
				} else {
					mp[sk] = typVal
				}
			default:
				mp[sk] = strutil.QuietString(v)
			}
		}
	default:
		c.addErrorf("value cannot be convert to map[string]string, key is %q", key)
		return
	}

	// add cache
	if c.opts.EnableCache {
		if c.sMapCache == nil {
			c.sMapCache = make(map[string]strMap)
		}
		c.sMapCache[key] = mp
	}
	return
}

// SubDataMap get sub config data as maputil.Map
func SubDataMap(key string) maputil.Map { return dc.SubDataMap(key) }

// SubDataMap get sub config data as maputil.Map
//
// TIP: will not enable parse Env and more
func (c *Config) SubDataMap(key string) maputil.Map {
	if mp, ok := c.GetValue(key); ok {
		if mmp, ok := mp.(map[string]any); ok {
			return mmp
		}
	}

	// keep is not nil
	return maputil.Map{}
}
