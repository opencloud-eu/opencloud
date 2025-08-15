package comdef

import (
	"fmt"
	"io"
)

// ByteStringWriter interface
type ByteStringWriter interface {
	io.Writer
	io.ByteWriter
	io.StringWriter
	fmt.Stringer
}

// StringWriteStringer interface
type StringWriteStringer interface {
	io.StringWriter
	fmt.Stringer
}

// Int64able interface
type Int64able interface {
	Int64() (int64, error)
}

// Float64able interface
type Float64able interface {
	Float64() (float64, error)
}

// MapFunc definition
type MapFunc func(val any) (any, error)

//
//
// Matcher type
//
//

// Matcher interface
type Matcher[T any] interface {
	Match(s T) bool
}

// MatchFunc definition. implements Matcher interface
type MatchFunc[T any] func(v T) bool

// Match satisfies the Matcher interface
func (fn MatchFunc[T]) Match(v T) bool {
	return fn(v)
}

// StringMatcher interface
type StringMatcher interface {
	Match(s string) bool
}

// StringMatchFunc definition
type StringMatchFunc func(s string) bool

// Match satisfies the StringMatcher interface
func (fn StringMatchFunc) Match(s string) bool {
	return fn(s)
}

// StringHandler interface
type StringHandler interface {
	Handle(s string) string
}

// StringHandleFunc definition
type StringHandleFunc func(s string) string

// Handle satisfies the StringHandler interface
func (fn StringHandleFunc) Handle(s string) string {
	return fn(s)
}
