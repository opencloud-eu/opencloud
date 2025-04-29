package strutil

import (
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gookit/goutil/byteutil"
)

// MustToTime convert date string to time.Time
func MustToTime(s string, layouts ...string) time.Time {
	t, err := ToTime(s, layouts...)
	if err != nil {
		panic(err)
	}
	return t
}

// auto match use some commonly layouts.
// key is layout length.
var layoutMap = map[int][]string{
	6:  {"200601", "060102", time.Kitchen},
	8:  {"20060102", "06-01-02"},
	10: {"2006-01-02"},
	13: {"2006-01-02 15"},
	15: {time.Stamp},
	16: {"2006-01-02 15:04"},
	19: {"2006-01-02 15:04:05", time.RFC822, time.StampMilli},
	20: {"2006-01-02 15:04:05Z"},
	21: {time.RFC822Z},
	22: {time.StampMicro},
	23: {"2006-01-02 15:04:05.000", "2006-01-02 15:04:05.999"},
	24: {time.ANSIC},
	25: {time.RFC3339, time.StampNano},
	// time.Layout}, // must go >= 1.19
	26: {"2006-01-02 15:04:05.000000"},
	28: {time.UnixDate},
	29: {time.RFC1123, "2006-01-02 15:04:05.000000000"},
	30: {time.RFC850},
	31: {time.RFC1123Z},
	35: {time.RFC3339Nano},
}

// ToTime convert date string to time.Time
//
// NOTE: always use local timezone.
func ToTime(s string, layouts ...string) (t time.Time, err error) {
	// custom layout
	if len(layouts) > 0 {
		if len(layouts[0]) > 0 {
			return time.ParseInLocation(layouts[0], s, time.Local)
		}

		err = ErrDateLayout
		return
	}

	// auto match use some commonly layouts.
	strLn := len(s)
	maybeLayouts, ok := layoutMap[strLn]
	if !ok {
		err = ErrInvalidParam
		return
	}

	var hasAlphaT bool
	if pos := strings.IndexByte(s, 'T'); pos > 0 && pos < 12 {
		hasAlphaT = true
	}

	hasSlashR := strings.IndexByte(s, '/') > 0
	for _, layout := range maybeLayouts {
		// date string has "T". eg: "2006-01-02T15:04:05"
		if hasAlphaT {
			layout = strings.Replace(layout, " ", "T", 1)
		}

		// date string has "/". eg: "2006/01/02 15:04:05"
		if hasSlashR {
			layout = strings.Replace(layout, "-", "/", -1)
		}

		t, err = time.ParseInLocation(layout, s, time.Local)
		if err == nil {
			return
		}
	}

	// t, err = time.ParseInLocation(layout, s, time.Local)
	return
}

// ParseSizeOpt parse size expression options
type ParseSizeOpt struct {
	// OneAsMax if only one size value, use it as max size. default is false
	OneAsMax bool
	// SepChar is the separator char for time range string. default is '~'
	SepChar byte
	// KeywordFn is the function for parse keyword time string.
	KeywordFn func(string) (min, max uint64, err error)
}

func ensureOpt(opt *ParseSizeOpt) *ParseSizeOpt {
	if opt == nil {
		opt = &ParseSizeOpt{SepChar: '~'}
	} else {
		if opt.SepChar == 0 {
			opt.SepChar = '~'
		}
	}
	return opt
}

// ErrInvalidSizeExpr invalid size expression error
var ErrInvalidSizeExpr = errors.New("invalid size expr")

// ParseSizeRange parse range size expression to min and max size.
//
// Expression format:
//
//	"1KB~2MB"       => 1KB to 2MB
//	"-1KB"          => <1KB
//	"~1MB"          => <1MB
//	"< 1KB"         => <1KB
//	"1KB"           => >1KB
//	"1KB~"          => >1KB
//	">1KB"          => >1KB
//	"+1KB"          => >1KB
func ParseSizeRange(expr string, opt *ParseSizeOpt) (min, max uint64, err error) {
	opt = ensureOpt(opt)
	expr = strings.TrimSpace(expr)
	if expr == "" {
		err = ErrInvalidSizeExpr
		return
	}

	// parse size range. eg: "1KB~2MB"
	if strings.IndexByte(expr, '~') > -1 {
		s1, s2 := TrimCut(expr, "~")
		if s1 != "" {
			min, err = ToByteSize(s1)
			if err != nil {
				return
			}
		}

		if s2 != "" {
			max, err = ToByteSize(s2)
		}
		return
	}

	// parse single size. eg: "1KB"
	if byteutil.IsNumChar(expr[0]) {
		min, err = ToByteSize(expr)
		if err != nil {
			return
		}
		if opt.OneAsMax {
			max = min
		}
		return
	}

	// parse with prefix. eg: "<1KB", ">= 1KB", "-1KB", "+1KB"
	switch expr[0] {
	case '<', '-':
		max, err = ToByteSize(strings.Trim(expr[1:], " ="))
	case '>', '+':
		min, err = ToByteSize(strings.Trim(expr[1:], " ="))
	default:
		// parse keyword. eg: "small", "large"
		if opt.KeywordFn != nil {
			min, max, err = opt.KeywordFn(expr)
		} else {
			err = ErrInvalidSizeExpr
		}
	}
	return
}

// SafeByteSize converts size string like 1GB/1g or 12mb/12M into an unsigned integer number of bytes
func SafeByteSize(sizeStr string) uint64 {
	size, _ := ToByteSize(sizeStr)
	return size
}

// ToByteSize converts size string like 1GB/1g or 12mb/12M into an unsigned integer number of bytes
func ToByteSize(sizeStr string) (uint64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	lastPos := len(sizeStr) - 1
	if lastPos < 0 {
		return 0, nil
	}

	if sizeStr[lastPos] == 'b' || sizeStr[lastPos] == 'B' {
		// last second char is k,m,g,t
		lastSec := sizeStr[lastPos-1]
		if lastSec > 'A' {
			lastPos--
		}
	} else if IsNumChar(sizeStr[lastPos]) { // not unit suffix. eg: 346
		return strconv.ParseUint(sizeStr, 10, 32)
	}

	multiplier := float64(1)
	switch unicode.ToLower(rune(sizeStr[lastPos])) {
	case 'k':
		multiplier = 1 << 10
	case 'm':
		multiplier = 1 << 20
	case 'g':
		multiplier = 1 << 30
	case 't':
		multiplier = 1 << 40
	case 'p':
		multiplier = 1 << 50
	default: // b
		multiplier = 1
	}

	sizeNum := strings.TrimSpace(sizeStr[:lastPos])
	size, err := strconv.ParseFloat(sizeNum, 64)
	if err != nil {
		return 0, err
	}
	return uint64(size * multiplier), nil
}
