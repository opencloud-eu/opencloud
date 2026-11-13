package httprate

import (
	"strconv"
	"time"

	"github.com/zeebo/xxh3"
)

func LimitCounterKey(key string, window time.Time) uint64 {
	h := xxh3.New()
	h.WriteString(key)
	h.WriteString(strconv.FormatInt(window.Unix(), 10))
	return h.Sum64()
}
