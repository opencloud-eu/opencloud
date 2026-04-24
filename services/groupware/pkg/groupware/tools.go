package groupware

import (
	"iter"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/structs"
)

func ptr[T any](t T) *T {
	return &t
}

func trimmed(it iter.Seq[string]) iter.Seq[string] {
	return structs.MapSeq(it, strings.TrimSpace)
}

func notEmptyString(it iter.Seq[string]) iter.Seq[string] {
	return structs.FilterSeq(it, func(s string) bool { return s != "" })
}

func uintPtr(v uint) *uint {
	return &v
}

var UintPtrOne *uint = uintPtr(1)
var UintPtrZero *uint = uintPtr(0)

func ptrIf[T any | uint | int | bool](t T, predicate bool) *T {
	if predicate {
		return &t
	} else {
		return nil
	}
}
