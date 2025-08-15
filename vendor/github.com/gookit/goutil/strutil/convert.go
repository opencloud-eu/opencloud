package strutil

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/gookit/goutil/internal/comfunc"
	"github.com/gookit/goutil/mathutil"
)

var (
	// ErrDateLayout error
	ErrDateLayout = errors.New("invalid date layout string")
	// ErrInvalidParam error
	ErrInvalidParam = errors.New("invalid input for parse time")

	// some regex for convert string.
	toSnakeReg  = regexp.MustCompile("[A-Z][a-z]")
	toCamelRegs = map[string]*regexp.Regexp{
		" ": regexp.MustCompile(" +[a-zA-Z]"),
		"-": regexp.MustCompile("-+[a-zA-Z]"),
		"_": regexp.MustCompile("_+[a-zA-Z]"),
	}
)

// Internal func refers:
// strconv.QuoteRune()
// strconv.QuoteToASCII()
// strconv.AppendQuote()
// strconv.AppendQuoteRune()

// Quote alias of strings.Quote
func Quote(s string) string { return strconv.Quote(s) }

// Unquote remove start and end quotes by single-quote or double-quote
//
// tip: strconv.Unquote cannot unquote single-quote
func Unquote(s string) string {
	ln := len(s)
	if ln < 2 {
		return s
	}

	qs, qe := s[0], s[ln-1]

	var valid bool
	if qs == '"' && qe == '"' {
		valid = true
	} else if qs == '\'' && qe == '\'' {
		valid = true
	}

	if valid {
		s = s[1 : ln-1] // exclude quotes
	}
	// strconv.Unquote cannot unquote single-quote
	// if ns, err := strconv.Unquote(s); err == nil {
	// 	return ns
	// }
	return s
}

// Join alias of strings.Join
func Join(sep string, ss ...string) string { return strings.Join(ss, sep) }

// JoinList alias of strings.Join
func JoinList(sep string, ss []string) string { return strings.Join(ss, sep) }

// JoinComma quick join strings by comma
func JoinComma(ss []string) string { return strings.Join(ss, ",") }

// JoinAny type to string
func JoinAny(sep string, parts ...any) string {
	ss := make([]string, 0, len(parts))
	for _, part := range parts {
		ss = append(ss, QuietString(part))
	}

	return strings.Join(ss, sep)
}

// Implode alias of strings.Join
func Implode(sep string, ss ...string) string { return strings.Join(ss, sep) }

/*************************************************************
 * convert value to string
 *************************************************************/

// String convert value to string, return error on failed
func String(val any) (string, error) { return ToStringWith(val) }

// ToString convert value to string, return error on failed
func ToString(val any) (string, error) { return ToStringWith(val) }

// StringOrErr convert value to string, return error on failed
func StringOrErr(val any) (string, error) { return ToStringWith(val) }

// QuietString convert value to string, will ignore error. same as SafeString()
func QuietString(val any) string { return SafeString(val) }

// SafeString convert value to string. Will ignore error
func SafeString(in any) string {
	s, _ := AnyToString(in, false)
	return s
}

// StringOrPanic convert value to string, will panic on error
func StringOrPanic(val any) string { return MustString(val) }

// MustString convert value to string. will panic on error
func MustString(val any) string {
	s, err := ToStringWith(val)
	if err != nil {
		panic(err)
	}
	return s
}

// StringOrDefault convert any value to string, return default value on failed
func StringOrDefault(val any, defVal string) string { return StringOr(val, defVal) }

// StringOr convert any value to string, return default value on failed
func StringOr(val any, defVal string) string {
	s, err := ToStringWith(val)
	if err != nil {
		return defVal
	}
	return s
}

// AnyToString convert any value to string.
//
// For defaultAsErr:
//
//   - False  will use fmt.Sprint convert unsupported type
//   - True   will return error on convert fail.
func AnyToString(val any, defaultAsErr bool) (s string, err error) {
	var optFn comfunc.ConvOptionFn
	if !defaultAsErr {
		optFn = comfunc.WithUserConvFn(comfunc.StrBySprintFn)
	}
	return ToStringWith(val, optFn)
}

// ToStringWith try to convert value to string. can with some option func, more see comfunc.ConvOption.
func ToStringWith(in any, optFns ...comfunc.ConvOptionFn) (string, error) {
	return comfunc.ToStringWith(in, optFns...)
}

/*************************************************************
 * convert string value to bool
 *************************************************************/

// ToBool convert string to bool
func ToBool(s string) (bool, error) {
	return comfunc.StrToBool(strings.TrimSpace(s))
}

// QuietBool convert to bool, will ignore error
func QuietBool(s string) bool { return SafeBool(s) }

// SafeBool convert to bool and will ignore error
func SafeBool(s string) bool {
	val, _ := comfunc.StrToBool(strings.TrimSpace(s))
	return val
}

// MustBool convert to bool and will panic on error
func MustBool(s string) bool {
	val, err := ToBool(s)
	if err != nil {
		panic(err)
	}
	return val
}

// Bool parse string to bool. like strconv.ParseBool()
func Bool(s string) (bool, error) {
	return comfunc.StrToBool(strings.TrimSpace(s))
}

/*************************************************************
 * convert string value to int
 *************************************************************/

// Int convert string to int, alias of ToInt()
func Int(s string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(s))
}

// ToInt convert string to int, return error on fail
func ToInt(s string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(s))
}

