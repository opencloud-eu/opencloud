package mathutil

import "github.com/gookit/goutil/comdef"

// IsNumeric returns true if the given character is a numeric, otherwise false.
func IsNumeric(c byte) bool {
	return c >= '0' && c <= '9'
}

// Compare any intX,floatX value by given op. returns `first op(=,!=,<,<=,>,>=) second`
//
// Usage:
//
//	mathutil.Compare(2, 3, ">") // false
//	mathutil.Compare(2, 1.3, ">") // true
//	mathutil.Compare(2.2, 1.3, ">") // true
//	mathutil.Compare(2.1, 2, ">") // true
func Compare(first, second any, op string) bool {
	if first == nil || second == nil {
		return false
	}

	switch fVal := first.(type) {
	case float64:
		if sVal, err := ToFloat(second); err == nil {
			return CompFloat(fVal, sVal, op)
		}
	case float32:
		if sVal, err := ToFloat(second); err == nil {
			return CompFloat(float64(fVal), sVal, op)
		}
	default: // as int64
		if int1, err := ToInt64(first); err == nil {
			if int2, err := ToInt64(second); err == nil {
				return CompInt64(int1, int2, op)
			}
		}
	}

	return false
}

// CompInt compare all intX,uintX type value. returns `first op(=,!=,<,<=,>,>=) second`
func CompInt[T comdef.Xint](first, second T, op string) (ok bool) {
	return CompValue(first, second, op)
}

// CompInt64 compare int64 value. returns `first op(=,!=,<,<=,>,>=) second`
func CompInt64(first, second int64, op string) bool {
	return CompValue(first, second, op)
}

// CompFloat compare float64,float32 value. returns `first op(=,!=,<,<=,>,>=) second`
func CompFloat[T comdef.Float](first, second T, op string) (ok bool) {
	return CompValue(first, second, op)
}

// CompValue compare intX,uintX,floatX value. returns `first op(=,!=,<,<=,>,>=) second`
func CompValue[T comdef.Number](first, second T, op string) (ok bool) {
	switch op {
	case "<", "lt":
		ok = first < second
	case "<=", "lte":
		ok = first <= second
	case ">", "gt":
		ok = first > second
	case ">=", "gte":
		ok = first >= second
	case "=", "eq":
		ok = first == second
	case "!=", "ne", "neq":
		ok = first != second
	}
	return
}

// InRange check if val in int/float range [min, max]
func InRange[T comdef.Number](val, min, max T) bool {
	return val >= min && val <= max
}

// OutRange check if val not in int/float range [min, max]
func OutRange[T comdef.Number](val, min, max T) bool {
	return val < min || val > max
}

// InUintRange check if val in unit range [min, max]
func InUintRange[T comdef.Uint](val, min, max T) bool {
	if max == 0 {
		return val >= min
	}
	return val >= min && val <= max
}
