# Bytes Util

Provide some common bytes util functions.

## Install

```shell
go get github.com/gookit/goutil/byteutil
```

## Go docs

- [Go docs](https://pkg.go.dev/github.com/gookit/goutil/byteutil)

## Functions API

> **Note**: doc by run `go doc ./byteutil`

```go
func AppendAny(dst []byte, v any) []byte
func FirstLine(bs []byte) []byte
func IsNumChar(c byte) bool
func Md5(src any) []byte
func Random(length int) ([]byte, error)
func SafeString(bs []byte, err error) string
func StrOrErr(bs []byte, err error) (string, error)
func String(b []byte) string
func ToString(b []byte) string
type Buffer struct{ ... }
func NewBuffer() *Buffer
type BytesEncoder interface{ ... }
type ChanPool struct{ ... }
func NewChanPool(maxSize int, width int, capWidth int) *ChanPool
type StdEncoder struct{ ... }
func NewStdEncoder(encFn func(src []byte) []byte, decFn func(src []byte) ([]byte, error)) *StdEncoder
```

## Code Check & Testing

```bash
gofmt -w -l ./
golint ./...
```

**Testing**:

```shell
go test -v ./byteutil/...
```

**Test limit by regexp**:

```shell
go test -v -run ^TestSetByKeys ./byteutil/...
```
