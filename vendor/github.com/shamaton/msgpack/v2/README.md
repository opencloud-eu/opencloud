# MessagePack for Golang

[![Go Reference](https://pkg.go.dev/badge/github.com/shamaton/msgpack.svg)](https://pkg.go.dev/github.com/shamaton/msgpack)
![test](https://github.com/shamaton/msgpack/workflows/test/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/shamaton/msgpack)](https://goreportcard.com/report/github.com/shamaton/msgpack)
[![codecov](https://codecov.io/gh/shamaton/msgpack/branch/master/graph/badge.svg?token=9PD2JUK5V3)](https://codecov.io/gh/shamaton/msgpack)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fshamaton%2Fmsgpack.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fshamaton%2Fmsgpack?ref=badge_shield)

## 📣 Announcement: `time.Time` decoding defaults to **UTC** in v3
Starting with **v3.0.0**, when decoding MessagePack **Timestamp** into Go’s `time.Time`,
the default `Location` will be **UTC** (previously `Local`). The instant is unchanged.
To keep the old behavior, use `SetDecodedTimeAsLocal()`.

## Features
* Supported types : primitive / array / slice / struct / map / interface{} and time.Time
* Renaming fields via `msgpack:"field_name"`
* Omitting fields via `msgpack:"-"`
* Omitting empty fields via `msgpack:"field_name,omitempty"`
* Supports extend encoder / decoder [(example)](./msgpack_example_test.go)
* Can also Encoding / Decoding struct as array

## Installation

Current version is **msgpack/v2**.
```sh
go get -u github.com/shamaton/msgpack/v2
```

## Quick Start
```go
package main

import (
  "github.com/shamaton/msgpack/v2"
  "net/http"
)

type Struct struct {
	String string
}

// simple
func main() {
	v := Struct{String: "msgpack"}

	d, err := msgpack.Marshal(v)
	if err != nil {
		panic(err)
	}
	r := Struct{}
	if err =  msgpack.Unmarshal(d, &r); err != nil {
		panic(err)
	}
}

// streaming
func handle(w http.ResponseWriter, r *http.Request) {
	var body Struct
	if err := msgpack.UnmarshalRead(r, &body); err != nil {
		panic(err)
    }
	if err := msgpack.MarshalWrite(w, body); err != nil {
		panic(err)
    }
}
```

## 📣 Announcement: `time.Time` decoding defaults to **UTC** in v3

**TL;DR:** Starting with **v3.0.0**, when decoding MessagePack **Timestamp** into Go’s `time.Time`, the default `Location` will be **UTC** (previously `Local`). The **instant** is unchanged—only the display/location changes. This avoids host-dependent differences and aligns with common distributed systems practice.

### What is changing?

* **Before (v2.x):** Decoded `time.Time` defaults to `Local`.
* **After (v3.0.0):** Decoded `time.Time` defaults to **UTC**.

MessagePack’s Timestamp encodes an **instant** (epoch seconds + nanoseconds) and does **not** carry timezone info. Your data’s point in time is the same; only `time.Time.Location()` differs.

### Why?

* Eliminate environment-dependent behavior (e.g., different hosts showing different local zones).
* Make “UTC by default” the safe, predictable baseline for logs, APIs, and distributed apps.

### Who is affected?

* Apps that **display local time** without explicitly converting from UTC.
* If your code already normalizes to UTC or explicitly sets a location, you’re likely unaffected.

### Keep the old behavior (Local)

If you want the v2 behavior on v3:

```go
msgpack.SetDecodedTimeAsLocal()
```

Or convert after the fact:

```go
var t time.Time
_ = msgpack.Unmarshal(data, &t)
t = t.In(time.Local)
```

### Preview the new behavior on v2 (optional)

You can opt into UTC today on v2.x:

```go
msgpack.SetDecodedTimeAsUTC()
```

## Benchmark
This result made from [shamaton/msgpack_bench](https://github.com/shamaton/msgpack_bench)

![msgpack_bench](https://github.com/user-attachments/assets/ed5bc4c5-a149-4083-98b8-ee6820c00eae)

## License

This library is under the MIT License.
