// Package mathutil provide math(int, number) util functions. eg: convert, math calc, random
package mathutil

import (
	"math"

	"github.com/gookit/goutil/comdef"
)

// OrElse return default value on val is zero, else return val
func OrElse[T comdef.Number](val, defVal T) T {
	return ZeroOr(val, defVal)
}

// ZeroOr return default value on val is zero, else return val
func ZeroOr[T comdef.Number](val, defVal T) T {
	if val != 0 {
		return val
	}
	return defVal
}

// LessOr return val on val < max, else return default value.
//
// Example:
//
//	LessOr(11, 10, 1) // 1
//	LessOr(2, 10, 1) // 2
//	LessOr(10, 10, 1) // 1
func LessOr[T comdef.Number](val, max, devVal T) T {
	if val < max {
		return val
	}
	return devVal
}

// LteOr return val on val <= max, else return default value.
//
// Example:
//
//	LteOr(11, 10, 1) // 11
//	LteOr(2, 10, 1) // 2
//	LteOr(10, 10, 1) // 10
func LteOr[T comdef.Number](val, max, devVal T) T {
	if val <= max {
		return val
	}
	return devVal
}

// GreaterOr return val on val > max, else return default value.
//
// Example:
//
//	GreaterOr(23, 0, 2) // 23
//	GreaterOr(0, 0, 2) // 2
func GreaterOr[T comdef.Number](val, min, defVal T) T {
	if val > min {
		return val
	}
	return defVal
}

// GteOr return val on val >= max, else return default value.
//
// Example:
//
//	GteOr(23, 0, 2) // 23
//	GteOr(0, 0, 2) // 0
func GteOr[T comdef.Number](val, min, defVal T) T {
	if val >= min {
		return val
	}
	return defVal
}

// Mul computes the `a*b` value, rounding the result.
func Mul[T1, T2 comdef.Number](a T1, b T2) float64 {
	return math.Round(SafeFloat(a) * SafeFloat(b))
}

// MulF2i computes the float64 type a * b value, rounding the result to an integer.
func MulF2i(a, b float64) int {
	return int(math.Round(a * b))
}

// Div computes the `a/b` value, result uses a round handle.
func Div[T1, T2 comdef.Number](a T1, b T2) float64 {
	return math.Round(SafeFloat(a) / SafeFloat(b))
}

// DivInt computes the int type a / b value, rounding the result to an integer.
func DivInt[T comdef.Integer](a, b T) int {
	fv := math.Round(float64(a) / float64(b))
	return int(fv)
}

// DivF2i computes the float64 type a / b value, rounding the result to an integer.
func DivF2i(a, b float64) int {
	return int(math.Round(a / b))
}

// Percent returns a value percentage of the total
func Percent(val, total int) float64 {
	if total == 0 {
		return float64(0)
	}
	return (float64(val) / float64(total)) * 100
}
