//go:build windows

package strutil

import (
	"math"
	"math/rand"
	"time"
	"unsafe"
)

const MaximumCapacity = math.MaxInt>>1 + 1

var rn = rand.NewSource(time.Now().UnixNano())

// nearestPowerOfTwo 返回一个大于等于cap的最近的2的整数次幂，参考java8的hashmap的tableSizeFor函数
//   - cap 输入参数
//
// 返回一个大于等于cap的最近的2的整数次幂
func nearestPowerOfTwo(cap int) int {
	n := cap - 1
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	if n < 0 {
		return 1
	}

	if n >= MaximumCapacity {
		return MaximumCapacity
	}
	return n + 1
}

// buildRandomString 生成随机字符串
//   - letters 字符串模板
//   - length 生成长度
//
// 返回一个指定长度的随机字符串
func buildRandomString(letters string, length int) string {
	// 仿照strings.Builder
	// 创建一个长度为 length 的字节切片
	bytes := make([]byte, length)
	strLength := len(letters)
	if strLength <= 0 {
		return ""
	}
	if strLength == 1 {
		for i := 0; i < length; i++ {
			bytes[i] = letters[0]
		}
		return *(*string)(unsafe.Pointer(&bytes))
	}

	// letters的字符需要使用多少个比特位数才能表示完
	// letterIdBits := int(math.Ceil(math.Log2(strLength))),下面比上面的代码快
	letterIdBits := int(math.Log2(float64(nearestPowerOfTwo(strLength))))
	// 最大的字母id掩码
	var letterIdMask int64 = 1<<letterIdBits - 1
	// 可用次数的最大值
	letterIdMax := 63 / letterIdBits

	// UnixNano: 1607400451937462000
	// 循环生成随机字符串
	for i, cache, remain := length-1, rn.Int63(), letterIdMax; i >= 0; {
		// 检查随机数生成器是否用尽所有随机数
		if remain == 0 {
			cache, remain = rn.Int63(), letterIdMax
		}
		// 从可用字符的字符串中随机选择一个字符
		if idx := int(cache & letterIdMask); idx < strLength {
			bytes[i] = letters[idx]
			i--
		}
		// 右移比特位数，为下次选择字符做准备
		cache >>= letterIdBits
		remain--
	}

	// 仿照strings.Builder用unsafe包返回一个字符串，避免拷贝
	// 将字节切片转换为字符串并返回
	return *(*string)(unsafe.Pointer(&bytes))
}
