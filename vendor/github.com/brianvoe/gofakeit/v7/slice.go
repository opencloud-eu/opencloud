package gofakeit

import (
	"reflect"
)

// Slice fills built-in types and exported fields of a struct with random data.
func Slice(v any) { sliceFunc(GlobalFaker, v) }

// Slice fills built-in types and exported fields of a struct with random data.
func (f *Faker) Slice(v any) { sliceFunc(f, v) }

func sliceFunc(f *Faker, v any) {
	// Note: We intentionally call r with size -1 instead of using structFunc.
	// structFunc starts with size 0, which would result in zero-length top-level
	// slices and maps. Passing -1 lets rSlice/rMap auto-size (random length)
	// when no fakesize tag is provided.
	r(f, reflect.TypeOf(v), reflect.ValueOf(v), "", -1, 0)
}
