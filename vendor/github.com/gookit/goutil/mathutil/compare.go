package mathutil

import (
	"math"

	"github.com/gookit/goutil/comdef"
)

// Min compare two value and return max value
func Min[T comdef.Number](x, y T) T {
	if x < y {
		return x
	}
	return y
}

// Max compare two value and return max value
func Max[T comdef.Number](x, y T) T {
	if x > y {
		return x
	}
	return y
}

// SwapMin compare and always return [min, max] value
func SwapMin[T comdef.Number](x, y T) (T, T) {
	if x < y {
		return x, y
	}
	return y, x
}

// SwapMax compare and always return [max, min] value
func SwapMax[T comdef.Number](x, y T) (T, T) {
	if x > y {
		return x, y
	}
	return y, x
}

// MaxInt compare and return max value
func MaxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// SwapMaxInt compare and return max, min value
func SwapMaxInt(x, y int) (int, int) {
	if x > y {
		return x, y
	}
	return y, x
}

// MaxI64 compare and return max value
func MaxI64(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

// SwapMaxI64 compare and return max, min value
func SwapMaxI64(x, y int64) (int64, int64) {
	if x > y {
		return x, y
	}
	return y, x
}

// MaxFloat compare and return max value
func MaxFloat(x, y float64) float64 {
	return math.Max(x, y)
}
