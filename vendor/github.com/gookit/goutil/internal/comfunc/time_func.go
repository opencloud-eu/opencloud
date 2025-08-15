package comfunc

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// check is duration string. TIP: extend unit d,w.  eg: "1d", "2w"
	//
	// time.ParseDuration() is max support hour "h".
	durStrReg = regexp.MustCompile(`^-?([0-9]+(?:\.[0-9]*)?(ns|us|µs|ms|s|m|h|d|w))+$`)

	// check long duration string. 验证整体格式是否符合
	//
	// eg: "1hour", "2hours", "3minutes", "4mins", "5days", "1weeks", "1month"
	//
	// time.ParseDuration() is not support long unit.
	durStrRegL = regexp.MustCompile(`^-?([0-9]+(?:\.[0-9]*)?[nuµsmhdw][a-zA-Z]{0,8})+$`)
	// use for parse duration string. see ToDuration()
	//
	// NOTE: 解析时，不能加最后的 `+` 会导致只匹配了最后一组 时间单位
	durStrRegL2 = regexp.MustCompile(`-?([0-9]+(?:\.[0-9]*)?)([nuµsmhdw][a-z]{0,8})`)
)

// IsDuration check the string is a duration string.
func IsDuration(s string) bool {
	if s == "0" || durStrReg.MatchString(s) {
		return true
	}
	return durStrRegL.MatchString(s)
}

// ToDuration parses a duration string. such as "300ms", "-1.5h" or "2h45m".
// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
//
// Diff of time.ParseDuration:
//   - support extends unit d, w at the end of string. such as "1d", "2w".
//   - support extends unit: month, week, day
//   - support long string unit at the end. such as "1hour", "2hours", "3minutes", "4mins", "5days", "1weeks".
//
// If the string is not a valid duration string, it will return an error.
func ToDuration(s string) (time.Duration, error) {
	ln := len(s)
	if ln == 0 {
		return 0, fmt.Errorf("empty duration string")
	}

	s = strings.ToLower(s)
	if s == "0" {
		return 0, nil
	}

	// check duration string is valid
	if !durStrRegL.MatchString(s) {
		return 0, fmt.Errorf("invalid duration string: %s", s)
	}

	// if ln < 4 AND end != d|w, directly call time.ParseDuration()
	if ln < 4 && s[ln-1] != 'd' && s[ln-1] != 'w' {
		return time.ParseDuration(s)
	}

	// time.ParseDuration() is not support long unit.
	ssList := durStrRegL2.FindAllStringSubmatch(s, -1)
	// fmt.Println(ssList)
	bts := make([]byte, 0, ln)
	if s[0] == '-' {
		bts = append(bts, '-')
	}

	// only one element. eg: "1day"
	if len(ssList) == 1 {
		bts = parseLongUnit(ssList[0], bts)
	} else {
		// more than one element. eg: "1day2hour3min"
		for _, ss := range ssList {
			if len(ss) == 3 {
				bts = parseLongUnit(ss, bts)
			}
		}
	}

	return time.ParseDuration(string(bts))
}

// convert to short unit
func parseLongUnit(ss []string, bts []byte) []byte {
	// eg: "3sec" -> ss=[3sec, -3, sec]
	num, unit := ss[1], ss[2]
	switch unit {
	case "month", "months":
		// time lib max unit is hour, so need convert by 24 * 30*n
		bts = appendNumToBytes(bts, num, 24*30)
		bts = append(bts, 'h')
	case "w", "week", "weeks":
		// time lib max unit is hour, so need convert by 24 * 7*n
		bts = appendNumToBytes(bts, num, 24*7)
		bts = append(bts, 'h')
	case "d", "day", "days":
		// time lib max unit is hour, so need convert by 24*n
		bts = appendNumToBytes(bts, num, 24)
		bts = append(bts, 'h')
	case "hour", "hours":
		bts = append(bts, num...)
		bts = append(bts, 'h')
	case "min", "mins", "minute", "minutes":
		bts = append(bts, num...)
		bts = append(bts, 'm')
	case "sec", "secs", "second", "seconds":
		bts = append(bts, num...)
		bts = append(bts, 's')
	default:
		first := ss[0]

		// '-' has been added on ToDuration()
		if first[0] == '-' {
			bts = append(bts, first[1:]...)
		} else {
			bts = append(bts, first...)
		}
	}

	return bts
}

func appendNumToBytes(bts []byte, num string, multiple int) []byte {
	if strings.ContainsRune(num, '.') {
		f, _ := strconv.ParseFloat(num, 64) // is float number
		val := f * float64(multiple)

		// 使用 Float 保留两位小数 -> 会始终有两位小数，即使是N.00
		// bts = strconv.AppendFloat(bts, val, 'f', 2, 64)

		// 四舍五入到两位小数
		rounded := math.Round(val*100) / 100
		// 使用 AppendFloat 自动去除末尾的 .0 或 .00
		bts = strconv.AppendFloat(bts, rounded, 'f', -1, 64)
	} else {
		n, _ := strconv.Atoi(num)
		bts = strconv.AppendInt(bts, int64(n*multiple), 10)
	}

	return bts
}
