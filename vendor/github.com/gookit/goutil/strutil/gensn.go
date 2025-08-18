package strutil

import (
	"hash/crc32"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/x/basefn"
)

// global id:
//
//	https://github.com/rs/xid
//	https://github.com/satori/go.uuid
var (
	DefMinInt = 1000
	DefMaxInt = 9999
)

// MicroTimeID generate.
//   - return like: 16074145697981929446(len: 20)
//
// Conv Base:
//
//	mtId := MicroTimeID() // eg: 16935349145643425047 len: 20
//	b16id := Base10Conv(mtId, 16) // eg: eb067252154a9d17 len: 16
//	b32id := Base10Conv(mtId, 32) // eg: em1jia8akl78n len: 13
//	b36id := Base10Conv(mtId, 36) // eg: 3ko088phiuoev len: 13
//	b62id := Base10Conv(mtId, 62) // eg: kb24SKgsQ9V len: 11
func MicroTimeID() string { return MTimeBaseID(10) }

// MicroTimeHexID micro time HEX ID generate.
//
// return like: 5b5f0588af1761ad3(len: 16-17)
func MicroTimeHexID() string { return MTimeHexID() }

// MTimeHexID micro time HEX ID generate.
//
// return like: 5b5f0588af1761ad3(len: 16-17)
func MTimeHexID() string { return MTimeBaseID(16) }

// MTimeBase36 micro time BASE36 id generate.
func MTimeBase36() string { return MTimeBaseID(36) }

// MTimeBaseID micro time BASE id generate. toBase: 2-36
//
// Examples:
//   - MTimeBaseID(16): 5b5f0588af1761ad3(len: 16-17)
//   - MTimeBaseID(36): gorntzvsa73mo(len: 13)
func MTimeBaseID(toBase int) string {
	ms := time.Now().UnixMicro()
	ri := mathutil.RandomInt(DefMinInt, DefMaxInt)
	return strconv.FormatInt(ms, toBase) + strconv.FormatInt(int64(ri), toBase)
}

// DatetimeNo generate. can use for order-no.
//
//   - No prefix, return like: 2023041410484904074285478388(len: 28)
//   - With prefix, return like: prefix2023041410484904074285478388(len: 28 + len(prefix))
func DatetimeNo(prefix string) string { return DateSN(prefix) }

// DateSN generate date serial number. PREFIX + yyyyMMddHHmmss + ext(微秒+随机数)
func DateSN(prefix string) string {
	nt := time.Now()
	pl := len(prefix)
	bs := make([]byte, 0, 28+pl)
	if pl > 0 {
		bs = append(bs, prefix...)
	}

	// micro datetime
	bs = nt.AppendFormat(bs, "20060102150405.000000")
	bs[14+pl] = '0'

	// host
	name, err := os.Hostname()
	if err != nil {
		name = "default"
	}
	c32 := crc32.ChecksumIEEE([]byte(name)) // eg: 4006367001
	bs = strconv.AppendUint(bs, uint64(c32%99), 10)

	// rand 1000 - 9999
	rs := rand.New(rand.NewSource(nt.UnixNano()))
	bs = strconv.AppendInt(bs, 1000+rs.Int63n(8999), 10)

	return string(bs)
}

// DateSNV2 generate date serial number.
//   - 2 < extBase <= 36
//   - return: PREFIX + yyyyMMddHHmmss + extBase(6bit micro + 5bit random number)
//
// Example:
//   - prefix=P, extBase=16, return: P2023091414361354b4490(len=22)
//   - prefix=P, extBase=36, return: P202309141436131gw3jg(len=21)
func DateSNV2(prefix string, extBase ...int) string {
	pl := len(prefix)
	bs := make([]byte, 0, 22+pl)
	if pl > 0 {
		bs = append(bs, prefix...)
	}

	// micro datetime
	nt := time.Now()
	bs = nt.AppendFormat(bs, "20060102150405.000000")

	// rand 10000 - 99999
	rs := rand.New(rand.NewSource(nt.UnixNano()))
	// 6bit micro + 5bit rand
	ext := strconv.AppendInt(bs[16+pl:], 10000+rs.Int63n(89999), 10)

	base := basefn.FirstOr(extBase, 16)
	// prefix + yyyyMMddHHmmss + ext(convert to base)
	bs = append(bs[:14+pl], strconv.FormatInt(SafeInt64(string(ext)), base)...)

	return string(bs)
}
