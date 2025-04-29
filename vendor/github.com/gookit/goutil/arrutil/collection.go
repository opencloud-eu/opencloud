package arrutil

import (
	"errors"

	"github.com/gookit/goutil/comdef"
	"github.com/gookit/goutil/reflects"
)

// ErrElementNotFound is the error returned when the element is not found.
var ErrElementNotFound = errors.New("element not found")

// Comparer Function to compare two elements.
type Comparer[T any] func(a, b T) int

// type Comparer  func(a, b any) int

// Predicate Function to predicate a struct/value satisfies a condition.
type Predicate[T any] func(v T) bool

// StringEqualsComparer Comparer for string. It will compare the string by their value.
//
// returns: 0 if equal, -1 if a != b
func StringEqualsComparer(a, b string) int {
	if a == b {
		return 0
	}
	return -1
}

// ValueEqualsComparer Comparer for comdef.Compared type. It will compare by their value.
//
// returns: 0 if equal, -1 if a != b
func ValueEqualsComparer[T comdef.Compared](a, b T) int {
	if a == b {
		return 0
	}
	return -1
}

// ReflectEqualsComparer Comparer for struct ptr. It will compare by reflect.Value
//
// returns: 0 if equal, -1 if a != b
func ReflectEqualsComparer[T any](a, b T) int {
	if reflects.IsEqual(a, b) {
		return 0
	}
	return -1
}

// ElemTypeEqualsComparer Comparer for struct/value. It will compare the struct by their element type.
//
// returns: 0 if same type, -1 if not.
func ElemTypeEqualsComparer[T any](a, b T) int {
	at := reflects.TypeOf(a).SafeElem()
	bt := reflects.TypeOf(b).SafeElem()

	if at == bt {
		return 0
	}
	return -1
}

// TwowaySearch find specialized element in a slice forward and backward in the same time, should be more quickly.
//
//   - data: the slice to search in. MUST BE A SLICE.
//   - item: the element to search.
//   - fn: the comparer function.
//   - return: the index of the element, or -1 if not found.
func TwowaySearch[T any](data []T, item T, fn Comparer[T]) (int, error) {
	if data == nil {
		return -1, errors.New("collections.TwowaySearch: data is nil")
	}
	if fn == nil {
		return -1, errors.New("collections.TwowaySearch: fn is nil")
	}

	if len(data) == 0 {
		return -1, errors.New("collections.TwowaySearch: data is empty")
	}

	forward := 0
	backward := len(data) - 1

	for forward <= backward {
		if fn(data[forward], item) == 0 {
			return forward, nil
		}

		if fn(data[backward], item) == 0 {
			return backward, nil
		}

		forward++
		backward--
	}

	return -1, ErrElementNotFound
}

// CloneSlice Clone a slice.
//
//	data: the slice to clone.
//	returns: the cloned slice.
func CloneSlice[T any](data []T) []T {
	nt := make([]T, 0, len(data))
	nt = append(nt, data...)
	return nt
}

// Diff Produces the set difference of two slice according to a comparer function. alias of Differences
func Diff[T any](first, second []T, fn Comparer[T]) []T {
	return Differences(first, second, fn)
}

// Differences Produces the set difference of two slice according to a comparer function.
//
//   - first: the first slice. MUST BE A SLICE.
//   - second: the second slice. MUST BE A SLICE.
//   - fn: the comparer function.
//   - returns: the difference of the two slices.
//
// Example:
//
//	// Output: []string{"c"}
//	Differences([]string{"a", "b", "c"}, []string{"a", "b"}, arrutil.StringEqualsComparer
func Differences[T any](first, second []T, fn Comparer[T]) []T {
	firstLen := len(first)
	if firstLen == 0 {
		return CloneSlice(second)
	}

	secondLen := len(second)
	if secondLen == 0 {
		return CloneSlice(first)
	}

	maxLn := firstLen
	if secondLen > firstLen {
		maxLn = secondLen
	}

	result := make([]T, 0)
	for i := 0; i < maxLn; i++ {
		if i < firstLen {
			s := first[i]
			if i, _ := TwowaySearch(second, s, fn); i < 0 {
				result = append(result, s)
			}
		}

		if i < secondLen {
			t := second[i]
			if i, _ := TwowaySearch(first, t, fn); i < 0 {
				result = append(result, t)
			}
		}
	}

	return result
}

