package reflects

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/gookit/goutil/x/basefn"
)

// FuncX wrap a go func. represent a function
type FuncX struct {
	CallOpt
	// Name of func. eg: "MyFunc"
	Name string
	// rv is the `reflect.Value` of func
	rv reflect.Value
	rt reflect.Type
}

// NewFunc instance. param fn support func and reflect.Value
func NewFunc(fn any) *FuncX {
	var ok bool
	var rv reflect.Value
	if rv, ok = fn.(reflect.Value); !ok {
		rv = reflect.ValueOf(fn)
	}

	rv = indirectInterface(rv)
	if !rv.IsValid() {
		panic("input func is nil")
	}

	typ := rv.Type()
	if typ.Kind() != reflect.Func {
		basefn.Panicf("non-function of type: %s", typ)
	}

	return &FuncX{rv: rv, rt: typ}
}

// NumIn get the number of func input args
func (f *FuncX) NumIn() int {
	return f.rt.NumIn()
}

// NumOut get the number of func output args
func (f *FuncX) NumOut() int {
	return f.rt.NumOut()
}

// Call the function with given arguments.
//
// Usage:
//
//	func main() {
//		fn := func(a, b int) int {
//			return a + b
//		}
//
//		fx := NewFunc(fn)
//		ret, err := fx.Call(1, 2)
//		fmt.Println(ret[0], err) // Output: 3 <nil>
//	}
func (f *FuncX) Call(args ...any) ([]any, error) {
	// convert args to []reflect.Value
	argRvs := make([]reflect.Value, len(args))
	for i, arg := range args {
		argRvs[i] = reflect.ValueOf(arg)
	}

	ret, err := f.CallRV(argRvs)
	if err != nil {
		return nil, err
	}

	// convert ret to []any
	rets := make([]any, len(ret))
	for i, r := range ret {
		rets[i] = r.Interface()
	}
	return rets, nil
}

// Call2 returns the result of evaluating the first argument as a function.
// The function must return 1 result, or 2 results, the second of which is an error.
//
//   - Only support func with 1 or 2 return values: (val) OR (val, err)
//   - Will check args and try to convert input args to func args type.
func (f *FuncX) Call2(args ...any) (any, error) {
	// convert args to []reflect.Value
	argRvs := make([]reflect.Value, len(args))
	for i, arg := range args {
		argRvs[i] = reflect.ValueOf(arg)
	}

	if f.TypeChecker == nil {
		f.TypeChecker = OneOrTwoOutChecker
	}

	// do call func
	ret, err := Call(f.rv, argRvs, &f.CallOpt)
	if err != nil {
		return emptyValue, err
	}

	// func return like: (val, err)
	if len(ret) == 2 && !ret[1].IsNil() {
		return ret[0].Interface(), ret[1].Interface().(error)
	}
	return ret[0].Interface(), nil
}

// CallRV call the function with given reflect.Value arguments.
func (f *FuncX) CallRV(args []reflect.Value) ([]reflect.Value, error) {
	return Call(f.rv, args, &f.CallOpt)
}

// WithTypeChecker set type checker
func (f *FuncX) WithTypeChecker(checker TypeCheckerFn) *FuncX {
	f.TypeChecker = checker
	return f
}

// WithEnhanceConv set enhance convert
func (f *FuncX) WithEnhanceConv() *FuncX {
	f.EnhanceConv = true
	return f
}

// String of func
func (f *FuncX) String() string {
	return f.rt.String()
}

// TypeCheckerFn type checker func
type TypeCheckerFn func(typ reflect.Type) error

// CallOpt call options
type CallOpt struct {
	// TypeChecker check func type before call func. eg: check return values
	TypeChecker TypeCheckerFn
	// EnhanceConv try to enhance auto convert args to func args type
	// 	- support more type: string, int, uint, float, bool
	EnhanceConv bool
}

// OneOrTwoOutChecker check func type. only allow 1 or 2 return values
//
// Allow func returns:
//   - 1 return: (value)
//   - 2 return: (value, error)
var OneOrTwoOutChecker = func(typ reflect.Type) error {
	if !good1or2outFunc(typ) {
		return errors.New("func allow with 1 result or 2 results where the second is an error")
	}
	return nil
}

//
// TIP:
// 	flow func refer from text/template package.
//
//

// reports whether the function or method has the right result signature.
func good1or2outFunc(typ reflect.Type) bool {
	// We allow functions with 1 result or 2 results where the second is an error.
	switch {
	case typ.NumOut() == 1:
		return true
	case typ.NumOut() == 2 && typ.Out(1) == errorType:
		return true
	}
	return false
}

