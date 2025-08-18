package config

import (
	"os"
	"reflect"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/gookit/goutil/envutil"
	"github.com/gookit/goutil/reflects"
)

// ValDecodeHookFunc returns a mapstructure.DecodeHookFunc
// that parse ENV var, and more custom parse
func ValDecodeHookFunc(parseEnv, parseTime bool) mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		var err error
		str := data.(string)
		if parseEnv {
			// https://docs.docker.com/compose/environment-variables/env-file/
			str, err = envutil.ParseOrErr(str)
			if err != nil {
				return nil, err
			}
		}
		if len(str) < 2 {
			return str, nil
		}

		// feat: support parse time or duration string. eg: 10s
		if parseTime && str[0] > '0' && str[0] <= '9' {
			return reflects.ToTimeOrDuration(str, t)
		}
		return str, nil
	}
}

// resolve format, check is alias
func (c *Config) resolveFormat(f string) string {
	if name, ok := c.aliasMap[f]; ok {
		return name
	}
	return f
}

/*************************************************************
 * Deprecated methods
 *************************************************************/

// SetDecoder add/set a format decoder
//
// Deprecated: please use driver instead
func SetDecoder(format string, decoder Decoder) {
	dc.SetDecoder(format, decoder)
}

// SetDecoder set decoder
//
// Deprecated: please use driver instead
func (c *Config) SetDecoder(format string, decoder Decoder) {
	format = c.resolveFormat(format)
	c.decoders[format] = decoder
}

// SetDecoders set decoders
//
// Deprecated: please use driver instead
func (c *Config) SetDecoders(decoders map[string]Decoder) {
	for format, decoder := range decoders {
		c.SetDecoder(format, decoder)
	}
}

// SetEncoder set a encoder for the format
//
// Deprecated: please use driver instead
func SetEncoder(format string, encoder Encoder) {
	dc.SetEncoder(format, encoder)
}

// SetEncoder set a encoder for the format
//
// Deprecated: please use driver instead
func (c *Config) SetEncoder(format string, encoder Encoder) {
	format = c.resolveFormat(format)
	c.encoders[format] = encoder
}

// SetEncoders set encoders
//
// Deprecated: please use driver instead
func (c *Config) SetEncoders(encoders map[string]Encoder) {
	for format, encoder := range encoders {
		c.SetEncoder(format, encoder)
	}
}

/*************************************************************
 * helper methods/functions
 *************************************************************/

// LoadENVFiles load
// func LoadENVFiles(filePaths ...string) error {
// 	return dotenv.LoadFiles(filePaths...)
// }

// GetEnv get os ENV value by name
func GetEnv(name string, defVal ...string) (val string) {
	return Getenv(name, defVal...)
}

// Getenv get os ENV value by name. like os.Getenv, but support default value
//
// Notice:
// - Key is not case-sensitive when getting
func Getenv(name string, defVal ...string) (val string) {
	if val = os.Getenv(name); val != "" {
		return
	}

	if len(defVal) > 0 {
		val = defVal[0]
	}
	return
}

func parseVarNameAndType(key string) (string, string, string) {
	var desc string
	typ := "string"
	key = strings.Trim(key, "-")

	// can set var type: int, uint, bool
	if strings.IndexByte(key, ':') > 0 {
		list := strings.SplitN(key, ":", 3)
		key, typ = list[0], list[1]
		if len(list) == 3 {
			desc = list[2]
		}

		// if type is not valid and has multi words, as desc message.
		if _, ok := validTypes[typ]; !ok {
			if desc == "" && strings.ContainsRune(typ, ' ') {
				desc = typ
			}
			typ = "string"
		}
	}
	return key, typ, desc
}

// format key
func formatKey(key, sep string) string {
	return strings.Trim(strings.TrimSpace(key), sep)
}
