# Config

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gookit/config?style=flat-square)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/1e0f0ca096d94ffdab375234ec4167ee)](https://app.codacy.com/gh/gookit/config?utm_source=github.com&utm_medium=referral&utm_content=gookit/config&utm_campaign=Badge_Grade_Settings)
[![Build Status](https://travis-ci.org/gookit/config.svg?branch=master)](https://travis-ci.org/gookit/config)
[![Actions Status](https://github.com/gookit/config/workflows/Unit-Tests/badge.svg)](https://github.com/gookit/config/actions)
[![Coverage Status](https://coveralls.io/repos/github/gookit/config/badge.svg?branch=master)](https://coveralls.io/github/gookit/config?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gookit/config)](https://goreportcard.com/report/github.com/gookit/config)
[![Go Reference](https://pkg.go.dev/badge/github.com/gookit/config/v2.svg)](https://pkg.go.dev/github.com/gookit/config/v2)

`config` - Simple, full-featured Go application configuration management tool library.

> **[中文说明](README.zh-CN.md)**

## Features

- Support multi format: `JSON`(default), `JSON5`, `INI`, `Properties`, `YAML`, `TOML`, `HCL`, `ENV`, `Flags`
  - `JSON` content support comments. will auto clear comments
  - Other drivers are used on demand, not used will not be loaded into the application.
    - Possibility to add custom driver for your specific format
- Support multi-file and multi-data loading
- Support for loading configuration from system ENV
- Support for loading configuration data from remote URLs
- Support for setting configuration data from command line(`flags`)
- Support listen and fire events on config data changed. 
  - allow events: `set.value`, `set.data`, `load.data`, `clean.data`, `reload.data`
- Support data overlay and merge, automatically load by key when loading multiple copies of data
- Support for binding all or part of the configuration data to the structure
  - Support init default value by struct tag `default:"def_value"`
  - Support init default value from ENV `default:"${APP_ENV | dev}"`
- Support get sub value by key-path, like `map.key` `arr.2`
- Support parse ENV name and allow with default value. like `envKey: ${SHELL|/bin/bash}` -> `envKey: /bin/zsh`
- Generic API: `Get` `Int` `Uint` `Int64` `Float` `String` `Bool` `Ints` `IntMap` `Strings` `StringMap` ...
- Complete unit test(code coverage > 95%)

## Only use INI

If you just want to use INI for simple config management, recommended use [gookit/ini](https://github.com/gookit/ini)

### Load dotenv file

On `gookit/ini`:  Provide a sub-package `dotenv` that supports importing data from files (eg `.env`) to ENV

```shell
go get github.com/gookit/ini/v2/dotenv
```

## GoDoc

- [godoc for github](https://pkg.go.dev/github.com/gookit/config)

## Install

```bash
go get github.com/gookit/config/v2
```

## Usage

Here using the yaml format as an example(`testdata/yml_other.yml`):

```yaml
name: app2
debug: false
baseKey: value2
shell: ${SHELL}
envKey1: ${NotExist|defValue}

map1:
    key: val2
    key2: val20

arr1:
    - val1
    - val21
```

### Load data

> examples code please see [_examples/yaml.go](_examples/yaml.go):

```go
package main

import (
    "github.com/gookit/config/v2"
    "github.com/gookit/config/v2/yaml"
)

// go run ./examples/yaml.go
func main() {
	// config.ParseEnv: will parse env var in string value. eg: shell: ${SHELL}
	config.WithOptions(config.ParseEnv)

	// add driver for support yaml content
	config.AddDriver(yaml.Driver)

	err := config.LoadFiles("testdata/yml_base.yml")
	if err != nil {
		panic(err)
	}

	// load more files
	err = config.LoadFiles("testdata/yml_other.yml")
	// can also load multi at once
	// err := config.LoadFiles("testdata/yml_base.yml", "testdata/yml_other.yml")
	if err != nil {
		panic(err)
	}

	// fmt.Printf("config data: \n %#v\n", config.Data())
}
```
**Usage tips**:

- More extra options can be added using `WithOptions()`. For example: `ParseEnv`, `ParseDefault`
- You can use `AddDriver()` to add the required format driver (`json` is loaded by default, no need to add)
- The configuration data can then be loaded using `LoadFiles()` `LoadStrings()` etc.
  - You can pass in multiple files or call multiple times
  - Data loaded multiple times will be automatically merged by key

## Bind Structure

> Note: The default binding mapping tag of a structure is `mapstructure`, which can be changed by setting the decoder's option `options.DecoderConfig.TagName`

```go
type User struct {
    Age  int  `mapstructure:"age"`
    Key  string `mapstructure:"key"`
    UserName  string `mapstructure:"user_name"`
    Tags []int  `mapstructure:"tags"`
}

user := User{}
err = config.BindStruct("user", &user)

fmt.Println(user.UserName) // inhere
```

**Change struct tag name**

```go
config.WithOptions(func(opt *Options) {
    options.DecoderConfig.TagName = "config"
})

// use custom tag name.
type User struct {
  Age  int  `config:"age"`
  Key  string `config:"key"`
  UserName  string `config:"user_name"`
  Tags []int  `config:"tags"`
}

user := User{}
err = config.Decode(&user)
```

**Can bind all config data to a struct**:

```go
config.Decode(&myConf)
// can also
config.BindStruct("", &myConf)
```

> `config.MapOnExists` like `BindStruct`，but map binding only if key exists

### Direct read data

- Get integer

```go
age := config.Int("age")
fmt.Print(age) // 100
```

- Get bool

```go
val := config.Bool("debug")
fmt.Print(val) // true
```

- Get string

```go
name := config.String("name")
fmt.Print(name) // inhere
```

- Get strings(slice)

```go
arr1 := config.Strings("arr1")
fmt.Printf("%#v", arr1) // []string{"val1", "val21"}
```

- Get string map

```go
val := config.StringMap("map1")
fmt.Printf("%#v",val) // map[string]string{"key":"val2", "key2":"val20"}
```

- Value contains ENV var

```go
value := config.String("shell")
fmt.Print(value) // "/bin/zsh"
```

- Get value by key path

```go
// from array
value := config.String("arr1.0")
fmt.Print(value) // "val1"

// from map
value := config.String("map1.key")
fmt.Print(value) // "val2"
```

- Setting new value

```go
// set value
config.Set("name", "new name")
name = config.String("name")
fmt.Print(name) // "new name"
```

## Load from ENV

Support load ENV vars to config data.

- Support set value to sub key in map.
  - eg: `{"DB_USERNAME": "db.username"}` value will set to `username` in `db`

```go
// os env: APP_NAME=config APP_DEBUG=true DB_USERNAME=someone

// load ENV info
config.LoadOSEnvs(map[string]string{"APP_NAME": "app_name", "APP_DEBUG": "app_debug", "DB_USERNAME": "db.username"})

// read
config.Bool("app_debug") // true
config.String("app_name") // "config"
```

## Load from flags

Support simple CLI flags parameter parsing, load to config data.

- define format: `name:type:desc` OR `name:type` OR `name:desc` (type, desc is optional)
  - `type` can set `flag` type. allow: `bool`, `int`, `string`(default)
  - `desc` can set `flag` description
- `name` can be in key path format. 
  - eg: `db.username`, input: `--db.username=someone` values will be mapped to `username` of the `db` configuration

```go
// 'debug' flag is bool type
config.LoadFlags([]string{"env", "debug:bool"})
// can with flag desc message
config.LoadFlags([]string{"env:set the run env"})
config.LoadFlags([]string{"debug:bool:set debug mode"})
// can set value to map key. eg: myapp --map1.sub-key=val
config.LoadFlags([]string{"map1.sub-key"})
```

Examples:

```go
// flags like: --name inhere --env dev --age 99 --debug --map1.sub-key=val

// load flag info
keys := []string{
	"name",
	"env:set the run env",
	"age:int",
	"debug:bool:set debug mode",
	"map1.sub-key",
}
err := config.LoadFlags(keys)

// read
config.String("name") // "inhere"
config.String("env") // "dev"
config.Int("age") // 99
config.Bool("debug") // true
config.Get("map1") // map[string]any{"sub-key":"val"}
```

## New config instance

You can create custom config instance

```go
// create new instance, will auto register JSON driver
myConf := config.New("my-conf")

// create empty instance
myConf := config.NewEmpty("my-conf")

// create and with some options
myConf := config.NewWithOptions("my-conf", config.ParseEnv, config.ReadOnly)
```

## Listen config change

Now, you can add a hook func for listen config data change. then, you can do something like: write data to file

**Add hook func on create config**:

```go
hookFn := func(event string, c *Config) {
    fmt.Println("fire the:", event)
}

c := NewWithOptions("test", config.WithHookFunc(hookFn))
// for global config
config.WithOptions(config.WithHookFunc(hookFn))
```

After that, when calling `LoadXXX, Set, SetData, ClearData` methods, it will output:

```text
fire the: load.data
fire the: set.value
fire the: set.data
fire the: clean.data
```

### Watch loaded config files

To listen for changes to loaded config files, and reload the config when it changes, you need to use the https://github.com/fsnotify/fsnotify library. 
For usage, please refer to the example [./_example/watch_file.go](_examples/watch_file.go)

Also, you need to listen to the `reload.data` event:

```go
config.WithOptions(config.WithHookFunc(func(event string, c *config.Config) {
    if event == config.OnReloadData {
        fmt.Println("config reloaded, you can do something ....")
    }
}))
```

When the configuration changes, you can do related things, for example: rebind the configuration to your struct.

## Dump config data

> Can use `config.DumpTo()` export the configuration data to the specified `writer`, such as: buffer,file

**Dump to JSON file**

```go
buf := new(bytes.Buffer)

_, err := config.DumpTo(buf, config.JSON)
ioutil.WriteFile("my-config.json", buf.Bytes(), 0755)
```

**Dump pretty JSON**

You can set the default var `JSONMarshalIndent` or custom a new JSON driver. 

```go
config.JSONMarshalIndent = "    "
```

**Dump to YAML file**

```go
_, err := config.DumpTo(buf, config.YAML)
ioutil.WriteFile("my-config.yaml", buf.Bytes(), 0755)
```

## Available options

```go
// Options config options
type Options struct {
	// parse env in string value. like: "${EnvName}" "${EnvName|default}"
	ParseEnv bool
    // ParseTime parses a duration string to time.Duration
    // eg: 10s, 2m
    ParseTime bool
	// config is readonly. default is False
	Readonly bool
	// enable config data cache. default is False
	EnableCache bool
	// parse key, allow find value by key path. default is True eg: 'key.sub' will find `map[key]sub`
	ParseKey bool
	// the delimiter char for split key path, if `FindByPath=true`. default is '.'
	Delimiter byte
	// default write format
	DumpFormat string
	// default input format
	ReadFormat string
	// DecoderConfig setting for binding data to struct
	DecoderConfig *mapstructure.DecoderConfig
	// HookFunc on data changed.
	HookFunc HookFunc
	// ParseDefault tag on binding data to struct. tag: default
	ParseDefault bool
}
```

> **TIP**: please visit https://pkg.go.dev/github.com/gookit/config/v2#Options to see the latest options information

Examples for set options:

```go
config.WithOptions(config.WithTagName("mytag"))
config.WithOptions(func(opt *Options) {
    opt.SetTagNames("config")
})
```

### Options: Parse default

Support parse default value by struct tag `default`, and support parse fields in sub struct.

> **NOTE**⚠️ If you want to parse a sub-struct, you need to set the `default:""` flag on the parent struct, 
> otherwise the fields of the sub-struct will not be resolved.

```go
// add option: config.ParseDefault
c := config.New("test").WithOptions(config.ParseDefault)

// only set name
c.SetData(map[string]any{
    "name": "inhere",
})

// age load from default tag
type User struct {
    Age  int `default:"30"`
    Name string
    Tags []int
}

user := &User{}
goutil.MustOk(c.Decode(user))
dump.Println(user)
```

**Output**:

```shell
&config_test.User {
  Age: int(30),
  Name: string("inhere"), #len=6
  Tags: []int [ #len=0
  ],
},
```

## API Methods Refer

### Load Config

- `LoadOSEnvs(nameToKeyMap map[string]string)` Load data from os ENV
- `LoadData(dataSource ...any) (err error)` Load from struts or maps
- `LoadFlags(keys []string) (err error)` Load from CLI flags
- `LoadExists(sourceFiles ...string) (err error)` 
- `LoadFiles(sourceFiles ...string) (err error)`
- `LoadFromDir(dirPath, format string) (err error)` Load custom format files from the given directory, the file name will be used as the key
- `LoadRemote(format, url string) (err error)`
- `LoadSources(format string, src []byte, more ...[]byte) (err error)`
- `LoadStrings(format string, str string, more ...string) (err error)`
- `LoadFilesByFormat(format string, sourceFiles ...string) (err error)`
- `LoadExistsByFormat(format string, sourceFiles ...string) error`

### Getting Values

- `Bool(key string, defVal ...bool) bool`
- `Int(key string, defVal ...int) int`
- `Uint(key string, defVal ...uint) uint`
- `Int64(key string, defVal ...int64) int64`
- `Ints(key string) (arr []int)`
- `IntMap(key string) (mp map[string]int)`
- `Float(key string, defVal ...float64) float64`
- `String(key string, defVal ...string) string`
- `Strings(key string) (arr []string)`
- `SubDataMap(key string) maputi.Data`
- `StringMap(key string) (mp map[string]string)`
- `Get(key string, findByPath ...bool) (value any)`

**Mapping data to struct:**

- `Decode(dst any) error`
- `BindStruct(key string, dst any) error`
- `MapOnExists(key string, dst any) error`

### Setting Values

- `Set(key string, val any, setByPath ...bool) (err error)`

### Useful Methods

- `Getenv(name string, defVal ...string) (val string)`
- `AddDriver(driver Driver)`
- `Data() map[string]any`
- `SetData(data map[string]any)` set data to override the Config.Data
- `Exists(key string, findByPath ...bool) bool`
- `DumpTo(out io.Writer, format string) (n int64, err error)`

## Run Tests

```bash
go test -cover
// contains all sub-folder
go test -cover ./...
```

## Projects using config

Check out these projects, which use https://github.com/gookit/config :

- https://github.com/JanDeDobbeleer/oh-my-posh A prompt theme engine for any shell.
- [+ See More](https://pkg.go.dev/github.com/gookit/config?tab=importedby)

## Gookit packages

- [gookit/ini](https://github.com/gookit/ini) Go config management, use INI files
- [gookit/rux](https://github.com/gookit/rux) Simple and fast request router for golang HTTP 
- [gookit/gcli](https://github.com/gookit/gcli) build CLI application, tool library, running CLI commands
- [gookit/event](https://github.com/gookit/event) Lightweight event manager and dispatcher implements by Go
- [gookit/cache](https://github.com/gookit/cache) Generic cache use and cache manager for golang. support File, Memory, Redis, Memcached.
- [gookit/config](https://github.com/gookit/config) Go config management. support JSON, YAML, TOML, INI, HCL, ENV and Flags
- [gookit/color](https://github.com/gookit/color) A command-line color library with true color support, universal API methods and Windows support
- [gookit/filter](https://github.com/gookit/filter) Provide filtering, sanitizing, and conversion of golang data
- [gookit/validate](https://github.com/gookit/validate) Use for data validation and filtering. support Map, Struct, Form data
- [gookit/goutil](https://github.com/gookit/goutil) Some utils for the Go: string, array/slice, map, format, cli, env, filesystem, test and more
- More, please see https://github.com/gookit

## See also

- Ini parser [gookit/ini/parser](https://github.com/gookit/ini/tree/master/parser)
- Properties parser [gookit/properties](https://github.com/gookit/properties)
- Json5 parser
  - [yosuke-furukawa/json5](https://github.com/yosuke-furukawa/json5)
  - [titanous/json5](https://github.com/titanous/json5)
- Json parser
  - [goccy/go-json](https://github.com/goccy/go-json)
  - [json-iterator/go](https://github.com/json-iterator/go)
- Yaml parser
  - [goccy/go-yaml](https://github.com/goccy/go-yaml)
  - [go-yaml/yaml](https://github.com/go-yaml/yaml)
- Toml parser [go toml](https://github.com/BurntSushi/toml)
- Data merge [mergo](https://github.com/imdario/mergo)
- Map structure [mapstructure](https://github.com/mitchellh/mapstructure)

## License

**MIT**
