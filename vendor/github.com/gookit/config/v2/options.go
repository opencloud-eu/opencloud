package config

import (
	"strings"

	"dario.cat/mergo"
	"github.com/go-viper/mapstructure/v2"
	"github.com/gookit/goutil"
)

// there are some event names for config data changed.
const (
	OnSetValue   = "set.value"
	OnSetData    = "set.data"
	OnLoadData   = "load.data"
	OnReloadData = "reload.data"
	OnCleanData  = "clean.data"
)

// HookFunc on config data changed.
type HookFunc func(event string, c *Config)

// Options config options
type Options struct {
	// ParseEnv parse env in string value and default value. default: false
	//
	//  - like: "${EnvName}" "${EnvName|default}"
	ParseEnv bool
	// ParseTime parses a duration string to `time.Duration`. default: false
	//
	// eg: 10s, 2m
	ParseTime bool
	// ParseDefault tag on binding data to struct. default: false
	//
	//  - tag: default
	//
	// NOTE: If you want to parse a substruct, you need to set the `default:""` flag on the struct,
	// otherwise the fields that will not resolve to it will not be resolved.
	ParseDefault bool
	// Readonly config is readonly. default: false
	Readonly bool
	// EnableCache enable config data cache. default: false
	EnableCache bool
	// ParseKey support key path, allow finding value by key path. default: true
	//
	// - eg: 'key.sub' will find `map[key]sub`
	ParseKey bool
	// TagName tag name for binding data to struct
	//
	// Deprecated: please set tag name by DecoderConfig, or use SetTagName()
	TagName string
	// Delimiter the delimiter char for split key path, on `ParseKey=true`.
	//
	// - default is '.'
	Delimiter byte
	// DumpFormat default write format. default is 'json'
	DumpFormat string
	// ReadFormat default input format. default is 'json'
	ReadFormat string
	// DecoderConfig setting for binding data to struct. such as: TagName
	DecoderConfig *mapstructure.DecoderConfig
	// MergeOptions settings for merge two data
	MergeOptions []func(*mergo.Config)
	// HookFunc on data changed. you can do something...
	HookFunc HookFunc
	// WatchChange bool
}

// OptionFn option func
type OptionFn func(*Options)

func newDefaultOption() *Options {
	return &Options{
		ParseKey:  true,
		TagName:   defaultStructTag,
		Delimiter: defaultDelimiter,
		// for export
		DumpFormat: JSON,
		ReadFormat: JSON,
		// struct decoder config
		DecoderConfig: newDefaultDecoderConfig(""),
		MergeOptions: []func(*mergo.Config){
			mergo.WithOverride,
			mergo.WithTypeCheck,
		},
	}
}

func newDefaultDecoderConfig(tagName string) *mapstructure.DecoderConfig {
	if tagName == "" {
		tagName = defaultStructTag
	}

	return &mapstructure.DecoderConfig{
		// tag name for binding struct
		TagName: tagName,
		// will auto convert string to int/uint
		WeaklyTypedInput: true,
	}
}

// SetTagName for mapping data to struct
func (o *Options) SetTagName(tagName string) {
	o.TagName = tagName
	o.DecoderConfig.TagName = tagName
}

func (o *Options) shouldAddHookFunc() bool {
	return o.ParseTime || o.ParseEnv
}

func (o *Options) makeDecoderConfig() *mapstructure.DecoderConfig {
	var bindConf *mapstructure.DecoderConfig
	if o.DecoderConfig == nil {
		bindConf = newDefaultDecoderConfig(o.TagName)
	} else {
		// copy new config for each binding.
		copyConf := *o.DecoderConfig
		bindConf = &copyConf

		// compatible with previous settings opts.TagName
		if bindConf.TagName == "" {
			bindConf.TagName = o.TagName
		}
	}

	// add hook on decode value to struct
	if bindConf.DecodeHook == nil && o.shouldAddHookFunc() {
		bindConf.DecodeHook = ValDecodeHookFunc(o.ParseEnv, o.ParseTime)
	}

	return bindConf
}

/*************************************************************
 * config setting
 *************************************************************/

// WithTagName set tag name for export to struct
func WithTagName(tagName string) func(*Options) {
	return func(opts *Options) {
		opts.SetTagName(tagName)
	}
}

// ParseEnv set parse env value
func ParseEnv(opts *Options) { opts.ParseEnv = true }

// ParseTime set parse time string.
func ParseTime(opts *Options) { opts.ParseTime = true }

// ParseDefault tag value on binding data to struct.
func ParseDefault(opts *Options) { opts.ParseDefault = true }

// Readonly set readonly
func Readonly(opts *Options) { opts.Readonly = true }

// Delimiter set delimiter char
func Delimiter(sep byte) func(*Options) {
	return func(opts *Options) {
		opts.Delimiter = sep
	}
}

// SaveFileOnSet set hook func, will panic on save error
func SaveFileOnSet(fileName string, format string) func(options *Options) {
	return func(opts *Options) {
		opts.HookFunc = func(event string, c *Config) {
			if strings.HasPrefix(event, "set.") {
				goutil.PanicErr(c.DumpToFile(fileName, format))
			}
		}
	}
}

// WithHookFunc set hook func
func WithHookFunc(fn HookFunc) func(*Options) {
	return func(opts *Options) {
		opts.HookFunc = fn
	}
}

// EnableCache set readonly
func EnableCache(opts *Options) { opts.EnableCache = true }

// WithOptions with options
func WithOptions(opts ...OptionFn) { dc.WithOptions(opts...) }

// WithOptions apply some options
func (c *Config) WithOptions(opts ...OptionFn) *Config {
	if !c.IsEmpty() {
		panic("config: Cannot set options after data has been loaded")
	}

	// apply options
	for _, opt := range opts {
		opt(c.opts)
	}
	return c
}

// GetOptions get options
func GetOptions() *Options { return dc.Options() }

// Options get
func (c *Config) Options() *Options {
	return c.opts
}

// With apply some options
func (c *Config) With(fn func(c *Config)) *Config {
	fn(c)
	return c
}

// Readonly disable set data to config.
//
// Usage:
//
//	config.LoadFiles(a, b, c)
//	config.Readonly()
func (c *Config) Readonly() {
	c.opts.Readonly = true
}
