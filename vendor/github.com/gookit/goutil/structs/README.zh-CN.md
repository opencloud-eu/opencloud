# Structs

`structs` 包为 go struct 提供一些有用的工具函数库。例如：convert、tag parse、struct data init

- 快速将结构体转换为 `map[string]any` 数据
- `structs.Aliases` - 实现了一个简单的字符串别名映射。
- 通过字段 `"default"` 标签快速初始化结构体默认值
- 通过map数据快速设置 struct 字段值
- 解析 struct 并收集 tag，解析 tag 值
- 以及更多 util 函数...

## Install

```shell
go get github.com/gookit/goutil/structs
```

## Go docs

- [Go docs](https://pkg.go.dev/github.com/gookit/goutil/structs)

## 初始化结构体

- 支持初始化使用环境变量
- 支持初始化 slice 字段，嵌套结构体

```go
type ExtraDefault struct {
    City   string `default:"some where"`
    Github string `default:"${ GITHUB_ADDR }"`
}

type User struct {
    Name  string        `default:"inhere"`
    Age   int           `default:"300"`
    Extra *ExtraDefault `default:""` // 标记需要初始化
}

optFn := func(opt *structs.InitOptions) {
    opt.ParseEnv = true
}

obj := &User{}
err := structs.InitDefaults(obj, optFn)
goutil.PanicErr(err)

dump.P(obj)
```

**初始化结果**:

```go
&structs_test.User {
  Name: string("inhere"), #len=6
  Age: int(300),
  Extra: &structs_test.ExtraDefault {
    City: string("some where"), #len=10
    Github: string("https://some .... url"), #len=21
  },
},
```