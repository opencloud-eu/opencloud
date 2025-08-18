package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/go-viper/mapstructure/v2"
	"github.com/gookit/goutil/structs"
)

// Decode all config data to the dst ptr
//
// Usage:
//
//	myConf := &MyConf{}
//	config.Decode(myConf)
func Decode(dst any) error { return dc.Decode(dst) }

// Decode all config data to the dst ptr.
//
// It's equals:
//
//	c.Structure("", dst)
func (c *Config) Decode(dst any) error {
	return c.Structure("", dst)
}

// MapStruct alias method of the 'Structure'
//
// Usage:
//
//	dbInfo := &Db{}
//	config.MapStruct("db", dbInfo)
func MapStruct(key string, dst any) error { return dc.MapStruct(key, dst) }

// MapStruct alias method of the 'Structure'
func (c *Config) MapStruct(key string, dst any) error { return c.Structure(key, dst) }

// BindStruct alias method of the 'Structure'
func BindStruct(key string, dst any) error { return dc.BindStruct(key, dst) }

// BindStruct alias method of the 'Structure'
func (c *Config) BindStruct(key string, dst any) error { return c.Structure(key, dst) }

// MapOnExists mapping data to the dst structure only on key exists.
func MapOnExists(key string, dst any) error {
	return dc.MapOnExists(key, dst)
}

// MapOnExists mapping data to the dst structure only on key exists.
//
//   - Support ParseEnv on mapping
//   - Support ParseDefault on mapping
func (c *Config) MapOnExists(key string, dst any) error {
	err := c.Structure(key, dst)
	if err != nil && err == ErrNotFound {
		return nil
	}
	return err
}

// Structure get config data and binding to the dst structure.
//
//   - Support ParseEnv on mapping
//   - Support ParseDefault on mapping
//
// Usage:
//
//	dbInfo := Db{}
//	config.Structure("db", &dbInfo)
func (c *Config) Structure(key string, dst any) (err error) {
	var data any
	// binding all data on key is empty.
	if key == "" {
		// fix: if c.data is nil, don't need to apply map structure
		if len(c.data) == 0 {
			// init default value by tag: default
			if c.opts.ParseDefault {
				err = structs.InitDefaults(dst, func(opt *structs.InitOptions) {
					opt.ParseEnv = c.opts.ParseEnv
					opt.ParseTime = c.opts.ParseTime // add ParseTime support on parse default value
				})
			}
			return
		}

		data = c.data
	} else {
		// binding sub-data of the config
		var ok bool
		data, ok = c.GetValue(key)
		if !ok {
			return ErrNotFound
		}
	}

	// map structure from data
	bindConf := c.opts.makeDecoderConfig()
	// set result struct ptr
	bindConf.Result = dst
	decoder, err := mapstructure.NewDecoder(bindConf)
	if err == nil {
		if err = decoder.Decode(data); err != nil {
			return err
		}
	}

	// init default value by tag: default
	if c.opts.ParseDefault {
		err = structs.InitDefaults(dst, func(opt *structs.InitOptions) {
			opt.ParseEnv = c.opts.ParseEnv
			opt.ParseTime = c.opts.ParseTime
		})
	}
	return err
}

// ToJSON string, will ignore error
func (c *Config) ToJSON() string {
	buf := &bytes.Buffer{}

	_, err := c.DumpTo(buf, JSON)
	if err != nil {
		return ""
	}
	return buf.String()
}

// WriteTo a writer
func WriteTo(out io.Writer) (int64, error) { return dc.WriteTo(out) }

// WriteTo Write out config data representing the current state to a writer.
func (c *Config) WriteTo(out io.Writer) (n int64, err error) {
	return c.DumpTo(out, c.opts.DumpFormat)
}

// DumpTo a writer and use format
func DumpTo(out io.Writer, format string) (int64, error) { return dc.DumpTo(out, format) }

// DumpTo use the format(json,yaml,toml) dump config data to a writer
func (c *Config) DumpTo(out io.Writer, format string) (n int64, err error) {
	var ok bool
	var encoder Encoder

	format = c.resolveFormat(format)
	if encoder, ok = c.encoders[format]; !ok {
		err = errors.New("not exists/register encoder for the format: " + format)
		return
	}

	// is empty
	if len(c.data) == 0 {
		return
	}

	// encode data to string
	encoded, err := encoder(c.data)
	if err != nil {
		return
	}

	// write content to out
	num, _ := fmt.Fprintln(out, string(encoded))
	return int64(num), nil
}

// DumpToFile use the format(json,yaml,toml) dump config data to a writer
func (c *Config) DumpToFile(fileName string, format string) (err error) {
	fsFlags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	f, err := os.OpenFile(fileName, fsFlags, os.ModePerm)
	if err != nil {
		return err
	}

	_, err = c.DumpTo(f, format)
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}