// IntOrDefault convert string to int, return default value on fail
func IntOrDefault(s string, defVal int) int {
	return IntOr(s, defVal)
}

// IntOr convert string to int, return default value on fail
func IntOr(s string, defVal int) int {
	val, err := ToInt(s)
	if err != nil {
		return defVal
	}
	return val
}

// SafeInt convert string to int, will ignore error
func SafeInt(s string) int {
	val, _ := ToInt(s)
	return val
}

// QuietInt convert string to int, will ignore error
func QuietInt(s string) int { return SafeInt(s) }

// MustInt convert string to int, will panic on error
func MustInt(s string) int { return IntOrPanic(s) }

// IntOrPanic convert value to int, will panic on error
func IntOrPanic(s string) int {
	val, err := ToInt(s)
	if err != nil {
		panic(err)
	}
	return val
}

/*************************************************************
 * convert string value to int64
 *************************************************************/

// Int64 convert string to int, will ignore error
func Int64(s string) int64 { return SafeInt64(s) }

// QuietInt64 convert string to int, will ignore error
func QuietInt64(s string) int64 { return SafeInt64(s) }

// SafeInt64 convert string to int, will ignore error
func SafeInt64(s string) int64 {
	val, _ := Int64OrErr(s)
	return val
}

// ToInt64 convert string to int, return error on fail
func ToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 0)
}

// Int64OrDefault convert string to int, return default value on fail
func Int64OrDefault(s string, defVal int64) int64 {
	return Int64Or(s, defVal)
}

// Int64Or convert string to int, return default value on fail
func Int64Or(s string, defVal int64) int64 {
	val, err := ToInt64(s)
	if err != nil {
		return defVal
	}
	return val
}

// Int64OrErr convert string to int, return error on fail
func Int64OrErr(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 0)
}

// MustInt64 convert value to int, will panic on error
func MustInt64(s string) int64 { return Int64OrPanic(s) }

// Int64OrPanic convert value to int, will panic on error
func Int64OrPanic(s string) int64 {
	val, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		panic(err)
	}
	return val
}

/*************************************************************
 * convert string value to uint
 *************************************************************/

// Uint convert string to uint, will ignore error
func Uint(s string) uint64 { return SafeUint(s) }

// SafeUint convert string to uint, will ignore error
func SafeUint(s string) uint64 {
	val, _ := UintOrErr(s)
	return val
}

// ToUint convert string to uint, return error on fail. alias of UintOrErr()
func ToUint(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 0)
}

// UintOrErr convert string to uint, return error on fail
func UintOrErr(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 0)
}

// MustUint convert value to uint, will panic on error. alias of UintOrPanic()
func MustUint(s string) uint64 { return UintOrPanic(s) }

// UintOrPanic convert value to uint, will panic on error
func UintOrPanic(s string) uint64 {
	val, err := UintOrErr(s)
	if err != nil {
		panic(err)
	}
	return val
}

// UintOrDefault convert string to uint, return default value on fail
func UintOrDefault(s string, defVal uint64) uint64 {
	return UintOr(s, defVal)
}

// UintOr convert string to uint, return default value on fail
func UintOr(s string, defVal uint64) uint64 {
	val, err := UintOrErr(s)
	if err != nil {
		return defVal
	}
	return val
}

/*************************************************************
 * convert string value to byte
 * refer from https://github.com/valyala/fastjson/blob/master/util.go
 *************************************************************/

// Byte2str convert bytes to string
func Byte2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Byte2string convert bytes to string
func Byte2string(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// ToBytes convert string to bytes
func ToBytes(s string) (b []byte) {
	strh := (*reflect.StringHeader)(unsafe.Pointer(&s))

	sh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh.Data = strh.Data
	sh.Len = strh.Len
	sh.Cap = strh.Len
	return b
}

/*************************************************************
 * convert string value to int/string slice, time.Time
 *************************************************************/

// Ints alias of the ToIntSlice(). default sep is comma(,)
func Ints(s string, sep ...string) []int {
	ints, _ := ToIntSlice(s, sep...)
	return ints
}

// ToInts alias of the ToIntSlice(). default sep is comma(,)
func ToInts(s string, sep ...string) ([]int, error) { return ToIntSlice(s, sep...) }

// ToIntSlice split string to slice and convert item to int.
//
// Default sep is comma
func ToIntSlice(s string, sep ...string) (ints []int, err error) {
	ss := ToSlice(s, sep...)
	for _, item := range ss {
		iVal, err := mathutil.ToInt(item)
		if err != nil {
			return []int{}, err
		}

		ints = append(ints, iVal)
	}
	return
}

// ToArray alias of the ToSlice()
func ToArray(s string, sep ...string) []string { return ToSlice(s, sep...) }

// Strings alias of the ToSlice()
func Strings(s string, sep ...string) []string { return ToSlice(s, sep...) }

// ToStrings alias of the ToSlice()
func ToStrings(s string, sep ...string) []string { return ToSlice(s, sep...) }

// ToSlice split string to array.
func ToSlice(s string, sep ...string) []string {
	if len(sep) > 0 {
		return Split(s, sep[0])
	}
	return Split(s, ",")
}

// ToOSArgs split string to string[](such as os.Args)
// func ToOSArgs(s string) []string {
// 	return cliutil.StringToOSArgs(s) // error: import cycle not allowed
// }

// ToDuration parses a duration string. such as "300ms", "-1.5h" or "2h45m".
// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
func ToDuration(s string) (time.Duration, error) {
	return comfunc.ToDuration(s)
}
