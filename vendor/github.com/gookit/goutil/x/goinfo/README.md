# GoInfo

 `goutil/x/goinfo` provide some useful info for golang.

> Github: https://github.com/gookit/goutil

## Install

```bash
go get github.com/gookit/goutil/x/goinfo
```

## Go docs

- [Go docs](https://pkg.go.dev/github.com/gookit/goutil)

## Usage

```go
gover := goinfo.GoVersion() // eg: "1.15.6"

```

## Testings

```shell
go test -v ./goinfo/...
```

Test limit by regexp:

```shell
go test -v -run ^TestSetByKeys ./goinfo/...
```
