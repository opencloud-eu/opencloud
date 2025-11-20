# displaywidth

A high-performance Go package for measuring the monospace display width of strings, UTF-8 bytes, and runes.

[![Documentation](https://pkg.go.dev/badge/github.com/clipperhouse/displaywidth.svg)](https://pkg.go.dev/github.com/clipperhouse/displaywidth)
[![Test](https://github.com/clipperhouse/displaywidth/actions/workflows/gotest.yml/badge.svg)](https://github.com/clipperhouse/displaywidth/actions/workflows/gotest.yml)
[![Fuzz](https://github.com/clipperhouse/displaywidth/actions/workflows/gofuzz.yml/badge.svg)](https://github.com/clipperhouse/displaywidth/actions/workflows/gofuzz.yml)
## Install
```bash
go get github.com/clipperhouse/displaywidth
```

## Usage

```go
package main

import (
    "fmt"
    "github.com/clipperhouse/displaywidth"
)

func main() {
    width := displaywidth.String("Hello, 世界!")
    fmt.Println(width)

    width = displaywidth.Bytes([]byte("🌍"))
    fmt.Println(width)

    width = displaywidth.Rune('🌍')
    fmt.Println(width)
}
```

### Options

You can specify East Asian Width and Strict Emoji Neutral settings. If
unspecified, the default is `EastAsianWidth: false, StrictEmojiNeutral: true`.

```go
options := displaywidth.Options{
    EastAsianWidth:     true,
    StrictEmojiNeutral: false,
}

width := options.String("Hello, 世界!")
fmt.Println(width)
```

## Details

This package implements the Unicode East Asian Width standard (UAX #11) and is
intended to be compatible with `go-runewidth`. It operates on bytes without
decoding runes for better performance.

## Prior Art

[mattn/go-runewidth](https://github.com/mattn/go-runewidth)

[x/text/width](https://pkg.go.dev/golang.org/x/text/width)

[x/text/internal/triegen](https://pkg.go.dev/golang.org/x/text/internal/triegen)

## Benchmarks

Part of my motivation is the insight that we can avoid decoding runes for better performance.

```bash
go test -bench=. -benchmem
```

```
goos: darwin
goarch: arm64
pkg: github.com/clipperhouse/displaywidth
cpu: Apple M2
BenchmarkStringDefault/displaywidth-8      	     10537 ns/op	     160.10 MB/s	       0 B/op	       0 allocs/op
BenchmarkStringDefault/go-runewidth-8      	     14162 ns/op	     119.12 MB/s	       0 B/op	       0 allocs/op
BenchmarkString_EAW/displaywidth-8         	     10776 ns/op	     156.55 MB/s	       0 B/op	       0 allocs/op
BenchmarkString_EAW/go-runewidth-8         	     23987 ns/op	      70.33 MB/s	       0 B/op	       0 allocs/op
BenchmarkString_StrictEmoji/displaywidth-8 	     10892 ns/op	     154.88 MB/s	       0 B/op	       0 allocs/op
BenchmarkString_StrictEmoji/go-runewidth-8 	     14552 ns/op	     115.93 MB/s	       0 B/op	       0 allocs/op
BenchmarkString_ASCII/displaywidth-8       	      1116 ns/op	     114.72 MB/s	       0 B/op	       0 allocs/op
BenchmarkString_ASCII/go-runewidth-8       	      1178 ns/op	     108.67 MB/s	       0 B/op	       0 allocs/op
BenchmarkString_Unicode/displaywidth-8     	       896.9 ns/op	     148.29 MB/s	       0 B/op	       0 allocs/op
BenchmarkString_Unicode/go-runewidth-8     	      1434 ns/op	      92.72 MB/s	       0 B/op	       0 allocs/op
BenchmarkStringWidth_Emoji/displaywidth-8  	      3033 ns/op	     238.74 MB/s	       0 B/op	       0 allocs/op
BenchmarkStringWidth_Emoji/go-runewidth-8  	      4841 ns/op	     149.56 MB/s	       0 B/op	       0 allocs/op
BenchmarkString_Mixed/displaywidth-8       	      4064 ns/op	     124.74 MB/s	       0 B/op	       0 allocs/op
BenchmarkString_Mixed/go-runewidth-8       	      4696 ns/op	     107.97 MB/s	       0 B/op	       0 allocs/op
BenchmarkString_ControlChars/displaywidth-8	       320.6 ns/op	     102.93 MB/s	       0 B/op	       0 allocs/op
BenchmarkString_ControlChars/go-runewidth-8	       373.8 ns/op	      88.28 MB/s	       0 B/op	       0 allocs/op
BenchmarkRuneDefault/displaywidth-8        	       335.5 ns/op	     411.35 MB/s	       0 B/op	       0 allocs/op
BenchmarkRuneDefault/go-runewidth-8        	       681.2 ns/op	     202.58 MB/s	       0 B/op	       0 allocs/op
BenchmarkRuneWidth_EAW/displaywidth-8      	       146.7 ns/op	     374.80 MB/s	       0 B/op	       0 allocs/op
BenchmarkRuneWidth_EAW/go-runewidth-8      	       495.6 ns/op	     110.98 MB/s	       0 B/op	       0 allocs/op
BenchmarkRuneWidth_ASCII/displaywidth-8    	        63.00 ns/op	     460.33 MB/s	       0 B/op	       0 allocs/op
BenchmarkRuneWidth_ASCII/go-runewidth-8    	        68.90 ns/op	     420.91 MB/s	       0 B/op	       0 allocs/op
```

I use a similar technique in [this grapheme cluster library](https://github.com/clipperhouse/uax29).

## Compatibility

`displaywidth` will mostly give the same outputs as `go-runewidth`, but there are some differences:

- Unicode category Mn (Nonspacing Mark): `displaywidth` will return width 0, `go-runewidth` may return width 1 for some runes.
- Unicode category Cf (Format): `displaywidth` will return width 0, `go-runewidth` may return width 1 for some runes.
- Unicode category Mc (Spacing Mark): `displaywidth` will return width 1, `go-runewidth` may return width 0 for some runes.
- Unicode category Cs (Surrogate): `displaywidth` will return width 0, `go-runewidth` may return width 1 for some runes. Surrogates are not valid UTF-8; some packages may turn them into the replacement character (U+FFFD).
- Unicode category Zl (Line separator): `displaywidth` will return width 0, `go-runewidth` may return width 1.
- Unicode category Zp (Paragraph separator): `displaywidth` will return width 0, `go-runewidth` may return width 1.
- Unicode Noncharacters (U+FFFE and U+FFFF): `displaywidth` will return width 0, `go-runewidth` may return width 1.

See `TestCompatibility` for more details.