// Excepts Produces the set difference of two slice according to a comparer function.
//
//   - first: the first slice. MUST BE A SLICE.
//   - second: the second slice. MUST BE A SLICE.
//   - fn: the comparer function.
//   - returns: the difference of the two slices.
//
// Example:
//
//	// Output: []string{"c"}
//	Excepts([]string{"a", "b", "c"}, []string{"a", "b"}, arrutil.StringEqualsComparer)
func Excepts[T any](first, second []T, fn Comparer[T]) []T {
	if len(first) == 0 {
		return make([]T, 0)
	}
	if len(second) == 0 {
		return CloneSlice(first)
	}

	result := make([]T, 0)
	for _, s := range first {
		if i, _ := TwowaySearch(second, s, fn); i < 0 {
			result = append(result, s)
		}
	}
	return result
}

// Intersects Produces to intersect of two slice according to a comparer function.
//
//   - first: the first slice. MUST BE A SLICE.
//   - second: the second slice. MUST BE A SLICE.
//   - fn: the comparer function.
//   - returns: to intersect of the two slices.
//
// Example:
//
//	// Output: []string{"a", "b"}
//	Intersects([]string{"a", "b", "c"}, []string{"a", "b"}, arrutil.ValueEqualsComparer)
func Intersects[T any](first, second []T, fn Comparer[T]) []T {
	if len(first) == 0 || len(second) == 0 {
		return make([]T, 0)
	}

	result := make([]T, 0)
	for _, s := range first {
		if i, _ := TwowaySearch(second, s, fn); i >= 0 {
			result = append(result, s)
		}
	}
	return result
}

// Union Produces the set union of two slice according to a comparer function
//
//   - first: the first slice. MUST BE A SLICE.
//   - second: the second slice. MUST BE A SLICE.
//   - fn: the comparer function.
//   - returns: the union of the two slices.
//
// Example:
//
//	// Output: []string{"a", "b", "c"}
//	sl := Union([]string{"a", "b", "c"}, []string{"a", "b"}, arrutil.ValueEqualsComparer)
func Union[T any](first, second []T, fn Comparer[T]) []T {
	if len(first) == 0 {
		return CloneSlice(second)
	}

	excepts := Excepts(second, first, fn)
	nt := make([]T, 0, len(first)+len(second))
	nt = append(nt, first...)
	return append(nt, excepts...)
}

// Find Produces the value of a slice according to a predicate function.
//
//   - source: the slice. MUST BE A SLICE.
//   - fn: the predicate function.
//   - returns: the struct/value of the slice.
//
// Example:
//
//	// Output: "c"
//	val := Find([]string{"a", "b", "c"}, func(s string) bool {
//		return s == "c"
//	})
func Find[T any](source []T, fn Predicate[T]) (v T, err error) {
	err = ErrElementNotFound
	if len(source) == 0 {
		return
	}

	for _, s := range source {
		if fn(s) {
			return s, nil
		}
	}
	return
}

// FindOrDefault Produce the value f a slice to a predicate function,
// Produce default value when predicate function not found.
//
//   - source: the slice. MUST BE A SLICE.
//   - fn: the predicate function.
//   - defaultValue: the default value.
//   - returns: the value of the slice.
//
// Example:
//
//	// Output: "d"
//	val := FindOrDefault([]string{"a", "b", "c"}, func(s string) bool {
//		return s == "d"
//	}, "d")
func FindOrDefault[T any](source []T, fn Predicate[T], defaultValue T) T {
	item, err := Find(source, fn)
	if err != nil {
		return defaultValue
	}
	return item
}

// TakeWhile Produce the set of a slice according to a predicate function,
// Produce empty slice when predicate function not matched.
//
//   - data: the slice. MUST BE A SLICE.
//   - fn: the predicate function.
//   - returns: the set of the slice.
//
// Example:
//
//	// Output: []string{"a", "b"}
//	sl := TakeWhile([]string{"a", "b", "c"}, func(s string) bool {
//		return s != "c"
//	})
func TakeWhile[T any](data []T, fn Predicate[T]) []T {
	result := make([]T, 0)
	if len(data) == 0 {
		return result
	}

	for _, v := range data {
		if fn(v) {
			result = append(result, v)
		}
	}
	return result
}

// ExceptWhile Produce the set of a slice except with a predicate function,
// Produce original slice when predicate function not match.
//
//   - data: the slice. MUST BE A SLICE.
//   - fn: the predicate function.
//   - returns: the set of the slice.
//
// Example:
//
//	// Output: []string{"a", "b"}
//	sl := ExceptWhile([]string{"a", "b", "c"}, func(s string) bool {
//		return s == "c"
//	})
func ExceptWhile[T any](data []T, fn Predicate[T]) []T {
	result := make([]T, 0)
	if len(data) == 0 {
		return result
	}

	for _, v := range data {
		if !fn(v) {
			result = append(result, v)
		}
	}
	return result
}