// Call2 returns the result of evaluating the first argument as a function.
// The function must return 1 result, or 2 results, the second of which is an error.
//
//   - Only support func with 1 or 2 return values: (val) OR (val, err)
//   - Will check args and try convert input args to func args type.
func Call2(fn reflect.Value, args []reflect.Value) (reflect.Value, error) {
	ret, err := Call(fn, args, &CallOpt{
		TypeChecker: OneOrTwoOutChecker,
	})
	if err != nil {
		return emptyValue, err
	}

	// func return like: (val, err)
	if len(ret) == 2 && !ret[1].IsNil() {
		return ret[0], ret[1].Interface().(error)
	}
	return ret[0], nil
}

// Call returns the result of evaluating the first argument as a function.
//
//   - Will check args and try convert input args to func args type.
//
// from text/template/funcs.go#call
func Call(fn reflect.Value, args []reflect.Value, opt *CallOpt) ([]reflect.Value, error) {
	fn = indirectInterface(fn)
	if !fn.IsValid() {
		return nil, fmt.Errorf("call of nil")
	}

	typ := fn.Type()
	if typ.Kind() != reflect.Func {
		return nil, fmt.Errorf("non-function of type %s", typ)
	}

	if opt == nil {
		opt = &CallOpt{}
	}
	if opt.TypeChecker != nil {
		if err := opt.TypeChecker(typ); err != nil {
			return nil, err
		}
	}

	numIn := typ.NumIn()
	var dddType reflect.Type
	if typ.IsVariadic() {
		if len(args) < numIn-1 {
			return nil, fmt.Errorf("wrong number of args: got %d want at least %d", len(args), numIn-1)
		}
		dddType = typ.In(numIn - 1).Elem()
	} else {
		if len(args) != numIn {
			return nil, fmt.Errorf("wrong number of args: got %d want %d", len(args), numIn)
		}
	}

	// Convert each arg to the type of the function's arg.
	argv := make([]reflect.Value, len(args))
	for i, arg := range args {
		arg = indirectInterface(arg)
		// Compute the expected type. Clumsy because of variadic.
		argType := dddType
		if !typ.IsVariadic() || i < numIn-1 {
			argType = typ.In(i)
		}

		var err error
		if argv[i], err = prepareArg(arg, argType, opt.EnhanceConv); err != nil {
			return nil, fmt.Errorf("arg %d: %w", i, err)
		}
	}

	return SafeCall(fn, argv)
}

// SafeCall2 runs fun.Call(args), and returns the resulting value and error, if
// any. If the call panics, the panic value is returned as an error.
//
// NOTE: Only support func with 1 or 2 return values: (val) OR (val, err)
//
// from text/template/funcs.go#safeCall
func SafeCall2(fun reflect.Value, args []reflect.Value) (val reflect.Value, err error) {
	ret, err := SafeCall(fun, args)
	if err != nil {
		return reflect.Value{}, err
	}

	// func return like: (val, err)
	if len(ret) == 2 && !ret[1].IsNil() {
		return ret[0], ret[1].Interface().(error)
	}
	return ret[0], nil
}

// SafeCall runs fun.Call(args), and returns the resulting values, or an error.
// If the call panics, the panic value is returned as an error.
func SafeCall(fun reflect.Value, args []reflect.Value) (ret []reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
		}
	}()

	ret = fun.Call(args)
	return
}

// prepareArg checks if value can be used as an argument of type argType, and
// converts an invalid value to appropriate zero if possible.
func prepareArg(value reflect.Value, argType reflect.Type, enhanced bool) (reflect.Value, error) {
	if !value.IsValid() {
		if !CanBeNil(argType) {
			return emptyValue, fmt.Errorf("value is nil; should be of type %s", argType)
		}

		value = reflect.Zero(argType)
	}

	if value.Type().AssignableTo(argType) {
		return value, nil
	}

	// If the argument is an int-like type, and the value is an int-like type, auto-convert.
	if IsIntLike(value.Kind()) && IsIntLike(argType.Kind()) && value.Type().ConvertibleTo(argType) {
		value = value.Convert(argType)
		return value, nil
	}

	// enhance convert value to argType, support more type: string, int, uint, float, bool
	if enhanced {
		return ValueByType(value.Interface(), argType)
	}
	return emptyValue, fmt.Errorf("value has type %s; should be %s", value.Type(), argType)
}
