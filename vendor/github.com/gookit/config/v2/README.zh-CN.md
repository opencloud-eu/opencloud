# Config

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gookit/config?style=flat-square)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/1e0f0ca096d94ffdab375234ec4167ee)](https://app.codacy.com/gh/gookit/config?utm_source=github.com&utm_medium=referral&utm_content=gookit/config&utm_campaign=Badge_Grade_Settings)
[![Build Status](https://travis-ci.org/gookit/config.svg?branch=master)](https://travis-ci.org/gookit/config)
[![Actions Status](https://github.com/gookit/config/workflows/Unit-Tests/badge.svg)](https://github.com/gookit/config/actions)
[![Coverage Status](https://coveralls.io/repos/github/gookit/config/badge.svg?branch=master)](https://coveralls.io/github/gookit/config?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gookit/config)](https://goreportcard.com/report/github.com/gookit/config)
[![Go Reference](https://pkg.go.dev/badge/github.com/gookit/config/v2.svg)](https://pkg.go.dev/github.com/gookit/config/v2)

`config` - 简洁、功能完善的 Go 应用程序配置管理工具库

> **[EN README](README.md)**

## 功能简介

- 支持多种格式: `JSON`(默认), `JSON5`, `INI`, `Properties`, `YAML`, `TOML`, `HCL`, `ENV`, `Flags`
  - `JSON` 内容支持注释，可以设置解析时清除注释
  - 其他驱动都是按需使用，不使用的不会加载编译到应用中
- 支持多个文件、多数据加载
- 支持从 OS ENV 变量数据加载配置
- 支持从远程 URL 加载配置数据
- 支持从命令行参数(`flags`)设置配置数据
- 数据自动覆盖合并，加载多份数据时将按`key`自动合并
- 支持丰富的自定义选项设置
  - `Readonly` 支持设置配置数据只读
  - `EnableCache` 支持设置配置数据缓存
  - `ParseEnv` 支持获取时自动解析string值里的ENV变量(`shell: ${SHELL}` -> `shell: /bin/zsh`)
  - `ParseDefault` 支持在绑定数据到结构体时解析默认值 (tag: `default:"def_value"`, 配合`ParseEnv`也支持ENV变量)
  - `ParseTime` 支持绑定数据到struct时自动转换 `10s`,`2m` 为 `time.Duration`
  - 完整选项设置请查看 `config.Options`
- 支持将全部或部分配置数据绑定到结构体 `config.BindStruct("key", &s)`
  - 支持通过结构体标签 `default` 解析并设置默认值. eg: `default:"def_value"`
  - 支持从 ENV 初始化设置字段值 `default:"${APP_ENV | dev}"`
- 支持通过 `.` 分隔符来按路径获取子级值，也支持自定义分隔符。 e.g `map.key` `arr.2`
- 支持在配置数据更改时触发事件
  - 可用事件: `set.value`, `set.data`, `load.data`, `clean.data`, `reload.data`
- 简洁的使用API `Get` `Int` `Uint` `Int64` `String` `Bool` `Ints` `IntMap` `Strings` `StringMap` ...
- 完善的单元测试(code coverage > 95%)

## 只使用INI

如果你仅仅想用INI来做简单配置管理，推荐使用 [gookit/ini](https://github.com/gookit/ini)

### 加载 .env 文件

`gookit/ini`: 提供一个子包 `dotenv`，支持从文件（eg `.env`）中导入数据到ENV

```shell
go get github.com/gookit/ini/v2/dotenv
```

## GoDoc

- [godoc for github](https://godoc.org/github.com/gookit/config)

## 安装包

```bash
go get github.com/gookit/config/v2
```

## 快速使用

这里使用yaml格式内容作为示例(`testdata/yml_other.yml`):

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

### 载入数据

> 示例代码请看 [_examples/yaml.go](_examples/yaml.go):

```go
package main

import (
    "github.com/gookit/config/v2"
    "github.com/gookit/config/v2/yaml"
)

// go run ./examples/yaml.go
func main() {
	// 设置选项支持ENV变量解析：当获取的值为string类型时，会尝试解析其中的ENV变量
	config.WithOptions(config.ParseEnv)

	// 添加驱动程序以支持yaml内容解析（除了JSON是默认支持，其他的则是按需使用）
	config.AddDriver(yaml.Driver)

	// 加载配置，可以同时传入多个文件
	err := config.LoadFiles("testdata/yml_base.yml")
	if err != nil {
		panic(err)
	}

	// fmt.Printf("config data: \n %#v\n", config.Data())

	// 加载更多文件
	err = config.LoadFiles("testdata/yml_other.yml")
	// 也可以一次性加载多个文件
	// err := config.LoadFiles("testdata/yml_base.yml", "testdata/yml_other.yml")
	if err != nil {
		panic(err)
	}
}
```

**使用提示**:

- 可以使用 `WithOptions()` 添加更多额外选项功能. 例如: `ParseEnv`, `ParseDefault`
- 可以使用 `AddDriver()` 添加需要使用的格式驱动器(`json` 是默认加载的的,无需添加)
- 然后就可以使用 `LoadFiles()` `LoadStrings()` 等方法加载配置数据
  - 可以传入多个文件,也可以调用多次
  - 多次加载的数据会自动按key进行合并处理

### 绑定数据到结构体

> 注意：结构体默认的绑定映射tag是 `mapstructure`，可以通过设置 `Options.TagName` 来更改它

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

**更改结构标签名称**

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

将所有配置数据绑定到结构:

```go
config.Decode(&myConf)
// 也可以
config.BindStruct("", &myConf)
```

> `config.MapOnExists` 与 `BindStruct` 一样，但仅当 key 存在时才进行映射绑定

### 快速获取数据

```go
// 获取整型
age := config.Int("age")
fmt.Print(age) // 100

// 获取布尔值
val := config.Bool("debug")
fmt.Print(val) // true

// 获取字符串
name := config.String("name")
fmt.Print(name) // inhere

// 获取字符串数组
arr1 := config.Strings("arr1")
fmt.Printf("%#v", arr1) // []string{"val1", "val21"}

// 获取字符串KV映射
val := config.StringMap("map1")
fmt.Printf("%#v",val) // map[string]string{"key":"val2", "key2":"val20"}

// 值包含ENV变量
value := config.String("shell")
fmt.Print(value) // /bin/zsh

// 通过key路径获取值
// from array
value := config.String("arr1.0")
fmt.Print(value) // "val1"

// from map
value := config.String("map1.key")
fmt.Print(value) // "val2"
```

### 设置新的值

```go
// set value
config.Set("name", "new name")
// get
name = config.String("name")
fmt.Print(name) // new name
```

## 加载配置文件

- `LoadExists(sourceFiles ...string) (err error)` 从存在的配置文件里加载数据，会忽略不存在的文件
- `LoadFiles(sourceFiles ...string) (err error)` 从给定的配置文件里加载数据，有文件不存在则会panic

> **TIP**: 更多加载方式请查看 `config.Load*` 相关方法

## 从ENV载入数据

`LoadOSEnvs` 支持从环境变量中读取数据，并解析为配置数据。格式为 `ENV_NAME: config_key`

- `config_key` 可以是 key path 格式。 eg: `{"DB_USERNAME": "db.username"}` 值将会映射到 `db` 配置的 `username`

```go
// os env: APP_NAME=config APP_DEBUG=true
// load ENV info
config.LoadOSEnvs(map[string]string{"APP_NAME": "app_name", "APP_DEBUG": "app_debug"})

// read
config.Bool("app_debug") // true
config.String("app_name") // "config"
```

## 从命令行参数载入数据

支持简单的从命令行 `flag` 参数解析，加载数据。

- 配置参数格式为 `name:type:desc` OR `name:type` OR `name:desc` (type, desc 是可选的)
  - `type` 可以设置 `flag` 的类型，支持 `bool`, `int`, `string`(默认)
  - `desc` 可以设置 `flag` 的描述信息
- `name` 可以是 key path 格式。 eg: `db.username`, input: `--db.username=someone` 值将会映射到 `db` 配置的 `username`

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

## 创建自定义实例

您可以创建自定义配置实例：

```go
// create new instance, will auto register JSON driver
myConf := config.New("my-conf")

// create empty instance
myConf := config.NewEmpty("my-conf")

// create and with some options
myConf := config.NewWithOptions("my-conf", config.ParseEnv, config.ReadOnly)
```

## 监听配置更改

现在，您可以添加一个钩子函数来监听配置数据更改。然后，您可以执行一些自定义操作, 例如：将数据写入文件

**在创建配置时添加钩子函数**:

```go
hookFn := func(event string, c *Config) {
    fmt.Println("fire the:", event)
}

c := NewWithOptions("test", WithHookFunc(hookFn))
// for global config
config.WithOptions(WithHookFunc(hookFn))
```

**之后**, 当调用 `LoadXXX, Set, SetData, ClearData` 等方法时, 就会输出:

```text
fire the: load.data
fire the: set.value
fire the: set.data
fire the: clean.data
```

### 监听载入的配置文件变动

想要监听载入的配置文件变动，并在变动时重新加载配置，你需要使用 https://github.com/fsnotify/fsnotify 库。
使用方法可以参考示例 [./_example/watch_file.go](_examples/watch_file.go)

同时，你需要监听 `reload.data` 事件:

```go
config.WithOptions(config.WithHookFunc(func(event string, c *config.Config) {
    if event == config.OnReloadData {
        fmt.Println("config reloaded, you can do something ....")
    }
}))
```

当配置发生变化并重新加载后，你可以做相关的事情，例如：重新绑定配置到你的结构体。

## 导出配置到文件

> 可以使用 `config.DumpTo(out io.Writer, format string)` 将整个配置数据导出到指定的writer, 比如 buffer,file。

**示例:导出为JSON文件**

```go
buf := new(bytes.Buffer)

_, err := config.DumpTo(buf, config.JSON)
ioutil.WriteFile("my-config.json", buf.Bytes(), 0755)
```

**示例:导出格式化的JSON**

可以设置默认变量 `JSONMarshalIndent` 的值 或 自定义新的 JSON 驱动程序。

```go
config.JSONMarshalIndent = "    "
```

**示例:导出为YAML文件**

```go
_, err := config.DumpTo(buf, config.YAML)
ioutil.WriteFile("my-config.yaml", buf.Bytes(), 0755)
```

## 可用选项

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
	// the delimiter char for split key, when `FindByPath=true`. default is '.'
	Delimiter byte
	// default write format. default is JSON
	DumpFormat string
	// default input format. default is JSON
	ReadFormat string
    // DecoderConfig setting for binding data to struct
    DecoderConfig *mapstructure.DecoderConfig
    // HookFunc on data changed.
    HookFunc HookFunc
	// ParseDefault tag on binding data to struct. tag: default
	ParseDefault bool
}
```

> **提示**: 访问 https://pkg.go.dev/github.com/gookit/config/v2#Options 查看最新的选项信息

Examples for set options:

```go
config.WithOptions(config.WithTagName("mytag"))
config.WithOptions(func(opt *Options) {
    opt.SetTagNames("config")
})
```

### 选项: 解析默认值

NEW: 支持通过结构标签 `default` 解析并设置默认值，支持嵌套解析处理。

> 注意 ⚠️ 如果想要解析子结构体字段，需要对父结构体设置 `default:""` 标记，否则不会解析子结构体的字段。

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

## API方法参考

### 载入配置

- `LoadData(dataSource ...any) (err error)` 从struct或map加载数据
- `LoadFlags(keys []string) (err error)` 从命令行参数载入数据
- `LoadOSEnvs(nameToKeyMap map[string]string)` 从ENV载入配置数据
- `LoadExists(sourceFiles ...string) (err error)` 从存在的配置文件里加载数据，会忽略不存在的文件
- `LoadFiles(sourceFiles ...string) (err error)` 从给定的配置文件里加载数据，有文件不存在则会panic
- `LoadFromDir(dirPath, format string) (err error)` 从给定目录里加载自定格式的文件,文件名会作为 key
- `LoadRemote(format, url string) (err error)` 从远程 URL 加载配置数据
- `LoadSources(format string, src []byte, more ...[]byte) (err error)` 从给定格式的字节数据加载配置
- `LoadStrings(format string, str string, more ...string) (err error)` 从给定格式的字符串配置里加载配置数据
- `LoadFilesByFormat(format string, sourceFiles ...string) (err error)` 从给定格式的文件加载配置
- `LoadExistsByFormat(format string, sourceFiles ...string) error` 从给定格式的文件加载配置，会忽略不存在的文件

### 获取值

- `Bool(key string, defVal ...bool) bool`
- `Int(key string, defVal ...int) int`
- `Uint(key string, defVal ...uint) uint`
- `Int64(key string, defVal ...int64) int64`
- `Ints(key string) (arr []int)`
- `IntMap(key string) (mp map[string]int)`
- `Float(key string, defVal ...float64) float64`
- `String(key string, defVal ...string) string`
- `Strings(key string) (arr []string)`
- `StringMap(key string) (mp map[string]string)`
- `SubDataMap(key string) maputi.Data`
- `Get(key string, findByPath ...bool) (value any)`

**将数据映射到结构体:**

- `BindStruct(key string, dst any) error`
- `MapOnExists(key string, dst any) error`

### 设置值

- `Set(key string, val any, setByPath ...bool) (err error)`

### 有用的方法

- `Getenv(name string, defVal ...string) (val string)`
- `AddDriver(driver Driver)`
- `Data() map[string]any`
- `Exists(key string, findByPath ...bool) bool`
- `DumpTo(out io.Writer, format string) (n int64, err error)`
- `SetData(data map[string]any)` 设置数据以覆盖 `Config.Data`

## 单元测试

```bash
go test -cover
// contains all sub-folder
go test -cover ./...
```

## 使用Config的项目

看看这些使用了 https://github.com/gookit/config 的项目:

- https://github.com/JanDeDobbeleer/oh-my-posh A prompt theme engine for any shell.
- [+ See More](https://pkg.go.dev/github.com/gookit/config?tab=importedby)

## Gookit 工具包

- [gookit/ini](https://github.com/gookit/ini) INI配置读取管理，支持多文件加载，数据覆盖合并, 解析ENV变量, 解析变量引用
- [gookit/rux](https://github.com/gookit/rux) Simple and fast request router for golang HTTP 
- [gookit/gcli](https://github.com/gookit/gcli) Go的命令行应用，工具库，运行CLI命令，支持命令行色彩，用户交互，进度显示，数据格式化显示
- [gookit/event](https://github.com/gookit/event) Go实现的轻量级的事件管理、调度程序库, 支持设置监听器的优先级, 支持对一组事件进行监听
- [gookit/cache](https://github.com/gookit/cache) 通用的缓存使用包装库，通过包装各种常用的驱动，来提供统一的使用API
- [gookit/config](https://github.com/gookit/config) Go应用配置管理，支持多种格式（JSON, YAML, TOML, INI, HCL, ENV, Flags），多文件加载，远程文件加载，数据合并
- [gookit/color](https://github.com/gookit/color) CLI 控制台颜色渲染工具库, 拥有简洁的使用API，支持16色，256色，RGB色彩渲染输出
- [gookit/filter](https://github.com/gookit/filter) 提供对Golang数据的过滤，净化，转换
- [gookit/validate](https://github.com/gookit/validate) Go通用的数据验证与过滤库，使用简单，内置大部分常用验证、过滤器
- [gookit/goutil](https://github.com/gookit/goutil) Go 的一些工具函数，格式化，特殊处理，常用信息获取等
- 更多请查看 https://github.com/gookit

## 相关包

- Ini 解析 [gookit/ini/parser](https://github.com/gookit/ini/tree/master/parser)
- Properties 解析 [gookit/properties](https://github.com/gookit/properties)
- Yaml 解析 [go-yaml](https://github.com/go-yaml/yaml)
- Toml 解析 [go toml](https://github.com/BurntSushi/toml)
- 数据合并 [mergo](https://github.com/imdario/mergo)
- 映射数据到结构体 [mapstructure](https://github.com/mitchellh/mapstructure)
- JSON5 解析
  - [yosuke-furukawa/json5](https://github.com/yosuke-furukawa/json5)
  - [titanous/json5](https://github.com/titanous/json5)

## License

**MIT**
