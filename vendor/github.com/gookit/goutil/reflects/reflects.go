// Package reflects Provide extends reflect util functions.
package reflects

import (
	"reflect"
	"time"
)

var emptyValue = reflect.Value{}

var (
	anyType   = reflect.TypeOf((*any)(nil)).Elem()
	errorType = reflect.TypeOf((*error)(nil)).Elem()

	// time.Time type
	timeType = reflect.TypeOf(time.Time{})
	// time.Duration type
	durationType = reflect.TypeOf(time.Duration(0))

	// fmtStringerType  = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	reflectValueType = reflect.TypeOf((*reflect.Value)(nil)).Elem()
)

// ConvFunc custom func convert input value to kind reflect.Value
type ConvFunc func(val any, kind reflect.Kind) (reflect.Value, error)
