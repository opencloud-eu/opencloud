//go:build !windows

package strutil

import (
	"math/rand"
	"time"
)

var rn = newRand()

func newRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

// buildRandomString 生成随机字符串
func buildRandomString(letters string, length int) string {
	// rn := newRand()
	cs := make([]byte, length)

	lettersN := len(letters)
	for i := 0; i < length; i++ {
		cs[i] = letters[rn.Intn(lettersN)]
	}

	return Byte2str(cs)
}
