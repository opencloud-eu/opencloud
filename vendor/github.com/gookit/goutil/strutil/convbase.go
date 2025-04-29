package strutil

import (
	"strconv"
	"strings"

	"github.com/gookit/goutil/basefn"
)

//
// -------------------- convert base --------------------
//

const (
	Base10Chars = "0123456789"
	Base16Chars = "0123456789abcdef"
	Base32Chars = "0123456789abcdefghjkmnpqrstvwxyz"
	Base36Chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	Base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	Base64Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ+/"
)

// Base10Conv convert base 10 string to new base string.
func Base10Conv(src string, to int) string {
	return BaseConv(src, 10, to)
}

// BaseConv convert base string by from and to base.
//
// Note: from and to base must be in [2, 64]
//
// Usage:
//
//	BaseConv("123", 10, 16) // Output: "7b"
//	BaseConv("7b", 16, 10) // Output: "123"
func BaseConv(src string, from, to int) string {
	if from > 64 || from < 2 {
		return ""
	}
	if to > 64 || to < 2 {
		return ""
	}
	return BaseConvByTpl(src, Base64Chars[:from], Base64Chars[:to])
}

// BaseConvByTpl convert base string by template.
//
// Usage:
//
//	BaseConvert("123", Base62Chars, Base16Chars) // Output: "1e"
//	BaseConvert("1e", Base16Chars, Base62Chars) // Output: "123"
func BaseConvByTpl(src string, fromBase, toBase string) string {
	if fromBase == toBase {
		return src
	}

	// convert to base 10
	var dec uint64
	if fromBase == Base10Chars {
		var err error
		dec, err = strconv.ParseUint(src, 10, 0)
		if err != nil {
			basefn.Panicf("input is not a valid decimal number: %s", src)
		}
	} else {
		fLen := uint64(len(fromBase))
		for _, c := range src {
			dec = dec*fLen + uint64(strings.IndexRune(fromBase, c))
		}
	}

	// convert to new base
	var res string
	toLen := uint64(len(toBase))
	for dec > 0 {
		res = string(toBase[dec%toLen]) + res
		dec /= toLen
	}
	return res
}
