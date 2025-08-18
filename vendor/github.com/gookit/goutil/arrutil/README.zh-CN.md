## ArrUtil

`arrutil` 是一个用于操作数组和切片的工具包，提供了丰富的功能来简化 Go 语言中数组和切片的处理。

## Install

```shell
go get github.com/gookit/goutil/arrutil
```

## Go docs

- [Go docs](https://pkg.go.dev/github.com/gookit/goutil/arrutil)

## 基本功能

主要包括以下功能：

1. **数组/切片的基本操作**：
    - `RandomOne`：从数组或切片中随机获取一个元素。
    - `Reverse`：反转数组或切片中的元素顺序。

2. **检查和查找**：
    - `Contains` 和 `HasValue`：检查数组或切片是否包含特定值。
    - `InStrings` 和 `StringsHas`：检查字符串切片中是否包含特定字符串。
    - `IntsHas` 和 `Int64sHas`：检查整数切片中是否包含特定整数值。
    - `Find` 和 `FindOrDefault`：根据谓词函数查找元素，如果没有找到则返回默认值。

3. **集合操作**：
    - `Union`：计算两个切片的并集。
    - `Intersects`：计算两个切片的交集。
    - `Excepts` 和 `Differences`：计算两个切片的差集。
    - `TwowaySearch`：在切片中双向搜索特定元素。

4. **转换和格式化**：
    - `ToInt64s` 和 `ToStrings`：将任意类型的切片转换为整数或字符串切片。
    - `JoinSlice` 和 `JoinStrings`：将切片中的元素连接成一个字符串。
    - `FormatIndent`：将数组或切片格式化为带有缩进的字符串。

5. **排序和过滤**：
    - `Sort`：对切片进行排序。
    - `Filter`：根据条件过滤切片中的元素。
    - `Remove`：从切片中移除特定元素。

6. **其他实用功能**：
    - `Unique`：去除切片中的重复元素。
    - `FirstOr`：获取切片的第一个元素，如果切片为空则返回默认值。

这些功能使得在 Go 语言中处理数组和切片变得更加方便和高效。无论是进行数据处理、集合运算还是字符串操作，`arrutil` 都提供了一系列简洁且易于使用的函数来帮助开发者完成任务。

