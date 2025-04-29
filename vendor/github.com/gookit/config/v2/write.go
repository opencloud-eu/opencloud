package config

import (
	"errors"
	"strings"

	"github.com/gookit/goutil/maputil"
)

// some common errors definitions
var (
	ErrReadonly   = errors.New("the config instance in 'readonly' mode")
	ErrKeyIsEmpty = errors.New("the config key is cannot be empty")
	ErrNotFound   = errors.New("this key does not exist in the config")
)

// SetData for override the Config.Data
func SetData(data map[string]any) {
	dc.SetData(data)
}

// SetData for override the Config.Data
func (c *Config) SetData(data map[string]any) {
	c.lock.Lock()
	c.data = data
	c.lock.Unlock()

	c.fireHook(OnSetData)
}

// Set value by key. setByPath default is true
func Set(key string, val any, setByPath ...bool) error {
	return dc.Set(key, val, setByPath...)
}

// Set a value by key string. setByPath default is true
func (c *Config) Set(key string, val any, setByPath ...bool) (err error) {
	if c.opts.Readonly {
		return ErrReadonly
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	sep := c.opts.Delimiter
	if key = formatKey(key, string(sep)); key == "" {
		return ErrKeyIsEmpty
	}

	defer c.fireHook(OnSetValue)
	if strings.IndexByte(key, sep) == -1 {
		c.data[key] = val
		return
	}

	// disable set by path.
	if len(setByPath) > 0 && !setByPath[0] {
		c.data[key] = val
		return
	}

	// set by path
	keys := strings.Split(key, string(sep))
	return maputil.SetByKeys(&c.data, keys, val)
}
