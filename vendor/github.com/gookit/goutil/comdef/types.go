package comdef

// Int interface type
type Int interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// Uint interface type
type Uint interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Xint interface type. alias of Integer
type Xint interface {
	Int | Uint
}

// Integer interface type. all int or uint types
type Integer interface {
	Int | Uint
}

// Float interface type
type Float interface {
	~float32 | ~float64
}

// IntOrFloat interface type. all int and float types
type IntOrFloat interface {
	Int | Float
}

// XintOrFloat interface type. all int, uint and float types
type XintOrFloat interface {
	Int | Uint | Float
}

// SortedType interface type. same of constraints.Ordered
//
// it can be ordered, that supports the operators < <= >= >.
//
// contains: (x)int, float, ~string types
type SortedType interface {
	Int | Uint | Float | ~string
}

// Compared type. alias of constraints.SortedType
//
// TODO: use type alias, will error on go1.18 Error: types.go:50: interface contains type constraints
// type Compared = SortedType
type Compared interface {
	Int | Uint | Float | ~string
}

// SimpleType interface type. alias of ScalarType
//
// contains: (x)int, float, ~string, ~bool types
type SimpleType interface {
	Int | Uint | Float | ~string | ~bool
}

// ScalarType basic interface type.
//
// TIP: has bool type, it cannot be ordered
//
// contains: (x)int, float, ~string, ~bool types
type ScalarType interface {
	Int | Uint | Float | ~string | ~bool
}
